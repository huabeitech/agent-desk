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

func ProductPostList(ctx *gin.Context) {
	if _, err := services.AuthService.RequirePermission(ctx, constants.PermissionProductView); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	req := request.ProductListRequest{}
	if err := params.ReadJSON(ctx, &req); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	list, paging := services.ProductService.List(req)
	httpx.WriteJSON(ctx, &web.PageResult{Results: builders.BuildProductList(list), Page: paging})
}

func ProductGetBy(ctx *gin.Context) {
	if _, err := services.AuthService.RequirePermission(ctx, constants.PermissionProductView); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	id, ok := httpx.GetPathInt64(ctx, "id")
	if !ok {
		return
	}
	httpx.WriteJSON(ctx, builders.BuildProduct(services.ProductService.Get(id)))
}

func ProductPostCreate(ctx *gin.Context) {
	operator, err := services.AuthService.RequirePermission(ctx, constants.PermissionProductCreate)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	req := request.SaveProductRequest{}
	if err := params.ReadJSON(ctx, &req); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	item, err := services.ProductService.Create(req, operator)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	httpx.WriteJSON(ctx, builders.BuildProduct(item))
}

func ProductPostUpdate(ctx *gin.Context) {
	operator, err := services.AuthService.RequirePermission(ctx, constants.PermissionProductUpdate)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	req := request.SaveProductRequest{}
	if err := params.ReadJSON(ctx, &req); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	if err := services.ProductService.Update(req, operator); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	httpx.WriteJSON(ctx, nil)
}

func ProductPostUpdate_status(ctx *gin.Context) {
	operator, err := services.AuthService.RequirePermission(ctx, constants.PermissionProductUpdate)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	req := request.UpdateProductStatusRequest{}
	if err := params.ReadJSON(ctx, &req); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	if err := services.ProductService.UpdateStatus(req, operator); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	httpx.WriteJSON(ctx, nil)
}

func ProductPostDelete(ctx *gin.Context) {
	operator, err := services.AuthService.RequirePermission(ctx, constants.PermissionProductDelete)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	req := request.DeleteProductRequest{}
	if err := params.ReadJSON(ctx, &req); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	if err := services.ProductService.Delete(req.ID, operator); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	httpx.WriteJSON(ctx, nil)
}

func ProductPostReindex(ctx *gin.Context) {
	if _, err := services.AuthService.RequirePermission(ctx, constants.PermissionProductReindex); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	req := request.ReindexProductRequest{}
	if err := params.ReadJSON(ctx, &req); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	if err := services.ProductService.Reindex(req.ID); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	httpx.WriteJSON(ctx, nil)
}

func ProductPostImport(ctx *gin.Context) {
	operator, err := services.AuthService.RequirePermission(ctx, constants.PermissionProductCreate)
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
	result, err := services.ProductService.ImportCSV(reader, operator)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	httpx.WriteJSON(ctx, result)
}

func ProductPostSeed_muse(ctx *gin.Context) {
	operator, err := services.AuthService.RequirePermission(ctx, constants.PermissionProductCreate)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	if err := services.ProductService.SeedMuseProducts(operator); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	httpx.WriteJSON(ctx, nil)
}
