package services

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"agent-desk/internal/ai/rag"
	"agent-desk/internal/models"
	"agent-desk/internal/pkg/dto"
	"agent-desk/internal/pkg/dto/request"
	"agent-desk/internal/pkg/dto/response"
	"agent-desk/internal/pkg/enums"
	"agent-desk/internal/pkg/errorsx"
	"agent-desk/internal/pkg/utils"
	"agent-desk/internal/repositories"

	"github.com/mlogclub/simple/sqls"
)

var PromotionService = newPromotionService()

func newPromotionService() *promotionService {
	return &promotionService{}
}

type promotionService struct {
}

func (s *promotionService) Get(id int64) *models.Promotion {
	return repositories.PromotionRepository.Get(sqls.DB(), id)
}

func (s *promotionService) List(req request.PromotionListRequest) (list []models.Promotion, paging *sqls.Paging) {
	tx := sqls.DB().Model(&models.Promotion{}).Where("status <> ?", enums.StatusDeleted)
	if kw := strings.TrimSpace(req.Keyword); kw != "" {
		pat := "%" + kw + "%"
		tx = tx.Where("name LIKE ? OR promotion_type LIKE ? OR description LIKE ? OR applicable_products LIKE ? OR discount_rule LIKE ? OR store_benefit LIKE ? OR appointment_benefit LIKE ?", pat, pat, pat, pat, pat, pat, pat)
	}
	if promotionType := strings.TrimSpace(req.PromotionType); promotionType != "" {
		tx = tx.Where("promotion_type = ?", promotionType)
	}
	if req.Status != nil {
		tx = tx.Where("status = ?", *req.Status)
	}
	if req.ActiveOnly {
		now := time.Now()
		tx = tx.Where("status = ?", enums.StatusOk).
			Where("(start_at IS NULL OR start_at <= ?)", now).
			Where("(end_at IS NULL OR end_at >= ?)", now)
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, &sqls.Paging{Page: req.GetPage(), Limit: req.GetLimit(), Total: 0}
	}
	if err := tx.Order("priority DESC, id DESC").Offset(req.Offset()).Limit(req.GetLimit()).Find(&list).Error; err != nil {
		return nil, &sqls.Paging{Page: req.GetPage(), Limit: req.GetLimit(), Total: total}
	}
	return list, &sqls.Paging{Page: req.GetPage(), Limit: req.GetLimit(), Total: total}
}

func (s *promotionService) Create(req request.SavePromotionRequest, operator *dto.AuthPrincipal) (*models.Promotion, error) {
	if operator == nil {
		return nil, errorsx.UnauthorizedI18n("error.auth.expired")
	}
	item, err := s.buildPromotionModel(req)
	if err != nil {
		return nil, err
	}
	item.AuditFields = utils.BuildAuditFields(operator)
	if err := repositories.PromotionRepository.Create(sqls.DB(), item); err != nil {
		return nil, err
	}
	if err := s.SyncKnowledgeFAQ(item.ID); err != nil {
		return item, err
	}
	return s.Get(item.ID), nil
}

func (s *promotionService) Update(req request.SavePromotionRequest, operator *dto.AuthPrincipal) error {
	if operator == nil {
		return errorsx.UnauthorizedI18n("error.auth.expired")
	}
	current := s.Get(req.ID)
	if current == nil {
		return errorsx.InvalidParam("promotion not found")
	}
	item, err := s.buildPromotionModel(req)
	if err != nil {
		return err
	}
	if err := repositories.PromotionRepository.Updates(sqls.DB(), current.ID, map[string]any{
		"name":                item.Name,
		"promotion_type":      item.PromotionType,
		"description":         item.Description,
		"applicable_products": item.ApplicableProducts,
		"start_at":            item.StartAt,
		"end_at":              item.EndAt,
		"discount_rule":       item.DiscountRule,
		"store_benefit":       item.StoreBenefit,
		"appointment_benefit": item.AppointmentBenefit,
		"script_suggestion":   item.ScriptSuggestion,
		"priority":            item.Priority,
		"knowledge_base_id":   item.KnowledgeBaseID,
		"status":              item.Status,
		"remark":              item.Remark,
		"updated_at":          time.Now(),
		"update_user_id":      operator.UserID,
		"update_user_name":    operator.Username,
	}); err != nil {
		return err
	}
	return s.SyncKnowledgeFAQ(current.ID)
}

