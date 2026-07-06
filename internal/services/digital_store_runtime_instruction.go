package services

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"agent-desk/internal/models"
	"agent-desk/internal/pkg/enums"

	"github.com/mlogclub/simple/sqls"
)

const digitalStoreRuntimeInstructionTitle = "AI数字店长运行上下文"

func (s *digitalStoreProfileService) BuildRuntimeInstruction() (ret string) {
	defer func() {
		if r := recover(); r != nil {
			slog.Debug("skip digital store runtime instruction", "recover", r)
			ret = ""
		}
	}()
	if sqls.DB() == nil {
		return ""
	}
	cfg := s.loadConfig()
	if !hasDigitalStoreRuntimeConfig(cfg) && !hasDigitalStoreRuntimeCatalog() {
		return ""
	}
	sections := make([]string, 0, 5)
	if section := buildDigitalStoreProfileRuntimeSection(cfg); section != "" {
		sections = append(sections, section)
	}
	if section := buildDigitalStoreProductRuntimeSection(); section != "" {
		sections = append(sections, section)
	}
	if section := buildDigitalStorePromotionRuntimeSection(time.Now()); section != "" {
		sections = append(sections, section)
	}
	sections = append(sections, buildDigitalStoreSafetyGuardrailRuntimeSection(cfg))
	sections = append(sections, buildDigitalStoreGuideRuntimeRules(cfg))
	return digitalStoreRuntimeInstructionTitle + "：\n" + strings.Join(sections, "\n\n")
}

func hasDigitalStoreRuntimeConfig(cfg digitalStoreProfileConfig) bool {
	return strings.TrimSpace(cfg.BrandName) != "" ||
		strings.TrimSpace(cfg.StoreName) != "" ||
		strings.TrimSpace(cfg.AIManagerName) != "" ||
		strings.TrimSpace(cfg.AIPersona) != ""
}

func hasDigitalStoreRuntimeCatalog() bool {
	var count int64
	if err := sqls.DB().Model(&models.Product{}).Where("status = ?", enums.StatusOk).Count(&count).Error; err == nil && count > 0 {
		return true
	}
	count = 0
	now := time.Now()
	if err := sqls.DB().Model(&models.Promotion{}).
		Where("status = ?", enums.StatusOk).
		Where("(start_at IS NULL OR start_at <= ?)", now).
		Where("(end_at IS NULL OR end_at >= ?)", now).
		Count(&count).Error; err != nil {
		return false
	}
	return count > 0
}

func buildDigitalStoreProfileRuntimeSection(cfg digitalStoreProfileConfig) string {
	lines := make([]string, 0, 9)
	appendRuntimeLine := func(label, value string) {
		value = strings.TrimSpace(value)
		if value != "" {
			lines = append(lines, label+"："+value)
		}
	}
	appendRuntimeLine("品牌", cfg.BrandName)
	appendRuntimeLine("行业", cfg.Industry)
	appendRuntimeLine("门店", cfg.StoreName)
	appendRuntimeLine("地址", cfg.StoreAddress)
	appendRuntimeLine("营业时间", cfg.BusinessHours)
	appendRuntimeLine("联系电话", cfg.ContactPhone)
	appendRuntimeLine("客服微信", cfg.ServiceWeChat)
	appendRuntimeLine("店长人设", cfg.AIPersona)
	appendRuntimeLine("回复风格", cfg.ReplyStyle)
	if len(lines) == 0 {
		return ""
	}
	return "店铺资料：\n" + strings.Join(lines, "\n")
}

func buildDigitalStoreProductRuntimeSection() string {
	var products []models.Product
	if err := sqls.DB().
		Where("status = ?", enums.StatusOk).
		Order("priority DESC, id DESC").
		Limit(5).
		Find(&products).Error; err != nil || len(products) == 0 {
		return ""
	}
	lines := make([]string, 0, len(products))
	for _, item := range products {
		parts := []string{strings.TrimSpace(item.Name)}
		if category := strings.TrimSpace(item.Category); category != "" {
			parts = append(parts, "分类 "+category)
		}
		if price := digitalStoreRuntimePriceText(item.PriceMin, item.PriceMax); price != "" {
			parts = append(parts, "价格 "+price)
		}
		if points := digitalStoreRuntimeLimit(item.SellingPoints, 80); points != "" {
			parts = append(parts, "卖点 "+points)
		}
		if people := digitalStoreRuntimeLimit(item.SuitablePeople, 60); people != "" {
			parts = append(parts, "适合 "+people)
		}
		lines = append(lines, "- "+strings.Join(parts, "；"))
	}
	return "主推产品参考（推荐时优先结合知识库证据，不确定库存/最低价需转人工确认）：\n" + strings.Join(lines, "\n")
}

