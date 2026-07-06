package services

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"agent-desk/internal/models"
	"agent-desk/internal/pkg/config"
	"agent-desk/internal/pkg/constants"
	"agent-desk/internal/pkg/dto"
	"agent-desk/internal/pkg/dto/request"
	"agent-desk/internal/pkg/enums"
	"agent-desk/internal/pkg/utils"

	"github.com/glebarez/sqlite"
	"github.com/mlogclub/simple/sqls"
	"gorm.io/gorm"
)

func TestBuildDigitalStoreProfileFAQContent(t *testing.T) {
	cfg := digitalStoreProfileConfig{
		DigitalStoreProfileRequest: request.DigitalStoreProfileRequest{
			BrandName:         "慕斯寝具",
			StoreName:         "慕斯旗舰店",
			StoreAddress:      "上海市徐汇区样板路88号",
			BusinessHours:     "10:00-21:00",
			ContactPhone:      "400-888-8888",
			AIManagerName:     "慕小眠",
			AppointmentPolicy: "留下姓名和手机号预约试躺",
			HandoffPolicy:     "客户留资后转人工",
		},
	}
	question, answer, similar := BuildDigitalStoreProfileFAQContent(cfg)
	if question != "门店与数字店长配置" {
		t.Fatalf("unexpected question: %s", question)
	}
	for _, want := range []string{"品牌名称：慕斯寝具", "门店地址：上海市徐汇区样板路88号", "营业时间：10:00-21:00", "转人工规则：客户留资后转人工"} {
		if !strings.Contains(answer, want) {
			t.Fatalf("answer missing %q: %s", want, answer)
		}
	}
	if len(similar) == 0 {
		t.Fatal("similar questions should not be empty")
	}
}

func TestDigitalStoreTemplatesIncludeIndustryMarketTemplates(t *testing.T) {
	templates := DigitalStoreProfileService.ListTemplates()
	if len(templates) < 5 {
		t.Fatalf("expected industry digital store templates, got %d", len(templates))
	}
	expected := map[string]string{
		"muse_bedding":    "家居寝具",
		"oral_clinic":     "口腔医疗",
		"kids_english":    "教育培训",
		"finance_advisor": "金融服务",
		"home_decoration": "家装装修",
	}
	for _, item := range templates {
		wantIndustry, ok := expected[item.Code]
		if !ok {
			continue
		}
		delete(expected, item.Code)
		if item.Industry != wantIndustry {
			t.Fatalf("unexpected industry for %s: %s", item.Code, item.Industry)
		}
		if item.Version == "" || item.Description == "" {
			t.Fatalf("template should expose version and description: %#v", item)
		}
	}
	if len(expected) > 0 {
		t.Fatalf("missing industry templates: %#v", expected)
	}
}

func TestDigitalStoreTemplateProfileSupportsOralClinic(t *testing.T) {
	cfg, err := digitalStoreTemplateProfile("oral_clinic")
	if err != nil {
		t.Fatalf("digitalStoreTemplateProfile() error = %v", err)
	}
	if cfg.BrandName != "皓齿口腔" || cfg.AIManagerName != "齿小顾" {
		t.Fatalf("unexpected oral clinic profile: %#v", cfg)
	}
	if !strings.Contains(cfg.ForbiddenClaims, "不得在线诊断") {
		t.Fatalf("oral clinic profile should include compliance boundary: %s", cfg.ForbiddenClaims)
	}
}

func TestDigitalStoreTemplateProfileSupportsIndustryMarketTemplates(t *testing.T) {
	for _, tc := range []struct {
		code       string
		brand      string
		industry   string
		forbidden  string
		handoffKey string
	}{
		{"kids_english", "启明星少儿英语", "教育培训", "不得承诺保过", "课程顾问"},
		{"finance_advisor", "安信金融顾问", "金融服务", "不得承诺收益", "持牌顾问"},
		{"home_decoration", "良木整装", "家装装修", "不得承诺一口价", "设计师"},
	} {
		t.Run(tc.code, func(t *testing.T) {
			cfg, err := digitalStoreTemplateProfile(tc.code)
			if err != nil {
				t.Fatalf("digitalStoreTemplateProfile() error = %v", err)
			}
			if cfg.BrandName != tc.brand || cfg.Industry != tc.industry {
				t.Fatalf("unexpected profile for %s: %#v", tc.code, cfg)
			}
			if !strings.Contains(cfg.ForbiddenClaims, tc.forbidden) {
				t.Fatalf("profile should include forbidden boundary %q: %s", tc.forbidden, cfg.ForbiddenClaims)
			}
			if !strings.Contains(cfg.HandoffPolicy, tc.handoffKey) {
				t.Fatalf("profile should include handoff key %q: %s", tc.handoffKey, cfg.HandoffPolicy)
			}
		})
	}
}

func TestDigitalStoreExportTemplateIncludesProfileCatalogAndAcceptance(t *testing.T) {
	got, err := DigitalStoreProfileService.ExportTemplate("muse_bedding")
	if err != nil {
		t.Fatalf("ExportTemplate() error = %v", err)
	}
	if got.SchemaVersion != "1.0" || got.Template.Code != "muse_bedding" || got.Template.Version == "" || got.Profile.BrandName != "慕斯寝具" {
		t.Fatalf("unexpected template export metadata: %#v", got)
	}
	if got.Profile.TemplateCode != got.Template.Code || got.Profile.TemplateVersion != got.Template.Version {
		t.Fatalf("exported profile should include template version: profile=%#v template=%#v", got.Profile, got.Template)
	}
	if len(got.Products) < 4 || len(got.Promotions) < 2 || len(got.AcceptanceItems) == 0 {
		t.Fatalf("template export should include catalog and acceptance items: products=%d promotions=%d acceptance=%d", len(got.Products), len(got.Promotions), len(got.AcceptanceItems))
	}
	if got.Products[0].IndustryAttributes == "" {
		t.Fatalf("template export should include product industry attributes: %#v", got.Products[0])
	}
	foundGuardrail := false
	for _, item := range got.AcceptanceItems {
		if item.Code == "A06" && strings.Contains(item.Expectation, "退款退货") {
			foundGuardrail = true
			break
		}
	}
	if !foundGuardrail {
		t.Fatalf("template export should include safety acceptance item: %#v", got.AcceptanceItems)
	}
}

func TestDigitalStoreExportTemplateSupportsOralClinic(t *testing.T) {
	got, err := DigitalStoreProfileService.ExportTemplate("oral_clinic")
	if err != nil {
		t.Fatalf("ExportTemplate() oral clinic error = %v", err)
	}
	if got.Template.Code != "oral_clinic" || got.Profile.Industry != "口腔医疗" {
		t.Fatalf("unexpected oral clinic export: %#v", got)
	}
	if !strings.Contains(got.Profile.ForbiddenClaims, "不得在线诊断") || len(got.Products) == 0 || len(got.Promotions) == 0 {
		t.Fatalf("oral clinic export should include compliance and catalog: %#v", got)
	}
	foundMedicalRule := false
	for _, item := range got.RiskRules {
		if item.Key == "medical" && strings.Contains(strings.Join(item.ForbiddenClaims, " "), "不得在线诊断") {
			foundMedicalRule = true
			break
		}
	}
	if !foundMedicalRule {
		t.Fatalf("oral clinic export should include medical risk rules: %#v", got.RiskRules)
	}
}

func TestDigitalStoreExportTemplateSupportsIndustryMarketTemplates(t *testing.T) {
	for _, tc := range []struct {
		code          string
		industry      string
		productKey    string
		promotionKey  string
		riskRuleKey   string
		forbiddenText string
	}{
		{"kids_english", "教育培训", "自然拼读", "试听", "education", "保过"},
		{"finance_advisor", "金融服务", "经营贷", "顾问", "finance", "保本"},
		{"home_decoration", "家装装修", "量房", "量房", "home_decoration", "绝不增项"},
	} {
		t.Run(tc.code, func(t *testing.T) {
			got, err := DigitalStoreProfileService.ExportTemplate(tc.code)
			if err != nil {
				t.Fatalf("ExportTemplate() error = %v", err)
			}
			if got.Template.Code != tc.code || got.Profile.Industry != tc.industry {
				t.Fatalf("unexpected export metadata: %#v", got)
			}
			foundProduct := false
			for _, item := range got.Products {
				productText := item.Name + item.Category + item.SellingPoints + item.SuitablePeople + item.UnsuitablePeople + item.Scenarios + item.Specs + item.IndustryAttributes
				if strings.Contains(productText, tc.productKey) {
					foundProduct = true
					break
				}
			}
			if !foundProduct {
				t.Fatalf("export should include product keyword %q: %#v", tc.productKey, got.Products)
			}
			foundPromotion := false
			for _, item := range got.Promotions {
				if strings.Contains(item.Name+item.Description+item.AppointmentBenefit, tc.promotionKey) {
					foundPromotion = true
					break
				}
			}
			if !foundPromotion {
				t.Fatalf("export should include promotion keyword %q: %#v", tc.promotionKey, got.Promotions)
			}
			foundRiskRule := false
			for _, item := range got.RiskRules {
				if item.Key == tc.riskRuleKey && strings.Contains(strings.Join(item.ForbiddenClaims, " "), tc.forbiddenText) {
					foundRiskRule = true
					break
				}
			}
			if !foundRiskRule {
				t.Fatalf("export should include %s risk rule with %q: %#v", tc.riskRuleKey, tc.forbiddenText, got.RiskRules)
			}
			if len(got.AcceptanceItems) == 0 {
				t.Fatalf("export should include acceptance items")
			}
		})
	}
}

func TestDigitalStoreExportTemplateRejectsUnsupportedCode(t *testing.T) {
	if _, err := DigitalStoreProfileService.ExportTemplate("unknown_template"); err == nil {
		t.Fatal("expected unsupported template error")
	}
}

