package utils

import (
	"strings"

	"github.com/gomarkdown/markdown"
	"golang.org/x/net/html"
)

// BuildContentSummary converts Markdown or HTML to plain text before truncating
// it by runes. This follows the bbs-go summary flow while keeping the helper
// independent from any feature-specific package.
func BuildContentSummary(contentType, content string, maxRunes int) string {
	content = strings.TrimSpace(content)
	if content == "" || maxRunes <= 0 {
		return ""
	}
	if strings.EqualFold(contentType, "markdown") {
		content = string(markdown.ToHTML([]byte(content), nil, nil))
	}
	if strings.EqualFold(contentType, "markdown") || strings.EqualFold(contentType, "html") {
		content = ExtractHTMLPlainText(content)
	} else {
		content = normalizeSummaryWhitespace(content)
	}

	runes := []rune(content)
	if len(runes) <= maxRunes {
		return content
	}
	return string(runes[:maxRunes]) + "..."
}

func ExtractHTMLPlainText(content string) string {
	doc, err := html.Parse(strings.NewReader("<div>" + content + "</div>"))
	if err != nil {
		return normalizeSummaryWhitespace(content)
	}
	var builder strings.Builder
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node == nil {
			return
		}
		if node.Type == html.ElementNode && (node.Data == "script" || node.Data == "style") {
			return
		}
		if node.Type == html.ElementNode && isSummaryBlockElement(node.Data) {
			builder.WriteByte(' ')
		}
		if node.Type == html.TextNode {
			builder.WriteString(node.Data)
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
		if node.Type == html.ElementNode && isSummaryBlockElement(node.Data) {
			builder.WriteByte(' ')
		}
	}
	walk(doc)
	return normalizeSummaryWhitespace(builder.String())
}

func isSummaryBlockElement(tag string) bool {
	switch tag {
	case "p", "div", "br", "li", "ul", "ol", "blockquote", "pre", "table", "tr", "td", "th", "h1", "h2", "h3", "h4", "h5", "h6":
		return true
	default:
		return false
	}
}

func normalizeSummaryWhitespace(content string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(content)), " ")
}
