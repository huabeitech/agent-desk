package services

import (
	"agent-desk/internal/models"
	"agent-desk/internal/pkg/dto"
	"agent-desk/internal/pkg/dto/request"
	"agent-desk/internal/pkg/enums"
	"agent-desk/internal/repositories"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/mlogclub/simple/sqls"
	"gorm.io/gorm"
)

func TestDocPageHierarchy(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+strings.ReplaceAll(t.Name(), "/", "_")+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.DocPage{}); err != nil {
		t.Fatalf("migrate help page: %v", err)
	}
	sqls.SetDB(db)
	operator := &dto.AuthPrincipal{UserID: 1, Username: "admin"}

	root, err := SupportService.SaveDocPage(request.SaveDocPageRequest{
		Title: "Getting Started", Slug: "getting-started", ContentType: "markdown", Content: "# Getting Started", Status: enums.DocPageStatusPublished,
	}, operator)
	if err != nil {
		t.Fatalf("create root page: %v", err)
	}
	child, err := SupportService.SaveDocPage(request.SaveDocPageRequest{
		ParentID: root.ID, Title: "Install", Slug: "install", ContentType: "markdown", Content: "# Install", Status: enums.DocPageStatusPublished,
	}, operator)
	if err != nil {
		t.Fatalf("create child page under a content page: %v", err)
	}
	secondChild, err := SupportService.SaveDocPage(request.SaveDocPageRequest{
		ParentID: root.ID, Title: "Configure", Slug: "configure", ContentType: "markdown", Content: "# Configure", Status: enums.DocPageStatusDraft,
	}, operator)
	if err != nil {
		t.Fatalf("create second child page: %v", err)
	}
	if err := SupportService.SortDocPages(request.SortDocPagesRequest{
		ParentID: root.ID,
		IDs:      []int64{secondChild.ID, child.ID},
	}); err != nil {
		t.Fatalf("sort sibling pages: %v", err)
	}
	sorted := SupportService.FindDocPages(sqls.NewCnd().Eq("parent_id", root.ID).Asc("sort_no"))
	if len(sorted) != 2 || sorted[0].ID != secondChild.ID || sorted[1].ID != child.ID {
		t.Fatalf("unexpected sorted pages: %#v", sorted)
	}
	if err := SupportService.SortDocPages(request.SortDocPagesRequest{
		ParentID: root.ID,
		IDs:      []int64{child.ID},
	}); err == nil {
		t.Fatal("expected incomplete sibling sort to fail")
	}
	if _, err := SupportService.ChangeDocPageStatus(request.ChangeDocPageStatusRequest{ID: root.ID, Status: enums.DocPageStatusDraft}, operator); err == nil {
		t.Fatal("expected withdrawing a parent with published children to fail")
	}
	if _, err := SupportService.ChangeDocPageStatus(request.ChangeDocPageStatusRequest{ID: child.ID, Status: enums.DocPageStatusDraft}, operator); err != nil {
		t.Fatalf("withdraw child page: %v", err)
	}
	if _, err := SupportService.ChangeDocPageStatus(request.ChangeDocPageStatusRequest{ID: root.ID, Status: enums.DocPageStatusDraft}, operator); err != nil {
		t.Fatalf("withdraw root page: %v", err)
	}
	if _, err := SupportService.ChangeDocPageStatus(request.ChangeDocPageStatusRequest{ID: child.ID, Status: enums.DocPageStatusPublished}, operator); err == nil {
		t.Fatal("expected publishing a child under a draft parent to fail")
	}
	if _, err := SupportService.ChangeDocPageStatus(request.ChangeDocPageStatusRequest{ID: root.ID, Status: enums.DocPageStatusPublished}, operator); err != nil {
		t.Fatalf("publish root page: %v", err)
	}
	if _, err := SupportService.ChangeDocPageStatus(request.ChangeDocPageStatusRequest{ID: child.ID, Status: enums.DocPageStatusPublished}, operator); err != nil {
		t.Fatalf("publish child page: %v", err)
	}

	root.ParentID = child.ID
	_, err = SupportService.SaveDocPage(request.SaveDocPageRequest{
		ID: root.ID, ParentID: child.ID, Title: root.Title, Slug: root.Slug, ContentType: root.ContentType, Content: root.Content, Status: root.Status,
	}, operator)
	if err == nil {
		t.Fatal("expected cycle validation error")
	}
	if err := SupportService.DeleteDocPage(root.ID); err == nil {
		t.Fatal("expected deleting a page with children to fail")
	}
	if err := SupportService.DeleteDocPage(child.ID); err != nil {
		t.Fatalf("delete leaf page: %v", err)
	}
	if err := SupportService.DeleteDocPage(secondChild.ID); err != nil {
		t.Fatalf("delete second leaf page: %v", err)
	}
}