func TestDigitalStorePreviewTemplateReportsCreateAndUpdate(t *testing.T) {
	setupDigitalStoreRuntimeSetupTestDB(t)
	if err := sqls.DB().Create(&models.Product{
		Name:   "慕斯脊护支撑款",
		Status: enums.StatusOk,
	}).Error; err != nil {
		t.Fatalf("create existing product: %v", err)
	}
	if err := sqls.DB().Create(&models.Promotion{
		Name:   "周末预约试躺礼",
		Status: enums.StatusOk,
	}).Error; err != nil {
		t.Fatalf("create existing promotion: %v", err)
	}
	cfg := museDigitalStoreProfile()
	cfg.KnowledgeBaseID = 99
	cfg.EnterpriseWebhookURL = "https://hooks.example.com/current"
	raw, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}
	if err := sqls.DB().Create(&models.SystemConfig{
		ConfigKey:   digitalStoreProfileConfigKey,
		ConfigValue: string(raw),
		GroupCode:   digitalStoreConfigGroup,
		Status:      enums.StatusOk,
	}).Error; err != nil {
		t.Fatalf("create config: %v", err)
	}

	got, err := DigitalStoreProfileService.PreviewTemplate("muse_bedding")
	if err != nil {
		t.Fatalf("PreviewTemplate() error = %v", err)
	}
	if got.Template.Version == "" || got.Profile.TemplateCode != got.Template.Code || got.Profile.TemplateVersion != got.Template.Version {
		t.Fatalf("preview should include target template version: %#v", got)
	}
	if len(got.RiskRules) == 0 {
		t.Fatalf("preview should include industry risk rules: %#v", got)
	}
	if got.ProfileAction != "update" || got.ProductUpdateTotal != 1 || got.ProductCreateTotal == 0 || got.PromotionUpdateTotal != 1 || got.PromotionCreateTotal == 0 {
		t.Fatalf("unexpected preview totals: %#v", got)
	}
	foundWarning := false
	for _, item := range got.Warnings {
		if item.Key == "webhook_preserved" {
			foundWarning = true
			break
		}
	}
	if !foundWarning {
		t.Fatalf("expected preserved webhook warning: %#v", got.Warnings)
	}
}

func TestDigitalStorePreviewImportedTemplateSupportsCustomIndustry(t *testing.T) {
	setupDigitalStoreRuntimeSetupTestDB(t)
	if err := sqls.DB().Create(&models.Product{
		Name:   "少儿英语体验课",
		Status: enums.StatusOk,
	}).Error; err != nil {
		t.Fatalf("create existing product: %v", err)
	}
	got, err := DigitalStoreProfileService.PreviewImportedTemplate(request.DigitalStoreTemplateImportRequest{
		SchemaVersion: "1.0",
		Template: request.DigitalStoreTemplateImportMetaRequest{
			Code:        "kids_english",
			Name:        "少儿英语培训",
			Industry:    "教育培训",
			Version:     "1.0.0",
			Description: "适合少儿英语课程顾问接待。",
		},
		Profile: request.DigitalStoreProfileRequest{
			BrandName:         "星芽英语",
			Industry:          "教育培训",
			StoreName:         "星芽英语徐汇校区",
			AIManagerName:     "星小顾",
			ForbiddenClaims:   "不得承诺保过、提分幅度、录取结果或固定老师。",
			HandoffPolicy:     "客户询问最终学费、合同、退费或留下联系方式时转人工。",
			AppointmentPolicy: "预约试听需留下手机号、学生年级、英语水平和期望时间。",
		},
		Products: []request.SaveProductRequest{
			{Name: "少儿英语体验课", Category: "体验课", SellingPoints: "适合了解孩子英语水平和课程方式。", IndustryAttributes: "课时：1节；班型：1对1/小班；年级：幼儿园-小学。", Status: int(enums.StatusOk)},
			{Name: "自然拼读进阶班", Category: "系统课", SellingPoints: "适合有基础的孩子系统学习自然拼读。", IndustryAttributes: "课时：24节；班型：小班。", Status: int(enums.StatusOk)},
		},
		Promotions: []request.SavePromotionRequest{
			{Name: "试听课预约礼", PromotionType: "预约权益", Description: "预约试听可优先安排测评时段。", Status: int(enums.StatusOk)},
		},
	})
	if err != nil {
		t.Fatalf("PreviewImportedTemplate() error = %v", err)
	}
	if got.Template.Code != "kids_english" || got.Profile.TemplateCode != "kids_english" || got.Profile.Industry != "教育培训" {
		t.Fatalf("unexpected imported template preview: %#v", got)
	}
	if got.ProductUpdateTotal != 1 || got.ProductCreateTotal != 1 || got.PromotionCreateTotal != 1 {
		t.Fatalf("unexpected imported template totals: %#v", got)
	}
	foundEducationRule := false
	for _, item := range got.RiskRules {
		if item.Key == "education" && strings.Contains(strings.Join(item.ForbiddenClaims, " "), "保过") {
			foundEducationRule = true
			break
		}
	}
	if !foundEducationRule {
		t.Fatalf("expected education risk rules: %#v", got.RiskRules)
	}
}

func TestDigitalStorePreviewImportedTemplateRejectsMissingCode(t *testing.T) {
	setupDigitalStoreRuntimeSetupTestDB(t)
	_, err := DigitalStoreProfileService.PreviewImportedTemplate(request.DigitalStoreTemplateImportRequest{
		Profile: request.DigitalStoreProfileRequest{
			BrandName: "星芽英语",
			StoreName: "星芽英语徐汇校区",
		},
		Products: []request.SaveProductRequest{
			{Name: "少儿英语体验课", Status: int(enums.StatusOk)},
		},
	})
	if err == nil {
		t.Fatal("expected missing template code error")
	}
}

func TestDigitalStoreIndustryRiskRulesMatchCommonIndustries(t *testing.T) {
	cases := []struct {
		name string
		cfg  digitalStoreProfileConfig
		want string
	}{
		{name: "bedding", cfg: museDigitalStoreProfile(), want: "bedding"},
		{name: "medical", cfg: oralClinicDigitalStoreProfile(), want: "medical"},
		{name: "education", cfg: digitalStoreProfileConfig{DigitalStoreProfileRequest: request.DigitalStoreProfileRequest{Industry: "教育培训"}}, want: "education"},
		{name: "finance", cfg: digitalStoreProfileConfig{DigitalStoreProfileRequest: request.DigitalStoreProfileRequest{Industry: "金融服务"}}, want: "finance"},
		{name: "home decoration", cfg: digitalStoreProfileConfig{DigitalStoreProfileRequest: request.DigitalStoreProfileRequest{Industry: "家装装修"}}, want: "home_decoration"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			rules := buildDigitalStoreIndustryRiskRuleResponses(tt.cfg)
			found := false
			for _, item := range rules {
				if item.Key == tt.want {
					found = len(item.ForbiddenClaims) > 0 && len(item.HandoffTriggers) > 0
					break
				}
			}
			if !found {
				t.Fatalf("expected risk rule %s in %#v", tt.want, rules)
			}
		})
	}
}

func TestDigitalStoreKnowledgeAssistantDetectsCoveredAndMissingFAQ(t *testing.T) {
	setupDigitalStoreRuntimeSetupTestDB(t)
	cfg := museDigitalStoreProfile()
	cfg.KnowledgeBaseID = 88
	raw, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}
	if err := sqls.DB().Create(&models.SystemConfig{
		ConfigKey:   digitalStoreProfileConfigKey,
		ConfigValue: string(raw),
		GroupCode:   digitalStoreConfigGroup,
		Status:      enums.StatusOk,
	}).Error; err != nil {
		t.Fatalf("create config: %v", err)
	}
	if err := sqls.DB().Create(&models.KnowledgeFAQ{
		KnowledgeBaseID: 88,
		Question:        "门店地址、营业时间和联系方式是什么？",
		Answer:          "门店地址在上海，营业时间 10:00-21:00，可电话或微信联系。",
		Status:          enums.StatusOk,
	}).Error; err != nil {
		t.Fatalf("create faq: %v", err)
	}

	got := DigitalStoreProfileService.GetKnowledgeAssistant()
	if got.KnowledgeBaseID != 88 || got.CoveredTotal == 0 || got.MissingTotal == 0 {
		t.Fatalf("unexpected knowledge assistant summary: %#v", got)
	}
	foundCovered := false
	for _, item := range got.Items {
		if item.Key == "store_basic" && item.Covered && item.MatchedFAQID > 0 {
			foundCovered = true
			break
		}
	}
	if !foundCovered {
		t.Fatalf("expected store basic faq covered: %#v", got.Items)
	}
}

func TestDigitalStoreKnowledgeAssistantIncludesMedicalFAQSuggestions(t *testing.T) {
	setupDigitalStoreRuntimeSetupTestDB(t)
	cfg := oralClinicDigitalStoreProfile()
	cfg.KnowledgeBaseID = 66
	raw, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}
	if err := sqls.DB().Create(&models.SystemConfig{
		ConfigKey:   digitalStoreProfileConfigKey,
		ConfigValue: string(raw),
		GroupCode:   digitalStoreConfigGroup,
		Status:      enums.StatusOk,
	}).Error; err != nil {
		t.Fatalf("create config: %v", err)
	}

	got := DigitalStoreProfileService.GetKnowledgeAssistant()
	found := false
	for _, item := range got.Items {
		if item.Key == "medical_diagnosis_boundary" && strings.Contains(item.Question, "医生面诊") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected medical faq suggestion: %#v", got.Items)
	}
}

func TestDigitalStorePreviewTemplateWarnsSameTemplateVersion(t *testing.T) {
	setupDigitalStoreRuntimeSetupTestDB(t)
	cfg := museDigitalStoreProfile()
	cfg.TemplateCode = "muse_bedding"
	cfg.TemplateVersion = "1.0.0"
	cfg.TemplateAppliedAt = "2026-01-01 10:00:00"
	raw, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}
	if err := sqls.DB().Create(&models.SystemConfig{
		ConfigKey:   digitalStoreProfileConfigKey,
		ConfigValue: string(raw),
		GroupCode:   digitalStoreConfigGroup,
		Status:      enums.StatusOk,
	}).Error; err != nil {
		t.Fatalf("create config: %v", err)
	}

	got, err := DigitalStoreProfileService.PreviewTemplate("muse_bedding")
	if err != nil {
		t.Fatalf("PreviewTemplate() error = %v", err)
	}
	found := false
	for _, item := range got.Warnings {
		if item.Key == "same_template_version" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected same template version warning: %#v", got.Warnings)
	}
}