func (s *promotionService) UpdateStatus(req request.UpdatePromotionStatusRequest, operator *dto.AuthPrincipal) error {
	if operator == nil {
		return errorsx.UnauthorizedI18n("error.auth.expired")
	}
	if !enums.IsValidStatus(req.Status) || enums.Status(req.Status) == enums.StatusDeleted {
		return errorsx.InvalidParam("invalid promotion status")
	}
	if s.Get(req.ID) == nil {
		return errorsx.InvalidParam("promotion not found")
	}
	if err := repositories.PromotionRepository.Updates(sqls.DB(), req.ID, map[string]any{
		"status":           enums.Status(req.Status),
		"updated_at":       time.Now(),
		"update_user_id":   operator.UserID,
		"update_user_name": operator.Username,
	}); err != nil {
		return err
	}
	return s.SyncKnowledgeFAQ(req.ID)
}

func (s *promotionService) Delete(id int64, operator *dto.AuthPrincipal) error {
	if operator == nil {
		return errorsx.UnauthorizedI18n("error.auth.expired")
	}
	item := s.Get(id)
	if item == nil {
		return errorsx.InvalidParam("promotion not found")
	}
	if err := repositories.PromotionRepository.Updates(sqls.DB(), id, map[string]any{
		"status":           enums.StatusDeleted,
		"updated_at":       time.Now(),
		"update_user_id":   operator.UserID,
		"update_user_name": operator.Username,
	}); err != nil {
		return err
	}
	if item.KnowledgeFAQID > 0 {
		if err := KnowledgeFAQService.DeleteKnowledgeFAQ(item.KnowledgeFAQID); err != nil {
			return err
		}
	}
	return nil
}

func (s *promotionService) Reindex(id int64) error {
	return s.SyncKnowledgeFAQ(id)
}

func (s *promotionService) SeedMusePromotions(operator *dto.AuthPrincipal) error {
	return s.SeedTemplatePromotions("muse_bedding", operator)
}

func (s *promotionService) SeedTemplatePromotions(templateCode string, operator *dto.AuthPrincipal) error {
	if operator == nil {
		return errorsx.UnauthorizedI18n("error.auth.expired")
	}
	seeds, err := promotionTemplateSeeds(templateCode, time.Now())
	if err != nil {
		return err
	}
	if err := s.UpsertTemplatePromotions(seeds, operator); err != nil {
		return err
	}
	return s.syncCurrentPromotionGuideFAQ(operator)
}

func (s *promotionService) UpsertTemplatePromotions(seeds []request.SavePromotionRequest, operator *dto.AuthPrincipal) error {
	if operator == nil {
		return errorsx.UnauthorizedI18n("error.auth.expired")
	}
	for _, req := range seeds {
		existing := repositories.PromotionRepository.FindOne(sqls.DB(), sqls.NewCnd().Eq("name", req.Name).Where("status <> ?", enums.StatusDeleted))
		if existing == nil {
			if _, err := s.Create(req, operator); err != nil {
				return err
			}
			continue
		}
		req.ID = existing.ID
		if req.KnowledgeBaseID == 0 {
			req.KnowledgeBaseID = existing.KnowledgeBaseID
		}
		if err := s.Update(req, operator); err != nil {
			return err
		}
	}
	return nil
}

