package third

import (
	"bytes"
	"io"
	"net/http"
	"net/url"
	"strings"

	"agent-desk/internal/services"

	"github.com/gin-gonic/gin"
)

// EmailPostWebhook receives incoming inbound email webhook events from Cloudflare, Brevo, SendGrid, Postmark, Mailgun, or SMTP forwarders.
func EmailPostWebhook(ctx *gin.Context) {
	channelID := strings.TrimSpace(ctx.Param("channel_id"))
	if channelID == "" {
		channelID = strings.TrimSpace(ctx.Query("channel_id"))
	}

	secretHeader := ctx.GetHeader("X-Webhook-Secret")
	if secretHeader == "" {
		secretHeader = ctx.GetHeader("X-Brevo-Webhook-Secret")
	}
	if secretHeader == "" {
		secretHeader = ctx.GetHeader("X-Postmark-Webhook-Secret")
	}
	if secretHeader == "" {
		secretHeader = ctx.Query("secret")
	}

	contentType := ctx.GetHeader("Content-Type")

	var formValues url.Values
	var bodyBytes []byte

	if strings.Contains(strings.ToLower(contentType), "multipart/form-data") {
		if err := ctx.Request.ParseMultipartForm(32 << 20); err == nil && ctx.Request.MultipartForm != nil {
			formValues = ctx.Request.MultipartForm.Value
		}
	} else if strings.Contains(strings.ToLower(contentType), "application/x-www-form-urlencoded") {
		if err := ctx.Request.ParseForm(); err == nil {
			formValues = ctx.Request.PostForm
		}
	}

	if ctx.Request.Body != nil {
		bodyBytes, _ = io.ReadAll(ctx.Request.Body)
		ctx.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	}

	if err := services.EmailInboundService.HandleWebhook(ctx.Request.Context(), channelID, secretHeader, contentType, bodyBytes, formValues); err != nil {
		ctx.JSON(http.StatusOK, gin.H{"ok": false, "error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"ok": true, "message": "email processed"})
}
