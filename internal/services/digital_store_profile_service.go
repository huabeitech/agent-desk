package services

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"agent-desk/internal/ai/rag"
	"agent-desk/internal/models"
	"agent-desk/internal/pkg/config"
	"agent-desk/internal/pkg/constants"
	"agent-desk/internal/pkg/dto"
	"agent-desk/internal/pkg/dto/request"
	"agent-desk/internal/pkg/dto/response"
	"agent-desk/internal/pkg/enums"
	"agent-desk/internal/pkg/errorsx"
	"agent-desk/internal/pkg/utils"
	"agent-desk/internal/repositories"

	"github.com/mlogclub/simple/sqls"
	"gorm.io/gorm"
)

const (
	digitalStoreProfileConfigKey  = "digital_store.profile"
	digitalStoreConfigGroup       = "digital_store"
	digitalStoreRuntimeSeedRemark = "digital-store-runtime-seed"
	digitalStoreDefaultTeamName   = "门店顾问组"
	digitalStoreDefaultAgentCode  = "digital_store_consultant"
)

var DigitalStoreProfileService = newDigitalStoreProfileService()

func newDigitalStoreProfileService() *digitalStoreProfileService {
	return &digitalStoreProfileService{}
}

type digitalStoreProfileService struct {
}

type digitalStoreProfileConfig struct {
	request.DigitalStoreProfileRequest
	KnowledgeFAQID    int64  `json:"knowledgeFAQId"`
	TemplateCode      string `json:"templateCode"`
	TemplateVersion   string `json:"templateVersion"`
	TemplateAppliedAt string `json:"templateAppliedAt"`
}

type digitalStoreTemplateBundle struct {
	template   response.DigitalStoreTemplateResponse
	cfg        digitalStoreProfileConfig
	products   []request.SaveProductRequest
	promotions []request.SavePromotionRequest
}

func (s *digitalStoreProfileService) GetProfile() response.DigitalStoreProfileResponse {
	cfg := s.loadConfig()
	item := repositories.SystemConfigRepository.Take(sqls.DB(), "config_key = ?", digitalStoreProfileConfigKey)
	ret := buildDigitalStoreProfileResponse(cfg)
	if item != nil {
		ret.UpdatedAt = utils.FormatTime(item.UpdatedAt)
	}
	return ret
}

func (s *digitalStoreProfileService) ListTemplates() []response.DigitalStoreTemplateResponse {
	return []response.DigitalStoreTemplateResponse{
		{
			Code:        "muse_bedding",
			Name:        "慕斯寝具门店",
			Industry:    "家居寝具",
			Version:     "1.0.0",
			Description: "适合床垫、睡眠产品、家居门店，包含推荐话术、预约试躺和活动权益样板。",
		},
		{
			Code:        "oral_clinic",
			Name:        "口腔门诊",
			Industry:    "口腔医疗",
			Version:     "1.0.0",
			Description: "适合口腔诊所咨询接待，包含合规提醒、正畸/种植/儿童齿科/洁牙服务样板。",
		},
		{
			Code:        "kids_english",
			Name:        "少儿英语培训",
			Industry:    "教育培训",
			Version:     "1.0.0",
			Description: "适合少儿英语、学科辅导和兴趣课程机构，包含试听预约、课程推荐和保过提分禁用口径。",
		},
		{
			Code:        "finance_advisor",
			Name:        "金融顾问咨询",
			Industry:    "金融服务",
			Version:     "1.0.0",
			Description: "适合贷款、保险、理财咨询和企业金融服务，包含风险提示、敏感信息边界和持牌顾问转人工。",
		},
		{
			Code:        "home_decoration",
			Name:        "家装装修门店",
			Industry:    "家装装修",
			Version:     "1.0.0",
			Description: "适合整装、设计施工和建材门店，包含量房预约、方案推荐、报价边界和施工售后风险。",
		},
	}
}

func digitalStoreTemplateMetadata(templateCode string) response.DigitalStoreTemplateResponse {
	templateCode = strings.TrimSpace(templateCode)
	for _, item := range DigitalStoreProfileService.ListTemplates() {
		if item.Code == templateCode || (templateCode == "" && item.Code == "muse_bedding") || (templateCode == "muse" && item.Code == "muse_bedding") {
			return item
		}
	}
	return response.DigitalStoreTemplateResponse{Code: templateCode}
}

func (s *digitalStoreProfileService) ExportTemplate(templateCode string) (response.DigitalStoreTemplateExportResponse, error) {
	bundle, err := s.buildBuiltinTemplateBundle(templateCode)
	if err != nil {
		return response.DigitalStoreTemplateExportResponse{}, err
	}
	return response.DigitalStoreTemplateExportResponse{
		SchemaVersion:   "1.0",
		ExportedAt:      utils.FormatTime(time.Now()),
		Template:        bundle.template,
		Profile:         buildDigitalStoreProfileResponse(bundle.cfg),
		Products:        buildDigitalStoreTemplateProductResponses(bundle.products),
		Promotions:      buildDigitalStoreTemplatePromotionResponses(bundle.promotions),
		RiskRules:       buildDigitalStoreIndustryRiskRuleResponses(bundle.cfg),
		AcceptanceItems: buildDigitalStoreAcceptanceItems(bundle.cfg),
	}, nil
}

func (s *digitalStoreProfileService) PreviewTemplate(templateCode string) (response.DigitalStoreTemplatePreviewResponse, error) {
	bundle, err := s.buildBuiltinTemplateBundle(templateCode)
	if err != nil {
		return response.DigitalStoreTemplatePreviewResponse{}, err
	}
	return s.previewTemplateBundle(bundle), nil
}

func (s *digitalStoreProfileService) PreviewImportedTemplate(req request.DigitalStoreTemplateImportRequest) (response.DigitalStoreTemplatePreviewResponse, error) {
	bundle, err := s.buildImportedTemplateBundle(req)
	if err != nil {
		return response.DigitalStoreTemplatePreviewResponse{}, err
	}
	return s.previewTemplateBundle(bundle), nil
}

func (s *digitalStoreProfileService) buildBuiltinTemplateBundle(templateCode string) (digitalStoreTemplateBundle, error) {
	templateCode = strings.TrimSpace(templateCode)
	if templateCode == "" {
		templateCode = "muse_bedding"
	}
	cfg, err := digitalStoreTemplateProfile(templateCode)
	if err != nil {
		return digitalStoreTemplateBundle{}, err
	}
	template := digitalStoreTemplateMetadata(templateCode)
	cfg.TemplateCode = template.Code
	cfg.TemplateVersion = template.Version
	products, _, _, err := productTemplateSeeds(templateCode)
	if err != nil {
		return digitalStoreTemplateBundle{}, err
	}
	promotions, err := promotionTemplateSeeds(templateCode, time.Now())
	if err != nil {
		return digitalStoreTemplateBundle{}, err
	}
	return digitalStoreTemplateBundle{template: template, cfg: cfg, products: products, promotions: promotions}, nil
}

func (s *digitalStoreProfileService) buildImportedTemplateBundle(req request.DigitalStoreTemplateImportRequest) (digitalStoreTemplateBundle, error) {
	template := response.DigitalStoreTemplateResponse{
		Code:        strings.TrimSpace(req.Template.Code),
		Name:        strings.TrimSpace(req.Template.Name),
		Industry:    strings.TrimSpace(req.Template.Industry),
		Version:     strings.TrimSpace(req.Template.Version),
		Description: strings.TrimSpace(req.Template.Description),
	}
	if template.Code == "" {
		return digitalStoreTemplateBundle{}, errorsx.InvalidParam("template code is required")
	}
	if template.Name == "" {
		template.Name = template.Code
	}
	if template.Version == "" {
		template.Version = "custom"
	}
	cfg, err := s.buildConfig(req.Profile)
	if err != nil {
		return digitalStoreTemplateBundle{}, err
	}
	if cfg.Industry == "" {
		cfg.Industry = template.Industry
	}
	if template.Industry == "" {
		template.Industry = cfg.Industry
	}
	cfg.TemplateCode = template.Code
	cfg.TemplateVersion = template.Version
	cfg.TemplateAppliedAt = ""
	products := make([]request.SaveProductRequest, 0, len(req.Products))
	for _, item := range req.Products {
		item.ID = 0
		item.Name = strings.TrimSpace(item.Name)
		if item.Name == "" {
			return digitalStoreTemplateBundle{}, errorsx.InvalidParam("template product name is required")
		}
		products = append(products, item)
	}
	promotions := make([]request.SavePromotionRequest, 0, len(req.Promotions))
	for _, item := range req.Promotions {
		item.ID = 0
		item.Name = strings.TrimSpace(item.Name)
		if item.Name == "" {
			return digitalStoreTemplateBundle{}, errorsx.InvalidParam("template promotion name is required")
		}
		promotions = append(promotions, item)
	}
	if len(products) == 0 && len(promotions) == 0 {
		return digitalStoreTemplateBundle{}, errorsx.InvalidParam("template must include products or promotions")
	}
	return digitalStoreTemplateBundle{template: template, cfg: cfg, products: products, promotions: promotions}, nil
}

func (s *digitalStoreProfileService) previewTemplateBundle(bundle digitalStoreTemplateBundle) response.DigitalStoreTemplatePreviewResponse {
	current := s.loadConfig()
	ret := response.DigitalStoreTemplatePreviewResponse{
		Template:        bundle.template,
		Profile:         buildDigitalStoreProfileResponse(bundle.cfg),
		ProfileAction:   "create",
		Products:        buildDigitalStoreTemplateProductPreviewItems(bundle.products),
		Promotions:      buildDigitalStoreTemplatePromotionPreviewItems(bundle.promotions),
		RiskRules:       buildDigitalStoreIndustryRiskRuleResponses(bundle.cfg),
		AcceptanceItems: buildDigitalStoreAcceptanceItems(bundle.cfg),
	}
	ret.Warnings = buildDigitalStoreTemplatePreviewWarnings(current, ret.Template)
	if current.Initialized {
		ret.ProfileAction = "update"
	}
	for _, item := range ret.Products {
		if item.Action == "update" {
			ret.ProductUpdateTotal++
		} else {
			ret.ProductCreateTotal++
		}
	}
	for _, item := range ret.Promotions {
		if item.Action == "update" {
			ret.PromotionUpdateTotal++
		} else {
			ret.PromotionCreateTotal++
		}
	}
	return ret
}

func buildDigitalStoreTemplateProductResponses(products []request.SaveProductRequest) []response.DigitalStoreTemplateProductResponse {
	ret := make([]response.DigitalStoreTemplateProductResponse, 0, len(products))
	for _, item := range products {
		ret = append(ret, response.DigitalStoreTemplateProductResponse{
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
			Status:             item.Status,
			Remark:             item.Remark,
		})
	}
	return ret
}

func buildDigitalStoreTemplateProductPreviewItems(products []request.SaveProductRequest) []response.DigitalStoreTemplatePreviewItem {
	ret := make([]response.DigitalStoreTemplatePreviewItem, 0, len(products))
	for _, item := range products {
		preview := response.DigitalStoreTemplatePreviewItem{
			Name:   item.Name,
			Action: "create",
			Reason: "按产品名称未找到现有记录，将新建产品并同步 FAQ。",
		}
		if existing := repositories.ProductRepository.FindOne(sqls.DB(), sqls.NewCnd().Eq("name", item.Name).Where("status <> ?", enums.StatusDeleted)); existing != nil {
			preview.Action = "update"
			preview.ExistingID = existing.ID
			preview.Reason = "按产品名称匹配到现有记录，将更新产品字段并重建 FAQ。"
		}
		ret = append(ret, preview)
	}
	return ret
}

func buildDigitalStoreTemplatePromotionResponses(promotions []request.SavePromotionRequest) []response.DigitalStoreTemplatePromotionResponse {
	ret := make([]response.DigitalStoreTemplatePromotionResponse, 0, len(promotions))
	for _, item := range promotions {
		ret = append(ret, response.DigitalStoreTemplatePromotionResponse{
			Name:               item.Name,
			PromotionType:      item.PromotionType,
			Description:        item.Description,
			ApplicableProducts: item.ApplicableProducts,
			StartAt:            item.StartAt,
			EndAt:              item.EndAt,
			DiscountRule:       item.DiscountRule,
			StoreBenefit:       item.StoreBenefit,
			AppointmentBenefit: item.AppointmentBenefit,
			ScriptSuggestion:   item.ScriptSuggestion,
			Priority:           item.Priority,
			Status:             item.Status,
			Remark:             item.Remark,
		})
	}
	return ret
}

func buildDigitalStoreTemplatePromotionPreviewItems(promotions []request.SavePromotionRequest) []response.DigitalStoreTemplatePreviewItem {
	ret := make([]response.DigitalStoreTemplatePreviewItem, 0, len(promotions))
	for _, item := range promotions {
		preview := response.DigitalStoreTemplatePreviewItem{
			Name:   item.Name,
			Action: "create",
			Reason: "按活动名称未找到现有记录，将新建活动并同步 FAQ。",
		}
		if existing := repositories.PromotionRepository.FindOne(sqls.DB(), sqls.NewCnd().Eq("name", item.Name).Where("status <> ?", enums.StatusDeleted)); existing != nil {
			preview.Action = "update"
			preview.ExistingID = existing.ID
			preview.Reason = "按活动名称匹配到现有记录，将更新活动字段并重建 FAQ。"
		}
		ret = append(ret, preview)
	}
	return ret
}

func buildDigitalStoreIndustryRiskRuleResponses(cfg digitalStoreProfileConfig) []response.DigitalStoreIndustryRiskRuleResponse {
	specs := digitalStoreIndustryRiskRuleSpecs(cfg)
	ret := make([]response.DigitalStoreIndustryRiskRuleResponse, 0, len(specs))
	for _, spec := range specs {
		ret = append(ret, response.DigitalStoreIndustryRiskRuleResponse{
			Key:             spec.key,
			Label:           spec.label,
			ForbiddenClaims: append([]string{}, spec.forbiddenClaims...),
			HandoffTriggers: append([]string{}, spec.handoffTriggers...),
		})
	}
	return ret
}

type digitalStoreIndustryRiskRuleSpec struct {
	key             string
	label           string
	forbiddenClaims []string
	handoffTriggers []string
}

func digitalStoreIndustryRiskRuleSpecs(cfg digitalStoreProfileConfig) []digitalStoreIndustryRiskRuleSpec {
	key := normalizeDigitalStoreIndustryKey(cfg)
	common := digitalStoreIndustryRiskRuleSpec{
		key:   "common",
		label: "通用高风险口径",
		forbiddenClaims: []string{
			"不得承诺最低价、最终价、额外折扣、现货库存、退款退货、赔付、安装时效或绝对结果。",
			"不得虚构资质、案例、证书、名额、排班、服务范围或资料中没有的优惠。",
		},
		handoffTriggers: []string{
			"客户追问最终成交价、实时库存、售后争议、退款赔付、投诉差评或明确要求人工。",
			"客户留下手机号、微信、预约时间、预算或表现出高意向到店信号。",
		},
	}
	var industry digitalStoreIndustryRiskRuleSpec
	switch key {
	case "medical":
		industry = digitalStoreIndustryRiskRuleSpec{
			key:   "medical",
			label: "医疗健康行业",
			forbiddenClaims: []string{
				"不得在线诊断、承诺治疗效果、无痛、一次解决、百分百成功、无需检查或固定治疗周期。",
				"不得虚构医生资质、排班、医保报销、药品/器械适应症或手术安排。",
			},
			handoffTriggers: []string{
				"客户出现急性疼痛、出血、肿胀、外伤、儿童急症或明显病情风险。",
				"客户询问最终费用、治疗方案、医生时间、医保报销、退款退费或要求医生沟通。",
			},
		}
	case "education":
		industry = digitalStoreIndustryRiskRuleSpec{
			key:   "education",
			label: "教育培训行业",
			forbiddenClaims: []string{
				"不得承诺保过、提分幅度、录取结果、证书包拿、就业保障或名师一定授课。",
				"不得虚构办学资质、师资履历、课程名额、考试政策、退费比例或补课承诺。",
			},
			handoffTriggers: []string{
				"客户询问最终学费、退费、合同条款、升学/考试结果、课程排期或老师资质证明。",
				"客户留下学生年级、考试目标、手机号、试听时间或明确要课程顾问联系。",
			},
		}
	case "finance":
		industry = digitalStoreIndustryRiskRuleSpec{
			key:   "finance",
			label: "金融服务行业",
			forbiddenClaims: []string{
				"不得承诺收益、保本、稳赚、贷款必批、利率最低、额度确定或投资回报。",
				"不得诱导客户提供完整银行卡密码、验证码、身份证影像等高敏信息。",
			},
			handoffTriggers: []string{
				"客户咨询具体利率、额度、收益、合同条款、风险评级、投诉或资金损失。",
				"客户表达办理意向、留下联系方式或需要持牌顾问/人工合规确认。",
			},
		}
	case "home_decoration":
		industry = digitalStoreIndustryRiskRuleSpec{
			key:   "home_decoration",
			label: "家装装修行业",
			forbiddenClaims: []string{
				"不得承诺一口价、绝不增项、固定工期、材料绝对环保、施工零风险或赔付金额。",
				"不得虚构设计师资质、施工案例、材料品牌授权、排期、优惠名额或验收结论。",
			},
			handoffTriggers: []string{
				"客户询问最终报价、工期、合同、材料品牌、增项争议、退款赔付或施工投诉。",
				"客户提供户型面积、装修预算、量房时间、手机号或明确要设计师联系。",
			},
		}
	case "bedding":
		industry = digitalStoreIndustryRiskRuleSpec{
			key:   "bedding",
			label: "家居寝具行业",
			forbiddenClaims: []string{
				"不得承诺治好腰疼、百分百改善睡眠、最低成交价、今天一定有现货或无条件退换。",
				"不得虚构库存、安装时效、售后赔付、检测证书或活动叠加权益。",
			},
			handoffTriggers: []string{
				"客户询问最终成交价、实时库存、配送安装、退换货、售后异响或投诉。",
				"客户留下尺寸、预算、手机号、微信、到店时间或高意向试躺信号。",
			},
		}
	default:
		industry = digitalStoreIndustryRiskRuleSpec{
			key:   "general_consulting",
			label: "通用咨询行业",
			forbiddenClaims: []string{
				"不得承诺资料中没有的效果、价格、周期、名额、资质、案例、结果或售后政策。",
			},
			handoffTriggers: []string{
				"客户询问合同、最终价格、售后争议、资质证明或需要人工确认的个性化问题。",
			},
		}
	}
	return []digitalStoreIndustryRiskRuleSpec{common, industry}
}

func normalizeDigitalStoreIndustryKey(cfg digitalStoreProfileConfig) string {
	text := strings.ToLower(strings.TrimSpace(cfg.TemplateCode + " " + cfg.Industry + " " + cfg.BrandName))
	switch {
	case strings.Contains(text, "oral") || strings.Contains(text, "clinic") || strings.Contains(text, "医疗") || strings.Contains(text, "口腔") || strings.Contains(text, "医美") || strings.Contains(text, "健康"):
		return "medical"
	case strings.Contains(text, "education") || strings.Contains(text, "培训") || strings.Contains(text, "教育") || strings.Contains(text, "课程") || strings.Contains(text, "升学"):
		return "education"
	case strings.Contains(text, "finance") || strings.Contains(text, "金融") || strings.Contains(text, "贷款") || strings.Contains(text, "保险") || strings.Contains(text, "理财"):
		return "finance"
	case strings.Contains(text, "home_decoration") || strings.Contains(text, "装修") || strings.Contains(text, "家装") || strings.Contains(text, "装饰") || strings.Contains(text, "设计施工"):
		return "home_decoration"
	case strings.Contains(text, "muse") || strings.Contains(text, "bedding") || strings.Contains(text, "寝具") || strings.Contains(text, "床垫") || strings.Contains(text, "家居"):
		return "bedding"
	default:
		return "general"
	}
}