func (s *promotionService) ImportCSV(reader io.Reader, operator *dto.AuthPrincipal) (response.PromotionImportResultResponse, error) {
	ret := response.PromotionImportResultResponse{Errors: make([]response.PromotionImportRowResponse, 0)}
	if operator == nil {
		return ret, errorsx.UnauthorizedI18n("error.auth.expired")
	}
	rows, err := parsePromotionCSV(reader)
	if err != nil {
		return ret, err
	}
	for _, row := range rows {
		ret.Total++
		req, err := buildPromotionImportRequest(row.Values)
		if err != nil {
			ret.Failed++
			ret.Errors = append(ret.Errors, response.PromotionImportRowResponse{Row: row.Row, Message: err.Error()})
			continue
		}
		existing := repositories.PromotionRepository.FindOne(sqls.DB(), sqls.NewCnd().Eq("name", req.Name).Where("status <> ?", enums.StatusDeleted))
		if existing == nil {
			if _, err := s.Create(req, operator); err != nil {
				ret.Failed++
				ret.Errors = append(ret.Errors, response.PromotionImportRowResponse{Row: row.Row, Message: err.Error()})
				continue
			}
			ret.Created++
			continue
		}
		req.ID = existing.ID
		if req.KnowledgeBaseID == 0 {
			req.KnowledgeBaseID = existing.KnowledgeBaseID
		}
		if err := s.Update(req, operator); err != nil {
			ret.Failed++
			ret.Errors = append(ret.Errors, response.PromotionImportRowResponse{Row: row.Row, Message: err.Error()})
			continue
		}
		ret.Updated++
	}
	return ret, nil
}

func (s *promotionService) SyncKnowledgeFAQ(promotionID int64) error {
	item := s.Get(promotionID)
	if item == nil {
		return errorsx.InvalidParam("promotion not found")
	}
	kbID, err := resolveDigitalStoreKnowledgeBaseID(item.KnowledgeBaseID)
	if err != nil {
		return err
	}
	question, answer, similarQuestions, remark := BuildPromotionKnowledgeFAQContent(item)
	similarJSON, err := json.Marshal(similarQuestions)
	if err != nil {
		return err
	}
	now := time.Now()
	faq := repositories.KnowledgeFAQRepository.Get(sqls.DB(), item.KnowledgeFAQID)
	if faq == nil && item.KnowledgeFAQID > 0 {
		item.KnowledgeFAQID = 0
	}
	if faq == nil {
		faq = &models.KnowledgeFAQ{
			KnowledgeBaseID:  kbID,
			Question:         question,
			Answer:           answer,
			SimilarQuestions: string(similarJSON),
			IndexStatus:      enums.KnowledgeDocumentIndexStatusPending,
			Status:           item.Status,
			Remark:           remark,
			AuditFields: models.AuditFields{
				CreatedAt:      now,
				CreateUserID:   item.UpdateUserID,
				CreateUserName: item.UpdateUserName,
				UpdatedAt:      now,
				UpdateUserID:   item.UpdateUserID,
				UpdateUserName: item.UpdateUserName,
			},
		}
		if err := repositories.KnowledgeFAQRepository.Create(sqls.DB(), faq); err != nil {
			return err
		}
		if err := repositories.PromotionRepository.Updates(sqls.DB(), item.ID, map[string]any{
			"knowledge_base_id": kbID,
			"knowledge_faq_id":  faq.ID,
			"updated_at":        now,
		}); err != nil {
			return err
		}
	} else {
		if err := repositories.KnowledgeFAQRepository.Updates(sqls.DB(), faq.ID, map[string]any{
			"knowledge_base_id": kbID,
			"question":          question,
			"answer":            answer,
			"similar_questions": string(similarJSON),
			"index_status":      enums.KnowledgeDocumentIndexStatusPending,
			"indexed_at":        nil,
			"index_error":       "",
			"status":            item.Status,
			"remark":            remark,
			"updated_at":        now,
			"update_user_id":    item.UpdateUserID,
			"update_user_name":  item.UpdateUserName,
		}); err != nil {
			return err
		}
		if item.KnowledgeBaseID != kbID {
			if err := repositories.PromotionRepository.Updates(sqls.DB(), item.ID, map[string]any{
				"knowledge_base_id": kbID,
				"updated_at":        now,
			}); err != nil {
				return err
			}
		}
	}
	return rag.Index.IndexFAQByID(context.Background(), faq.ID)
}

