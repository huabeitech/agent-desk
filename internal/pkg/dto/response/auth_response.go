package response

import "agent-desk/internal/pkg/enums"

type AuthUserResponse struct {
	ID       int64          `json:"id"`
	Username string         `json:"username"`
	Nickname string         `json:"nickname"`
	Avatar   string         `json:"avatar"`
	Email    string         `json:"email,omitempty"`
	UserType enums.UserType `json:"userType"`
	Status   enums.Status   `json:"status"`
	Roles    []string       `json:"roles"`
}

type LoginResponse struct {
	AccessToken string            `json:"accessToken"`
	ExpiresAt   string            `json:"expiresAt"`
	User        *AuthUserResponse `json:"user"`
	Permissions []string          `json:"permissions"`
	Roles       []string          `json:"roles"`
}

type PublicConfigResponse struct {
	Language             string `json:"language"`
	CompanyName          string `json:"companyName,omitempty"`
	CompanyLogoURL       string `json:"companyLogoUrl,omitempty"`
	CompanyFaviconURL    string `json:"companyFaviconUrl,omitempty"`
	PasswordLoginEnabled bool   `json:"passwordLoginEnabled"`
	WxWorkEnabled        bool   `json:"wxworkEnabled"`
	OIDCEnabled          bool   `json:"oidcEnabled"`
}