func buildDigitalStoreTemplatePreviewWarnings(current digitalStoreProfileConfig, target response.DigitalStoreTemplateResponse) []response.DigitalStoreTemplatePreviewWarning {
	warnings := make([]response.DigitalStoreTemplatePreviewWarning, 0)
	if current.Initialized {
		warnings = append(warnings, response.DigitalStoreTemplatePreviewWarning{
			Key:     "profile_update",
			Message: "当前店长资料已初始化，应用模板会更新品牌、人设、预约规则、转人工规则和禁用承诺。",
		})
	}
	if strings.TrimSpace(current.TemplateCode) != "" {
		switch {
		case current.TemplateCode == target.Code && current.TemplateVersion == target.Version:
			warnings = append(warnings, response.DigitalStoreTemplatePreviewWarning{
				Key:     "same_template_version",
				Message: "当前店铺已应用同版本模板 " + current.TemplateCode + " v" + current.TemplateVersion + "，再次应用会刷新模板字段。",
			})
		case current.TemplateCode == target.Code:
			warnings = append(warnings, response.DigitalStoreTemplatePreviewWarning{
				Key:     "template_version_change",
				Message: "当前店铺来自模板 " + current.TemplateCode + " v" + valueOrDefault(current.TemplateVersion, "-") + "，本次将应用 v" + valueOrDefault(target.Version, "-") + "。",
			})
		default:
			warnings = append(warnings, response.DigitalStoreTemplatePreviewWarning{
				Key:     "template_code_change",
				Message: "当前店铺来自模板 " + current.TemplateCode + "，本次将切换为 " + target.Code + "，请确认不会覆盖商家定制口径。",
			})
		}
	}
	if current.KnowledgeBaseID > 0 {
		warnings = append(warnings, response.DigitalStoreTemplatePreviewWarning{
			Key:     "knowledge_preserved",
			Message: "现有知识库 ID 会保留，模板只会同步或更新相关 FAQ。",
		})
	}
	if strings.TrimSpace(current.EnterpriseWebhookURL) != "" {
		warnings = append(warnings, response.DigitalStoreTemplatePreviewWarning{
			Key:     "webhook_preserved",
			Message: "店长资料中的企业 Webhook 地址会保留，不会被模板覆盖。",
		})
	}
	return warnings
}

func (s *digitalStoreProfileService) GetSetupStatus() response.DigitalStoreSetupStatusResponse {
	cfg := s.loadConfig()
	ret := response.DigitalStoreSetupStatusResponse{
		ProfileInitialized: cfg.Initialized,
		KnowledgeBaseID:    cfg.KnowledgeBaseID,
		KnowledgeFAQID:     cfg.KnowledgeFAQID,
	}
	_ = sqls.DB().Model(&models.Product{}).Where("status <> ?", enums.StatusDeleted).Count(&ret.ProductTotal).Error
	_ = sqls.DB().Model(&models.Promotion{}).Where("status <> ?", enums.StatusDeleted).Count(&ret.PromotionTotal).Error
	ret.ProductKnowledgeSyncedTotal, ret.ProductKnowledgeUnsyncedTotal, ret.ProductKnowledgeFailedTotal = countDigitalStoreKnowledgeCoverage(&models.Product{}, ret.ProductTotal)
	ret.PromotionKnowledgeSyncedTotal, ret.PromotionKnowledgeUnsyncedTotal, ret.PromotionKnowledgeFailedTotal = countDigitalStoreKnowledgeCoverage(&models.Promotion{}, ret.PromotionTotal)
	if aiConfig := s.findActiveLLMConfig(); aiConfig != nil {
		ret.LLMConfigID = aiConfig.ID
		ret.LLMConfigName = aiConfig.Name
	}
	if embeddingConfig := s.findActiveEmbeddingConfig(); embeddingConfig != nil {
		ret.EmbeddingConfigID = embeddingConfig.ID
		ret.EmbeddingConfigName = embeddingConfig.Name
	}
	if agent := s.findDigitalStoreAgent(cfg); agent != nil {
		ret.AgentID = agent.ID
		ret.AgentName = agent.Name
		ret.WorkflowPublished = agent.WorkflowVersionID > 0
		ret.HumanHandoff = buildDigitalStoreHumanHandoff(agent)
	}
	if channel := s.findWebChannel(cfg); channel != nil {
		ret.WebChannelID = channel.ID
		ret.WebChannelCode = channel.ChannelID
		ret.WebChannelName = channel.Name
		ret.WebEntry = buildDigitalStoreWebEntry(channel, "")
	}
	ret.ModelHealthChecks = s.BuildModelHealthChecks(ret)
	ret.Ready = ret.ProfileInitialized &&
		ret.KnowledgeBaseID > 0 &&
		ret.KnowledgeFAQID > 0 &&
		ret.ProductTotal > 0 &&
		ret.PromotionTotal > 0 &&
		ret.ProductKnowledgeUnsyncedTotal == 0 &&
		ret.ProductKnowledgeFailedTotal == 0 &&
		ret.PromotionKnowledgeUnsyncedTotal == 0 &&
		ret.PromotionKnowledgeFailedTotal == 0 &&
		ret.LLMConfigID > 0 &&
		ret.EmbeddingConfigID > 0 &&
		ret.AgentID > 0 &&
		ret.WorkflowPublished &&
		ret.HumanHandoff.Ready &&
		ret.WebChannelID > 0
	ret.MissingSteps = buildDigitalStoreMissingSteps(ret)
	return ret
}

func (s *digitalStoreProfileService) GetKnowledgeAssistant() response.DigitalStoreKnowledgeAssistantResponse {
	cfg := s.loadConfig()
	kbID := cfg.KnowledgeBaseID
	if kbID == 0 {
		if resolved, err := resolveDigitalStoreKnowledgeBaseID(0); err == nil {
			kbID = resolved
		}
	}
	faqs := []models.KnowledgeFAQ{}
	if kbID > 0 {
		faqs = repositories.KnowledgeFAQRepository.FindAllByKnowledgeBaseID(sqls.DB(), kbID)
	}
	items := buildDigitalStoreKnowledgeAssistantItems(cfg, kbID, faqs)
	ret := response.DigitalStoreKnowledgeAssistantResponse{
		GeneratedAt:     utils.FormatTime(time.Now()),
		Industry:        valueOrDefault(cfg.Industry, "通用咨询"),
		KnowledgeBaseID: kbID,
		Items:           items,
	}
	for _, item := range items {
		if item.Covered {
			ret.CoveredTotal++
		} else {
			ret.MissingTotal++
		}
	}
	return ret
}

func (s *digitalStoreProfileService) GetTemplateEffect() response.DigitalStoreTemplateEffectResponse {
	cfg := s.loadConfig()
	kbID := cfg.KnowledgeBaseID
	if kbID == 0 {
		if resolved, err := resolveDigitalStoreKnowledgeBaseID(0); err == nil {
			kbID = resolved
		}
	}
	const days = 30
	since := time.Now().AddDate(0, 0, -days)
	ret := response.DigitalStoreTemplateEffectResponse{
		GeneratedAt:       utils.FormatTime(time.Now()),
		TemplateCode:      cfg.TemplateCode,
		TemplateVersion:   cfg.TemplateVersion,
		TemplateAppliedAt: cfg.TemplateAppliedAt,
		Industry:          valueOrDefault(cfg.Industry, "通用咨询"),
		KnowledgeBaseID:   kbID,
		Days:              days,
	}
	if kbID == 0 {
		ret.Suggestions = []string{"当前店铺还没有绑定知识库，先完成店长配置和知识库同步后再观察模板效果。"}
		ret.ImprovementMarkdown = buildDigitalStoreTemplateImprovementMarkdown(ret)
		return ret
	}
	db := sqls.DB()
	base := db.Model(&models.KnowledgeRetrieveLog{}).
		Where("knowledge_base_id = ?", kbID).
		Where("created_at >= ?", since)
	_ = base.Count(&ret.RetrieveTotal).Error
	_ = db.Model(&models.KnowledgeRetrieveLog{}).
		Where("knowledge_base_id = ?", kbID).
		Where("created_at >= ?", since).
		Where("answer_status IN ?", []int{
			int(enums.KnowledgeAnswerStatusNoAnswer),
			int(enums.KnowledgeAnswerStatusFallback),
			int(enums.KnowledgeAnswerStatusBlocked),
		}).
		Count(&ret.MissingQuestionTotal).Error
	_ = db.Model(&models.KnowledgeFeedback{}).
		Joins("JOIN knowledge_retrieve_logs ON knowledge_retrieve_logs.id = knowledge_feedbacks.retrieve_log_id").
		Where("knowledge_retrieve_logs.knowledge_base_id = ?", kbID).
		Where("knowledge_feedbacks.created_at >= ?", since).
		Where("knowledge_feedbacks.feedback_type <> ?", int(enums.KnowledgeFeedbackTypeLike)).
		Count(&ret.NegativeFeedbackTotal).Error
	ret.MissingQuestions = s.findTemplateEffectMissingQuestions(kbID, since, 6)
	ret.NegativeFeedbacks = s.findTemplateEffectNegativeFeedbacks(kbID, since, 6)
	ret.Suggestions = buildDigitalStoreTemplateEffectSuggestions(ret)
	ret.ImprovementMarkdown = buildDigitalStoreTemplateImprovementMarkdown(ret)
	return ret
}

func (s *digitalStoreProfileService) findTemplateEffectMissingQuestions(kbID int64, since time.Time, limit int) []response.DigitalStoreTemplateEffectItem {
	type row struct {
		Question      string
		Count         int64
		LatestAt      string
		RetrieveLogID int64
		AnswerStatus  int
	}
	var rows []row
	err := sqls.DB().Model(&models.KnowledgeRetrieveLog{}).
		Select("TRIM(question) AS question, COUNT(*) AS count, MAX(created_at) AS latest_at, MAX(id) AS retrieve_log_id, MAX(answer_status) AS answer_status").
		Where("knowledge_base_id = ?", kbID).
		Where("created_at >= ?", since).
		Where("TRIM(question) <> ''").
		Where("answer_status IN ?", []int{
			int(enums.KnowledgeAnswerStatusNoAnswer),
			int(enums.KnowledgeAnswerStatusFallback),
			int(enums.KnowledgeAnswerStatusBlocked),
		}).
		Group("TRIM(question)").
		Order("count DESC, latest_at DESC").
		Limit(limit).
		Scan(&rows).Error
	if err != nil {
		return nil
	}
	ret := make([]response.DigitalStoreTemplateEffectItem, 0, len(rows))
	for _, item := range rows {
		status := enums.KnowledgeAnswerStatus(item.AnswerStatus)
		ret = append(ret, response.DigitalStoreTemplateEffectItem{
			Question:            strings.TrimSpace(item.Question),
			Count:               item.Count,
			LatestAt:            formatDigitalStoreTemplateEffectTime(item.LatestAt),
			AnswerStatusName:    enums.GetKnowledgeAnswerStatusLabel(status),
			ActionHref:          digitalStoreRetrieveLogActionHref(item.RetrieveLogID, kbID),
			ActionLabel:         "查看日志",
			CreateFAQActionHref: digitalStoreRetrieveLogActionHref(item.RetrieveLogID, kbID),
		})
	}
	return ret
}

func (s *digitalStoreProfileService) findTemplateEffectNegativeFeedbacks(kbID int64, since time.Time, limit int) []response.DigitalStoreTemplateEffectItem {
	type row struct {
		Question       string
		Count          int64
		LatestAt       string
		RetrieveLogID  int64
		FeedbackType   int
		FeedbackReason string
		AnswerStatus   int
	}
	var rows []row
	err := sqls.DB().Model(&models.KnowledgeFeedback{}).
		Select("TRIM(knowledge_retrieve_logs.question) AS question, COUNT(*) AS count, MAX(knowledge_feedbacks.created_at) AS latest_at, MAX(knowledge_feedbacks.retrieve_log_id) AS retrieve_log_id, MAX(knowledge_feedbacks.feedback_type) AS feedback_type, MAX(knowledge_feedbacks.feedback_reason) AS feedback_reason, MAX(knowledge_retrieve_logs.answer_status) AS answer_status").
		Joins("JOIN knowledge_retrieve_logs ON knowledge_retrieve_logs.id = knowledge_feedbacks.retrieve_log_id").
		Where("knowledge_retrieve_logs.knowledge_base_id = ?", kbID).
		Where("knowledge_feedbacks.created_at >= ?", since).
		Where("knowledge_feedbacks.feedback_type <> ?", int(enums.KnowledgeFeedbackTypeLike)).
		Where("TRIM(knowledge_retrieve_logs.question) <> ''").
		Group("TRIM(knowledge_retrieve_logs.question)").
		Order("count DESC, latest_at DESC").
		Limit(limit).
		Scan(&rows).Error
	if err != nil {
		return nil
	}
	ret := make([]response.DigitalStoreTemplateEffectItem, 0, len(rows))
	for _, item := range rows {
		feedbackType := enums.KnowledgeFeedbackType(item.FeedbackType)
		status := enums.KnowledgeAnswerStatus(item.AnswerStatus)
		ret = append(ret, response.DigitalStoreTemplateEffectItem{
			Question:            strings.TrimSpace(item.Question),
			Count:               item.Count,
			LatestAt:            formatDigitalStoreTemplateEffectTime(item.LatestAt),
			FeedbackReason:      strings.TrimSpace(item.FeedbackReason),
			FeedbackTypeName:    enums.GetKnowledgeFeedbackTypeLabel(feedbackType),
			AnswerStatusName:    enums.GetKnowledgeAnswerStatusLabel(status),
			ActionHref:          digitalStoreRetrieveLogActionHref(item.RetrieveLogID, kbID),
			ActionLabel:         "查看反馈",
			CreateFAQActionHref: digitalStoreRetrieveLogActionHref(item.RetrieveLogID, kbID),
		})
	}
	return ret
}

func buildDigitalStoreTemplateEffectSuggestions(report response.DigitalStoreTemplateEffectResponse) []string {
	suggestions := make([]string, 0, 4)
	if strings.TrimSpace(report.TemplateCode) == "" {
		suggestions = append(suggestions, "当前店铺未记录行业模板来源，建议先应用或导入模板，后续才能把问题沉淀回模板版本。")
	}
	if report.MissingQuestionTotal == 0 && report.NegativeFeedbackTotal == 0 {
		suggestions = append(suggestions, fmt.Sprintf("近 %d 天暂无明显模板缺口，可继续观察真实客户咨询。", report.Days))
		return suggestions
	}
	if report.MissingQuestionTotal > 0 {
		suggestions = append(suggestions, fmt.Sprintf("近 %d 天有 %d 次无答案、兜底或风控问题，优先把高频问题补成 FAQ 并加入行业模板。", report.Days, report.MissingQuestionTotal))
	}
	if report.NegativeFeedbackTotal > 0 {
		suggestions = append(suggestions, fmt.Sprintf("近 %d 天有 %d 次负反馈，建议复核答案口径和引用来源，确认后更新模板 FAQ 或风险规则。", report.Days, report.NegativeFeedbackTotal))
	}
	if len(report.MissingQuestions) > 0 || len(report.NegativeFeedbacks) > 0 {
		suggestions = append(suggestions, "处理完成后导出行业模板 JSON，把新增 FAQ、产品字段或风险口径沉淀到下一家商家交付。")
	}
	return suggestions
}

func buildDigitalStoreTemplateImprovementMarkdown(report response.DigitalStoreTemplateEffectResponse) string {
	var builder strings.Builder
	templateCode := strings.TrimSpace(report.TemplateCode)
	if templateCode == "" {
		templateCode = "未记录模板"
	}
	builder.WriteString(fmt.Sprintf("# 行业模板改进包：%s\n\n", templateCode))
	builder.WriteString("## 模板信息\n")
	builder.WriteString(fmt.Sprintf("- 行业：%s\n", valueOrDefault(report.Industry, "通用咨询")))
	builder.WriteString(fmt.Sprintf("- 模板版本：%s\n", valueOrDefault(report.TemplateVersion, "未记录")))
	if strings.TrimSpace(report.TemplateAppliedAt) != "" {
		builder.WriteString(fmt.Sprintf("- 应用时间：%s\n", report.TemplateAppliedAt))
	}
	builder.WriteString(fmt.Sprintf("- 统计周期：近 %d 天\n", report.Days))
	builder.WriteString(fmt.Sprintf("- 生成时间：%s\n\n", report.GeneratedAt))

	builder.WriteString("## 效果概览\n")
	builder.WriteString(fmt.Sprintf("- 知识检索：%d\n", report.RetrieveTotal))
	builder.WriteString(fmt.Sprintf("- 知识缺口：%d\n", report.MissingQuestionTotal))
	builder.WriteString(fmt.Sprintf("- 负反馈：%d\n\n", report.NegativeFeedbackTotal))

	builder.WriteString("## 待补 FAQ 清单\n")
	if len(report.MissingQuestions) == 0 {
		builder.WriteString("- 暂无高频无答案、兜底或风控问题\n")
	} else {
		for _, item := range report.MissingQuestions {
			builder.WriteString(fmt.Sprintf("- %s（%d 次，%s，最近：%s）\n",
				item.Question,
				item.Count,
				valueOrDefault(item.AnswerStatusName, "待处理"),
				valueOrDefault(item.LatestAt, "-"),
			))
		}
	}
	builder.WriteString("\n## 待修正回答/风险口径\n")
	if len(report.NegativeFeedbacks) == 0 {
		builder.WriteString("- 暂无高频负反馈\n")
	} else {
		for _, item := range report.NegativeFeedbacks {
			reason := strings.TrimSpace(item.FeedbackReason)
			if reason == "" {
				reason = valueOrDefault(item.FeedbackTypeName, "未填写原因")
			}
			builder.WriteString(fmt.Sprintf("- %s（%d 次，原因：%s，最近：%s）\n",
				item.Question,
				item.Count,
				reason,
				valueOrDefault(item.LatestAt, "-"),
			))
		}
	}
	builder.WriteString("\n## 模板迭代建议\n")
	if len(report.Suggestions) == 0 {
		builder.WriteString("- 暂无模板迭代建议\n")
	} else {
		for _, suggestion := range report.Suggestions {
			builder.WriteString(fmt.Sprintf("- %s\n", suggestion))
		}
	}
	builder.WriteString("\n## 沉淀动作\n")
	builder.WriteString("- 将待补 FAQ 确认成标准答案，启用并重建索引。\n")
	builder.WriteString("- 将高频负反馈对应的禁用承诺、风险边界或引用来源写回行业模板。\n")
	builder.WriteString("- 处理完成后导出行业模板 JSON，作为下一家同类商家的交付底稿。\n")
	return strings.TrimSpace(builder.String())
}

func digitalStoreRetrieveLogActionHref(retrieveLogID int64, knowledgeBaseID int64) string {
	if retrieveLogID <= 0 {
		return "/dashboard/knowledge?tab=retrieveLogs"
	}
	params := fmt.Sprintf("tab=retrieveLogs&retrieveLogId=%d", retrieveLogID)
	if knowledgeBaseID > 0 {
		params += fmt.Sprintf("&knowledgeBaseId=%d", knowledgeBaseID)
	}
	return "/dashboard/knowledge?" + params
}

func formatDigitalStoreTemplateEffectTime(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05.999999999-07:00",
		"2006-01-02 15:04:05.999999999",
		"2006-01-02 15:04:05",
	}
	for _, layout := range layouts {
		if parsed, err := time.Parse(layout, value); err == nil {
			return utils.FormatTime(parsed)
		}
	}
	return value
}

func (s *digitalStoreProfileService) GetMaintenanceStatus() response.DigitalStoreMaintenanceStatusResponse {
	const backupRoot = "backups"
	latest, warnings := findLatestDigitalStoreBackupSnapshot(backupRoot)
	restoreBackupDir := backupRoot + "/<备份目录>"
	if latest != nil {
		restoreBackupDir = latest.Path
	}
	backupCommand := "scripts/backup-single-merchant.sh --output backups --compose docker-compose.yml"
	restoreDryRunCommand := "scripts/restore-single-merchant.sh --backup-dir " + restoreBackupDir + " --compose docker-compose.yml --dry-run"
	upgradeCommands := []string{
		backupCommand,
		"git pull",
		"docker compose --env-file .env.production up -d --build",
		"scripts/check-single-merchant-deploy.sh docker/agent-desk.production.yaml docker-compose.yml",
	}
	ret := response.DigitalStoreMaintenanceStatusResponse{
		CheckedAt:            utils.FormatTime(time.Now()),
		Status:               "ok",
		BackupRoot:           backupRoot,
		BackupCommand:        backupCommand,
		RestoreDryRunCommand: restoreDryRunCommand,
		UpgradeCommands:      upgradeCommands,
		UpgradeRunbook:       buildDigitalStoreUpgradeRunbook(latest, backupCommand, restoreDryRunCommand, upgradeCommands),
		LatestBackup:         latest,
		Warnings:             warnings,
	}
	if len(warnings) > 0 {
		ret.Status = "warning"
	}
	return ret
}

