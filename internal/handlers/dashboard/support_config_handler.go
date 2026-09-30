package dashboard

import (
	"encoding/json"

	"agent-desk/internal/pkg/constants"
	"agent-desk/internal/pkg/errorsx"
	"agent-desk/internal/pkg/httpx"
	"agent-desk/internal/pkg/httpx/params"
	"agent-desk/internal/pkg/i18nx"
	"agent-desk/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/mlogclub/simple/web"
)

func SupportConfigGetConfig(ctx *gin.Context) {
	if _, err := services.AuthService.RequirePermission(ctx, constants.PermissionSupportConfigView); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	httpx.WriteJSON(ctx, services.SystemConfigService.GetDashboardSupportConfig())
}

func SupportConfigPostSave(ctx *gin.Context) {
	operator, err := services.AuthService.RequirePermission(ctx, constants.PermissionSupportConfigUpdate)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	req := map[string]json.RawMessage{}
	if err := params.ReadJSON(ctx, &req); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	config, err := services.SystemConfigService.SaveSupportConfig(req, operator)
	if err != nil {
		if validationErr, ok := err.(*services.SystemConfigValidationError); ok {
			locale := i18nx.Locale(ctx)
			httpx.WriteJSON(ctx, web.JsonErrorData(errorsx.CodeInvalidParam, validationErr.Message(locale), gin.H{
				"errors": validationErr.FieldErrorsLocale(locale),
			}))
			return
		}
		httpx.WriteJSON(ctx, err)
		return
	}
	httpx.WriteJSON(ctx, config)
}

func SystemConfigGetConfig(ctx *gin.Context) {
	if _, err := services.AuthService.RequirePermission(ctx, constants.PermissionSystemConfigView); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	httpx.WriteJSON(ctx, services.SystemConfigService.GetDashboardSystemConfig())
}

func SystemConfigPostSave(ctx *gin.Context) {
	operator, err := services.AuthService.RequirePermission(ctx, constants.PermissionSystemConfigUpdate)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	req := map[string]json.RawMessage{}
	if err := params.ReadJSON(ctx, &req); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	config, err := services.SystemConfigService.SaveSystemConfig(req, operator)
	if err != nil {
		if validationErr, ok := err.(*services.SystemConfigValidationError); ok {
			locale := i18nx.Locale(ctx)
			httpx.WriteJSON(ctx, web.JsonErrorData(errorsx.CodeInvalidParam, validationErr.Message(locale), gin.H{
				"errors": validationErr.FieldErrorsLocale(locale),
			}))
			return
		}
		httpx.WriteJSON(ctx, err)
		return
	}
	httpx.WriteJSON(ctx, config)
}