func (s *promotionService) buildPromotionModel(req request.SavePromotionRequest) (*models.Promotion, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, errorsx.InvalidParam("promotion name is required")
	}
	status := enums.Status(req.Status)
	if req.Status == 0 {
		status = enums.StatusOk
	}
	if !enums.IsValidStatus(int(status)) || status == enums.StatusDeleted {
		return nil, errorsx.InvalidParam("invalid promotion status")
	}
	startAt, err := parsePromotionTime(req.StartAt)
	if err != nil {
		return nil, err
	}
	endAt, err := parsePromotionTime(req.EndAt)
	if err != nil {
		return nil, err
	}
	if startAt != nil && endAt != nil && startAt.After(*endAt) {
		return nil, errorsx.InvalidParam("promotion start time cannot be after end time")
	}
	if req.KnowledgeBaseID > 0 {
		if _, err := resolveDigitalStoreKnowledgeBaseID(req.KnowledgeBaseID); err != nil {
			return nil, err
		}
	}
	return &models.Promotion{
		Name:               name,
		PromotionType:      strings.TrimSpace(req.PromotionType),
		Description:        strings.TrimSpace(req.Description),
		ApplicableProducts: strings.TrimSpace(req.ApplicableProducts),
		StartAt:            startAt,
		EndAt:              endAt,
		DiscountRule:       strings.TrimSpace(req.DiscountRule),
		StoreBenefit:       strings.TrimSpace(req.StoreBenefit),
		AppointmentBenefit: strings.TrimSpace(req.AppointmentBenefit),
		ScriptSuggestion:   strings.TrimSpace(req.ScriptSuggestion),
		Priority:           req.Priority,
		KnowledgeBaseID:    req.KnowledgeBaseID,
		Status:             status,
		Remark:             strings.TrimSpace(req.Remark),
	}, nil
}

func parsePromotionTime(raw string) (*time.Time, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return nil, nil
	}
	for _, layout := range []string{time.DateTime, time.DateOnly} {
		parsed, err := time.ParseInLocation(layout, value, time.Local)
		if err == nil {
			return &parsed, nil
		}
	}
	return nil, errorsx.InvalidParam("invalid promotion time")
}

type promotionCSVRow struct {
	Row    int
	Values map[string]string
}

func parsePromotionCSV(reader io.Reader) ([]promotionCSVRow, error) {
	csvReader := csv.NewReader(reader)
	csvReader.TrimLeadingSpace = true
	csvReader.FieldsPerRecord = -1
	records, err := csvReader.ReadAll()
	if err != nil {
		return nil, errorsx.InvalidParam("invalid promotion csv file")
	}
	if len(records) == 0 {
		return nil, errorsx.InvalidParam("promotion csv is empty")
	}
	headers := normalizePromotionCSVHeaders(records[0])
	if len(headers) == 0 {
		return nil, errorsx.InvalidParam("promotion csv header is empty")
	}
	rows := make([]promotionCSVRow, 0, len(records)-1)
	for index, record := range records[1:] {
		values := make(map[string]string, len(headers))
		hasValue := false
		for i, header := range headers {
			if header == "" || i >= len(record) {
				continue
			}
			value := strings.TrimSpace(record[i])
			if value != "" {
				hasValue = true
			}
			values[header] = value
		}
		if !hasValue {
			continue
		}
		rows = append(rows, promotionCSVRow{Row: index + 2, Values: values})
	}
	return rows, nil
}

func normalizePromotionCSVHeaders(raw []string) []string {
	headers := make([]string, 0, len(raw))
	for _, item := range raw {
		key := strings.TrimSpace(strings.TrimPrefix(item, "\ufeff"))
		key = strings.ToLower(strings.ReplaceAll(key, " ", ""))
		switch key {
		case "活动名称", "名称", "name", "promotionname":
			headers = append(headers, "name")
		case "活动类型", "类型", "promotiontype", "type":
			headers = append(headers, "promotionType")
		case "活动描述", "活动说明", "描述", "description":
			headers = append(headers, "description")
		case "适用产品", "适用商品", "products", "applicableproducts":
			headers = append(headers, "applicableProducts")
		case "开始时间", "开始日期", "startat", "start":
			headers = append(headers, "startAt")
		case "结束时间", "结束日期", "endat", "end":
			headers = append(headers, "endAt")
		case "优惠规则", "discount", "discountrule":
			headers = append(headers, "discountRule")
		case "到店权益", "到店礼", "storebenefit":
			headers = append(headers, "storeBenefit")
		case "预约权益", "预约礼", "appointmentbenefit":
			headers = append(headers, "appointmentBenefit")
		case "话术建议", "推荐话术", "scriptsuggestion":
			headers = append(headers, "scriptSuggestion")
		case "推荐优先级", "优先级", "priority":
			headers = append(headers, "priority")
		case "知识库id", "知识库ID", "knowledgebaseid", "knowledgebase":
			headers = append(headers, "knowledgeBaseId")
		case "状态", "status":
			headers = append(headers, "status")
		case "备注", "remark":
			headers = append(headers, "remark")
		default:
			headers = append(headers, "")
		}
	}
	return headers
}