func TestFindPublicDocNavigationSelectsPublishedMetadataInTreeOrder(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+strings.ReplaceAll(t.Name(), "/", "_")+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.DocPage{}); err != nil {
		t.Fatalf("migrate help page: %v", err)
	}
	sqls.SetDB(db)
	items := []*models.DocPage{
		{ParentID: 0, Title: "Install", Slug: "install", Content: "install content", Status: enums.DocPageStatusPublished, SortNo: 0},
		{ParentID: 0, Title: "Overview", Slug: "overview", Content: "overview content", Status: enums.DocPageStatusPublished, SortNo: 1},
		{ParentID: 2, Title: "Change log", Slug: "changelog", Content: "change log content", Status: enums.DocPageStatusPublished, SortNo: 0},
		{ParentID: 0, Title: "Draft", Slug: "draft", Content: "draft content", Status: enums.DocPageStatusDraft, SortNo: 0},
	}
	for _, item := range items {
		if err := db.Create(item).Error; err != nil {
			t.Fatalf("create help page: %v", err)
		}
	}

	list := SupportService.FindPublicDocNavigation()
	if len(list) != 3 || list[0].Slug != "install" || list[1].Slug != "changelog" || list[2].Slug != "overview" {
		t.Fatalf("unexpected navigation rows: %#v", list)
	}
	if list[0].Content != "" || list[1].Content != "" || list[2].Content != "" {
		t.Fatalf("navigation query must not load article content: %#v", list)
	}
}

func TestSupportSlugAllowsLettersNumbersAndHyphens(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+strings.ReplaceAll(t.Name(), "/", "_")+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.DocPage{}, &models.Category{}); err != nil {
		t.Fatalf("migrate support models: %v", err)
	}
	sqls.SetDB(db)
	operator := &dto.AuthPrincipal{UserID: 1, Username: "admin"}

	page, err := SupportService.SaveDocPage(request.SaveDocPageRequest{
		Title: "Release notes", Slug: "release-2026-08", ContentType: "markdown", Status: enums.DocPageStatusDraft,
	}, operator)
	if err != nil {
		t.Fatalf("save slug containing hyphens: %v", err)
	}
	if page.Slug != "release-2026-08" {
		t.Fatalf("unexpected saved slug: %q", page.Slug)
	}
	category, err := SupportService.SaveCategory(request.SaveCategoryRequest{
		Name: "Release notes", Slug: "release-2026-08",
	}, operator)
	if err != nil {
		t.Fatalf("save category slug containing hyphens: %v", err)
	}
	if category.Slug != "release-2026-08" {
		t.Fatalf("unexpected saved category slug: %q", category.Slug)
	}
	if _, err := SupportService.SaveDocPage(request.SaveDocPageRequest{
		Title: "Invalid slug", Slug: "release_notes", ContentType: "markdown", Status: enums.DocPageStatusDraft,
	}, operator); err == nil {
		t.Fatal("expected underscore slug to fail validation")
	}
	if err := db.Create(&models.DocPage{ParentID: 0, Title: "Root", Slug: "shared-slug"}).Error; err != nil {
		t.Fatalf("create first root slug: %v", err)
	}
	if err := db.Create(&models.DocPage{ParentID: 1, Title: "Child", Slug: "shared-slug"}).Error; err != nil {
		t.Fatalf("allow the same slug under another parent: %v", err)
	}
	if err := db.Create(&models.DocPage{ParentID: 0, Title: "Duplicate root", Slug: "shared-slug"}).Error; err == nil {
		t.Fatal("expected duplicate slug under the same parent to fail")
	}
}

