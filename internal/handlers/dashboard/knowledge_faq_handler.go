package dashboard

import (
	"agent-desk/internal/builders"
	"agent-desk/internal/pkg/constants"
	"agent-desk/internal/pkg/dto/request"
	"agent-desk/internal/pkg/dto/response"
	"agent-desk/internal/pkg/errorsx"
	"agent-desk/internal/pkg/httpx"
	"agent-desk/internal/pkg/i18nx"
	"agent-desk/internal/services"
	"fmt"
	"net/http"

	"agent-desk/internal/pkg/httpx/params"

	"github.com/gin-gonic/gin"
	"github.com/mlogclub/simple/web"
)

func KnowledgeFAQGetImport_template(ctx *gin.Context) {
	if _, err := services.AuthService.RequirePermission(ctx, constants.PermissionKnowledgeFAQView); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	file, err := services.KnowledgeFAQService.BuildKnowledgeFAQImportTemplate()
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	writeKnowledgeFAQExcelFile(ctx, file)
}

func KnowledgeFAQGetExport(ctx *gin.Context) {
	if _, err := services.AuthService.RequirePermission(ctx, constants.PermissionKnowledgeFAQView); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	knowledgeBaseID, ok := params.GetInt64(ctx, "knowledgeBaseId")
	if !ok || knowledgeBaseID <= 0 {
		httpx.WriteJSON(ctx, errorsx.InvalidParamI18n("error.e0283"))
		return
	}
	file, err := services.KnowledgeFAQService.ExportKnowledgeFAQs(knowledgeBaseID)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	writeKnowledgeFAQExcelFile(ctx, file)
}

func KnowledgeFAQPostImport(ctx *gin.Context) {
	operator, err := services.AuthService.RequirePermission(ctx, constants.PermissionKnowledgeFAQCreate)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	mode := request.KnowledgeFAQImportMode(ctx.PostForm("mode"))
	if mode == request.KnowledgeFAQImportModeOverwrite {
		if _, err := services.AuthService.RequirePermission(ctx, constants.PermissionKnowledgeFAQUpdate); err != nil {
			httpx.WriteJSON(ctx, err)
			return
		}
	}
	knowledgeBaseID, ok := params.GetInt64(ctx, "knowledgeBaseId")
	if !ok || knowledgeBaseID <= 0 {
		httpx.WriteJSON(ctx, errorsx.InvalidParamI18n("error.e0283"))
		return
	}
	header, err := ctx.FormFile("file")
	if err != nil {
		httpx.WriteJSON(ctx, errorsx.InvalidParamI18n("error.e0327"))
		return
	}
	file, err := header.Open()
	if err != nil {
		httpx.WriteJSON(ctx, errorsx.InvalidParamI18n("error.e0176"))
		return
	}
	defer file.Close()
	result, err := services.KnowledgeFAQService.ImportKnowledgeFAQs(request.ImportKnowledgeFAQRequest{
		KnowledgeBaseID: knowledgeBaseID,
		Mode:            mode,
		Filename:        header.Filename,
		Reader:          file,
		Locale:          i18nx.Locale(ctx),
	}, operator)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	httpx.WriteJSON(ctx, result)
}

func KnowledgeFAQAnyList(ctx *gin.Context) {
	if _, err := services.AuthService.RequirePermission(ctx, constants.PermissionKnowledgeFAQView); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}

	cnd := params.NewPagedSqlCnd(ctx,
		params.QueryFilter{ParamName: "knowledgeBaseId"},
		params.QueryFilter{ParamName: "question", Op: params.Like},
		params.QueryFilter{ParamName: "status"},
		params.QueryFilter{ParamName: "indexStatus"},
	).Desc("id")
	knowledgeBaseID, _ := params.GetInt64(ctx, "knowledgeBaseId")
	if directoryID, ok := params.GetInt64(ctx, "directoryId"); ok {
		cnd.Where("directory_id = ?", directoryID)
	}
	list, paging := services.KnowledgeFAQService.FindPageByCnd(cnd)
	directoryPaths := services.KnowledgeDirectoryService.PathMap(knowledgeBaseID)
	results := make([]response.KnowledgeFAQResponse, 0, len(list))
	for _, item := range list {
		resp := builders.BuildKnowledgeFAQ(&item)
		fillKnowledgeFAQDirectory(&resp, directoryPaths)
		results = append(results, resp)
	}
	httpx.WriteJSON(ctx, &web.PageResult{Results: results, Page: paging})
}

