package builders

import (
	"agent-desk/internal/models"
	"agent-desk/internal/pkg/enums"
	"encoding/json"
	"strings"
	"testing"
)

func TestBuildPostListUsesSimpleUserInfoAndSummary(t *testing.T) {
	posts := []models.Post{{
		ID:          1,
		CategoryID:  2,
		UserID:      3,
		Title:       "标题",
		ContentType: "markdown",
		Content:     "# 正文\n\n这是帖子内容",
		Status:      enums.PostStatusNormal,
	}}
	items := BuildPostList(posts,
		map[int64]*models.Category{2: {ID: 2, Name: "产品使用"}},
		map[int64]*models.User{3: {ID: 3, Username: "alice", Nickname: "Alice", Avatar: "https://example.com/a.png", UserType: enums.UserTypeUser}},
	)
	if len(items) != 1 {
		t.Fatalf("items length = %d, want 1", len(items))
	}
	item := items[0]
	if item.Summary != "正文 这是帖子内容" {
		t.Fatalf("summary = %q", item.Summary)
	}
	if item.User.ID != 3 || item.User.DisplayName != "Alice" || item.User.Avatar == "" {
		t.Fatalf("unexpected simple user info: %#v", item.User)
	}
	encoded, err := json.Marshal(item)
	if err != nil {
		t.Fatalf("marshal post list item: %v", err)
	}
	if strings.Contains(string(encoded), "content") {
		t.Fatalf("post list item must not include full content: %s", encoded)
	}
}

func TestBuildCommentUsesSimpleUserInfo(t *testing.T) {
	comment := BuildComment(&models.Comment{ID: 1, AuthorID: 2, ContentType: "markdown", Content: "评论", Status: enums.CommentStatusNormal}, &models.User{
		ID: 2, Username: "bob", Nickname: "Bob", UserType: enums.UserTypeUser,
	})
	if comment.User.ID != 2 || comment.User.DisplayName != "Bob" {
		t.Fatalf("unexpected simple user info: %#v", comment.User)
	}
	encoded, err := json.Marshal(comment)
	if err != nil {
		t.Fatalf("marshal comment: %v", err)
	}
	if strings.Contains(string(encoded), "authorId") || strings.Contains(string(encoded), "authorName") {
		t.Fatalf("comment must use user instead of flat author fields: %s", encoded)
	}
}
