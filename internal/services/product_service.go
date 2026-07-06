package services

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
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

var ProductService = newProductService()

func newProductService() *productService {
	return &productService{}
}

type productService struct {
}

func (s *productService) Get(id int64) *models.Product {
	return repositories.ProductRepository.Get(sqls.DB(), id)
}

func (s *productService) List(req request.ProductListRequest) (list []models.Product, paging *sqls.Paging) {
	tx := sqls.DB().Model(&models.Product{}).Where("status <> ?", enums.StatusDeleted)
	if kw := strings.TrimSpace(req.Keyword); kw != "" {
		pat := "%" + kw + "%"
		tx = tx.Where("name LIKE ? OR category LIKE ? OR selling_points LIKE ? OR suitable_people LIKE ? OR scenarios LIKE ? OR specs LIKE ?", pat, pat, pat, pat, pat, pat)
	}
	if category := strings.TrimSpace(req.Category); category != "" {
		tx = tx.Where("category = ?", category)
	}
	if req.Status != nil {
		tx = tx.Where("status = ?", *req.Status)
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		slog.Error("product list count failed", "error", err)
	}
	if err := tx.Order("priority DESC, id DESC").Offset(req.Offset()).Limit(req.GetLimit()).Find(&list).Error; err != nil {
		slog.Error("product list scan failed", "error", err)
	}
	return list, &sqls.Paging{Page: req.GetPage(), Limit: req.GetLimit(), Total: total}
}

func (s *productService) Create(req request.SaveProductRequest, operator *dto.AuthPrincipal) (*models.Product, error) {
	if operator == nil {
		return nil, errorsx.UnauthorizedI18n("error.auth.expired")
	}
	item, err := s.buildProductModel(req)
	if err != nil {
		return nil, err
	}
	item.AuditFields = utils.BuildAuditFields(operator)
	if err := repositories.ProductRepository.Create(sqls.DB(), item); err != nil {
		return nil, err
	}
	if err := s.SyncKnowledgeFAQ(item.ID); err != nil {
		return item, err
	}
	return s.Get(item.ID), nil
}

func (s *productService) Update(req request.SaveProductRequest, operator *dto.AuthPrincipal) error {
	if operator == nil {
		return errorsx.UnauthorizedI18n("error.auth.expired")
	}
	current := s.Get(req.ID)
	if current == nil {
		return errorsx.InvalidParam("product not found")
	}
	item, err := s.buildProductModel(req)
	if err != nil {
		return err
	}
	updates := map[string]any{
		"name":                item.Name,
		"category":            item.Category,
		"price_min":           item.PriceMin,
		"price_max":           item.PriceMax,
		"selling_points":      item.SellingPoints,
		"suitable_people":     item.SuitablePeople,
		"unsuitable_people":   item.UnsuitablePeople,
		"scenarios":           item.Scenarios,
		"specs":               item.Specs,
		"industry_attributes": item.IndustryAttributes,
		"image_url":           item.ImageURL,
		"priority":            item.Priority,
		"knowledge_base_id":   item.KnowledgeBaseID,
		"status":              item.Status,
		"remark":              item.Remark,
		"updated_at":          time.Now(),
		"update_user_id":      operator.UserID,
		"update_user_name":    operator.Username,
	}
	if err := repositories.ProductRepository.Updates(sqls.DB(), current.ID, updates); err != nil {
		return err
	}
	return s.SyncKnowledgeFAQ(current.ID)
}

func (s *productService) UpdateStatus(req request.UpdateProductStatusRequest, operator *dto.AuthPrincipal) error {
	if operator == nil {
		return errorsx.UnauthorizedI18n("error.auth.expired")
	}
	if !enums.IsValidStatus(req.Status) || enums.Status(req.Status) == enums.StatusDeleted {
		return errorsx.InvalidParam("invalid product status")
	}
	if s.Get(req.ID) == nil {
		return errorsx.InvalidParam("product not found")
	}
	if err := repositories.ProductRepository.Updates(sqls.DB(), req.ID, map[string]any{
		"status":           enums.Status(req.Status),
		"updated_at":       time.Now(),
		"update_user_id":   operator.UserID,
		"update_user_name": operator.Username,
	}); err != nil {
		return err
	}
	return s.SyncKnowledgeFAQ(req.ID)
}

func (s *productService) Delete(id int64, operator *dto.AuthPrincipal) error {
	if operator == nil {
		return errorsx.UnauthorizedI18n("error.auth.expired")
	}
	item := s.Get(id)
	if item == nil {
		return errorsx.InvalidParam("product not found")
	}
	if err := repositories.ProductRepository.Updates(sqls.DB(), id, map[string]any{
		"status":           enums.StatusDeleted,
		"updated_at":       time.Now(),
		"update_user_id":   operator.UserID,
		"update_user_name": operator.Username,
	}); err != nil {
		return err
	}
	if item.KnowledgeFAQID > 0 {
		if err := KnowledgeFAQService.DeleteKnowledgeFAQ(item.KnowledgeFAQID); err != nil {
			slog.Error("failed to delete product knowledge faq", "productId", id, "faqId", item.KnowledgeFAQID, "error", err)
		}
	}
	return nil
}

func (s *productService) Reindex(id int64) error {
	return s.SyncKnowledgeFAQ(id)
}

