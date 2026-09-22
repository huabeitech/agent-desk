package dashboard

import (
	"agent-desk/internal/pkg/dto/request"
	"agent-desk/internal/pkg/errorsx"
	"agent-desk/internal/pkg/httpx"
	"agent-desk/internal/pkg/httpx/params"
	"agent-desk/internal/services"

	"github.com/gin-gonic/gin"
)

func OrganizationUserList(ctx *gin.Context) {
	principal := services.AuthService.GetAuthPrincipal(ctx)
	if principal == nil {
		httpx.WriteJSON(ctx, nil)
		return
	}

	ret, err := services.OrganizationService.GetUserOrganizations(principal.UserID)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	httpx.WriteJSON(ctx, ret)
}

func OrganizationPostCreate(ctx *gin.Context) {
	principal := services.AuthService.GetAuthPrincipal(ctx)
	if principal == nil {
		httpx.WriteJSON(ctx, errorsx.UnauthorizedI18n("error.auth.expired"))
		return
	}

	req := request.OrganizationCreateRequest{}
	if err := params.ReadJSON(ctx, &req); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}

	ret, err := services.OrganizationService.CreateOrganization(principal.UserID, req)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	httpx.WriteJSON(ctx, ret)
}

func OrganizationSwitch(ctx *gin.Context) {
	principal := services.AuthService.GetAuthPrincipal(ctx)
	if principal == nil {
		httpx.WriteJSON(ctx, errorsx.UnauthorizedI18n("error.auth.expired"))
		return
	}

	req := request.OrganizationSwitchRequest{}
	if err := params.ReadJSON(ctx, &req); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}

	org, err := services.OrganizationService.SwitchActiveOrganization(principal.UserID, req.OrganizationID)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}

	httpx.WriteJSON(ctx, org)
}

func OrganizationGetMembers(ctx *gin.Context) {
	principal := services.AuthService.GetAuthPrincipal(ctx)
	if principal == nil {
		httpx.WriteJSON(ctx, errorsx.UnauthorizedI18n("error.auth.expired"))
		return
	}

	activeOrg := services.OrganizationService.GetActiveOrganization(principal)
	if activeOrg == nil {
		httpx.WriteJSON(ctx, []any{})
		return
	}

	members, err := services.OrganizationService.GetOrganizationMembers(principal.UserID, activeOrg.ID)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	httpx.WriteJSON(ctx, members)
}

func OrganizationPostAddMember(ctx *gin.Context) {
	principal := services.AuthService.GetAuthPrincipal(ctx)
	if principal == nil {
		httpx.WriteJSON(ctx, errorsx.UnauthorizedI18n("error.auth.expired"))
		return
	}

	activeOrg := services.OrganizationService.GetActiveOrganization(principal)
	if activeOrg == nil {
		httpx.WriteJSON(ctx, errorsx.InvalidParam("no active organization selected"))
		return
	}

	req := request.OrganizationAddMemberRequest{}
	if err := params.ReadJSON(ctx, &req); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}

	member, err := services.OrganizationService.AddMember(principal.UserID, activeOrg.ID, req)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	httpx.WriteJSON(ctx, member)
}

func OrganizationPostRemoveMember(ctx *gin.Context) {
	principal := services.AuthService.GetAuthPrincipal(ctx)
	if principal == nil {
		httpx.WriteJSON(ctx, errorsx.UnauthorizedI18n("error.auth.expired"))
		return
	}

	activeOrg := services.OrganizationService.GetActiveOrganization(principal)
	if activeOrg == nil {
		httpx.WriteJSON(ctx, errorsx.InvalidParam("no active organization selected"))
		return
	}

	req := request.OrganizationRemoveMemberRequest{}
	if err := params.ReadJSON(ctx, &req); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}

	if req.UserID <= 0 {
		httpx.WriteJSON(ctx, errorsx.InvalidParam("invalid user id"))
		return
	}

	if err := services.OrganizationService.RemoveMember(principal.UserID, activeOrg.ID, req.UserID); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	httpx.WriteJSON(ctx, gin.H{"success": true})
}

func OrganizationPostUpdate(ctx *gin.Context) {
	principal := services.AuthService.GetAuthPrincipal(ctx)
	if principal == nil {
		httpx.WriteJSON(ctx, errorsx.UnauthorizedI18n("error.auth.expired"))
		return
	}

	activeOrg := services.OrganizationService.GetActiveOrganization(principal)
	if activeOrg == nil {
		httpx.WriteJSON(ctx, errorsx.InvalidParam("no active organization selected"))
		return
	}

	req := request.OrganizationUpdateRequest{}
	if err := params.ReadJSON(ctx, &req); err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}

	org, err := services.OrganizationService.UpdateOrganization(principal.UserID, activeOrg.ID, req)
	if err != nil {
		httpx.WriteJSON(ctx, err)
		return
	}
	httpx.WriteJSON(ctx, org)
}
