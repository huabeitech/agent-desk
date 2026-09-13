package third

import (
	"bytes"
	"io"
	"net/http"
	"strings"

	"agent-desk/internal/services"

	"github.com/gin-gonic/gin"
)

// SlackPostWebhook receives incoming Events API payloads from Slack.
func SlackPostWebhook(ctx *gin.Context) {
	channelID := strings.TrimSpace(ctx.Param("channel_id"))
	if channelID == "" {
		channelID = strings.TrimSpace(ctx.Query("channel_id"))
	}

	timestampHeader := ctx.GetHeader("X-Slack-Request-Timestamp")
	signatureHeader := ctx.GetHeader("X-Slack-Signature")

	bodyBytes, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "failed to read body"})
		return
	}
	ctx.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	challenge, err := services.SlackInboundService.HandleWebhook(ctx.Request.Context(), channelID, timestampHeader, signatureHeader, bodyBytes)
	if err != nil {
		ctx.JSON(http.StatusOK, gin.H{"ok": false, "error": err.Error()})
		return
	}

	if challenge != nil {
		ctx.JSON(http.StatusOK, gin.H{"challenge": *challenge})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"ok": true})
}