func TestBuildDigitalStoreRuntimeInstruction(t *testing.T) {
	setupDigitalStoreRuntimeInstructionTestDB(t)
	now := time.Now()
	cfg := digitalStoreProfileConfig{
		DigitalStoreProfileRequest: request.DigitalStoreProfileRequest{
			BrandName:         "慕斯寝具",
			Industry:          "家居寝具",
			StoreName:         "徐汇体验店",
			StoreAddress:      "上海市徐汇区样板路88号",
			BusinessHours:     "10:00-21:00",
			ContactPhone:      "400-888-8888",
			AIManagerName:     "慕小眠",
			AIPersona:         "专业耐心的睡眠顾问",
			ReplyStyle:        "先回答，再推荐，再引导预约",
			AppointmentPolicy: "预约试躺需留下姓名、手机号、到店日期和人数",
			HandoffPolicy:     "客户留资或询问最终成交价时转人工",
			ForbiddenClaims:   "不得承诺治疗疾病",
			Initialized:       true,
		},
	}
	raw, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}
	if err := sqls.DB().Create(&models.SystemConfig{
		ConfigKey:   digitalStoreProfileConfigKey,
		ConfigValue: string(raw),
		GroupCode:   digitalStoreConfigGroup,
		Status:      enums.StatusOk,
	}).Error; err != nil {
		t.Fatalf("create config: %v", err)
	}
	if err := sqls.DB().Create(&models.Product{
		Name:           "慕斯脊护支撑款",
		Category:       "床垫",
		PriceMin:       12000,
		PriceMax:       18000,
		SellingPoints:  "分区承托、偏硬支撑",
		SuitablePeople: "老人、腰背压力明显的人群",
		Priority:       100,
		Status:         enums.StatusOk,
	}).Error; err != nil {
		t.Fatalf("create product: %v", err)
	}
	if err := sqls.DB().Create(&models.Product{
		Name:     "下架旧款",
		Priority: 1,
		Status:   enums.StatusDeleted,
	}).Error; err != nil {
		t.Fatalf("create deleted product: %v", err)
	}
	start := now.Add(-time.Hour)
	end := now.Add(time.Hour)
	expiredEnd := now.Add(-time.Hour)
	if err := sqls.DB().Create(&models.Promotion{
		Name:               "周末预约试躺礼",
		ApplicableProducts: "慕斯脊护支撑款",
		StartAt:            &start,
		EndAt:              &end,
		DiscountRule:       "成交价到店确认",
		AppointmentBenefit: "护睡礼包",
		ScriptSuggestion:   "引导客户留下手机号预约",
		Priority:           100,
		Status:             enums.StatusOk,
	}).Error; err != nil {
		t.Fatalf("create promotion: %v", err)
	}
	if err := sqls.DB().Create(&models.Promotion{
		Name:     "过期活动",
		EndAt:    &expiredEnd,
		Priority: 1,
		Status:   enums.StatusOk,
	}).Error; err != nil {
		t.Fatalf("create expired promotion: %v", err)
	}

	got := DigitalStoreProfileService.BuildRuntimeInstruction()
	for _, want := range []string{
		"AI数字店长运行上下文",
		"品牌：慕斯寝具",
		"门店：徐汇体验店",
		"慕斯脊护支撑款",
		"价格 12000-18000元",
		"周末预约试躺礼",
		"预约权益 护睡礼包",
		"先直接回答客户核心问题",
		"AI 回复安全护栏",
		"不得承诺最低价",
		"不得自行承诺退款",
		"不得使用一定、保证、百分百",
		"不得承诺治疗疾病",
		"家居寝具行业禁用承诺",
		"库存是实时信息",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("runtime instruction missing %q:\n%s", want, got)
		}
	}
	for _, forbidden := range []string{"下架旧款", "过期活动"} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("runtime instruction should not contain %q:\n%s", forbidden, got)
		}
	}
}

func TestDigitalStoreMaintenanceStatusFindsLatestBackup(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	createDigitalStoreBackupFixture(t, "backups/20260101-120000", "20260101-120000", "2026-01-01T12:00:00Z")
	createDigitalStoreBackupFixture(t, "backups/20260102-090000", "20260102-090000", "2026-01-02T09:00:00Z")

	got := DigitalStoreProfileService.GetMaintenanceStatus()
	if got.Status != "ok" || got.LatestBackup == nil {
		t.Fatalf("unexpected maintenance status: %#v", got)
	}
	if got.LatestBackup.Timestamp != "20260102-090000" {
		t.Fatalf("expected latest backup timestamp, got %#v", got.LatestBackup)
	}
	if !got.LatestBackup.HasManifest || !got.LatestBackup.HasMySQLDump || !got.LatestBackup.HasDataArchive || !got.LatestBackup.HasDockerConfigArchive || !got.LatestBackup.HasConfigSnapshot {
		t.Fatalf("expected complete backup snapshot, got %#v", got.LatestBackup)
	}
	if !strings.Contains(got.RestoreDryRunCommand, "backups/20260102-090000") {
		t.Fatalf("restore command should point to latest backup: %s", got.RestoreDryRunCommand)
	}
	if len(got.UpgradeCommands) == 0 || !strings.Contains(strings.Join(got.UpgradeCommands, "\n"), "docker compose") {
		t.Fatalf("expected upgrade commands, got %#v", got.UpgradeCommands)
	}
	for _, want := range []string{"单商家升级 Runbook", "backups/20260102-090000", "MUSE_ACCEPTANCE_TIMEOUT_MS=70000", "发送关键通知测试"} {
		if !strings.Contains(got.UpgradeRunbook, want) {
			t.Fatalf("upgrade runbook missing %q:\n%s", want, got.UpgradeRunbook)
		}
	}
}

func TestDigitalStoreMaintenanceStatusWarnsWhenNoBackup(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)

	got := DigitalStoreProfileService.GetMaintenanceStatus()
	if got.Status != "warning" || got.LatestBackup != nil || len(got.Warnings) == 0 {
		t.Fatalf("expected no-backup warning, got %#v", got)
	}
	if !strings.Contains(got.RestoreDryRunCommand, "backups/<备份目录>") {
		t.Fatalf("restore command should keep placeholder, got %s", got.RestoreDryRunCommand)
	}
	if !strings.Contains(got.UpgradeRunbook, "未发现本地备份") || !strings.Contains(got.UpgradeRunbook, got.BackupCommand) {
		t.Fatalf("upgrade runbook should explain missing backup:\n%s", got.UpgradeRunbook)
	}
}

func TestDigitalStoreTemplateEffectSummarizesKnowledgeGapsAndFeedback(t *testing.T) {
	setupDigitalStoreRuntimeSetupTestDB(t)
	now := time.Now()
	cfg := digitalStoreProfileConfig{
		DigitalStoreProfileRequest: request.DigitalStoreProfileRequest{
			BrandName:       "慕斯寝具",
			Industry:        "家居寝具",
			KnowledgeBaseID: 88,
			Initialized:     true,
		},
		TemplateCode:      "muse_bedding",
		TemplateVersion:   "1.0.0",
		TemplateAppliedAt: utils.FormatTime(now.AddDate(0, 0, -7)),
	}
	if err := DigitalStoreProfileService.saveConfig(cfg, &dto.AuthPrincipal{UserID: 1, Username: "admin"}); err != nil {
		t.Fatalf("save digital store profile config: %v", err)
	}
	logs := []models.KnowledgeRetrieveLog{
		{KnowledgeBaseID: 88, Question: "活动能不能叠加？", AnswerStatus: int(enums.KnowledgeAnswerStatusNoAnswer), CreatedAt: now.Add(-2 * time.Hour)},
		{KnowledgeBaseID: 88, Question: "活动能不能叠加？", AnswerStatus: int(enums.KnowledgeAnswerStatusFallback), CreatedAt: now.Add(-time.Hour)},
		{KnowledgeBaseID: 88, Question: "床垫保修多久？", AnswerStatus: int(enums.KnowledgeAnswerStatusNormal), CreatedAt: now.Add(-3 * time.Hour)},
		{KnowledgeBaseID: 88, Question: "旧问题", AnswerStatus: int(enums.KnowledgeAnswerStatusNoAnswer), CreatedAt: now.AddDate(0, 0, -40)},
		{KnowledgeBaseID: 99, Question: "其他知识库问题", AnswerStatus: int(enums.KnowledgeAnswerStatusNoAnswer), CreatedAt: now},
	}
	if err := sqls.DB().Create(&logs).Error; err != nil {
		t.Fatalf("create retrieve logs: %v", err)
	}
	feedbacks := []models.KnowledgeFeedback{
		{RetrieveLogID: logs[2].ID, FeedbackType: int(enums.KnowledgeFeedbackTypeDislike), FeedbackReason: "保修说错", CreatedAt: now.Add(-90 * time.Minute)},
		{RetrieveLogID: logs[2].ID, FeedbackType: int(enums.KnowledgeFeedbackTypeWrongCitation), FeedbackReason: "引用不准", CreatedAt: now.Add(-30 * time.Minute)},
		{RetrieveLogID: logs[0].ID, FeedbackType: int(enums.KnowledgeFeedbackTypeLike), FeedbackReason: "清楚", CreatedAt: now},
	}
	if err := sqls.DB().Create(&feedbacks).Error; err != nil {
		t.Fatalf("create feedbacks: %v", err)
	}

	got := DigitalStoreProfileService.GetTemplateEffect()
	if got.TemplateCode != "muse_bedding" || got.TemplateVersion != "1.0.0" || got.KnowledgeBaseID != 88 {
		t.Fatalf("unexpected template metadata: %#v", got)
	}
	if got.RetrieveTotal != 3 || got.MissingQuestionTotal != 2 || got.NegativeFeedbackTotal != 2 {
		t.Fatalf("unexpected totals: %#v", got)
	}
	if len(got.MissingQuestions) == 0 || got.MissingQuestions[0].Question != "活动能不能叠加？" || got.MissingQuestions[0].Count != 2 {
		t.Fatalf("unexpected missing questions: %#v", got.MissingQuestions)
	}
	if len(got.NegativeFeedbacks) == 0 || got.NegativeFeedbacks[0].Question != "床垫保修多久？" || got.NegativeFeedbacks[0].Count != 2 {
		t.Fatalf("unexpected negative feedbacks: %#v", got.NegativeFeedbacks)
	}
	if got.NegativeFeedbacks[0].ActionHref == "" || len(got.Suggestions) == 0 {
		t.Fatalf("expected action href and suggestions: %#v", got)
	}
	if !strings.Contains(got.ImprovementMarkdown, "行业模板改进包：muse_bedding") ||
		!strings.Contains(got.ImprovementMarkdown, "活动能不能叠加？") ||
		!strings.Contains(got.ImprovementMarkdown, "床垫保修多久？") ||
		!strings.Contains(got.ImprovementMarkdown, "导出行业模板 JSON") {
		t.Fatalf("unexpected improvement markdown: %s", got.ImprovementMarkdown)
	}
}