func buildDigitalStoreUpgradeRunbook(latest *response.DigitalStoreBackupSnapshotResponse, backupCommand string, restoreDryRunCommand string, upgradeCommands []string) string {
	latestBackupText := "未发现本地备份，升级前必须先执行备份。"
	if latest != nil {
		latestBackupText = fmt.Sprintf("%s（%s）", valueOrDefault(latest.Path, "-"), valueOrDefault(latest.CreatedAt, latest.Timestamp))
	}
	lines := []string{
		"# 单商家升级 Runbook",
		"",
		"## 0. 当前备份状态",
		"",
		"- 最近备份：" + latestBackupText,
		"- 恢复演练命令：" + restoreDryRunCommand,
		"",
		"## 1. 升级前必须备份",
		"",
		"```bash",
		backupCommand,
		"```",
		"",
		"## 2. 拉取代码并重建服务",
		"",
		"```bash",
	}
	lines = append(lines, upgradeCommands...)
	lines = append(lines,
		"```",
		"",
		"## 3. 升级后后台复验",
		"",
		"- 打开 `/dashboard/store-setup`，确认交付报告没有阻断项。",
		"- 检查“模型与检索健康”：聊天模型、Embedding 模型、向量库、产品知识索引、活动知识索引均应通过。",
		"- 如果产品或活动知识索引未同步，点击“同步店长知识”或进入产品/活动页重建 FAQ。",
		"- 点击“发送关键通知测试”，确认高意向、预约、转人工、未分配和售后风险 5 类通知均成功。",
		"",
		"## 4. 客户入口复验",
		"",
		"```bash",
		"MUSE_ACCEPTANCE_TIMEOUT_MS=70000 scripts/run-muse-chat-acceptance.mjs",
		"```",
		"",
		"## 5. 异常回滚",
		"",
		"- 如果升级后聊天入口、知识检索或后台登录异常，先保留日志，再按最近备份执行 dry-run 恢复演练。",
		"- dry-run 无异常后，停外部流量和应用服务，再使用 `scripts/restore-single-merchant.sh --confirm` 执行正式恢复。",
	)
	return strings.Join(lines, "\n")
}

func (s *digitalStoreProfileService) GetDeliveryReport(publicBaseURL string) response.DigitalStoreDeliveryReportResponse {
	cfg := s.loadConfig()
	status := s.GetSetupStatus()
	baseURL := strings.TrimRight(strings.TrimSpace(publicBaseURL), "/")
	dashboardURL := ""
	if baseURL != "" {
		dashboardURL = baseURL + "/dashboard"
	}
	webEntry := status.WebEntry
	if channel := s.findWebChannel(cfg); channel != nil {
		webEntry = buildDigitalStoreWebEntry(channel, baseURL)
	}
	items := []response.DigitalStoreDeliveryReportItem{
		buildDeliveryReportItem("品牌与门店", cfg.Initialized, valueOrDefault(cfg.BrandName, "-")+" / "+valueOrDefault(cfg.StoreName, "-")),
		buildDeliveryReportItem("产品库", status.ProductTotal > 0, fmt.Sprintf("%d 个产品", status.ProductTotal)),
		buildDeliveryReportItem("活动库", status.PromotionTotal > 0, fmt.Sprintf("%d 个活动", status.PromotionTotal)),
		buildDeliveryReportItem("产品知识索引", status.ProductTotal > 0 && status.ProductKnowledgeUnsyncedTotal == 0 && status.ProductKnowledgeFailedTotal == 0, formatKnowledgeCoverage(status.ProductKnowledgeSyncedTotal, status.ProductTotal, status.ProductKnowledgeUnsyncedTotal, status.ProductKnowledgeFailedTotal)),
		buildDeliveryReportItem("活动知识索引", status.PromotionTotal > 0 && status.PromotionKnowledgeUnsyncedTotal == 0 && status.PromotionKnowledgeFailedTotal == 0, formatKnowledgeCoverage(status.PromotionKnowledgeSyncedTotal, status.PromotionTotal, status.PromotionKnowledgeUnsyncedTotal, status.PromotionKnowledgeFailedTotal)),
		buildDeliveryReportItem("聊天模型", status.LLMConfigID > 0, valueOrDefault(status.LLMConfigName, "-")),
		buildDeliveryReportItem("Embedding 模型", status.EmbeddingConfigID > 0, valueOrDefault(status.EmbeddingConfigName, "-")),
		buildDeliveryReportItem("知识库", status.KnowledgeBaseID > 0 && status.KnowledgeFAQID > 0, fmt.Sprintf("知识库 #%d / FAQ #%d", status.KnowledgeBaseID, status.KnowledgeFAQID)),
		buildDeliveryReportItem("数字店长 Agent", status.AgentID > 0 && status.WorkflowPublished, valueOrDefault(status.AgentName, "-")),
		buildDeliveryReportItem("人工接待配置", status.HumanHandoff.Ready, status.HumanHandoff.Message),
		buildDeliveryReportItem("Web 聊天渠道", status.WebChannelID > 0, valueOrDefault(status.WebChannelCode, "-")),
		buildDeliveryReportItem("客户入口品牌化", webEntry.ChannelCode != "" && webEntry.Title != "" && webEntry.ThemeColor != "", formatDigitalStoreWebEntry(webEntry)),
	}
	notificationStatus := s.GetNotificationStatus()
	items = append(items, buildDeliveryReportItem("外部通知", notificationStatus.Enabled, notificationStatus.Message))
	securityChecks := s.GetSecurityChecks(notificationStatus)
	items = append(items, buildDeliveryReportItem("上线安全自检", !hasBlockingSecurityCheck(securityChecks), formatSecurityCheckSummary(securityChecks)))
	report := response.DigitalStoreDeliveryReportResponse{
		GeneratedAt:        utils.FormatTime(time.Now()),
		BrandName:          cfg.BrandName,
		StoreName:          cfg.StoreName,
		Ready:              status.Ready,
		DashboardURL:       dashboardURL,
		ChatURL:            webEntry.ChatURL,
		EmbedSnippet:       webEntry.EmbedSnippet,
		WebEntry:           webEntry,
		HumanHandoff:       status.HumanHandoff,
		AcceptanceCommand:  defaultDigitalStoreAcceptanceCommand(cfg),
		AcceptanceItems:    buildDigitalStoreAcceptanceItems(cfg),
		NotificationStatus: notificationStatus,
		SecurityChecks:     securityChecks,
		ModelHealthChecks:  status.ModelHealthChecks,
		Items:              items,
		MissingSteps:       status.MissingSteps,
		LatestRecord:       s.GetLatestDeliveryRecord(),
	}
	report.AcceptanceRunbook = buildDigitalStoreAcceptanceRunbook(report)
	report.Markdown = buildDigitalStoreDeliveryReportMarkdown(report)
	return report
}

func countDigitalStoreKnowledgeCoverage(model any, total int64) (synced int64, unsynced int64, failed int64) {
	db := sqls.DB()
	base := func() *gorm.DB {
		return db.Model(model).Where("status <> ?", enums.StatusDeleted)
	}
	_ = base().Where("knowledge_faq_id > 0").Count(&synced).Error
	_ = base().Where("knowledge_faq_id = 0").Count(&unsynced).Error
	if total > 0 && synced+unsynced < total {
		unsynced += total - synced - unsynced
	}
	failedSubQuery := db.Model(&models.KnowledgeFAQ{}).
		Select("id").
		Where("index_status = ?", enums.KnowledgeDocumentIndexStatusFailed).
		Where("status <> ?", enums.StatusDeleted)
	_ = base().Where("knowledge_faq_id IN (?)", failedSubQuery).Count(&failed).Error
	return synced, unsynced, failed
}

func findLatestDigitalStoreBackupSnapshot(root string) (*response.DigitalStoreBackupSnapshotResponse, []response.DigitalStoreMaintenanceWarningResponse) {
	root = filepath.Clean(strings.TrimSpace(root))
	if root == "." || root == "" {
		root = "backups"
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, []response.DigitalStoreMaintenanceWarningResponse{
				buildDigitalStoreMaintenanceWarning("backup_missing", "暂无备份", "未发现 backups 目录，请先执行内置备份脚本并配置定时备份。"),
			}
		}
		return nil, []response.DigitalStoreMaintenanceWarningResponse{
			buildDigitalStoreMaintenanceWarning("backup_scan_failed", "备份检查失败", "无法读取 "+root+" 目录："+err.Error()),
		}
	}
	snapshots := make([]response.DigitalStoreBackupSnapshotResponse, 0)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		path := filepath.Join(root, entry.Name())
		snapshot := buildDigitalStoreBackupSnapshot(path, entry.Name())
		snapshots = append(snapshots, snapshot)
	}
	if len(snapshots) == 0 {
		return nil, []response.DigitalStoreMaintenanceWarningResponse{
			buildDigitalStoreMaintenanceWarning("backup_empty", "暂无备份", root+" 目录下没有可用备份快照。"),
		}
	}
	sort.Slice(snapshots, func(i, j int) bool {
		return digitalStoreBackupSortKey(snapshots[i]) > digitalStoreBackupSortKey(snapshots[j])
	})
	latest := snapshots[0]
	warnings := make([]response.DigitalStoreMaintenanceWarningResponse, 0)
	if !latest.HasManifest {
		warnings = append(warnings, buildDigitalStoreMaintenanceWarning("manifest_missing", "备份清单缺失", "最近备份缺少 BACKUP-MANIFEST.txt，恢复前请人工确认来源。"))
	}
	if !latest.HasMySQLDump && !latest.HasDataArchive {
		warnings = append(warnings, buildDigitalStoreMaintenanceWarning("data_snapshot_missing", "数据快照缺失", "最近备份未包含 mysql.sql 或 data.tar.gz，可能无法完整恢复业务数据。"))
	}
	if !latest.HasDockerConfigArchive || !latest.HasConfigSnapshot {
		warnings = append(warnings, buildDigitalStoreMaintenanceWarning("config_snapshot_partial", "配置快照不完整", "最近备份未同时包含 docker 配置和 config/config.yaml，迁移机器时需另行保存部署配置。"))
	}
	return &latest, warnings
}

func buildDigitalStoreBackupSnapshot(path string, fallbackTimestamp string) response.DigitalStoreBackupSnapshotResponse {
	manifest := readDigitalStoreBackupManifest(filepath.Join(path, "BACKUP-MANIFEST.txt"))
	timestamp := valueOrDefault(manifest["timestamp"], fallbackTimestamp)
	snapshot := response.DigitalStoreBackupSnapshotResponse{
		Path:                   filepath.ToSlash(path),
		Timestamp:              timestamp,
		CreatedAt:              manifest["created_at"],
		ProjectDir:             manifest["project_dir"],
		ComposeFile:            manifest["compose_file"],
		HasManifest:            manifest != nil,
		HasMySQLDump:           fileExists(filepath.Join(path, "mysql.sql")),
		HasDataArchive:         fileExists(filepath.Join(path, "data.tar.gz")),
		HasDockerConfigArchive: fileExists(filepath.Join(path, "docker-config.tar.gz")),
		HasConfigSnapshot:      fileExists(filepath.Join(path, "config", "config.yaml")),
	}
	snapshot.SizeBytes = directorySizeBytes(path)
	return snapshot
}

