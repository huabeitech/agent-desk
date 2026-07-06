package dashboard

import (
	"bytes"
	"encoding/csv"
	"net/http"
	"strconv"
	"strings"
	"time"

	"agent-desk/internal/builders"
	"agent-desk/internal/models"
	"agent-desk/internal/pkg/constants"
	"agent-desk/internal/pkg/dto/request"
	"agent-desk/internal/pkg/dto/response"
	"agent-desk/internal/pkg/enums"
	"agent-desk/internal/pkg/httpx"
	"agent-desk/internal/pkg/httpx/params"
	"agent-desk/internal/pkg/utils"
	"agent-desk/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/mlogclub/simple/web"
)

func SalesLeadPostList(ctx *gin.Context) {
	if _, err := services.AuthService.RequirePermission(ctx, constants.PermissionSalesLeadView); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	req := request.SalesLeadListRequest{}
	if err := params.ReadJSON(ctx, &req); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	list, paging := services.SalesLeadService.List(req)
	httpx.WriteJSON(ctx, &web.PageResult{Results: builders.BuildSalesLeadList(list), Page: paging})
}

func SalesLeadGetExport(ctx *gin.Context) {
	if _, err := services.AuthService.RequirePermission(ctx, constants.PermissionSalesLeadView); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	req := request.SalesLeadListRequest{
		Keyword:           strings.TrimSpace(ctx.Query("keyword")),
		Status:            strings.TrimSpace(ctx.Query("status")),
		Intent:            strings.TrimSpace(ctx.Query("intent")),
		TaskView:          strings.TrimSpace(ctx.Query("taskView")),
		FollowUpStatus:    strings.TrimSpace(ctx.Query("followUpStatus")),
		AppointmentStatus: strings.TrimSpace(ctx.Query("appointmentStatus")),
	}
	if ownerUserID, ok := params.GetInt64(ctx, "ownerUserId"); ok && ownerUserID != 0 {
		req.OwnerUserID = &ownerUserID
	}

	var buffer bytes.Buffer
	buffer.WriteString("\xEF\xBB\xBF")
	writer := csv.NewWriter(&buffer)
	if err := writer.Write(salesLeadExportHeaders()); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	ownerNames := map[int64]string{}
	for _, item := range services.SalesLeadService.Export(req) {
		if err := writer.Write(salesLeadExportRow(item, ownerNames)); err != nil {
			httpx.WriteJSON(ctx, err)
			return
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}

	filename := "sales-leads-" + time.Now().Format("20060102150405") + ".csv"
	ctx.Header("Content-Type", "text/csv; charset=utf-8")
	ctx.Header("Content-Disposition", `attachment; filename="`+filename+`"`)
	ctx.Data(http.StatusOK, "text/csv; charset=utf-8", buffer.Bytes())
}

func SalesLeadGetBy(ctx *gin.Context) {
	if _, err := services.AuthService.RequirePermission(ctx, constants.PermissionSalesLeadView); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	id, ok := httpx.GetPathInt64(ctx, "id")
	if !ok {
		return
	}
	item := services.SalesLeadService.Get(id)
	if item == nil {
		httpx.WriteJSON(ctx, nil)
		return
	}
	lead := builders.BuildSalesLead(item)
	followUps := services.SalesLeadService.FindFollowUps(id)
	detail := response.SalesLeadDetailResponse{
		Lead:           *lead,
		FollowUps:      builders.BuildLeadFollowUps(followUps),
		FollowUpAdvice: services.SalesLeadService.BuildFollowUpAdvice(item, followUps),
	}
	httpx.WriteJSON(ctx, &detail)
}

func SalesLeadPostUpdate(ctx *gin.Context) {
	operator, err := services.AuthService.RequirePermission(ctx, constants.PermissionSalesLeadUpdate)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	req := request.UpdateSalesLeadRequest{}
	if err := params.ReadJSON(ctx, &req); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	if err := services.SalesLeadService.Update(req, operator); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	httpx.WriteJSON(ctx, nil)
}

func SalesLeadPostUpdateStatus(ctx *gin.Context) {
	operator, err := services.AuthService.RequirePermission(ctx, constants.PermissionSalesLeadUpdate)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	req := request.UpdateSalesLeadStatusRequest{}
	if err := params.ReadJSON(ctx, &req); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	if err := services.SalesLeadService.UpdateStatus(req, operator); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	httpx.WriteJSON(ctx, nil)
}

func SalesLeadPostCrmSync(ctx *gin.Context) {
	operator, err := services.AuthService.RequirePermission(ctx, constants.PermissionSalesLeadView)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	req := request.SyncSalesLeadToCRMRequest{}
	if err := params.ReadJSON(ctx, &req); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	ret, err := services.SalesLeadService.SyncToCRM(req, operator)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	httpx.WriteJSON(ctx, ret)
}

func SalesLeadPostAssign(ctx *gin.Context) {
	operator, err := services.AuthService.RequirePermission(ctx, constants.PermissionSalesLeadAssign)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	req := request.AssignSalesLeadRequest{}
	if err := params.ReadJSON(ctx, &req); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	if err := services.SalesLeadService.Assign(req, operator); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	httpx.WriteJSON(ctx, nil)
}

func SalesLeadPostClaimUnassigned(ctx *gin.Context) {
	operator, err := services.AuthService.RequirePermission(ctx, constants.PermissionSalesLeadAssign)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	req := request.ClaimUnassignedSalesLeadsRequest{}
	if err := params.ReadJSON(ctx, &req); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	ret, err := services.SalesLeadService.ClaimUnassigned(req, operator)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	httpx.WriteJSON(ctx, ret)
}

func SalesLeadPostFollowUpCreate(ctx *gin.Context) {
	operator, err := services.AuthService.RequirePermission(ctx, constants.PermissionSalesLeadFollowUp)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	req := request.CreateLeadFollowUpRequest{}
	if err := params.ReadJSON(ctx, &req); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	item, err := services.SalesLeadService.CreateFollowUp(req, operator)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	httpx.WriteJSON(ctx, builders.BuildLeadFollowUp(item))
}

func SalesLeadPostFollowUpReminderSummary(ctx *gin.Context) {
	if _, err := services.AuthService.RequirePermission(ctx, constants.PermissionSalesLeadView); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	req := request.SalesLeadFollowUpReminderRequest{}
	if err := params.ReadJSON(ctx, &req); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	httpx.WriteJSON(ctx, services.SalesLeadService.GetFollowUpReminderSummary(req))
}

func SalesLeadPostAppointmentSummary(ctx *gin.Context) {
	if _, err := services.AuthService.RequirePermission(ctx, constants.PermissionSalesLeadView); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	req := request.SalesLeadAppointmentSummaryRequest{}
	if err := params.ReadJSON(ctx, &req); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	httpx.WriteJSON(ctx, services.SalesLeadService.GetAppointmentSummary(req))
}

func SalesLeadPostAppointmentReminderSend(ctx *gin.Context) {
	operator, err := services.AuthService.RequirePermission(ctx, constants.PermissionSalesLeadFollowUp)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	req := request.SalesLeadAppointmentSummaryRequest{}
	if err := params.ReadJSON(ctx, &req); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	ret, err := services.SalesLeadService.SendAppointmentReminder(req, operator)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	httpx.WriteJSON(ctx, ret)
}

func SalesLeadPostFollowUpReminderSend(ctx *gin.Context) {
	operator, err := services.AuthService.RequirePermission(ctx, constants.PermissionSalesLeadFollowUp)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	req := request.SalesLeadFollowUpReminderRequest{}
	if err := params.ReadJSON(ctx, &req); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	ret, err := services.SalesLeadService.SendFollowUpReminder(req, operator)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	httpx.WriteJSON(ctx, ret)
}

func salesLeadExportHeaders() []string {
	return []string{
		"线索ID",
		"客户姓名",
		"手机号",
		"微信",
		"城市",
		"地址/小区",
		"预算下限",
		"预算上限",
		"意向产品",
		"需求摘要",
		"意向等级",
		"购买阶段",
		"预约时间",
		"预约时间描述",
		"预约门店",
		"到店人数",
		"预约备注",
		"来源渠道",
		"负责人ID",
		"负责人",
		"状态",
		"下次跟进",
		"会话ID",
		"归并方式",
		"归并说明",
		"归并时间",
		"创建时间",
		"更新时间",
		"备注",
	}
}

func salesLeadExportRow(item models.SalesLead, ownerNames map[int64]string) []string {
	return []string{
		strconv.FormatInt(item.ID, 10),
		item.CustomerName,
		item.Phone,
		item.WeChat,
		item.City,
		item.AddressHint,
		formatInt64OrBlank(item.BudgetMin),
		formatInt64OrBlank(item.BudgetMax),
		item.InterestedProducts,
		item.DemandSummary,
		salesLeadIntentLabel(item.IntentLevel),
		salesLeadStageLabel(item.BuyingStage),
		utils.FormatTimePtr(item.AppointmentAt),
		item.AppointmentTimeText,
		item.AppointmentStore,
		formatIntOrBlank(item.AppointmentPeople),
		item.AppointmentRemark,
		item.SourceChannel,
		formatInt64OrBlank(item.OwnerUserID),
		salesLeadOwnerName(item.OwnerUserID, ownerNames),
		salesLeadStatusLabel(item.Status),
		utils.FormatTimePtr(item.NextFollowUpAt),
		formatInt64OrBlank(item.ConversationID),
		salesLeadMergeKeyLabel(item.MergeKey),
		item.MergeReason,
		utils.FormatTimePtr(item.MergedAt),
		utils.FormatTime(item.CreatedAt),
		utils.FormatTime(item.UpdatedAt),
		item.Remark,
	}
}

func salesLeadMergeKeyLabel(value string) string {
	switch strings.TrimSpace(value) {
	case "new":
		return "新建"
	case "conversation":
		return "同会话"
	case "phone":
		return "同手机号"
	case "wechat":
		return "同微信"
	case "customer":
		return "同客户"
	default:
		return strings.TrimSpace(value)
	}
}

func salesLeadOwnerName(ownerUserID int64, ownerNames map[int64]string) string {
	if ownerUserID <= 0 {
		return ""
	}
	if value, ok := ownerNames[ownerUserID]; ok {
		return value
	}
	if owner := services.UserService.Get(ownerUserID); owner != nil {
		ownerNames[ownerUserID] = owner.Username
		return owner.Username
	}
	ownerNames[ownerUserID] = ""
	return ""
}

func formatInt64OrBlank(value int64) string {
	if value <= 0 {
		return ""
	}
	return strconv.FormatInt(value, 10)
}

func formatIntOrBlank(value int) string {
	if value <= 0 {
		return ""
	}
	return strconv.Itoa(value)
}

func salesLeadIntentLabel(value enums.SalesLeadIntent) string {
	switch value {
	case enums.SalesLeadIntentHigh:
		return "高意向"
	case enums.SalesLeadIntentMedium:
		return "中意向"
	case enums.SalesLeadIntentLow:
		return "低意向"
	default:
		return "未知"
	}
}

func salesLeadStageLabel(value enums.SalesLeadStage) string {
	switch value {
	case enums.SalesLeadStageConsulting:
		return "咨询了解"
	case enums.SalesLeadStageComparing:
		return "对比决策"
	case enums.SalesLeadStageAppointment:
		return "预约到店"
	case enums.SalesLeadStageReadyToBuy:
		return "准备购买"
	case enums.SalesLeadStageAfterSales:
		return "售后问题"
	default:
		return "未知"
	}
}

func salesLeadStatusLabel(value enums.SalesLeadStatus) string {
	switch value {
	case enums.SalesLeadStatusNew:
		return "新线索"
	case enums.SalesLeadStatusFollowing:
		return "跟进中"
	case enums.SalesLeadStatusVisited:
		return "已到店"
	case enums.SalesLeadStatusConverted:
		return "已转化"
	case enums.SalesLeadStatusInvalid:
		return "无效"
	case enums.SalesLeadStatusClosed:
		return "已关闭"
	default:
		return string(value)
	}
}
