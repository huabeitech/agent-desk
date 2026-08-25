package openidentity

import (
	"agent-desk/internal/pkg/config"
	"agent-desk/internal/pkg/enums"
	"agent-desk/internal/pkg/errorsx"
	"errors"
	"net/url"
	"strings"

	"agent-desk/internal/pkg/httpx/params"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/mlogclub/simple/common/strs"
)

// ExternalUser 外部访客身份（IM 客户），与站内 AuthPrincipal 区分。
type ExternalUser struct {
	ExternalSource enums.ExternalSource `json:"externalSource"`
	ExternalID     string               `json:"externalId"`
	ExternalName   string               `json:"externalName"`
}

type UserTokenClaims struct {
	TokenType string `json:"typ,omitempty"`
	UserID    string `json:"userId"`
	Name      string `json:"name"`
	jwt.RegisteredClaims
}

const SupportUserTokenType = "support_user"

func GetExternalUser(ctx *gin.Context, externalUserSecret string) (*ExternalUser, error) {
	if userToken := getUserToken(ctx); strs.IsNotBlank(userToken) {
		supportUserSecret := config.Current().CustomerSession.Secret
		if strs.IsNotBlank(supportUserSecret) {
			if claims, err := verifySupportUserToken(userToken, supportUserSecret); err == nil {
				return &ExternalUser{
					ExternalSource: enums.ExternalSourceUser,
					ExternalID:     claims.UserID,
					ExternalName:   claims.Name,
				}, nil
			}
		}
		claims, err := verifyUserToken(userToken, externalUserSecret)
		if err != nil {
			return nil, err
		}
		return &ExternalUser{
			ExternalSource: enums.ExternalSourceExternal,
			ExternalID:     claims.UserID,
			ExternalName:   claims.Name,
		}, nil
	}
	return getGuestUser(ctx)
}

func verifyUserToken(userToken, secret string) (*UserTokenClaims, error) {
	if strs.IsBlank(userToken) {
		return nil, errorsx.UnauthorizedI18n("error.e0263")
	}
	if strs.IsBlank(secret) {
		return nil, errorsx.UnauthorizedI18n("error.e0266")
	}

	claims := &UserTokenClaims{}
	token, err := jwt.ParseWithClaims(userToken, claims, func(token *jwt.Token) (any, error) {
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
			return nil, errorsx.UnauthorizedI18n("error.e0264")
		}
		return nil, errorsx.UnauthorizedI18n("error.e0265")
	}
	if token == nil || !token.Valid {
		return nil, errorsx.UnauthorizedI18n("error.e0265")
	}

	if strs.IsBlank(claims.UserID) {
		return nil, errorsx.UnauthorizedI18n("error.e0262")
	}
	if strs.IsBlank(claims.Name) {
		return nil, errorsx.UnauthorizedI18n("error.e0261")
	}
	if claims.ExpiresAt == nil {
		return nil, errorsx.UnauthorizedI18n("error.e0264")
	}

	return claims, nil
}

func verifySupportUserToken(userToken, secret string) (*UserTokenClaims, error) {
	claims, err := verifyUserToken(userToken, secret)
	if err != nil {
		return nil, err
	}
	if claims.TokenType != SupportUserTokenType {
		return nil, errorsx.UnauthorizedI18n("error.e0265")
	}
	return claims, nil
}

func getUserToken(ctx *gin.Context) string {
	auth := strings.TrimSpace(ctx.GetHeader("Authorization"))
	if len(auth) > 7 && strings.EqualFold(auth[:7], "Bearer ") {
		if token := strings.TrimSpace(auth[7:]); token != "" {
			return token
		}
	}
	userToken, _ := params.Get(ctx, "userToken")
	return strings.TrimSpace(userToken)
}

func getGuestUser(ctx *gin.Context) (*ExternalUser, error) {
	externalID := getExternalID(ctx)
	if strs.IsBlank(externalID) {
		return nil, errorsx.UnauthorizedI18n("error.e0262")
	}
	return &ExternalUser{
		ExternalSource: enums.ExternalSourceGuest,
		ExternalID:     externalID,
		ExternalName:   getExternalName(ctx),
	}, nil
}

func getExternalID(ctx *gin.Context) string {
	externalID := ctx.GetHeader("X-External-Id")
	if strs.IsBlank(externalID) {
		externalID, _ = params.Get(ctx, "externalId")
	}
	return externalID
}

func getExternalName(ctx *gin.Context) string {
	externalName := ctx.GetHeader("X-External-Name")
	if strs.IsBlank(externalName) {
		externalName, _ = params.Get(ctx, "externalName")
	}
	if strs.IsNotBlank(externalName) {
		externalName, _ = url.QueryUnescape(externalName)
	}
	return externalName
}

func decodeExternalDisplayName(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	dec, err := url.QueryUnescape(s)
	if err != nil {
		return s
	}
	return strings.TrimSpace(dec)
}
