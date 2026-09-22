package api

import (
	"agent-desk/internal/builders"
	"agent-desk/internal/pkg/config"
	"agent-desk/internal/pkg/dto/request"
	"agent-desk/internal/pkg/dto/response"
	"agent-desk/internal/pkg/errorsx"
	"agent-desk/internal/pkg/httpx"
	"agent-desk/internal/pkg/httpx/params"
	"agent-desk/internal/services"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
)

func Login(ctx *gin.Context) {
	cfg := config.Current()
	if !cfg.Auth.IsPasswordLoginEnabled() {
		httpx.WriteJSON(ctx, errorsx.ForbiddenI18n("error.auth.passwordLoginDisabled"))
		return
	}
	req := request.LoginRequest{}
	if err := params.ReadJSON(ctx, &req); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}

	ret, err := services.AuthService.Login(req, cfg.Auth, ctx.ClientIP(), ctx.GetHeader("User-Agent"))
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	httpx.WriteJSON(ctx, ret)
}

func PublicConfig(ctx *gin.Context) {
	cfg := config.Current()
	httpx.WriteJSON(ctx, &response.PublicConfigResponse{
		Language:             cfg.LanguageOrDefault(),
		CompanyName:          cfg.Server.CompanyName,
		CompanyLogoURL:       cfg.Server.CompanyLogoURL,
		CompanyFaviconURL:    cfg.Server.CompanyFaviconURL,
		PasswordLoginEnabled: cfg.Auth.IsPasswordLoginEnabled(),
		WxWorkEnabled:        cfg.WxWork.Enabled,
		OIDCEnabled:          cfg.OIDC.Enabled,
	})
}

func WxWorkLogin(ctx *gin.Context) {
	loginURL, err := services.WxWorkLoginService.BuildWxWorkLoginURL(ctx.Query("next"))
	if err != nil {
		ctx.Redirect(http.StatusFound, "/login?wxworkError="+url.QueryEscape(wxWorkErrorMessage(err.Error())))
		return
	}
	ctx.Redirect(http.StatusFound, loginURL)
}

func WxWorkQRLogin(ctx *gin.Context) {
	loginURL, err := services.WxWorkLoginService.BuildWxWorkQRCodeLoginURL(ctx.Query("next"))
	if err != nil {
		ctx.Redirect(http.StatusFound, "/login?wxworkError="+url.QueryEscape(wxWorkErrorMessage(err.Error())))
		return
	}
	ctx.Redirect(http.StatusFound, loginURL)
}

func WxWorkCallback(ctx *gin.Context) {
	cfg := config.Current()
	ticket, next, err := services.WxWorkLoginService.LoginByWxWork(
		ctx.Query("code"),
		ctx.Query("state"),
		cfg.Auth,
		ctx.ClientIP(),
		ctx.GetHeader("User-Agent"),
	)
	if err != nil {
		ctx.Redirect(http.StatusFound, "/login?wxworkError="+url.QueryEscape(wxWorkErrorMessage(err.Error())))
		return
	}
	ctx.Redirect(http.StatusFound, "/dashboard/login/wxwork/callback?ticket="+url.QueryEscape(ticket)+"&next="+url.QueryEscape(next))
}

func WxWorkExchange(ctx *gin.Context) {
	req := request.WxWorkExchangeRequest{}
	if err := params.ReadJSON(ctx, &req); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	ret, err := services.WxWorkLoginService.ExchangeWxWorkLoginTicket(req.Ticket)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	httpx.WriteJSON(ctx, ret)
}

func OIDCLogin(ctx *gin.Context) {
	loginURL, err := services.OIDCLoginService.BuildOIDCLoginURL(ctx.Query("next"))
	if err != nil {
		ctx.Redirect(http.StatusFound, "/dashboard/login?oidcError="+url.QueryEscape(loginErrorMessage(err.Error())))
		return
	}
	ctx.Redirect(http.StatusFound, loginURL)
}

func OIDCCallback(ctx *gin.Context) {
	if oauthErr := ctx.Query("error"); oauthErr != "" {
		desc := ctx.Query("error_description")
		slog.Warn("oidc callback returned oauth error", "error", oauthErr, "description", desc)
		errMsg := oauthErr
		if desc != "" {
			errMsg += ": " + desc
		}
		ctx.Redirect(http.StatusFound, "/dashboard/login?oidcError="+url.QueryEscape(errMsg))
		return
	}

	code := ctx.Query("code")
	state := ctx.Query("state")
	if strings.TrimSpace(code) == "" {
		slog.Warn("oidc callback missing code", "rawQuery", ctx.Request.URL.RawQuery)
	}

	cfg := config.Current()
	ticket, next, err := services.OIDCLoginService.LoginByOIDC(
		ctx.Request.Context(),
		code,
		state,
		cfg.Auth,
		ctx.ClientIP(),
		ctx.GetHeader("User-Agent"),
	)
	if err != nil {
		slog.Error("oidc callback login failed", "error", err)
		ctx.Redirect(http.StatusFound, "/dashboard/login?oidcError="+url.QueryEscape(loginErrorMessage(err.Error())))
		return
	}
	ctx.Redirect(http.StatusFound, "/dashboard/login/oidc/callback?ticket="+url.QueryEscape(ticket)+"&next="+url.QueryEscape(next))
}

func OIDCExchange(ctx *gin.Context) {
	req := request.OIDCExchangeRequest{}
	if err := params.ReadJSON(ctx, &req); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	ret, err := services.OIDCLoginService.ExchangeOIDCLoginTicket(req.Ticket)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	httpx.WriteJSON(ctx, ret)
}

func Logout(ctx *gin.Context) {
	if err := services.AuthService.Logout(ctx.GetHeader("Authorization")); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	httpx.WriteJSON(ctx, nil)
}

func Profile(ctx *gin.Context) {
	ret, err := services.AuthService.CurrentProfile(ctx)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	httpx.WriteJSON(ctx, ret)
}

func UpdateProfile(ctx *gin.Context) {
	req := request.UpdateProfileRequest{}
	if err := params.ReadJSON(ctx, &req); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	ret, err := services.AuthService.UpdateProfile(ctx, req)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	httpx.WriteJSON(ctx, ret)
}

func UploadProfileAvatar(ctx *gin.Context) {
	principal, err := services.AuthService.Authenticate(ctx)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}

	header, err := ctx.FormFile("file")
	if err != nil {
		httpx.WriteJSON(ctx, httpx.JsonErrorMsg(ctx, "error.e0323"))
		return
	}
	if !strings.HasPrefix(strings.ToLower(header.Header.Get("Content-Type")), "image/") {
		httpx.WriteJSON(ctx, httpx.JsonErrorMsg(ctx, "error.e0090"))
		return
	}

	item, err := services.AssetService.UploadFile(header, "avatars", principal)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	httpx.WriteJSON(ctx, builders.BuildAsset(item))
}

func wxWorkErrorMessage(message string) string {
	return loginErrorMessage(message)
}

func loginErrorMessage(message string) string {
	if idx := strings.Index(message, ": "); idx >= 0 {
		message = message[idx+2:]
	}
	return message
}
