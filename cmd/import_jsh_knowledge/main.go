// Package main 是一次性数据导入工具：将本地 jsh-knowledge 目录按层级
// 导入 t_knowledge_directory / t_knowledge_document。
//
// 用法：
//
//	go run ./cmd/import_jsh_knowledge -kb 3 -root ./jsh-knowledge
//
// 工具可重复执行（幂等）：已存在的同名目录会复用，同目录下同名文档会跳过。
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"agent-desk/internal/ai/rag"
	"agent-desk/internal/models"
	"agent-desk/internal/pkg/enums"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

// frontmatterRe 匹配文件开头的 YAML frontmatter（---\n ... \n---\n）。
var frontmatterRe = regexp.MustCompile(`(?s)\A---\r?\n.*?\r?\n---\r?\n?`)

// leadingNumberRe 提取目录名开头的数字前缀（如 "02-合作平台" -> 2）。
var leadingNumberRe = regexp.MustCompile(`^\d+`)

type options struct {
	dsn              string
	root             string
	kbID             int64
	stripFrontmatter bool
	dryRun           bool
}

func main() {
	opt := options{}
	flag.StringVar(&opt.dsn, "dsn", "root:123456abcde@tcp(127.0.0.1:3306)/cs_ai_agent?charset=utf8mb4&parseTime=True&loc=Local", "MySQL DSN")
	flag.StringVar(&opt.root, "root", "jsh-knowledge", "待导入的知识目录根路径")
	flag.Int64Var(&opt.kbID, "kb", 3, "目标知识库 ID（t_knowledge_base.id）")
	flag.BoolVar(&opt.stripFrontmatter, "strip-frontmatter", true, "是否剥离 Markdown 开头的 YAML frontmatter")
	flag.BoolVar(&opt.dryRun, "dry-run", false, "只演练不写库")
	flag.Parse()

	if err := run(opt); err != nil {
		slog.Error("import failed", "error", err)
		os.Exit(1)
	}
}

func run(opt options) error {
	entries, err := scanRoot(opt.root)
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		return fmt.Errorf("目录 %s 下没有可导入的子目录", opt.root)
	}

	db, err := gorm.Open(mysql.Open(opt.dsn), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{TablePrefix: "t_", SingularTable: true},
		Logger: logger.New(
			log.New(os.Stdout, "\r\n", log.LstdFlags),
			logger.Config{
				SlowThreshold:             time.Second,
				LogLevel:                  logger.Warn,
				IgnoreRecordNotFoundError: true,
			},
		),
	})
	if err != nil {
		return fmt.Errorf("连接数据库失败: %w", err)
	}

	// 校验目标知识库存在。
	var kb models.KnowledgeBase
	if err := db.Select("id", "name", "knowledge_type", "status").
		First(&kb, opt.kbID).Error; err != nil {
		return fmt.Errorf("知识库 id=%d 不存在或不可读: %w", opt.kbID, err)
	}
	slog.Info("目标知识库", "id", kb.ID, "name", kb.Name, "type", kb.KnowledgeType, "status", kb.Status)

	var dirCreated, dirReused, docCreated, docSkipped int
	now := time.Now()

	for idx, entry := range entries {
		sortNo := parseLeadingNumber(entry.name)
		if sortNo == 0 {
			sortNo = idx + 1
		}

		// 每个分类目录一个事务：目录行与其下文档要么全部落库要么整体回滚。
		var dirID int64
		var dirNew bool
		var n importCounters
		err := db.Transaction(func(tx *gorm.DB) error {
			var terr error
			dirID, dirNew, terr = findOrCreateDirectory(tx, opt, entry.name, sortNo, now)
			if terr != nil {
				return terr
			}
			n, terr = importDocuments(tx, opt, dirID, entry.files, now)
			return terr
		})
		if err != nil {
			return fmt.Errorf("目录 %s 处理失败: %w", entry.name, err)
		}
		if dirNew {
			dirCreated++
		} else {
			dirReused++
		}
		docCreated += n.created
		docSkipped += n.skipped
		slog.Info("目录完成",
			"name", entry.name, "dir_id", dirID, "new_dir", dirNew,
			"new_docs", n.created, "skipped_docs", n.skipped)
	}

	slog.Info("导入完成",
		"dry_run", opt.dryRun,
		"directories_created", dirCreated,
		"directories_reused", dirReused,
		"documents_created", docCreated,
		"documents_skipped", docSkipped)
	return nil
}

type dirEntry struct {
	name  string
	files []string
}

