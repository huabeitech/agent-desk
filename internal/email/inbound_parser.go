package email

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

// UniversalInboundWebhook captures all possible field names across providers.
type UniversalInboundWebhook struct {
	// From fields
	From     string `json:"from"`
	FromCaps string `json:"From"`
	Sender   string `json:"sender"`
	FromName string `json:"from_name"`

	// To fields
	To        string `json:"to"`
	ToCaps    string `json:"To"`
	Recipient string `json:"recipient"`
	ToName    string `json:"to_name"`

	// Subject
	Subject     string `json:"subject"`
	SubjectCaps string `json:"Subject"`

	// Body fields
	Text        string `json:"text"`
	Body        string `json:"body"`
	BodyPlain   string `json:"body-plain"`
	TextBody    string `json:"TextBody"`
	RawTextBody string `json:"RawTextBody"`
	HTML        string `json:"html"`
	BodyHTML    string `json:"body-html"`
	HtmlBody    string `json:"HtmlBody"`
	RawHtmlBody string `json:"RawHtmlBody"`

	// Message IDs & Threading
	MessageID     string `json:"message_id"`
	MessageIDCaps string `json:"MessageID"`
	InReplyTo     string `json:"in_reply_to"`
	References    string `json:"references"`
	MailboxHash   string `json:"MailboxHash"`

	// Headers can be a map or an array of objects
	Headers any `json:"headers"`
	HeadersCaps any `json:"Headers"`

	// Brevo items array
	Items []BrevoInboundItem `json:"items"`
}

// ParseInboundWebhook parses raw webhook payload from various email providers into a slice of normalized InboundEmailPayloads.
func ParseInboundWebhook(contentType string, rawBody []byte, form url.Values) ([]InboundEmailPayload, error) {
	contentType = strings.ToLower(contentType)

	// 1. Form-data (Mailgun, SendGrid Inbound Parse)
	if len(form) > 0 {
		fromRaw := form.Get("from")
		if fromRaw == "" {
			fromRaw = form.Get("sender")
		}
		fromEmail, fromName := ParseAddress(fromRaw)

		toRaw := form.Get("to")
		if toRaw == "" {
			toRaw = form.Get("recipient")
		}
		if toRaw == "" {
			toRaw = form.Get("To")
		}
		toEmail, toName := ParseAddress(toRaw)

		bodyText := form.Get("body-plain")
		if bodyText == "" {
			bodyText = form.Get("stripped-text")
		}
		if bodyText == "" {
			bodyText = form.Get("text")
		}
		bodyHTML := form.Get("body-html")
		if bodyHTML == "" {
			bodyHTML = form.Get("stripped-html")
		}
		if bodyHTML == "" {
			bodyHTML = form.Get("html")
		}

		return []InboundEmailPayload{
			{
				FromEmail:  fromEmail,
				FromName:   fromName,
				ToEmail:    toEmail,
				ToName:     toName,
				Subject:    strings.TrimSpace(form.Get("subject")),
				BodyText:   strings.TrimSpace(bodyText),
				BodyHTML:   strings.TrimSpace(bodyHTML),
				MessageID:  strings.TrimSpace(form.Get("Message-Id")),
				InReplyTo:  strings.TrimSpace(form.Get("In-Reply-To")),
				References: strings.TrimSpace(form.Get("References")),
			},
		}, nil
	}

	rawStr := strings.TrimSpace(string(rawBody))
	if rawStr == "" {
		return nil, nil
	}

	// 2. Parse JSON
	var u UniversalInboundWebhook
	if err := json.Unmarshal(rawBody, &u); err != nil {
		return nil, fmt.Errorf("unmarshal email json failed: %w", err)
	}

	// Check Brevo items format
	if len(u.Items) > 0 {
		var results []InboundEmailPayload
		for _, item := range u.Items {
			fromEmail, fromName := ParseAddress(item.Sender)
			toEmail, toName := ParseAddress(item.Recipient)
			msgID := ""
			if len(item.UUID) > 0 {
				msgID = item.UUID[0]
			}
			results = append(results, InboundEmailPayload{
				FromEmail: fromEmail,
				FromName:  fromName,
				ToEmail:   toEmail,
				ToName:    toName,
				Subject:   strings.TrimSpace(item.Subject),
				BodyText:  strings.TrimSpace(item.RawTextBody),
				BodyHTML:  strings.TrimSpace(item.RawHTMLBody),
				MessageID: msgID,
				Headers:   item.Headers,
			})
		}
		return results, nil
	}

	// Resolve From
	fromRaw := firstNonEmpty(u.From, u.FromCaps, u.Sender)
	fromEmail, fromName := ParseAddress(fromRaw)
	if u.FromName != "" {
		fromName = u.FromName
	}

	// Resolve To
	toRaw := firstNonEmpty(u.To, u.ToCaps, u.Recipient)
	toEmail, toName := ParseAddress(toRaw)
	if u.ToName != "" {
		toName = u.ToName
	}

	// Resolve Subject
	subject := firstNonEmpty(u.Subject, u.SubjectCaps)

	// Resolve Body Text
	bodyText := firstNonEmpty(u.Text, u.Body, u.BodyPlain, u.TextBody, u.RawTextBody)

	// Resolve Body HTML
	bodyHTML := firstNonEmpty(u.HTML, u.BodyHTML, u.HtmlBody, u.RawHtmlBody)

	// Resolve Message ID
	messageID := firstNonEmpty(u.MessageID, u.MessageIDCaps)

	// Extract headers & In-Reply-To / References
	headersMap, inReplyTo, references := extractHeadersAndThreading(u.Headers, u.HeadersCaps, u.InReplyTo, u.References)

	if fromEmail == "" && toEmail == "" && subject == "" && bodyText == "" && bodyHTML == "" {
		return nil, fmt.Errorf("unrecognized email webhook format")
	}

	return []InboundEmailPayload{
		{
			FromEmail:  fromEmail,
			FromName:   fromName,
			ToEmail:    toEmail,
			ToName:     toName,
			Subject:    strings.TrimSpace(subject),
			BodyText:   strings.TrimSpace(bodyText),
			BodyHTML:   strings.TrimSpace(bodyHTML),
			MessageID:  strings.TrimSpace(messageID),
			InReplyTo:  strings.TrimSpace(inReplyTo),
			References: strings.TrimSpace(references),
			Headers:    headersMap,
		},
	}, nil
}

