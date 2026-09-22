package email

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestParseAddress(t *testing.T) {
	tests := []struct {
		input     string
		wantEmail string
		wantName  string
	}{
		{"John Doe <john@example.com>", "john@example.com", "John Doe"},
		{"<support@crove.com>", "support@crove.com", ""},
		{"plain@example.com", "plain@example.com", ""},
		{"  Alice Smith <ALICE@DOMAIN.COM>  ", "alice@domain.com", "Alice Smith"},
		{"", "", ""},
	}

	for _, tt := range tests {
		gotEmail, gotName := ParseAddress(tt.input)
		if gotEmail != tt.wantEmail || gotName != tt.wantName {
			t.Errorf("ParseAddress(%q) = (%q, %q), want (%q, %q)", tt.input, gotEmail, gotName, tt.wantEmail, tt.wantName)
		}
	}
}

func TestParseInboundWebhook_Formats(t *testing.T) {
	// 1. Generic / Cloudflare format
	genericJSON := []byte(`{
		"from": "user@example.com",
		"from_name": "Test User",
		"to": "help@crove.com",
		"subject": "Hello Support",
		"text": "Please help with login.",
		"message_id": "<msg-001@example.com>"
	}`)

	items, err := ParseInboundWebhook("application/json", genericJSON, nil)
	if err != nil {
		t.Fatalf("ParseInboundWebhook generic failed: %v", err)
	}
	if len(items) != 1 || items[0].FromEmail != "user@example.com" || items[0].ToEmail != "help@crove.com" {
		t.Errorf("unexpected generic parsed output: %+v", items)
	}

	// 2. Brevo format
	brevoJSON := []byte(`{
		"items": [
			{
				"Sender": "Alice <alice@test.com>",
				"Recipient": "support@crove.com",
				"Subject": "Brevo Inquiry",
				"RawTextBody": "Brevo message text"
			}
		]
	}`)
	items, err = ParseInboundWebhook("application/json", brevoJSON, nil)
	if err != nil {
		t.Fatalf("ParseInboundWebhook brevo failed: %v", err)
	}
	if len(items) != 1 || items[0].FromEmail != "alice@test.com" || items[0].FromName != "Alice" {
		t.Errorf("unexpected brevo parsed output: %+v", items)
	}

	// 3. Postmark format
	postmarkJSON := []byte(`{
		"From": "bob@domain.org",
		"FromName": "Bob Developer",
		"To": "help@crove.com",
		"Subject": "Postmark Question",
		"TextBody": "Text from Postmark",
		"MessageID": "pm-12345",
		"Headers": [
			{"Name": "In-Reply-To", "Value": "<parent-msg@crove.com>"}
		]
	}`)
	items, err = ParseInboundWebhook("application/json", postmarkJSON, nil)
	if err != nil {
		t.Fatalf("ParseInboundWebhook postmark failed: %v", err)
	}
	if len(items) != 1 || items[0].FromEmail != "bob@domain.org" || items[0].InReplyTo != "<parent-msg@crove.com>" {
		t.Errorf("unexpected postmark parsed output: %+v", items)
	}

	// 4. Mailgun form data
	mgForm := url.Values{}
	mgForm.Set("sender", "developer@client.com")
	mgForm.Set("from", "Dev <developer@client.com>")
	mgForm.Set("recipient", "help@crove.com")
	mgForm.Set("subject", "Mailgun Support")
	mgForm.Set("body-plain", "Plain body from mailgun")
	mgForm.Set("In-Reply-To", "<msg-999@crove.com>")

	items, err = ParseInboundWebhook("application/x-www-form-urlencoded", nil, mgForm)
	if err != nil {
		t.Fatalf("ParseInboundWebhook mailgun failed: %v", err)
	}
	if len(items) != 1 || items[0].FromEmail != "developer@client.com" || items[0].InReplyTo != "<msg-999@crove.com>" {
		t.Errorf("unexpected mailgun parsed output: %+v", items)
	}
}

func TestSendGridSendEmail(t *testing.T) {
	var receivedAuth string
	var receivedPath string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("Authorization")
		receivedPath = r.URL.Path
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	client := NewClient(ClientConfig{
		Provider: ProviderSendGrid,
		APIKey:   "SG.test-key",
		BaseURL:  server.URL,
	})

	err := client.SendEmail(context.Background(), SendEmailParams{
		FromEmail: "support@crove.com",
		ToEmail:   "user@example.com",
		Subject:   "SendGrid Test",
		BodyText:  "Hello via SendGrid",
	})

	if err != nil {
		t.Fatalf("SendEmail SendGrid failed: %v", err)
	}
	if receivedAuth != "Bearer SG.test-key" {
		t.Errorf("expected Bearer SG.test-key, got %s", receivedAuth)
	}
	if receivedPath != "/mail/send" {
		t.Errorf("expected /mail/send, got %s", receivedPath)
	}
}

func TestResendSendEmail(t *testing.T) {
	var receivedAuth string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"resend-123"}`))
	}))
	defer server.Close()

	client := NewClient(ClientConfig{
		Provider: ProviderResend,
		APIKey:   "re_test_123",
		BaseURL:  server.URL,
	})

	err := client.SendEmail(context.Background(), SendEmailParams{
		FromEmail: "support@crove.com",
		ToEmail:   "user@example.com",
		Subject:   "Resend Test",
		BodyText:  "Hello via Resend",
	})

	if err != nil {
		t.Fatalf("SendEmail Resend failed: %v", err)
	}
	if receivedAuth != "Bearer re_test_123" {
		t.Errorf("expected Bearer re_test_123, got %s", receivedAuth)
	}
}
