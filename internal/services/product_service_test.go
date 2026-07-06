package services

import (
	"strings"
	"testing"

	"agent-desk/internal/models"
)

func TestBuildProductKnowledgeFAQContent(t *testing.T) {
	product := &models.Product{
		ID:                 42,
		Name:               "慕斯脊护支撑款",
		Category:           "床垫",
		PriceMin:           12000,
		PriceMax:           18000,
		SellingPoints:      "分区承托、偏硬支撑",
		SuitablePeople:     "腰背压力明显的人群",
		Scenarios:          "老人房",
		Specs:              "1.8m",
		IndustryAttributes: "睡感：偏硬；支撑：分区承托",
	}

	question, answer, similar, remark := BuildProductKnowledgeFAQContent(product)
	if question != "产品推荐：慕斯脊护支撑款" {
		t.Fatalf("unexpected question: %s", question)
	}
	for _, want := range []string{"价格区间：12000-18000元", "核心卖点：分区承托、偏硬支撑", "适合人群：腰背压力明显的人群", "行业扩展属性：睡感：偏硬", "推荐话术"} {
		if !strings.Contains(answer, want) {
			t.Fatalf("answer missing %q: %s", want, answer)
		}
	}
	if len(similar) == 0 {
		t.Fatal("similar questions should not be empty")
	}
	if remark != "product:42" {
		t.Fatalf("unexpected remark: %s", remark)
	}
}

func TestParseProductCSVAndBuildImportRequest(t *testing.T) {
	input := "\ufeff产品名称,品类,最低价,最高价,核心卖点,适合人群,行业属性,推荐优先级,状态\n" +
		"慕斯脊护支撑款,床垫,12000,18000,分区承托,老人腰背压力明显,睡感：偏硬,90,启用\n" +
		"\n"
	rows, err := parseProductCSV(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parseProductCSV() error = %v", err)
	}
	if len(rows) != 1 || rows[0].Row != 2 {
		t.Fatalf("unexpected rows: %#v", rows)
	}
	req, err := buildProductImportRequest(rows[0].Values)
	if err != nil {
		t.Fatalf("buildProductImportRequest() error = %v", err)
	}
	if req.Name != "慕斯脊护支撑款" || req.Category != "床垫" || req.PriceMin != 12000 || req.PriceMax != 18000 || req.Priority != 90 {
		t.Fatalf("unexpected import request: %#v", req)
	}
	if req.IndustryAttributes != "睡感：偏硬" {
		t.Fatalf("expected industry attributes parsed, got %#v", req)
	}
}

func TestBuildProductImportRequestRejectsInvalidNumber(t *testing.T) {
	_, err := buildProductImportRequest(map[string]string{
		"name":     "慕斯脊护支撑款",
		"priceMin": "一万",
	})
	if err == nil || !strings.Contains(err.Error(), "最低价必须是整数") {
		t.Fatalf("expected invalid number error, got %v", err)
	}
}

func TestProductTemplateSeedsSupportsOralClinic(t *testing.T) {
	seeds, faqs, remark, err := productTemplateSeeds("oral_clinic")
	if err != nil {
		t.Fatalf("productTemplateSeeds() error = %v", err)
	}
	if len(seeds) < 4 {
		t.Fatalf("expected oral clinic product seeds, got %d", len(seeds))
	}
	if len(faqs) == 0 {
		t.Fatal("expected oral clinic FAQ seeds")
	}
	if remark != "oral-clinic-guide-seed" {
		t.Fatalf("unexpected remark: %s", remark)
	}
	if seeds[0].Name != "隐形矫正初诊评估" {
		t.Fatalf("unexpected first oral clinic seed: %#v", seeds[0])
	}
	if !strings.Contains(seeds[0].IndustryAttributes, "诊疗项目") {
		t.Fatalf("oral clinic seed should include industry attributes: %#v", seeds[0])
	}
	if !strings.Contains(faqs[0].answer, "不得承诺治疗效果") {
		t.Fatalf("oral clinic FAQ should include compliance boundary: %s", faqs[0].answer)
	}
}

func TestProductTemplateSeedsSupportIndustryMarketTemplates(t *testing.T) {
	for _, tc := range []struct {
		code        string
		firstName   string
		attribute   string
		faqBoundary string
		remark      string
	}{
		{"kids_english", "自然拼读进阶班", "班型", "不得承诺保过", "kids-english-guide-seed"},
		{"finance_advisor", "经营贷资质初评", "禁用", "不得索要银行卡密码", "finance-advisor-guide-seed"},
		{"home_decoration", "全案设计咨询", "面积", "不得承诺一口价", "home-decoration-guide-seed"},
	} {
		t.Run(tc.code, func(t *testing.T) {
			seeds, faqs, remark, err := productTemplateSeeds(tc.code)
			if err != nil {
				t.Fatalf("productTemplateSeeds() error = %v", err)
			}
			if len(seeds) == 0 || len(faqs) == 0 {
				t.Fatalf("expected product and FAQ seeds: products=%d faqs=%d", len(seeds), len(faqs))
			}
			if seeds[0].Name != tc.firstName {
				t.Fatalf("unexpected first product seed: %#v", seeds[0])
			}
			if !strings.Contains(seeds[0].IndustryAttributes, tc.attribute) {
				t.Fatalf("seed should include industry attribute %q: %#v", tc.attribute, seeds[0])
			}
			if !strings.Contains(faqs[0].answer, tc.faqBoundary) && !strings.Contains(faqs[len(faqs)-1].answer, tc.faqBoundary) {
				t.Fatalf("FAQ seeds should include boundary %q: %#v", tc.faqBoundary, faqs)
			}
			if remark != tc.remark {
				t.Fatalf("unexpected remark: %s", remark)
			}
		})
	}
}