func KnowledgeFAQGetBy(ctx *gin.Context) {
	id, ok := httpx.GetPathInt64(ctx, "id")
	if !ok {
		return
	}
	if _, err := services.AuthService.RequirePermission(ctx, constants.PermissionKnowledgeFAQView); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}

	item := services.KnowledgeFAQService.Get(id)
	if item == nil {
		httpx.WriteJSON(ctx, httpx.JsonErrorMsg(ctx, "error.e0025"))
		return
	}
	resp := builders.BuildKnowledgeFAQ(item)
	fillKnowledgeFAQDirectory(&resp, services.KnowledgeDirectoryService.PathMap(item.KnowledgeBaseID))
	httpx.WriteJSON(ctx, resp)
}

func KnowledgeFAQPostCreate(ctx *gin.Context) {
	operator, err := services.AuthService.RequirePermission(ctx, constants.PermissionKnowledgeFAQCreate)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	req := request.CreateKnowledgeFAQRequest{}
	if err := params.ReadJSON(ctx, &req); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	item, err := services.KnowledgeFAQService.CreateKnowledgeFAQ(req, operator)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	httpx.WriteJSON(ctx, builders.BuildKnowledgeFAQ(item))
}

func KnowledgeFAQPostUpdate(ctx *gin.Context) {
	operator, err := services.AuthService.RequirePermission(ctx, constants.PermissionKnowledgeFAQUpdate)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	req := request.UpdateKnowledgeFAQRequest{}
	if err := params.ReadJSON(ctx, &req); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	if err := services.KnowledgeFAQService.UpdateKnowledgeFAQ(req, operator); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	httpx.WriteJSON(ctx, nil)
}

func KnowledgeFAQPostUpdate_status(ctx *gin.Context) {
	operator, err := services.AuthService.RequirePermission(ctx, constants.PermissionKnowledgeFAQUpdate)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	req := request.UpdateKnowledgeFAQStatusRequest{}
	if err := params.ReadJSON(ctx, &req); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	if err := services.KnowledgeFAQService.UpdateStatus(req.ID, req.Status, operator); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	httpx.WriteJSON(ctx, nil)
}

func KnowledgeFAQPostDelete(ctx *gin.Context) {
	if _, err := services.AuthService.RequirePermission(ctx, constants.PermissionKnowledgeFAQDelete); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	var req struct {
		ID int64 `json:"id"`
	}
	if err := params.ReadJSON(ctx, &req); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	if err := services.KnowledgeFAQService.DeleteKnowledgeFAQ(req.ID); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	httpx.WriteJSON(ctx, nil)
}

func KnowledgeFAQPostBatch_move(ctx *gin.Context) {
	operator, err := services.AuthService.RequirePermission(ctx, constants.PermissionKnowledgeFAQUpdate)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	req := request.BatchMoveKnowledgeFAQRequest{}
	if err := params.ReadJSON(ctx, &req); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	if err := services.KnowledgeFAQService.BatchMoveKnowledgeFAQs(req, operator); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	httpx.WriteJSON(ctx, nil)
}

func KnowledgeFAQPostBatch_delete(ctx *gin.Context) {
	if _, err := services.AuthService.RequirePermission(ctx, constants.PermissionKnowledgeFAQDelete); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	req := request.BatchDeleteKnowledgeFAQRequest{}
	if err := params.ReadJSON(ctx, &req); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	if err := services.KnowledgeFAQService.BatchDeleteKnowledgeFAQs(req); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	httpx.WriteJSON(ctx, nil)
}

func writeKnowledgeFAQExcelFile(ctx *gin.Context, file *response.KnowledgeFAQExportedFile) {
	ctx.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, file.Filename))
	ctx.Header("Cache-Control", "no-store")
	ctx.Data(http.StatusOK, file.ContentType, file.Data)
}

func fillKnowledgeFAQDirectory(resp *response.KnowledgeFAQResponse, directoryPaths map[int64]string) {
	resp.DirectoryPath = directoryPaths[resp.DirectoryID]
	resp.DirectoryName = resp.DirectoryPath
}