func TestUpdateDocPageSettingsOnlyUpdatesSettingsFields(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+strings.ReplaceAll(t.Name(), "/", "_")+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.DocPage{}); err != nil {
		t.Fatalf("migrate help page: %v", err)
	}
	sqls.SetDB(db)
	operator := &dto.AuthPrincipal{UserID: 1, Username: "admin"}
	parent, err := SupportService.SaveDocPage(request.SaveDocPageRequest{
		Title: "Parent", Slug: "parent", ContentType: "markdown", Content: "parent content", Status: enums.DocPageStatusDraft,
	}, operator)
	if err != nil {
		t.Fatalf("create parent: %v", err)
	}
	page, err := SupportService.SaveDocPage(request.SaveDocPageRequest{
		Title: "Original title", Slug: "original", Summary: "original summary", ContentType: "markdown", Content: "original content", Tags: []string{"original"}, Status: enums.DocPageStatusDraft,
	}, operator)
	if err != nil {
		t.Fatalf("create page: %v", err)
	}
	saved, err := SupportService.UpdateDocPageSettings(request.UpdateDocPageSettingsRequest{
		ID: page.ID, ParentID: parent.ID, Slug: "updated", Summary: "updated summary",
	}, operator)
	if err != nil {
		t.Fatalf("update page settings: %v", err)
	}
	if saved.ParentID != parent.ID || saved.Slug != "updated" || saved.Summary != "updated summary" {
		t.Fatalf("settings were not updated: %#v", saved)
	}
	if saved.Title != page.Title || saved.Content != page.Content || saved.ContentType != page.ContentType || saved.TagsJSON != page.TagsJSON || saved.Status != page.Status {
		t.Fatalf("document fields changed while saving settings: %#v", saved)
	}
}

func TestPostAndCommentContentType(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+strings.ReplaceAll(t.Name(), "/", "_")+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.Post{}, &models.Comment{}, &models.Category{}); err != nil {
		t.Fatalf("migrate support post models: %v", err)
	}
	sqls.SetDB(db)
	principal := &dto.AuthPrincipal{UserID: 1, Username: "customer"}

	post, err := SupportService.CreatePost(request.CreatePostRequest{
		CategoryID:  1,
		Title:       "Rich text post",
		ContentType: "html",
		Content:     "<p>Hello</p>",
	}, principal)
	if err != nil {
		t.Fatalf("create post: %v", err)
	}
	if post.ContentType != "html" {
		t.Fatalf("post content type = %q, want html", post.ContentType)
	}
	comment, err := SupportService.CreateCustomerComment(request.CreateCommentRequest{
		PostID:      post.ID,
		ContentType: "markdown",
		Content:     "**Comment**",
	}, principal)
	if err != nil {
		t.Fatalf("create comment: %v", err)
	}
	if comment.ContentType != "markdown" {
		t.Fatalf("comment content type = %q, want markdown", comment.ContentType)
	}
	defaultPost, err := SupportService.CreatePost(request.CreatePostRequest{
		CategoryID: 1,
		Title:      "Default post",
		Content:    "plain content",
	}, principal)
	if err != nil {
		t.Fatalf("create default post: %v", err)
	}
	if defaultPost.ContentType != "markdown" {
		t.Fatalf("default post content type = %q, want markdown", defaultPost.ContentType)
	}
}