func TestBuildDigitalStoreSafetyGuardrailIncludesMedicalRiskRules(t *testing.T) {
	got := buildDigitalStoreSafetyGuardrailRuntimeSection(oralClinicDigitalStoreProfile())
	for _, want := range []string{"医疗健康行业禁用承诺", "不得在线诊断", "急性疼痛", "医保报销"} {
		if !strings.Contains(got, want) {
			t.Fatalf("medical guardrail missing %q:\n%s", want, got)
		}
	}
}

func TestDigitalStoreEnsureAgentWorkflowAndWebChannel(t *testing.T) {
	setupDigitalStoreRuntimeSetupTestDB(t)
	operator := &dto.AuthPrincipal{UserID: 1, Username: "admin"}
	aiConfig := &models.AIConfig{
		Name:      "default llm",
		Provider:  enums.AIProviderOpenAI,
		ModelType: enums.AIModelTypeLLM,
		ModelName: "deepseek-v4-flash",
		Status:    enums.StatusOk,
	}
	if err := sqls.DB().Create(aiConfig).Error; err != nil {
		t.Fatalf("create ai config: %v", err)
	}
	kb := &models.KnowledgeBase{
		Name:          "数字店长 FAQ 知识库",
		KnowledgeType: string(enums.KnowledgeBaseTypeFAQ),
		Status:        enums.StatusOk,
	}
	if err := sqls.DB().Create(kb).Error; err != nil {
		t.Fatalf("create knowledge base: %v", err)
	}
	if err := sqls.DB().Create(&models.User{
		Username: "admin",
		Nickname: "门店顾问",
		Status:   enums.StatusOk,
	}).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	cfg := digitalStoreProfileConfig{
		DigitalStoreProfileRequest: request.DigitalStoreProfileRequest{
			BrandName:       "慕斯寝具",
			StoreName:       "徐汇体验店",
			AIManagerName:   "慕小眠",
			KnowledgeBaseID: kb.ID,
			Initialized:     true,
		},
	}

	defaultTeamID, err := DigitalStoreProfileService.ensureDefaultHumanHandoffRuntime(operator)
	if err != nil {
		t.Fatalf("ensureDefaultHumanHandoffRuntime() error = %v", err)
	}
	if defaultTeamID <= 0 {
		t.Fatal("expected default human handoff team")
	}
	agent, err := DigitalStoreProfileService.ensureAgent(cfg, aiConfig.ID, defaultTeamID, operator)
	if err != nil {
		t.Fatalf("ensureAgent() error = %v", err)
	}
	if agent == nil || agent.ID == 0 {
		t.Fatalf("expected agent created, got %#v", agent)
	}
	if !strings.Contains(agent.SystemPrompt, "库存属于实时信息") {
		t.Fatalf("expected inventory safety rule in agent prompt, got %q", agent.SystemPrompt)
	}
	if !slices.Contains(utils.SplitInt64s(agent.TeamIDs), defaultTeamID) {
		t.Fatalf("expected agent bound to default team %d, got %q", defaultTeamID, agent.TeamIDs)
	}
	if handoff := buildDigitalStoreHumanHandoff(agent); !handoff.Ready {
		t.Fatalf("expected human handoff ready, got %#v", handoff)
	}
	if err := DigitalStoreProfileService.ensureAgentWorkflowPublished(agent.ID, operator); err != nil {
		t.Fatalf("ensureAgentWorkflowPublished() error = %v", err)
	}
	agent = AIAgentService.Get(agent.ID)
	if agent.WorkflowVersionID <= 0 {
		t.Fatalf("expected published workflow version, got agent %#v", agent)
	}
	if err := DigitalStoreProfileService.ensureWebChannel(cfg, agent, operator); err != nil {
		t.Fatalf("ensureWebChannel() error = %v", err)
	}
	channel := DigitalStoreProfileService.findWebChannel(cfg)
	if channel == nil || channel.AIAgentID != agent.ID || channel.Status != enums.StatusOk || strings.TrimSpace(channel.ChannelID) == "" {
		t.Fatalf("unexpected web channel: %#v", channel)
	}
	webCfg, err := ChannelService.ParseWebChannelConfig(channel.ConfigJSON)
	if err != nil {
		t.Fatalf("parse web channel config: %v", err)
	}
	if webCfg.Title != "慕小眠" || webCfg.Subtitle != "慕斯寝具" || webCfg.ThemeColor != "#2563eb" {
		t.Fatalf("unexpected digital store web config: %#v", webCfg)
	}
}

func TestDigitalStoreIndustryTemplateRuntimeUsesBrandedAgentAndWebChannel(t *testing.T) {
	setupDigitalStoreRuntimeSetupTestDB(t)
	operator := &dto.AuthPrincipal{UserID: 1, Username: "admin"}
	aiConfig := &models.AIConfig{
		Name:      "default llm",
		Provider:  enums.AIProviderOpenAI,
		ModelType: enums.AIModelTypeLLM,
		ModelName: "deepseek-v4-flash",
		Status:    enums.StatusOk,
	}
	if err := sqls.DB().Create(aiConfig).Error; err != nil {
		t.Fatalf("create ai config: %v", err)
	}
	kb := &models.KnowledgeBase{
		Name:          "数字店长 FAQ 知识库",
		KnowledgeType: string(enums.KnowledgeBaseTypeFAQ),
		Status:        enums.StatusOk,
	}
	if err := sqls.DB().Create(kb).Error; err != nil {
		t.Fatalf("create knowledge base: %v", err)
	}
	if err := sqls.DB().Create(&models.User{
		Username: "admin",
		Nickname: "门店顾问",
		Status:   enums.StatusOk,
	}).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	cfg := homeDecorationDigitalStoreProfile()
	cfg.KnowledgeBaseID = kb.ID

	defaultTeamID, err := DigitalStoreProfileService.ensureDefaultHumanHandoffRuntime(operator)
	if err != nil {
		t.Fatalf("ensureDefaultHumanHandoffRuntime() error = %v", err)
	}
	agent, err := DigitalStoreProfileService.ensureAgent(cfg, aiConfig.ID, defaultTeamID, operator)
	if err != nil {
		t.Fatalf("ensureAgent() error = %v", err)
	}
	if agent.Name != "木小顾 AI数字店长" {
		t.Fatalf("unexpected branded agent name: %s", agent.Name)
	}
	if !strings.Contains(agent.SystemPrompt, "良木整装") || !strings.Contains(agent.SystemPrompt, "不得承诺一口价") {
		t.Fatalf("agent prompt should include industry brand and guardrail: %s", agent.SystemPrompt)
	}
	if err := DigitalStoreProfileService.ensureAgentWorkflowPublished(agent.ID, operator); err != nil {
		t.Fatalf("ensureAgentWorkflowPublished() error = %v", err)
	}
	agent = AIAgentService.Get(agent.ID)
	if err := DigitalStoreProfileService.ensureWebChannel(cfg, agent, operator); err != nil {
		t.Fatalf("ensureWebChannel() error = %v", err)
	}
	channel := DigitalStoreProfileService.findWebChannel(cfg)
	if channel == nil || channel.AIAgentID != agent.ID || channel.ChannelID == "" {
		t.Fatalf("expected branded web channel, got %#v", channel)
	}
	webCfg, err := ChannelService.ParseWebChannelConfig(channel.ConfigJSON)
	if err != nil {
		t.Fatalf("parse web channel config: %v", err)
	}
	if webCfg.Title != "木小顾" || webCfg.Subtitle != "良木整装" {
		t.Fatalf("unexpected industry web channel config: %#v", webCfg)
	}
}

func createDigitalStoreBackupFixture(t *testing.T, dir string, timestamp string, createdAt string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, "config"), 0o755); err != nil {
		t.Fatalf("mkdir backup fixture: %v", err)
	}
	manifest := strings.Join([]string{
		"timestamp=" + timestamp,
		"project_dir=/opt/agent-desk",
		"compose_file=docker-compose.yml",
		"created_at=" + createdAt,
	}, "\n")
	files := map[string]string{
		filepath.Join(dir, "BACKUP-MANIFEST.txt"):   manifest,
		filepath.Join(dir, "mysql.sql"):             "dump",
		filepath.Join(dir, "data.tar.gz"):           "data",
		filepath.Join(dir, "docker-config.tar.gz"):  "docker",
		filepath.Join(dir, "config", "config.yaml"): "config",
	}
	for path, content := range files {
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write backup fixture %s: %v", path, err)
		}
	}
}