func extractHeadersAndThreading(headers1, headers2 any, fallbackInReply, fallbackRef string) (map[string]string, string, string) {
	headersMap := make(map[string]string)
	inReplyTo := fallbackInReply
	references := fallbackRef

	for _, h := range []any{headers1, headers2} {
		if h == nil {
			continue
		}
		switch val := h.(type) {
		case map[string]any:
			for k, v := range val {
				s := fmt.Sprintf("%v", v)
				headersMap[k] = s
				if strings.EqualFold(k, "In-Reply-To") && inReplyTo == "" {
					inReplyTo = s
				}
				if strings.EqualFold(k, "References") && references == "" {
					references = s
				}
			}
		case map[string]string:
			for k, v := range val {
				headersMap[k] = v
				if strings.EqualFold(k, "In-Reply-To") && inReplyTo == "" {
					inReplyTo = v
				}
				if strings.EqualFold(k, "References") && references == "" {
					references = v
				}
			}
		case []any:
			for _, item := range val {
				if itemMap, ok := item.(map[string]any); ok {
					name := fmt.Sprintf("%v", itemMap["Name"])
					if name == "" {
						name = fmt.Sprintf("%v", itemMap["name"])
					}
					value := fmt.Sprintf("%v", itemMap["Value"])
					if value == "" {
						value = fmt.Sprintf("%v", itemMap["value"])
					}
					if name != "" {
						headersMap[name] = value
						if strings.EqualFold(name, "In-Reply-To") && inReplyTo == "" {
							inReplyTo = value
						}
						if strings.EqualFold(name, "References") && references == "" {
							references = value
						}
					}
				}
			}
		}
	}

	return headersMap, inReplyTo, references
}

func firstNonEmpty(strs ...string) string {
	for _, s := range strs {
		if trimmed := strings.TrimSpace(s); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
