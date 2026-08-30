package dashboard

import (
	"agent-desk/internal/builders"
	"agent-desk/internal/models"
	"agent-desk/internal/pkg/constants"
	"agent-desk/internal/pkg/dto/request"
	"agent-desk/internal/pkg/dto/response"
	"agent-desk/internal/pkg/httpx"
	"agent-desk/internal/pkg/httpx/params"
	"agent-desk/internal/repositories"
	"agent-desk/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/mlogclub/simple/sqls"
	"github.com/mlogclub/simple/web"
)

func DocPageAnyList(ctx *gin.Context) {
	if _, err := services.AuthService.RequirePermission(ctx, constants.PermissionDocPageView); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	list, paging := services.SupportService.FindDocPagePage(params.NewPagedSqlCnd(ctx,
		params.QueryFilter{ParamName: "parentId"},
		params.QueryFilter{ParamName: "status"},
		params.QueryFilter{ParamName: "title", Op: params.Like},
	).Asc("sort_no").Desc("id"))
	httpx.WriteJSON(ctx, &web.PageResult{Results: buildDashboardDocPages(list, false), Page: paging})
}

func DocPageGetList_all(ctx *gin.Context) {
	if _, err := services.AuthService.RequirePermission(ctx, constants.PermissionDocPageView); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	list := services.SupportService.FindDocPages(sqls.NewCnd().Asc("sort_no").Asc("id"))
	httpx.WriteJSON(ctx, buildDashboardDocPages(list, false))
}

func DocPageGetBy(ctx *gin.Context) {
	if _, err := services.AuthService.RequirePermission(ctx, constants.PermissionDocPageView); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	id, ok := httpx.GetPathInt64(ctx, "id")
	if !ok {
		return
	}
	item := services.SupportService.FindDocPageByID(id)
	if item == nil {
		httpx.WriteJSON(ctx, httpx.JsonErrorMsg(ctx, "error.notFound"))
		return
	}
	httpx.WriteJSON(ctx, builders.BuildDocPage(item, true))
}

func DocPagePostCreate(ctx *gin.Context) {
	operator, err := services.AuthService.RequirePermission(ctx, constants.PermissionDocPageCreate)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	req := request.SaveDocPageRequest{}
	if err := params.ReadJSON(ctx, &req); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	item, err := services.SupportService.SaveDocPage(req, operator)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	httpx.WriteJSON(ctx, builders.BuildDocPage(item, true))
}

func DocPagePostUpdate(ctx *gin.Context) {
	operator, err := services.AuthService.RequirePermission(ctx, constants.PermissionDocPageUpdate)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	req := request.SaveDocPageRequest{}
	if err := params.ReadJSON(ctx, &req); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	item, err := services.SupportService.SaveDocPage(req, operator)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	httpx.WriteJSON(ctx, builders.BuildDocPage(item, true))
}

func DocPagePostUpdate_settings(ctx *gin.Context) {
	operator, err := services.AuthService.RequirePermission(ctx, constants.PermissionDocPageUpdate)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	req := request.UpdateDocPageSettingsRequest{}
	if err := params.ReadJSON(ctx, &req); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	item, err := services.SupportService.UpdateDocPageSettings(req, operator)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	httpx.WriteJSON(ctx, builders.BuildDocPage(item, true))
}

func DocPagePostDelete(ctx *gin.Context) {
	if _, err := services.AuthService.RequirePermission(ctx, constants.PermissionDocPageDelete); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	req := request.DeleteByIDRequest{}
	if err := params.ReadJSON(ctx, &req); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	httpx.WriteJSON(ctx, services.SupportService.DeleteDocPage(req.ID))
}

func DocPagePostUpdate_sort(ctx *gin.Context) {
	if _, err := services.AuthService.RequirePermission(ctx, constants.PermissionDocPageUpdate); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	req := request.SortDocPagesRequest{}
	if err := params.ReadJSON(ctx, &req); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	httpx.WriteJSON(ctx, services.SupportService.SortDocPages(req))
}

func DocPagePostChange_status(ctx *gin.Context) {
	operator, err := services.AuthService.RequirePermission(ctx, constants.PermissionDocPageUpdate)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	req := request.ChangeDocPageStatusRequest{}
	if err := params.ReadJSON(ctx, &req); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	item, err := services.SupportService.ChangeDocPageStatus(req, operator)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	httpx.WriteJSON(ctx, builders.BuildDocPage(item, true))
}

func CategoryAnyList(ctx *gin.Context) {
	if _, err := services.AuthService.RequirePermission(ctx, constants.PermissionCommunityView); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	list, paging := repositories.CategoryRepository.FindPageByCnd(sqls.DB(), params.NewPagedSqlCnd(ctx,
		params.QueryFilter{ParamName: "status"},
		params.QueryFilter{ParamName: "name", Op: params.Like},
	).Asc("sort_no").Desc("id"))
	httpx.WriteJSON(ctx, &web.PageResult{Results: builders.BuildPostCategories(list), Page: paging})
}

func CategoryGetList_all(ctx *gin.Context) {
	if _, err := services.AuthService.RequirePermission(ctx, constants.PermissionCommunityView); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	list := repositories.CategoryRepository.Find(sqls.DB(), sqls.NewCnd().Asc("sort_no").Desc("id"))
	httpx.WriteJSON(ctx, builders.BuildPostCategories(list))
}

