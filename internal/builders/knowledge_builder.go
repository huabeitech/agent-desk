package builders

import (
	"agent-desk/internal/models"
	"agent-desk/internal/pkg/dto/response"
	"agent-desk/internal/pkg/enums"
	"encoding/json"
)

func BuildKnowledgeBase(item *models.KnowledgeBase) response.KnowledgeBaseResponse {
	return response.KnowledgeBaseResponse{
		ID:                    item.ID,
		Name:                  item.Name,
		Description:           item.Description,
		KnowledgeType:         item.KnowledgeType,
		KnowledgeTypeName:     enums.GetKnowledgeBaseTypeLabel(enums.KnowledgeBaseType(item.KnowledgeType)),
		Status:                item.Status,
		StatusName:            enums.GetStatusLabel(item.Status),
		DefaultTopK:           item.DefaultTopK,
		DefaultScoreThreshold: item.DefaultScoreThreshold,
		DefaultRerankLimit:    item.DefaultRerankLimit,
		ChunkProvider:         item.ChunkProvider,
		ChunkTargetTokens:     item.ChunkTargetTokens,
		ChunkMaxTokens:        item.ChunkMaxTokens,
		ChunkOverlapTokens:    item.ChunkOverlapTokens,
		AnswerMode:            item.AnswerMode,
		AnswerModeName:        enums.GetKnowledgeAnswerModeLabel(enums.KnowledgeAnswerMode(item.AnswerMode)),
		Remark:                item.Remark,
		CreatedAt:             item.CreatedAt,
		UpdatedAt:             item.UpdatedAt,
		CreateUserName:        item.CreateUserName,
		UpdateUserName:        item.UpdateUserName,
	}
}

func BuildKnowledgeDocument(item *models.KnowledgeDocument) response.KnowledgeDocumentResponse {
	return response.KnowledgeDocumentResponse{
		ID:              item.ID,
		KnowledgeBaseID: item.KnowledgeBaseID,
		DirectoryID:     item.DirectoryID,
		Title:           item.Title,
		Status:          item.Status,
		StatusName:      enums.GetStatusLabel(item.Status),
		IndexStatus:     item.IndexStatus,
		IndexStatusName: enums.GetKnowledgeDocumentIndexStatusLabel(item.IndexStatus),
		IndexedAt:       item.IndexedAt,
		IndexError:      item.IndexError,
		ContentHash:     item.ContentHash,
		ContentType:     item.ContentType,
		Content:         item.Content,
		CreatedAt:       item.CreatedAt,
		UpdatedAt:       item.UpdatedAt,
		CreateUserName:  item.CreateUserName,
		UpdateUserName:  item.UpdateUserName,
	}
}

func BuildKnowledgeDocumentList(item *models.KnowledgeDocument) response.KnowledgeDocumentListResponse {
	return response.KnowledgeDocumentListResponse{
		ID:              item.ID,
		KnowledgeBaseID: item.KnowledgeBaseID,
		DirectoryID:     item.DirectoryID,
		Title:           item.Title,
		Status:          item.Status,
		StatusName:      enums.GetStatusLabel(item.Status),
		IndexStatus:     item.IndexStatus,
		IndexStatusName: enums.GetKnowledgeDocumentIndexStatusLabel(item.IndexStatus),
		IndexedAt:       item.IndexedAt,
		IndexError:      item.IndexError,
		ContentHash:     item.ContentHash,
		ContentType:     item.ContentType,
		CreatedAt:       item.CreatedAt,
		UpdatedAt:       item.UpdatedAt,
		CreateUserName:  item.CreateUserName,
		UpdateUserName:  item.UpdateUserName,
	}
}

func BuildKnowledgeFAQ(item *models.KnowledgeFAQ) response.KnowledgeFAQResponse {
	return response.KnowledgeFAQResponse{
		ID:               item.ID,
		KnowledgeBaseID:  item.KnowledgeBaseID,
		DirectoryID:      item.DirectoryID,
		Question:         item.Question,
		Answer:           item.Answer,
		SimilarQuestions: parseSimilarQuestions(item.SimilarQuestions),
		Status:           item.Status,
		StatusName:       enums.GetStatusLabel(item.Status),
		IndexStatus:      item.IndexStatus,
		IndexStatusName:  enums.GetKnowledgeDocumentIndexStatusLabel(item.IndexStatus),
		IndexedAt:        item.IndexedAt,
		IndexError:       item.IndexError,
		Remark:           item.Remark,
		CreatedAt:        item.CreatedAt,
		UpdatedAt:        item.UpdatedAt,
		CreateUserName:   item.CreateUserName,
		UpdateUserName:   item.UpdateUserName,
	}
}

