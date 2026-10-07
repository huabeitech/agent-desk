package services

import (
	"agent-desk/internal/models"
	"agent-desk/internal/pkg/config"
	"agent-desk/internal/pkg/dto"
	"agent-desk/internal/pkg/dto/request"
	"agent-desk/internal/pkg/dto/response"
	"agent-desk/internal/pkg/enums"
	"agent-desk/internal/pkg/errorsx"
	"agent-desk/internal/repositories"
	"encoding/json"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mlogclub/simple/sqls"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var supportSlugPattern = regexp.MustCompile(`^[a-z0-9-]+$`)

const (
	supportUserContextKey = "supportUser"
)

var SupportService = &supportService{}

type supportService struct{}

type CommentListResult struct {
	Comments []models.Comment
	Replies  map[int64][]models.Comment
	Paging   *sqls.Paging
}

// CommunityResponseData contains the shared, batch-loaded relations needed by
// community response builders. It keeps handlers and builders free of per-row
// repository lookups.
type CommunityResponseData struct {
	Users      map[int64]*models.User
	Categories map[int64]*models.Category
}

func mapKeys(values map[int64]struct{}) []int64 {
	keys := make([]int64, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	return keys
}

func (s *supportService) LoadCommunityResponseData(posts []models.Post, comments []models.Comment, replies map[int64][]models.Comment) CommunityResponseData {
	userIDs := make(map[int64]struct{})
	categoryIDs := make(map[int64]struct{})
	for _, post := range posts {
		if post.UserID > 0 {
			userIDs[post.UserID] = struct{}{}
		}
		if post.CategoryID > 0 {
			categoryIDs[post.CategoryID] = struct{}{}
		}
	}
	collectCommentUserIDs := func(items []models.Comment) {
		for _, item := range items {
			if item.AuthorID > 0 {
				userIDs[item.AuthorID] = struct{}{}
			}
		}
	}
	collectCommentUserIDs(comments)
	for _, items := range replies {
		collectCommentUserIDs(items)
	}

	users := repositories.UserRepository.FindSimpleInfoByIDs(sqls.DB(), mapKeys(userIDs))
	categories := repositories.CategoryRepository.FindByIDs(sqls.DB(), mapKeys(categoryIDs))
	data := CommunityResponseData{
		Users:      make(map[int64]*models.User, len(users)),
		Categories: make(map[int64]*models.Category, len(categories)),
	}
	for i := range users {
		data.Users[users[i].ID] = &users[i]
	}
	for i := range categories {
		data.Categories[categories[i].ID] = &categories[i]
	}
	return data
}

func (s *supportService) RegisterUser(req request.SupportCustomerRegisterRequest, authCfg config.AuthConfig, clientIP, userAgent string) (*response.LoginResponse, error) {
	name := strings.TrimSpace(req.Name)
	email := normalizeSupportEmail(req.Email)
	password := strings.TrimSpace(req.Password)
	if name == "" || email == "" || len(password) < 8 {
		return nil, errorsx.InvalidParam("name, email and at least 8 characters password are required")
	}
	if repositories.UserRepository.GetByUsername(sqls.DB(), email) != nil || repositories.UserRepository.GetByEmail(sqls.DB(), email) != nil {
		return nil, errorsx.InvalidParam("email is already registered")
	}
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	user := &models.User{
		Username:     email,
		Nickname:     name,
		Email:        &email,
		Password:     string(passwordHash),
		PasswordSalt: "",
		UserType:     enums.UserTypeUser,
		Status:       enums.StatusOk,
		AuditFields:  supportAuditFields(0, name, now),
	}
	if err := sqls.WithTransaction(func(ctx *sqls.TxContext) error {
		if err := repositories.UserRepository.Create(ctx.Tx, user); err != nil {
			return err
		}
		return repositories.CustomerRepository.Create(ctx.Tx, &models.Customer{
			UserID:       user.ID,
			Name:         name,
			PrimaryEmail: email,
			Status:       enums.StatusOk,
			AuditFields:  supportAuditFields(user.ID, name, now),
		})
	}); err != nil {
		return nil, err
	}
	return AuthService.Login(request.LoginRequest{Username: email, Password: password}, authCfg, clientIP, userAgent)
}

func (s *supportService) RequireSupportUser(ctx *gin.Context) (*dto.AuthPrincipal, error) {
	if principal := s.GetSupportUser(ctx); principal != nil {
		return principal, nil
	}
	principal, err := AuthService.Authenticate(ctx)
	if err != nil {
		return nil, err
	}
	ctx.Set(supportUserContextKey, principal)
	return principal, nil
}

func (s *supportService) OptionalSupportUser(ctx *gin.Context) (*dto.AuthPrincipal, error) {
	if principal := s.GetSupportUser(ctx); principal != nil {
		return principal, nil
	}
	if ctx == nil || (strings.TrimSpace(ctx.GetHeader("Authorization")) == "" && strings.TrimSpace(ctx.Query("accessToken")) == "") {
		return nil, nil
	}
	return s.RequireSupportUser(ctx)
}

func (s *supportService) GetSupportUser(ctx *gin.Context) *dto.AuthPrincipal {
	if ctx == nil {
		return nil
	}
	value, _ := ctx.Get(supportUserContextKey)
	if principal, ok := value.(*dto.AuthPrincipal); ok {
		return principal
	}
	return nil
}

func (s *supportService) FindDocPages(cnd *sqls.Cnd) []models.DocPage {
	return repositories.DocPageRepository.Find(sqls.DB(), cnd)
}

func (s *supportService) FindPublicDocNavigation() []models.DocPage {
	return repositories.DocPageRepository.FindNavigationItems(sqls.DB(), sqls.NewCnd().Eq("status", enums.DocPageStatusPublished).Asc("sort_no").Asc("id"))
}

func (s *supportService) FindDocPageByID(id int64) *models.DocPage {
	return repositories.DocPageRepository.Get(sqls.DB(), id)
}

func (s *supportService) FindDocPagePage(cnd *sqls.Cnd) ([]models.DocPage, *sqls.Paging) {
	return repositories.DocPageRepository.FindPageByCnd(sqls.DB(), cnd)
}

func (s *supportService) SortDocPages(req request.SortDocPagesRequest) error {
	pages := repositories.DocPageRepository.Find(
		sqls.DB(),
		sqls.NewCnd().Eq("parent_id", req.ParentID).Asc("sort_no").Asc("id"),
	)
	if len(req.IDs) != len(pages) {
		return errorsx.InvalidParamI18n("error.docPage.sortScopeMismatch")
	}
	existing := make(map[int64]struct{}, len(pages))
	for _, page := range pages {
		existing[page.ID] = struct{}{}
	}
	seen := make(map[int64]struct{}, len(req.IDs))
	for _, id := range req.IDs {
		if _, ok := existing[id]; !ok {
			return errorsx.InvalidParamI18n("error.docPage.sortScopeMismatch")
		}
		if _, ok := seen[id]; ok {
			return errorsx.InvalidParamI18n("error.docPage.sortDuplicate")
		}
		seen[id] = struct{}{}
	}
	return sqls.WithTransaction(func(ctx *sqls.TxContext) error {
		for sortNo, id := range req.IDs {
			if err := repositories.DocPageRepository.UpdateSort(ctx.Tx, id, sortNo); err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *supportService) validateDocPageParent(id, parentID int64) error {
	if parentID == 0 {
		return nil
	}
	if id > 0 && id == parentID {
		return errorsx.InvalidParam("page cannot be its own parent")
	}
	parent := repositories.DocPageRepository.Get(sqls.DB(), parentID)
	if parent == nil {
		return errorsx.InvalidParam("parent page not found")
	}
	depth := 2
	visited := map[int64]bool{parentID: true}
	for parent.ParentID > 0 {
		if parent.ParentID == id || visited[parent.ParentID] {
			return errorsx.InvalidParam("page hierarchy contains a cycle")
		}
		visited[parent.ParentID] = true
		parent = repositories.DocPageRepository.Get(sqls.DB(), parent.ParentID)
		if parent == nil {
			return errorsx.InvalidParam("parent page not found")
		}
		depth++
		if depth > 4 {
			return errorsx.InvalidParam("page hierarchy supports at most four levels")
		}
	}
	return nil
}

func (s *supportService) SaveDocPage(req request.SaveDocPageRequest, operator *dto.AuthPrincipal) (*models.DocPage, error) {
	title, slug := strings.TrimSpace(req.Title), normalizeSupportSlug(req.Slug)
	if title == "" || slug == "" {
		return nil, errorsx.InvalidParam("title and slug are required")
	}
	if err := validateSupportSlug(slug); err != nil {
		return nil, err
	}
	if err := s.validateDocPageParent(req.ID, req.ParentID); err != nil {
		return nil, err
	}
	if existing := repositories.DocPageRepository.GetByParentIDAndSlug(sqls.DB(), req.ParentID, slug); existing != nil && existing.ID != req.ID {
		return nil, errorsx.InvalidParam("page slug already exists")
	}
	status := normalizeDocPageStatus(req.Status)
	var current *models.DocPage
	if req.ID > 0 {
		current = repositories.DocPageRepository.Get(sqls.DB(), req.ID)
		if current == nil {
			return nil, errorsx.InvalidParam("page not found")
		}
		// Publication status is changed only by ChangeDocPageStatus so saving
		// content cannot accidentally publish or withdraw a public document.
		status = current.Status
	}
	if status == enums.DocPageStatusPublished && req.ParentID > 0 {
		parent := repositories.DocPageRepository.Get(sqls.DB(), req.ParentID)
		if parent == nil || parent.Status != enums.DocPageStatusPublished {
			return nil, errorsx.InvalidParam("publish the parent page first")
		}
	}
	if req.ID > 0 && status != enums.DocPageStatusPublished {
		publishedChildren := repositories.DocPageRepository.Find(sqls.DB(), sqls.NewCnd().Eq("parent_id", req.ID).Eq("status", enums.DocPageStatusPublished).Page(1, 1))
		if len(publishedChildren) > 0 {
			return nil, errorsx.InvalidParam("unpublish child pages first")
		}
	}
	tags, _ := json.Marshal(normalizeTags(req.Tags))
	now := time.Now()
	publishedAt := (*time.Time)(nil)
	if status == enums.DocPageStatusPublished && req.ID == 0 {
		publishedAt = &now
	}
	columns := map[string]any{"parent_id": req.ParentID, "title": title, "slug": slug, "summary": strings.TrimSpace(req.Summary), "content_type": normalizeContentType(req.ContentType), "content": req.Content, "cover_url": strings.TrimSpace(req.CoverURL), "tags_json": string(tags), "status": status, "sort_no": req.SortNo, "remark": strings.TrimSpace(req.Remark), "updated_at": now, "update_user_id": operator.UserID, "update_user_name": operator.Username}
	if publishedAt != nil {
		columns["published_at"] = publishedAt
	}
	if req.ID > 0 {
		if err := repositories.DocPageRepository.Updates(sqls.DB(), req.ID, columns); err != nil {
			return nil, err
		}
		return repositories.DocPageRepository.Get(sqls.DB(), req.ID), nil
	}
	item := &models.DocPage{ParentID: req.ParentID, Title: title, Slug: slug, Summary: strings.TrimSpace(req.Summary), ContentType: normalizeContentType(req.ContentType), Content: req.Content, CoverURL: strings.TrimSpace(req.CoverURL), TagsJSON: string(tags), Status: status, SortNo: req.SortNo, PublishedAt: publishedAt, Remark: strings.TrimSpace(req.Remark), AuditFields: auditFieldsFromOperator(operator, now)}
	if err := repositories.DocPageRepository.Create(sqls.DB(), item); err != nil {
		return nil, err
	}
	return item, nil
}

func (s *supportService) UpdateDocPageSettings(req request.UpdateDocPageSettingsRequest, operator *dto.AuthPrincipal) (*models.DocPage, error) {
	item := repositories.DocPageRepository.Get(sqls.DB(), req.ID)
	if item == nil {
		return nil, errorsx.InvalidParamI18n("error.docPage.notFound")
	}
	slug := normalizeSupportSlug(req.Slug)
	if slug == "" {
		return nil, errorsx.InvalidParamI18n("error.docPage.slugRequired")
	}
	if err := validateSupportSlug(slug); err != nil {
		return nil, err
	}
	if err := s.validateDocPageParent(item.ID, req.ParentID); err != nil {
		return nil, err
	}
	if existing := repositories.DocPageRepository.GetByParentIDAndSlug(sqls.DB(), req.ParentID, slug); existing != nil && existing.ID != item.ID {
		return nil, errorsx.InvalidParamI18n("error.docPage.slugDuplicate")
	}
	if item.Status == enums.DocPageStatusPublished && req.ParentID > 0 {
		parent := repositories.DocPageRepository.Get(sqls.DB(), req.ParentID)
		if parent == nil || parent.Status != enums.DocPageStatusPublished {
			return nil, errorsx.InvalidParamI18n("error.docPage.publishParentFirst")
		}
	}
	sortNo := item.SortNo
	if req.ParentID != item.ParentID {
		sortNo = len(repositories.DocPageRepository.Find(sqls.DB(), sqls.NewCnd().Eq("parent_id", req.ParentID)))
	}
	now := time.Now()
	columns := map[string]any{
		"parent_id":        req.ParentID,
		"slug":             slug,
		"summary":          strings.TrimSpace(req.Summary),
		"sort_no":          sortNo,
		"updated_at":       now,
		"update_user_id":   operator.UserID,
		"update_user_name": operator.Username,
	}
	if err := repositories.DocPageRepository.Updates(sqls.DB(), item.ID, columns); err != nil {
		return nil, err
	}
	return repositories.DocPageRepository.Get(sqls.DB(), item.ID), nil
}

func (s *supportService) ChangeDocPageStatus(req request.ChangeDocPageStatusRequest, operator *dto.AuthPrincipal) (*models.DocPage, error) {
	item := repositories.DocPageRepository.Get(sqls.DB(), req.ID)
	if item == nil {
		return nil, errorsx.InvalidParamI18n("error.docPage.notFound")
	}
	status := req.Status
	if status != enums.DocPageStatusDraft && status != enums.DocPageStatusPublished && status != enums.DocPageStatusHidden {
		return nil, errorsx.InvalidParamI18n("error.docPage.invalidStatus")
	}
	if status == item.Status {
		return item, nil
	}
	if status == enums.DocPageStatusPublished && item.ParentID > 0 {
		parent := repositories.DocPageRepository.Get(sqls.DB(), item.ParentID)
		if parent == nil || parent.Status != enums.DocPageStatusPublished {
			return nil, errorsx.InvalidParamI18n("error.docPage.publishParentFirst")
		}
	}
	if status != enums.DocPageStatusPublished {
		publishedChildren := repositories.DocPageRepository.Find(sqls.DB(), sqls.NewCnd().Eq("parent_id", item.ID).Eq("status", enums.DocPageStatusPublished).Page(1, 1))
		if len(publishedChildren) > 0 {
			return nil, errorsx.InvalidParamI18n("error.docPage.unpublishChildrenFirst")
		}
	}
	now := time.Now()
	columns := map[string]any{
		"status":           status,
		"updated_at":       now,
		"update_user_id":   operator.UserID,
		"update_user_name": operator.Username,
	}
	if status == enums.DocPageStatusPublished {
		columns["published_at"] = now
	}
	if err := repositories.DocPageRepository.Updates(sqls.DB(), item.ID, columns); err != nil {
		return nil, err
	}
	return repositories.DocPageRepository.Get(sqls.DB(), item.ID), nil
}

func (s *supportService) DeleteDocPage(id int64) error {
	if repositories.DocPageRepository.Get(sqls.DB(), id) == nil {
		return errorsx.InvalidParam("page not found")
	}
	children := repositories.DocPageRepository.Find(sqls.DB(), sqls.NewCnd().Eq("parent_id", id).Page(1, 1))
	if len(children) > 0 {
		return errorsx.InvalidParam("page still has child pages")
	}
	return repositories.DocPageRepository.Delete(sqls.DB(), id)
}

func (s *supportService) SaveCategory(req request.SaveCategoryRequest, operator *dto.AuthPrincipal) (*models.Category, error) {
	name, slug := strings.TrimSpace(req.Name), normalizeSupportSlug(req.Slug)
	if name == "" || slug == "" {
		return nil, errorsx.InvalidParam("name and slug are required")
	}
	if err := validateSupportSlug(slug); err != nil {
		return nil, err
	}
	now := time.Now()
	if req.ID > 0 {
		if repositories.CategoryRepository.Get(sqls.DB(), req.ID) == nil {
			return nil, errorsx.InvalidParam("category not found")
		}
		if err := repositories.CategoryRepository.Updates(sqls.DB(), req.ID, map[string]any{"name": name, "slug": slug, "description": strings.TrimSpace(req.Description), "status": req.Status, "remark": strings.TrimSpace(req.Remark), "update_user_id": operator.UserID, "update_user_name": operator.Username, "updated_at": now}); err != nil {
			return nil, err
		}
		return repositories.CategoryRepository.Get(sqls.DB(), req.ID), nil
	}
	item := &models.Category{Name: name, Slug: slug, Description: strings.TrimSpace(req.Description), Status: req.Status, Remark: strings.TrimSpace(req.Remark), AuditFields: auditFieldsFromOperator(operator, now)}
	if item.Status == 0 {
		item.Status = enums.StatusOk
	}
	if err := repositories.CategoryRepository.Create(sqls.DB(), item); err != nil {
		return nil, err
	}
	return item, nil
}

func (s *supportService) UpdateCategorySort(ids []int64) error {
	return sqls.WithTransaction(func(ctx *sqls.TxContext) error {
		for i, id := range ids {
			if err := repositories.CategoryRepository.UpdateColumn(ctx.Tx, id, "sort_no", i); err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *supportService) DeleteCategory(id int64) error {
	if repositories.CategoryRepository.Get(sqls.DB(), id) == nil {
		return errorsx.InvalidParam("category not found")
	}
	_, paging := repositories.PostRepository.FindPageByCnd(
		sqls.DB(),
		sqls.NewCnd().Eq("category_id", id).Page(1, 1),
	)
	if paging.Total > 0 {
		return errorsx.InvalidParam("category is still used by posts")
	}
	return repositories.CategoryRepository.Delete(sqls.DB(), id)
}

func (s *supportService) CreatePost(req request.CreatePostRequest, principal *dto.AuthPrincipal) (*models.Post, error) {
	title, content := strings.TrimSpace(req.Title), strings.TrimSpace(req.Content)
	if principal == nil || principal.UserID <= 0 {
		return nil, errorsx.Unauthorized("login is required")
	}
	if title == "" || content == "" {
		return nil, errorsx.InvalidParam("title and content are required")
	}
	tags, _ := json.Marshal(normalizeTags(req.Tags))
	now := time.Now()
	item := &models.Post{CategoryID: req.CategoryID, UserID: principal.UserID, Title: title, ContentType: normalizeContentType(req.ContentType), Content: content, TagsJSON: string(tags), Status: enums.PostStatusNormal, AuditFields: supportAuditFields(principal.UserID, supportPrincipalName(principal), now)}
	if err := repositories.PostRepository.Create(sqls.DB(), item); err != nil {
		return nil, err
	}
	return item, nil
}

func (s *supportService) UpdatePost(req request.UpdatePostRequest, principal *dto.AuthPrincipal) error {
	item := repositories.PostRepository.Get(sqls.DB(), req.ID)
	if item == nil {
		return errorsx.InvalidParam("post not found")
	}
	if principal == nil || item.UserID != principal.UserID {
		return errorsx.Forbidden("only the post owner can update it")
	}
	if item.Status == enums.PostStatusResolved || item.Status == enums.PostStatusClosed {
		return errorsx.BusinessError(1, "resolved or closed post cannot be edited")
	}
	tags, _ := json.Marshal(normalizeTags(req.Tags))
	return repositories.PostRepository.Updates(sqls.DB(), req.ID, map[string]any{"category_id": req.CategoryID, "title": strings.TrimSpace(req.Title), "content_type": normalizeContentType(req.ContentType), "content": strings.TrimSpace(req.Content), "tags_json": string(tags), "update_user_id": principal.UserID, "update_user_name": supportPrincipalName(principal), "updated_at": time.Now()})
}

func (s *supportService) CreateCustomerComment(req request.CreateCommentRequest, principal *dto.AuthPrincipal) (*models.Comment, error) {
	if principal == nil {
		return nil, errorsx.Unauthorized("login is required")
	}
	return s.createComment(req.PostID, req.ParentID, normalizeContentType(req.ContentType), strings.TrimSpace(req.Content), supportAuthorType(principal), principal.UserID, supportPrincipalName(principal))
}

func (s *supportService) CreateUserComment(req request.CreateCommentRequest, operator *dto.AuthPrincipal) (*models.Comment, error) {
	if operator == nil {
		return nil, errorsx.Unauthorized("login is required")
	}
	return s.createComment(req.PostID, req.ParentID, normalizeContentType(req.ContentType), strings.TrimSpace(req.Content), supportAuthorType(operator), operator.UserID, supportPrincipalName(operator))
}

func (s *supportService) createComment(postID, parentID int64, contentType, content string, authorType enums.CommentAuthorType, authorID int64, authorName string) (*models.Comment, error) {
	if content == "" {
		return nil, errorsx.InvalidParam("comment content is required")
	}
	post := repositories.PostRepository.Get(sqls.DB(), postID)
	if post == nil || post.Status == enums.PostStatusDeleted || post.Status == enums.PostStatusClosed {
		return nil, errorsx.InvalidParam("post is unavailable")
	}
	var parent *models.Comment
	if parentID > 0 {
		parent = repositories.CommentRepository.Get(sqls.DB(), parentID)
		if parent == nil || parent.PostID != postID || parent.ParentID != 0 || parent.Status != enums.CommentStatusNormal {
			return nil, errorsx.InvalidParam("parent comment is unavailable")
		}
	}
	now := time.Now()
	comment := &models.Comment{PostID: postID, ParentID: parentID, AuthorType: authorType, AuthorID: authorID, ContentType: contentType, Content: content, Status: enums.CommentStatusNormal, AuditFields: supportAuditFields(authorID, authorName, now)}
	if err := sqls.WithTransaction(func(ctx *sqls.TxContext) error {
		if err := repositories.CommentRepository.Create(ctx.Tx, comment); err != nil {
			return err
		}
		if parent != nil {
			if err := repositories.CommentRepository.UpdateColumn(ctx.Tx, parent.ID, "reply_count", gorm.Expr("reply_count + ?", 1)); err != nil {
				return err
			}
		}
		return repositories.PostRepository.Updates(ctx.Tx, postID, map[string]any{"comment_count": gorm.Expr("comment_count + ?", 1), "last_commented_at": now, "last_comment_user_type": authorType, "last_comment_user_id": authorID, "updated_at": now})
	}); err != nil {
		return nil, err
	}
	return comment, nil
}

func (s *supportService) ListPostComments(postID, parentID int64, sort string, page, limit int) (*CommentListResult, error) {
	post := repositories.PostRepository.Get(sqls.DB(), postID)
	if post == nil || post.Status == enums.PostStatusHidden || post.Status == enums.PostStatusDeleted {
		return nil, errorsx.InvalidParam("post not found")
	}
	if parentID > 0 {
		parent := repositories.CommentRepository.Get(sqls.DB(), parentID)
		if parent == nil || parent.PostID != postID || parent.Status == enums.CommentStatusHidden {
			return nil, errorsx.InvalidParam("parent comment not found")
		}
	}
	if page <= 0 {
		page = 1
	}
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	cnd := sqls.NewCnd().Eq("post_id", postID).Eq("parent_id", parentID).Where("status IN ?", []enums.CommentStatus{enums.CommentStatusNormal, enums.CommentStatusDeleted}).Page(page, limit)
	if parentID > 0 {
		cnd.Asc("id")
	} else {
		switch sort {
		case "latest":
			cnd.Desc("id")
		case "hot":
			cnd.Desc("reaction_count").Desc("reply_count").Desc("id")
		default:
			cnd.Desc("is_accepted").Asc("id")
		}
	}
	comments, paging := repositories.CommentRepository.FindPageByCnd(sqls.DB(), cnd)
	result := &CommentListResult{Comments: comments, Replies: map[int64][]models.Comment{}, Paging: paging}
	if parentID > 0 || len(comments) == 0 {
		return result, nil
	}
	for _, comment := range comments {
		if comment.ReplyCount <= 0 {
			continue
		}
		result.Replies[comment.ID] = repositories.CommentRepository.Find(sqls.DB(), sqls.NewCnd().Eq("post_id", postID).Eq("parent_id", comment.ID).Where("status IN ?", []enums.CommentStatus{enums.CommentStatusNormal, enums.CommentStatusDeleted}).Asc("id").Page(1, 2))
	}
	return result, nil
}

func (s *supportService) UpdateComment(req request.UpdateCommentRequest, principal *dto.AuthPrincipal) error {
	comment := repositories.CommentRepository.Get(sqls.DB(), req.ID)
	if comment == nil || comment.Status != enums.CommentStatusNormal {
		return errorsx.InvalidParam("comment not found")
	}
	if principal == nil || comment.AuthorID != principal.UserID {
		return errorsx.Forbidden("only the comment author can update it")
	}
	content := strings.TrimSpace(req.Content)
	if content == "" {
		return errorsx.InvalidParam("comment content is required")
	}
	return repositories.CommentRepository.Updates(sqls.DB(), comment.ID, map[string]any{"content_type": normalizeContentType(req.ContentType), "content": content, "update_user_id": principal.UserID, "update_user_name": supportPrincipalName(principal), "updated_at": time.Now()})
}

func (s *supportService) DeleteComment(commentID int64, principal *dto.AuthPrincipal) error {
	comment := repositories.CommentRepository.Get(sqls.DB(), commentID)
	if comment == nil || comment.Status == enums.CommentStatusDeleted {
		return errorsx.InvalidParam("comment not found")
	}
	if principal == nil || comment.AuthorID != principal.UserID {
		return errorsx.Forbidden("only the comment author can delete it")
	}
	now := time.Now()
	return sqls.WithTransaction(func(ctx *sqls.TxContext) error {
		if err := repositories.CommentRepository.Updates(ctx.Tx, comment.ID, map[string]any{"status": enums.CommentStatusDeleted, "is_accepted": false, "updated_at": now, "update_user_id": principal.UserID, "update_user_name": supportPrincipalName(principal)}); err != nil {
			return err
		}
		columns := map[string]any{"updated_at": now}
		if comment.IsAccepted {
			columns["accepted_comment_id"] = int64(0)
			columns["status"] = enums.PostStatusNormal
		}
		return repositories.PostRepository.Updates(ctx.Tx, comment.PostID, columns)
	})
}

func (s *supportService) ReportComment(req request.ReportCommentRequest, principal *dto.AuthPrincipal) error {
	if principal == nil {
		return errorsx.Unauthorized("login is required")
	}
	comment := repositories.CommentRepository.Get(sqls.DB(), req.ID)
	if comment == nil || comment.Status != enums.CommentStatusNormal {
		return errorsx.InvalidParam("comment not found")
	}
	if existing := repositories.CommentReportRepository.Get(sqls.DB(), comment.ID, principal.UserID); existing != nil {
		return nil
	}
	now := time.Now()
	reason := strings.TrimSpace(req.Reason)
	if len([]rune(reason)) > 255 {
		reason = string([]rune(reason)[:255])
	}
	return sqls.WithTransaction(func(ctx *sqls.TxContext) error {
		if err := repositories.CommentReportRepository.Create(ctx.Tx, &models.CommentReport{CommentID: comment.ID, UserID: principal.UserID, Reason: reason, CreatedAt: now}); err != nil {
			return err
		}
		return repositories.CommentRepository.UpdateColumn(ctx.Tx, comment.ID, "report_count", gorm.Expr("report_count + ?", 1))
	})
}

func (s *supportService) AcceptComment(req request.AcceptCommentRequest, principal *dto.AuthPrincipal, operator *dto.AuthPrincipal) error {
	post := repositories.PostRepository.Get(sqls.DB(), req.PostID)
	comment := repositories.CommentRepository.Get(sqls.DB(), req.CommentID)
	if post == nil || comment == nil || comment.PostID != post.ID {
		return errorsx.InvalidParam("post or comment not found")
	}
	if comment.ParentID != 0 || comment.Status != enums.CommentStatusNormal {
		return errorsx.InvalidParam("only top-level normal comments can be accepted")
	}
	if operator == nil {
		if principal == nil || post.UserID != principal.UserID {
			return errorsx.Forbidden("only owner or admin can accept the best comment")
		}
	}
	now := time.Now()
	return sqls.WithTransaction(func(ctx *sqls.TxContext) error {
		if err := ctx.Tx.Model(&models.Comment{}).Where("post_id = ?", post.ID).Update("is_accepted", false).Error; err != nil {
			return err
		}
		if err := repositories.CommentRepository.Updates(ctx.Tx, comment.ID, map[string]any{"is_accepted": true, "updated_at": now}); err != nil {
			return err
		}
		return repositories.PostRepository.Updates(ctx.Tx, post.ID, map[string]any{"accepted_comment_id": comment.ID, "status": enums.PostStatusResolved, "updated_at": now})
	})
}

func (s *supportService) ToggleReaction(targetType enums.ReactionTarget, targetID int64, reactionType enums.ReactionType, principal *dto.AuthPrincipal) error {
	if principal == nil {
		return errorsx.Unauthorized("login is required")
	}
	if reactionType == "" {
		reactionType = enums.ReactionTypeLike
	}
	if reactionType != enums.ReactionTypeLike {
		return errorsx.InvalidParam("reaction type is unsupported")
	}
	updateReactionCount := func(db *gorm.DB, delta int) error {
		switch targetType {
		case enums.ReactionTargetPost:
			if repositories.PostRepository.Get(sqls.DB(), targetID) == nil {
				return errorsx.InvalidParam("post not found")
			}
			return repositories.PostRepository.UpdateColumn(db, targetID, "reaction_count", gorm.Expr("reaction_count + ?", delta))
		case enums.ReactionTargetComment:
			if repositories.CommentRepository.Get(sqls.DB(), targetID) == nil {
				return errorsx.InvalidParam("comment not found")
			}
			return repositories.CommentRepository.Updates(db, targetID, map[string]any{"reaction_count": gorm.Expr("reaction_count + ?", delta), "updated_at": time.Now()})
		default:
			return errorsx.InvalidParam("reaction target type is unsupported")
		}
	}
	return sqls.WithTransaction(func(ctx *sqls.TxContext) error {
		existing := repositories.ReactionRepository.Get(ctx.Tx, string(targetType), targetID, principal.UserID, string(reactionType))
		delta := 1
		if existing != nil {
			delta = -1
			if err := repositories.ReactionRepository.Delete(ctx.Tx, string(targetType), targetID, principal.UserID, string(reactionType)); err != nil {
				return err
			}
		} else {
			now := time.Now()
			reaction := &models.Reaction{TargetType: targetType, TargetID: targetID, UserID: principal.UserID, ReactionType: reactionType, CreatedAt: now, UpdatedAt: now}
			if err := repositories.ReactionRepository.Create(ctx.Tx, reaction); err != nil {
				return err
			}
		}
		return updateReactionCount(ctx.Tx, delta)
	})
}

func (s *supportService) HasReaction(targetType enums.ReactionTarget, targetID int64, reactionType enums.ReactionType, principal *dto.AuthPrincipal) bool {
	if principal == nil || principal.UserID <= 0 {
		return false
	}
	return repositories.ReactionRepository.Get(sqls.DB(), string(targetType), targetID, principal.UserID, string(reactionType)) != nil
}

func (s *supportService) ModeratePost(req request.ModeratePostRequest) error {
	if repositories.PostRepository.Get(sqls.DB(), req.ID) == nil {
		return errorsx.InvalidParam("post not found")
	}
	return repositories.PostRepository.Updates(sqls.DB(), req.ID, map[string]any{"status": req.Status, "updated_at": time.Now()})
}

func (s *supportService) ModerateComment(req request.ModerateCommentRequest) error {
	if repositories.CommentRepository.Get(sqls.DB(), req.ID) == nil {
		return errorsx.InvalidParam("comment not found")
	}
	return repositories.CommentRepository.Updates(sqls.DB(), req.ID, map[string]any{"status": req.Status, "updated_at": time.Now()})
}

func (s *supportService) FeedbackDocPage(req request.DocPageFeedbackRequest) error {
	column := "unhelpful_count"
	if req.Helpful {
		column = "helpful_count"
	}
	return repositories.DocPageRepository.UpdateColumn(sqls.DB(), req.ID, column, gorm.Expr(column+" + ?", 1))
}

func normalizeSupportEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func normalizeSupportSlug(slug string) string {
	return strings.ToLower(strings.TrimSpace(slug))
}

func validateSupportSlug(slug string) error {
	if !supportSlugPattern.MatchString(slug) {
		return errorsx.InvalidParamI18n("error.docPage.invalidSlug")
	}
	return nil
}

func normalizeContentType(contentType string) string {
	contentType = strings.ToLower(strings.TrimSpace(contentType))
	if contentType == "html" {
		return "html"
	}
	return "markdown"
}

func normalizeDocPageStatus(status enums.DocPageStatus) enums.DocPageStatus {
	if status == "" {
		return enums.DocPageStatusDraft
	}
	return status
}

func normalizeTags(tags []string) []string {
	ret := make([]string, 0, len(tags))
	seen := map[string]struct{}{}
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			continue
		}
		if _, ok := seen[tag]; ok {
			continue
		}
		seen[tag] = struct{}{}
		ret = append(ret, tag)
	}
	return ret
}

func auditFieldsFromOperator(operator *dto.AuthPrincipal, now time.Time) models.AuditFields {
	if operator == nil {
		return supportAuditFields(0, "system", now)
	}
	return supportAuditFields(operator.UserID, operator.Username, now)
}

func supportAuditFields(userID int64, username string, now time.Time) models.AuditFields {
	return models.AuditFields{CreatedAt: now, CreateUserID: userID, CreateUserName: username, UpdatedAt: now, UpdateUserID: userID, UpdateUserName: username}
}

func supportPrincipalName(principal *dto.AuthPrincipal) string {
	if principal == nil {
		return ""
	}
	if strings.TrimSpace(principal.Nickname) != "" {
		return strings.TrimSpace(principal.Nickname)
	}
	return strings.TrimSpace(principal.Username)
}

func supportAuthorType(principal *dto.AuthPrincipal) enums.CommentAuthorType {
	if principal != nil && principal.UserType == enums.UserTypeEmployee {
		return enums.CommentAuthorTypeEmployee
	}
	return enums.CommentAuthorTypeUser
}
