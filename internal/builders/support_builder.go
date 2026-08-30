package builders

import (
	"agent-desk/internal/models"
	"agent-desk/internal/pkg/dto/response"
	"agent-desk/internal/pkg/enums"
	"agent-desk/internal/pkg/utils"
	"encoding/json"
	"strconv"
	"time"
)

func BuildDocPage(item *models.DocPage, includeContent bool) *response.DocPageResponse {
	if item == nil {
		return nil
	}
	content := item.Content
	if !includeContent {
		content = ""
	}
	return &response.DocPageResponse{
		ID:                        item.ID,
		ParentID:                  item.ParentID,
		Title:                     item.Title,
		Slug:                      item.Slug,
		Summary:                   item.Summary,
		ContentType:               item.ContentType,
		Content:                   content,
		CoverURL:                  item.CoverURL,
		Tags:                      parseSupportTags(item.TagsJSON),
		Status:                    item.Status,
		SortNo:                    item.SortNo,
		ViewCount:                 item.ViewCount,
		HelpfulCount:              item.HelpfulCount,
		UnhelpfulCount:            item.UnhelpfulCount,
		PublishedAt:               formatSupportTime(item.PublishedAt),
		SyncedKnowledgeDocumentID: item.SyncedKnowledgeDocumentID,
		Remark:                    item.Remark,
		CreatedAt:                 formatSupportTime(&item.CreatedAt),
		UpdatedAt:                 formatSupportTime(&item.UpdatedAt),
	}
}

func BuildDocPageNavigationTree(list []models.DocPage) []*response.DocPageNavigationResponse {
	nodes := make(map[int64]*response.DocPageNavigationResponse, len(list))
	for i := range list {
		item := &list[i]
		nodes[item.ID] = &response.DocPageNavigationResponse{
			ID: item.ID, ParentID: item.ParentID, Title: item.Title, Slug: item.Slug, SortNo: item.SortNo,
			Children: make([]*response.DocPageNavigationResponse, 0),
		}
	}
	roots := make([]*response.DocPageNavigationResponse, 0)
	for i := range list {
		item := &list[i]
		node := nodes[item.ID]
		if item.ParentID == 0 || nodes[item.ParentID] == nil {
			roots = append(roots, node)
			continue
		}
		nodes[item.ParentID].Children = append(nodes[item.ParentID].Children, node)
	}
	return roots
}

func BuildCategory(item *models.Category) *response.CategoryResponse {
	if item == nil {
		return nil
	}
	return &response.CategoryResponse{
		ID:          item.ID,
		Name:        item.Name,
		Slug:        item.Slug,
		Description: item.Description,
		SortNo:      item.SortNo,
		Status:      item.Status,
		Remark:      item.Remark,
		CreatedAt:   formatSupportTime(&item.CreatedAt),
		UpdatedAt:   formatSupportTime(&item.UpdatedAt),
	}
}

func BuildPostCategories(list []models.Category) []response.CategoryResponse {
	ret := make([]response.CategoryResponse, 0, len(list))
	for _, item := range list {
		if resp := BuildCategory(&item); resp != nil {
			ret = append(ret, *resp)
		}
	}
	return ret
}

const communityPostSummaryLength = 128

func BuildSimpleUserInfo(item *models.User) response.SimpleUserInfo {
	if item == nil {
		return response.SimpleUserInfo{}
	}
	displayName := item.Nickname
	if displayName == "" {
		displayName = item.Username
	}
	return response.SimpleUserInfo{
		ID:          item.ID,
		Username:    item.Username,
		Nickname:    item.Nickname,
		DisplayName: displayName,
		Avatar:      utils.BuildAvatarURL(item.Avatar, "/api/avatar/user/"+strconv.FormatInt(item.ID, 10)),
		UserType:    item.UserType,
	}
}

func BuildPost(item *models.Post, categoryName string, user *models.User) *response.PostResponse {
	if item == nil {
		return nil
	}
	return &response.PostResponse{
		ID:                  item.ID,
		CategoryID:          item.CategoryID,
		CategoryName:        categoryName,
		User:                BuildSimpleUserInfo(user),
		Title:               item.Title,
		ContentType:         item.ContentType,
		Content:             item.Content,
		Tags:                parseSupportTags(item.TagsJSON),
		Status:              item.Status,
		AcceptedCommentID:   item.AcceptedCommentID,
		CommentCount:        item.CommentCount,
		ReactionCount:       item.ReactionCount,
		ViewCount:           item.ViewCount,
		LastCommentedAt:     formatSupportTime(item.LastCommentedAt),
		LastCommentUserType: item.LastCommentUserType,
		LastCommentUserID:   item.LastCommentUserID,
		CreatedAt:           formatSupportTime(&item.CreatedAt),
		UpdatedAt:           formatSupportTime(&item.UpdatedAt),
	}
}

func BuildPostList(items []models.Post, categories map[int64]*models.Category, users map[int64]*models.User) []response.PostListItemResponse {
	results := make([]response.PostListItemResponse, 0, len(items))
	for i := range items {
		item := &items[i]
		categoryName := ""
		if category := categories[item.CategoryID]; category != nil {
			categoryName = category.Name
		}
		results = append(results, response.PostListItemResponse{
			ID:                  item.ID,
			CategoryID:          item.CategoryID,
			CategoryName:        categoryName,
			User:                BuildSimpleUserInfo(users[item.UserID]),
			Title:               item.Title,
			Summary:             utils.BuildContentSummary(item.ContentType, item.Content, communityPostSummaryLength),
			Tags:                parseSupportTags(item.TagsJSON),
			Status:              item.Status,
			AcceptedCommentID:   item.AcceptedCommentID,
			CommentCount:        item.CommentCount,
			ReactionCount:       item.ReactionCount,
			ViewCount:           item.ViewCount,
			LastCommentedAt:     formatSupportTime(item.LastCommentedAt),
			LastCommentUserType: item.LastCommentUserType,
			LastCommentUserID:   item.LastCommentUserID,
			CreatedAt:           formatSupportTime(&item.CreatedAt),
			UpdatedAt:           formatSupportTime(&item.UpdatedAt),
		})
	}
	return results
}

func BuildComment(item *models.Comment, user *models.User) *response.CommentResponse {
	if item == nil {
		return nil
	}
	contentType := item.ContentType
	content := item.Content
	if item.Status == enums.CommentStatusDeleted {
		contentType = "markdown"
		content = ""
	}
	return &response.CommentResponse{
		ID:            item.ID,
		PostID:        item.PostID,
		ParentID:      item.ParentID,
		AuthorType:    item.AuthorType,
		User:          BuildSimpleUserInfo(user),
		ContentType:   contentType,
		Content:       content,
		Status:        item.Status,
		ReactionCount: item.ReactionCount,
		ReplyCount:    item.ReplyCount,
		ReportCount:   item.ReportCount,
		IsAccepted:    item.IsAccepted,
		Replies:       []response.CommentResponse{},
		CreatedAt:     formatSupportTime(&item.CreatedAt),
		UpdatedAt:     formatSupportTime(&item.UpdatedAt),
	}
}

func parseSupportTags(raw string) []string {
	var ret []string
	if raw == "" {
		return []string{}
	}
	if err := json.Unmarshal([]byte(raw), &ret); err != nil {
		return []string{}
	}
	return ret
}

func formatSupportTime(t *time.Time) string {
	if t == nil || t.IsZero() {
		return ""
	}
	return t.Format(time.DateTime)
}