func TestDigitalStoreRuntimeDoesNotHijackExistingAgentOrChannel(t *testing.T) {
	setupDigitalStoreRuntimeSetupTestDB(t)
	operator := &dto.AuthPrincipal{UserID: 1, Username: "admin"}
	aiConfig := &models.AIConfig{
		Name:      "default llm",
		Provider:  enums.AIProviderOpenAI,
		ModelType: enums.AIModelTypeLLM,
		ModelName: "deepseek-v4-flash",
		Status:    enums.StatusOk,
	}
	if err := sqls.DB().Create(aiConfig).Error; err != nil {
		t.Fatalf("create ai config: %v", err)
	}
	kb := &models.KnowledgeBase{
		Name:          "数字店长 FAQ 知识库",
		KnowledgeType: string(enums.KnowledgeBaseTypeFAQ),
		Status:        enums.StatusOk,
	}
	if err := sqls.DB().Create(kb).Error; err != nil {
		t.Fatalf("create knowledge base: %v", err)
	}
	existingAgent := &models.AIAgent{
		Name:         "普通客服 Agent",
		Status:       enums.StatusOk,
		AIConfigID:   999,
		KnowledgeIDs: "999",
	}
	if err := sqls.DB().Create(existingAgent).Error; err != nil {
		t.Fatalf("create existing agent: %v", err)
	}
	existingChannel := &models.Channel{
		Name:        "已有官网客服",
		ChannelType: enums.ChannelTypeWeb,
		ChannelID:   "web_existing",
		AIAgentID:   existingAgent.ID,
		Status:      enums.StatusOk,
	}
	if err := sqls.DB().Create(existingChannel).Error; err != nil {
		t.Fatalf("create existing channel: %v", err)
	}
	cfg := digitalStoreProfileConfig{
		DigitalStoreProfileRequest: request.DigitalStoreProfileRequest{
			BrandName:       "慕斯寝具",
			StoreName:       "徐汇体验店",
			AIManagerName:   "慕小眠",
			KnowledgeBaseID: kb.ID,
			Initialized:     true,
		},
	}

	agent, err := DigitalStoreProfileService.ensureAgent(cfg, aiConfig.ID, 0, operator)
	if err != nil {
		t.Fatalf("ensureAgent() error = %v", err)
	}
	if agent.ID == existingAgent.ID {
		t.Fatalf("expected a digital-store agent, got existing agent %#v", agent)
	}
	if err := DigitalStoreProfileService.ensureAgentWorkflowPublished(agent.ID, operator); err != nil {
		t.Fatalf("ensureAgentWorkflowPublished() error = %v", err)
	}
	agent = AIAgentService.Get(agent.ID)
	if err := DigitalStoreProfileService.ensureWebChannel(cfg, agent, operator); err != nil {
		t.Fatalf("ensureWebChannel() error = %v", err)
	}
	var unchanged models.Channel
	if err := sqls.DB().First(&unchanged, existingChannel.ID).Error; err != nil {
		t.Fatalf("load existing channel: %v", err)
	}
	if unchanged.AIAgentID != existingAgent.ID {
		t.Fatalf("existing channel was hijacked: %#v", unchanged)
	}
	channel := DigitalStoreProfileService.findWebChannel(cfg)
	if channel == nil || channel.ID == existingChannel.ID || channel.AIAgentID != agent.ID {
		t.Fatalf("unexpected digital store channel: %#v", channel)
	}
}

func TestDigitalStoreDeliveryReportContainsHandoffArtifacts(t *testing.T) {
	setupDigitalStoreDeliveryReportFixture(t)
	config.SetCurrent(&config.Config{})
	t.Cleanup(func() { config.SetCurrent(&config.Config{}) })

	report := DigitalStoreProfileService.GetDeliveryReport("https://muse.example.com/")
	if !report.Ready {
		t.Fatalf("expected report ready, got %#v", report)
	}
	if report.AcceptanceCommand != "MUSE_ACCEPTANCE_TIMEOUT_MS=70000 scripts/run-muse-chat-acceptance.mjs" {
		t.Fatalf("unexpected acceptance command: %s", report.AcceptanceCommand)
	}
	if len(report.AcceptanceItems) == 0 {
		t.Fatal("expected acceptance checklist items")
	}
	for _, want := range []string{
		"AI 数字店长上线验收执行清单",
		"| [ ] | A04 | 预约试躺留资",
		"客户聊天入口：https://muse.example.com/support/chat/?channelId=web_muse_test",
		"阻断项未通过时不得上线",
	} {
		if !strings.Contains(report.AcceptanceRunbook, want) {
			t.Fatalf("acceptance runbook missing %q:\n%s", want, report.AcceptanceRunbook)
		}
	}
	for _, want := range []string{
		"https://muse.example.com/dashboard",
		"https://muse.example.com/support/chat/?channelId=web_muse_test",
		"agent-desk-sdk.min.js",
		"MUSE_ACCEPTANCE_TIMEOUT_MS=70000 scripts/run-muse-chat-acceptance.mjs",
		"A04 预约试躺留资",
		"慕斯寝具",
		"配置检查",
		"模型与检索健康",
		"外部通知",
		"上线安全自检",
		"Embedding 模型",
		"人工接待配置",
		"顾问可自动接待",
		"客户入口品牌化",
		"浮窗标题：慕小眠",
		"themeColor: \"#2563eb\"",
		"baseUrl: \"https://muse.example.com\"",
	} {
		if !strings.Contains(report.Markdown, want) {
			t.Fatalf("delivery report missing %q:\n%s", want, report.Markdown)
		}
	}
	if report.WebEntry.Title != "慕小眠" || report.WebEntry.Subtitle != "慕斯寝具" || report.WebEntry.ChatURL == "" {
		t.Fatalf("unexpected web entry in report: %#v", report.WebEntry)
	}
	if report.WebEntry.ChannelID <= 0 {
		t.Fatalf("expected web entry channel id, got %#v", report.WebEntry)
	}
	if !report.HumanHandoff.Ready || report.HumanHandoff.CandidateProfiles == 0 {
		t.Fatalf("expected human handoff ready in report, got %#v", report.HumanHandoff)
	}
}

func TestDigitalStoreSetupStatusRequiresEmbeddingConfig(t *testing.T) {
	setupDigitalStoreDeliveryReportFixture(t)
	if err := sqls.DB().Model(&models.AIConfig{}).
		Where("model_type = ?", enums.AIModelTypeEmbedding).
		Update("status", enums.StatusDeleted).Error; err != nil {
		t.Fatalf("disable embedding config: %v", err)
	}

	status := DigitalStoreProfileService.GetSetupStatus()
	if status.Ready {
		t.Fatalf("expected setup status not ready without embedding config: %#v", status)
	}
	if status.EmbeddingConfigID != 0 {
		t.Fatalf("expected no active embedding config, got %#v", status)
	}
	found := false
	for _, step := range status.MissingSteps {
		if strings.Contains(step, "Embedding") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("missing steps should mention embedding config: %#v", status.MissingSteps)
	}
	foundHealth := false
	for _, item := range status.ModelHealthChecks {
		if item.Key == "embedding" && item.Status == "blocking" && item.ActionHref == "/dashboard/ai-configs" {
			foundHealth = true
			break
		}
	}
	if !foundHealth {
		t.Fatalf("model health should flag missing embedding config: %#v", status.ModelHealthChecks)
	}
}

func TestDigitalStoreSetupStatusRequiresProductKnowledgeCoverage(t *testing.T) {
	setupDigitalStoreDeliveryReportFixture(t)
	if err := sqls.DB().Model(&models.Product{}).
		Where("name = ?", "慕斯脊护支撑款").
		Update("knowledge_faq_id", 0).Error; err != nil {
		t.Fatalf("clear product faq: %v", err)
	}

	status := DigitalStoreProfileService.GetSetupStatus()
	if status.Ready {
		t.Fatalf("expected setup status not ready without product knowledge coverage: %#v", status)
	}
	if status.ProductKnowledgeUnsyncedTotal != 1 || status.ProductKnowledgeSyncedTotal != 0 {
		t.Fatalf("unexpected product knowledge coverage: %#v", status)
	}
	found := false
	for _, step := range status.MissingSteps {
		if strings.Contains(step, "产品知识索引") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("missing steps should mention product knowledge index: %#v", status.MissingSteps)
	}
	report := DigitalStoreProfileService.GetDeliveryReport("https://muse.example.com/")
	if !strings.Contains(report.Markdown, "产品知识索引") || !strings.Contains(report.Markdown, "未同步 1") {
		t.Fatalf("delivery report should include product knowledge coverage:\n%s", report.Markdown)
	}
	foundAction := false
	for _, item := range report.Items {
		if item.Label == "产品知识索引" {
			foundAction = item.ActionHref == "/dashboard/products" && item.ActionLabel == "去同步"
			break
		}
	}
	if !foundAction {
		t.Fatalf("product knowledge report item should include fix action: %#v", report.Items)
	}
	foundHealth := false
	for _, item := range report.ModelHealthChecks {
		if item.Key == "product_index" && item.Status == "blocking" && item.ActionHref == "/dashboard/products" {
			foundHealth = true
			break
		}
	}
	if !foundHealth {
		t.Fatalf("model health should flag product knowledge index: %#v", report.ModelHealthChecks)
	}
}

func TestDigitalStoreSecurityChecksFlagUnsafeDefaults(t *testing.T) {
	setupDigitalStoreDeliveryReportFixture(t)
	t.Setenv(constants.EnvBootstrapAdminPassword, "")
	config.SetCurrent(&config.Config{})
	t.Cleanup(func() { config.SetCurrent(&config.Config{}) })

	report := DigitalStoreProfileService.GetDeliveryReport("https://muse.example.com/")
	if len(report.SecurityChecks) == 0 {
		t.Fatal("expected security checks")
	}
	foundBlocking := false
	for _, item := range report.SecurityChecks {
		if item.Status == "blocking" {
			foundBlocking = true
			break
		}
	}
	if !foundBlocking {
		t.Fatalf("expected blocking security check, got %#v", report.SecurityChecks)
	}
	if !strings.Contains(report.Markdown, "首次管理员密码") || !strings.Contains(report.Markdown, "客户聊天密钥") {
		t.Fatalf("security checks missing from markdown:\n%s", report.Markdown)
	}
	foundAction := false
	for _, item := range report.SecurityChecks {
		if item.Status == "blocking" && item.ActionHref != "" && item.ActionLabel != "" {
			foundAction = true
			break
		}
	}
	if !foundAction {
		t.Fatalf("blocking security checks should include action targets: %#v", report.SecurityChecks)
	}
}

func TestDigitalStoreSecurityChecksWarnWebhookWithoutSecret(t *testing.T) {
	setupDigitalStoreDeliveryReportFixture(t)
	t.Setenv(constants.EnvBootstrapAdminPassword, "merchant-admin-password-2026")
	config.SetCurrent(&config.Config{
		Server: config.ServerConfig{
			CORS: config.CORSConfig{AllowedOrigins: []string{"https://muse.example.com"}},
		},
		DB:              config.DBConfig{Type: "mysql"},
		Auth:            config.AuthConfig{MaxFailedAttempts: 5},
		CustomerSession: config.CustomerSessionConfig{Secret: "merchant-customer-session-secret-2026"},
		VectorDB:        config.VectorDBConfig{Type: "qdrant"},
		Notify: config.NotifyConfig{
			Webhook: config.WebhookNotifyConfig{
				Enabled: true,
				URL:     "https://hooks.example.com/agent-desk",
				Format:  "generic",
			},
		},
	})
	t.Cleanup(func() { config.SetCurrent(&config.Config{}) })

	report := DigitalStoreProfileService.GetDeliveryReport("https://muse.example.com/")
	found := false
	for _, item := range report.SecurityChecks {
		if item.Key == "webhook_secret" {
			found = item.Status == "warning" && strings.Contains(item.Message, "未配置")
			break
		}
	}
	if !found {
		t.Fatalf("expected webhook secret warning, got %#v", report.SecurityChecks)
	}
}

