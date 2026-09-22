package request

type OrganizationSwitchRequest struct {
	OrganizationID int64 `json:"organizationId"`
}

type OrganizationCreateRequest struct {
	Name string `json:"name"`
	Code string `json:"code"`
	Logo string `json:"logo"`
}

type OrganizationUpdateRequest struct {
	Name string `json:"name"`
	Logo string `json:"logo"`
}

type OrganizationAddMemberRequest struct {
	EmailOrUsername string `json:"emailOrUsername"`
	Role            string `json:"role"`
}

type OrganizationRemoveMemberRequest struct {
	UserID int64 `json:"userId"`
}