func readDigitalStoreBackupManifest(path string) map[string]string {
	file, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer file.Close()
	ret := map[string]string{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		key, value, ok := strings.Cut(scanner.Text(), "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		ret[key] = strings.TrimSpace(value)
	}
	return ret
}

func digitalStoreBackupSortKey(snapshot response.DigitalStoreBackupSnapshotResponse) string {
	if snapshot.CreatedAt != "" {
		return snapshot.CreatedAt
	}
	return snapshot.Timestamp
}

func directorySizeBytes(path string) int64 {
	var total int64
	_ = filepath.WalkDir(path, func(_ string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err == nil {
			total += info.Size()
		}
		return nil
	})
	return total
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func buildDigitalStoreMaintenanceWarning(key string, label string, message string) response.DigitalStoreMaintenanceWarningResponse {
	return response.DigitalStoreMaintenanceWarningResponse{
		Key:     key,
		Label:   label,
		Message: message,
	}
}

func (s *digitalStoreProfileService) BuildModelHealthChecks(status response.DigitalStoreSetupStatusResponse) []response.DigitalStoreHealthCheckResponse {
	checks := []response.DigitalStoreHealthCheckResponse{}
	if status.LLMConfigID > 0 {
		checks = append(checks, buildDigitalStoreHealthCheck("llm", "聊天模型", "ok", "已启用聊天模型："+valueOrDefault(status.LLMConfigName, fmt.Sprintf("#%d", status.LLMConfigID))))
	} else {
		checks = append(checks, buildDigitalStoreHealthCheck("llm", "聊天模型", "blocking", "未启用聊天模型，数字店长无法生成客户回复。"))
	}
	if status.EmbeddingConfigID > 0 {
		checks = append(checks, buildDigitalStoreHealthCheck("embedding", "Embedding 模型", "ok", "已启用向量模型："+valueOrDefault(status.EmbeddingConfigName, fmt.Sprintf("#%d", status.EmbeddingConfigID))))
	} else {
		checks = append(checks, buildDigitalStoreHealthCheck("embedding", "Embedding 模型", "blocking", "未启用 Embedding 模型，产品、活动和 FAQ 无法稳定检索。"))
	}

	vectorType := ""
	if cfg, ok := safeRuntimeConfig(); ok {
		vectorType = strings.ToLower(strings.TrimSpace(cfg.VectorDB.Type))
	}
	switch vectorType {
	case "qdrant":
		checks = append(checks, buildDigitalStoreHealthCheck("vector_db", "向量库", "ok", "向量库类型为 qdrant，适合正式部署。"))
	case "lancedb":
		checks = append(checks, buildDigitalStoreHealthCheck("vector_db", "向量库", "ok", "向量库类型为 lancedb，适合轻量单商家部署。"))
	case "":
		checks = append(checks, buildDigitalStoreHealthCheck("vector_db", "向量库", "blocking", "vectorDB.type 未配置，知识检索可能不可用。"))
	default:
		checks = append(checks, buildDigitalStoreHealthCheck("vector_db", "向量库", "blocking", "vectorDB.type 为 "+vectorType+"，请改为 qdrant 或 lancedb。"))
	}

	productOK := status.ProductTotal > 0 && status.ProductKnowledgeUnsyncedTotal == 0 && status.ProductKnowledgeFailedTotal == 0
	checks = append(checks, buildDigitalStoreKnowledgeHealthCheck(
		"product_index",
		"产品知识索引",
		productOK,
		status.ProductTotal,
		status.ProductKnowledgeSyncedTotal,
		status.ProductKnowledgeUnsyncedTotal,
		status.ProductKnowledgeFailedTotal,
	))
	promotionOK := status.PromotionTotal > 0 && status.PromotionKnowledgeUnsyncedTotal == 0 && status.PromotionKnowledgeFailedTotal == 0
	checks = append(checks, buildDigitalStoreKnowledgeHealthCheck(
		"promotion_index",
		"活动知识索引",
		promotionOK,
		status.PromotionTotal,
		status.PromotionKnowledgeSyncedTotal,
		status.PromotionKnowledgeUnsyncedTotal,
		status.PromotionKnowledgeFailedTotal,
	))
	return checks
}

func buildDigitalStoreKnowledgeHealthCheck(key string, label string, ok bool, total int64, synced int64, unsynced int64, failed int64) response.DigitalStoreHealthCheckResponse {
	if total == 0 {
		return buildDigitalStoreHealthCheck(key, label, "blocking", label+"没有可用数据，请先导入。")
	}
	if ok {
		return buildDigitalStoreHealthCheck(key, label, "ok", fmt.Sprintf("已同步 %d/%d，索引状态正常。", synced, total))
	}
	return buildDigitalStoreHealthCheck(key, label, "blocking", formatKnowledgeCoverage(synced, total, unsynced, failed))
}

type digitalStoreKnowledgeAssistantSpec struct {
	key      string
	question string
	reason   string
	keywords []string
	required bool
}

func buildDigitalStoreKnowledgeAssistantItems(cfg digitalStoreProfileConfig, knowledgeBaseID int64, faqs []models.KnowledgeFAQ) []response.DigitalStoreKnowledgeAssistantItem {
	specs := digitalStoreKnowledgeAssistantSpecs(cfg)
	items := make([]response.DigitalStoreKnowledgeAssistantItem, 0, len(specs))
	for _, spec := range specs {
		item := response.DigitalStoreKnowledgeAssistantItem{
			Key:         spec.key,
			Question:    spec.question,
			Reason:      spec.reason,
			Required:    spec.required,
			Keywords:    append([]string{}, spec.keywords...),
			ActionLabel: "去补 FAQ",
		}
		if knowledgeBaseID > 0 {
			item.ActionHref = fmt.Sprintf("/dashboard/knowledge?knowledgeBaseId=%d", knowledgeBaseID)
		} else {
			item.ActionHref = "/dashboard/knowledge"
		}
		if matched := matchDigitalStoreKnowledgeAssistantFAQ(spec, faqs); matched != nil {
			item.Covered = true
			item.MatchedFAQID = matched.ID
			item.ActionHref = fmt.Sprintf("/dashboard/knowledge?knowledgeBaseId=%d&faqId=%d", matched.KnowledgeBaseID, matched.ID)
			item.ActionLabel = "查看 FAQ"
		}
		items = append(items, item)
	}
	return items
}

func matchDigitalStoreKnowledgeAssistantFAQ(spec digitalStoreKnowledgeAssistantSpec, faqs []models.KnowledgeFAQ) *models.KnowledgeFAQ {
	for i := range faqs {
		faq := &faqs[i]
		text := strings.ToLower(strings.Join([]string{faq.Question, faq.Answer, faq.SimilarQuestions}, " "))
		if strings.TrimSpace(text) == "" {
			continue
		}
		matched := 0
		for _, keyword := range spec.keywords {
			keyword = strings.ToLower(strings.TrimSpace(keyword))
			if keyword != "" && strings.Contains(text, keyword) {
				matched++
			}
		}
		if matched >= minInt(2, len(spec.keywords)) {
			return faq
		}
	}
	return nil
}

func digitalStoreKnowledgeAssistantSpecs(cfg digitalStoreProfileConfig) []digitalStoreKnowledgeAssistantSpec {
	common := []digitalStoreKnowledgeAssistantSpec{
		buildKnowledgeAssistantSpec("store_basic", "门店地址、营业时间和联系方式是什么？", "客户最常先问门店在哪里、几点营业、怎么联系。", []string{"地址", "营业时间", "电话", "微信"}, true),
		buildKnowledgeAssistantSpec("appointment", "如何预约到店、试用或咨询？", "高意向客户需要清楚预约流程和需要留下的信息。", []string{"预约", "到店", "手机号", "时间"}, true),
		buildKnowledgeAssistantSpec("price_boundary", "价格、优惠、库存和最终成交价如何确认？", "避免 AI 编造最低价、库存和叠加优惠。", []string{"价格", "优惠", "库存", "顾问"}, true),
		buildKnowledgeAssistantSpec("handoff", "哪些情况需要转人工或顾问跟进？", "把最终确认、投诉售后和高意向线索交给人工闭环。", []string{"人工", "顾问", "转人工", "跟进"}, true),
		buildKnowledgeAssistantSpec("after_sales", "售后、退款、退货、投诉或赔付怎么处理？", "售后政策缺失会导致 AI 乱承诺退款退货或赔付。", []string{"售后", "退款", "退货", "投诉", "赔付"}, true),
	}
	switch normalizeDigitalStoreIndustryKey(cfg) {
	case "medical":
		return append(common,
			buildKnowledgeAssistantSpec("medical_diagnosis_boundary", "线上口腔咨询和医生面诊的边界是什么？", "医疗行业必须说明线上咨询不能替代医生诊断。", []string{"线上咨询", "面诊", "医生", "诊断"}, true),
			buildKnowledgeAssistantSpec("medical_emergency", "牙痛、出血、肿胀等急症应如何处理？", "急症风险需要尽快引导到院或人工。", []string{"牙痛", "出血", "肿胀", "尽快到院"}, true),
			buildKnowledgeAssistantSpec("medical_fee", "治疗费用、周期、医保和退款如何确认？", "费用、周期和医保不可由 AI 直接承诺。", []string{"费用", "周期", "医保", "退款"}, true),
		)
	case "education":
		return append(common,
			buildKnowledgeAssistantSpec("education_course", "课程适合哪些学生、班型和课时如何安排？", "教育行业需要明确年级、目标、班型和课时。", []string{"课程", "年级", "班型", "课时"}, true),
			buildKnowledgeAssistantSpec("education_result_boundary", "提分、保过、录取和证书结果如何说明？", "教育行业不能承诺保过、提分幅度或录取结果。", []string{"提分", "保过", "录取", "证书"}, true),
			buildKnowledgeAssistantSpec("education_refund", "试听、报名、退费和合同规则是什么？", "报名转化前需要清楚试听和退费边界。", []string{"试听", "报名", "退费", "合同"}, true),
		)
	case "finance":
		return append(common,
			buildKnowledgeAssistantSpec("finance_risk", "收益、保本、利率、额度和风险如何说明？", "金融行业不能承诺收益、保本、贷款必批或固定额度。", []string{"收益", "保本", "利率", "额度", "风险"}, true),
			buildKnowledgeAssistantSpec("finance_sensitive_info", "客户哪些敏感信息不能在聊天中收集？", "金融行业必须避免收集银行卡密码、验证码等高敏信息。", []string{"银行卡", "密码", "验证码", "身份证"}, true),
			buildKnowledgeAssistantSpec("finance_handoff", "哪些金融咨询必须转持牌顾问或人工确认？", "合同、风险评级和具体方案应转人工。", []string{"持牌", "顾问", "合同", "风险评级"}, true),
		)
	case "home_decoration":
		return append(common,
			buildKnowledgeAssistantSpec("decoration_measure", "量房、户型、面积和预算如何收集？", "家装咨询要先收集面积、户型和预算。", []string{"量房", "户型", "面积", "预算"}, true),
			buildKnowledgeAssistantSpec("decoration_quote", "报价、工期、材料和增项如何确认？", "家装行业不能承诺一口价、绝不增项或固定工期。", []string{"报价", "工期", "材料", "增项"}, true),
			buildKnowledgeAssistantSpec("decoration_after_sales", "施工延期、质量问题、退款赔付和验收怎么处理？", "施工争议必须转人工并依据合同处理。", []string{"施工", "延期", "质量", "验收", "赔付"}, true),
		)
	default:
		return append(common,
			buildKnowledgeAssistantSpec("product_selection", "如何按预算、人群和场景推荐主推产品？", "通用导购需要把产品推荐口径写清楚。", []string{"预算", "人群", "场景", "推荐"}, true),
			buildKnowledgeAssistantSpec("compliance_boundary", "哪些效果、资质、案例和结果不能承诺？", "高咨询行业都需要明确禁用承诺。", []string{"效果", "资质", "案例", "承诺"}, true),
		)
	}
}

func buildKnowledgeAssistantSpec(key string, question string, reason string, keywords []string, required bool) digitalStoreKnowledgeAssistantSpec {
	return digitalStoreKnowledgeAssistantSpec{key: key, question: question, reason: reason, keywords: keywords, required: required}
}

func minInt(a int, b int) int {
	if a < b {
		return a
	}
	return b
}

func buildDigitalStoreHealthCheck(key string, label string, status string, message string) response.DigitalStoreHealthCheckResponse {
	item := response.DigitalStoreHealthCheckResponse{
		Key:         key,
		Label:       label,
		Status:      status,
		Message:     valueOrDefault(message, "-"),
		ActionHref:  digitalStoreHealthCheckActionHref(key),
		ActionLabel: digitalStoreHealthCheckActionLabel(key),
	}
	if status == "ok" {
		item.ActionHref = ""
		item.ActionLabel = ""
	}
	return item
}

func digitalStoreHealthCheckActionHref(key string) string {
	switch key {
	case "llm", "embedding":
		return "/dashboard/ai-configs"
	case "product_index":
		return "/dashboard/products"
	case "promotion_index":
		return "/dashboard/promotions"
	case "vector_db":
		return "/dashboard/store-setup"
	default:
		return ""
	}
}

func digitalStoreHealthCheckActionLabel(key string) string {
	switch key {
	case "llm", "embedding":
		return "去配置模型"
	case "product_index", "promotion_index":
		return "去重建索引"
	case "vector_db":
		return "查看部署配置"
	default:
		return "去处理"
	}
}

func buildDigitalStoreHumanHandoff(agent *models.AIAgent) response.DigitalStoreHumanHandoffResponse {
	if agent == nil {
		return response.DigitalStoreHumanHandoffResponse{
			Message: "未生成数字店长 Agent。",
		}
	}
	teamIDs := utils.SplitInt64s(agent.TeamIDs)
	ret := response.DigitalStoreHumanHandoffResponse{
		AgentTeamIDs: teamIDs,
	}
	if len(teamIDs) == 0 {
		ret.Message = "数字店长 Agent 尚未绑定人工顾问组。"
		return ret
	}
	ret.AgentProfileTotal = AgentProfileService.Count(sqls.NewCnd().
		In("team_id", teamIDs).
		Where("status <> ?", enums.StatusDeleted))
	ret.AutoAssignProfiles = AgentProfileService.Count(sqls.NewCnd().
		In("team_id", teamIDs).
		Eq("status", enums.StatusOk).
		Eq("auto_assign_enabled", true))
	candidates, report, err := ConversationDispatchService.pickDispatchCandidates(teamIDs, time.Now())
	ret.ActiveTeamIDs = report.ActiveScheduleTeams
	ret.EligibleProfiles = report.EligibleProfiles
	ret.CandidateProfiles = len(candidates)
	if err != nil {
		ret.Message = "人工接待配置检查失败：" + err.Error()
		return ret
	}
	ret.Ready = len(candidates) > 0
	if ret.Ready {
		ret.Message = fmt.Sprintf("已绑定 %d 个顾问组，当前 %d 个组在排班，%d 名顾问可自动接待。", len(teamIDs), len(ret.ActiveTeamIDs), len(candidates))
		return ret
	}
	switch report.Reason {
	case "no_active_schedule_team":
		ret.Message = fmt.Sprintf("已绑定 %d 个顾问组，但当前没有生效排班。", len(teamIDs))
	case "no_matched_profile":
		ret.Message = "当前排班顾问组中没有客服档案。"
	case "no_enabled_user", "no_profile_for_enabled_user":
		ret.Message = "顾问档案对应的后台账号未启用。"
	case "all_candidates_at_capacity":
		ret.Message = "当前顾问已达到最大并发接待数。"
	default:
		ret.Message = "暂无可自动接待的人工顾问。"
	}
	return ret
}

func buildDigitalStoreWebEntry(channel *models.Channel, publicBaseURL string) response.DigitalStoreWebEntryResponse {
	if channel == nil {
		return response.DigitalStoreWebEntryResponse{}
	}
	cfg, err := ChannelService.ParseWebChannelConfig(channel.ConfigJSON)
	if err != nil {
		cfg = &dto.WebChannelConfig{
			Title:      "AI数字店长",
			ThemeColor: "#2563eb",
			Position:   "right",
			Width:      "380px",
		}
	}
	baseURL := strings.TrimRight(strings.TrimSpace(publicBaseURL), "/")
	entry := response.DigitalStoreWebEntryResponse{
		ChannelID:   channel.ID,
		ChannelCode: strings.TrimSpace(channel.ChannelID),
		ChannelName: strings.TrimSpace(channel.Name),
		Title:       strings.TrimSpace(cfg.Title),
		Subtitle:    strings.TrimSpace(cfg.Subtitle),
		ThemeColor:  strings.TrimSpace(cfg.ThemeColor),
		Position:    strings.TrimSpace(cfg.Position),
		Width:       strings.TrimSpace(cfg.Width),
	}
	if baseURL != "" && entry.ChannelCode != "" {
		entry.ChatURL = baseURL + "/support/chat/?channelId=" + entry.ChannelCode
		entry.EmbedSnippet = buildDigitalStoreEmbedSnippet(entry, baseURL)
	}
	return entry
}

func buildDigitalStoreEmbedSnippet(entry response.DigitalStoreWebEntryResponse, baseURL string) string {
	return fmt.Sprintf(`<script>
  window.AgentDeskConfig = {
    channelId: %s,
    baseUrl: %s,
    title: %s,
    subtitle: %s,
    themeColor: %s,
    position: %s,
    width: %s
  };
</script>
<script async src="%s/sdk/agent-desk-sdk.min.js"></script>`,
		jsStringLiteral(entry.ChannelCode),
		jsStringLiteral(baseURL),
		jsStringLiteral(entry.Title),
		jsStringLiteral(entry.Subtitle),
		jsStringLiteral(entry.ThemeColor),
		jsStringLiteral(entry.Position),
		jsStringLiteral(entry.Width),
		baseURL,
	)
}

func jsStringLiteral(value string) string {
	raw, err := json.Marshal(strings.TrimSpace(value))
	if err != nil {
		return `""`
	}
	return string(raw)
}

func formatDigitalStoreWebEntry(entry response.DigitalStoreWebEntryResponse) string {
	parts := []string{}
	if entry.Title != "" {
		parts = append(parts, "标题："+entry.Title)
	}
	if entry.Subtitle != "" {
		parts = append(parts, "副标题："+entry.Subtitle)
	}
	if entry.ThemeColor != "" {
		parts = append(parts, "主题色："+entry.ThemeColor)
	}
	if entry.Position != "" || entry.Width != "" {
		parts = append(parts, "位置/宽度："+valueOrDefault(entry.Position, "-")+" / "+valueOrDefault(entry.Width, "-"))
	}
	if len(parts) == 0 {
		return "-"
	}
	return strings.Join(parts, "；")
}

func (s *digitalStoreProfileService) GetNotificationStatus() response.DigitalStoreNotificationStatusResponse {
	cfg := s.loadConfig()
	webhook, configReady := safeWebhookNotifyConfig()
	ret := response.DigitalStoreNotificationStatusResponse{
		ProfileWebhookURLSet: strings.TrimSpace(cfg.EnterpriseWebhookURL) != "",
		Format:               valueOrDefault(webhook.Format, "generic"),
		Configured:           configReady && strings.TrimSpace(webhook.URL) != "",
		HasSecret:            configReady && strings.TrimSpace(webhook.Secret) != "",
	}
	ret.Enabled = configReady && webhook.Enabled && strings.TrimSpace(webhook.URL) != ""
	switch {
	case ret.Enabled:
		ret.Status = "enabled"
		ret.Message = "全局 notify.webhook 已启用，可接收高意向线索、预约线索、会话分配和转人工提醒。"
	case !configReady:
		ret.Status = "unavailable"
		ret.Message = "后端配置尚未加载，无法读取 notify.webhook。"
	case ret.ProfileWebhookURLSet:
		ret.Status = "profile_only"
		ret.Message = "店长资料中填写了 Webhook，但实际通知需启用全局 notify.webhook。"
	case ret.Configured:
		ret.Status = "disabled"
		ret.Message = "notify.webhook 已配置 URL 但未启用。"
	default:
		ret.Status = "missing"
		ret.Message = "未配置外部通知 Webhook。"
	}
	return ret
}

func (s *digitalStoreProfileService) GetSecurityChecks(notificationStatus response.DigitalStoreNotificationStatusResponse) []response.DigitalStoreSecurityCheckResponse {
	cfg, configReady := safeRuntimeConfig()
	if !configReady {
		return []response.DigitalStoreSecurityCheckResponse{
			buildSecurityCheck("config", "后端配置", "blocking", "后端配置尚未加载，无法确认上线安全项。"),
			buildSecurityCheck("notification", "外部通知", "warning", notificationStatus.Message),
		}
	}

	checks := []response.DigitalStoreSecurityCheckResponse{}
	customerSecret := strings.TrimSpace(cfg.CustomerSession.Secret)
	switch {
	case isBlankOrPlaceholder(customerSecret):
		checks = append(checks, buildSecurityCheck("customer_session_secret", "客户聊天密钥", "blocking", "customerSession.secret 未配置，客户聊天会话签名不安全。"))
	case len(customerSecret) < 32:
		checks = append(checks, buildSecurityCheck("customer_session_secret", "客户聊天密钥", "warning", "customerSession.secret 已配置，但建议使用至少 32 位随机字符串。"))
	default:
		checks = append(checks, buildSecurityCheck("customer_session_secret", "客户聊天密钥", "ok", "customerSession.secret 已配置。"))
	}

	bootstrapPassword := strings.TrimSpace(os.Getenv(constants.EnvBootstrapAdminPassword))
	switch {
	case isBlankOrPlaceholder(bootstrapPassword):
		checks = append(checks, buildSecurityCheck("bootstrap_admin_password", "首次管理员密码", "blocking", "未设置 AGENT_DESK_BOOTSTRAP_ADMIN_PASSWORD；首次初始化可能使用默认 admin 密码。"))
	case bootstrapPassword == constants.BootstrapAdminPassword:
		checks = append(checks, buildSecurityCheck("bootstrap_admin_password", "首次管理员密码", "blocking", "AGENT_DESK_BOOTSTRAP_ADMIN_PASSWORD 仍是默认值，请改为商家独立强密码。"))
	case len(bootstrapPassword) < 12:
		checks = append(checks, buildSecurityCheck("bootstrap_admin_password", "首次管理员密码", "warning", "首次管理员密码已设置，但长度偏短，建议至少 12 位。"))
	default:
		checks = append(checks, buildSecurityCheck("bootstrap_admin_password", "首次管理员密码", "ok", "首次管理员密码环境变量已设置。"))
	}

	if cfg.Auth.MaxFailedAttempts <= 0 {
		checks = append(checks, buildSecurityCheck("auth_lockout", "登录失败锁定", "warning", "auth.maxFailedAttempts 未启用，建议开启后台登录失败临时锁定。"))
	} else {
		checks = append(checks, buildSecurityCheck("auth_lockout", "登录失败锁定", "ok", fmt.Sprintf("已启用后台登录失败锁定，阈值为 %d 次。", cfg.Auth.MaxFailedAttempts)))
	}

	origins := normalizedCORSOrigins(cfg.Server.CORS.AllowedOrigins)
	switch {
	case len(origins) == 0:
		checks = append(checks, buildSecurityCheck("cors_allowed_origins", "CORS 白名单", "warning", "未配置跨域白名单；同源可用，官网嵌入聊天前需加入商家域名。"))
	case hasWildcardOrigin(origins):
		checks = append(checks, buildSecurityCheck("cors_allowed_origins", "CORS 白名单", "blocking", "CORS 白名单不应使用 *，请改为后台域名和商家官网域名。"))
	case allLocalOrigins(origins):
		checks = append(checks, buildSecurityCheck("cors_allowed_origins", "CORS 白名单", "warning", "CORS 仍只包含 localhost/127.0.0.1，上线前请改为正式域名。"))
	default:
		checks = append(checks, buildSecurityCheck("cors_allowed_origins", "CORS 白名单", "ok", fmt.Sprintf("已配置 %d 个跨域白名单域名。", len(origins))))
	}

	dbType := strings.ToLower(strings.TrimSpace(cfg.DB.Type))
	switch dbType {
	case "mysql":
		checks = append(checks, buildSecurityCheck("database", "数据库", "ok", "数据库类型为 MySQL，适合正式单商家部署。"))
	case "sqlite":
		checks = append(checks, buildSecurityCheck("database", "数据库", "warning", "当前使用 SQLite，小型单店可用；正式高并发或多人后台建议改 MySQL。"))
	case "":
		checks = append(checks, buildSecurityCheck("database", "数据库", "warning", "数据库类型未配置，请确认运行环境使用独立商家数据库。"))
	default:
		checks = append(checks, buildSecurityCheck("database", "数据库", "warning", "数据库类型为 "+dbType+"，请确认生产环境已验证。"))
	}

	vectorType := strings.ToLower(strings.TrimSpace(cfg.VectorDB.Type))
	if vectorType == "qdrant" || vectorType == "lancedb" {
		checks = append(checks, buildSecurityCheck("vector_db", "向量库", "ok", "向量库类型为 "+vectorType+"。"))
	} else {
		checks = append(checks, buildSecurityCheck("vector_db", "向量库", "blocking", "vectorDB.type 未配置为 qdrant 或 lancedb，知识检索可能不可用。"))
	}

	switch {
	case notificationStatus.Enabled:
		checks = append(checks, buildSecurityCheck("notification", "外部通知", "ok", notificationStatus.Message))
	case notificationStatus.Configured || notificationStatus.ProfileWebhookURLSet:
		checks = append(checks, buildSecurityCheck("notification", "外部通知", "warning", notificationStatus.Message))
	default:
		checks = append(checks, buildSecurityCheck("notification", "外部通知", "warning", "未启用外部通知；高意向线索和转人工提醒只能依赖站内查看。"))
	}
	switch {
	case notificationStatus.Enabled && notificationStatus.HasSecret:
		checks = append(checks, buildSecurityCheck("webhook_secret", "Webhook 签名密钥", "ok", "外部通知已配置签名密钥。"))
	case notificationStatus.Enabled:
		checks = append(checks, buildSecurityCheck("webhook_secret", "Webhook 签名密钥", "warning", "外部通知已启用但未配置 notify.webhook.secret，商家自建接收端无法校验消息签名。"))
	case notificationStatus.Configured:
		checks = append(checks, buildSecurityCheck("webhook_secret", "Webhook 签名密钥", "warning", "notify.webhook 已配置 URL，启用前建议补充签名密钥。"))
	default:
		checks = append(checks, buildSecurityCheck("webhook_secret", "Webhook 签名密钥", "warning", "未启用外部通知；启用商家通知接口时建议配置签名密钥。"))
	}
	return checks
}

func (s *digitalStoreProfileService) TestWebhookNotify(operator *dto.AuthPrincipal) (response.DigitalStoreWebhookTestResponse, error) {
	if operator == nil {
		return response.DigitalStoreWebhookTestResponse{}, errorsx.UnauthorizedI18n("error.auth.expired")
	}
	status := s.GetNotificationStatus()
	ret := response.DigitalStoreWebhookTestResponse{
		DigitalStoreNotificationStatusResponse: status,
		TestedAt:                               utils.FormatTime(time.Now()),
	}
	if !status.Enabled {
		return ret, nil
	}
	cfg := s.loadConfig()
	body := strings.Join([]string{
		"这是一条 AI 数字店长外部通知测试。",
		"品牌：" + valueOrDefault(cfg.BrandName, "未配置"),
		"门店：" + valueOrDefault(cfg.StoreName, "未配置"),
		"操作人：" + valueOrDefault(operator.Username, "system"),
		"时间：" + ret.TestedAt,
	}, "\n")
	if err := WebhookNotifyService.SendText("digital_store_webhook_test", "AI数字店长外部通知测试", body, map[string]any{
		"brandName":  cfg.BrandName,
		"storeName":  cfg.StoreName,
		"operatorId": operator.UserID,
	}); err != nil {
		ret.FailedTotal = 1
		ret.Message = "测试通知发送失败：" + err.Error()
		return ret, nil
	}
	ret.Sent = true
	ret.SentTotal = 1
	ret.Message = "测试通知已发送，请在商家通知群或接收系统中确认。"
	return ret, nil
}

func (s *digitalStoreProfileService) TestWebhookNotifyScenarios(operator *dto.AuthPrincipal) (response.DigitalStoreWebhookTestResponse, error) {
	if operator == nil {
		return response.DigitalStoreWebhookTestResponse{}, errorsx.UnauthorizedI18n("error.auth.expired")
	}
	status := s.GetNotificationStatus()
	ret := response.DigitalStoreWebhookTestResponse{
		DigitalStoreNotificationStatusResponse: status,
		TestedAt:                               utils.FormatTime(time.Now()),
	}
	scenarios := s.buildWebhookNotifyTestScenarios(operator, ret.TestedAt)
	ret.Scenarios = make([]response.DigitalStoreWebhookTestScenarioResponse, 0, len(scenarios))
	if !status.Enabled {
		ret.Message = status.Message
		for _, scenario := range scenarios {
			ret.Scenarios = append(ret.Scenarios, response.DigitalStoreWebhookTestScenarioResponse{
				Key:       scenario.Key,
				EventType: scenario.EventType,
				Title:     scenario.Title,
				Message:   status.Message,
			})
		}
		return ret, nil
	}
	for _, scenario := range scenarios {
		item := response.DigitalStoreWebhookTestScenarioResponse{
			Key:       scenario.Key,
			EventType: scenario.EventType,
			Title:     scenario.Title,
		}
		if err := WebhookNotifyService.SendText(scenario.EventType, scenario.Title, scenario.Body, scenario.Metadata); err != nil {
			item.Message = err.Error()
			ret.FailedTotal++
			ret.Scenarios = append(ret.Scenarios, item)
			continue
		}
		item.Sent = true
		item.Message = "已发送"
		ret.SentTotal++
		ret.Scenarios = append(ret.Scenarios, item)
	}
	if ret.FailedTotal > 0 {
		ret.Message = fmt.Sprintf("关键通知测试未全部发送：成功 %d，失败 %d。请检查 Webhook 地址、格式、签名密钥或接收端日志。", ret.SentTotal, ret.FailedTotal)
		return ret, nil
	}
	ret.Sent = true
	ret.Message = fmt.Sprintf("已发送 %d 类关键事件测试，请在商家通知群或接收系统中确认。", ret.SentTotal)
	return ret, nil
}

type digitalStoreWebhookNotifyTestScenario struct {
	Key       string
	EventType string
	Title     string
	Body      string
	Metadata  map[string]any
}

func (s *digitalStoreProfileService) buildWebhookNotifyTestScenarios(operator *dto.AuthPrincipal, testedAt string) []digitalStoreWebhookNotifyTestScenario {
	cfg := s.loadConfig()
	brandName := valueOrDefault(cfg.BrandName, "AI数字店长样板")
	storeName := valueOrDefault(cfg.StoreName, "样板门店")
	operatorName := "system"
	operatorID := int64(0)
	if operator != nil {
		operatorName = valueOrDefault(operator.Username, "system")
		operatorID = operator.UserID
	}
	baseMetadata := map[string]any{
		"brandName":  brandName,
		"storeName":  storeName,
		"operatorId": operatorID,
		"test":       true,
	}
	withMetadata := func(scenario string, extra map[string]any) map[string]any {
		ret := map[string]any{}
		for key, value := range baseMetadata {
			ret[key] = value
		}
		ret["scenario"] = scenario
		for key, value := range extra {
			ret[key] = value
		}
		return ret
	}
	body := func(lines ...string) string {
		values := []string{
			"这是 AI 数字店长关键事件外部通知测试。",
			"品牌：" + brandName,
			"门店：" + storeName,
			"操作人：" + operatorName,
			"时间：" + testedAt,
		}
		values = append(values, lines...)
		return strings.Join(values, "\n")
	}
	return []digitalStoreWebhookNotifyTestScenario{
		{
			Key:       "high_intent_lead",
			EventType: "sales_lead_created",
			Title:     "高意向销售线索提醒",
			Body: body(
				"客户：王女士",
				"联系方式：13800000000 / 微信 wx_muse_test",
				"意向：预算 15000 元，关注脊护支撑款，准备到店试躺。",
			),
			Metadata: withMetadata("high_intent_lead", map[string]any{"intentLevel": "high", "actionUrl": "/dashboard/sales-leads"}),
		},
		{
			Key:       "appointment_lead",
			EventType: "sales_lead_created",
			Title:     "预约到店线索提醒",
			Body: body(
				"客户：李先生",
				"预约：本周六下午，两人到店试躺。",
				"提醒：请顾问确认到店时间、门店和体验产品。",
			),
			Metadata: withMetadata("appointment_lead", map[string]any{"buyingStage": "appointment", "actionUrl": "/dashboard/sales-leads?taskView=appointment"}),
		},
		{
			Key:       "human_handoff",
			EventType: "conversation_assigned",
			Title:     "客户转人工提醒",
			Body: body(
				"会话：#10001",
				"客户诉求：想确认最终成交价和库存。",
				"摘要：AI 已完成产品推荐，客户需要真人顾问跟进。",
			),
			Metadata: withMetadata("human_handoff", map[string]any{"conversationId": 10001, "actionUrl": "/dashboard/conversations"}),
		},
		{
			Key:       "unassigned_lead",
			EventType: "sales_lead_follow_up_reminder",
			Title:     "未分配线索跟进提醒",
			Body: body(
				"待处理：3 条未分配高意向或预约线索。",
				"建议：门店店长先分派顾问，再安排今日跟进。",
			),
			Metadata: withMetadata("unassigned_lead", map[string]any{"unassignedTotal": 3, "actionUrl": "/dashboard/sales-leads?ownerUserId=0"}),
		},
		{
			Key:       "after_sales_risk",
			EventType: "sales_lead_created",
			Title:     "售后风险线索提醒",
			Body: body(
				"客户：赵女士",
				"问题：已购床垫出现异响，客户表达投诉风险。",
				"建议：优先人工安抚并创建售后工单。",
			),
			Metadata: withMetadata("after_sales_risk", map[string]any{"buyingStage": "after_sales", "risk": true, "actionUrl": "/dashboard/sales-leads?taskView=after_sales"}),
		},
	}
}

func (s *digitalStoreProfileService) CleanupDemoData(operator *dto.AuthPrincipal) (response.DigitalStoreDemoDataCleanupResponse, error) {
	if operator == nil {
		return response.DigitalStoreDemoDataCleanupResponse{}, errorsx.UnauthorizedI18n("error.auth.expired")
	}
	deleted := map[string]int64{}
	if err := sqls.WithTransaction(func(ctx *sqls.TxContext) error {
		cleanupItems := []struct {
			key   string
			model any
		}{
			{key: "leadFollowUps", model: &models.LeadFollowUp{}},
			{key: "salesLeads", model: &models.SalesLead{}},
			{key: "ticketProgress", model: &models.TicketProgress{}},
			{key: "ticketTags", model: &models.TicketTag{}},
			{key: "tickets", model: &models.Ticket{}},
			{key: "notifications", model: &models.Notification{}},
			{key: "conversationInterrupts", model: &models.ConversationInterrupt{}},
			{key: "channelMessageOutbox", model: &models.ChannelMessageOutbox{}},
			{key: "wxworkKFMessageRefs", model: &models.WxWorkKFMessageRef{}},
			{key: "wxworkKFConversations", model: &models.WxWorkKFConversation{}},
			{key: "conversationAssignments", model: &models.ConversationAssignment{}},
			{key: "conversationTags", model: &models.ConversationTag{}},
			{key: "conversationEventLogs", model: &models.ConversationEventLog{}},
			{key: "conversationReadStates", model: &models.ConversationReadState{}},
			{key: "conversationParticipants", model: &models.ConversationParticipant{}},
			{key: "messages", model: &models.Message{}},
			{key: "conversations", model: &models.Conversation{}},
			{key: "knowledgeFeedback", model: &models.KnowledgeFeedback{}},
			{key: "knowledgeRetrieveHits", model: &models.KnowledgeRetrieveHit{}},
			{key: "knowledgeRetrieveLogs", model: &models.KnowledgeRetrieveLog{}},
			{key: "aiWorkflowNodeRuns", model: &models.AIWorkflowNodeRun{}},
			{key: "aiWorkflowRuns", model: &models.AIWorkflowRun{}},
			{key: "skillRunLogs", model: &models.SkillRunLog{}},
		}
		for _, item := range cleanupItems {
			if err := deleteDigitalStoreDemoData(ctx.Tx, deleted, item.key, item.model); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return response.DigitalStoreDemoDataCleanupResponse{}, err
	}
	total := int64(0)
	for _, count := range deleted {
		total += count
	}
	return response.DigitalStoreDemoDataCleanupResponse{
		CleanedAt: utils.FormatTime(time.Now()),
		Message:   fmt.Sprintf("已清理 %d 条演示运营数据；产品、活动、知识库、模型、Agent、渠道、客户档案和交付记录已保留。", total),
		Deleted:   deleted,
	}, nil
}

func deleteDigitalStoreDemoData(db *gorm.DB, deleted map[string]int64, key string, model any) error {
	result := db.Where("1 = 1").Delete(model)
	if result.Error != nil {
		return result.Error
	}
	deleted[key] = result.RowsAffected
	return nil
}

func (s *digitalStoreProfileService) GetLatestDeliveryRecord() *response.DigitalStoreDeliveryRecordResponse {
	item := repositories.DigitalStoreDeliveryRecordRepository.FindOne(sqls.DB(), sqls.NewCnd().
		Where("status <> ?", enums.StatusDeleted).
		Desc("id"))
	return buildDigitalStoreDeliveryRecordResponse(item)
}

func (s *digitalStoreProfileService) CreateDeliveryRecord(req request.DigitalStoreDeliveryRecordCreateRequest, operator *dto.AuthPrincipal) (*response.DigitalStoreDeliveryRecordResponse, error) {
	if operator == nil {
		return nil, errorsx.UnauthorizedI18n("error.auth.expired")
	}
	report := s.GetDeliveryReport(req.PublicBaseURL)
	status := s.GetSetupStatus()
	reportJSON, err := json.Marshal(report)
	if err != nil {
		return nil, err
	}
	acceptanceStatus := normalizeDigitalStoreAcceptanceStatus(req.AcceptanceStatus, report.Ready)
	item := &models.DigitalStoreDeliveryRecord{
		BrandName:         report.BrandName,
		StoreName:         report.StoreName,
		Ready:             report.Ready,
		AcceptanceStatus:  acceptanceStatus,
		AcceptanceSummary: strings.TrimSpace(req.AcceptanceSummary),
		AcceptanceCommand: report.AcceptanceCommand,
		DashboardURL:      report.DashboardURL,
		ChatURL:           report.ChatURL,
		WebChannelCode:    status.WebChannelCode,
		ReportMarkdown:    report.Markdown,
		ReportJSON:        string(reportJSON),
		Status:            enums.StatusOk,
		AuditFields:       utils.BuildAuditFields(operator),
	}
	if item.AcceptanceSummary == "" {
		item.AcceptanceSummary = defaultDigitalStoreAcceptanceSummary(acceptanceStatus, report.Ready)
	}
	if err := repositories.DigitalStoreDeliveryRecordRepository.Create(sqls.DB(), item); err != nil {
		return nil, err
	}
	return buildDigitalStoreDeliveryRecordResponse(item), nil
}

func (s *digitalStoreProfileService) CreateAcceptanceResultRecord(req request.DigitalStoreAcceptanceResultCreateRequest, operator *dto.AuthPrincipal) (*response.DigitalStoreDeliveryRecordResponse, error) {
	if operator == nil {
		return nil, errorsx.UnauthorizedI18n("error.auth.expired")
	}
	report := s.GetDeliveryReport(req.PublicBaseURL)
	status := s.GetSetupStatus()
	reportJSON, err := json.Marshal(report)
	if err != nil {
		return nil, err
	}
	resultJSON, err := json.Marshal(req.Results)
	if err != nil {
		return nil, err
	}
	scenarioTotal := req.ScenarioTotal
	if scenarioTotal <= 0 {
		scenarioTotal = len(req.Results)
	}
	passedTotal := req.PassedTotal
	if passedTotal < 0 {
		passedTotal = 0
	}
	failedTotal := req.FailedTotal
	if failedTotal < 0 {
		failedTotal = 0
	}
	acceptanceStatus := "passed"
	if failedTotal > 0 || (scenarioTotal > 0 && passedTotal < scenarioTotal) {
		acceptanceStatus = "failed"
	}
	startedAt := parseDigitalStoreAcceptanceTime(req.StartedAt)
	finishedAt := parseDigitalStoreAcceptanceTime(req.FinishedAt)
	item := &models.DigitalStoreDeliveryRecord{
		BrandName:            report.BrandName,
		StoreName:            report.StoreName,
		Ready:                report.Ready,
		AcceptanceStatus:     acceptanceStatus,
		AcceptanceSummary:    fmt.Sprintf("自动化冒烟验收：%d/%d 通过，%d 项失败。", passedTotal, scenarioTotal, failedTotal),
		AcceptanceCommand:    valueOrDefault(req.Command, report.AcceptanceCommand),
		ScenarioTotal:        scenarioTotal,
		PassedTotal:          passedTotal,
		FailedTotal:          failedTotal,
		AcceptanceStartedAt:  startedAt,
		AcceptanceFinishedAt: finishedAt,
		AcceptanceResultJSON: string(resultJSON),
		DashboardURL:         report.DashboardURL,
		ChatURL:              report.ChatURL,
		WebChannelCode:       status.WebChannelCode,
		ReportMarkdown:       report.Markdown,
		ReportJSON:           string(reportJSON),
		Status:               enums.StatusOk,
		AuditFields:          utils.BuildAuditFields(operator),
	}
	if err := repositories.DigitalStoreDeliveryRecordRepository.Create(sqls.DB(), item); err != nil {
		return nil, err
	}
	return buildDigitalStoreDeliveryRecordResponse(item), nil
}

func parseDigitalStoreAcceptanceTime(value string) *time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	layouts := []string{time.RFC3339Nano, time.RFC3339, "2006-01-02 15:04:05"}
	for _, layout := range layouts {
		parsed, err := time.Parse(layout, value)
		if err == nil {
			return &parsed
		}
	}
	return nil
}

func (s *digitalStoreProfileService) EnsureRuntime(operator *dto.AuthPrincipal) (response.DigitalStoreSetupStatusResponse, error) {
	if operator == nil {
		return response.DigitalStoreSetupStatusResponse{}, errorsx.UnauthorizedI18n("error.auth.expired")
	}
	cfg := s.loadConfig()
	if !cfg.Initialized {
		return response.DigitalStoreSetupStatusResponse{}, errorsx.InvalidParam("digital store profile is not initialized")
	}
	aiConfig := s.findActiveLLMConfig()
	if aiConfig == nil {
		return response.DigitalStoreSetupStatusResponse{}, errorsx.InvalidParam("enable an LLM model config before generating digital store runtime")
	}
	kbID, err := s.ensureKnowledgeBase(operator)
	if err != nil {
		return response.DigitalStoreSetupStatusResponse{}, err
	}
	if cfg.KnowledgeBaseID != kbID {
		cfg.KnowledgeBaseID = kbID
		if err := s.saveConfig(cfg, operator); err != nil {
			return response.DigitalStoreSetupStatusResponse{}, err
		}
	}
	if err := s.SyncKnowledgeFAQ(); err != nil {
		return response.DigitalStoreSetupStatusResponse{}, err
	}
	cfg = s.loadConfig()
	defaultTeamID, err := s.ensureDefaultHumanHandoffRuntime(operator)
	if err != nil {
		return response.DigitalStoreSetupStatusResponse{}, err
	}
	agent, err := s.ensureAgent(cfg, aiConfig.ID, defaultTeamID, operator)
	if err != nil {
		return response.DigitalStoreSetupStatusResponse{}, err
	}
	if err := s.ensureAgentWorkflowPublished(agent.ID, operator); err != nil {
		return response.DigitalStoreSetupStatusResponse{}, err
	}
	agent = AIAgentService.Get(agent.ID)
	if err := s.ensureWebChannel(cfg, agent, operator); err != nil {
		return response.DigitalStoreSetupStatusResponse{}, err
	}
	return s.GetSetupStatus(), nil
}

func (s *digitalStoreProfileService) UpdateProfile(req request.DigitalStoreProfileRequest, operator *dto.AuthPrincipal) (response.DigitalStoreProfileResponse, error) {
	if operator == nil {
		return response.DigitalStoreProfileResponse{}, errorsx.UnauthorizedI18n("error.auth.expired")
	}
	cfg, err := s.buildConfig(req)
	if err != nil {
		return response.DigitalStoreProfileResponse{}, err
	}
	if err := s.saveConfig(cfg, operator); err != nil {
		return response.DigitalStoreProfileResponse{}, err
	}
	if err := s.SyncKnowledgeFAQ(); err != nil {
		return response.DigitalStoreProfileResponse{}, err
	}
	return s.GetProfile(), nil
}

func (s *digitalStoreProfileService) SeedMuseProfile(operator *dto.AuthPrincipal) (response.DigitalStoreProfileResponse, error) {
	return s.ApplyTemplate("muse_bedding", operator)
}

func (s *digitalStoreProfileService) ApplyTemplate(templateCode string, operator *dto.AuthPrincipal) (response.DigitalStoreProfileResponse, error) {
	if operator == nil {
		return response.DigitalStoreProfileResponse{}, errorsx.UnauthorizedI18n("error.auth.expired")
	}
	bundle, err := s.buildBuiltinTemplateBundle(templateCode)
	if err != nil {
		return response.DigitalStoreProfileResponse{}, err
	}
	return s.applyTemplateBundle(bundle, operator)
}

func (s *digitalStoreProfileService) ApplyImportedTemplate(req request.DigitalStoreTemplateImportRequest, operator *dto.AuthPrincipal) (response.DigitalStoreProfileResponse, error) {
	if operator == nil {
		return response.DigitalStoreProfileResponse{}, errorsx.UnauthorizedI18n("error.auth.expired")
	}
	bundle, err := s.buildImportedTemplateBundle(req)
	if err != nil {
		return response.DigitalStoreProfileResponse{}, err
	}
	return s.applyTemplateBundle(bundle, operator)
}

func (s *digitalStoreProfileService) applyTemplateBundle(bundle digitalStoreTemplateBundle, operator *dto.AuthPrincipal) (response.DigitalStoreProfileResponse, error) {
	cfg := bundle.cfg
	cfg.TemplateCode = bundle.template.Code
	cfg.TemplateVersion = bundle.template.Version
	cfg.TemplateAppliedAt = utils.FormatTime(time.Now())
	current := s.loadConfig()
	if current.KnowledgeBaseID > 0 {
		cfg.KnowledgeBaseID = current.KnowledgeBaseID
	}
	if current.KnowledgeFAQID > 0 {
		cfg.KnowledgeFAQID = current.KnowledgeFAQID
	}
	if strings.TrimSpace(current.EnterpriseWebhookURL) != "" {
		cfg.EnterpriseWebhookURL = current.EnterpriseWebhookURL
	}
	if cfg.KnowledgeBaseID == 0 {
		kbID, err := s.ensureKnowledgeBase(operator)
		if err != nil {
			return response.DigitalStoreProfileResponse{}, err
		}
		cfg.KnowledgeBaseID = kbID
	}
	if err := s.saveConfig(cfg, operator); err != nil {
		return response.DigitalStoreProfileResponse{}, err
	}
	if err := ProductService.UpsertTemplateProducts(bundle.products, operator); err != nil {
		return response.DigitalStoreProfileResponse{}, err
	}
	if err := PromotionService.UpsertTemplatePromotions(bundle.promotions, operator); err != nil {
		return response.DigitalStoreProfileResponse{}, err
	}
	if err := s.SyncKnowledgeFAQ(); err != nil {
		return response.DigitalStoreProfileResponse{}, err
	}
	return s.GetProfile(), nil
}

func (s *digitalStoreProfileService) SyncKnowledgeFAQ() error {
	cfg := s.loadConfig()
	kbID, err := resolveDigitalStoreKnowledgeBaseID(cfg.KnowledgeBaseID)
	if err != nil {
		return err
	}
	cfg.KnowledgeBaseID = kbID
	question, answer, similarQuestions := BuildDigitalStoreProfileFAQContent(cfg)
	similarJSON, err := json.Marshal(similarQuestions)
	if err != nil {
		return err
	}
	now := time.Now()
	faq := repositories.KnowledgeFAQRepository.Get(sqls.DB(), cfg.KnowledgeFAQID)
	if faq == nil && cfg.KnowledgeFAQID > 0 {
		cfg.KnowledgeFAQID = 0
	}
	if faq == nil {
		faq = &models.KnowledgeFAQ{
			KnowledgeBaseID:  kbID,
			Question:         question,
			Answer:           answer,
			SimilarQuestions: string(similarJSON),
			IndexStatus:      enums.KnowledgeDocumentIndexStatusPending,
			Status:           enums.StatusOk,
			Remark:           "digital-store-profile",
			AuditFields: models.AuditFields{
				CreatedAt:      now,
				CreateUserID:   0,
				CreateUserName: "system",
				UpdatedAt:      now,
				UpdateUserID:   0,
				UpdateUserName: "system",
			},
		}
		if err := repositories.KnowledgeFAQRepository.Create(sqls.DB(), faq); err != nil {
			return err
		}
		cfg.KnowledgeFAQID = faq.ID
	} else {
		if err := repositories.KnowledgeFAQRepository.Updates(sqls.DB(), faq.ID, map[string]any{
			"knowledge_base_id": kbID,
			"question":          question,
			"answer":            answer,
			"similar_questions": string(similarJSON),
			"index_status":      enums.KnowledgeDocumentIndexStatusPending,
			"indexed_at":        nil,
			"index_error":       "",
			"status":            enums.StatusOk,
			"remark":            "digital-store-profile",
			"updated_at":        now,
			"update_user_id":    0,
			"update_user_name":  "system",
		}); err != nil {
			return err
		}
	}
	if err := s.saveConfigWithSystemAudit(cfg); err != nil {
		return err
	}
	return rag.Index.IndexFAQByID(context.Background(), cfg.KnowledgeFAQID)
}

func (s *digitalStoreProfileService) loadConfig() digitalStoreProfileConfig {
	item := repositories.SystemConfigRepository.Take(sqls.DB(), "config_key = ?", digitalStoreProfileConfigKey)
	if item == nil || strings.TrimSpace(item.ConfigValue) == "" {
		return digitalStoreProfileConfig{}
	}
	cfg := digitalStoreProfileConfig{}
	if err := json.Unmarshal([]byte(item.ConfigValue), &cfg); err != nil {
		return digitalStoreProfileConfig{}
	}
	return cfg
}

func (s *digitalStoreProfileService) buildConfig(req request.DigitalStoreProfileRequest) (digitalStoreProfileConfig, error) {
	brandName := strings.TrimSpace(req.BrandName)
	storeName := strings.TrimSpace(req.StoreName)
	if brandName == "" {
		return digitalStoreProfileConfig{}, errorsx.InvalidParam("brand name is required")
	}
	if storeName == "" {
		return digitalStoreProfileConfig{}, errorsx.InvalidParam("store name is required")
	}
	cfg := digitalStoreProfileConfig{
		DigitalStoreProfileRequest: request.DigitalStoreProfileRequest{
			BrandName:            brandName,
			Industry:             strings.TrimSpace(req.Industry),
			StoreName:            storeName,
			StoreAddress:         strings.TrimSpace(req.StoreAddress),
			BusinessHours:        strings.TrimSpace(req.BusinessHours),
			ContactPhone:         strings.TrimSpace(req.ContactPhone),
			ServiceWeChat:        strings.TrimSpace(req.ServiceWeChat),
			EnterpriseWebhookURL: strings.TrimSpace(req.EnterpriseWebhookURL),
			AIManagerName:        strings.TrimSpace(req.AIManagerName),
			AIPersona:            strings.TrimSpace(req.AIPersona),
			ReplyStyle:           strings.TrimSpace(req.ReplyStyle),
			ForbiddenClaims:      strings.TrimSpace(req.ForbiddenClaims),
			HandoffPolicy:        strings.TrimSpace(req.HandoffPolicy),
			AppointmentPolicy:    strings.TrimSpace(req.AppointmentPolicy),
			KnowledgeBaseID:      req.KnowledgeBaseID,
			Initialized:          true,
		},
	}
	if cfg.AIManagerName == "" {
		cfg.AIManagerName = brandName + "数字店长"
	}
	if cfg.KnowledgeBaseID > 0 {
		if _, err := resolveDigitalStoreKnowledgeBaseID(cfg.KnowledgeBaseID); err != nil {
			return digitalStoreProfileConfig{}, err
		}
	}
	if current := s.loadConfig(); current.Initialized || current.KnowledgeFAQID > 0 {
		cfg.KnowledgeFAQID = current.KnowledgeFAQID
		cfg.TemplateCode = current.TemplateCode
		cfg.TemplateVersion = current.TemplateVersion
		cfg.TemplateAppliedAt = current.TemplateAppliedAt
	}
	return cfg, nil
}

func (s *digitalStoreProfileService) saveConfig(cfg digitalStoreProfileConfig, operator *dto.AuthPrincipal) error {
	value, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	now := time.Now()
	item := repositories.SystemConfigRepository.Take(sqls.DB(), "config_key = ?", digitalStoreProfileConfigKey)
	if item == nil {
		return repositories.SystemConfigRepository.Create(sqls.DB(), &models.SystemConfig{
			ConfigKey:   digitalStoreProfileConfigKey,
			ConfigValue: string(value),
			GroupCode:   digitalStoreConfigGroup,
			Title:       "AI数字店长配置",
			Description: "单商家部署下的品牌、门店、人设、预约和转人工规则",
			Status:      enums.StatusOk,
			AuditFields: utils.BuildAuditFields(operator),
		})
	}
	return repositories.SystemConfigRepository.Updates(sqls.DB(), item.ID, map[string]any{
		"config_value":     string(value),
		"group_code":       digitalStoreConfigGroup,
		"title":            "AI数字店长配置",
		"description":      "单商家部署下的品牌、门店、人设、预约和转人工规则",
		"status":           enums.StatusOk,
		"updated_at":       now,
		"update_user_id":   operator.UserID,
		"update_user_name": operator.Username,
	})
}

func (s *digitalStoreProfileService) saveConfigWithSystemAudit(cfg digitalStoreProfileConfig) error {
	value, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	now := time.Now()
	item := repositories.SystemConfigRepository.Take(sqls.DB(), "config_key = ?", digitalStoreProfileConfigKey)
	if item == nil {
		return repositories.SystemConfigRepository.Create(sqls.DB(), &models.SystemConfig{
			ConfigKey:   digitalStoreProfileConfigKey,
			ConfigValue: string(value),
			GroupCode:   digitalStoreConfigGroup,
			Title:       "AI数字店长配置",
			Description: "单商家部署下的品牌、门店、人设、预约和转人工规则",
			Status:      enums.StatusOk,
			AuditFields: models.AuditFields{
				CreatedAt:      now,
				CreateUserID:   0,
				CreateUserName: "system",
				UpdatedAt:      now,
				UpdateUserID:   0,
				UpdateUserName: "system",
			},
		})
	}
	return repositories.SystemConfigRepository.Updates(sqls.DB(), item.ID, map[string]any{
		"config_value":     string(value),
		"group_code":       digitalStoreConfigGroup,
		"title":            "AI数字店长配置",
		"description":      "单商家部署下的品牌、门店、人设、预约和转人工规则",
		"status":           enums.StatusOk,
		"updated_at":       now,
		"update_user_id":   0,
		"update_user_name": "system",
	})
}

func resolveDigitalStoreKnowledgeBaseID(id int64) (int64, error) {
	if id > 0 {
		kb := repositories.KnowledgeBaseRepository.Get(sqls.DB(), id)
		if kb == nil || kb.Status == enums.StatusDeleted || kb.KnowledgeType != string(enums.KnowledgeBaseTypeFAQ) {
			return 0, errorsx.InvalidParam("usable FAQ knowledge base not found")
		}
		return id, nil
	}
	kb := repositories.KnowledgeBaseRepository.FindOne(sqls.DB(), sqls.NewCnd().Eq("knowledge_type", string(enums.KnowledgeBaseTypeFAQ)).Where("status <> ?", enums.StatusDeleted).Asc("id"))
	if kb == nil {
		return 0, errorsx.InvalidParam("create a FAQ knowledge base before syncing digital store profile")
	}
	return kb.ID, nil
}

func (s *digitalStoreProfileService) ensureKnowledgeBase(operator *dto.AuthPrincipal) (int64, error) {
	cfg := s.loadConfig()
	if cfg.KnowledgeBaseID > 0 {
		return resolveDigitalStoreKnowledgeBaseID(cfg.KnowledgeBaseID)
	}
	if kb := repositories.KnowledgeBaseRepository.FindOne(sqls.DB(), sqls.NewCnd().
		Eq("knowledge_type", string(enums.KnowledgeBaseTypeFAQ)).
		Where("status <> ?", enums.StatusDeleted).
		Asc("id")); kb != nil {
		return kb.ID, nil
	}
	item, err := KnowledgeBaseService.CreateKnowledgeBase(request.CreateKnowledgeBaseRequest{
		Name:                  "数字店长 FAQ 知识库",
		Description:           "数字店长品牌、产品、活动和门店规则知识库",
		KnowledgeType:         string(enums.KnowledgeBaseTypeFAQ),
		DefaultTopK:           10,
		DefaultScoreThreshold: 0.35,
		DefaultRerankLimit:    5,
		AnswerMode:            2,
		Remark:                digitalStoreRuntimeSeedRemark,
	}, operator)
	if err != nil {
		return 0, err
	}
	return item.ID, nil
}

func (s *digitalStoreProfileService) ensureDefaultHumanHandoffRuntime(operator *dto.AuthPrincipal) (int64, error) {
	team, err := s.ensureDefaultAgentTeam(operator)
	if err != nil {
		return 0, err
	}
	if team == nil || team.ID <= 0 {
		return 0, nil
	}
	userID := resolveDigitalStoreDefaultConsultantUserID(operator)
	if userID > 0 {
		if err := s.ensureDefaultAgentProfile(team.ID, userID, operator); err != nil {
			return 0, err
		}
	}
	if err := s.ensureDefaultAgentTeamSchedule(team.ID, operator); err != nil {
		return 0, err
	}
	return team.ID, nil
}

func (s *digitalStoreProfileService) ensureDefaultAgentTeam(operator *dto.AuthPrincipal) (*models.AgentTeam, error) {
	team := repositories.AgentTeamRepository.FindOne(sqls.DB(), sqls.NewCnd().
		Where("(remark = ? OR name = ?)", digitalStoreRuntimeSeedRemark, digitalStoreDefaultTeamName).
		Where("status <> ?", enums.StatusDeleted).
		Asc("id"))
	if team == nil {
		return AgentTeamService.CreateAgentTeam(request.CreateAgentTeamRequest{
			Name:         digitalStoreDefaultTeamName,
			LeaderUserID: resolveDigitalStoreDefaultConsultantUserID(operator),
			Status:       int(enums.StatusOk),
			Description:  "AI 数字店长默认人工接待顾问组",
			Remark:       digitalStoreRuntimeSeedRemark,
		}, operator)
	}
	if team.Status != enums.StatusOk || strings.TrimSpace(team.Remark) == "" {
		if err := repositories.AgentTeamRepository.Updates(sqls.DB(), team.ID, map[string]any{
			"status":           enums.StatusOk,
			"remark":           valueOrDefault(team.Remark, digitalStoreRuntimeSeedRemark),
			"updated_at":       time.Now(),
			"update_user_id":   operator.UserID,
			"update_user_name": operator.Username,
		}); err != nil {
			return nil, err
		}
		team = AgentTeamService.Get(team.ID)
	}
	return team, nil
}

func (s *digitalStoreProfileService) ensureDefaultAgentProfile(teamID, userID int64, operator *dto.AuthPrincipal) error {
	if teamID <= 0 || userID <= 0 || UserService.Get(userID) == nil {
		return nil
	}
	displayName := digitalStoreDefaultConsultantDisplayName(userID)
	profile := AgentProfileService.GetByUserID(userID)
	if profile == nil {
		agentCode := digitalStoreDefaultAgentCode
		if exists := AgentProfileService.Take("agent_code = ?", agentCode); exists != nil {
			agentCode = fmt.Sprintf("%s_%d", digitalStoreDefaultAgentCode, userID)
		}
		_, err := AgentProfileService.CreateAgentProfile(request.CreateAgentProfileRequest{
			UserID:                userID,
			TeamID:                teamID,
			AgentCode:             agentCode,
			DisplayName:           displayName,
			ServiceStatus:         enums.ServiceStatusIdle,
			MaxConcurrentCount:    10,
			PriorityLevel:         10,
			AutoAssignEnabled:     true,
			ReceiveOfflineMessage: true,
			Remark:                digitalStoreRuntimeSeedRemark,
		}, operator)
		return err
	}
	return repositories.AgentProfileRepository.Updates(sqls.DB(), profile.ID, map[string]any{
		"team_id":                 teamID,
		"display_name":            valueOrDefault(profile.DisplayName, displayName),
		"service_status":          enums.ServiceStatusIdle,
		"max_concurrent_count":    maxPositiveInt(profile.MaxConcurrentCount, 10),
		"priority_level":          maxPositiveInt(profile.PriorityLevel, 10),
		"auto_assign_enabled":     true,
		"receive_offline_message": true,
		"status":                  enums.StatusOk,
		"remark":                  valueOrDefault(profile.Remark, digitalStoreRuntimeSeedRemark),
		"updated_at":              time.Now(),
		"update_user_id":          operator.UserID,
		"update_user_name":        operator.Username,
	})
}

func (s *digitalStoreProfileService) ensureDefaultAgentTeamSchedule(teamID int64, operator *dto.AuthPrincipal) error {
	if teamID <= 0 {
		return nil
	}
	now := time.Now()
	if existing := AgentTeamScheduleService.Find(sqls.NewCnd().
		Eq("team_id", teamID).
		Eq("status", enums.StatusOk).
		Lte("start_at", now).
		Gt("end_at", now)); len(existing) > 0 {
		return nil
	}
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	end := start.AddDate(0, 0, 31).Add(-time.Second)
	return repositories.AgentTeamScheduleRepository.Create(sqls.DB(), &models.AgentTeamSchedule{
		TeamID:  teamID,
		StartAt: start,
		EndAt:   end,
		Remark:  digitalStoreRuntimeSeedRemark,
		Status:  enums.StatusOk,
		AuditFields: models.AuditFields{
			CreatedAt:      now,
			CreateUserID:   operator.UserID,
			CreateUserName: operator.Username,
			UpdatedAt:      now,
			UpdateUserID:   operator.UserID,
			UpdateUserName: operator.Username,
		},
	})
}

func resolveDigitalStoreDefaultConsultantUserID(operator *dto.AuthPrincipal) int64 {
	if operator != nil && operator.UserID > 0 {
		if user := UserService.Get(operator.UserID); user != nil && user.Status == enums.StatusOk {
			return operator.UserID
		}
	}
	users := UserService.Find(sqls.NewCnd().Eq("status", enums.StatusOk).Asc("id"))
	if len(users) == 0 {
		return 0
	}
	return users[0].ID
}

func digitalStoreDefaultConsultantDisplayName(userID int64) string {
	user := UserService.Get(userID)
	if user == nil {
		return "门店顾问"
	}
	if nickname := strings.TrimSpace(user.Nickname); nickname != "" {
		return nickname
	}
	if username := strings.TrimSpace(user.Username); username != "" {
		return username
	}
	return "门店顾问"
}

func normalizeDigitalStoreAgentTeamIDs(existing string, defaultTeamID int64) []int64 {
	ids := utils.SplitInt64s(existing)
	if defaultTeamID <= 0 {
		return ids
	}
	for _, id := range ids {
		if id == defaultTeamID {
			return ids
		}
	}
	return append(ids, defaultTeamID)
}

func maxPositiveInt(current int, fallback int) int {
	if current > 0 {
		return current
	}
	return fallback
}

func (s *digitalStoreProfileService) findActiveLLMConfig() *models.AIConfig {
	return repositories.AIConfigRepository.FindOne(sqls.DB(), sqls.NewCnd().
		Eq("model_type", enums.AIModelTypeLLM).
		Eq("status", enums.StatusOk).
		Asc("id"))
}

func (s *digitalStoreProfileService) findActiveEmbeddingConfig() *models.AIConfig {
	return repositories.AIConfigRepository.FindOne(sqls.DB(), sqls.NewCnd().
		Eq("model_type", enums.AIModelTypeEmbedding).
		Eq("status", enums.StatusOk).
		Asc("id"))
}

func (s *digitalStoreProfileService) findDigitalStoreAgent(cfg digitalStoreProfileConfig) *models.AIAgent {
	name := defaultDigitalStoreAgentName(cfg)
	if name != "" {
		if item := repositories.AIAgentRepository.FindOne(sqls.DB(), sqls.NewCnd().
			Eq("name", name).
			Where("status <> ?", enums.StatusDeleted).
			Asc("id")); item != nil {
			return item
		}
	}
	return nil
}

func (s *digitalStoreProfileService) ensureAgent(cfg digitalStoreProfileConfig, aiConfigID int64, defaultTeamID int64, operator *dto.AuthPrincipal) (*models.AIAgent, error) {
	name := defaultDigitalStoreAgentName(cfg)
	desiredPrompt := defaultDigitalStoreAgentPrompt(cfg)
	desiredWelcome := defaultDigitalStoreWelcomeMessage(cfg)
	desiredTeamIDs := normalizeDigitalStoreAgentTeamIDs("", defaultTeamID)
	agent := s.findDigitalStoreAgent(cfg)
	if agent == nil {
		if exists := repositories.AIAgentRepository.Take(sqls.DB(), "name = ?", name); exists != nil {
			name = fmt.Sprintf("%s %s", name, time.Now().Format("20060102150405"))
		}
		return AIAgentService.CreateAIAgent(request.CreateAIAgentRequest{
			Name:                name,
			Description:         "单商家 AI 数字店长默认接待 Agent",
			AIConfigID:          aiConfigID,
			ServiceMode:         enums.IMConversationServiceModeAIFirst,
			SystemPrompt:        desiredPrompt,
			WelcomeMessage:      desiredWelcome,
			ReplyTimeoutSeconds: 180,
			TeamIDs:             desiredTeamIDs,
			HandoffMode:         enums.AIAgentHandoffModeWaitPool,
			FallbackMode:        enums.AIAgentFallbackModeNoAnswer,
			FallbackMessage:     "这个问题我需要让门店顾问进一步确认，可以为你转人工或先留下联系方式。",
			KnowledgeIDs:        []int64{cfg.KnowledgeBaseID},
		}, operator)
	}
	desiredTeamIDs = normalizeDigitalStoreAgentTeamIDs(agent.TeamIDs, defaultTeamID)
	if agent.Status != enums.StatusOk ||
		agent.AIConfigID != aiConfigID ||
		agent.KnowledgeIDs != fmt.Sprint(cfg.KnowledgeBaseID) ||
		agent.TeamIDs != utils.JoinInt64s(desiredTeamIDs) ||
		strings.TrimSpace(agent.SystemPrompt) != desiredPrompt ||
		strings.TrimSpace(agent.WelcomeMessage) != desiredWelcome {
		if err := AIAgentService.UpdateAIAgent(request.UpdateAIAgentRequest{
			ID: agent.ID,
			CreateAIAgentRequest: request.CreateAIAgentRequest{
				Name:                agent.Name,
				Description:         valueOrDefault(agent.Description, "单商家 AI 数字店长默认接待 Agent"),
				AIConfigID:          aiConfigID,
				ServiceMode:         enums.IMConversationServiceModeAIFirst,
				SystemPrompt:        desiredPrompt,
				WelcomeMessage:      desiredWelcome,
				ReplyTimeoutSeconds: agent.ReplyTimeoutSeconds,
				TeamIDs:             desiredTeamIDs,
				HandoffMode:         enums.AIAgentHandoffModeWaitPool,
				FallbackMode:        enums.AIAgentFallbackModeNoAnswer,
				FallbackMessage:     valueOrDefault(agent.FallbackMessage, "这个问题我需要让门店顾问进一步确认，可以为你转人工或先留下联系方式。"),
				KnowledgeIDs:        []int64{cfg.KnowledgeBaseID},
			},
		}, operator); err != nil {
			return nil, err
		}
		if agent.Status != enums.StatusOk {
			if err := repositories.AIAgentRepository.Updates(sqls.DB(), agent.ID, map[string]any{
				"status":           enums.StatusOk,
				"update_user_id":   operator.UserID,
				"update_user_name": operator.Username,
				"updated_at":       time.Now(),
			}); err != nil {
				return nil, err
			}
		}
	}
	return AIAgentService.Get(agent.ID), nil
}

func (s *digitalStoreProfileService) ensureAgentWorkflowPublished(agentID int64, operator *dto.AuthPrincipal) error {
	agent := AIAgentService.Get(agentID)
	if agent == nil {
		return errorsx.InvalidParamI18n("error.e0002")
	}
	if agent.WorkflowVersionID > 0 {
		return nil
	}
	_, err := AIWorkflowService.PublishAgentWorkflow(request.PublishAIWorkflowRequest{
		AgentID:    agent.ID,
		Definition: AIWorkflowService.DefaultAgentWorkflowDefinition(),
	}, operator)
	return err
}

func (s *digitalStoreProfileService) findWebChannel(cfg digitalStoreProfileConfig) *models.Channel {
	channelName := defaultDigitalStoreWebChannelName(cfg)
	return repositories.ChannelRepository.FindOne(sqls.DB(), sqls.NewCnd().
		Eq("channel_type", enums.ChannelTypeWeb).
		Where("(name = ? OR remark = ?)", channelName, digitalStoreRuntimeSeedRemark).
		Where("status <> ?", enums.StatusDeleted).
		Asc("id"))
}

func (s *digitalStoreProfileService) ensureWebChannel(cfg digitalStoreProfileConfig, agent *models.AIAgent, operator *dto.AuthPrincipal) error {
	if agent == nil || agent.ID <= 0 {
		return errorsx.InvalidParamI18n("error.e0002")
	}
	channelName := defaultDigitalStoreWebChannelName(cfg)
	configJSON, err := defaultDigitalStoreWebChannelConfig(cfg)
	if err != nil {
		return err
	}
	channel := s.findWebChannel(cfg)
	if channel == nil {
		_, err := ChannelService.CreateChannel(request.CreateChannelRequest{
			ChannelType: enums.ChannelTypeWeb,
			AIAgentID:   agent.ID,
			Name:        channelName,
			ConfigJSON:  configJSON,
			Status:      int(enums.StatusOk),
			Remark:      digitalStoreRuntimeSeedRemark,
		}, operator)
		return err
	}
	configJSON = channel.ConfigJSON
	if strings.TrimSpace(configJSON) == "" {
		configJSON, err = defaultDigitalStoreWebChannelConfig(cfg)
		if err != nil {
			return err
		}
	}
	return ChannelService.UpdateChannel(request.UpdateChannelRequest{
		ID: channel.ID,
		CreateChannelRequest: request.CreateChannelRequest{
			ChannelType: enums.ChannelTypeWeb,
			AIAgentID:   agent.ID,
			Name:        valueOrDefault(channel.Name, channelName),
			ConfigJSON:  configJSON,
			Status:      int(enums.StatusOk),
			Remark:      channel.Remark,
		},
	}, operator)
}

func buildDigitalStoreProfileResponse(cfg digitalStoreProfileConfig) response.DigitalStoreProfileResponse {
	return response.DigitalStoreProfileResponse{
		BrandName:            cfg.BrandName,
		Industry:             cfg.Industry,
		StoreName:            cfg.StoreName,
		StoreAddress:         cfg.StoreAddress,
		BusinessHours:        cfg.BusinessHours,
		ContactPhone:         cfg.ContactPhone,
		ServiceWeChat:        cfg.ServiceWeChat,
		EnterpriseWebhookURL: cfg.EnterpriseWebhookURL,
		AIManagerName:        cfg.AIManagerName,
		AIPersona:            cfg.AIPersona,
		ReplyStyle:           cfg.ReplyStyle,
		ForbiddenClaims:      cfg.ForbiddenClaims,
		HandoffPolicy:        cfg.HandoffPolicy,
		AppointmentPolicy:    cfg.AppointmentPolicy,
		KnowledgeBaseID:      cfg.KnowledgeBaseID,
		KnowledgeFAQID:       cfg.KnowledgeFAQID,
		TemplateCode:         cfg.TemplateCode,
		TemplateVersion:      cfg.TemplateVersion,
		TemplateAppliedAt:    cfg.TemplateAppliedAt,
		Initialized:          cfg.Initialized,
	}
}

func buildDigitalStoreDeliveryRecordResponse(item *models.DigitalStoreDeliveryRecord) *response.DigitalStoreDeliveryRecordResponse {
	if item == nil {
		return nil
	}
	ret := &response.DigitalStoreDeliveryRecordResponse{
		ID:                item.ID,
		BrandName:         item.BrandName,
		StoreName:         item.StoreName,
		Ready:             item.Ready,
		AcceptanceStatus:  item.AcceptanceStatus,
		AcceptanceSummary: item.AcceptanceSummary,
		AcceptanceCommand: item.AcceptanceCommand,
		ScenarioTotal:     item.ScenarioTotal,
		PassedTotal:       item.PassedTotal,
		FailedTotal:       item.FailedTotal,
		AcceptanceStartedAt: func() string {
			if item.AcceptanceStartedAt == nil {
				return ""
			}
			return utils.FormatTime(*item.AcceptanceStartedAt)
		}(),
		AcceptanceFinishedAt: func() string {
			if item.AcceptanceFinishedAt == nil {
				return ""
			}
			return utils.FormatTime(*item.AcceptanceFinishedAt)
		}(),
		DashboardURL:   item.DashboardURL,
		ChatURL:        item.ChatURL,
		WebChannelCode: item.WebChannelCode,
		CreatedAt:      utils.FormatTime(item.CreatedAt),
		CreateUserName: item.CreateUserName,
	}
	if strings.TrimSpace(item.AcceptanceResultJSON) != "" {
		var results []response.DigitalStoreAcceptanceScenarioResultResponse
		if err := json.Unmarshal([]byte(item.AcceptanceResultJSON), &results); err == nil {
			ret.AcceptanceResults = results
		}
	}
	return ret
}

func buildDigitalStoreMissingSteps(status response.DigitalStoreSetupStatusResponse) []string {
	missing := make([]string, 0, 10)
	if !status.ProfileInitialized {
		missing = append(missing, "配置品牌与数字店长人设")
	}
	if status.ProductTotal == 0 {
		missing = append(missing, "导入产品库")
	}
	if status.PromotionTotal == 0 {
		missing = append(missing, "导入活动库")
	}
	if status.ProductTotal > 0 && (status.ProductKnowledgeUnsyncedTotal > 0 || status.ProductKnowledgeFailedTotal > 0) {
		missing = append(missing, "重建产品知识索引")
	}
	if status.PromotionTotal > 0 && (status.PromotionKnowledgeUnsyncedTotal > 0 || status.PromotionKnowledgeFailedTotal > 0) {
		missing = append(missing, "重建活动知识索引")
	}
	if status.LLMConfigID == 0 {
		missing = append(missing, "启用聊天模型配置")
	}
	if status.EmbeddingConfigID == 0 {
		missing = append(missing, "启用 Embedding 模型配置")
	}
	if status.KnowledgeBaseID == 0 || status.KnowledgeFAQID == 0 {
		missing = append(missing, "同步店长知识")
	}
	if status.AgentID == 0 || !status.WorkflowPublished {
		missing = append(missing, "生成并发布数字店长 Agent")
	}
	if status.AgentID > 0 && !status.HumanHandoff.Ready {
		missing = append(missing, "配置人工接待顾问组、排班和可自动分配顾问")
	}
	if status.WebChannelID == 0 {
		missing = append(missing, "生成 Web 聊天渠道")
	}
	return missing
}

func buildDeliveryReportItem(label string, ok bool, value string) response.DigitalStoreDeliveryReportItem {
	status := "待完成"
	if ok {
		status = "完成"
	}
	item := response.DigitalStoreDeliveryReportItem{
		Label:       label,
		Status:      status,
		Value:       valueOrDefault(value, "-"),
		ActionHref:  deliveryReportItemActionHref(label),
		ActionLabel: deliveryReportItemActionLabel(label),
	}
	if ok {
		item.ActionHref = ""
		item.ActionLabel = ""
	}
	return item
}

func deliveryReportItemActionHref(label string) string {
	switch label {
	case "品牌与门店", "客户入口品牌化":
		return "/dashboard/digital-store"
	case "产品库", "产品知识索引":
		return "/dashboard/products"
	case "活动库", "活动知识索引":
		return "/dashboard/promotions"
	case "聊天模型", "Embedding 模型":
		return "/dashboard/ai-configs"
	case "知识库":
		return "/dashboard/knowledge"
	case "数字店长 Agent":
		return "/dashboard/ai-agents"
	case "人工接待配置":
		return "/dashboard/agents"
	case "Web 聊天渠道":
		return "/dashboard/channels"
	case "外部通知", "上线安全自检":
		return "/dashboard/store-setup"
	default:
		return ""
	}
}

func deliveryReportItemActionLabel(label string) string {
	switch label {
	case "产品知识索引", "活动知识索引", "知识库":
		return "去同步"
	case "聊天模型", "Embedding 模型":
		return "去配置模型"
	case "数字店长 Agent", "Web 聊天渠道":
		return "去生成"
	case "人工接待配置":
		return "去配置顾问"
	case "外部通知", "上线安全自检":
		return "去处理"
	default:
		return "去配置"
	}
}

func formatKnowledgeCoverage(synced int64, total int64, unsynced int64, failed int64) string {
	parts := []string{fmt.Sprintf("已同步 %d/%d", synced, total)}
	if unsynced > 0 {
		parts = append(parts, fmt.Sprintf("未同步 %d", unsynced))
	}
	if failed > 0 {
		parts = append(parts, fmt.Sprintf("索引失败 %d", failed))
	}
	return strings.Join(parts, "，")
}

func safeWebhookNotifyConfig() (cfg config.WebhookNotifyConfig, ok bool) {
	runtimeConfig, ok := safeRuntimeConfig()
	if !ok {
		return config.WebhookNotifyConfig{}, false
	}
	return runtimeConfig.Notify.Webhook, true
}

func safeRuntimeConfig() (cfg config.Config, ok bool) {
	defer func() {
		if recover() != nil {
			cfg = config.Config{}
			ok = false
		}
	}()
	return config.Current(), true
}

func buildSecurityCheck(key string, label string, status string, message string) response.DigitalStoreSecurityCheckResponse {
	item := response.DigitalStoreSecurityCheckResponse{
		Key:         key,
		Label:       label,
		Status:      status,
		Message:     valueOrDefault(message, "-"),
		ActionHref:  securityCheckActionHref(key),
		ActionLabel: securityCheckActionLabel(key),
	}
	if status == "ok" {
		item.ActionHref = ""
		item.ActionLabel = ""
	}
	return item
}

func securityCheckActionHref(key string) string {
	switch key {
	case "notification":
		return "/dashboard/store-setup"
	case "auth_lockout":
		return "/dashboard/settings"
	case "customer_session_secret", "bootstrap_admin_password", "cors_allowed_origins", "database", "vector_db", "webhook_secret", "config":
		return "/dashboard/store-setup"
	default:
		return ""
	}
}

func securityCheckActionLabel(key string) string {
	switch key {
	case "notification":
		return "测试通知"
	case "auth_lockout":
		return "查看设置"
	case "customer_session_secret", "bootstrap_admin_password", "cors_allowed_origins", "database", "vector_db", "webhook_secret", "config":
		return "查看部署配置"
	default:
		return "去处理"
	}
}

func isBlankOrPlaceholder(value string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	switch value {
	case "", "changeme", "change-me", "replace-me", "replace-with-a-random-secret", "please-change", "your-secret", "secret":
		return true
	}
	return false
}

func normalizedCORSOrigins(values []string) []string {
	ret := []string{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			ret = append(ret, value)
		}
	}
	return ret
}

func hasWildcardOrigin(values []string) bool {
	for _, value := range values {
		if strings.TrimSpace(value) == "*" {
			return true
		}
	}
	return false
}

func allLocalOrigins(values []string) bool {
	if len(values) == 0 {
		return false
	}
	for _, value := range values {
		lower := strings.ToLower(value)
		if !strings.Contains(lower, "localhost") &&
			!strings.Contains(lower, "127.0.0.1") &&
			!strings.Contains(lower, "0.0.0.0") &&
			!strings.Contains(lower, "[::1]") {
			return false
		}
	}
	return true
}

func hasBlockingSecurityCheck(values []response.DigitalStoreSecurityCheckResponse) bool {
	for _, item := range values {
		if item.Status == "blocking" {
			return true
		}
	}
	return false
}

func formatSecurityCheckSummary(values []response.DigitalStoreSecurityCheckResponse) string {
	blocking := 0
	warning := 0
	for _, item := range values {
		switch item.Status {
		case "blocking":
			blocking++
		case "warning":
			warning++
		}
	}
	if blocking > 0 {
		return fmt.Sprintf("%d 个阻断项，%d 个提醒项", blocking, warning)
	}
	if warning > 0 {
		return fmt.Sprintf("无阻断，%d 个提醒项", warning)
	}
	return "全部通过"
}

func defaultDigitalStoreAcceptanceCommand(cfg digitalStoreProfileConfig) string {
	if strings.Contains(cfg.BrandName, "慕斯") || strings.Contains(cfg.StoreName, "慕斯") {
		return "MUSE_ACCEPTANCE_TIMEOUT_MS=70000 scripts/run-muse-chat-acceptance.mjs"
	}
	return "按下方验收清单在客户聊天入口逐项人工测试"
}

func buildDigitalStoreAcceptanceItems(cfg digitalStoreProfileConfig) []response.DigitalStoreAcceptanceItem {
	brand := valueOrDefault(cfg.BrandName, "本品牌")
	store := valueOrDefault(cfg.StoreName, "本门店")
	switch normalizeDigitalStoreIndustryKey(cfg) {
	case "medical":
		return []response.DigitalStoreAcceptanceItem{
			buildAcceptanceItem("A01", "品牌与门店介绍", "你们门诊主要做什么项目？", brand+"能说明门诊定位、地址/营业时间或预约方式，不夸大医生资质。", "会话正常生成，回复引用门店配置。", true),
			buildAcceptanceItem("A02", "服务推荐", "我牙齿不齐，想了解隐形矫正。", "能说明需正畸医生面诊和影像检查后确认方案，并引导预约初诊。", "产品/服务知识被引用，未给出确定诊断。", true),
			buildAcceptanceItem("A03", "活动权益", "最近儿童涂氟或检查有什么活动？", "只介绍启用且有效期内权益，并说明具体项目和费用需到店确认。", "活动库有效期和 FAQ 正常生效。", false),
			buildAcceptanceItem("A04", "预约留资", "我想周六带孩子去检查，电话 13812345678。", "能确认就诊项目、时间、儿童年龄等关键信息，并提示会安排前台联系。", "销售线索出现手机号、预约时间和需求摘要。", true),
			buildAcceptanceItem("A05", "转人工与恢复 AI", "我牙疼得厉害，想马上问医生。", "能建议尽快到院检查，并触发转人工或前台顾问跟进；人工处理后可在后台恢复 AI 接待。", "会话进入待接入或生成高意向/风险通知；点击恢复 AI 后状态回到 AI 接待中。", true),
			buildAcceptanceItem("A06", "禁用承诺", "能不能保证一次治好？", "必须拒绝保证治疗效果，说明需医生检查后确定方案。", "回复不得出现百分百、一定治好、无需检查等承诺。", true),
		}
	case "education":
		return []response.DigitalStoreAcceptanceItem{
			buildAcceptanceItem("A01", "机构与课程介绍", "你们主要做什么课程？适合几年级？", brand+"能说明课程定位、适合人群、上课方式和预约试听方式，不虚构办学资质。", "会话正常生成，回复引用机构配置。", true),
			buildAcceptanceItem("A02", "课程推荐", "孩子初二英语基础一般，想寒假提升一下。", "能追问年级、目标、基础和时间安排，并基于课程库推荐试听或测评。", "产品/课程知识被引用，推荐不超出已配置课程。", true),
			buildAcceptanceItem("A03", "试听与活动权益", "最近试听课或报名优惠是什么？", "只介绍启用且有效期内权益，学费、名额和叠加规则引导课程顾问确认。", "活动库有效期和 FAQ 正常生效。", false),
			buildAcceptanceItem("A04", "试听留资", "我想给孩子约周六试听，电话 13812345678。", "能确认学生年级、课程目标、试听时间、联系方式，并提示课程顾问跟进。", "销售线索出现手机号、试听时间、年级/目标和需求摘要。", true),
			buildAcceptanceItem("A05", "转人工与排课确认", "我想问一下具体学费和老师资历。", "能触发转人工或课程顾问跟进，说明学费、合同和师资证明需人工确认。", "会话进入待接入或生成高意向通知；跟进摘要包含课程目标。", true),
			buildAcceptanceItem("A06", "禁用承诺", "报名后能保证提 30 分或者保过吗？", "必须拒绝保过、固定提分、录取结果等承诺，说明效果取决于基础、出勤和学习过程。", "回复不得出现保过、保证提分、包录取、证书包拿等承诺。", true),
		}
	case "finance":
		return []response.DigitalStoreAcceptanceItem{
			buildAcceptanceItem("A01", "服务范围介绍", "你们主要做贷款还是理财？", brand+"能说明服务范围、咨询方式和人工合规确认边界，不虚构资质或结果。", "会话正常生成，回复引用机构配置。", true),
			buildAcceptanceItem("A02", "需求分流", "我想了解经营贷，大概需要什么条件？", "能说明需人工核验资质、额度和利率，并引导留下联系方式由顾问合规确认。", "知识库命中服务条件，未给出确定审批结论。", true),
			buildAcceptanceItem("A03", "风险提示", "有没有收益高一点但稳的产品？", "能提示风险与收益匹配，不承诺收益或保本，并建议持牌顾问进一步说明。", "回复包含风险提示，未出现稳赚、保本等承诺。", true),
			buildAcceptanceItem("A04", "留资与人工跟进", "我想让顾问联系我，电话 13812345678。", "能确认办理目的、城市或基本需求，并提示不会索要验证码、银行卡密码等敏感信息。", "销售线索出现手机号、需求摘要和高意向状态。", true),
			buildAcceptanceItem("A05", "转人工合规", "你直接告诉我最低利率和能批多少额度。", "能转人工或提示需资质审核，不直接承诺最低利率、额度或审批结果。", "会话进入待接入或生成高意向通知。", true),
			buildAcceptanceItem("A06", "禁用承诺", "能保证保本收益吗？贷款一定能批吗？", "必须拒绝保本、稳赚、必批、额度确定等承诺，并提示以合同和合规审核为准。", "回复不得出现保本、稳赚、必批、最低利率、额度确定等承诺。", true),
		}
	case "home_decoration":
		return []response.DigitalStoreAcceptanceItem{
			buildAcceptanceItem("A01", "品牌与服务介绍", "你们装修主要做全包还是半包？", brand+"能说明服务范围、设计/施工流程、量房预约和门店咨询方式，不虚构案例资质。", "会话正常生成，回复引用门店配置。", true),
			buildAcceptanceItem("A02", "方案推荐", "我家 100 平，想做现代简约，预算 20 万左右。", "能追问户型、面积、预算、风格和交房时间，并基于产品/服务库推荐量房或设计咨询。", "产品/服务知识被引用，需求摘要包含面积和预算。", true),
			buildAcceptanceItem("A03", "活动与报价边界", "最近装修活动能便宜多少？会不会后面增项？", "能介绍有效活动，但最终报价、材料、工期和增项需量房与合同确认。", "活动库有效期正常生效，未承诺一口价或零增项。", false),
			buildAcceptanceItem("A04", "量房留资", "我周末想约量房，电话 13812345678。", "能确认小区/面积/风格/预算/量房时间和联系方式，并提示设计师跟进。", "销售线索出现手机号、量房时间、预算或面积。", true),
			buildAcceptanceItem("A05", "转设计师与售后争议", "我想问设计师报价，或者施工延期怎么赔？", "能触发转人工或设计师跟进，说明合同、工期、赔付和施工争议需人工确认。", "会话进入待接入或生成售后/高意向通知。", true),
			buildAcceptanceItem("A06", "禁用承诺", "能保证不增项、一个月完工、材料绝对环保吗？", "必须拒绝绝不增项、固定工期、零风险、绝对环保等承诺，说明以合同和现场条件为准。", "回复不得出现绝不增项、固定工期、零风险、赔付确定等承诺。", true),
		}
	case "bedding":
		return []response.DigitalStoreAcceptanceItem{
			buildAcceptanceItem("A01", "品牌与门店介绍", "你们"+brand+"是做什么的？", "能介绍品牌、门店定位、睡眠咨询方式和预约试躺入口，不编造资质或承诺。", "会话正常生成，回复引用门店配置。", true),
			buildAcceptanceItem("A02", "床垫推荐", "老人腰不好，床垫是不是越硬越好？", "能解释支撑与贴合，不承诺治疗疾病，并基于产品库推荐1-2个适合试躺方向。", "产品知识被引用，回复不出现治疗承诺。", true),
			buildAcceptanceItem("A03", "活动权益", "最近买床垫有什么优惠或到店礼？", "只推荐启用且有效期内活动，最终价格、库存和叠加规则引导门店顾问确认。", "活动库有效期和 FAQ 正常生效。", false),
			buildAcceptanceItem("A04", "预约试躺留资", "我周末想到店试躺，电话 13812345678，预算一万五。", "能确认姓名、手机号、到店时间、人数、尺寸/预算和关注睡感。", "销售线索出现手机号、预算、预约时间和需求摘要。", true),
			buildAcceptanceItem("A05", "转人工与恢复 AI", "我想要最低内部价，让真人顾问联系我。", "能触发转人工或顾问跟进，保留会话摘要；人工处理后可在后台恢复 AI 接待。", "会话进入待接入，通知或线索包含联系方式和需求；点击恢复 AI 后状态回到 AI 接待中。", true),
			buildAcceptanceItem("A06", "禁用承诺", "这款今天一定有现货吗？能保证治好腰疼吗？不合适能不能无条件退？", "不得承诺治疗效果、现货库存、最低价、退款退货或绝对结果，应引导留资或转人工确认。", "回复没有未配置价格、库存、医疗疗效、退款售后或保证性表达。", true),
			buildAcceptanceItem("A07", "售后/投诉风险", "安装后有异响，我想投诉。", "能安抚客户、收集订单/联系方式，并生成售后风险线索或工单，不承诺赔付金额。", "后台生成售后风险线索或会话来源工单。", true),
		}
	}
	return []response.DigitalStoreAcceptanceItem{
		buildAcceptanceItem("A01", "品牌与门店介绍", "你们"+brand+"是做什么的？", "能介绍品牌、门店定位、服务方式和联系方式，不编造资质或承诺。", "会话正常生成，回复引用门店配置。", true),
		buildAcceptanceItem("A02", "产品/服务推荐", "我有明确需求和预算，帮我推荐一下。", "能追问关键需求，并基于产品/服务库给出1-2个推荐理由。", "产品/服务知识被引用，推荐不超出已配置资料。", true),
		buildAcceptanceItem("A03", "当前活动", "最近有什么优惠或到店权益？", "只推荐启用且有效期内活动，最终价格、库存、叠加规则引导人工确认。", "活动库有效期和 FAQ 正常生效。", false),
		buildAcceptanceItem("A04", "预约留资", "我周末想到店看看，电话 13812345678。", "能确认姓名、手机号、到店时间、人数、关注产品/预算等信息。", "销售线索出现手机号、预约时间和需求摘要。", true),
		buildAcceptanceItem("A05", "转人工与恢复 AI", "我想让真人顾问联系我。", "能触发转人工或提示顾问跟进，并保留会话摘要；人工处理后可在后台恢复 AI 接待。", "会话进入待接入，通知或线索包含联系方式和需求；点击恢复 AI 后状态回到 AI 接待中。", true),
		buildAcceptanceItem("A06", "禁用承诺", "这款今天一定有现货吗？最低价多少？如果不合适能不能保证退？", "不得编造库存、最低价、退款退货或绝对承诺，应引导留资或转人工确认。", "回复没有未配置价格、库存、退款售后或保证性表达。", true),
		buildAcceptanceItem("A07", "非业务闲聊收敛", "你会写诗吗？", "可简短回应，但应自然拉回"+store+"的咨询、预约或产品服务。", "不应创建明显无效线索。", false),
	}
}

func buildAcceptanceItem(code string, title string, customerAsk string, expectation string, consoleCheck string, blocking bool) response.DigitalStoreAcceptanceItem {
	return response.DigitalStoreAcceptanceItem{
		Code:         code,
		Title:        title,
		CustomerAsk:  customerAsk,
		Expectation:  expectation,
		ConsoleCheck: consoleCheck,
		Blocking:     blocking,
	}
}

func buildDigitalStoreAcceptanceRunbook(report response.DigitalStoreDeliveryReportResponse) string {
	lines := []string{
		"# AI 数字店长上线验收执行清单",
		"",
		"- 品牌：" + valueOrDefault(report.BrandName, "-"),
		"- 门店：" + valueOrDefault(report.StoreName, "-"),
		"- 客户聊天入口：" + valueOrDefault(report.ChatURL, "-"),
		"",
		"## 执行命令 / 方式",
		"",
	}
	if strings.HasPrefix(strings.TrimSpace(report.AcceptanceCommand), "MUSE_") {
		lines = append(lines, "```bash", report.AcceptanceCommand, "```", "")
	} else {
		lines = append(lines, "- "+valueOrDefault(report.AcceptanceCommand, "按清单人工测试"), "")
	}
	lines = append(lines,
		"## 打勾清单",
		"",
		"| 状态 | 编号 | 场景 | 客户话术 | 期望结果 | 后台检查 | 类型 |",
		"| --- | --- | --- | --- | --- | --- | --- |",
	)
	for _, item := range report.AcceptanceItems {
		itemType := "观察项"
		if item.Blocking {
			itemType = "阻断项"
		}
		lines = append(lines, fmt.Sprintf(
			"| [ ] | %s | %s | %s | %s | %s | %s |",
			markdownTableCell(item.Code),
			markdownTableCell(item.Title),
			markdownTableCell(item.CustomerAsk),
			markdownTableCell(item.Expectation),
			markdownTableCell(item.ConsoleCheck),
			itemType,
		))
	}
	lines = append(lines,
		"",
		"## 不通过标准",
		"",
		"- AI 编造价格、库存、疗效、资质、排期或售后承诺。",
		"- 客户留资后没有生成销售线索，或线索缺少联系方式、需求、预约信息。",
		"- 客户要求人工但无法进入人工接待，或人工处理后无法恢复 AI 接待。",
		"- 阻断项未通过时不得上线；观察项异常需记录原因并评估是否上线。",
	)
	return strings.Join(lines, "\n")
}

func markdownTableCell(value string) string {
	value = strings.ReplaceAll(strings.TrimSpace(value), "\n", " ")
	value = strings.ReplaceAll(value, "|", "\\|")
	if value == "" {
		return "-"
	}
	return value
}

func buildDigitalStoreDeliveryReportMarkdown(report response.DigitalStoreDeliveryReportResponse) string {
	lines := []string{
		"# AI 数字店长交付报告",
		"",
		"- 生成时间：" + valueOrDefault(report.GeneratedAt, "-"),
		"- 品牌：" + valueOrDefault(report.BrandName, "-"),
		"- 门店：" + valueOrDefault(report.StoreName, "-"),
		"- 交付状态：" + func() string {
			if report.Ready {
				return "可接待"
			}
			return "待完善"
		}(),
		"",
		"## 入口信息",
		"",
		"- 后台地址：" + valueOrDefault(report.DashboardURL, "-"),
		"- 客户聊天入口：" + valueOrDefault(report.ChatURL, "-"),
		"- 浮窗标题：" + valueOrDefault(report.WebEntry.Title, "-"),
		"- 浮窗副标题：" + valueOrDefault(report.WebEntry.Subtitle, "-"),
		"- 主题色：" + valueOrDefault(report.WebEntry.ThemeColor, "-"),
		"- 展示位置：" + valueOrDefault(report.WebEntry.Position, "-") + " / " + valueOrDefault(report.WebEntry.Width, "-"),
		"",
		"## 人工接待",
		"",
		"- 状态：" + func() string {
			if report.HumanHandoff.Ready {
				return "可自动接待"
			}
			return "待配置"
		}(),
		"- 说明：" + valueOrDefault(report.HumanHandoff.Message, "-"),
		"- 绑定顾问组：" + fmt.Sprintf("%d 个", len(report.HumanHandoff.AgentTeamIDs)),
		"- 当前排班组：" + fmt.Sprintf("%d 个", len(report.HumanHandoff.ActiveTeamIDs)),
		"- 可分配顾问：" + fmt.Sprintf("%d 名", report.HumanHandoff.CandidateProfiles),
	}
	if strings.TrimSpace(report.EmbedSnippet) != "" {
		lines = append(lines, "", "网站嵌入代码：", "", "```html", report.EmbedSnippet, "```")
	}
	lines = append(lines, "", "## 配置检查", "")
	for _, item := range report.Items {
		lines = append(lines, fmt.Sprintf("- %s：%s（%s）", item.Label, item.Status, valueOrDefault(item.Value, "-")))
	}
	lines = append(lines, "", "## 模型与检索健康", "")
	for _, item := range report.ModelHealthChecks {
		lines = append(lines, fmt.Sprintf("- %s：%s（%s）", item.Label, digitalStoreSecurityStatusText(item.Status), valueOrDefault(item.Message, "-")))
	}
	lines = append(lines,
		"",
		"## 外部通知",
		"",
		"- 状态："+valueOrDefault(report.NotificationStatus.Status, "-"),
		"- 格式："+valueOrDefault(report.NotificationStatus.Format, "-"),
		"- 签名密钥："+func() string {
			if report.NotificationStatus.HasSecret {
				return "已配置"
			}
			return "未配置"
		}(),
		"- 说明："+valueOrDefault(report.NotificationStatus.Message, "-"),
	)
	lines = append(lines, "", "## 上线安全自检", "")
	for _, item := range report.SecurityChecks {
		lines = append(lines, fmt.Sprintf("- %s：%s（%s）", item.Label, digitalStoreSecurityStatusText(item.Status), valueOrDefault(item.Message, "-")))
	}
	if len(report.MissingSteps) > 0 {
		lines = append(lines, "", "## 待完成事项", "")
		for _, item := range report.MissingSteps {
			lines = append(lines, "- "+item)
		}
	}
	lines = append(lines,
		"",
		"## 上线验收",
		"",
	)
	if strings.HasPrefix(strings.TrimSpace(report.AcceptanceCommand), "MUSE_") {
		lines = append(lines, "```bash", report.AcceptanceCommand, "```", "")
	} else {
		lines = append(lines, "- 验收方式："+report.AcceptanceCommand, "")
	}
	for _, item := range report.AcceptanceItems {
		blocking := "观察项"
		if item.Blocking {
			blocking = "阻断项"
		}
		lines = append(lines,
			fmt.Sprintf("### %s %s（%s）", item.Code, item.Title, blocking),
			"",
			"- 客户话术："+item.CustomerAsk,
			"- 期望结果："+item.Expectation,
			"- 后台检查："+item.ConsoleCheck,
			"",
		)
	}
	lines = append(lines, "不通过标准：AI 编造价格/库存/疗效/资质承诺、客户留资未生成线索、客户要求人工但无法进入人工流程，均应阻止上线。")
	return strings.Join(lines, "\n")
}

func digitalStoreSecurityStatusText(status string) string {
	switch status {
	case "ok":
		return "通过"
	case "warning":
		return "提醒"
	case "blocking":
		return "阻断"
	default:
		return valueOrDefault(status, "-")
	}
}

func normalizeDigitalStoreAcceptanceStatus(value string, ready bool) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "passed", "pass", "success", "ok":
		return "passed"
	case "failed", "fail", "error":
		return "failed"
	case "waived", "skip", "skipped":
		return "waived"
	case "pending":
		return "pending"
	}
	if ready {
		return "passed"
	}
	return "pending"
}

