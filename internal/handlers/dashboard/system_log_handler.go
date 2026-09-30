package dashboard

import (
	"agent-desk/internal/builders"
	"agent-desk/internal/pkg/constants"
	"agent-desk/internal/pkg/dto/response"
	"agent-desk/internal/pkg/httpx"
	"agent-desk/internal/pkg/httpx/params"
	"agent-desk/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/mlogclub/simple/web"
)

// SystemLogAnyList 系统日志分页列表，支持按级别、关键字、时间范围筛选。
func SystemLogAnyList(ctx *gin.Context) {
	if _, err := services.AuthService.RequirePermission(ctx, constants.PermissionSystemLogView); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}

	cnd := params.NewPagedSqlCnd(ctx,
		params.QueryFilter{ParamName: "level", Op: params.Eq},
		params.QueryFilter{ParamName: "message", Op: params.Like},
		params.QueryFilter{ParamName: "source", Op: params.Like},
		params.QueryFilter{ParamName: "startTime", Op: params.Gte, ColumnName: "created_at"},
		params.QueryFilter{ParamName: "endTime", Op: params.Lte, ColumnName: "created_at"},
	).Desc("id")

	list, paging := services.SystemLogService.FindPageByCnd(cnd)
	results := make([]response.SystemLogResponse, 0, len(list))
	for i := range list {
		results = append(results, builders.BuildSystemLog(&list[i]))
	}
	httpx.WriteJSON(ctx, &web.PageResult{Results: results, Page: paging})
}

// SystemLogGetBy 系统日志详情。
func SystemLogGetBy(ctx *gin.Context) {
	if _, err := services.AuthService.RequirePermission(ctx, constants.PermissionSystemLogView); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}

	id, ok := httpx.GetPathInt64(ctx, "id")
	if !ok {
		return
	}

	item := services.SystemLogService.Get(id)
	if item == nil {
		httpx.WriteJSON(ctx, httpx.JsonErrorMsg(ctx, "error.notFound"))
		return
	}
	httpx.WriteJSON(ctx, builders.BuildSystemLog(item))
}
