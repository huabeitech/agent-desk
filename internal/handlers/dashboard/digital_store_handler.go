package dashboard

import (
	"agent-desk/internal/pkg/constants"
	"agent-desk/internal/pkg/dto/request"
	"agent-desk/internal/pkg/httpx"
	"agent-desk/internal/pkg/httpx/params"
	"agent-desk/internal/services"

	"github.com/gin-gonic/gin"
)

func DigitalStoreGetProfile(ctx *gin.Context) {
	if _, err := services.AuthService.RequirePermission(ctx, constants.PermissionDigitalStoreView); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	httpx.WriteJSON(ctx, services.DigitalStoreProfileService.GetProfile())
}

func DigitalStoreGetTemplates(ctx *gin.Context) {
	if _, err := services.AuthService.RequirePermission(ctx, constants.PermissionDigitalStoreView); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	httpx.WriteJSON(ctx, services.DigitalStoreProfileService.ListTemplates())
}

func DigitalStoreGetTemplate_export(ctx *gin.Context) {
	if _, err := services.AuthService.RequirePermission(ctx, constants.PermissionDigitalStoreView); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	templateCode, _ := params.Get(ctx, "templateCode")
	resp, err := services.DigitalStoreProfileService.ExportTemplate(templateCode)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	httpx.WriteJSON(ctx, resp)
}

func DigitalStoreGetTemplate_preview(ctx *gin.Context) {
	if _, err := services.AuthService.RequirePermission(ctx, constants.PermissionDigitalStoreView); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	templateCode, _ := params.Get(ctx, "templateCode")
	resp, err := services.DigitalStoreProfileService.PreviewTemplate(templateCode)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	httpx.WriteJSON(ctx, resp)
}

func DigitalStorePostTemplate_import_preview(ctx *gin.Context) {
	if _, err := services.AuthService.RequirePermission(ctx, constants.PermissionDigitalStoreView); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	req := request.DigitalStoreTemplateImportRequest{}
	if err := params.ReadJSON(ctx, &req); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	resp, err := services.DigitalStoreProfileService.PreviewImportedTemplate(req)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	httpx.WriteJSON(ctx, resp)
}

func DigitalStoreGetSetup_status(ctx *gin.Context) {
	if _, err := services.AuthService.RequirePermission(ctx, constants.PermissionDigitalStoreView); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	httpx.WriteJSON(ctx, services.DigitalStoreProfileService.GetSetupStatus())
}

func DigitalStoreGetMaintenance_status(ctx *gin.Context) {
	if _, err := services.AuthService.RequirePermission(ctx, constants.PermissionDigitalStoreView); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	httpx.WriteJSON(ctx, services.DigitalStoreProfileService.GetMaintenanceStatus())
}

func DigitalStoreGetKnowledge_assistant(ctx *gin.Context) {
	if _, err := services.AuthService.RequirePermission(ctx, constants.PermissionDigitalStoreView); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	httpx.WriteJSON(ctx, services.DigitalStoreProfileService.GetKnowledgeAssistant())
}

func DigitalStoreGetTemplate_effect(ctx *gin.Context) {
	if _, err := services.AuthService.RequirePermission(ctx, constants.PermissionDigitalStoreView); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	httpx.WriteJSON(ctx, services.DigitalStoreProfileService.GetTemplateEffect())
}

func DigitalStoreGetDelivery_report(ctx *gin.Context) {
	if _, err := services.AuthService.RequirePermission(ctx, constants.PermissionDigitalStoreView); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	publicBaseURL, _ := params.Get(ctx, "publicBaseUrl")
	httpx.WriteJSON(ctx, services.DigitalStoreProfileService.GetDeliveryReport(publicBaseURL))
}

func DigitalStoreGetDelivery_record_latest(ctx *gin.Context) {
	if _, err := services.AuthService.RequirePermission(ctx, constants.PermissionDigitalStoreView); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	httpx.WriteJSON(ctx, services.DigitalStoreProfileService.GetLatestDeliveryRecord())
}