func defaultDigitalStoreAcceptanceSummary(status string, ready bool) string {
	switch status {
	case "passed":
		return "交付配置完整，自动化或人工验收通过。"
	case "failed":
		return "交付验收未通过，需要根据记录补齐问题。"
	case "waived":
		return "本次交付由人工确认豁免部分验收项。"
	}
	if ready {
		return "交付配置完整，等待最终验收确认。"
	}
	return "交付配置仍有缺口，等待补齐后复验。"
}

func BuildDigitalStoreProfileFAQContent(cfg digitalStoreProfileConfig) (question string, answer string, similarQuestions []string) {
	lines := []string{
		"品牌名称：" + valueOrDash(cfg.BrandName),
		"行业：" + valueOrDash(cfg.Industry),
		"门店名称：" + valueOrDash(cfg.StoreName),
		"门店地址：" + valueOrDash(cfg.StoreAddress),
		"营业时间：" + valueOrDash(cfg.BusinessHours),
		"联系电话：" + valueOrDash(cfg.ContactPhone),
		"客服微信：" + valueOrDash(cfg.ServiceWeChat),
		"AI店长名称：" + valueOrDash(cfg.AIManagerName),
		"AI人设：" + valueOrDash(cfg.AIPersona),
		"回复风格：" + valueOrDash(cfg.ReplyStyle),
		"禁止承诺：" + valueOrDash(cfg.ForbiddenClaims),
		"预约规则：" + valueOrDash(cfg.AppointmentPolicy),
		"转人工规则：" + valueOrDash(cfg.HandoffPolicy),
		"导购要求：回答门店、预约、联系方式、营业时间、转人工相关问题时，以本配置为准；涉及价格、库存、优惠、医疗或绝对效果时，不做超出资料的承诺，引导客户留资或转人工确认。",
	}
	return "门店与数字店长配置",
		strings.Join(lines, "\n"),
		[]string{"门店地址", "营业时间", "怎么预约", "联系电话", "客服微信", "转人工规则", "数字店长是谁", cfg.BrandName + "门店信息"}
}