func buildDigitalStorePromotionRuntimeSection(now time.Time) string {
	var promotions []models.Promotion
	if err := sqls.DB().
		Where("status = ?", enums.StatusOk).
		Where("(start_at IS NULL OR start_at <= ?)", now).
		Where("(end_at IS NULL OR end_at >= ?)", now).
		Order("priority DESC, id DESC").
		Limit(3).
		Find(&promotions).Error; err != nil || len(promotions) == 0 {
		return ""
	}
	lines := make([]string, 0, len(promotions))
	for _, item := range promotions {
		parts := []string{strings.TrimSpace(item.Name)}
		if products := strings.TrimSpace(item.ApplicableProducts); products != "" {
			parts = append(parts, "适用 "+products)
		}
		if rule := digitalStoreRuntimeLimit(item.DiscountRule, 70); rule != "" {
			parts = append(parts, "优惠 "+rule)
		}
		if benefit := digitalStoreRuntimeLimit(item.AppointmentBenefit, 70); benefit != "" {
			parts = append(parts, "预约权益 "+benefit)
		}
		if script := digitalStoreRuntimeLimit(item.ScriptSuggestion, 70); script != "" {
			parts = append(parts, "话术 "+script)
		}
		lines = append(lines, "- "+strings.Join(parts, "；"))
	}
	return "当前有效活动：\n" + strings.Join(lines, "\n")
}

func buildDigitalStoreGuideRuntimeRules(cfg digitalStoreProfileConfig) string {
	lines := []string{
		"先直接回答客户核心问题，再给1-3个推荐方案，说明适合原因和差异。",
		"推荐时主动追问关键成交信息：使用人群、睡眠/使用痛点、尺寸、预算、城市/门店、到店时间。",
		"客户出现购买、预算、联系方式、预约、到店、询价等信号时，要自然引导留下姓名、手机号或微信，并说明会安排顾问跟进。",
		"客户明确要人工、要求最终成交价/库存/配送安装确认、投诉售后、高意向预约时，应建议转人工。",
	}
	if appointment := strings.TrimSpace(cfg.AppointmentPolicy); appointment != "" {
		lines = append(lines, "预约规则："+appointment)
	}
	if handoff := strings.TrimSpace(cfg.HandoffPolicy); handoff != "" {
		lines = append(lines, "转人工规则："+handoff)
	}
	if forbidden := strings.TrimSpace(cfg.ForbiddenClaims); forbidden != "" {
		lines = append(lines, "禁止承诺："+forbidden)
	}
	return "导购回复规则：\n" + strings.Join(lines, "\n")
}

func buildDigitalStoreSafetyGuardrailRuntimeSection(cfg digitalStoreProfileConfig) string {
	lines := []string{
		"价格：只能引用产品库、活动库或知识库中明确出现的价格区间/活动规则；不得承诺最低价、保底价、最终成交价、额外折扣或私自叠加优惠；客户追问成交价时引导留资或转人工确认。",
		"库存：库存是实时信息，除非资料明确写有库存，否则不得说现货、有货、可直接提货、常规尺寸都有；只能说明库存需要门店顾问实时确认。",
		"疗效/效果：不得承诺治疗疾病、治好疼痛、百分百改善睡眠、无痛、一次解决、绝对成功等结果；可建议结合专业检查、试躺体验或到店/到院评估。",
		"绝对承诺：不得使用一定、保证、百分百、必然、永久、最低、最便宜、全网最低等绝对化表达；不确定时明确说明需要人工确认。",
		"退款/退货/售后：不得自行承诺退款、退货、换货、赔付、上门时间、安装时效或保修结论；应说明需依据商家售后政策和订单信息由顾问/售后确认。",
		"资质/合规：不得虚构品牌授权、医生/专家资质、检测证书、排班、案例、活动名额或服务范围；未在资料中出现的内容不要编造。",
	}
	if forbidden := strings.TrimSpace(cfg.ForbiddenClaims); forbidden != "" {
		lines = append(lines, "商家自定义禁用承诺："+forbidden)
	}
	for _, rule := range buildDigitalStoreIndustryRiskRuleResponses(cfg) {
		if rule.Key == "common" {
			continue
		}
		if len(rule.ForbiddenClaims) > 0 {
			lines = append(lines, rule.Label+"禁用承诺："+strings.Join(rule.ForbiddenClaims, "；"))
		}
		if len(rule.HandoffTriggers) > 0 {
			lines = append(lines, rule.Label+"转人工触发："+strings.Join(rule.HandoffTriggers, "；"))
		}
	}
	return "AI 回复安全护栏：\n" + strings.Join(lines, "\n")
}

func digitalStoreRuntimePriceText(minPrice int64, maxPrice int64) string {
	if minPrice > 0 && maxPrice > 0 {
		return fmt.Sprintf("%d-%d元", minPrice, maxPrice)
	}
	if maxPrice > 0 {
		return fmt.Sprintf("%d元左右", maxPrice)
	}
	if minPrice > 0 {
		return fmt.Sprintf("%d元以上", minPrice)
	}
	return ""
}

func digitalStoreRuntimeLimit(value string, maxRunes int) string {
	value = strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
	if value == "" || maxRunes <= 0 {
		return value
	}
	runes := []rune(value)
	if len(runes) <= maxRunes {
		return value
	}
	return string(runes[:maxRunes]) + "..."
}
