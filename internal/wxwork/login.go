package wxwork

import (
	"agent-desk/internal/pkg/dto/response"
	"agent-desk/internal/pkg/errorsx"
	"agent-desk/internal/pkg/i18nx"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/silenceper/wechat/v2/work/oauth"
)

const (
	StateTTL             = 5 * time.Minute
	LoginTicketTTL       = 1 * time.Minute
	defaultLoginNextPath = "/dashboard"
)

var (
	errStateInvalid  = i18nx.Errorf("error.e0110")
	loginTicketStore sync.Map
)

type statePayload struct {
	Next      string `json:"next"`
	Nonce     string `json:"nonce"`
	ExpiredAt int64  `json:"expiredAt"`
}

type loginTicket struct {
	Response  *response.LoginResponse
	ExpiredAt time.Time
}

func BuildLoginURL(state string) (string, error) {
	if !Enabled() {
		return "", i18nx.Errorf("error.e0109")
	}
	if strings.TrimSpace(wxCfg.OAuthRedirect) == "" {
		return "", i18nx.Errorf("error.e0107")
	}
	agentID := DefaultAgentID()
	if strings.TrimSpace(agentID) == "" {
		return "", i18nx.Errorf("error.e0093")
	}
	return fmt.Sprintf(
		"https://open.weixin.qq.com/connect/oauth2/authorize?appid=%s&redirect_uri=%s&response_type=code&scope=snsapi_privateinfo&agentid=%s&state=%s#wechat_redirect",
		url.QueryEscape(strings.TrimSpace(wxCfg.CorpID)),
		url.QueryEscape(strings.TrimSpace(wxCfg.OAuthRedirect)),
		url.QueryEscape(strings.TrimSpace(agentID)),
		url.QueryEscape(strings.TrimSpace(state)),
	), nil
}

func BuildQRCodeLoginURL(state string) (string, error) {
	if !Enabled() {
		return "", i18nx.Errorf("error.e0109")
	}
	if strings.TrimSpace(wxCfg.OAuthRedirect) == "" {
		return "", i18nx.Errorf("error.e0107")
	}
	agentID := DefaultAgentID()
	if strings.TrimSpace(agentID) == "" {
		return "", i18nx.Errorf("error.e0093")
	}
	return fmt.Sprintf(
		"https://open.work.weixin.qq.com/wwopen/sso/qrConnect?appid=%s&agentid=%s&redirect_uri=%s&state=%s",
		url.QueryEscape(strings.TrimSpace(wxCfg.CorpID)),
		url.QueryEscape(strings.TrimSpace(agentID)),
		url.QueryEscape(strings.TrimSpace(wxCfg.OAuthRedirect)),
		url.QueryEscape(strings.TrimSpace(state)),
	), nil
}

func CreateState(next string) (string, error) {
	secret := strings.TrimSpace(StateSecret())
	if secret == "" {
		return "", errorsx.BusinessErrorI18n(1, "error.wxwork.loginSecretMissing")
	}
	payload := statePayload{
		Next:      sanitizeNextPath(next),
		ExpiredAt: time.Now().Add(StateTTL).Unix(),
	}
	nonce, err := randomToken("ws_")
	if err != nil {
		return "", err
	}
	payload.Nonce = nonce

	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	encoded := base64.RawURLEncoding.EncodeToString(body)
	return encoded + "." + signState(encoded, secret), nil
}

func ParseState(state string) (string, error) {
	secret := strings.TrimSpace(StateSecret())
	if secret == "" {
		return "", errStateInvalid
	}
	parts := strings.Split(strings.TrimSpace(state), ".")
	if len(parts) != 2 {
		return "", errStateInvalid
	}
	if !hmac.Equal([]byte(parts[1]), []byte(signState(parts[0], secret))) {
		return "", errStateInvalid
	}
	body, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return "", errStateInvalid
	}
	payload := statePayload{}
	if err = json.Unmarshal(body, &payload); err != nil {
		return "", errStateInvalid
	}
	if payload.ExpiredAt <= time.Now().Unix() {
		return "", errStateInvalid
	}
	return sanitizeNextPath(payload.Next), nil
}

