package mcp

import (
	"context"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestIsLoopbackAddr(t *testing.T) {
	cases := map[string]bool{
		"localhost:8089":   true,
		"127.0.0.1:8089":   true,
		"[::1]:8089":       true,
		":8089":            true,
		"0.0.0.0:8089":     false,
		"192.168.1.10:80":  false,
		"10.0.0.5:9090":    false,
		"[2001:db8::1]:80": false,
		// A hostname can resolve anywhere, so it is not verifiably loopback and
		// must go through the explicit allow-remote opt-in.
		"unknownhost:8089": false,
	}
	for addr, want := range cases {
		if got := isLoopbackAddr(addr); got != want {
			t.Errorf("isLoopbackAddr(%q) = %v, want %v", addr, got, want)
		}
	}
}

func TestTokenFromRequest(t *testing.T) {
	r, _ := http.NewRequest("POST", "/mcp", nil)
	r.Header.Set("Authorization", "Bearer abc123")
	if got := tokenFromRequest(r); got != "abc123" {
		t.Errorf("Bearer header: got %q", got)
	}

	r2, _ := http.NewRequest("POST", "/mcp", nil)
	r2.Header.Set("X-RedC-Token", "xyz789")
	if got := tokenFromRequest(r2); got != "xyz789" {
		t.Errorf("X-RedC-Token header: got %q", got)
	}

	// Query parameters must not be accepted as a token source.
	r3, _ := http.NewRequest("POST", "/mcp?token=leaked", nil)
	if got := tokenFromRequest(r3); got != "" {
		t.Errorf("query param must not be accepted, got %q", got)
	}
}

// startSSE starts a manager on an ephemeral loopback port and returns its address.
func startSSE(t *testing.T, token string) (*MCPServerManager, string) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skipf("cannot allocate local port: %v", err)
	}
	addr := ln.Addr().String()
	ln.Close()

	m := NewMCPServerManager(nil, nil)
	m.SetLogCallback(func(string) {})
	m.SetToken(token)
	if err := m.Start(TransportSSE, addr); err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() { _ = m.Stop() })

	// Wait for the listener to accept connections.
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		c, err := net.DialTimeout("tcp", addr, 200*time.Millisecond)
		if err == nil {
			c.Close()
			return m, addr
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("server did not become reachable on %s", addr)
	return nil, ""
}

func postMCP(t *testing.T, addr, path string, headers map[string]string) *http.Response {
	t.Helper()
	body := strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`)
	req, err := http.NewRequest("POST", "http://"+addr+path, body)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	t.Cleanup(func() { resp.Body.Close() })
	return resp
}

func TestSSERejectsMissingToken(t *testing.T) {
	_, addr := startSSE(t, "secret-token")
	for _, path := range []string{"/mcp", "/sse", "/message"} {
		resp := postMCP(t, addr, path, nil)
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("%s without token: got %d, want 401", path, resp.StatusCode)
		}
	}
}

func TestSSERejectsWrongToken(t *testing.T) {
	_, addr := startSSE(t, "secret-token")
	resp := postMCP(t, addr, "/mcp", map[string]string{"Authorization": "Bearer wrong"})
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("wrong token: got %d, want 401", resp.StatusCode)
	}
}

func TestSSEAcceptsBearerToken(t *testing.T) {
	_, addr := startSSE(t, "secret-token")
	resp := postMCP(t, addr, "/mcp", map[string]string{"Authorization": "Bearer secret-token"})
	if resp.StatusCode == http.StatusUnauthorized {
		t.Fatalf("valid bearer token was rejected")
	}
}

func TestSSEAcceptsXRedCToken(t *testing.T) {
	_, addr := startSSE(t, "secret-token")
	resp := postMCP(t, addr, "/message", map[string]string{"X-RedC-Token": "secret-token"})
	if resp.StatusCode == http.StatusUnauthorized {
		t.Fatalf("valid X-RedC-Token was rejected")
	}
}

func TestSSEDoesNotAllowWildcardOrigin(t *testing.T) {
	_, addr := startSSE(t, "secret-token")
	resp := postMCP(t, addr, "/mcp", map[string]string{"Authorization": "Bearer secret-token"})
	if got := resp.Header.Get("Access-Control-Allow-Origin"); got == "*" {
		t.Errorf("Access-Control-Allow-Origin must not be a wildcard, got %q", got)
	}
}

func TestSSERefusesNonLoopbackBind(t *testing.T) {
	m := NewMCPServerManager(nil, nil)
	m.SetLogCallback(func(string) {})
	m.SetToken("t")
	err := m.Start(TransportSSE, "0.0.0.0:0")
	if err == nil {
		_ = m.Stop()
		t.Fatal("expected Start to refuse a non-loopback bind without allowRemote")
	}
	if !strings.Contains(err.Error(), "non-loopback") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestSSERefusesNonLoopbackBindWithoutToken(t *testing.T) {
	m := NewMCPServerManager(nil, nil)
	m.SetLogCallback(func(string) {})
	m.SetAllowRemote(true)
	err := m.Start(TransportSSE, "0.0.0.0:0")
	if err == nil {
		_ = m.Stop()
		t.Fatal("expected Start to refuse a remote bind without a token")
	}
	if !strings.Contains(err.Error(), "token") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestStopIsIdempotentError(t *testing.T) {
	m, _ := startSSE(t, "secret-token")
	if err := m.Stop(); err != nil {
		t.Fatalf("first Stop: %v", err)
	}
	if err := m.Stop(); err == nil {
		t.Error("second Stop should report the server is not running")
	}
	_ = context.Background()
}
