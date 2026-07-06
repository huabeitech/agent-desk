package dashboard

import (
	"agent-desk/internal/builders"
	"agent-desk/internal/pkg/constants"
	"agent-desk/internal/pkg/dto/request"
	"agent-desk/internal/pkg/dto/response"
	"agent-desk/internal/pkg/enums"
	"agent-desk/internal/pkg/httpx"
	"agent-desk/internal/pkg/i18nx"
	"agent-desk/internal/services"
	"strconv"
	"strings"

	"agent-desk/internal/pkg/httpx/params"

	"github.com/gin-gonic/gin"
	"github.com/mlogclub/simple/common/strs"
	"github.com/mlogclub/simple/web"
	"github.com/spf13/cast"
)

func ConversationAnyList(ctx *gin.Context) {
	if _, err := services.AuthService.RequirePermission(ctx, constants.PermissionConversationView); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}

	cnd := params.NewPagedSqlCnd(ctx,
		params.QueryFilter{ParamName: "status"},
		params.QueryFilter{ParamName: "serviceMode"},
		params.QueryFilter{ParamName: "currentAssigneeId"},
	).Desc("last_message_at").Desc("id")

	paging := params.GetPaging(ctx)

	if keyword, _ := params.Get(ctx, "keyword"); strs.IsNotBlank(keyword) {
		keywordLike := "%" + strings.TrimSpace(keyword) + "%"
		cnd.Where("customer_name LIKE ? OR last_message_summary LIKE ?", keywordLike, keywordLike)
	}

	// 标签搜索
	if tagID, _ := params.GetInt64(ctx, "tagId"); tagID > 0 {
		tagIDs := services.TagService.GetSelfAndDescendantIDs(tagID)
		if len(tagIDs) == 0 {
			httpx.WriteJSON(ctx, &web.PageResult{
				Results: []response.ConversationResponse{},
				Page:    paging,
			})
			return
		}
		cnd.Where("id IN (SELECT conversation_id FROM conversation_tag_rels WHERE tag_id IN (?))", tagIDs)
	}
	if agentTeamID, _ := params.GetInt64(ctx, "agentTeamId"); agentTeamID > 0 {
		userIDs := services.AgentProfileService.GetUserIDsByTeamID(agentTeamID)
		if len(userIDs) == 0 {
			httpx.WriteJSON(ctx, &web.PageResult{
				Results: []response.ConversationResponse{},
				Page:    paging,
			})
			return
		}
		cnd.In("current_assignee_id", userIDs)
	}

	list, paging := services.ConversationService.FindPageByCnd(cnd)
	results := make([]response.ConversationResponse, 0, len(list))
	for _, item := range list {
		results = append(results, builders.BuildConversationWithLocale(&item, i18nx.Locale(ctx)))
	}
	httpx.WriteJSON(ctx, &web.PageResult{Results: results, Page: paging})
}

func ConversationAnyConversations(ctx *gin.Context) {
	operator, err := services.AuthService.RequirePermission(ctx, constants.PermissionConversationView)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}

	filterValue, _ := params.Get(ctx, "filter")
	keyword, _ := params.Get(ctx, "keyword")
	paging := params.GetPaging(ctx)

	list, paging, err := services.ConversationService.ListConversations(
		operator.UserID,
		request.AgentConversationFilter(strings.TrimSpace(filterValue)),
		keyword,
		paging,
	)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}

	results := make([]response.ConversationResponse, 0, len(list))
	for _, item := range list {
		results = append(results, builders.BuildConversationWithLocale(&item, i18nx.Locale(ctx)))
	}
	httpx.WriteJSON(ctx, &web.PageResult{Results: results, Page: paging})
}

func ConversationGetBy(ctx *gin.Context) {
	id, ok := httpx.GetPathInt64(ctx, "id")
	if !ok {
		return
	}
	if _, err := services.AuthService.RequirePermission(ctx, constants.PermissionConversationView); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}

	item := services.ConversationService.Get(id)
	if item == nil {
		httpx.WriteJSON(ctx, httpx.JsonErrorMsg(ctx, "error.e0116"))
		return
	}

	detail := response.ConversationDetailResponse{
		ConversationResponse: builders.BuildConversationWithLocale(item, i18nx.Locale(ctx)),
		Participants:         builders.BuildParticipantResponses(id),
	}
	httpx.WriteJSON(ctx, detail)
}