func TestDigitalStoreSecurityChecksPassProductionConfig(t *testing.T) {
	setupDigitalStoreDeliveryReportFixture(t)
	t.Setenv(constants.EnvBootstrapAdminPassword, "merchant-admin-password-2026")
	config.SetCurrent(&config.Config{
		Server: config.ServerConfig{
			CORS: config.CORSConfig{AllowedOrigins: []string{"https://muse.example.com", "https://www.muse.example.com"}},
		},
		DB: config.DBConfig{
			Type: "mysql",
			DSN:  "merchant:strong-password@tcp(mysql:3306)/agent_desk?parseTime=True",
		},
		Auth: config.AuthConfig{
			MaxFailedAttempts:    5,
			CredentialLockMinute: 15,
		},
		CustomerSession: config.CustomerSessionConfig{
			Secret: "merchant-customer-session-secret-2026",
		},
		VectorDB: config.VectorDBConfig{Type: "qdrant"},
		Notify: config.NotifyConfig{
			Webhook: config.WebhookNotifyConfig{
				Enabled: true,
				URL:     "https://hooks.example.com/agent-desk",
				Format:  "generic",
				Secret:  "webhook-signing-secret",
			},
		},
	})
	t.Cleanup(func() { config.SetCurrent(&config.Config{}) })

	report := DigitalStoreProfileService.GetDeliveryReport("https://muse.example.com/")
	foundVectorHealth := false
	for _, item := range report.ModelHealthChecks {
		if item.Key == "vector_db" && item.Status == "ok" {
			foundVectorHealth = true
			break
		}
	}
	if !foundVectorHealth {
		t.Fatalf("expected vector db health ok, got %#v", report.ModelHealthChecks)
	}
	for _, item := range report.SecurityChecks {
		if item.Status == "blocking" {
			t.Fatalf("unexpected blocking security check: %#v", report.SecurityChecks)
		}
		if item.Status == "ok" && (item.ActionHref != "" || item.ActionLabel != "") {
			t.Fatalf("ok security check should not expose action target: %#v", item)
		}
	}
	if !strings.Contains(report.Markdown, "上线安全自检") || !strings.Contains(report.Markdown, "全部通过") {
		t.Fatalf("security check summary missing from markdown:\n%s", report.Markdown)
	}
}

func TestDigitalStoreWebhookNotifyTestSendsConfiguredWebhook(t *testing.T) {
	setupDigitalStoreDeliveryReportFixture(t)
	var got map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatalf("decode webhook payload: %v", err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	config.SetCurrent(&config.Config{
		Notify: config.NotifyConfig{
			Webhook: config.WebhookNotifyConfig{
				Enabled: true,
				URL:     server.URL,
				Format:  "generic",
			},
		},
	})
	t.Cleanup(func() { config.SetCurrent(&config.Config{}) })

	resp, err := DigitalStoreProfileService.TestWebhookNotify(&dto.AuthPrincipal{UserID: 9, Username: "delivery"})
	if err != nil {
		t.Fatalf("TestWebhookNotify() error = %v", err)
	}
	if !resp.Sent || !resp.Enabled {
		t.Fatalf("expected sent enabled webhook test, got %#v", resp)
	}
	if got["eventType"] != "digital_store_webhook_test" || !strings.Contains(got["text"].(string), "AI 数字店长外部通知测试") {
		t.Fatalf("unexpected webhook payload: %#v", got)
	}
}

func TestDigitalStoreWebhookNotifyScenarioTestSendsKeyEvents(t *testing.T) {
	setupDigitalStoreDeliveryReportFixture(t)
	got := []map[string]any{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode webhook payload: %v", err)
		}
		got = append(got, payload)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	config.SetCurrent(&config.Config{
		Notify: config.NotifyConfig{
			Webhook: config.WebhookNotifyConfig{
				Enabled: true,
				URL:     server.URL,
				Format:  "generic",
			},
		},
	})
	t.Cleanup(func() { config.SetCurrent(&config.Config{}) })

	resp, err := DigitalStoreProfileService.TestWebhookNotifyScenarios(&dto.AuthPrincipal{UserID: 9, Username: "delivery"})
	if err != nil {
		t.Fatalf("TestWebhookNotifyScenarios() error = %v", err)
	}
	if !resp.Sent || len(resp.Scenarios) != 5 || len(got) != 5 {
		t.Fatalf("expected five sent scenarios, resp=%#v got=%#v", resp, got)
	}
	joined := ""
	for _, payload := range got {
		joined += fmt.Sprint(payload["eventType"]) + "|" + fmt.Sprint(payload["title"]) + "\n"
	}
	for _, want := range []string{"sales_lead_created|高意向销售线索提醒", "sales_lead_created|预约到店线索提醒", "conversation_assigned|客户转人工提醒", "sales_lead_follow_up_reminder|未分配线索跟进提醒", "sales_lead_created|售后风险线索提醒"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("scenario webhook missing %q in:\n%s", want, joined)
		}
	}
}

func TestDigitalStoreWebhookNotifyScenarioTestReturnsFailureDetails(t *testing.T) {
	setupDigitalStoreDeliveryReportFixture(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("merchant webhook unavailable"))
	}))
	defer server.Close()
	config.SetCurrent(&config.Config{
		Notify: config.NotifyConfig{
			Webhook: config.WebhookNotifyConfig{
				Enabled: true,
				URL:     server.URL,
				Format:  "generic",
			},
		},
	})
	t.Cleanup(func() { config.SetCurrent(&config.Config{}) })

	resp, err := DigitalStoreProfileService.TestWebhookNotifyScenarios(&dto.AuthPrincipal{UserID: 9, Username: "delivery"})
	if err != nil {
		t.Fatalf("TestWebhookNotifyScenarios() should return structured failure result, got error = %v", err)
	}
	if resp.Sent || resp.SentTotal != 0 || resp.FailedTotal != 5 || len(resp.Scenarios) != 5 {
		t.Fatalf("unexpected failed scenario response: %#v", resp)
	}
	for _, item := range resp.Scenarios {
		if item.Sent || !strings.Contains(item.Message, "status=500") || !strings.Contains(item.Message, "merchant webhook unavailable") {
			t.Fatalf("expected scenario failure details, got %#v", item)
		}
	}
	if !strings.Contains(resp.Message, "成功 0，失败 5") {
		t.Fatalf("expected aggregate failure message, got %q", resp.Message)
	}
}

func TestDigitalStoreWebhookNotifyScenarioTestReportsDisabled(t *testing.T) {
	setupDigitalStoreDeliveryReportFixture(t)
	config.SetCurrent(&config.Config{})
	t.Cleanup(func() { config.SetCurrent(&config.Config{}) })

	resp, err := DigitalStoreProfileService.TestWebhookNotifyScenarios(&dto.AuthPrincipal{UserID: 9, Username: "delivery"})
	if err != nil {
		t.Fatalf("TestWebhookNotifyScenarios() error = %v", err)
	}
	if resp.Sent || resp.Enabled || len(resp.Scenarios) != 5 {
		t.Fatalf("expected disabled scenario response, got %#v", resp)
	}
	for _, item := range resp.Scenarios {
		if item.Sent || item.EventType == "" || item.Title == "" {
			t.Fatalf("unexpected disabled scenario item: %#v", item)
		}
	}
}

func TestDigitalStoreWebhookNotifyTestReportsDisabled(t *testing.T) {
	setupDigitalStoreDeliveryReportFixture(t)
	config.SetCurrent(&config.Config{})
	t.Cleanup(func() { config.SetCurrent(&config.Config{}) })

	resp, err := DigitalStoreProfileService.TestWebhookNotify(&dto.AuthPrincipal{UserID: 9, Username: "delivery"})
	if err != nil {
		t.Fatalf("TestWebhookNotify() error = %v", err)
	}
	if resp.Sent || resp.Enabled || resp.Status == "" {
		t.Fatalf("expected disabled webhook test response, got %#v", resp)
	}
}

func TestDigitalStoreAcceptanceItemsForOralClinic(t *testing.T) {
	cfg := oralClinicDigitalStoreProfile()
	if command := defaultDigitalStoreAcceptanceCommand(cfg); strings.Contains(command, "run-muse-chat-acceptance") {
		t.Fatalf("oral clinic should not point to muse acceptance script: %s", command)
	}
	items := buildDigitalStoreAcceptanceItems(cfg)
	if len(items) < 6 {
		t.Fatalf("expected oral clinic acceptance checklist, got %d items", len(items))
	}
	joined := ""
	for _, item := range items {
		joined += item.CustomerAsk + item.Expectation + item.ConsoleCheck
	}
	for _, want := range []string{"隐形矫正", "医生面诊", "不得出现百分百"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("oral clinic acceptance checklist missing %q: %#v", want, items)
		}
	}
}

func TestDigitalStoreAcceptanceItemsForIndustryMatrices(t *testing.T) {
	cases := []struct {
		name     string
		cfg      digitalStoreProfileConfig
		keywords []string
	}{
		{
			name: "education",
			cfg: digitalStoreProfileConfig{
				TemplateCode: "kids_english",
				DigitalStoreProfileRequest: request.DigitalStoreProfileRequest{
					Industry:  "教育培训",
					BrandName: "启明星英语",
					StoreName: "浦东校区",
				},
			},
			keywords: []string{"试听", "学生年级", "保过", "保证提分"},
		},
		{
			name: "finance",
			cfg: digitalStoreProfileConfig{
				TemplateCode: "finance_advisor",
				DigitalStoreProfileRequest: request.DigitalStoreProfileRequest{
					Industry:  "金融服务",
					BrandName: "安信顾问",
					StoreName: "线上咨询中心",
				},
			},
			keywords: []string{"风险提示", "持牌顾问", "保本", "贷款一定能批"},
		},
		{
			name: "home_decoration",
			cfg: digitalStoreProfileConfig{
				TemplateCode: "home_decoration",
				DigitalStoreProfileRequest: request.DigitalStoreProfileRequest{
					Industry:  "家装装修",
					BrandName: "良木整装",
					StoreName: "城西店",
				},
			},
			keywords: []string{"量房", "面积", "不增项", "固定工期"},
		},
		{
			name: "bedding",
			cfg: digitalStoreProfileConfig{
				TemplateCode: "muse_bedding",
				DigitalStoreProfileRequest: request.DigitalStoreProfileRequest{
					Industry:  "家居寝具",
					BrandName: "慕斯寝具",
					StoreName: "徐汇体验店",
				},
			},
			keywords: []string{"试躺", "治好腰疼", "售后风险", "异响"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			items := buildDigitalStoreAcceptanceItems(tc.cfg)
			if len(items) < 6 {
				t.Fatalf("expected industry acceptance checklist, got %d items: %#v", len(items), items)
			}
			joined := ""
			blocking := 0
			for _, item := range items {
				joined += item.Title + item.CustomerAsk + item.Expectation + item.ConsoleCheck
				if item.Blocking {
					blocking++
				}
			}
			if blocking < 4 {
				t.Fatalf("expected enough blocking acceptance items, got %d: %#v", blocking, items)
			}
			for _, want := range tc.keywords {
				if !strings.Contains(joined, want) {
					t.Fatalf("%s acceptance checklist missing %q: %#v", tc.name, want, items)
				}
			}
		})
	}
}

