package email

// DeliveryProvider identifies email sending service.
type DeliveryProvider string

const (
	ProviderSMTP     DeliveryProvider = "smtp"
	ProviderBrevo    DeliveryProvider = "brevo"
	ProviderSendGrid DeliveryProvider = "sendgrid"
	ProviderResend   DeliveryProvider = "resend"
	ProviderPostmark DeliveryProvider = "postmark"
	ProviderMailgun  DeliveryProvider = "mailgun"
)

// SendEmailParams defines parameters for sending an email.
type SendEmailParams struct {
	FromEmail   string            `json:"fromEmail"`
	FromName    string            `json:"fromName,omitempty"`
	ToEmail     string            `json:"toEmail"`
	ToName      string            `json:"toName,omitempty"`
	ReplyTo     string            `json:"replyTo,omitempty"`
	Subject     string            `json:"subject"`
	BodyText    string            `json:"bodyText"`
	BodyHTML    string            `json:"bodyHtml,omitempty"`
	MessageID   string            `json:"messageId,omitempty"`
	InReplyTo   string            `json:"inReplyTo,omitempty"`
	References  string            `json:"references,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
	Attachments []EmailAttachment `json:"attachments,omitempty"`
}

// EmailAttachment defines file attachments in email.
type EmailAttachment struct {
	Filename    string `json:"filename"`
	ContentType string `json:"contentType"`
	ContentB64  string `json:"contentB64"`
}

// ClientConfig holds configuration for initializing Email client.
type ClientConfig struct {
	Provider     DeliveryProvider
	APIKey       string
	BaseURL      string
	SMTPHost     string
	SMTPPort     int
	SMTPUser     string
	SMTPPassword string
	SMTPUseTLS   bool
	Domain       string // For Mailgun (e.g. mg.example.com)
}

// InboundEmailPayload represents normalized parsed inbound email.
type InboundEmailPayload struct {
	FromEmail   string            `json:"fromEmail"`
	FromName    string            `json:"fromName,omitempty"`
	ToEmail     string            `json:"toEmail"`
	ToName      string            `json:"toName,omitempty"`
	Subject     string            `json:"subject"`
	BodyText    string            `json:"bodyText"`
	BodyHTML    string            `json:"bodyHtml,omitempty"`
	MessageID   string            `json:"messageId,omitempty"`
	InReplyTo   string            `json:"inReplyTo,omitempty"`
	References  string            `json:"references,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
	Attachments []EmailAttachment `json:"attachments,omitempty"`
}

// --- Provider specific Inbound & Outbound Structs ---

// BrevoSendEmailRequest payload for Brevo v3 API.
type BrevoSendEmailRequest struct {
	Sender      BrevoEmailContact   `json:"sender"`
	To          []BrevoEmailContact `json:"to"`
	ReplyTo     *BrevoEmailContact  `json:"replyTo,omitempty"`
	Subject     string              `json:"subject"`
	HTMLContent string              `json:"htmlContent,omitempty"`
	TextContent string              `json:"textContent,omitempty"`
	Headers     map[string]string   `json:"headers,omitempty"`
}

type BrevoEmailContact struct {
	Name  string `json:"name,omitempty"`
	Email string `json:"email"`
}

// SendGridSendEmailRequest payload for SendGrid v3 API.
type SendGridSendEmailRequest struct {
	Personalizations []SendGridPersonalization `json:"personalizations"`
	From             SendGridContact           `json:"from"`
	ReplyTo          *SendGridContact          `json:"reply_to,omitempty"`
	Subject          string                    `json:"subject"`
	Content          []SendGridContent         `json:"content"`
	Headers          map[string]string         `json:"headers,omitempty"`
}

type SendGridPersonalization struct {
	To      []SendGridContact `json:"to"`
	Subject string            `json:"subject,omitempty"`
}

type SendGridContact struct {
	Email string `json:"email"`
	Name  string `json:"name,omitempty"`
}

type SendGridContent struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// ResendSendEmailRequest payload for Resend API.
type ResendSendEmailRequest struct {
	From        string            `json:"from"`
	To          []string          `json:"to"`
	ReplyTo     string            `json:"reply_to,omitempty"`
	Subject     string            `json:"subject"`
	Text        string            `json:"text,omitempty"`
	HTML        string            `json:"html,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
	Attachments []ResendAttachment`json:"attachments,omitempty"`
}

type ResendAttachment struct {
	Filename string `json:"filename"`
	Content  string `json:"content"`
}

// PostmarkSendEmailRequest payload for Postmark API.
type PostmarkSendEmailRequest struct {
	From        string            `json:"From"`
	To          string            `json:"To"`
	ReplyTo     string            `json:"ReplyTo,omitempty"`
	Subject     string            `json:"Subject"`
	TextBody    string            `json:"TextBody,omitempty"`
	HtmlBody    string            `json:"HtmlBody,omitempty"`
	Headers     []PostmarkHeader  `json:"Headers,omitempty"`
	Attachments []PostmarkAttach  `json:"Attachments,omitempty"`
}

type PostmarkHeader struct {
	Name  string `json:"Name"`
	Value string `json:"Value"`
}

type PostmarkAttach struct {
	Name        string `json:"Name"`
	Content     string `json:"Content"`
	ContentType string `json:"ContentType"`
}

// GenericInboundWebhook represents standard webhook JSON format (Cloudflare Email Routing, AWS SES, Generic).
type GenericInboundWebhook struct {
	From       string            `json:"from"`
	FromName   string            `json:"from_name,omitempty"`
	To         string            `json:"to"`
	ToName     string            `json:"to_name,omitempty"`
	Subject    string            `json:"subject"`
	Text       string            `json:"text,omitempty"`
	HTML       string            `json:"html,omitempty"`
	Body       string            `json:"body,omitempty"`
	MessageID  string            `json:"message_id,omitempty"`
	InReplyTo  string            `json:"in_reply_to,omitempty"`
	References string            `json:"references,omitempty"`
	Headers    map[string]string `json:"headers,omitempty"`
}

// BrevoInboundItem represents an item in Brevo inbound webhook.
type BrevoInboundItem struct {
	UUID        []string          `json:"Uuid,omitempty"`
	Sender      string            `json:"Sender,omitempty"`
	Recipient   string            `json:"Recipient,omitempty"`
	Subject     string            `json:"Subject,omitempty"`
	RawHTMLBody string            `json:"RawHtmlBody,omitempty"`
	RawTextBody string            `json:"RawTextBody,omitempty"`
	Headers     map[string]string `json:"Headers,omitempty"`
}

type BrevoInboundWebhook struct {
	Items []BrevoInboundItem `json:"items,omitempty"`
}

// PostmarkInboundWebhook represents Postmark inbound email payload.
type PostmarkInboundWebhook struct {
	From        string            `json:"From"`
	FromName    string            `json:"FromName,omitempty"`
	To          string            `json:"To"`
	Subject     string            `json:"Subject"`
	TextBody    string            `json:"TextBody,omitempty"`
	HtmlBody    string            `json:"HtmlBody,omitempty"`
	MessageID   string            `json:"MessageID,omitempty"`
	MailboxHash string            `json:"MailboxHash,omitempty"`
	Headers     []PostmarkHeader  `json:"Headers,omitempty"`
}