func scanRoot(root string) ([]dirEntry, error) {
	top, err := os.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("读取根目录失败: %w", err)
	}
	var out []dirEntry
	for _, d := range top {
		if !d.IsDir() {
			continue
		}
		full := filepath.Join(root, d.Name())
		files, err := collectMarkdownFiles(full)
		if err != nil {
			return nil, err
		}
		if len(files) == 0 {
			slog.Warn("目录下没有 md 文件，已跳过", "dir", full)
			continue
		}
		out = append(out, dirEntry{name: d.Name(), files: files})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].name < out[j].name })
	return out, nil
}

// collectMarkdownFiles 递归收集 .md 文件（当前数据只有一层，递归以兼容未来嵌套）。
func collectMarkdownFiles(dir string) ([]string, error) {
	var files []string

	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if path != dir {
				// 当前导入器只支持一层分类目录；出现嵌套时显式告警，
				// 避免文件被静默平铺到错误的目录下。
				slog.Warn("发现嵌套子目录，其内文件将被平铺到一级目录（如需多层请扩展导入器）", "subdir", path)
			}
			return nil
		}
		if strings.EqualFold(filepath.Ext(d.Name()), ".md") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	return files, nil
}

func parseLeadingNumber(name string) int {
	m := leadingNumberRe.FindString(name)
	if m == "" {
		return 0
	}
	n, err := strconv.Atoi(m)
	if err != nil {
		return 0
	}
	return n
}

// findOrCreateDirectory 在同一知识库/同一父级(0)下按名称查找目录，没有则创建。
func findOrCreateDirectory(db *gorm.DB, opt options, name string, sortNo int, now time.Time) (int64, bool, error) {
	var existing models.KnowledgeDirectory
	err := db.Where("knowledge_base_id = ? AND parent_id = 0 AND name = ?", opt.kbID, name).
		First(&existing).Error
	if err == nil {
		return existing.ID, false, nil
	}
	if err != gorm.ErrRecordNotFound {
		return 0, false, err
	}

	dir := &models.KnowledgeDirectory{
		KnowledgeBaseID: opt.kbID,
		ParentID:        0,
		Name:            name,
		SortNo:          sortNo,
		Status:          enums.StatusOk,
		AuditFields:     buildAuditFields(now),
	}
	if opt.dryRun {
		slog.Info("[dry-run] 将创建目录", "name", name, "sort_no", sortNo)
		return int64(sortNo), true, nil
	}
	if err := db.Create(dir).Error; err != nil {
		return 0, false, err
	}
	return dir.ID, true, nil
}

type importCounters struct {
	created int
	skipped int
}

func importDocuments(db *gorm.DB, opt options, dirID int64, files []string, now time.Time) (importCounters, error) {
	var c importCounters
	for _, path := range files {
		title := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))

		var existing models.KnowledgeDocument
		err := db.Where("knowledge_base_id = ? AND directory_id = ? AND title = ?", opt.kbID, dirID, title).
			First(&existing).Error
		if err == nil {
			c.skipped++
			continue
		}
		if err != gorm.ErrRecordNotFound {
			return c, err
		}

		raw, err := os.ReadFile(path)
		if err != nil {
			return c, fmt.Errorf("读取文件 %s 失败: %w", path, err)
		}
		content := string(raw)
		if opt.stripFrontmatter {
			content = frontmatterRe.ReplaceAllString(content, "")
		}
		content = strings.TrimSpace(content)
		if content == "" {
			slog.Warn("文档内容为空，仍将导入", "file", path)
		}

		doc := &models.KnowledgeDocument{
			KnowledgeBaseID: opt.kbID,
			DirectoryID:     dirID,
			Title:           title,
			ContentType:     enums.KnowledgeDocumentContentTypeMarkdown,
			Content:         content,
			Status:          enums.StatusOk,
			IndexStatus:     enums.KnowledgeDocumentIndexStatusPending,
			AuditFields:     buildAuditFields(now),
		}
		// 与正常建文档流程保持一致：hash 取正文纯文本的 SHA-256。
		if plain := rag.ExtractPlainText(content, enums.KnowledgeDocumentContentTypeMarkdown); plain != "" {
			sum := sha256.Sum256([]byte(plain))
			doc.ContentHash = hex.EncodeToString(sum[:])
		}

		if opt.dryRun {
			c.created++
			continue
		}
		if err := db.Create(doc).Error; err != nil {
			return c, fmt.Errorf("写入文档 %s 失败: %w", path, err)
		}
		c.created++
	}
	return c, nil
}

func buildAuditFields(now time.Time) models.AuditFields {
	return models.AuditFields{
		CreatedAt:      now,
		CreateUserID:   0,
		CreateUserName: "system",
		UpdatedAt:      now,
		UpdateUserID:   0,
		UpdateUserName: "system",
	}
}