func TestDigitalStoreCreateDeliveryRecordArchivesLatestReport(t *testing.T) {
	setupDigitalStoreDeliveryReportFixture(t)
	operator := &dto.AuthPrincipal{UserID: 7, Username: "delivery"}
	record, err := DigitalStoreProfileService.CreateDeliveryRecord(request.DigitalStoreDeliveryRecordCreateRequest{
		PublicBaseURL:     "https://muse.example.com",
		AcceptanceStatus:  "passed",
		AcceptanceSummary: "M01-M15 自动化验收通过。",
	}, operator)
	if err != nil {
		t.Fatalf("CreateDeliveryRecord() error = %v", err)
	}
	if record == nil || record.ID == 0 || record.AcceptanceStatus != "passed" || record.CreateUserName != "delivery" {
		t.Fatalf("unexpected delivery record: %#v", record)
	}
	if !strings.Contains(record.AcceptanceCommand, "run-muse-chat-acceptance") {
		t.Fatalf("expected acceptance command on record, got %#v", record)
	}
	latest := DigitalStoreProfileService.GetLatestDeliveryRecord()
	if latest == nil || latest.ID != record.ID {
		t.Fatalf("unexpected latest delivery record: %#v", latest)
	}
	report := DigitalStoreProfileService.GetDeliveryReport("https://muse.example.com")
	if report.LatestRecord == nil || report.LatestRecord.ID != record.ID {
		t.Fatalf("expected latest record in report, got %#v", report.LatestRecord)
	}
}

func TestDigitalStoreCreateAcceptanceResultRecordArchivesScriptResult(t *testing.T) {
	setupDigitalStoreDeliveryReportFixture(t)
	record, err := DigitalStoreProfileService.CreateAcceptanceResultRecord(request.DigitalStoreAcceptanceResultCreateRequest{
		PublicBaseURL: "https://muse.example.com",
		Command:       "MUSE_ACCEPTANCE_TIMEOUT_MS=70000 scripts/run-muse-chat-acceptance.mjs",
		ScenarioTotal: 2,
		PassedTotal:   1,
		FailedTotal:   1,
		StartedAt:     "2026-07-06T10:00:00Z",
		FinishedAt:    "2026-07-06T10:01:00Z",
		Results: []request.DigitalStoreAcceptanceScenarioResultRequest{
			{Code: "M01", Title: "品牌介绍", Passed: true, Reason: "ok", ConversationID: 10, Reply: "慕斯寝具"},
			{Code: "M13", Title: "库存确认", Passed: false, Reason: "contains banned phrase", FailureType: "banned_phrase", Detail: "回复命中禁用承诺「一定有货」。", Suggestion: "检查库存口径。", ConversationID: 11, ConversationURL: "https://muse.example.com/dashboard/conversations?conversationId=11", Reply: "一定有货", MatchedBanned: "一定有货"},
		},
	}, &dto.AuthPrincipal{UserID: 8, Username: "acceptance"})
	if err != nil {
		t.Fatalf("CreateAcceptanceResultRecord() error = %v", err)
	}
	if record == nil || record.AcceptanceStatus != "failed" || record.ScenarioTotal != 2 || record.PassedTotal != 1 || record.FailedTotal != 1 {
		t.Fatalf("unexpected acceptance result record: %#v", record)
	}
	if record.AcceptanceStartedAt == "" || record.AcceptanceFinishedAt == "" {
		t.Fatalf("expected acceptance timestamps: %#v", record)
	}
	latest := DigitalStoreProfileService.GetLatestDeliveryRecord()
	if latest == nil || latest.ID != record.ID || !strings.Contains(latest.AcceptanceSummary, "1/2 通过") {
		t.Fatalf("unexpected latest acceptance record: %#v", latest)
	}
	if len(latest.AcceptanceResults) != 2 || latest.AcceptanceResults[1].FailureType != "banned_phrase" || latest.AcceptanceResults[1].ConversationURL == "" {
		t.Fatalf("expected archived acceptance diagnostics, got %#v", latest.AcceptanceResults)
	}
	report := DigitalStoreProfileService.GetDeliveryReport("https://muse.example.com")
	if report.LatestRecord == nil || report.LatestRecord.ID != record.ID || report.LatestRecord.FailedTotal != 1 {
		t.Fatalf("expected latest acceptance record in report, got %#v", report.LatestRecord)
	}
}

func TestDigitalStoreCleanupDemoDataClearsOperationalRecords(t *testing.T) {
	setupDigitalStoreDeliveryReportFixture(t)
	db := sqls.DB()
	now := time.Now()
	customer := &models.Customer{
		Name:          "真实客户",
		PrimaryMobile: "13800000000",
		Status:        enums.StatusOk,
	}
	if err := db.Create(customer).Error; err != nil {
		t.Fatalf("create customer: %v", err)
	}
	conversation := &models.Conversation{
		CustomerID:         customer.ID,
		CustomerName:       customer.Name,
		Status:             enums.IMConversationStatusAIServing,
		ServiceMode:        enums.IMConversationServiceModeAIFirst,
		LastMessageAt:      now,
		LastActiveAt:       now,
		LastMessageSummary: "测试会话",
	}
	if err := db.Create(conversation).Error; err != nil {
		t.Fatalf("create conversation: %v", err)
	}
	message := &models.Message{
		ConversationID: conversation.ID,
		ClientMsgID:    "demo-cleanup-msg",
		SenderType:     enums.IMSenderTypeCustomer,
		MessageType:    enums.IMMessageTypeText,
		Content:        "测试消息",
		SendStatus:     enums.IMMessageStatusSent,
		SentAt:         &now,
	}
	if err := db.Create(message).Error; err != nil {
		t.Fatalf("create message: %v", err)
	}
	workflowRun := &models.AIWorkflowRun{
		ConversationID: conversation.ID,
		MessageID:      message.ID,
		Status:         1,
		StartedAt:      now,
	}
	if err := db.Create(workflowRun).Error; err != nil {
		t.Fatalf("create workflow run: %v", err)
	}
	cleanupFixtures := []any{
		&models.SalesLead{CustomerID: customer.ID, ConversationID: conversation.ID, CustomerName: customer.Name, Phone: "13800000000", IntentLevel: enums.SalesLeadIntentHigh, Status: enums.SalesLeadStatusNew},
		&models.LeadFollowUp{LeadID: 1, OperatorName: "顾问", Content: "测试跟进", CreatedAt: now},
		&models.Ticket{TicketNo: "T-DEMO-001", Title: "测试工单", Source: enums.TicketSourceConversation, ConversationID: conversation.ID, CustomerID: customer.ID, Status: enums.TicketStatusPending},
		&models.TicketProgress{TicketID: 1, Content: "测试处理", CreatedAt: now},
		&models.TicketTag{TicketID: 1, TagID: 1},
		&models.Notification{RecipientUserID: 1, Title: "测试通知", Content: "测试", Status: enums.StatusOk, CreatedAt: now},
		&models.ConversationInterrupt{ConversationID: conversation.ID, SourceMessageID: message.ID, WorkflowRunID: workflowRun.ID, CheckPointID: "demo-checkpoint", Status: "waiting", CreatedAt: now, UpdatedAt: now},
		&models.ChannelMessageOutbox{ChannelType: "wxwork_kf", ConversationID: conversation.ID, MessageID: message.ID, SendStatus: "pending"},
		&models.WxWorkKFMessageRef{ConversationID: conversation.ID, MessageID: message.ID, WxMsgID: "demo-wx-msg", Direction: "in", Status: enums.StatusOk},
		&models.WxWorkKFConversation{ConversationID: conversation.ID, ChannelID: 1, OpenKfID: "kf_demo", ExternalUserID: "external_demo", Status: enums.StatusOk},
		&models.ConversationAssignment{ConversationID: conversation.ID, ToUserID: 1, AssignType: "auto", Status: enums.IMAssignmentStatusActive, CreatedAt: now},
		&models.ConversationTag{ConversationID: conversation.ID, TagID: 1},
		&models.ConversationEventLog{ConversationID: conversation.ID, EventType: enums.IMEventTypeCreate, OperatorType: enums.IMSenderTypeSystem, Content: "测试事件", CreatedAt: now},
		&models.ConversationReadState{ConversationID: conversation.ID, ReaderType: enums.IMSenderTypeAgent, ReaderID: 1, LastReadMessageID: message.ID},
		&models.ConversationParticipant{ConversationID: conversation.ID, ParticipantType: "customer", ParticipantID: customer.ID, Status: enums.StatusOk},
		&models.KnowledgeRetrieveLog{KnowledgeBaseID: 1, Channel: "im", Scene: "first_response", ConversationID: conversation.ID, Question: "测试问题", CreatedAt: now},
		&models.KnowledgeRetrieveHit{RetrieveLogID: 1, KnowledgeBaseID: 1, ChunkID: 1, CreatedAt: now},
		&models.KnowledgeFeedback{RetrieveLogID: 1, FeedbackType: 2, CreatedAt: now},
		&models.AIWorkflowNodeRun{WorkflowRunID: workflowRun.ID, NodeID: "reply", NodeType: "llm", Status: 1, StartedAt: now},
		&models.SkillRunLog{ConversationID: conversation.ID, AIAgentID: 1, UserMessage: "测试技能", CreatedAt: now},
		&models.DigitalStoreDeliveryRecord{BrandName: "慕斯寝具", StoreName: "徐汇体验店", Status: enums.StatusOk},
	}
	for _, item := range cleanupFixtures {
		if err := db.Create(item).Error; err != nil {
			t.Fatalf("create cleanup fixture %T: %v", item, err)
		}
	}

	resp, err := DigitalStoreProfileService.CleanupDemoData(&dto.AuthPrincipal{UserID: 9, Username: "delivery"})
	if err != nil {
		t.Fatalf("CleanupDemoData() error = %v", err)
	}
	for _, key := range []string{"messages", "conversations", "salesLeads", "tickets", "knowledgeRetrieveLogs", "aiWorkflowRuns"} {
		if resp.Deleted[key] == 0 {
			t.Fatalf("expected deleted count for %s, got %#v", key, resp.Deleted)
		}
	}
	assertTableCount(t, db, &models.Message{}, 0, "messages")
	assertTableCount(t, db, &models.Conversation{}, 0, "conversations")
	assertTableCount(t, db, &models.SalesLead{}, 0, "sales leads")
	assertTableCount(t, db, &models.Ticket{}, 0, "tickets")
	assertTableCount(t, db, &models.KnowledgeRetrieveLog{}, 0, "retrieve logs")
	assertTableCount(t, db, &models.AIWorkflowRun{}, 0, "workflow runs")
	assertTableCount(t, db, &models.Customer{}, 1, "customers")
	assertTableCount(t, db, &models.Product{}, 1, "products")
	assertTableCount(t, db, &models.Promotion{}, 1, "promotions")
	assertTableCount(t, db, &models.KnowledgeFAQ{}, 2, "knowledge faqs")
	assertTableCount(t, db, &models.DigitalStoreDeliveryRecord{}, 1, "delivery records")
	if !strings.Contains(resp.Message, "已清理") || resp.CleanedAt == "" {
		t.Fatalf("unexpected cleanup response: %#v", resp)
	}
}

