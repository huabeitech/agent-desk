package dashboard

import (
	"agent-desk/internal/builders"
	"agent-desk/internal/pkg/constants"
	"agent-desk/internal/pkg/dto/request"
	"agent-desk/internal/pkg/httpx"
	"agent-desk/internal/pkg/httpx/params"
	"agent-desk/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/mlogclub/simple/web"
)

func PromotionPostList(ctx *gin.Context) {
	if _, err := services.AuthService.RequirePermission(ctx, constants.PermissionPromotionView); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	req := request.PromotionListRequest{}
	if err := params.ReadJSON(ctx, &req); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	list, paging := services.PromotionService.List(req)
	httpx.WriteJSON(ctx, &web.PageResult{Results: builders.BuildPromotionList(list), Page: paging})
}

func PromotionGetBy(ctx *gin.Context) {
	if _, err := services.AuthService.RequirePermission(ctx, constants.PermissionPromotionView); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	id, ok := httpx.GetPathInt64(ctx, "id")
	if !ok {
		return
	}
	httpx.WriteJSON(ctx, builders.BuildPromotion(services.PromotionService.Get(id)))
}

func PromotionPostCreate(ctx *gin.Context) {
	operator, err := services.AuthService.RequirePermission(ctx, constants.PermissionPromotionCreate)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	req := request.SavePromotionRequest{}
	if err := params.ReadJSON(ctx, &req); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	item, err := services.PromotionService.Create(req, operator)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	httpx.WriteJSON(ctx, builders.BuildPromotion(item))
}

func PromotionPostUpdate(ctx *gin.Context) {
	operator, err := services.AuthService.RequirePermission(ctx, constants.PermissionPromotionUpdate)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	req := request.SavePromotionRequest{}
	if err := params.ReadJSON(ctx, &req); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	if err := services.PromotionService.Update(req, operator); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	httpx.WriteJSON(ctx, nil)
}

func PromotionPostUpdate_status(ctx *gin.Context) {
	operator, err := services.AuthService.RequirePermission(ctx, constants.PermissionPromotionUpdate)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	req := request.UpdatePromotionStatusRequest{}
	if err := params.ReadJSON(ctx, &req); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	if err := services.PromotionService.UpdateStatus(req, operator); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	httpx.WriteJSON(ctx, nil)
}

func PromotionPostDelete(ctx *gin.Context) {
	operator, err := services.AuthService.RequirePermission(ctx, constants.PermissionPromotionDelete)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	req := request.DeletePromotionRequest{}
	if err := params.ReadJSON(ctx, &req); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	if err := services.PromotionService.Delete(req.ID, operator); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	httpx.WriteJSON(ctx, nil)
}

func PromotionPostReindex(ctx *gin.Context) {
	if _, err := services.AuthService.RequirePermission(ctx, constants.PermissionPromotionReindex); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	req := request.ReindexPromotionRequest{}
	if err := params.ReadJSON(ctx, &req); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	if err := services.PromotionService.Reindex(req.ID); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	httpx.WriteJSON(ctx, nil)
}

func PromotionPostImport(ctx *gin.Context) {
	operator, err := services.AuthService.RequirePermission(ctx, constants.PermissionPromotionCreate)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	file, err := ctx.FormFile("file")
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	reader, err := file.Open()
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	defer reader.Close()
	result, err := services.PromotionService.ImportCSV(reader, operator)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	httpx.WriteJSON(ctx, result)
}

func PromotionPostSeed_muse(ctx *gin.Context) {
	operator, err := services.AuthService.RequirePermission(ctx, constants.PermissionPromotionCreate)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	if err := services.PromotionService.SeedMusePromotions(operator); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	httpx.WriteJSON(ctx, nil)
}