func (s *productService) SeedMuseProducts(operator *dto.AuthPrincipal) error {
	return s.SeedTemplateProducts("muse_bedding", operator)
}

func (s *productService) SeedTemplateProducts(templateCode string, operator *dto.AuthPrincipal) error {
	if operator == nil {
		return errorsx.UnauthorizedI18n("error.auth.expired")
	}
	seeds, faqs, remark, err := productTemplateSeeds(templateCode)
	if err != nil {
		return err
	}
	if err := s.UpsertTemplateProducts(seeds, operator); err != nil {
		return err
	}
	return s.seedGuideFAQs(faqs, remark, operator)
}

func (s *productService) UpsertTemplateProducts(seeds []request.SaveProductRequest, operator *dto.AuthPrincipal) error {
	if operator == nil {
		return errorsx.UnauthorizedI18n("error.auth.expired")
	}
	for _, req := range seeds {
		existing := repositories.ProductRepository.FindOne(sqls.DB(), sqls.NewCnd().Eq("name", req.Name).Where("status <> ?", enums.StatusDeleted))
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

func (s *productService) ImportCSV(reader io.Reader, operator *dto.AuthPrincipal) (response.ProductImportResultResponse, error) {
	ret := response.ProductImportResultResponse{Errors: make([]response.ProductImportRowResponse, 0)}
	if operator == nil {
		return ret, errorsx.UnauthorizedI18n("error.auth.expired")
	}
	rows, err := parseProductCSV(reader)
	if err != nil {
		return ret, err
	}
	for _, row := range rows {
		ret.Total++
		req, err := buildProductImportRequest(row.Values)
		if err != nil {
			ret.Failed++
			ret.Errors = append(ret.Errors, response.ProductImportRowResponse{Row: row.Row, Message: err.Error()})
			continue
		}
		existing := repositories.ProductRepository.FindOne(sqls.DB(), sqls.NewCnd().Eq("name", req.Name).Where("status <> ?", enums.StatusDeleted))
		if existing == nil {
			if _, err := s.Create(req, operator); err != nil {
				ret.Failed++
				ret.Errors = append(ret.Errors, response.ProductImportRowResponse{Row: row.Row, Message: err.Error()})
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
			ret.Errors = append(ret.Errors, response.ProductImportRowResponse{Row: row.Row, Message: err.Error()})
			continue
		}
		ret.Updated++
	}
	if ret.Total == 0 {
		ret.Skipped = 0
	}
	return ret, nil
}

func (s *productService) SyncKnowledgeFAQ(productID int64) error {
	item := s.Get(productID)
	if item == nil {
		return errorsx.InvalidParam("product not found")
	}
	kbID, err := s.resolveKnowledgeBaseID(item.KnowledgeBaseID)
	if err != nil {
		return err
	}
	question, answer, similarQuestions, remark := BuildProductKnowledgeFAQContent(item)
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
		if err := repositories.ProductRepository.Updates(sqls.DB(), item.ID, map[string]any{
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
			if err := repositories.ProductRepository.Updates(sqls.DB(), item.ID, map[string]any{
				"knowledge_base_id": kbID,
				"updated_at":        now,
			}); err != nil {
				return err
			}
		}
	}
	if err := rag.Index.IndexFAQByID(context.Background(), faq.ID); err != nil {
		return err
	}
	return nil
}

func (s *productService) buildProductModel(req request.SaveProductRequest) (*models.Product, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, errorsx.InvalidParam("product name is required")
	}
	status := enums.Status(req.Status)
	if req.Status == 0 {
		status = enums.StatusOk
	}
	if !enums.IsValidStatus(int(status)) || status == enums.StatusDeleted {
		return nil, errorsx.InvalidParam("invalid product status")
	}
	if req.PriceMin < 0 || req.PriceMax < 0 {
		return nil, errorsx.InvalidParam("product price must be greater than or equal to 0")
	}
	if req.PriceMax > 0 && req.PriceMin > req.PriceMax {
		return nil, errorsx.InvalidParam("product min price cannot exceed max price")
	}
	kbID := req.KnowledgeBaseID
	if kbID > 0 {
		if _, err := s.resolveKnowledgeBaseID(kbID); err != nil {
			return nil, err
		}
	}
	return &models.Product{
		Name:               name,
		Category:           strings.TrimSpace(req.Category),
		PriceMin:           req.PriceMin,
		PriceMax:           req.PriceMax,
		SellingPoints:      strings.TrimSpace(req.SellingPoints),
		SuitablePeople:     strings.TrimSpace(req.SuitablePeople),
		UnsuitablePeople:   strings.TrimSpace(req.UnsuitablePeople),
		Scenarios:          strings.TrimSpace(req.Scenarios),
		Specs:              strings.TrimSpace(req.Specs),
		IndustryAttributes: strings.TrimSpace(req.IndustryAttributes),
		ImageURL:           strings.TrimSpace(req.ImageURL),
		Priority:           req.Priority,
		KnowledgeBaseID:    kbID,
		Status:             status,
		Remark:             strings.TrimSpace(req.Remark),
	}, nil
}

type productCSVRow struct {
	Row    int
	Values map[string]string
}

func parseProductCSV(reader io.Reader) ([]productCSVRow, error) {
	csvReader := csv.NewReader(reader)
	csvReader.TrimLeadingSpace = true
	csvReader.FieldsPerRecord = -1
	records, err := csvReader.ReadAll()
	if err != nil {
		return nil, errorsx.InvalidParam("invalid product csv file")
	}
	if len(records) == 0 {
		return nil, errorsx.InvalidParam("product csv is empty")
	}
	headers := normalizeProductCSVHeaders(records[0])
	if len(headers) == 0 {
		return nil, errorsx.InvalidParam("product csv header is empty")
	}
	rows := make([]productCSVRow, 0, len(records)-1)
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
		rows = append(rows, productCSVRow{Row: index + 2, Values: values})
	}
	return rows, nil
}

func normalizeProductCSVHeaders(raw []string) []string {
	headers := make([]string, 0, len(raw))
	for _, item := range raw {
		key := strings.TrimSpace(strings.TrimPrefix(item, "\ufeff"))
		key = strings.ToLower(strings.ReplaceAll(key, " ", ""))
		switch key {
		case "产品名称", "名称", "name", "productname":
			headers = append(headers, "name")
		case "品类", "分类", "category":
			headers = append(headers, "category")
		case "最低价", "最低价格", "pricemin", "minprice":
			headers = append(headers, "priceMin")
		case "最高价", "最高价格", "pricemax", "maxprice":
			headers = append(headers, "priceMax")
		case "核心卖点", "卖点", "sellingpoints", "sellingpoint":
			headers = append(headers, "sellingPoints")
		case "适合人群", "suitablepeople":
			headers = append(headers, "suitablePeople")
		case "不适合人群", "unsuitablepeople":
			headers = append(headers, "unsuitablePeople")
		case "使用场景", "场景", "scenarios", "scenario":
			headers = append(headers, "scenarios")
		case "规格参数", "规格", "specs", "spec":
			headers = append(headers, "specs")
		case "行业属性", "扩展属性", "行业扩展属性", "industryattributes", "attributes":
			headers = append(headers, "industryAttributes")
		case "图片链接", "图片", "imageurl", "image":
			headers = append(headers, "imageUrl")
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

func buildProductImportRequest(values map[string]string) (request.SaveProductRequest, error) {
	req := request.SaveProductRequest{
		Name:               strings.TrimSpace(values["name"]),
		Category:           strings.TrimSpace(values["category"]),
		SellingPoints:      strings.TrimSpace(values["sellingPoints"]),
		SuitablePeople:     strings.TrimSpace(values["suitablePeople"]),
		UnsuitablePeople:   strings.TrimSpace(values["unsuitablePeople"]),
		Scenarios:          strings.TrimSpace(values["scenarios"]),
		Specs:              strings.TrimSpace(values["specs"]),
		IndustryAttributes: strings.TrimSpace(values["industryAttributes"]),
		ImageURL:           strings.TrimSpace(values["imageUrl"]),
		Remark:             strings.TrimSpace(values["remark"]),
		Status:             int(enums.StatusOk),
	}
	var err error
	if req.PriceMin, err = parseProductImportInt64(values["priceMin"], "最低价"); err != nil {
		return req, err
	}
	if req.PriceMax, err = parseProductImportInt64(values["priceMax"], "最高价"); err != nil {
		return req, err
	}
	priority, err := parseProductImportInt(values["priority"], "推荐优先级")
	if err != nil {
		return req, err
	}
	req.Priority = priority
	if req.KnowledgeBaseID, err = parseProductImportInt64(values["knowledgeBaseId"], "知识库ID"); err != nil {
		return req, err
	}
	if status, ok, err := parseProductImportStatus(values["status"]); err != nil {
		return req, err
	} else if ok {
		req.Status = int(status)
	}
	if strings.TrimSpace(req.Name) == "" {
		return req, fmt.Errorf("产品名称不能为空")
	}
	return req, nil
}

func parseProductImportInt(value string, field string) (int, error) {
	if strings.TrimSpace(value) == "" {
		return 0, nil
	}
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return 0, fmt.Errorf("%s必须是整数", field)
	}
	return parsed, nil
}

func parseProductImportInt64(value string, field string) (int64, error) {
	if strings.TrimSpace(value) == "" {
		return 0, nil
	}
	parsed, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%s必须是整数", field)
	}
	return parsed, nil
}

func parseProductImportStatus(value string) (enums.Status, bool, error) {
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

func (s *productService) resolveKnowledgeBaseID(id int64) (int64, error) {
	if id > 0 {
		kb := repositories.KnowledgeBaseRepository.Get(sqls.DB(), id)
		if kb == nil || kb.Status == enums.StatusDeleted || kb.KnowledgeType != string(enums.KnowledgeBaseTypeFAQ) {
			return 0, errorsx.InvalidParam("usable FAQ knowledge base not found")
		}
		return id, nil
	}
	kb := repositories.KnowledgeBaseRepository.FindOne(sqls.DB(), sqls.NewCnd().Eq("knowledge_type", string(enums.KnowledgeBaseTypeFAQ)).Where("status <> ?", enums.StatusDeleted).Asc("id"))
	if kb == nil {
		return 0, errorsx.InvalidParam("create a FAQ knowledge base before syncing products")
	}
	return kb.ID, nil
}

func (s *productService) seedMuseGuideFAQs(operator *dto.AuthPrincipal) error {
	return s.seedGuideFAQs(museGuideFAQSeeds(), "muse-guide-seed", operator)
}

func (s *productService) seedGuideFAQs(items []guideFAQSeed, remark string, operator *dto.AuthPrincipal) error {
	kbID, err := s.resolveKnowledgeBaseID(0)
	if err != nil {
		return err
	}
	for _, item := range items {
		if err := s.upsertGuideFAQ(kbID, item.question, item.answer, item.similarQuestions, remark, operator); err != nil {
			return err
		}
	}
	return nil
}

type guideFAQSeed struct {
	question         string
	answer           string
	similarQuestions []string
}

func (s *productService) upsertGuideFAQ(kbID int64, question string, answer string, similarQuestions []string, remark string, operator *dto.AuthPrincipal) error {
	similarJSON, err := json.Marshal(similarQuestions)
	if err != nil {
		return err
	}
	now := time.Now()
	existing := repositories.KnowledgeFAQRepository.FindByKnowledgeBaseIDAndQuestions(sqls.DB(), kbID, []string{question})
	if len(existing) == 0 {
		item := &models.KnowledgeFAQ{
			KnowledgeBaseID:  kbID,
			Question:         question,
			Answer:           answer,
			SimilarQuestions: string(similarJSON),
			IndexStatus:      enums.KnowledgeDocumentIndexStatusPending,
			Status:           enums.StatusOk,
			Remark:           remark,
			AuditFields:      utils.BuildAuditFields(operator),
		}
		if err := repositories.KnowledgeFAQRepository.Create(sqls.DB(), item); err != nil {
			return err
		}
		return rag.Index.IndexFAQByID(context.Background(), item.ID)
	}
	item := existing[0]
	if err := repositories.KnowledgeFAQRepository.Updates(sqls.DB(), item.ID, map[string]any{
		"answer":            answer,
		"similar_questions": string(similarJSON),
		"index_status":      enums.KnowledgeDocumentIndexStatusPending,
		"indexed_at":        nil,
		"index_error":       "",
		"status":            enums.StatusOk,
		"remark":            remark,
		"updated_at":        now,
		"update_user_id":    operator.UserID,
		"update_user_name":  operator.Username,
	}); err != nil {
		return err
	}
	return rag.Index.IndexFAQByID(context.Background(), item.ID)
}

func BuildProductKnowledgeFAQContent(item *models.Product) (question string, answer string, similarQuestions []string, remark string) {
	price := "未设置"
	if item.PriceMin > 0 && item.PriceMax > 0 {
		price = fmt.Sprintf("%d-%d元", item.PriceMin, item.PriceMax)
	} else if item.PriceMin > 0 {
		price = fmt.Sprintf("%d元起", item.PriceMin)
	} else if item.PriceMax > 0 {
		price = fmt.Sprintf("%d元左右", item.PriceMax)
	}
	category := strings.TrimSpace(item.Category)
	if category == "" {
		category = "产品"
	}
	lines := []string{
		fmt.Sprintf("产品名称：%s", item.Name),
		fmt.Sprintf("品类：%s", category),
		fmt.Sprintf("价格区间：%s", price),
	}
	appendField := func(label string, value string) {
		if v := strings.TrimSpace(value); v != "" {
			lines = append(lines, fmt.Sprintf("%s：%s", label, v))
		}
	}
	appendField("核心卖点", item.SellingPoints)
	appendField("适合人群", item.SuitablePeople)
	appendField("不适合人群", item.UnsuitablePeople)
	appendField("使用场景", item.Scenarios)
	appendField("规格参数", item.Specs)
	appendField("行业扩展属性", item.IndustryAttributes)
	lines = append(lines, "推荐话术：当客户预算、睡眠问题或使用场景与适合人群匹配时，可优先推荐该产品；如客户条件落入不适合人群，应主动说明并推荐其他款式。")
	return "产品推荐：" + item.Name,
		strings.Join(lines, "\n"),
		[]string{
			item.Name,
			item.Name + "适合什么人",
			item.Name + "价格",
			category + "推荐",
			"预算内怎么选" + category,
		},
		"product:" + fmt.Sprint(item.ID)
}

func museProductSeeds() []request.SaveProductRequest {
	return []request.SaveProductRequest{
		{Name: "慕斯脊护支撑款", Category: "床垫", PriceMin: 12000, PriceMax: 18000, SellingPoints: "分区承托、偏硬支撑、护脊睡感，适合重视腰背支撑的家庭。", SuitablePeople: "腰背压力明显、喜欢偏硬睡感、老人或长期久坐人群。", UnsuitablePeople: "明确偏好很软包裹感的客户。", Scenarios: "主卧、老人房、改善腰背支撑。", Specs: "常见规格：1.5m、1.8m；可结合门店库存确认。", IndustryAttributes: "睡感：偏硬；支撑：分区承托；常问尺寸：1.5m/1.8m；到店体验：建议试躺确认软硬。", Priority: 90, Status: int(enums.StatusOk), Remark: "慕斯寝具模拟样板产品"},
		{Name: "慕斯云感舒睡款", Category: "床垫", PriceMin: 8000, PriceMax: 13000, SellingPoints: "柔和释压、包裹感好、适合日常舒适睡眠升级。", SuitablePeople: "年轻夫妻、侧睡较多、喜欢柔软贴合感的客户。", UnsuitablePeople: "强烈要求硬支撑或体重较大的客户。", Scenarios: "主卧、婚房、租房升级。", Specs: "常见规格：1.5m、1.8m。", IndustryAttributes: "睡感：偏软包裹；支撑：舒适释压；常问尺寸：1.5m/1.8m。", Priority: 80, Status: int(enums.StatusOk), Remark: "慕斯寝具模拟样板产品"},
		{Name: "慕斯儿童成长款", Category: "儿童床垫", PriceMin: 5000, PriceMax: 9000, SellingPoints: "软硬适中、支撑成长发育，面料亲肤。", SuitablePeople: "儿童房、青少年成长阶段、家长关注支撑和环保。", UnsuitablePeople: "成人大体重长期使用。", Scenarios: "儿童房、学生房。", Specs: "常见规格：1.2m、1.5m。", IndustryAttributes: "年龄段：儿童/青少年；睡感：软硬适中；关注点：成长支撑、亲肤面料。", Priority: 70, Status: int(enums.StatusOk), Remark: "慕斯寝具模拟样板产品"},
		{Name: "慕斯智能电动床", Category: "电动床", PriceMin: 16000, PriceMax: 28000, SellingPoints: "头脚升降、阅读观影模式、适合改善睡前休息体验。", SuitablePeople: "老人、孕妇、喜欢床上阅读观影、关注起身便利的人群。", UnsuitablePeople: "预算较低或只需要基础床架的客户。", Scenarios: "主卧、老人房、康养场景。", Specs: "建议与适配床垫组合选购。", IndustryAttributes: "功能：头脚升降、阅读观影模式；搭配：建议确认适配床垫；体验：到店演示更清楚。", Priority: 85, Status: int(enums.StatusOk), Remark: "慕斯寝具模拟样板产品"},
	}
}

func productTemplateSeeds(templateCode string) ([]request.SaveProductRequest, []guideFAQSeed, string, error) {
	switch strings.TrimSpace(templateCode) {
	case "", "muse_bedding", "muse":
		return museProductSeeds(), museGuideFAQSeeds(), "muse-guide-seed", nil
	case "oral_clinic":
		return oralClinicProductSeeds(), oralClinicGuideFAQSeeds(), "oral-clinic-guide-seed", nil
	case "kids_english":
		return kidsEnglishProductSeeds(), kidsEnglishGuideFAQSeeds(), "kids-english-guide-seed", nil
	case "finance_advisor":
		return financeAdvisorProductSeeds(), financeAdvisorGuideFAQSeeds(), "finance-advisor-guide-seed", nil
	case "home_decoration":
		return homeDecorationProductSeeds(), homeDecorationGuideFAQSeeds(), "home-decoration-guide-seed", nil
	default:
		return nil, nil, "", errorsx.InvalidParam("unsupported digital store template")
	}
}

func museGuideFAQSeeds() []guideFAQSeed {
	return []guideFAQSeed{
		{
			question: "主推床垫有哪些",
			answer: strings.Join([]string{
				"慕斯寝具模拟门店当前主推：",
				"1. 慕斯脊护支撑款：预算12000-18000元，分区承托、偏硬支撑、护脊睡感，适合老人、腰背压力明显、喜欢稳定支撑的客户。",
				"2. 慕斯云感舒睡款：预算8000-13000元，柔和释压、包裹感好，适合年轻夫妻、侧睡较多、喜欢柔软贴合感的客户。",
				"3. 慕斯儿童成长款：预算5000-9000元，软硬适中、支撑成长发育，适合儿童房和青少年成长阶段。",
				"4. 慕斯智能电动床：预算16000-28000元，头脚升降、阅读观影模式，适合老人、孕妇、起身便利和康养场景。",
				"导购原则：先确认预算、睡感偏好、身高体重、是否腰背不适、使用房间和到店试躺时间，再给出1-2个主推方案。",
			}, "\n"),
			similarQuestions: []string{"慕斯主推产品", "床垫推荐", "一万五预算床垫怎么选", "门店有哪些主推款"},
		},
		{
			question: "如何根据人群推荐床垫",
			answer: strings.Join([]string{
				"按人群推荐慕斯产品：",
				"老人、腰不好、久坐人群：优先推荐慕斯脊护支撑款，预算12000-18000元，重点说明分区承托、偏硬支撑和到店试躺确认。",
				"年轻夫妻、侧睡多、喜欢包裹感：优先推荐慕斯云感舒睡款，预算8000-13000元，重点说明柔和释压和舒适睡感。",
				"儿童青少年：优先推荐慕斯儿童成长款，预算5000-9000元，重点说明软硬适中、成长支撑和儿童房使用。",
				"老人房、孕妇、需要起身便利、阅读观影：可推荐慕斯智能电动床，预算16000-28000元，建议搭配适配床垫。",
				"如果客户预算约15000且提到老人或腰背不适，首推慕斯脊护支撑款；如果客户愿意升级康养体验，再追加介绍慕斯智能电动床组合方案。",
			}, "\n"),
			similarQuestions: []string{"老人腰不好推荐哪款床垫", "腰背不适床垫推荐", "一万五老人床垫", "按人群怎么推荐慕斯床垫"},
		},
	}
}

func oralClinicProductSeeds() []request.SaveProductRequest {
	return []request.SaveProductRequest{
		{Name: "隐形矫正初诊评估", Category: "正畸服务", PriceMin: 0, PriceMax: 0, SellingPoints: "由正畸医生面诊评估牙列情况、拍片检查后给出矫正方案方向。", SuitablePeople: "牙齿拥挤、牙缝、龅牙、地包天、希望了解隐形矫正周期和预算的人群。", UnsuitablePeople: "急性口腔炎症未处理、需要先完成基础治疗的人群。", Scenarios: "正畸咨询、方案评估、预算了解。", Specs: "最终方案、周期和费用需以医生面诊和影像检查为准。", IndustryAttributes: "诊疗项目：正畸初诊；检查资料：口扫/拍片/医生面诊；费用周期：面诊后确认。", Priority: 95, Status: int(enums.StatusOk), Remark: "口腔门诊模拟样板服务"},
		{Name: "种植牙方案咨询", Category: "种植修复", PriceMin: 0, PriceMax: 0, SellingPoints: "医生根据缺牙位置、骨量、口腔健康状况评估种植修复可行性。", SuitablePeople: "单颗/多颗缺牙、活动假牙不适、希望了解种植牙方案的人群。", UnsuitablePeople: "严重全身疾病控制不佳或口腔炎症未处理者需先由医生评估。", Scenarios: "缺牙修复咨询、种植方案预约。", Specs: "费用、品牌和手术安排需面诊确认，不在线承诺效果。", IndustryAttributes: "诊疗项目：种植修复咨询；检查资料：口腔检查/影像评估；禁用口径：不承诺一定能种。", Priority: 90, Status: int(enums.StatusOk), Remark: "口腔门诊模拟样板服务"},
		{Name: "儿童涂氟与窝沟封闭", Category: "儿童齿科", PriceMin: 200, PriceMax: 800, SellingPoints: "帮助儿童建立预防龋齿管理习惯，医生检查后确认是否适合涂氟或窝沟封闭。", SuitablePeople: "3岁以上儿童、换牙期儿童、家长关注龋齿预防的人群。", UnsuitablePeople: "已出现明显疼痛或龋洞时需先检查治疗。", Scenarios: "儿童口腔检查、龋齿预防、家长咨询。", Specs: "具体项目和次数由儿童牙医检查后确认。", IndustryAttributes: "诊疗项目：儿童预防齿科；年龄：3岁以上更常见；是否适合：医生检查后确认。", Priority: 80, Status: int(enums.StatusOk), Remark: "口腔门诊模拟样板服务"},
		{Name: "舒适洁牙套餐", Category: "牙周护理", PriceMin: 300, PriceMax: 900, SellingPoints: "适合日常牙结石、牙渍清洁和牙周基础维护，洁牙前由医生检查口腔情况。", SuitablePeople: "半年到一年未洁牙、牙结石明显、口气困扰、备孕或正畸前检查人群。", UnsuitablePeople: "急性牙龈肿痛、严重牙周问题需医生先评估。", Scenarios: "洁牙预约、口腔基础护理。", Specs: "洁牙方式、是否需要牙周治疗需到店检查确认。", IndustryAttributes: "诊疗项目：洁牙/牙周基础护理；频次建议：需结合口腔情况；急症：先医生评估。", Priority: 75, Status: int(enums.StatusOk), Remark: "口腔门诊模拟样板服务"},
	}
}

func oralClinicGuideFAQSeeds() []guideFAQSeed {
	return []guideFAQSeed{
		{
			question: "口腔门诊主推服务有哪些",
			answer: strings.Join([]string{
				"口腔门诊模拟样板当前主推：",
				"1. 隐形矫正初诊评估：适合牙齿拥挤、牙缝、龅牙、地包天或想了解矫正周期预算的客户，需医生面诊和影像检查后确认方案。",
				"2. 种植牙方案咨询：适合缺牙、活动假牙不适或想了解种植修复的人群，费用和可行性需医生评估。",
				"3. 儿童涂氟与窝沟封闭：适合关注儿童龋齿预防的家长，需儿童牙医检查后确认。",
				"4. 舒适洁牙套餐：适合牙结石、牙渍、口气困扰或半年以上未洁牙的人群。",
				"导购原则：先确认客户症状、年龄、期望、预算、就诊时间和联系方式，再引导预约初诊；不得承诺治疗效果、最低价或无需检查即可确定方案。",
			}, "\n"),
			similarQuestions: []string{"口腔门诊有哪些项目", "牙齿矫正怎么咨询", "种植牙怎么预约", "儿童涂氟适合吗", "洁牙多少钱"},
		},
		{
			question: "口腔咨询如何合规回复",
			answer: strings.Join([]string{
				"口腔咨询合规口径：",
				"牙痛、牙龈出血、缺牙、牙齿不齐、儿童龋齿等问题，可以先做基础解释和就诊建议，但不能在线诊断。",
				"涉及治疗方案、周期、费用、是否适合种植/矫正、是否需要拔牙，必须说明需要医生面诊、拍片或口腔检查后确认。",
				"客户留下手机号或微信、明确要预约、询问医生时间或最终费用时，应引导留资并安排前台/顾问联系。",
				"不得承诺无痛、包治好、一次解决、百分百成功、最低价、当天一定能做。",
			}, "\n"),
			similarQuestions: []string{"牙疼怎么办", "能不能保证矫正效果", "种植牙是不是一定能做", "洁牙会不会伤牙", "口腔咨询禁用承诺"},
		},
	}
}

func kidsEnglishProductSeeds() []request.SaveProductRequest {
	return []request.SaveProductRequest{
		{Name: "自然拼读进阶班", Category: "少儿英语", PriceMin: 3600, PriceMax: 6800, SellingPoints: "系统学习字母组合、拼读规则和高频词，帮助孩子提升自主阅读基础。", SuitablePeople: "小学低年级、能识别基础字母、想提升阅读启蒙的学生。", UnsuitablePeople: "零基础低龄儿童需先做入门测评。", Scenarios: "英语启蒙、阅读基础、寒暑假提升。", Specs: "常见班型：8-12人小班；课时和开班时间以校区确认为准。", IndustryAttributes: "年级：小学低年级；目标：自然拼读/阅读启蒙；班型：小班；试听：建议先测评。", Priority: 95, Status: int(enums.StatusOk), Remark: "教育培训模拟样板课程"},
		{Name: "剑桥少儿英语能力班", Category: "少儿英语", PriceMin: 6800, PriceMax: 12800, SellingPoints: "围绕听说读写和阶段测评训练，适合有一定基础的孩子持续提升。", SuitablePeople: "小学中高年级、希望系统提升听说读写和阶段测评能力的学生。", UnsuitablePeople: "只想短期保分或要求固定提分结果的客户。", Scenarios: "校内英语提升、能力测评、长期课程规划。", Specs: "最终班型、课时和学费需课程顾问结合测评确认。", IndustryAttributes: "年级：小学中高年级；目标：听说读写综合；禁用：不承诺保过提分。", Priority: 90, Status: int(enums.StatusOk), Remark: "教育培训模拟样板课程"},
		{Name: "一对一学习规划咨询", Category: "学习规划", PriceMin: 0, PriceMax: 0, SellingPoints: "课程顾问根据学生基础、目标和时间安排，给出试听与课程规划建议。", SuitablePeople: "目标不明确、需要先测评或家长想了解课程体系的客户。", UnsuitablePeople: "要求直接承诺升学、录取或固定分数结果的客户。", Scenarios: "入学测评、课程规划、试听预约。", Specs: "需留下学生年级、学习目标、联系电话和方便沟通时间。", IndustryAttributes: "咨询类型：测评/规划；留资字段：年级、目标、手机号、试听时间。", Priority: 85, Status: int(enums.StatusOk), Remark: "教育培训模拟样板服务"},
	}
}

func kidsEnglishGuideFAQSeeds() []guideFAQSeed {
	return []guideFAQSeed{
		{
			question: "少儿英语课程怎么推荐",
			answer: strings.Join([]string{
				"少儿英语样板课程推荐原则：",
				"1. 小学低年级、想提升阅读启蒙：优先推荐自然拼读进阶班。",
				"2. 小学中高年级、希望系统提升听说读写：推荐剑桥少儿英语能力班。",
				"3. 目标不明确或基础不清楚：先推荐一对一学习规划咨询或试听测评。",
				"导购时先问学生年级、英语基础、学习目标、可上课时间和家长联系方式；不得承诺保过、固定提分或录取结果。",
			}, "\n"),
			similarQuestions: []string{"孩子英语怎么选课", "自然拼读适合几年级", "英语试听怎么预约", "课程怎么推荐"},
		},
		{
			question: "教育培训咨询有哪些禁用承诺",
			answer: strings.Join([]string{
				"教育培训咨询禁用口径：",
				"不得承诺保过、固定提分、包录取、证书包拿、名师一定授课、课程名额一定保留。",
				"学费、退费、合同、老师资质、具体排课和考试政策都需要课程顾问人工确认。",
				"客户留下手机号、学生年级、试听时间或询问最终学费时，应安排课程顾问跟进。",
			}, "\n"),
			similarQuestions: []string{"能不能保过", "能提高多少分", "退费政策", "老师资质", "学费多少"},
		},
	}
}

func financeAdvisorProductSeeds() []request.SaveProductRequest {
	return []request.SaveProductRequest{
		{Name: "经营贷资质初评", Category: "贷款咨询", PriceMin: 0, PriceMax: 0, SellingPoints: "根据企业经营年限、流水、资产和征信情况做基础资料清单说明。", SuitablePeople: "小微企业主、个体工商户、需要经营周转资金的客户。", UnsuitablePeople: "要求必批、最低利率或不愿做资质审核的客户。", Scenarios: "经营贷咨询、资料准备、顾问回访。", Specs: "额度、利率和审批结果需以持牌机构审核为准。", IndustryAttributes: "类型：贷款咨询；关键字段：城市、资金用途、企业情况；禁用：不承诺必批/额度/利率。", Priority: 95, Status: int(enums.StatusOk), Remark: "金融服务模拟样板"},
		{Name: "家庭保障方案咨询", Category: "保险咨询", PriceMin: 0, PriceMax: 0, SellingPoints: "围绕家庭成员、预算、保障缺口和缴费能力做基础保障方向说明。", SuitablePeople: "家庭保障规划、重疾/医疗/意外保障咨询客户。", UnsuitablePeople: "要求保证理赔、收益或绕过健康告知的客户。", Scenarios: "保险咨询、保障规划、顾问预约。", Specs: "具体产品、条款、费率和承保结论需持牌顾问确认。", IndustryAttributes: "类型：保险咨询；关键字段：家庭成员、预算、保障目标；禁用：不承诺理赔/收益。", Priority: 85, Status: int(enums.StatusOk), Remark: "金融服务模拟样板"},
		{Name: "资产配置风险测评预约", Category: "理财咨询", PriceMin: 0, PriceMax: 0, SellingPoints: "引导客户先完成风险偏好、资金期限和流动性需求确认。", SuitablePeople: "希望了解资产配置、现金管理或长期规划的客户。", UnsuitablePeople: "只追求保本高收益、拒绝风险测评的客户。", Scenarios: "理财咨询、风险测评、顾问沟通。", Specs: "收益、风险等级和适配方案需以合规测评和合同为准。", IndustryAttributes: "类型：理财咨询；关键字段：风险偏好、期限、资金用途；禁用：不承诺收益/保本。", Priority: 80, Status: int(enums.StatusOk), Remark: "金融服务模拟样板"},
	}
}

func financeAdvisorGuideFAQSeeds() []guideFAQSeed {
	return []guideFAQSeed{
		{
			question: "金融咨询如何合规承接",
			answer: strings.Join([]string{
				"金融服务样板咨询原则：",
				"经营贷、保险和资产配置都只能做基础咨询和资料清单说明。",
				"涉及额度、利率、收益、保本、合同条款、风险评级、承保或审批结果，必须转持牌顾问人工确认。",
				"不得索要银行卡密码、验证码、完整证件影像等高敏信息；客户表达办理意向时引导留下手机号和方便沟通时间。",
			}, "\n"),
			similarQuestions: []string{"贷款能批多少", "利率最低多少", "理财保本吗", "保险能不能赔", "金融咨询怎么转人工"},
		},
	}
}

func homeDecorationProductSeeds() []request.SaveProductRequest {
	return []request.SaveProductRequest{
		{Name: "全案设计咨询", Category: "装修设计", PriceMin: 0, PriceMax: 0, SellingPoints: "根据户型面积、风格偏好、预算和居住需求，安排设计师初步沟通。", SuitablePeople: "准备装修、需要整体风格规划或空间改造的客户。", UnsuitablePeople: "要求不量房直接给最终报价的客户。", Scenarios: "装修咨询、设计预约、量房前沟通。", Specs: "最终方案和报价需量房、设计沟通和合同确认。", IndustryAttributes: "项目：全案设计；关键字段：面积、预算、风格、交房时间；禁用：不承诺最终价。", Priority: 95, Status: int(enums.StatusOk), Remark: "家装装修模拟样板服务"},
		{Name: "整装施工套餐咨询", Category: "整装施工", PriceMin: 80000, PriceMax: 300000, SellingPoints: "覆盖基础施工、主材选择和项目管理，适合希望省心整装的客户。", SuitablePeople: "新房装修、旧房翻新、希望设计施工一体化的客户。", UnsuitablePeople: "要求绝不增项、固定工期或未量房先签最终价的客户。", Scenarios: "整装咨询、报价初筛、设计师跟进。", Specs: "价格区间仅为样板范围，实际以量房、材料和合同为准。", IndustryAttributes: "项目：整装施工；关注：主材、工期、增项；禁用：绝不增项/固定工期。", Priority: 90, Status: int(enums.StatusOk), Remark: "家装装修模拟样板服务"},
		{Name: "旧房翻新评估", Category: "旧改翻新", PriceMin: 50000, PriceMax: 180000, SellingPoints: "关注拆改、水电、收纳和居住动线，需结合房龄与现场情况评估。", SuitablePeople: "老房翻新、局部改造、改善收纳和居住体验的客户。", UnsuitablePeople: "需要结构改动但不愿现场评估的客户。", Scenarios: "旧房改造、局部翻新、量房预约。", Specs: "拆改、工期和预算需设计师现场确认。", IndustryAttributes: "项目：旧房翻新；关键字段：房龄、面积、改造范围；风险：拆改和增项需人工确认。", Priority: 82, Status: int(enums.StatusOk), Remark: "家装装修模拟样板服务"},
	}
}

func homeDecorationGuideFAQSeeds() []guideFAQSeed {
	return []guideFAQSeed{
		{
			question: "装修客户怎么推荐服务",
			answer: strings.Join([]string{
				"家装装修样板推荐原则：",
				"1. 还没明确方案：先推荐全案设计咨询，确认面积、预算、风格、交房时间。",
				"2. 想省心整装：推荐整装施工套餐咨询，但说明最终报价需量房和合同确认。",
				"3. 老房或局部改造：推荐旧房翻新评估，重点追问房龄、改造范围和是否可现场量房。",
				"不得承诺一口价、绝不增项、固定工期、材料绝对环保或赔付金额。",
			}, "\n"),
			similarQuestions: []string{"装修怎么报价", "100平怎么装修", "旧房翻新多少钱", "能不能不增项", "预约量房"},
		},
	}
}
