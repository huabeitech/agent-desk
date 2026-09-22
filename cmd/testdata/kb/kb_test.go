package kb

import (
	"agent-desk/cmd/testdata/seedlang"
	"agent-desk/cmd/testdata/seeds"
	"agent-desk/internal/models"
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/mlogclub/simple/sqls"
	"gorm.io/gorm"
)

var hanTextPattern = regexp.MustCompile(`\p{Han}`)

func TestEnglishKnowledgeBaseTextDoesNotContainChineseText(t *testing.T) {
	seed := seeds.FAQKnowledgeBaseSeed(seedlang.English)
	for _, value := range []string{seed.Name, seed.Description, seed.Remark} {
		if hanTextPattern.MatchString(value) {
			t.Fatalf("english knowledge base text contains Chinese text: %q", value)
		}
	}
}

func TestEnglishKnowledgeFAQSeedsDoNotContainChineseText(t *testing.T) {
	seedItems := seeds.KnowledgeFAQSeeds(seedlang.English)
	if len(seedItems) == 0 {
		t.Fatal("english FAQ seeds are empty")
	}
	for _, seed := range seedItems {
		values := []string{seed.Question, seed.Answer, seed.Remark}
		values = append(values, seed.SimilarQuestions...)
		for _, value := range values {
			if hanTextPattern.MatchString(value) {
				t.Fatalf("english FAQ seed contains Chinese text: %q", value)
			}
		}
	}
}

func TestInitCroveDeskKB(t *testing.T) {
	dbName := fmt.Sprintf("file:memdb_kb_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dbName), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open sqlite memory db: %v", err)
	}
	if err := db.AutoMigrate(models.KnowledgeBase{}, models.KnowledgeFAQ{}, models.KnowledgeChunk{}); err != nil {
		t.Fatalf("failed to auto migrate: %v", err)
	}
	sqls.SetDB(db)

	res, err := InitCroveDeskKB()
	if err != nil {
		t.Fatalf("InitCroveDeskKB failed: %v", err)
	}
	if res.KnowledgeBaseID <= 0 {
		t.Fatalf("expected positive KnowledgeBaseID, got %d", res.KnowledgeBaseID)
	}
	if res.CreatedFAQs != res.TotalFAQs {
		t.Fatalf("expected createdFAQs == %d, got %d", res.TotalFAQs, res.CreatedFAQs)
	}
}
