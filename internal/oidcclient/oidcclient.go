package oidcclient

import (
	"agent-desk/internal/pkg/config"
	"agent-desk/internal/pkg/dto/response"
	"agent-desk/internal/pkg/errorsx"
	"agent-desk/internal/pkg/i18nx"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"strings"
	"sync"
	"time"

	gooidc "github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

const (
	StateTTL             = 5 * time.Minute
	LoginTicketTTL       = 1 * time.Minute
	defaultLoginNextPath = "/dashboard"
)

var (
	oidcMu           sync.Mutex
	oidcCfg          config.OIDCConfig
	provider         *gooidc.Provider
	oauthConfig      *oauth2.Config
	idTokenVerifier  *gooidc.IDTokenVerifier
	loginTicketStore sync.Map
)

type OrganizationClaim struct {
	ID   string `json:"id"`
	Slug string `json:"slug,omitempty"`
	Name string `json:"name"`
	Role string `json:"role"`
}

type Profile struct {
	Subject           string              `json:"sub"`
	Email             string              `json:"email,omitempty"`
	PreferredUsername string              `json:"preferred_username,omitempty"`
	Name              string              `json:"name,omitempty"`
	Picture           string              `json:"picture,omitempty"`
	ActiveOrgID       string              `json:"active_org_id,omitempty"`
	Organizations     []OrganizationClaim `json:"organizations,omitempty"`
	RawProfile        string              `json:"-"`
}

type statePayload struct {
	Next      string `json:"next"`
	Nonce     string `json:"nonce"`
	Verifier  string `json:"verifier,omitempty"`
	ExpiredAt int64  `json:"expiredAt"`
}

type loginTicket struct {
	Response  *response.LoginResponse
	ExpiredAt time.Time
}

func Init(ctx context.Context) error {
	oidcMu.Lock()
	defer oidcMu.Unlock()

	provider = nil
	oauthConfig = nil
	idTokenVerifier = nil
	oidcCfg = config.OIDCConfig{}

	cfg := config.Current().OIDC
	if !cfg.Enabled {
		return nil
	}
	oidcCfg = cfg
	if strings.TrimSpace(cfg.Issuer) == "" {
		return i18nx.Errorf("error.e0039")
	}
	if strings.TrimSpace(cfg.ClientID) == "" {
		return i18nx.Errorf("error.e0036")
	}
	if strings.TrimSpace(cfg.ClientSecret) == "" {
		return i18nx.Errorf("error.e0037")
	}
	if strings.TrimSpace(cfg.RedirectURL) == "" {
		return i18nx.Errorf("error.e0040")
	}

	if ctx == nil {
		ctx = context.Background()
	}
	connCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	p, err := gooidc.NewProvider(connCtx, strings.TrimSpace(cfg.Issuer))
	if err != nil {
		return err
	}
	scopes := cfg.Scopes
	if len(scopes) == 0 {
		scopes = []string{gooidc.ScopeOpenID, "profile", "email", "offline_access"}
	}
	endpoint := p.Endpoint()
	authStyle := oauth2.AuthStyleInHeader
	switch strings.ToLower(strings.TrimSpace(cfg.AuthStyle)) {
	case "post", "params", "inparams", "client_secret_post":
		authStyle = oauth2.AuthStyleInParams
	case "basic", "header", "inheader", "client_secret_basic":
		authStyle = oauth2.AuthStyleInHeader
	case "auto", "autodetect":
		authStyle = oauth2.AuthStyleAutoDetect
	}
	endpoint.AuthStyle = authStyle

	provider = p
	oauthConfig = &oauth2.Config{
		ClientID:     strings.TrimSpace(cfg.ClientID),
		ClientSecret: strings.TrimSpace(cfg.ClientSecret),
		Endpoint:     endpoint,
		RedirectURL:  strings.TrimSpace(cfg.RedirectURL),
		Scopes:       scopes,
	}
	idTokenVerifier = p.Verifier(&gooidc.Config{ClientID: strings.TrimSpace(cfg.ClientID)})
	return nil
}

func ensureInitialized(ctx context.Context) {
	if !Enabled() && config.Current().OIDC.Enabled {
		_ = Init(ctx)
	}
}

func Enabled() bool {
	oidcMu.Lock()
	defer oidcMu.Unlock()
	return oidcCfg.Enabled && provider != nil && oauthConfig != nil && idTokenVerifier != nil
}

func BuildAuthCodeURL(next string) (string, error) {
	ensureInitialized(context.Background())
	if !Enabled() {
		return "", errorsx.BusinessErrorI18n(1, "error.oidc.loginDisabled")
	}

	rawVerifier := make([]byte, 32)
	if _, err := rand.Read(rawVerifier); err != nil {
		return "", err
	}
	verifier := base64.RawURLEncoding.EncodeToString(rawVerifier)
	h := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(h[:])

	state, err := CreateState(next, verifier)
	if err != nil {
		return "", err
	}
	oidcMu.Lock()
	defer oidcMu.Unlock()
	return oauthConfig.AuthCodeURL(
		state,
		oauth2.SetAuthURLParam("code_challenge", challenge),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
	), nil
}

func ExchangeCode(ctx context.Context, code string, verifier string) (*Profile, error) {
	ensureInitialized(ctx)
	if !Enabled() {
		return nil, errorsx.BusinessErrorI18n(1, "error.oidc.loginDisabled")
	}
	code = strings.TrimSpace(code)
	if code == "" {
		return nil, errorsx.InvalidParamI18n("error.e0041")
	}

	oidcMu.Lock()
	idVerifier := idTokenVerifier
	oauthCfg := oauthConfig
	prov := provider
	oidcMu.Unlock()

	var opts []oauth2.AuthCodeOption
	if strings.TrimSpace(verifier) != "" {
		opts = append(opts, oauth2.SetAuthURLParam("code_verifier", strings.TrimSpace(verifier)))
	}

	token, err := oauthCfg.Exchange(ctx, code, opts...)
	if err != nil {
		return nil, err
	}
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok || strings.TrimSpace(rawIDToken) == "" {
		return nil, errorsx.UnauthorizedI18n("error.e0038")
	}
	idToken, err := idVerifier.Verify(ctx, rawIDToken)
	if err != nil {
		return nil, err
	}
	profile, err := profileFromIDToken(idToken)
	if err != nil {
		return nil, err
	}
	userInfo, err := prov.UserInfo(ctx, oauth2.StaticTokenSource(token))
	if err == nil && userInfo != nil && strings.TrimSpace(userInfo.Subject) == profile.Subject {
		if mergedProfile, mergeErr := profileFromUserInfo(userInfo, profile); mergeErr == nil {
			profile = mergedProfile
		}
	}
	return profile, nil
}

func CreateState(next string, verifier ...string) (string, error) {
	secret := stateSecret()
	if secret == "" {
		return "", errorsx.BusinessErrorI18n(2, "error.oidc.stateSecretMissing")
	}
	nonce, err := randomToken("os_")
	if err != nil {
		return "", err
	}
	v := ""
	if len(verifier) > 0 {
		v = verifier[0]
	}
	payload := statePayload{
		Next:      sanitizeNextPath(next),
		Nonce:     nonce,
		Verifier:  v,
		ExpiredAt: time.Now().Add(StateTTL).Unix(),
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	encoded := base64.RawURLEncoding.EncodeToString(body)
	return encoded + "." + signState(encoded, secret), nil
}

func ParseState(state string) (string, string, error) {
	secret := stateSecret()
	if secret == "" {
		return "", "", errorsx.UnauthorizedI18n("error.e0046")
	}
	parts := strings.Split(strings.TrimSpace(state), ".")
	if len(parts) != 2 {
		return "", "", errorsx.UnauthorizedI18n("error.e0046")
	}
	if !hmac.Equal([]byte(parts[1]), []byte(signState(parts[0], secret))) {
		return "", "", errorsx.UnauthorizedI18n("error.e0046")
	}
	body, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return "", "", errorsx.UnauthorizedI18n("error.e0046")
	}
	payload := statePayload{}
	if err = json.Unmarshal(body, &payload); err != nil {
		return "", "", errorsx.UnauthorizedI18n("error.e0046")
	}
	if payload.ExpiredAt <= time.Now().Unix() {
		return "", "", errorsx.UnauthorizedI18n("error.e0046")
	}
	return sanitizeNextPath(payload.Next), payload.Verifier, nil
}

func IssueLoginTicket(loginResp *response.LoginResponse) (string, error) {
	if loginResp == nil {
		return "", i18nx.Errorf("error.e0272")
	}
	ticket, err := randomToken("olt_")
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

func profileFromIDToken(idToken *gooidc.IDToken) (*Profile, error) {
	var claims map[string]any
	if err := idToken.Claims(&claims); err != nil {
		return nil, err
	}
	raw, _ := json.Marshal(claims)
	profile := &Profile{
		Subject:           claimString(claims, "sub"),
		Email:             claimString(claims, "email"),
		PreferredUsername: firstNonEmpty(claimString(claims, "preferred_username"), claimString(claims, "user_name"), claimString(claims, "nickname")),
		Name:              firstNonEmpty(claimString(claims, "name"), claimString(claims, "full_name")),
		Picture:           firstNonEmpty(claimString(claims, "picture"), claimString(claims, "avatar_url")),
		ActiveOrgID:       firstNonEmpty(claimString(claims, "active_org_id"), claimString(claims, "activeOrgId")),
		Organizations:     claimOrganizations(claims),
		RawProfile:        string(raw),
	}
	if strings.TrimSpace(profile.Subject) == "" {
		return nil, errorsx.UnauthorizedI18n("error.e0043")
	}
	return profile, nil
}

func profileFromUserInfo(userInfo *gooidc.UserInfo, fallback *Profile) (*Profile, error) {
	var claims map[string]any
	if err := userInfo.Claims(&claims); err != nil {
		return nil, err
	}
	raw, _ := json.Marshal(claims)
	profile := &Profile{
		Subject:           strings.TrimSpace(userInfo.Subject),
		Email:             firstNonEmpty(claimString(claims, "email"), userInfo.Email),
		PreferredUsername: firstNonEmpty(claimString(claims, "preferred_username"), claimString(claims, "user_name"), claimString(claims, "nickname")),
		Name:              firstNonEmpty(claimString(claims, "name"), claimString(claims, "full_name")),
		Picture:           firstNonEmpty(claimString(claims, "picture"), claimString(claims, "avatar_url")),
		ActiveOrgID:       firstNonEmpty(claimString(claims, "active_org_id"), claimString(claims, "activeOrgId")),
		Organizations:     claimOrganizations(claims),
		RawProfile:        string(raw),
	}
	if fallback != nil {
		profile.Email = firstNonEmpty(profile.Email, fallback.Email)
		profile.PreferredUsername = firstNonEmpty(profile.PreferredUsername, fallback.PreferredUsername)
		profile.Name = firstNonEmpty(profile.Name, fallback.Name)
		profile.Picture = firstNonEmpty(profile.Picture, fallback.Picture)
		if profile.ActiveOrgID == "" {
			profile.ActiveOrgID = fallback.ActiveOrgID
		}
		if len(profile.Organizations) == 0 {
			profile.Organizations = fallback.Organizations
		}
	}
	if profile.RawProfile == "" {
		if fallback != nil {
			profile.RawProfile = fallback.RawProfile
		}
	}
	if strings.TrimSpace(profile.Subject) == "" {
		return nil, errorsx.UnauthorizedI18n("error.e0043")
	}
	return profile, nil
}

func claimOrganizations(claims map[string]any) []OrganizationClaim {
	raw, ok := claims["organizations"]
	if !ok || raw == nil {
		raw, ok = claims["orgs"]
	}
	if !ok || raw == nil {
		return nil
	}

	bytes, err := json.Marshal(raw)
	if err != nil {
		return nil
	}
	var orgs []OrganizationClaim
	if err := json.Unmarshal(bytes, &orgs); err == nil && len(orgs) > 0 {
		return orgs
	}

	var list []map[string]any
	if err := json.Unmarshal(bytes, &list); err == nil {
		for _, item := range list {
			id := firstNonEmpty(claimString(item, "id"), claimString(item, "org_id"), claimString(item, "slug"), claimString(item, "code"))
			slug := claimString(item, "slug")
			name := firstNonEmpty(claimString(item, "name"), claimString(item, "org_name"), slug, id)
			role := firstNonEmpty(claimString(item, "role"), "MEMBER")
			if id != "" {
				orgs = append(orgs, OrganizationClaim{
					ID:   id,
					Slug: slug,
					Name: name,
					Role: strings.ToUpper(role),
				})
			}
		}
	}
	return orgs
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}

func claimString(claims map[string]any, key string) string {
	value, _ := claims[key].(string)
	return strings.TrimSpace(value)
}

func stateSecret() string {
	if strings.TrimSpace(oidcCfg.StateSecret) != "" {
		return strings.TrimSpace(oidcCfg.StateSecret)
	}
	return strings.TrimSpace(oidcCfg.ClientSecret)
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
