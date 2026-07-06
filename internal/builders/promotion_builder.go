package builders

import (
	"agent-desk/internal/models"
	"agent-desk/internal/pkg/dto/response"
	"agent-desk/internal/pkg/utils"
)

func BuildPromotion(item *models.Promotion) *response.PromotionResponse {
	if item == nil {
		return nil
	}
	return &response.PromotionResponse{
		ID:                 item.ID,
		Name:               item.Name,
		PromotionType:      item.PromotionType,
		Description:        item.Description,
		ApplicableProducts: item.ApplicableProducts,
		StartAt:            utils.FormatTimePtr(item.StartAt),
		EndAt:              utils.FormatTimePtr(item.EndAt),
		DiscountRule:       item.DiscountRule,
		StoreBenefit:       item.StoreBenefit,
		AppointmentBenefit: item.AppointmentBenefit,
		ScriptSuggestion:   item.ScriptSuggestion,
		Priority:           item.Priority,
		KnowledgeBaseID:    item.KnowledgeBaseID,
		KnowledgeFAQID:     item.KnowledgeFAQID,
		Status:             item.Status,
		Remark:             item.Remark,
		CreatedAt:          utils.FormatTime(item.CreatedAt),
		UpdatedAt:          utils.FormatTime(item.UpdatedAt),
	}
}

func BuildPromotionList(list []models.Promotion) []response.PromotionResponse {
	results := make([]response.PromotionResponse, 0, len(list))
	for i := range list {
		if item := BuildPromotion(&list[i]); item != nil {
			results = append(results, *item)
		}
	}
	return results
}