func buildPromotionImportRequest(values map[string]string) (request.SavePromotionRequest, error) {
	req := request.SavePromotionRequest{
		Name:               strings.TrimSpace(values["name"]),
		PromotionType:      strings.TrimSpace(values["promotionType"]),
		Description:        strings.TrimSpace(values["description"]),
		ApplicableProducts: strings.TrimSpace(values["applicableProducts"]),
		DiscountRule:       strings.TrimSpace(values["discountRule"]),
		StoreBenefit:       strings.TrimSpace(values["storeBenefit"]),
		AppointmentBenefit: strings.TrimSpace(values["appointmentBenefit"]),
		ScriptSuggestion:   strings.TrimSpace(values["scriptSuggestion"]),
		Remark:             strings.TrimSpace(values["remark"]),
		Status:             int(enums.StatusOk),
	}
	startAt, err := parsePromotionImportDate(values["startAt"], "开始时间", false)
	if err != nil {
		return req, err
	}
	req.StartAt = startAt
	endAt, err := parsePromotionImportDate(values["endAt"], "结束时间", true)
	if err != nil {
		return req, err
	}
	req.EndAt = endAt
	if req.Priority, err = parsePromotionImportInt(values["priority"], "推荐优先级"); err != nil {
		return req, err
	}
	if req.KnowledgeBaseID, err = parsePromotionImportInt64(values["knowledgeBaseId"], "知识库ID"); err != nil {
		return req, err
	}
	if status, ok, err := parsePromotionImportStatus(values["status"]); err != nil {
		return req, err
	} else if ok {
		req.Status = int(status)
	}
	if req.Name == "" {
		return req, fmt.Errorf("活动名称不能为空")
	}
	return req, nil
}

func parsePromotionImportDate(value string, field string, endOfDay bool) (string, error) {
	value = strings.TrimSpace(strings.ReplaceAll(value, "/", "-"))
	if value == "" {
		return "", nil
	}
	for _, layout := range []string{time.DateTime, "2006-01-02 15:04", time.DateOnly} {
		parsed, err := time.ParseInLocation(layout, value, time.Local)
		if err != nil {
			continue
		}
		if layout == time.DateOnly && endOfDay {
			parsed = parsed.Add(24*time.Hour - time.Second)
		}
		return parsed.Format(time.DateTime), nil
	}
	return "", fmt.Errorf("%s格式需为 YYYY-MM-DD 或 YYYY-MM-DD HH:mm:ss", field)
}

func parsePromotionImportInt(value string, field string) (int, error) {
	if strings.TrimSpace(value) == "" {
		return 0, nil
	}
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return 0, fmt.Errorf("%s必须是整数", field)
	}
	return parsed, nil
}

func parsePromotionImportInt64(value string, field string) (int64, error) {
	if strings.TrimSpace(value) == "" {
		return 0, nil
	}
	parsed, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%s必须是整数", field)
	}
	return parsed, nil
}

func parsePromotionImportStatus(value string) (enums.Status, bool, error) {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return enums.StatusOk, false, nil
	}
	switch value {
	case "0", "启用", "上架", "ok", "enabled", "enable":
		return enums.StatusOk, true, nil
	case "1", "禁用", "下架", "disabled", "disable":
		return enums.StatusDisabled, true, nil
	default:
		return enums.StatusOk, false, fmt.Errorf("状态只支持启用或禁用")
	}
}