func TestCommentDiscussionWorkflow(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+strings.ReplaceAll(t.Name(), "/", "_")+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.Post{}, &models.Comment{}, &models.Reaction{}, &models.CommentReport{}); err != nil {
		t.Fatalf("migrate support comment models: %v", err)
	}
	sqls.SetDB(db)
	owner := &dto.AuthPrincipal{UserID: 1, Username: "owner"}
	commenter := &dto.AuthPrincipal{UserID: 2, Username: "commenter"}

	post, err := SupportService.CreatePost(request.CreatePostRequest{
		CategoryID: 1,
		Title:      "Discussion post",
		Content:    "post body",
	}, owner)
	if err != nil {
		t.Fatalf("create post: %v", err)
	}
	if SupportService.HasReaction(enums.ReactionTargetPost, post.ID, enums.ReactionTypeLike, commenter) {
		t.Fatal("new post should not be liked by the commenter")
	}
	if err := SupportService.ToggleReaction(enums.ReactionTargetPost, post.ID, enums.ReactionTypeLike, commenter); err != nil {
		t.Fatalf("like post: %v", err)
	}
	if !SupportService.HasReaction(enums.ReactionTargetPost, post.ID, enums.ReactionTypeLike, commenter) {
		t.Fatal("post should be liked by the commenter")
	}
	if err := SupportService.ToggleReaction(enums.ReactionTargetPost, post.ID, enums.ReactionTypeLike, commenter); err != nil {
		t.Fatalf("unlike post: %v", err)
	}
	if SupportService.HasReaction(enums.ReactionTargetPost, post.ID, enums.ReactionTypeLike, commenter) {
		t.Fatal("post should not be liked after toggling again")
	}
	comment, err := SupportService.CreateCustomerComment(request.CreateCommentRequest{
		PostID:  post.ID,
		Content: "top level comment",
	}, commenter)
	if err != nil {
		t.Fatalf("create comment: %v", err)
	}
	reply, err := SupportService.CreateCustomerComment(request.CreateCommentRequest{
		PostID:   post.ID,
		ParentID: comment.ID,
		Content:  "reply",
	}, owner)
	if err != nil {
		t.Fatalf("create reply: %v", err)
	}
	if reply.ParentID != comment.ID {
		t.Fatalf("reply parent id = %d, want %d", reply.ParentID, comment.ID)
	}
	list, err := SupportService.ListPostComments(post.ID, 0, "default", 1, 20)
	if err != nil {
		t.Fatalf("list comments: %v", err)
	}
	if len(list.Comments) != 1 || len(list.Replies[comment.ID]) != 1 {
		t.Fatalf("expected top-level comment with one preview reply, got %#v replies=%#v", list.Comments, list.Replies)
	}
	if list.Paging.Total != 1 {
		t.Fatalf("top-level paging total = %d, want 1", list.Paging.Total)
	}
	if err := SupportService.UpdateComment(request.UpdateCommentRequest{ID: comment.ID, ContentType: "markdown", Content: "updated"}, commenter); err != nil {
		t.Fatalf("update own comment: %v", err)
	}
	if err := SupportService.ReportComment(request.ReportCommentRequest{ID: comment.ID, Reason: "spam"}, owner); err != nil {
		t.Fatalf("report comment: %v", err)
	}
	if err := SupportService.ReportComment(request.ReportCommentRequest{ID: comment.ID, Reason: "spam again"}, owner); err != nil {
		t.Fatalf("report comment twice: %v", err)
	}
	updated := repositories.CommentRepository.Get(sqls.DB(), comment.ID)
	if updated.Content != "updated" || updated.ReportCount != 1 || updated.ReplyCount != 1 {
		t.Fatalf("unexpected updated comment: %#v", updated)
	}
	if err := SupportService.DeleteComment(reply.ID, owner); err != nil {
		t.Fatalf("delete reply: %v", err)
	}
	updated = repositories.CommentRepository.Get(sqls.DB(), comment.ID)
	if updated.ReplyCount != 1 {
		t.Fatalf("reply count after soft delete = %d, want 1", updated.ReplyCount)
	}
	replies, err := SupportService.ListPostComments(post.ID, comment.ID, "default", 1, 20)
	if err != nil {
		t.Fatalf("list replies after delete: %v", err)
	}
	if len(replies.Comments) != 1 || replies.Comments[0].Status != enums.CommentStatusDeleted {
		t.Fatalf("expected deleted reply placeholder, got %#v", replies.Comments)
	}
}

func TestCategorySort(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+strings.ReplaceAll(t.Name(), "/", "_")+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.Category{}); err != nil {
		t.Fatalf("migrate category: %v", err)
	}
	sqls.SetDB(db)
	categories := []*models.Category{
		{Name: "First", Slug: "first", SortNo: 0, Status: enums.StatusOk},
		{Name: "Second", Slug: "second", SortNo: 1, Status: enums.StatusOk},
		{Name: "Third", Slug: "third", SortNo: 2, Status: enums.StatusOk},
	}
	for _, category := range categories {
		if err := db.Create(category).Error; err != nil {
			t.Fatalf("create category: %v", err)
		}
	}
	if err := SupportService.UpdateCategorySort([]int64{categories[2].ID, categories[0].ID, categories[1].ID}); err != nil {
		t.Fatalf("sort categories: %v", err)
	}
	sorted := repositories.CategoryRepository.Find(sqls.DB(), sqls.NewCnd().Asc("sort_no"))
	if len(sorted) != 3 || sorted[0].ID != categories[2].ID || sorted[1].ID != categories[0].ID || sorted[2].ID != categories[1].ID {
		t.Fatalf("unexpected sorted categories: %#v", sorted)
	}
}