func ConversationAnyMessage_list(ctx *gin.Context) {
	if _, err := services.AuthService.RequirePermission(ctx, constants.PermissionConversationView); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}

	var (
		conversationID, _ = params.GetInt64(ctx, "conversationId")
		senderType, _     = params.Get(ctx, "senderType")
		messageType, _    = params.Get(ctx, "messageType")
		cursor, _         = params.GetInt64(ctx, "cursor")
		limit, _          = params.GetInt(ctx, "limit")
	)
	if conversation := services.ConversationService.Get(conversationID); conversation == nil {
		httpx.WriteJSON(ctx, httpx.JsonErrorMsg(ctx, "error.e0116"))
		return
	}

	list, nextCursor, hasMore := services.MessageService.FindByConversationIDCursor(
		conversationID, cursor, limit, senderType, messageType,
	)
	results := builders.BuildMessagesWithLocale(list, i18nx.Locale(ctx))

	httpx.WriteJSON(ctx, httpx.CursorData(results, cast.ToString(nextCursor), hasMore))
}

func ConversationPostAssign(ctx *gin.Context) {
	operator, err := services.AuthService.RequirePermission(ctx, constants.PermissionConversationAssign)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}

	req := request.AssignConversationRequest{}
	if err := params.ReadJSON(ctx, &req); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	if err := services.ConversationService.AssignConversation(req, operator); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	httpx.WriteJSON(ctx, nil)
}

func ConversationPostDispatch(ctx *gin.Context) {
	operator, err := services.AuthService.RequirePermission(ctx, constants.PermissionConversationAssign)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}

	req := request.DispatchConversationRequest{}
	if err := params.ReadJSON(ctx, &req); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	if err := services.ConversationService.AutoAssignConversation(req.ConversationID, operator); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	httpx.WriteJSON(ctx, nil)
}

func ConversationPostTransfer(ctx *gin.Context) {
	operator, err := services.AuthService.RequirePermission(ctx, constants.PermissionConversationTransfer)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}

	req := request.TransferConversationRequest{}
	if err := params.ReadJSON(ctx, &req); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	if err := services.ConversationService.TransferConversation(req.ConversationID, req.ToUserID, req.Reason, operator); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	httpx.WriteJSON(ctx, nil)
}

func ConversationPostClose(ctx *gin.Context) {
	operator, err := services.AuthService.RequirePermission(ctx, constants.PermissionConversationClose)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}

	req := request.CloseConversationRequest{}
	if err := params.ReadJSON(ctx, &req); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	if err := services.ConversationService.CloseConversation(req.ConversationID, req.CloseReason, operator); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	httpx.WriteJSON(ctx, nil)
}

func ConversationPostResume_ai(ctx *gin.Context) {
	operator, err := services.AuthService.RequirePermission(ctx, constants.PermissionConversationTransfer)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}

	req := request.ResumeAIConversationRequest{}
	if err := params.ReadJSON(ctx, &req); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	if err := services.ConversationService.ResumeAIConversation(req.ConversationID, req.Reason, operator); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	httpx.WriteJSON(ctx, nil)
}

func ConversationPostFollow_up_advice(ctx *gin.Context) {
	if _, err := services.AuthService.RequirePermission(ctx, constants.PermissionConversationView); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}

	req := request.ConversationFollowUpAdviceRequest{}
	if err := params.ReadJSON(ctx, &req); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	ret, err := services.ConversationService.BuildFollowUpAdvice(req.ConversationID)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	httpx.WriteJSON(ctx, ret)
}

func ConversationPostLink_customer(ctx *gin.Context) {
	operator, err := services.AuthService.RequirePermission(ctx, constants.PermissionConversationLinkCustomer)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	req := request.LinkConversationCustomerRequest{}
	if err := params.ReadJSON(ctx, &req); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	if err := services.ConversationService.LinkConversationCustomer(req.ConversationID, req.CustomerID, operator); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	httpx.WriteJSON(ctx, nil)
}

func ConversationPostSend_message(ctx *gin.Context) {
	operator, err := services.AuthService.RequirePermission(ctx, constants.PermissionConversationSend)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}

	req := request.SendConversationMessageRequest{}
	if err := params.ReadJSON(ctx, &req); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	item, err := services.MessageService.SendAgentMessageWithRequestID(req.ConversationID, 0, req.ClientMsgID, req.MessageType, req.Content, req.Payload, operator, httpx.GetRequestID(ctx))
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	httpx.WriteJSON(ctx, builders.BuildMessageWithLocale(item, i18nx.Locale(ctx)))
}

