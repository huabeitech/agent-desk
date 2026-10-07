package request

import "agent-desk/internal/pkg/enums"

type SupportCustomerRegisterRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type SaveDocPageRequest struct {
	ID          int64               `json:"id"`
	ParentID    int64               `json:"parentId"`
	Title       string              `json:"title"`
	Slug        string              `json:"slug"`
	Summary     string              `json:"summary"`
	ContentType string              `json:"contentType"`
	Content     string              `json:"content"`
	CoverURL    string              `json:"coverUrl"`
	Tags        []string            `json:"tags"`
	Status      enums.DocPageStatus `json:"status"`
	SortNo      int                 `json:"sortNo"`
	Remark      string              `json:"remark"`
}

type UpdateDocPageSettingsRequest struct {
	ID       int64  `json:"id"`
	ParentID int64  `json:"parentId"`
	Slug     string `json:"slug"`
	Summary  string `json:"summary"`
}

type SortDocPagesRequest struct {
	ParentID int64   `json:"parentId"`
	IDs      []int64 `json:"ids"`
}

type ChangeDocPageStatusRequest struct {
	ID     int64               `json:"id"`
	Status enums.DocPageStatus `json:"status"`
}

type SaveCategoryRequest struct {
	ID          int64        `json:"id"`
	Name        string       `json:"name"`
	Slug        string       `json:"slug"`
	Description string       `json:"description"`
	Status      enums.Status `json:"status"`
	Remark      string       `json:"remark"`
}

type CreatePostRequest struct {
	CategoryID  int64    `json:"categoryId"`
	Title       string   `json:"title"`
	ContentType string   `json:"contentType"`
	Content     string   `json:"content"`
	Tags        []string `json:"tags"`
}

type UpdatePostRequest struct {
	ID          int64    `json:"id"`
	CategoryID  int64    `json:"categoryId"`
	Title       string   `json:"title"`
	ContentType string   `json:"contentType"`
	Content     string   `json:"content"`
	Tags        []string `json:"tags"`
}

type ModeratePostRequest struct {
	ID     int64            `json:"id"`
	Status enums.PostStatus `json:"status"`
}

type CreateCommentRequest struct {
	PostID      int64  `json:"postId"`
	ParentID    int64  `json:"parentId"`
	ContentType string `json:"contentType"`
	Content     string `json:"content"`
}

type UpdateCommentRequest struct {
	ID          int64  `json:"id"`
	ContentType string `json:"contentType"`
	Content     string `json:"content"`
}

type ModerateCommentRequest struct {
	ID     int64               `json:"id"`
	Status enums.CommentStatus `json:"status"`
}

type IDRequest struct {
	ID int64 `json:"id"`
}

type ReactionRequest struct {
	TargetType   enums.ReactionTarget `json:"targetType"`
	TargetID     int64                `json:"targetId"`
	ReactionType enums.ReactionType   `json:"reactionType"`
}

type ReportCommentRequest struct {
	ID     int64  `json:"id"`
	Reason string `json:"reason"`
}

type AcceptCommentRequest struct {
	PostID    int64 `json:"postId"`
	CommentID int64 `json:"commentId"`
}

type DocPageFeedbackRequest struct {
	ID      int64 `json:"id"`
	Helpful bool  `json:"helpful"`
}

type DeleteByIDRequest struct {
	ID int64 `json:"id"`
}

type SupportNavigationMenuItemRequest struct {
	ID              string                             `json:"id"`
	Title           string                             `json:"title"`
	URL             string                             `json:"url"`
	OpenInNewWindow bool                               `json:"openInNewWindow"`
	Visible         *bool                              `json:"visible"`
	Children        []SupportNavigationMenuItemRequest `json:"children"`
}

type SupportAICustomerServiceConfigRequest struct {
	Enabled   bool   `json:"enabled"`
	ChannelID string `json:"channelId"`
}