func BuildPromotionKnowledgeFAQContent(item *models.Promotion) (question string, answer string, similarQuestions []string, remark string) {
	timeRange := "长期有效"
	if item.StartAt != nil && item.EndAt != nil {
		timeRange = fmt.Sprintf("%s 至 %s", item.StartAt.Format(time.DateOnly), item.EndAt.Format(time.DateOnly))
	} else if item.StartAt != nil {
		timeRange = item.StartAt.Format(time.DateOnly) + " 起"
	} else if item.EndAt != nil {
		timeRange = item.EndAt.Format(time.DateOnly) + " 前有效"
	}
	promotionType := strings.TrimSpace(item.PromotionType)
	if promotionType == "" {
		promotionType = "促销活动"
	}
	lines := []string{
		"活动名称：" + item.Name,
		"活动类型：" + promotionType,
		"有效期：" + timeRange,
	}
	appendField := func(label string, value string) {
		if v := strings.TrimSpace(value); v != "" {
			lines = append(lines, label+"："+v)
		}
	}
	appendField("活动说明", item.Description)
	appendField("适用产品", item.ApplicableProducts)
	appendField("优惠规则", item.DiscountRule)
	appendField("到店权益", item.StoreBenefit)
	appendField("预约权益", item.AppointmentBenefit)
	appendField("推荐话术", item.ScriptSuggestion)
	lines = append(lines, "导购要求：仅在活动启用且处于有效期时主动推荐；如果客户询问最终成交价、库存或叠加优惠，应引导留资或转人工确认。")
	return "活动优惠：" + item.Name,
		strings.Join(lines, "\n"),
		[]string{
			item.Name,
			"现在有什么优惠",
			"到店有什么权益",
			"预约试躺有什么礼品",
			"活动怎么参加",
			promotionType + "活动",
		},
		"promotion:" + fmt.Sprint(item.ID)
}

func (s *promotionService) syncCurrentPromotionGuideFAQ(operator *dto.AuthPrincipal) error {
	kbID, err := resolveDigitalStoreKnowledgeBaseID(0)
	if err != nil {
		return err
	}
	now := time.Now()
	list, _ := s.List(request.PromotionListRequest{Page: 1, Limit: 20, ActiveOnly: true})
	lines := []string{"当前可主动推荐的活动优惠："}
	if len(list) == 0 {
		lines = append(lines, "暂无启用且在有效期内的活动。")
	}
	for i, item := range list {
		_, answer, _, _ := BuildPromotionKnowledgeFAQContent(&item)
		lines = append(lines, fmt.Sprintf("%d. %s\n%s", i+1, item.Name, answer))
	}
	similarJSON, err := json.Marshal([]string{"当前活动", "现在有什么优惠", "预约权益", "到店权益", "近期促销"})
	if err != nil {
		return err
	}
	const question = "当前活动优惠总览"
	existing := repositories.KnowledgeFAQRepository.FindByKnowledgeBaseIDAndQuestions(sqls.DB(), kbID, []string{question})
	if len(existing) == 0 {
		item := &models.KnowledgeFAQ{
			KnowledgeBaseID:  kbID,
			Question:         question,
			Answer:           strings.Join(lines, "\n\n"),
			SimilarQuestions: string(similarJSON),
			IndexStatus:      enums.KnowledgeDocumentIndexStatusPending,
			Status:           enums.StatusOk,
			Remark:           "promotion-guide-seed",
			AuditFields:      utils.BuildAuditFields(operator),
		}
		if err := repositories.KnowledgeFAQRepository.Create(sqls.DB(), item); err != nil {
			return err
		}
		return rag.Index.IndexFAQByID(context.Background(), item.ID)
	}
	item := existing[0]
	if err := repositories.KnowledgeFAQRepository.Updates(sqls.DB(), item.ID, map[string]any{
		"answer":            strings.Join(lines, "\n\n"),
		"similar_questions": string(similarJSON),
		"index_status":      enums.KnowledgeDocumentIndexStatusPending,
		"indexed_at":        nil,
		"index_error":       "",
		"status":            enums.StatusOk,
		"remark":            "promotion-guide-seed",
		"updated_at":        now,
		"update_user_id":    operator.UserID,
		"update_user_name":  operator.Username,
	}); err != nil {
		return err
	}
	return rag.Index.IndexFAQByID(context.Background(), item.ID)
}

