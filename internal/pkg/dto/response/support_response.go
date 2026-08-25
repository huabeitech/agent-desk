package response

import "agent-desk/internal/pkg/enums"

type ConfigFieldError struct {
	Path       string `json:"path"`
	Code       string `json:"code"`
	MessageKey string `json:"-"`
	Message    string `json:"message"`
}

type SupportUserResponse struct {
	ID       int64          `json:"id"`
	Name     string         `json:"name"`
	Email    string         `json:"email"`
	UserType enums.UserType `json:"userType"`
}

type SupportNavigationMenuItemResponse struct {
	ID              string                              `json:"id"`
	Title           string                              `json:"title"`
	URL             string                              `json:"url"`
	OpenInNewWindow bool                                `json:"openInNewWindow"`
	Visible         bool                                `json:"visible"`
	SortNo          int                                 `json:"sortNo"`
	Children        []SupportNavigationMenuItemResponse `json:"children,omitempty"`
}

type PublicSupportConfigResponse struct {
	NavigationMenu    []SupportNavigationMenuItemResponse    `json:"navigationMenu"`
	AICustomerService SupportAICustomerServiceConfigResponse `json:"aiCustomerService"`
}

type DashboardSupportConfigResponse struct {
	NavigationMenu    []SupportNavigationMenuItemResponse    `json:"navigationMenu"`
	AICustomerService SupportAICustomerServiceConfigResponse `json:"aiCustomerService"`
}

type SupportAICustomerServiceConfigResponse struct {
	Enabled   bool   `json:"enabled"`
	ChannelID string `json:"channelId"`
}

type SupportAICustomerServiceUserTokenResponse struct {
	UserToken string `json:"userToken"`
	ExpiresAt string `json:"expiresAt"`
}

type DocPageResponse struct {
	ID                        int64               `json:"id"`
	ParentID                  int64               `json:"parentId"`
	Title                     string              `json:"title"`
	Slug                      string              `json:"slug"`
	Summary                   string              `json:"summary"`
	ContentType               string              `json:"contentType"`
	Content                   string              `json:"content"`
	CoverURL                  string              `json:"coverUrl"`
	Tags                      []string            `json:"tags"`
	Status                    enums.DocPageStatus `json:"status"`
	SortNo                    int                 `json:"sortNo"`
	ViewCount                 int64               `json:"viewCount"`
	HelpfulCount              int64               `json:"helpfulCount"`
	UnhelpfulCount            int64               `json:"unhelpfulCount"`
	PublishedAt               string              `json:"publishedAt"`
	SyncedKnowledgeDocumentID int64               `json:"syncedKnowledgeDocumentId"`
	Remark                    string              `json:"remark"`
	CreatedAt                 string              `json:"createdAt"`
	UpdatedAt                 string              `json:"updatedAt"`
}

// DocPageNavigationResponse is the lightweight public document tree.
// Content remains available only from the doc-page detail endpoint.
type DocPageNavigationResponse struct {
	ID       int64                        `json:"id"`
	ParentID int64                        `json:"parentId"`
	Title    string                       `json:"title"`
	Slug     string                       `json:"slug"`
	SortNo   int                          `json:"sortNo"`
	Children []*DocPageNavigationResponse `json:"children"`
}

type CategoryResponse struct {
	ID          int64        `json:"id"`
	Name        string       `json:"name"`
	Slug        string       `json:"slug"`
	Description string       `json:"description"`
	SortNo      int          `json:"sortNo"`
	Status      enums.Status `json:"status"`
	Remark      string       `json:"remark"`
	CreatedAt   string       `json:"createdAt"`
	UpdatedAt   string       `json:"updatedAt"`
}

type PostResponse struct {
	ID                  int64                   `json:"id"`
	CategoryID          int64                   `json:"categoryId"`
	CategoryName        string                  `json:"categoryName"`
	UserID              int64                   `json:"userId"`
	UserName            string                  `json:"userName"`
	UserType            enums.UserType          `json:"userType"`
	Title               string                  `json:"title"`
	ContentType         string                  `json:"contentType"`
	Content             string                  `json:"content"`
	Tags                []string                `json:"tags"`
	Status              enums.PostStatus        `json:"status"`
	AcceptedCommentID   int64                   `json:"acceptedCommentId"`
	CommentCount        int64                   `json:"commentCount"`
	ReactionCount       int64                   `json:"reactionCount"`
	ViewCount           int64                   `json:"viewCount"`
	LastCommentedAt     string                  `json:"lastCommentedAt"`
	LastCommentUserType enums.CommentAuthorType `json:"lastCommentUserType"`
	LastCommentUserID   int64                   `json:"lastCommentUserId"`
	CreatedAt           string                  `json:"createdAt"`
	UpdatedAt           string                  `json:"updatedAt"`
}

type CommentResponse struct {
	ID            int64                   `json:"id"`
	PostID        int64                   `json:"postId"`
	ParentID      int64                   `json:"parentId"`
	AuthorType    enums.CommentAuthorType `json:"authorType"`
	AuthorID      int64                   `json:"authorId"`
	AuthorName    string                  `json:"authorName"`
	ContentType   string                  `json:"contentType"`
	Content       string                  `json:"content"`
	Status        enums.CommentStatus     `json:"status"`
	ReactionCount int64                   `json:"reactionCount"`
	ReplyCount    int64                   `json:"replyCount"`
	ReportCount   int64                   `json:"reportCount"`
	IsAccepted    bool                    `json:"isAccepted"`
	Replies       []CommentResponse       `json:"replies"`
	CreatedAt     string                  `json:"createdAt"`
	UpdatedAt     string                  `json:"updatedAt"`
}

type PostDetailResponse struct {
	Post     PostResponse      `json:"post"`
	Comments []CommentResponse `json:"comments"`
}