func CategoryPostCreate(ctx *gin.Context) {
	operator, err := services.AuthService.RequirePermission(ctx, constants.PermissionCommunityUpdate)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	req := request.SaveCategoryRequest{}
	if err := params.ReadJSON(ctx, &req); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	item, err := services.SupportService.SaveCategory(req, operator)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	httpx.WriteJSON(ctx, builders.BuildCategory(item))
}

func CategoryPostUpdate(ctx *gin.Context) {
	CategoryPostCreate(ctx)
}

func CategoryPostUpdateSort(ctx *gin.Context) {
	if _, err := services.AuthService.RequirePermission(ctx, constants.PermissionCommunityUpdate); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	var ids []int64
	if err := params.ReadJSON(ctx, &ids); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	if err := services.SupportService.UpdateCategorySort(ids); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	httpx.WriteJSON(ctx, nil)
}

func CategoryPostDelete(ctx *gin.Context) {
	if _, err := services.AuthService.RequirePermission(ctx, constants.PermissionCommunityUpdate); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	req := request.DeleteByIDRequest{}
	if err := params.ReadJSON(ctx, &req); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	httpx.WriteJSON(ctx, services.SupportService.DeleteCategory(req.ID))
}

func PostAnyList(ctx *gin.Context) {
	if _, err := services.AuthService.RequirePermission(ctx, constants.PermissionCommunityView); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	list, paging := repositories.PostRepository.FindPageByCnd(sqls.DB(), params.NewPagedSqlCnd(ctx,
		params.QueryFilter{ParamName: "categoryId"},
		params.QueryFilter{ParamName: "status"},
		params.QueryFilter{ParamName: "title", Op: params.Like},
	).Desc("id"))
	data := services.SupportService.LoadCommunityResponseData(list, nil, nil)
	httpx.WriteJSON(ctx, &web.PageResult{Results: builders.BuildPostList(list, data.Categories, data.Users), Page: paging})
}

func PostGetBy(ctx *gin.Context) {
	if _, err := services.AuthService.RequirePermission(ctx, constants.PermissionCommunityView); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	id, ok := httpx.GetPathInt64(ctx, "id")
	if !ok {
		return
	}
	post := repositories.PostRepository.Get(sqls.DB(), id)
	if post == nil {
		httpx.WriteJSON(ctx, httpx.JsonErrorMsg(ctx, "error.notFound"))
		return
	}
	comments := repositories.CommentRepository.Find(sqls.DB(), sqls.NewCnd().Eq("post_id", id).Desc("is_accepted").Asc("id"))
	data := services.SupportService.LoadCommunityResponseData([]models.Post{*post}, comments, nil)
	categoryName := ""
	if category := data.Categories[post.CategoryID]; category != nil {
		categoryName = category.Name
	}
	httpx.WriteJSON(ctx, response.PostDetailResponse{Post: *builders.BuildPost(post, categoryName, data.Users[post.UserID]), Comments: buildDashboardComments(comments, data.Users)})
}

func PostPostModerate(ctx *gin.Context) {
	if _, err := services.AuthService.RequirePermission(ctx, constants.PermissionCommunityUpdate); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	req := request.ModeratePostRequest{}
	if err := params.ReadJSON(ctx, &req); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	httpx.WriteJSON(ctx, services.SupportService.ModeratePost(req))
}

func PostPostAcceptComment(ctx *gin.Context) {
	operator, err := services.AuthService.RequirePermission(ctx, constants.PermissionCommunityUpdate)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	req := request.AcceptCommentRequest{}
	if err := params.ReadJSON(ctx, &req); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	httpx.WriteJSON(ctx, services.SupportService.AcceptComment(req, nil, operator))
}

func CommentPostCreate(ctx *gin.Context) {
	operator, err := services.AuthService.RequirePermission(ctx, constants.PermissionCommunityUpdate)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	req := request.CreateCommentRequest{}
	if err := params.ReadJSON(ctx, &req); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	item, err := services.SupportService.CreateUserComment(req, operator)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	data := services.SupportService.LoadCommunityResponseData(nil, []models.Comment{*item}, nil)
	httpx.WriteJSON(ctx, builders.BuildComment(item, data.Users[item.AuthorID]))
}

func CommentPostModerate(ctx *gin.Context) {
	if _, err := services.AuthService.RequirePermission(ctx, constants.PermissionCommunityUpdate); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	req := request.ModerateCommentRequest{}
	if err := params.ReadJSON(ctx, &req); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	httpx.WriteJSON(ctx, services.SupportService.ModerateComment(req))
}

func buildDashboardDocPages(list []models.DocPage, includeContent bool) []response.DocPageResponse {
	results := make([]response.DocPageResponse, 0, len(list))
	for _, item := range list {
		if resp := builders.BuildDocPage(&item, includeContent); resp != nil {
			results = append(results, *resp)
		}
	}
	return results
}

func buildDashboardComments(list []models.Comment, users map[int64]*models.User) []response.CommentResponse {
	results := make([]response.CommentResponse, 0, len(list))
	for _, item := range list {
		if resp := builders.BuildComment(&item, users[item.AuthorID]); resp != nil {
			results = append(results, *resp)
		}
	}
	return results
}