func musePromotionSeeds(now time.Time) []request.SavePromotionRequest {
	start := now.AddDate(0, 0, -7).Format(time.DateOnly)
	end := now.AddDate(0, 1, 0).Format(time.DateOnly)
	return []request.SavePromotionRequest{
		{
			Name:               "周末预约试躺礼",
			PromotionType:      "预约权益",
			Description:        "客户提前预约周末到店试躺，可安排睡眠顾问预留体验时段。",
			ApplicableProducts: "慕斯脊护支撑款、慕斯云感舒睡款、慕斯智能电动床",
			StartAt:            start,
			EndAt:              end,
			DiscountRule:       "具体成交价和叠加优惠以门店顾问确认为准。",
			StoreBenefit:       "到店可享免费睡眠咨询和床垫软硬度试躺对比。",
			AppointmentBenefit: "提前预约并留下手机号，可预留周末体验时段；到店可领取护睡礼包一份，数量以门店为准。",
			ScriptSuggestion:   "如果客户提到周末、到店、试躺或老人选床垫，优先邀请预约试躺并留下姓名、手机号、到店时间和预算。",
			Priority:           90,
			Status:             int(enums.StatusOk),
			Remark:             "慕斯寝具模拟活动",
		},
		{
			Name:               "智能电动床组合体验季",
			PromotionType:      "组合权益",
			Description:        "智能电动床搭配适配床垫的体验活动，适合老人房、孕妇、阅读观影和康养场景。",
			ApplicableProducts: "慕斯智能电动床、慕斯脊护支撑款",
			StartAt:            start,
			EndAt:              end,
			DiscountRule:       "组合方案可到店咨询顾问确认预算和适配规格。",
			StoreBenefit:       "到店可体验头脚升降、阅读观影模式和床垫适配方案。",
			AppointmentBenefit: "预约体验可优先安排电动床演示和顾问一对一讲解。",
			ScriptSuggestion:   "客户提到老人起身、孕妇、床上阅读观影或康养需求时，可补充介绍智能电动床组合体验。",
			Priority:           80,
			Status:             int(enums.StatusOk),
			Remark:             "慕斯寝具模拟活动",
		},
	}
}

func promotionTemplateSeeds(templateCode string, now time.Time) ([]request.SavePromotionRequest, error) {
	switch strings.TrimSpace(templateCode) {
	case "", "muse_bedding", "muse":
		return musePromotionSeeds(now), nil
	case "oral_clinic":
		return oralClinicPromotionSeeds(now), nil
	case "kids_english":
		return kidsEnglishPromotionSeeds(now), nil
	case "finance_advisor":
		return financeAdvisorPromotionSeeds(now), nil
	case "home_decoration":
		return homeDecorationPromotionSeeds(now), nil
	default:
		return nil, errorsx.InvalidParam("unsupported digital store template")
	}
}

func oralClinicPromotionSeeds(now time.Time) []request.SavePromotionRequest {
	start := now.AddDate(0, 0, -7).Format(time.DateOnly)
	end := now.AddDate(0, 1, 0).Format(time.DateOnly)
	return []request.SavePromotionRequest{
		{
			Name:               "正畸初诊评估预约礼",
			PromotionType:      "预约权益",
			Description:        "客户预约隐形矫正初诊评估，可优先安排正畸医生评估时段。",
			ApplicableProducts: "隐形矫正初诊评估",
			StartAt:            start,
			EndAt:              end,
			DiscountRule:       "初诊检查项目、影像检查和最终费用以门诊确认及医生评估为准。",
			StoreBenefit:       "到店可由顾问协助完成基础资料登记，并安排医生评估牙列情况。",
			AppointmentBenefit: "提前预约并留下手机号、期望时间和主要诉求，可优先匹配正畸咨询时段。",
			ScriptSuggestion:   "客户提到牙齿不齐、牙缝、龅牙、地包天或想了解矫正预算时，先声明需医生面诊确认，再引导预约初诊并留联系方式。",
			Priority:           90,
			Status:             int(enums.StatusOk),
			Remark:             "口腔门诊模拟活动",
		},
		{
			Name:               "儿童口腔检查关爱周",
			PromotionType:      "儿童齿科",
			Description:        "面向关注儿童龋齿预防的家长，提供儿童口腔检查和预防项目咨询。",
			ApplicableProducts: "儿童涂氟与窝沟封闭",
			StartAt:            start,
			EndAt:              end,
			DiscountRule:       "儿童涂氟、窝沟封闭是否适合及具体费用需儿童牙医检查后确认。",
			StoreBenefit:       "到店可了解儿童龋齿预防建议和定期检查安排。",
			AppointmentBenefit: "预约可优先安排儿童齿科时段，建议留下儿童年龄、是否首次就诊和期望日期。",
			ScriptSuggestion:   "客户询问孩子蛀牙预防、涂氟、窝沟封闭或换牙期问题时，提醒不能在线诊断，并引导家长预约儿童牙医检查。",
			Priority:           80,
			Status:             int(enums.StatusOk),
			Remark:             "口腔门诊模拟活动",
		},
	}
}

