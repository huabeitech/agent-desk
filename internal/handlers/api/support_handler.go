package api

import (
	"agent-desk/internal/builders"
	"agent-desk/internal/models"
	"agent-desk/internal/pkg/config"
	"agent-desk/internal/pkg/dto/request"
	"agent-desk/internal/pkg/dto/response"
	"agent-desk/internal/pkg/enums"
	"agent-desk/internal/pkg/errorsx"
	"agent-desk/internal/pkg/httpx"
	"agent-desk/internal/pkg/httpx/params"
	"agent-desk/internal/repositories"
	"agent-desk/internal/services"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/mlogclub/simple/sqls"
	"github.com/mlogclub/simple/web"
	"github.com/spf13/cast"
	"gorm.io/gorm"
)

func SupportAuthPostRegister(ctx *gin.Context) {
	cfg := config.Current()
	if !cfg.Auth.IsPasswordLoginEnabled() {
		httpx.WriteJSON(ctx, errorsx.ForbiddenI18n("error.auth.passwordLoginDisabled"))
		return
	}

	req := request.SupportCustomerRegisterRequest{}
	if err := params.ReadJSON(ctx, &req); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	ret, err := services.SupportService.RegisterUser(req, cfg.Auth, ctx.ClientIP(), ctx.GetHeader("User-Agent"))
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	httpx.WriteJSON(ctx, ret)
}

func SupportGetMe(ctx *gin.Context) {
	principal, err := services.SupportService.RequireSupportUser(ctx)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	user := supportUser(principal.UserID)
	email := ""
	if user != nil && user.Email != nil {
		email = *user.Email
	}
	httpx.WriteJSON(ctx, response.SupportUserResponse{ID: principal.UserID, Name: supportPrincipalDisplayName(principal.UserID), Email: email, UserType: principal.UserType})
}

func DocPageAnyList(ctx *gin.Context) {
	cnd := params.NewPagedSqlCnd(ctx,
		params.QueryFilter{ParamName: "parentId"},
		params.QueryFilter{ParamName: "title", Op: params.Like},
	).Eq("status", enums.DocPageStatusPublished).Asc("sort_no").Desc("id")
	if keyword := strings.TrimSpace(ctx.Query("keyword")); keyword != "" {
		pattern := "%" + keyword + "%"
		cnd.Where("(title LIKE ? OR summary LIKE ? OR content LIKE ? OR tags_json LIKE ? OR slug LIKE ?)", pattern, pattern, pattern, pattern, pattern)
	}
	list, paging := repositories.DocPageRepository.FindPageByCnd(sqls.DB(), cnd)
	results := buildDocPageList(list, false)
	httpx.WriteJSON(ctx, &web.PageResult{Results: results, Page: paging})
}

func DocPageGetNavigation(ctx *gin.Context) {
	list := services.SupportService.FindPublicDocNavigation()
	httpx.WriteJSON(ctx, builders.BuildDocPageNavigationTree(list))
}

func DocPageGetBy(ctx *gin.Context) {
	id, ok := httpx.GetPathInt64(ctx, "id")
	if !ok {
		return
	}
	item := repositories.DocPageRepository.Get(sqls.DB(), id)
	if item == nil || item.Status != enums.DocPageStatusPublished {
		httpx.WriteJSON(ctx, httpx.JsonErrorMsg(ctx, "error.notFound"))
		return
	}
	_ = repositories.DocPageRepository.UpdateColumn(sqls.DB(), item.ID, "view_count", gorm.Expr("view_count + ?", 1))
	httpx.WriteJSON(ctx, builders.BuildDocPage(item, true))
}

func DocPagePostFeedback(ctx *gin.Context) {
	req := request.DocPageFeedbackRequest{}
	if err := params.ReadJSON(ctx, &req); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	httpx.WriteJSON(ctx, services.SupportService.FeedbackDocPage(req))
}

func CategoryAnyList(ctx *gin.Context) {
	list := repositories.CategoryRepository.Find(sqls.DB(), sqls.NewCnd().Eq("status", enums.StatusOk).Asc("sort_no").Desc("id"))
	httpx.WriteJSON(ctx, builders.BuildPostCategories(list))
}

func PostAnyList(ctx *gin.Context) {
	cursor, _ := params.GetInt64(ctx, "cursor")
	limit, _ := params.GetInt(ctx, "limit")
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	cnd := params.NewSqlCnd(ctx,
		params.QueryFilter{ParamName: "categoryId"},
		params.QueryFilter{ParamName: "userId"},
		params.QueryFilter{ParamName: "status"},
		params.QueryFilter{ParamName: "title", Op: params.Like},
	).Where("status NOT IN ?", []enums.PostStatus{enums.PostStatusHidden, enums.PostStatusDeleted}).Desc("id").Limit(limit + 1)
	if cursor > 0 {
		cnd.Lt("id", cursor)
	}
	list := repositories.PostRepository.Find(sqls.DB(), cnd)
	hasMore := len(list) > limit
	if hasMore {
		list = list[:limit]
	}
	nextCursor := ""
	if hasMore && len(list) > 0 {
		nextCursor = cast.ToString(list[len(list)-1].ID)
	}
	data := services.SupportService.LoadCommunityResponseData(list, nil, nil)
	httpx.WriteJSON(ctx, httpx.CursorData(builders.BuildPostList(list, data.Categories, data.Users), nextCursor, hasMore))
}