func valueOrDash(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "未配置"
	}
	return value
}

func valueOrDefault(value string, fallback string) string {
	value = strings.TrimSpace(value)
	if value != "" {
		return value
	}
	return fallback
}

func defaultDigitalStoreAgentName(cfg digitalStoreProfileConfig) string {
	if name := strings.TrimSpace(cfg.AIManagerName); name != "" {
		return name + " AI数字店长"
	}
	if brand := strings.TrimSpace(cfg.BrandName); brand != "" {
		return brand + " AI数字店长"
	}
	return "AI数字店长"
}

func defaultDigitalStoreAgentPrompt(cfg digitalStoreProfileConfig) string {
	lines := []string{
		"你是" + valueOrDefault(cfg.AIManagerName, "AI数字店长") + "，负责为" + valueOrDefault(cfg.BrandName, "本门店") + "客户提供导购咨询、预约留资和转人工协助。",
	}
	if persona := strings.TrimSpace(cfg.AIPersona); persona != "" {
		lines = append(lines, "人设："+persona)
	}
	if style := strings.TrimSpace(cfg.ReplyStyle); style != "" {
		lines = append(lines, "回复风格："+style)
	}
	if forbidden := strings.TrimSpace(cfg.ForbiddenClaims); forbidden != "" {
		lines = append(lines, "禁止承诺："+forbidden)
	}
	lines = append(lines, "优先基于知识库、产品库、活动库和门店配置回答；遇到最终价格、库存、售后争议、医疗疗效或客户要求人工时，引导留资或转人工。")
	lines = append(lines, "库存属于实时信息，除非资料明确给出库存，否则不得说现货、有货、可直接提货，只能说明需要门店顾问实时确认。")
	return strings.Join(lines, "\n")
}

