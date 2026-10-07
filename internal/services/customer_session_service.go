package services

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"agent-desk/internal/models"
	"agent-desk/internal/pkg/config"
	"agent-desk/internal/pkg/dto/response"
	"agent-desk/internal/pkg/enums"
	"agent-desk/internal/pkg/errorsx"
	"agent-desk/internal/pkg/openidentity"
	"agent-desk/internal/repositories"

	"agent-desk/internal/pkg/httpx/params"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/mlogclub/simple/sqls"
)

const (
	customerSessionTokenType = "customer_session"
	customerSessionHeader    = "X-Customer-Session-Token"
	customerSessionExpHeader = "X-Customer-Session-Expires-At"
	supportUserTokenTTL      = 10 * time.Minute
)

var CustomerSessionService = newCustomerSessionService()

func newCustomerSessionService() *customerSessionService {
	return &customerSessionService{}
}

type customerSessionService struct {
}

type customerSessionClaims struct {
	TokenType    string `json:"typ"`
	ChannelID    int64  `json:"channelId"`
	ChannelCode  string `json:"channelCode"`
	CustomerID   int64  `json:"customerId"`
	CustomerName string `json:"customerName"`
	IdentityKey  string `json:"identityKey"`
	jwt.RegisteredClaims
}

type CustomerSessionVerifyResult struct {
	ExternalUser *openidentity.ExternalUser
	Token        string
	ExpiresAt    time.Time
	Refreshed    bool
}

func (s *customerSessionService) Exchange(channel *models.Channel, externalUser openidentity.ExternalUser) (*response.CustomerSessionExchangeResponse, error) {
	if channel == nil || channel.Status != enums.StatusOk {
		return nil, errorsx.InvalidParamI18n("error.e0209")
	}
	var customerID int64
	if err := sqls.WithTransaction(func(ctx *sqls.TxContext) error {
		id, err := CustomerService.EnsureExternalCustomer(ctx, externalUser)
		if err != nil {
			return err
		}
		customerID = id
		return nil
	}); err != nil {
		return nil, err
	}
	customer := CustomerService.Get(customerID)
	if customer == nil || customer.Status == enums.StatusDeleted {
		return nil, errorsx.InvalidParamI18n("error.e0155")
	}
	token, expiresAt, err := s.Sign(channel, customer, externalUser)
	if err != nil {
		return nil, err
	}
	return &response.CustomerSessionExchangeResponse{
		CustomerSessionToken: token,
		ExpiresAt:            expiresAt.Format(time.DateTime),
		IdentityKey:          s.identityKey(externalUser),
		Customer: response.CustomerSessionCustomerResponse{
			ID:   customer.ID,
			Name: strings.TrimSpace(customer.Name),
		},
	}, nil
}

func (s *customerSessionService) SignSupportUserToken(channel *models.Channel, user *models.User) (*response.SupportAICustomerServiceUserTokenResponse, error) {
	if channel == nil || channel.Status != enums.StatusOk || channel.ChannelType != enums.ChannelTypeWeb {
		return nil, errorsx.InvalidParamI18n("error.e0209")
	}
	if user == nil || user.Status != enums.StatusOk {
		return nil, errorsx.UnauthorizedI18n("error.e0256")
	}
	secret := strings.TrimSpace(config.Current().CustomerSession.Secret)
	if strings.TrimSpace(secret) == "" {
		return nil, errorsx.BusinessErrorI18n(1, "error.customerSession.secretMissing")
	}
	now := time.Now()
	expiresAt := now.Add(supportUserTokenTTL)
	name := strings.TrimSpace(user.Nickname)
	if name == "" {
		name = strings.TrimSpace(user.Username)
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"typ":    openidentity.SupportUserTokenType,
		"userId": strconv.FormatInt(user.ID, 10),
		"name":   name,
		"iat":    now.Unix(),
		"exp":    expiresAt.Unix(),
	}).SignedString([]byte(secret))
	if err != nil {
		return nil, err
	}
	return &response.SupportAICustomerServiceUserTokenResponse{
		UserToken: token,
		ExpiresAt: expiresAt.Format(time.DateTime),
	}, nil
}