func kidsEnglishPromotionSeeds(now time.Time) []request.SavePromotionRequest {
	start := now.AddDate(0, 0, -7).Format(time.DateOnly)
	end := now.AddDate(0, 1, 0).Format(time.DateOnly)
	return []request.SavePromotionRequest{
		{
			Name:               "少儿英语试听测评预约礼",
			PromotionType:      "试听权益",
			Description:        "客户预约试听或测评，可优先安排课程顾问做年级、基础和目标沟通。",
			ApplicableProducts: "自然拼读进阶班,剑桥少儿英语能力班,一对一学习规划咨询",
			StartAt:            start,
			EndAt:              end,
			DiscountRule:       "试听课、测评方式、班型名额和最终学费以校区课程顾问确认为准。",
			StoreBenefit:       "到校可了解课程体系、班型安排和阶段学习建议。",
			AppointmentBenefit: "提前预约并留下学生年级、学习目标、手机号和试听时间，可优先匹配顾问时段。",
			ScriptSuggestion:   "客户提到孩子年级、英语基础、试听或学费时，先追问目标与时间，再引导预约试听测评。",
			Priority:           90,
			Status:             int(enums.StatusOk),
			Remark:             "教育培训模拟活动",
		},
	}
}

func financeAdvisorPromotionSeeds(now time.Time) []request.SavePromotionRequest {
	start := now.AddDate(0, 0, -7).Format(time.DateOnly)
	end := now.AddDate(0, 1, 0).Format(time.DateOnly)
	return []request.SavePromotionRequest{
		{
			Name:               "金融顾问合规初评预约",
			PromotionType:      "咨询预约",
			Description:        "客户预约后由顾问进行基础需求沟通和资料清单说明。",
			ApplicableProducts: "经营贷资质初评,家庭保障方案咨询,资产配置风险测评预约",
			StartAt:            start,
			EndAt:              end,
			DiscountRule:       "咨询不代表审批、承保或收益承诺；具体方案、费率、合同和风险等级以持牌顾问确认为准。",
			StoreBenefit:       "顾问可协助梳理基础资料、风险提示和下一步沟通方式。",
			AppointmentBenefit: "提前预约并留下姓名、手机号、所在城市和咨询方向，可优先安排顾问回访。",
			ScriptSuggestion:   "客户询问利率、额度、收益或保险方案时，先做合规边界说明，再引导留下联系方式由持牌顾问确认。",
			Priority:           90,
			Status:             int(enums.StatusOk),
			Remark:             "金融服务模拟活动",
		},
	}
}

func homeDecorationPromotionSeeds(now time.Time) []request.SavePromotionRequest {
	start := now.AddDate(0, 0, -7).Format(time.DateOnly)
	end := now.AddDate(0, 1, 0).Format(time.DateOnly)
	return []request.SavePromotionRequest{
		{
			Name:               "免费量房与设计咨询季",
			PromotionType:      "量房预约",
			Description:        "预约量房后由设计师了解户型、面积、预算、风格和装修阶段。",
			ApplicableProducts: "全案设计咨询,整装施工套餐咨询,旧房翻新评估",
			StartAt:            start,
			EndAt:              end,
			DiscountRule:       "最终报价、材料、工期、增项和合同权益需量房与设计沟通后确认。",
			StoreBenefit:       "到店可了解设计案例、材料展厅和施工流程。",
			AppointmentBenefit: "提前预约并留下小区/面积/预算/风格/手机号，可优先安排设计师量房沟通。",
			ScriptSuggestion:   "客户提到面积、预算、风格、交房或旧房翻新时，先追问量房条件，再引导预约设计师。",
			Priority:           90,
			Status:             int(enums.StatusOk),
			Remark:             "家装装修模拟活动",
		},
	}
}
