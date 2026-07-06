package builders

import (
	"agent-desk/internal/models"
	"agent-desk/internal/pkg/dto/response"
	"agent-desk/internal/pkg/utils"
)

func BuildProduct(item *models.Product) *response.ProductResponse {
	if item == nil {
		return nil
	}
	return &response.ProductResponse{
		ID:                 item.ID,
		Name:               item.Name,
		Category:           item.Category,
		PriceMin:           item.PriceMin,
		PriceMax:           item.PriceMax,
		SellingPoints:      item.SellingPoints,
		SuitablePeople:     item.SuitablePeople,
		UnsuitablePeople:   item.UnsuitablePeople,
		Scenarios:          item.Scenarios,
		Specs:              item.Specs,
		IndustryAttributes: item.IndustryAttributes,
		ImageURL:           item.ImageURL,
		Priority:           item.Priority,
		KnowledgeBaseID:    item.KnowledgeBaseID,
		KnowledgeFAQID:     item.KnowledgeFAQID,
		Status:             item.Status,
		Remark:             item.Remark,
		CreatedAt:          utils.FormatTime(item.CreatedAt),
		UpdatedAt:          utils.FormatTime(item.UpdatedAt),
	}
}

func BuildProductList(list []models.Product) []response.ProductResponse {
	results := make([]response.ProductResponse, 0, len(list))
	for i := range list {
		if item := BuildProduct(&list[i]); item != nil {
			results = append(results, *item)
		}
	}
	return results
}