func (s *customerSessionService) Sign(channel *models.Channel, customer *models.Customer, externalUser openidentity.ExternalUser) (string, time.Time, error) {
	cfg := config.Current().CustomerSession
	secret := strings.TrimSpace(cfg.Secret)
	if secret == "" {
		return "", time.Time{}, errorsx.BusinessErrorI18n(1, "error.customerSession.secretMissing")
	}
	if channel == nil || customer == nil {
		return "", time.Time{}, errorsx.InvalidParamI18n("error.e0158")
	}
	now := time.Now()
	expiresAt := now.Add(time.Duration(cfg.TTL()) * time.Minute)
	claims := customerSessionClaims{
		TokenType:    customerSessionTokenType,
		ChannelID:    channel.ID,
		ChannelCode:  strings.TrimSpace(channel.ChannelID),
		CustomerID:   customer.ID,
		CustomerName: strings.TrimSpace(customer.Name),
		IdentityKey:  s.identityKey(externalUser),
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
	if err != nil {
		return "", time.Time{}, err
	}
	return token, expiresAt, nil
}

func (s *customerSessionService) VerifyRequest(ctx *gin.Context, channel *models.Channel) (*CustomerSessionVerifyResult, error) {
	token := s.getCustomerSessionToken(ctx)
	if token == "" {
		return nil, errorsx.UnauthorizedI18n("error.e0157")
	}
	claims, err := s.verifyToken(token)
	if err != nil {
		return nil, err
	}
	if channel == nil || channel.Status != enums.StatusOk {
		return nil, errorsx.InvalidParamI18n("error.e0209")
	}
	if claims.ChannelID != channel.ID || strings.TrimSpace(claims.ChannelCode) != strings.TrimSpace(channel.ChannelID) {
		return nil, errorsx.UnauthorizedI18n("error.e0161")
	}
	customer := CustomerService.Get(claims.CustomerID)
	if customer == nil || customer.Status == enums.StatusDeleted {
		return nil, errorsx.UnauthorizedI18n("error.e0161")
	}
	external, err := s.externalUserFromClaims(claims, customer)
	if err != nil {
		return nil, err
	}
	result := &CustomerSessionVerifyResult{
		ExternalUser: external,
		Token:        token,
		ExpiresAt:    claims.ExpiresAt.Time,
	}
	if s.shouldRefresh(claims.ExpiresAt.Time) {
		newToken, expiresAt, err := s.Sign(channel, customer, *external)
		if err != nil {
			return nil, err
		}
		result.Token = newToken
		result.ExpiresAt = expiresAt
		result.Refreshed = true
	}
	return result, nil
}

func (s *customerSessionService) SetRefreshHeaders(ctx *gin.Context, result *CustomerSessionVerifyResult) {
	if ctx == nil || result == nil || !result.Refreshed {
		return
	}
	ctx.Header(customerSessionHeader, result.Token)
	ctx.Header(customerSessionExpHeader, result.ExpiresAt.Format(time.DateTime))
}

func (s *customerSessionService) verifyToken(rawToken string) (*customerSessionClaims, error) {
	cfg := config.Current().CustomerSession
	secret := strings.TrimSpace(cfg.Secret)
	if secret == "" {
		return nil, errorsx.BusinessErrorI18n(1, "error.customerSession.secretMissing")
	}
	claims := &customerSessionClaims{}
	token, err := jwt.ParseWithClaims(rawToken, claims, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unsupported signing method")
		}
		return []byte(secret), nil
	}, jwt.WithExpirationRequired(), jwt.WithValidMethods([]string{
		jwt.SigningMethodHS256.Alg(),
		jwt.SigningMethodHS384.Alg(),
		jwt.SigningMethodHS512.Alg(),
	}))
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, errorsx.UnauthorizedI18n("error.e0160")
		}
		return nil, errorsx.UnauthorizedI18n("error.e0161")
	}
	if token == nil || !token.Valid || claims.TokenType != customerSessionTokenType || claims.ExpiresAt == nil {
		return nil, errorsx.UnauthorizedI18n("error.e0161")
	}
	if claims.ChannelID <= 0 || strings.TrimSpace(claims.ChannelCode) == "" || claims.CustomerID <= 0 || strings.TrimSpace(claims.IdentityKey) == "" {
		return nil, errorsx.UnauthorizedI18n("error.e0161")
	}
	return claims, nil
}

func (s *customerSessionService) externalUserFromClaims(claims *customerSessionClaims, customer *models.Customer) (*openidentity.ExternalUser, error) {
	identityKey := strings.TrimSpace(claims.IdentityKey)
	parts := strings.SplitN(identityKey, ":", 2)
	if len(parts) != 2 || strings.TrimSpace(parts[1]) == "" {
		return nil, errorsx.UnauthorizedI18n("error.e0161")
	}
	var source enums.ExternalSource
	switch parts[0] {
	case "user":
		source = enums.ExternalSourceUser
	case "external":
		source = enums.ExternalSourceExternal
	case "guest":
		source = enums.ExternalSourceGuest
	default:
		return nil, errorsx.UnauthorizedI18n("error.e0161")
	}
	identity := repositories.CustomerIdentityRepository.GetBy(sqls.DB(), source, parts[1])
	if identity == nil || identity.CustomerID != claims.CustomerID {
		return nil, errorsx.UnauthorizedI18n("error.e0161")
	}
	name := strings.TrimSpace(claims.CustomerName)
	if customer != nil && strings.TrimSpace(customer.Name) != "" {
		name = strings.TrimSpace(customer.Name)
	}
	return &openidentity.ExternalUser{
		ExternalSource: source,
		ExternalID:     parts[1],
		ExternalName:   name,
	}, nil
}

func (s *customerSessionService) shouldRefresh(expiresAt time.Time) bool {
	threshold := config.Current().CustomerSession.RefreshThreshold()
	return time.Until(expiresAt) <= time.Duration(threshold)*time.Minute
}

func (s *customerSessionService) identityKey(externalUser openidentity.ExternalUser) string {
	switch externalUser.ExternalSource {
	case enums.ExternalSourceUser:
		return "user:" + strings.TrimSpace(externalUser.ExternalID)
	case enums.ExternalSourceExternal:
		return "external:" + strings.TrimSpace(externalUser.ExternalID)
	default:
		return "guest:" + strings.TrimSpace(externalUser.ExternalID)
	}
}

func (s *customerSessionService) getCustomerSessionToken(ctx *gin.Context) string {
	auth := strings.TrimSpace(ctx.GetHeader("Authorization"))
	if len(auth) > 7 && strings.EqualFold(auth[:7], "Bearer ") {
		if token := strings.TrimSpace(auth[7:]); token != "" {
			return token
		}
	}
	token, _ := params.Get(ctx, "customerSessionToken")
	return strings.TrimSpace(token)
}
