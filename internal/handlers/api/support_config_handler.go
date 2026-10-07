package api

import (
	"agent-desk/internal/pkg/httpx"
	"agent-desk/internal/services"

	"github.com/gin-gonic/gin"
)

func SupportConfigGetConfig(ctx *gin.Context) {
	httpx.WriteJSON(ctx, services.SystemConfigService.GetPublicSupportConfig())
}

func SupportConfigGetAICustomerServiceUserToken(ctx *gin.Context) {
	principal, err := services.AuthService.Authenticate(ctx)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	channel := services.SystemConfigService.GetPublicSupportAICustomerServiceChannel()
	if channel == nil {
		httpx.WriteJSON(ctx, httpx.JsonErrorMsg(ctx, "error.e0209"))
		return
	}
	user := services.UserService.Get(principal.UserID)
	if user == nil {
		httpx.WriteJSON(ctx, httpx.JsonErrorMsg(ctx, "error.e0256"))
		return
	}
	token, err := services.CustomerSessionService.SignSupportUserToken(channel, user)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	httpx.WriteJSON(ctx, token)
}