func BuildKnowledgeDirectory(item *models.KnowledgeDirectory) response.KnowledgeDirectoryResponse {
	return response.KnowledgeDirectoryResponse{
		ID:              item.ID,
		KnowledgeBaseID: item.KnowledgeBaseID,
		ParentID:        item.ParentID,
		Name:            item.Name,
		SortNo:          item.SortNo,
		Status:          item.Status,
		StatusName:      enums.GetStatusLabel(item.Status),
		Remark:          item.Remark,
		CreatedAt:       item.CreatedAt,
		UpdatedAt:       item.UpdatedAt,
		CreateUserName:  item.CreateUserName,
		UpdateUserName:  item.UpdateUserName,
		Children:        []response.KnowledgeDirectoryResponse{},
	}
}

func BuildKnowledgeRetrieveLog(item *models.KnowledgeRetrieveLog) response.KnowledgeRetrieveLogResponse {
	return response.KnowledgeRetrieveLogResponse{
		ID:                 item.ID,
		KnowledgeBaseID:    item.KnowledgeBaseID,
		Channel:            item.Channel,
		ChannelName:        enums.GetKnowledgeRetrieveChannelLabel(enums.KnowledgeRetrieveChannel(item.Channel)),
		Scene:              item.Scene,
		SceneName:          enums.GetKnowledgeRetrieveSceneLabel(enums.KnowledgeRetrieveScene(item.Scene)),
		SessionID:          item.SessionID,
		ConversationID:     item.ConversationID,
		RequestID:          item.RequestID,
		Question:           item.Question,
		RewriteQuestion:    item.RewriteQuestion,
		Answer:             item.Answer,
		AnswerStatus:       item.AnswerStatus,
		AnswerStatusName:   enums.GetKnowledgeAnswerStatusLabel(enums.KnowledgeAnswerStatus(item.AnswerStatus)),
		HitCount:           item.HitCount,
		TopScore:           item.TopScore,
		ChunkProvider:      item.ChunkProvider,
		ChunkTargetTokens:  item.ChunkTargetTokens,
		ChunkMaxTokens:     item.ChunkMaxTokens,
		ChunkOverlapTokens: item.ChunkOverlapTokens,
		RerankEnabled:      item.RerankEnabled,
		RerankLimit:        item.RerankLimit,
		CitationCount:      item.CitationCount,
		UsedChunkCount:     item.UsedChunkCount,
		LatencyMs:          item.LatencyMs,
		RetrieveMs:         item.RetrieveMs,
		GenerateMs:         item.GenerateMs,
		PromptTokens:       item.PromptTokens,
		CompletionTokens:   item.CompletionTokens,
		ModelName:          item.ModelName,
		TraceData:          item.TraceData,
		CreatedAt:          item.CreatedAt,
	}
}

func BuildKnowledgeRetrieveHitResponse(item *models.KnowledgeRetrieveHit) response.KnowledgeRetrieveHitResponse {
	return response.KnowledgeRetrieveHitResponse{
		ID:              item.ID,
		RetrieveLogID:   item.RetrieveLogID,
		KnowledgeBaseID: item.KnowledgeBaseID,
		ChunkID:         item.ChunkID,
		DocumentID:      item.DocumentID,
		DocumentTitle:   item.DocumentTitle,
		FaqID:           item.FaqID,
		FaqQuestion:     item.FaqQuestion,
		ChunkNo:         item.ChunkNo,
		Title:           item.Title,
		SectionPath:     item.SectionPath,
		ChunkType:       item.ChunkType,
		ChunkTypeName:   enums.GetKnowledgeChunkTypeLabel(enums.KnowledgeChunkType(item.ChunkType)),
		Provider:        item.Provider,
		RankNo:          item.RankNo,
		Score:           item.Score,
		RerankScore:     item.RerankScore,
		UsedInAnswer:    item.UsedInAnswer,
		IsCitation:      item.IsCitation,
		Snippet:         item.Snippet,
		CreatedAt:       item.CreatedAt,
	}
}

func BuildKnowledgeFeedback(item *models.KnowledgeFeedback) response.KnowledgeFeedbackResponse {
	return response.KnowledgeFeedbackResponse{
		ID:               item.ID,
		RetrieveLogID:    item.RetrieveLogID,
		FeedbackType:     item.FeedbackType,
		FeedbackTypeName: enums.GetKnowledgeFeedbackTypeLabel(enums.KnowledgeFeedbackType(item.FeedbackType)),
		FeedbackReason:   item.FeedbackReason,
		UserID:           item.UserID,
		AgentID:          item.AgentID,
		Remark:           item.Remark,
		CreatedAt:        item.CreatedAt,
	}
}

func parseSimilarQuestions(raw string) []string {
	if raw == "" {
		return []string{}
	}
	var items []string
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return []string{}
	}
	return items
}
