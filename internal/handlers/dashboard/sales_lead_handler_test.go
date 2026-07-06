package dashboard

import (
	"strings"
	"testing"
	"time"

	"agent-desk/internal/models"
)

func TestSalesLeadExportIncludesMergeExplanation(t *testing.T) {
	mergedAt := time.Date(2026, 7, 6, 10, 30, 0, 0, time.Local)
	lead := models.SalesLead{
		ID:           12,
		CustomerName: "王先生",
		MergeKey:     "phone",
		MergeReason:  "手机号 13800001111 命中活跃线索 #12，跨会话合并到该线索。",
		MergedAt:     &mergedAt,
	}

	headers := strings.Join(salesLeadExportHeaders(), ",")
	if !strings.Contains(headers, "归并方式") || !strings.Contains(headers, "归并说明") || !strings.Contains(headers, "归并时间") {
		t.Fatalf("export headers should include merge explanation fields: %s", headers)
	}

	row := strings.Join(salesLeadExportRow(lead, map[int64]string{}), "\n")
	if !strings.Contains(row, "同手机号") || !strings.Contains(row, "手机号 13800001111") || !strings.Contains(row, "2026-07-06 10:30:00") {
		t.Fatalf("export row should include merge explanation values: %s", row)
	}
}
