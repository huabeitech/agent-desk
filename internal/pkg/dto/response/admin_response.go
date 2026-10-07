package response

import "agent-desk/internal/pkg/enums"

type PermissionResponse struct {
	ID        int64        `json:"id"`
	Name      string       `json:"name"`
	Code      string       `json:"code"`
	Type      string       `json:"type"`
	GroupName string       `json:"groupName"`
	Method    string       `json:"method"`
	ApiPath   string       `json:"apiPath"`
	Status    enums.Status `json:"status"`
	SortNo    int          `json:"sortNo"`
}

type PermissionSyncResponse struct {
	Created              int `json:"created"`
	Updated              int `json:"updated"`
	RolePermissionsAdded int `json:"rolePermissionsAdded"`
}

type RoleResponse struct {
	ID          int64        `json:"id"`
	Name        string       `json:"name"`
	Code        string       `json:"code"`
	Status      enums.Status `json:"status"`
	IsSystem    bool         `json:"isSystem"`
	SortNo      int          `json:"sortNo"`
	Permissions []string     `json:"permissions,omitempty"`
}

type UserResponse struct {
	ID            int64          `json:"id"`
	Username      string         `json:"username"`
	Nickname      string         `json:"nickname"`
	Avatar        string         `json:"avatar"`
	AvatarAssetID string         `json:"avatarAssetId,omitempty"`
	Mobile        string         `json:"mobile,omitempty"`
	Email         string         `json:"email,omitempty"`
	UserType      enums.UserType `json:"userType"`
	Status        enums.Status   `json:"status"`
	LastLoginAt   string         `json:"lastLoginAt,omitempty"`
	LastLoginIP   string         `json:"lastLoginIp,omitempty"`
	Roles         []RoleResponse `json:"roles,omitempty"`
	Permissions   []string       `json:"permissions,omitempty"`
}

// CreateUserResultResponse 创建用户成功响应；password 仅在本次响应中返回一次。
type CreateUserResultResponse struct {
	User     *UserResponse `json:"user"`
	Password string        `json:"password"`
}

type SessionResponse struct {
	ID         int64  `json:"id"`
	UserID     int64  `json:"userId"`
	Username   string `json:"username"`
	ClientType string `json:"clientType"`
	ClientIP   string `json:"clientIp"`
	UserAgent  string `json:"userAgent"`
	ExpiredAt  string `json:"expiredAt"`
	RevokedAt  string `json:"revokedAt"`
	LastSeenAt string `json:"lastSeenAt"`
}
