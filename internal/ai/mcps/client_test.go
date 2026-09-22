package mcps

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"
)

func TestClient_SystemServer(t *testing.T) {
	handler := NewHTTPHandler()
	server := httptest.NewServer(handler)
	defer server.Close()

	client := NewClient()
	cfg := ServerConfig{
		Code:      "system",
		Endpoint:  server.URL,
		TimeoutMS: 5000,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := client.TestConnection(ctx, cfg)
	if err != nil {
		t.Fatalf("TestConnection failed: %v", err)
	}
	if conn.ServerName != "agent-desk-mcp-server" {
		t.Errorf("expected ServerName 'agent-desk-mcp-server', got %s", conn.ServerName)
	}

	tools, err := client.ListTools(ctx, cfg)
	if err != nil {
		t.Fatalf("ListTools failed: %v", err)
	}
	if len(tools) == 0 {
		t.Errorf("expected at least 1 tool, got %d", len(tools))
	}
}