func assertTableCount(t *testing.T, db *gorm.DB, model any, want int64, label string) {
	t.Helper()
	var got int64
	if err := db.Model(model).Count(&got).Error; err != nil {
		t.Fatalf("count %s: %v", label, err)
	}
	if got != want {
		t.Fatalf("expected %s count %d, got %d", label, want, got)
	}
}

func setupDigitalStoreDeliveryReportFixture(t *testing.T) {
	t.Helper()
	setupDigitalStoreRuntimeSetupTestDB(t)
	cfg := digitalStoreProfileConfig{
		DigitalStoreProfileRequest: request.DigitalStoreProfileRequest{
			BrandName:       "慕斯寝具",
			StoreName:       "徐汇体验店",
			AIManagerName:   "慕小眠",
			KnowledgeBaseID: 1,
			Initialized:     true,
		},
		KnowledgeFAQID: 2,
	}
	raw, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}
	if err := sqls.DB().Create(&models.SystemConfig{
		ConfigKey:   digitalStoreProfileConfigKey,
		ConfigValue: string(raw),
		GroupCode:   digitalStoreConfigGroup,
		Status:      enums.StatusOk,
	}).Error; err != nil {
		t.Fatalf("create config: %v", err)
	}
	productFAQ := &models.KnowledgeFAQ{
		KnowledgeBaseID: 1,
		Question:        "慕斯脊护支撑款",
		Answer:          "分区承托，适合腰背支撑需求。",
		IndexStatus:     enums.KnowledgeDocumentIndexStatusIndexed,
		Status:          enums.StatusOk,
	}
	if err := sqls.DB().Create(productFAQ).Error; err != nil {
		t.Fatalf("create product faq: %v", err)
	}
	promotionFAQ := &models.KnowledgeFAQ{
		KnowledgeBaseID: 1,
		Question:        "周末预约试躺礼",
		Answer:          "周末预约到店可领取试躺礼。",
		IndexStatus:     enums.KnowledgeDocumentIndexStatusIndexed,
		Status:          enums.StatusOk,
	}
	if err := sqls.DB().Create(promotionFAQ).Error; err != nil {
		t.Fatalf("create promotion faq: %v", err)
	}
	if err := sqls.DB().Create(&models.Product{Name: "慕斯脊护支撑款", KnowledgeBaseID: 1, KnowledgeFAQID: productFAQ.ID, Status: enums.StatusOk}).Error; err != nil {
		t.Fatalf("create product: %v", err)
	}
	if err := sqls.DB().Create(&models.Promotion{Name: "周末预约试躺礼", KnowledgeBaseID: 1, KnowledgeFAQID: promotionFAQ.ID, Status: enums.StatusOk}).Error; err != nil {
		t.Fatalf("create promotion: %v", err)
	}
	if err := sqls.DB().Create(&models.AIConfig{
		Name:      "DeepSeek",
		Provider:  enums.AIProviderOpenAI,
		ModelType: enums.AIModelTypeLLM,
		ModelName: "deepseek-v4-flash",
		Status:    enums.StatusOk,
	}).Error; err != nil {
		t.Fatalf("create ai config: %v", err)
	}
	if err := sqls.DB().Create(&models.AIConfig{
		Name:      "OpenAI Embedding",
		Provider:  enums.AIProviderOpenAI,
		ModelType: enums.AIModelTypeEmbedding,
		ModelName: "text-embedding-3-small",
		Dimension: 1536,
		Status:    enums.StatusOk,
	}).Error; err != nil {
		t.Fatalf("create embedding config: %v", err)
	}
	if err := sqls.DB().Create(&models.User{
		Username: "consultant",
		Nickname: "慕斯顾问",
		Status:   enums.StatusOk,
	}).Error; err != nil {
		t.Fatalf("create consultant user: %v", err)
	}
	team := &models.AgentTeam{
		Name:         digitalStoreDefaultTeamName,
		LeaderUserID: 1,
		Status:       enums.StatusOk,
		Remark:       digitalStoreRuntimeSeedRemark,
	}
	if err := sqls.DB().Create(team).Error; err != nil {
		t.Fatalf("create agent team: %v", err)
	}
	if err := sqls.DB().Create(&models.AgentProfile{
		UserID:                1,
		TeamID:                team.ID,
		AgentCode:             "muse_consultant",
		DisplayName:           "慕斯顾问",
		ServiceStatus:         enums.ServiceStatusIdle,
		MaxConcurrentCount:    10,
		PriorityLevel:         10,
		AutoAssignEnabled:     true,
		ReceiveOfflineMessage: true,
		Status:                enums.StatusOk,
	}).Error; err != nil {
		t.Fatalf("create agent profile: %v", err)
	}
	now := time.Now()
	if err := sqls.DB().Create(&models.AgentTeamSchedule{
		TeamID:  team.ID,
		StartAt: now.Add(-time.Hour),
		EndAt:   now.Add(24 * time.Hour),
		Status:  enums.StatusOk,
		Remark:  digitalStoreRuntimeSeedRemark,
	}).Error; err != nil {
		t.Fatalf("create agent team schedule: %v", err)
	}
	if err := sqls.DB().Create(&models.AIAgent{
		Name:              "慕小眠 AI数字店长",
		Status:            enums.StatusOk,
		WorkflowVersionID: 10,
		TeamIDs:           utils.JoinInt64s([]int64{team.ID}),
	}).Error; err != nil {
		t.Fatalf("create agent: %v", err)
	}
	webChannelConfig, err := defaultDigitalStoreWebChannelConfig(cfg)
	if err != nil {
		t.Fatalf("build web channel config: %v", err)
	}
	if err := sqls.DB().Create(&models.Channel{
		Name:        "慕斯寝具官网客服",
		ChannelType: enums.ChannelTypeWeb,
		ChannelID:   "web_muse_test",
		ConfigJSON:  webChannelConfig,
		Status:      enums.StatusOk,
		Remark:      digitalStoreRuntimeSeedRemark,
	}).Error; err != nil {
		t.Fatalf("create channel: %v", err)
	}
}

func setupDigitalStoreRuntimeInstructionTestDB(t *testing.T) {
	t.Helper()
	dbName := strings.NewReplacer("/", "_", " ", "_").Replace(t.Name())
	db, err := gorm.Open(sqlite.Open("file:"+dbName+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			_ = sqlDB.Close()
		}
	})
	if err := db.AutoMigrate(&models.SystemConfig{}, &models.Product{}, &models.Promotion{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	sqls.SetDB(db)
}

func setupDigitalStoreRuntimeSetupTestDB(t *testing.T) {
	t.Helper()
	dbName := strings.NewReplacer("/", "_", " ", "_").Replace(t.Name())
	db, err := gorm.Open(sqlite.Open("file:"+dbName+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			_ = sqlDB.Close()
		}
	})
	if err := db.AutoMigrate(
		&models.SystemConfig{},
		&models.User{},
		&models.AgentTeam{},
		&models.AgentProfile{},
		&models.AgentTeamSchedule{},
		&models.Customer{},
		&models.SalesLead{},
		&models.LeadFollowUp{},
		&models.Conversation{},
		&models.ConversationParticipant{},
		&models.ConversationReadState{},
		&models.Message{},
		&models.WxWorkKFConversation{},
		&models.WxWorkKFMessageRef{},
		&models.ChannelMessageOutbox{},
		&models.ConversationAssignment{},
		&models.ConversationTag{},
		&models.ConversationEventLog{},
		&models.ConversationInterrupt{},
		&models.Ticket{},
		&models.TicketProgress{},
		&models.TicketTag{},
		&models.Notification{},
		&models.Product{},
		&models.Promotion{},
		&models.AIConfig{},
		&models.AIAgent{},
		&models.AIWorkflow{},
		&models.AIWorkflowVersion{},
		&models.AIWorkflowRun{},
		&models.AIWorkflowNodeRun{},
		&models.KnowledgeBase{},
		&models.KnowledgeFAQ{},
		&models.KnowledgeRetrieveLog{},
		&models.KnowledgeRetrieveHit{},
		&models.KnowledgeFeedback{},
		&models.SkillRunLog{},
		&models.Channel{},
		&models.DigitalStoreDeliveryRecord{},
	); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	sqls.SetDB(db)
}