func IssueLoginTicket(loginResp *response.LoginResponse) (string, error) {
	if loginResp == nil {
		return "", i18nx.Errorf("error.e0272")
	}
	ticket, err := randomToken("wlt_")
	if err != nil {
		return "", err
	}
	cleanupExpiredLoginTickets()
	loginTicketStore.Store(ticket, loginTicket{
		Response:  loginResp,
		ExpiredAt: time.Now().Add(LoginTicketTTL),
	})
	return ticket, nil
}

func ConsumeLoginTicket(ticket string) (*response.LoginResponse, error) {
	ticket = strings.TrimSpace(ticket)
	if ticket == "" {
		return nil, errorsx.InvalidParamI18n("error.e0072")
	}
	value, ok := loginTicketStore.LoadAndDelete(ticket)
	if !ok {
		return nil, errorsx.UnauthorizedI18n("error.e0271")
	}
	record, ok := value.(loginTicket)
	if !ok || record.Response == nil || time.Now().After(record.ExpiredAt) {
		return nil, errorsx.UnauthorizedI18n("error.e0271")
	}
	return record.Response, nil
}

func signState(content, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(content))
	return hex.EncodeToString(mac.Sum(nil))
}

func cleanupExpiredLoginTickets() {
	now := time.Now()
	loginTicketStore.Range(func(key, value any) bool {
		record, ok := value.(loginTicket)
		if !ok || now.After(record.ExpiredAt) {
			loginTicketStore.Delete(key)
		}
		return true
	})
}

func sanitizeNextPath(next string) string {
	next = strings.TrimSpace(next)
	if next == "" || !strings.HasPrefix(next, "/") || strings.HasPrefix(next, "//") {
		return defaultLoginNextPath
	}
	return next
}

func randomToken(prefix string) (string, error) {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return prefix + hex.EncodeToString(buf), nil
}

func GetUserDetail(code string) (*LoginUser, error) {
	if !Enabled() {
		return nil, i18nx.Errorf("error.e0109")
	}
	code = strings.TrimSpace(code)
	if code == "" {
		return nil, i18nx.Errorf("error.e0202")
	}

	oauthClient := defaultWork.GetOauth()
	userInfo, err := oauthClient.GetUserInfo(code)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(userInfo.UserID) == "" {
		return nil, i18nx.Errorf("error.e0198")
	}

	ret := &LoginUser{
		CorpID:         wxCfg.CorpID,
		UserID:         strings.TrimSpace(userInfo.UserID),
		OpenID:         strings.TrimSpace(userInfo.OpenID),
		ExternalUserID: strings.TrimSpace(userInfo.ExternalUserID),
		UserTicket:     strings.TrimSpace(userInfo.UserTicket),
		UserInfo:       userInfo,
	}

	if ret.UserTicket != "" {
		if detail, detailErr := oauthClient.GetUserDetail(&oauth.GetUserDetailRequest{UserTicket: ret.UserTicket}); detailErr == nil {
			ret.UserDetail = detail
			ret.Avatar = strings.TrimSpace(detail.Avatar)
			ret.Mobile = strings.TrimSpace(detail.Mobile)
			ret.Email = strings.TrimSpace(detail.Email)
			ret.BizMail = strings.TrimSpace(detail.BizMail)
		}
	}

	if profile, profileErr := defaultWork.GetAddressList().UserGet(ret.UserID); profileErr == nil {
		ret.UserProfile = profile
		if strings.TrimSpace(profile.Name) != "" {
			ret.Name = strings.TrimSpace(profile.Name)
		}
		if strings.TrimSpace(profile.Avatar) != "" {
			ret.Avatar = strings.TrimSpace(profile.Avatar)
		}
		if ret.Mobile == "" && strings.TrimSpace(profile.Mobile) != "" {
			ret.Mobile = strings.TrimSpace(profile.Mobile)
		}
		if ret.Email == "" && strings.TrimSpace(profile.Email) != "" {
			ret.Email = strings.TrimSpace(profile.Email)
		}
		if ret.BizMail == "" && strings.TrimSpace(profile.BizMail) != "" {
			ret.BizMail = strings.TrimSpace(profile.BizMail)
		}
	}

	if ret.Name == "" {
		ret.Name = ret.UserID
	}
	return ret, nil
}