func PostGetBy(ctx *gin.Context) {
	id, ok := httpx.GetPathInt64(ctx, "id")
	if !ok {
		return
	}
	post := repositories.PostRepository.Get(sqls.DB(), id)
	if post == nil || post.Status == enums.PostStatusHidden || post.Status == enums.PostStatusDeleted {
		httpx.WriteJSON(ctx, httpx.JsonErrorMsg(ctx, "error.notFound"))
		return
	}
	_ = repositories.PostRepository.UpdateColumn(sqls.DB(), post.ID, "view_count", gorm.Expr("view_count + ?", 1))
	comments, err := services.SupportService.ListPostComments(id, 0, "default", 1, 20)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	data := services.SupportService.LoadCommunityResponseData([]models.Post{*post}, comments.Comments, comments.Replies)
	categoryName := ""
	if category := data.Categories[post.CategoryID]; category != nil {
		categoryName = category.Name
	}
	httpx.WriteJSON(ctx, response.PostDetailResponse{Post: *builders.BuildPost(post, categoryName, data.Users[post.UserID]), Comments: buildCommentListWithReplies(comments.Comments, comments.Replies, data.Users)})
}

func PostPostCreate(ctx *gin.Context) {
	principal, err := services.SupportService.RequireSupportUser(ctx)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	req := request.CreatePostRequest{}
	if err := params.ReadJSON(ctx, &req); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	item, err := services.SupportService.CreatePost(req, principal)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	data := services.SupportService.LoadCommunityResponseData([]models.Post{*item}, nil, nil)
	categoryName := ""
	if category := data.Categories[item.CategoryID]; category != nil {
		categoryName = category.Name
	}
	httpx.WriteJSON(ctx, builders.BuildPost(item, categoryName, data.Users[item.UserID]))
}

func PostPostUpdate(ctx *gin.Context) {
	principal, err := services.SupportService.RequireSupportUser(ctx)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	req := request.UpdatePostRequest{}
	if err := params.ReadJSON(ctx, &req); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	httpx.WriteJSON(ctx, services.SupportService.UpdatePost(req, principal))
}

func PostPostAcceptComment(ctx *gin.Context) {
	principal, err := services.SupportService.RequireSupportUser(ctx)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	req := request.AcceptCommentRequest{}
	if err := params.ReadJSON(ctx, &req); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	httpx.WriteJSON(ctx, services.SupportService.AcceptComment(req, principal, nil))
}

func CommentPostCreate(ctx *gin.Context) {
	principal, err := services.SupportService.RequireSupportUser(ctx)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	req := request.CreateCommentRequest{}
	if err := params.ReadJSON(ctx, &req); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	item, err := services.SupportService.CreateCustomerComment(req, principal)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	data := services.SupportService.LoadCommunityResponseData(nil, []models.Comment{*item}, nil)
	httpx.WriteJSON(ctx, builders.BuildComment(item, data.Users[item.AuthorID]))
}

func CommentAnyList(ctx *gin.Context) {
	postID, _ := params.GetInt64(ctx, "postId")
	parentID, _ := params.GetInt64(ctx, "parentId")
	page, _ := params.GetInt(ctx, "page")
	limit, _ := params.GetInt(ctx, "limit")
	sort, _ := params.Get(ctx, "sort")
	result, err := services.SupportService.ListPostComments(postID, parentID, sort, page, limit)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	data := services.SupportService.LoadCommunityResponseData(nil, result.Comments, result.Replies)
	httpx.WriteJSON(ctx, &web.PageResult{Results: buildCommentListWithReplies(result.Comments, result.Replies, data.Users), Page: result.Paging})
}

func CommentPostUpdate(ctx *gin.Context) {
	principal, err := services.SupportService.RequireSupportUser(ctx)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	req := request.UpdateCommentRequest{}
	if err := params.ReadJSON(ctx, &req); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	httpx.WriteJSON(ctx, services.SupportService.UpdateComment(req, principal))
}

func CommentPostDelete(ctx *gin.Context) {
	principal, err := services.SupportService.RequireSupportUser(ctx)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	req := request.IDRequest{}
	if err := params.ReadJSON(ctx, &req); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	httpx.WriteJSON(ctx, services.SupportService.DeleteComment(req.ID, principal))
}

func CommentPostReport(ctx *gin.Context) {
	principal, err := services.SupportService.RequireSupportUser(ctx)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	req := request.ReportCommentRequest{}
	if err := params.ReadJSON(ctx, &req); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	httpx.WriteJSON(ctx, services.SupportService.ReportComment(req, principal))
}

func ReactionPostToggle(ctx *gin.Context) {
	principal, err := services.SupportService.RequireSupportUser(ctx)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	req := request.ReactionRequest{}
	if err := params.ReadJSON(ctx, &req); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	httpx.WriteJSON(ctx, services.SupportService.ToggleReaction(req.TargetType, req.TargetID, req.ReactionType, principal))
}

func buildDocPageList(list []models.DocPage, includeContent bool) []response.DocPageResponse {
	results := make([]response.DocPageResponse, 0, len(list))
	for _, item := range list {
		if resp := builders.BuildDocPage(&item, includeContent); resp != nil {
			results = append(results, *resp)
		}
	}
	return results
}

func buildCommentListWithReplies(list []models.Comment, replies map[int64][]models.Comment, users map[int64]*models.User) []response.CommentResponse {
	results := make([]response.CommentResponse, 0, len(list))
	for _, item := range list {
		if resp := builders.BuildComment(&item, users[item.AuthorID]); resp != nil {
			if len(replies[item.ID]) > 0 {
				resp.Replies = buildCommentListWithReplies(replies[item.ID], nil, users)
			}
			results = append(results, *resp)
		}
	}
	return results
}

func supportUser(id int64) *models.User {
	return repositories.UserRepository.Get(sqls.DB(), id)
}

func supportPrincipalDisplayName(id int64) string {
	user := supportUser(id)
	if user == nil {
		return ""
	}
	if user.Nickname != "" {
		return user.Nickname
	}
	return user.Username
}
