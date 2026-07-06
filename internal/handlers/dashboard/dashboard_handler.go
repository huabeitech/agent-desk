package dashboard

import (
	"agent-desk/internal/pkg/constants"
	"agent-desk/internal/pkg/httpx"
	"agent-desk/internal/services"

	"agent-desk/internal/pkg/httpx/params"
	"agent-desk/internal/pkg/i18nx"

	"github.com/gin-gonic/gin"
)

func DashboardGetOverview(ctx *gin.Context) {
	rangeValue, _ := params.Get(ctx, "range")
	httpx.WriteJSON(ctx, services.DashboardService.GetOverview(rangeValue, i18nx.Locale(ctx)))
}

func DashboardGetDailyBusinessReport(ctx *gin.Context) {
	if _, err := services.AuthService.RequirePermission(ctx, constants.PermissionSalesLeadView); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	dateValue, _ := params.Get(ctx, "date")
	httpx.WriteJSON(ctx, services.DashboardService.GetDailyBusinessReport(dateValue, i18nx.Locale(ctx)))
}

func DashboardPostDailyBusinessReportSend(ctx *gin.Context) {
	operator, err := services.AuthService.RequirePermission(ctx, constants.PermissionSalesLeadView)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	dateValue, _ := params.Get(ctx, "date")
	resp, err := services.DashboardService.SendDailyBusinessReportWebhook(dateValue, i18nx.Locale(ctx), operator.UserID)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	httpx.WriteJSON(ctx, resp)
}

func DashboardGetAIQualityReport(ctx *gin.Context) {
	if _, err := services.AuthService.RequirePermission(ctx, constants.PermissionSalesLeadView); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	rangeValue, _ := params.Get(ctx, "range")
	httpx.WriteJSON(ctx, services.DashboardService.GetAIQualityReport(rangeValue, i18nx.Locale(ctx)))
}

func DashboardGetSalesFunnelReport(ctx *gin.Context) {
	if _, err := services.AuthService.RequirePermission(ctx, constants.PermissionSalesLeadView); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	rangeValue, _ := params.Get(ctx, "range")
	httpx.WriteJSON(ctx, services.DashboardService.GetSalesFunnelReport(rangeValue, i18nx.Locale(ctx)))
}

func DashboardGetBusinessTrendReport(ctx *gin.Context) {
	if _, err := services.AuthService.RequirePermission(ctx, constants.PermissionSalesLeadView); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	rangeValue, _ := params.Get(ctx, "range")
	httpx.WriteJSON(ctx, services.DashboardService.GetBusinessTrendReport(rangeValue, i18nx.Locale(ctx)))
}

func DashboardGetABTestReport(ctx *gin.Context) {
	if _, err := services.AuthService.RequirePermission(ctx, constants.PermissionSalesLeadView); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	rangeValue, _ := params.Get(ctx, "range")
	httpx.WriteJSON(ctx, services.DashboardService.GetABTestReport(rangeValue, i18nx.Locale(ctx)))
}
