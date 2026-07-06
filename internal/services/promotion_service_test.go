package services

import (
	"strings"
	"testing"
	"time"

	"agent-desk/internal/models"
)

func TestBuildPromotionKnowledgeFAQContent(t *testing.T) {
	start := time.Date(2026, 7, 1, 0, 0, 0, 0, time.Local)
	end := time.Date(2026, 7, 31, 0, 0, 0, 0, time.Local)
	promotion := &models.Promotion{
		ID:                 7,
		Name:               "周末预约试躺礼",
		PromotionType:      "预约权益",
		Description:        "提前预约周末试躺",
		ApplicableProducts: "慕斯脊护支撑款",
		StartAt:            &start,
		EndAt:              &end,
		DiscountRule:       "成交价到店确认",
		StoreBenefit:       "免费睡眠咨询",
		AppointmentBenefit: "护睡礼包",
		ScriptSuggestion:   "引导客户留下手机号预约",
	}

	question, answer, similar, remark := BuildPromotionKnowledgeFAQContent(promotion)
	if question != "活动优惠：周末预约试躺礼" {
		t.Fatalf("unexpected question: %s", question)
	}
	for _, want := range []string{"活动类型：预约权益", "有效期：2026-07-01 至 2026-07-31", "预约权益：护睡礼包", "推荐话术：引导客户留下手机号预约"} {
		if !strings.Contains(answer, want) {
			t.Fatalf("answer missing %q: %s", want, answer)
		}
	}
	if len(similar) == 0 {
		t.Fatal("similar questions should not be empty")
	}
	if remark != "promotion:7" {
		t.Fatalf("unexpected remark: %s", remark)
	}
}

func TestParsePromotionCSVAndBuildImportRequest(t *testing.T) {
	input := "\ufeff活动名称,活动类型,活动描述,适用产品,开始时间,结束时间,优惠规则,到店权益,预约权益,话术建议,推荐优先级,状态\n" +
		"周末预约试躺礼,预约权益,提前预约周末试躺,慕斯脊护支撑款,2026-07-01,2026-07-31,成交价到店确认,免费睡眠咨询,护睡礼包,引导客户留下手机号预约,90,启用\n" +
		"\n"
	rows, err := parsePromotionCSV(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parsePromotionCSV() error = %v", err)
	}
	if len(rows) != 1 || rows[0].Row != 2 {
		t.Fatalf("unexpected rows: %#v", rows)
	}
	req, err := buildPromotionImportRequest(rows[0].Values)
	if err != nil {
		t.Fatalf("buildPromotionImportRequest() error = %v", err)
	}
	if req.Name != "周末预约试躺礼" || req.PromotionType != "预约权益" || req.Priority != 90 || req.Status != 0 {
		t.Fatalf("unexpected import request: %#v", req)
	}
	if req.StartAt != "2026-07-01 00:00:00" || req.EndAt != "2026-07-31 23:59:59" {
		t.Fatalf("unexpected date range: %s - %s", req.StartAt, req.EndAt)
	}
}

func TestBuildPromotionImportRequestRejectsInvalidDate(t *testing.T) {
	_, err := buildPromotionImportRequest(map[string]string{
		"name":    "周末预约试躺礼",
		"startAt": "2026年7月1日",
	})
	if err == nil || !strings.Contains(err.Error(), "开始时间格式需为") {
		t.Fatalf("expected invalid date error, got %v", err)
	}
}

func TestPromotionTemplateSeedsSupportsOralClinic(t *testing.T) {
	seeds, err := promotionTemplateSeeds("oral_clinic", time.Date(2026, 7, 1, 12, 0, 0, 0, time.Local))
	if err != nil {
		t.Fatalf("promotionTemplateSeeds() error = %v", err)
	}
	if len(seeds) != 2 {
		t.Fatalf("expected 2 oral clinic promotion seeds, got %d", len(seeds))
	}
	if seeds[0].Name != "正畸初诊评估预约礼" {
		t.Fatalf("unexpected first oral clinic promotion: %#v", seeds[0])
	}
	if !strings.Contains(seeds[0].DiscountRule, "医生评估") {
		t.Fatalf("oral clinic promotion should keep medical confirmation boundary: %s", seeds[0].DiscountRule)
	}
}

func TestPromotionTemplateSeedsSupportIndustryMarketTemplates(t *testing.T) {
	now := time.Date(2026, 7, 1, 12, 0, 0, 0, time.Local)
	for _, tc := range []struct {
		code         string
		name         string
		typeName     string
		boundaryText string
		scriptKey    string
	}{
		{"kids_english", "少儿英语试听测评预约礼", "试听权益", "最终学费", "试听测评"},
		{"finance_advisor", "金融顾问合规初评预约", "咨询预约", "收益承诺", "持牌顾问"},
		{"home_decoration", "免费量房与设计咨询季", "量房预约", "最终报价", "预约设计师"},
	} {
		t.Run(tc.code, func(t *testing.T) {
			seeds, err := promotionTemplateSeeds(tc.code, now)
			if err != nil {
				t.Fatalf("promotionTemplateSeeds() error = %v", err)
			}
			if len(seeds) != 1 {
				t.Fatalf("expected one promotion seed, got %d", len(seeds))
			}
			if seeds[0].Name != tc.name || seeds[0].PromotionType != tc.typeName {
				t.Fatalf("unexpected promotion seed: %#v", seeds[0])
			}
			if !strings.Contains(seeds[0].DiscountRule, tc.boundaryText) {
				t.Fatalf("promotion should include boundary %q: %s", tc.boundaryText, seeds[0].DiscountRule)
			}
			if !strings.Contains(seeds[0].ScriptSuggestion, tc.scriptKey) && !strings.Contains(seeds[0].AppointmentBenefit, tc.scriptKey) {
				t.Fatalf("promotion should include script key %q: %#v", tc.scriptKey, seeds[0])
			}
		})
	}
}
