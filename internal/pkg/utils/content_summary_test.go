package utils

import "testing"

func TestBuildContentSummary(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		content     string
		maxRunes    int
		want        string
	}{
		{name: "markdown", contentType: "markdown", content: "# 标题\n\n**你好**，世界", maxRunes: 128, want: "标题 你好，世界"},
		{name: "html", contentType: "html", content: "<p>Hello&nbsp;<strong>world</strong></p><script>hidden()</script>", maxRunes: 128, want: "Hello world"},
		{name: "rune truncation", contentType: "markdown", content: "你好世界", maxRunes: 3, want: "你好世..."},
		{name: "plain text", contentType: "text", content: " first\n second ", maxRunes: 128, want: "first second"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := BuildContentSummary(tt.contentType, tt.content, tt.maxRunes); got != tt.want {
				t.Fatalf("BuildContentSummary() = %q, want %q", got, tt.want)
			}
		})
	}
}