func DigitalStorePostDelivery_record_create(ctx *gin.Context) {
	operator, err := services.AuthService.RequirePermission(ctx, constants.PermissionDigitalStoreUpdate)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	req := request.DigitalStoreDeliveryRecordCreateRequest{}
	if err := params.ReadJSON(ctx, &req); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	resp, err := services.DigitalStoreProfileService.CreateDeliveryRecord(req, operator)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	httpx.WriteJSON(ctx, resp)
}

func DigitalStorePostDelivery_record_acceptance_result(ctx *gin.Context) {
	operator, err := services.AuthService.RequirePermission(ctx, constants.PermissionDigitalStoreUpdate)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	req := request.DigitalStoreAcceptanceResultCreateRequest{}
	if err := params.ReadJSON(ctx, &req); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	resp, err := services.DigitalStoreProfileService.CreateAcceptanceResultRecord(req, operator)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	httpx.WriteJSON(ctx, resp)
}

func DigitalStorePostProfile(ctx *gin.Context) {
	operator, err := services.AuthService.RequirePermission(ctx, constants.PermissionDigitalStoreUpdate)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	req := request.DigitalStoreProfileRequest{}
	if err := params.ReadJSON(ctx, &req); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	resp, err := services.DigitalStoreProfileService.UpdateProfile(req, operator)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	httpx.WriteJSON(ctx, resp)
}

func DigitalStorePostEnsure_runtime(ctx *gin.Context) {
	operator, err := services.AuthService.RequirePermission(ctx, constants.PermissionDigitalStoreUpdate)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	resp, err := services.DigitalStoreProfileService.EnsureRuntime(operator)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	httpx.WriteJSON(ctx, resp)
}

func DigitalStorePostTest_webhook_notify(ctx *gin.Context) {
	operator, err := services.AuthService.RequirePermission(ctx, constants.PermissionDigitalStoreUpdate)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	resp, err := services.DigitalStoreProfileService.TestWebhookNotify(operator)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	httpx.WriteJSON(ctx, resp)
}

func DigitalStorePostTest_webhook_notify_scenarios(ctx *gin.Context) {
	operator, err := services.AuthService.RequirePermission(ctx, constants.PermissionDigitalStoreUpdate)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	resp, err := services.DigitalStoreProfileService.TestWebhookNotifyScenarios(operator)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	httpx.WriteJSON(ctx, resp)
}

func DigitalStorePostCleanup_demo_data(ctx *gin.Context) {
	operator, err := services.AuthService.RequirePermission(ctx, constants.PermissionDigitalStoreUpdate)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	resp, err := services.DigitalStoreProfileService.CleanupDemoData(operator)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	httpx.WriteJSON(ctx, resp)
}

func DigitalStorePostSeed_muse(ctx *gin.Context) {
	operator, err := services.AuthService.RequirePermission(ctx, constants.PermissionDigitalStoreUpdate)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	resp, err := services.DigitalStoreProfileService.SeedMuseProfile(operator)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	httpx.WriteJSON(ctx, resp)
}

func DigitalStorePostApply_template(ctx *gin.Context) {
	operator, err := services.AuthService.RequirePermission(ctx, constants.PermissionDigitalStoreUpdate)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	req := request.DigitalStoreApplyTemplateRequest{}
	if err := params.ReadJSON(ctx, &req); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	resp, err := services.DigitalStoreProfileService.ApplyTemplate(req.TemplateCode, operator)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	httpx.WriteJSON(ctx, resp)
}

func DigitalStorePostApply_imported_template(ctx *gin.Context) {
	operator, err := services.AuthService.RequirePermission(ctx, constants.PermissionDigitalStoreUpdate)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	req := request.DigitalStoreTemplateImportRequest{}
	if err := params.ReadJSON(ctx, &req); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	resp, err := services.DigitalStoreProfileService.ApplyImportedTemplate(req, operator)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	httpx.WriteJSON(ctx, resp)
}

func DigitalStorePostSync_knowledge(ctx *gin.Context) {
	if _, err := services.AuthService.RequirePermission(ctx, constants.PermissionDigitalStoreUpdate); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	if err := services.DigitalStoreProfileService.SyncKnowledgeFAQ(); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	httpx.WriteJSON(ctx, services.DigitalStoreProfileService.GetProfile())
}