func ConversationPostRecall_message(ctx *gin.Context) {
	operator, err := services.AuthService.RequirePermission(ctx, constants.PermissionConversationSend)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}

	req := request.RecallConversationMessageRequest{}
	if err := params.ReadJSON(ctx, &req); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	item, err := services.MessageService.RecallAgentMessage(req.MessageID, operator)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	httpx.WriteJSON(ctx, builders.BuildMessageWithLocale(item, i18nx.Locale(ctx)))
}

func ConversationPostRead(ctx *gin.Context) {
	operator, err := services.AuthService.RequirePermission(ctx, constants.PermissionConversationView)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}

	req := request.ReadConversationRequest{}
	if err := params.ReadJSON(ctx, &req); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	if err := services.ConversationService.MarkAgentConversationReadToMessage(req.ConversationID, req.MessageID, operator); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	httpx.WriteJSON(ctx, nil)
}

func ConversationPostUpload_image(ctx *gin.Context) {
	operator, err := services.AuthService.RequirePermission(ctx, constants.PermissionConversationSend)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}

	rawConv := strings.TrimSpace(params.FormValue(ctx, "conversationId"))
	if rawConv == "" {
		httpx.WriteJSON(ctx, httpx.JsonErrorMsg(ctx, "error.e0064"))
		return
	}
	conversationID, err := strconv.ParseInt(rawConv, 10, 64)
	if err != nil || conversationID <= 0 {
		httpx.WriteJSON(ctx, httpx.JsonErrorMsg(ctx, "error.e0064"))
		return
	}
	if _, err := services.MessageService.ValidateConversationSender(conversationID, enums.IMSenderTypeAgent, operator, nil); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}

	header, err := ctx.FormFile("file")
	if err != nil {
		httpx.WriteJSON(ctx, httpx.JsonErrorMsg(ctx, "error.e0322"))
		return
	}
	if !strings.HasPrefix(strings.ToLower(header.Header.Get("Content-Type")), "image/") {
		httpx.WriteJSON(ctx, httpx.JsonErrorMsg(ctx, "error.e0090"))
		return
	}

	item, err := services.AssetService.UploadFile(header, "images", operator)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	httpx.WriteJSON(ctx, builders.BuildAsset(item))
}

func ConversationPostUpload_attachment(ctx *gin.Context) {
	operator, err := services.AuthService.RequirePermission(ctx, constants.PermissionConversationSend)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}

	rawConv := strings.TrimSpace(params.FormValue(ctx, "conversationId"))
	if rawConv == "" {
		httpx.WriteJSON(ctx, httpx.JsonErrorMsg(ctx, "error.e0064"))
		return
	}
	conversationID, err := strconv.ParseInt(rawConv, 10, 64)
	if err != nil || conversationID <= 0 {
		httpx.WriteJSON(ctx, httpx.JsonErrorMsg(ctx, "error.e0064"))
		return
	}
	if _, err := services.MessageService.ValidateConversationSender(conversationID, enums.IMSenderTypeAgent, operator, nil); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}

	header, err := ctx.FormFile("file")
	if err != nil {
		httpx.WriteJSON(ctx, httpx.JsonErrorMsg(ctx, "error.e0324"))
		return
	}
	item, err := services.AssetService.UploadFile(header, "attachments", operator)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	httpx.WriteJSON(ctx, builders.BuildAsset(item))
}

func ConversationPostAdd_tag(ctx *gin.Context) {
	operator, err := services.AuthService.RequirePermission(ctx, constants.PermissionConversationTag)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}

	req := request.AddConversationTagRequest{}
	if err := params.ReadJSON(ctx, &req); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	if err := services.ConversationTagService.AddTag(req, operator); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	httpx.WriteJSON(ctx, nil)
}

func ConversationPostRemove_tag(ctx *gin.Context) {
	if _, err := services.AuthService.RequirePermission(ctx, constants.PermissionConversationTag); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}

	req := request.RemoveConversationTagRequest{}
	if err := params.ReadJSON(ctx, &req); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	if err := services.ConversationTagService.RemoveTag(req); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	httpx.WriteJSON(ctx, nil)
}