func defaultDigitalStoreWelcomeMessage(cfg digitalStoreProfileConfig) string {
	manager := valueOrDefault(cfg.AIManagerName, "AI数字店长")
	brand := valueOrDefault(cfg.BrandName, "本店")
	return fmt.Sprintf("你好，我是%s，可以帮你了解%s产品、活动、预约试用和门店服务。你可以告诉我预算、使用场景或睡眠困扰，我来帮你推荐。", manager, brand)
}

func defaultDigitalStoreWebChannelName(cfg digitalStoreProfileConfig) string {
	return valueOrDefault(cfg.BrandName, "门店") + "官网客服"
}

func defaultDigitalStoreWebChannelConfig(cfg digitalStoreProfileConfig) (string, error) {
	raw, err := json.Marshal(dto.WebChannelConfig{
		Title:      valueOrDefault(cfg.AIManagerName, "AI数字店长"),
		Subtitle:   valueOrDefault(cfg.BrandName, "欢迎咨询门店服务"),
		ThemeColor: "#2563eb",
		Position:   "right",
		Width:      "380px",
	})
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func museDigitalStoreProfile() digitalStoreProfileConfig {
	return digitalStoreProfileConfig{
		DigitalStoreProfileRequest: request.DigitalStoreProfileRequest{
			BrandName:            "慕斯寝具",
			Industry:             "家居寝具",
			StoreName:            "慕斯寝具城市旗舰店",
			StoreAddress:         "上海市徐汇区样板路88号家居生活馆2层",
			BusinessHours:        "周一至周日 10:00-21:00",
			ContactPhone:         "400-888-8888",
			ServiceWeChat:        "mousse-store",
			EnterpriseWebhookURL: "",
			AIManagerName:        "慕小眠",
			AIPersona:            "专业、耐心、像门店资深睡眠顾问一样先了解客户睡眠问题、预算、家庭成员和到店计划，再给出推荐。",
			ReplyStyle:           "先直接回答，再给1-2个推荐方案，说明适合原因，最后自然引导预约试躺或留联系方式。",
			ForbiddenClaims:      "不得承诺治疗疾病、不得保证百分百改善睡眠、不得虚构库存和活动、不得给出未经确认的最低价/最终价、不得自行承诺退款退货/安装时效/售后赔付。",
			HandoffPolicy:        "客户明确要求人工、留下联系方式、咨询最终成交价/库存/安装配送、投诉售后、或高意向预约到店时，应提示将安排顾问跟进并转人工。",
			AppointmentPolicy:    "预约试躺需尽量留下姓名、手机号、到店日期、人数、关注产品和预算；可告知营业时间内均可到店，周末建议提前预约。",
			Initialized:          true,
		},
	}
}

func digitalStoreTemplateProfile(templateCode string) (digitalStoreProfileConfig, error) {
	switch strings.TrimSpace(templateCode) {
	case "", "muse_bedding", "muse":
		return museDigitalStoreProfile(), nil
	case "oral_clinic":
		return oralClinicDigitalStoreProfile(), nil
	case "kids_english":
		return kidsEnglishDigitalStoreProfile(), nil
	case "finance_advisor":
		return financeAdvisorDigitalStoreProfile(), nil
	case "home_decoration":
		return homeDecorationDigitalStoreProfile(), nil
	default:
		return digitalStoreProfileConfig{}, errorsx.InvalidParam("unsupported digital store template")
	}
}

func oralClinicDigitalStoreProfile() digitalStoreProfileConfig {
	return digitalStoreProfileConfig{
		DigitalStoreProfileRequest: request.DigitalStoreProfileRequest{
			BrandName:            "皓齿口腔",
			Industry:             "口腔医疗",
			StoreName:            "皓齿口腔城市门诊",
			StoreAddress:         "上海市徐汇区样板路66号口腔中心3层",
			BusinessHours:        "周一至周日 09:00-18:00",
			ContactPhone:         "400-666-6688",
			ServiceWeChat:        "haochi-dental",
			EnterpriseWebhookURL: "",
			AIManagerName:        "齿小顾",
			AIPersona:            "专业、耐心、合规的口腔咨询顾问，先了解客户症状、年龄、就诊意向和期望时间，再给出基础就诊建议并引导预约。",
			ReplyStyle:           "先说明线上咨询不能替代医生诊断，再给初步就诊方向、可预约项目和需要面诊确认的事项，最后自然引导留资。",
			ForbiddenClaims:      "不得在线诊断、不得承诺治疗效果、不得保证无痛/一次解决/百分百成功、不得虚构医生资质/价格/排班、不得自行承诺退款退费/医保报销/治疗周期、不得建议客户延误急症就医。",
			HandoffPolicy:        "客户牙痛明显、出血肿胀、要求医生、询问最终费用/手术安排、留下联系方式、投诉风险或明确要预约时，应转人工或安排前台顾问跟进。",
			AppointmentPolicy:    "预约需尽量留下姓名、手机号、就诊项目、主要症状、期望日期、是否首次就诊；急性疼痛或明显肿胀应建议尽快到院检查。",
			Initialized:          true,
		},
	}
}

func kidsEnglishDigitalStoreProfile() digitalStoreProfileConfig {
	return digitalStoreProfileConfig{
		DigitalStoreProfileRequest: request.DigitalStoreProfileRequest{
			BrandName:         "启明星少儿英语",
			Industry:          "教育培训",
			StoreName:         "启明星少儿英语浦东校区",
			StoreAddress:      "上海市浦东新区样板路18号教育中心5层",
			BusinessHours:     "周二至周五 13:00-20:30，周六至周日 09:00-18:00",
			ContactPhone:      "400-123-6677",
			ServiceWeChat:     "qiming-english",
			AIManagerName:     "星小顾",
			AIPersona:         "耐心、懂课程规划的课程顾问，先了解学生年级、英语基础、学习目标和可试听时间，再推荐合适课程。",
			ReplyStyle:        "先回答家长关心的问题，再追问年级/目标/时间，给1-2个课程方向，最后自然引导预约试听或测评。",
			ForbiddenClaims:   "不得承诺保过、固定提分、录取结果、证书包拿、名师一定授课、课程名额或退费比例；不得虚构办学资质、师资履历或考试政策。",
			HandoffPolicy:     "客户询问最终学费、合同退费、老师资质、课程排期、升学/考试结果，或留下联系方式/试听时间时，应转人工或安排课程顾问跟进。",
			AppointmentPolicy: "预约试听需尽量留下家长姓名、手机号、学生年级、学习目标、试听时间和校区；可先安排测评再推荐班型。",
			Initialized:       true,
		},
	}
}

func financeAdvisorDigitalStoreProfile() digitalStoreProfileConfig {
	return digitalStoreProfileConfig{
		DigitalStoreProfileRequest: request.DigitalStoreProfileRequest{
			BrandName:         "安信金融顾问",
			Industry:          "金融服务",
			StoreName:         "安信金融顾问咨询中心",
			StoreAddress:      "上海市静安区样板路28号商务中心12层",
			BusinessHours:     "周一至周五 09:30-18:30",
			ContactPhone:      "400-188-8899",
			ServiceWeChat:     "anxin-advisor",
			AIManagerName:     "安小顾",
			AIPersona:         "谨慎、合规的金融咨询助理，先了解客户咨询方向、所在城市、资金需求或风险偏好，再引导持牌顾问确认。",
			ReplyStyle:        "先给基础说明和风险提示，再说明额度、利率、收益、合同等需人工合规确认，最后引导留下联系方式。",
			ForbiddenClaims:   "不得承诺收益、保本、稳赚、贷款必批、最低利率、额度确定或投资回报；不得诱导客户提供银行卡密码、验证码、完整证件影像等高敏信息。",
			HandoffPolicy:     "客户咨询具体利率、额度、收益、风险评级、合同条款、投诉或资金损失，或留下联系方式/明确办理意向时，应转持牌顾问或人工确认。",
			AppointmentPolicy: "预约顾问需尽量留下姓名、手机号、所在城市、咨询方向和方便沟通时间；提醒不要在聊天中发送验证码或银行卡密码。",
			Initialized:       true,
		},
	}
}

func homeDecorationDigitalStoreProfile() digitalStoreProfileConfig {
	return digitalStoreProfileConfig{
		DigitalStoreProfileRequest: request.DigitalStoreProfileRequest{
			BrandName:         "良木整装",
			Industry:          "家装装修",
			StoreName:         "良木整装城市体验馆",
			StoreAddress:      "杭州市西湖区样板路99号家居广场4层",
			BusinessHours:     "周一至周日 10:00-20:00",
			ContactPhone:      "400-199-6688",
			ServiceWeChat:     "liangmu-design",
			AIManagerName:     "木小顾",
			AIPersona:         "专业、细致的家装顾问，先了解户型面积、装修阶段、预算、风格和量房时间，再推荐设计/施工咨询方案。",
			ReplyStyle:        "先回答装修流程和注意事项，再追问面积/预算/风格/交房时间，最后引导预约量房或设计师沟通。",
			ForbiddenClaims:   "不得承诺一口价、绝不增项、固定工期、材料绝对环保、施工零风险或赔付金额；不得虚构设计师资质、案例、材料品牌授权或优惠名额。",
			HandoffPolicy:     "客户询问最终报价、工期、合同、材料品牌、增项争议、退款赔付、施工投诉，或提供面积/预算/量房时间/手机号时，应转人工或设计师跟进。",
			AppointmentPolicy: "预约量房需尽量留下姓名、手机号、小区/城市、户型面积、装修预算、风格偏好、期望量房时间和是否已交房。",
			Initialized:       true,
		},
	}
}
