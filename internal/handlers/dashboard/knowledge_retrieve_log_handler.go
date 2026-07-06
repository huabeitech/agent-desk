package dashboard

import (
	"agent-desk/internal/builders"
	"agent-desk/internal/pkg/constants"
	"agent-desk/internal/pkg/dto/request"
	"agent-desk/internal/pkg/dto/response"
	"agent-desk/internal/pkg/httpx"
	"agent-desk/internal/services"

	"agent-desk/internal/pkg/httpx/params"

	"github.com/gin-gonic/gin"
	"github.com/mlogclub/simple/web"
)

func KnowledgeRetrieveLogAnyList(ctx *gin.Context) {
	if _, err := services.AuthService.RequirePermission(ctx, constants.PermissionKnowledgeDocumentView); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}

	cnd := params.NewPagedSqlCnd(ctx,
		params.QueryFilter{ParamName: "knowledgeBaseId"},
		params.QueryFilter{ParamName: "question", Op: params.Like},
		params.QueryFilter{ParamName: "channel"},
		params.QueryFilter{ParamName: "scene"},
		params.QueryFilter{ParamName: "chunkProvider"},
	).Desc("id")

	if answerStatus, ok := params.GetInt64(ctx, "answerStatus"); ok && answerStatus > 0 {
		cnd.Where("answer_status = ?", answerStatus)
	}
	if rerankEnabled, ok := params.GetInt64(ctx, "rerankEnabled"); ok {
		cnd.Where("rerank_enabled = ?", rerankEnabled > 0)
	}
	if feedbackState, ok := params.Get(ctx, "feedbackState"); ok {
		services.KnowledgeRetrieveLogService.ApplyFeedbackStateFilter(cnd, feedbackState)
	}

	queryParams := params.NewQueryParams(ctx)
	queryParams.Cnd = *cnd
	list, paging := services.KnowledgeRetrieveLogService.FindPageByParams(queryParams)
	ids := make([]int64, 0, len(list))
	for _, item := range list {
		ids = append(ids, item.ID)
	}
	feedbackSummaries := services.KnowledgeRetrieveLogService.FindFeedbackSummariesByRetrieveLogIDs(ids)
	results := make([]response.KnowledgeRetrieveLogResponse, 0, len(list))
	for _, item := range list {
		resp := builders.BuildKnowledgeRetrieveLog(&item)
		if knowledgeBase := services.KnowledgeBaseService.Get(item.KnowledgeBaseID); knowledgeBase != nil {
			resp.KnowledgeBaseName = knowledgeBase.Name
		}
		if summary, ok := feedbackSummaries[item.ID]; ok {
			resp.FeedbackCount = summary.FeedbackCount
			resp.NegativeFeedbackCount = summary.NegativeFeedbackCount
			resp.LatestFeedbackType = summary.LatestFeedbackType
			resp.LatestFeedbackTypeName = summary.LatestFeedbackTypeName
			resp.LatestFeedbackReason = summary.LatestFeedbackReason
		}
		results = append(results, resp)
	}
	httpx.WriteJSON(ctx, &web.PageResult{Results: results, Page: paging})
}

func KnowledgeRetrieveLogGetBy(ctx *gin.Context) {
	id, ok := httpx.GetPathInt64(ctx, "id")
	if !ok {
		return
	}
	if _, err := services.AuthService.RequirePermission(ctx, constants.PermissionKnowledgeDocumentView); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}

	logItem := services.KnowledgeRetrieveLogService.Get(id)
	if logItem == nil {
		httpx.WriteJSON(ctx, httpx.JsonErrorMsg(ctx, "error.e0241"))
		return
	}

	logResp := builders.BuildKnowledgeRetrieveLog(logItem)
	if knowledgeBase := services.KnowledgeBaseService.Get(logItem.KnowledgeBaseID); knowledgeBase != nil {
		logResp.KnowledgeBaseName = knowledgeBase.Name
	}

	hits := services.KnowledgeRetrieveLogService.FindHitsByRetrieveLogID(id)
	hitResults := make([]response.KnowledgeRetrieveHitResponse, 0, len(hits))
	for _, item := range hits {
		hitResults = append(hitResults, builders.BuildKnowledgeRetrieveHitResponse(&item))
	}

	feedbacks := services.KnowledgeRetrieveLogService.FindFeedbacksByRetrieveLogID(id)
	feedbackResults := make([]response.KnowledgeFeedbackResponse, 0, len(feedbacks))
	for _, item := range feedbacks {
		feedbackResults = append(feedbackResults, builders.BuildKnowledgeFeedback(&item))
	}

	httpx.WriteJSON(ctx, response.KnowledgeRetrieveLogDetailResponse{
		Log:       logResp,
		Hits:      hitResults,
		Feedbacks: feedbackResults,
	})
}

func KnowledgeRetrieveLogPostFeedback_create(ctx *gin.Context) {
	operator, err := services.AuthService.RequirePermission(ctx, constants.PermissionKnowledgeDocumentView)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}

	req := request.CreateKnowledgeFeedbackRequest{}
	if err := params.ReadJSON(ctx, &req); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	item, err := services.KnowledgeRetrieveLogService.CreateFeedback(req, operator)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	httpx.WriteJSON(ctx, builders.BuildKnowledgeFeedback(item))
}

func KnowledgeRetrieveLogPostFaq_draft_create(ctx *gin.Context) {
	operator, err := services.AuthService.RequirePermission(ctx, constants.PermissionKnowledgeFAQCreate)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}

	req := request.CreateKnowledgeFAQDraftFromRetrieveLogRequest{}
	if err := params.ReadJSON(ctx, &req); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	item, err := services.KnowledgeFAQService.CreateDraftFromRetrieveLog(req, operator)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	httpx.WriteJSON(ctx, builders.BuildKnowledgeFAQ(item))
}

func KnowledgeRetrieveLogPostFaq_draft_batch_create(ctx *gin.Context) {
	operator, err := services.AuthService.RequirePermission(ctx, constants.PermissionKnowledgeFAQCreate)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}

	req := request.BatchCreateKnowledgeFAQDraftsFromRetrieveLogsRequest{}
	if err := params.ReadJSON(ctx, &req); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	ret, err := services.KnowledgeFAQService.BatchCreateDraftsFromRetrieveLogs(req, operator)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	httpx.WriteJSON(ctx, ret)
}
