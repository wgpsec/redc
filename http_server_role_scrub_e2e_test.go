package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	redc "red-cloud/mod"
	"red-cloud/mod/mcp"
	"testing"
)

// newScrubTestServer builds an HTTPServer with one viewer and one admin token,
// and an MCP manager holding a known SSE token, then returns a test server
// alongside those tokens.
func newScrubTestServer(t *testing.T, sseToken string) (*httptest.Server, string, string) {
	t.Helper()

	app := NewApp()
	app.mcpManager = mcp.NewMCPServerManager(nil, app)
	app.mcpManager.SetToken(sseToken)

	const masterToken = "master-token-admin"
	const viewerToken = "viewer-token-abc"

	srv := NewHTTPServer(app, "127.0.0.1", 0, masterToken, []redc.HTTPUser{
		{Username: "alice", Role: "viewer", Token: viewerToken},
	})
	ts := httptest.NewServer(srv.buildAPIMux())
	t.Cleanup(ts.Close)
	return ts, masterToken, viewerToken
}

// call invokes an App method through the HTTP dispatch path.
func call(t *testing.T, ts *httptest.Server, token, method string, args ...interface{}) map[string]interface{} {
	t.Helper()
	if args == nil {
		args = []interface{}{}
	}
	body, err := json.Marshal(map[string]interface{}{"method": method, "args": args})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	req, err := http.NewRequest("POST", ts.URL+"/api/call", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	defer resp.Body.Close()

	var out map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("status %d: %v", resp.StatusCode, out)
	}
	return out
}

func TestGetMCPStatus_TokenHiddenFromViewer(t *testing.T) {
	const sseToken = "c0ffee1234567890abcdef1234567890"
	ts, masterToken, viewerToken := newScrubTestServer(t, sseToken)

	// viewer 可以调用 GetMCPStatus，但必须拿不到 SSE token
	out := call(t, ts, viewerToken, "GetMCPStatus")
	result, ok := out["result"].(map[string]interface{})
	if !ok {
		t.Fatalf("no result: %v", out)
	}
	if got := result["token"]; got != "" {
		t.Errorf("viewer received token %q, want empty", got)
	}
	// 非敏感字段仍在，前端状态显示不受影响
	if result["address"] == nil {
		t.Errorf("address missing from scrubbed payload: %v", result)
	}

	// admin 仍然拿得到，否则 GUI/HTTP 管理员无法配置 MCP 客户端
	out = call(t, ts, masterToken, "GetMCPStatus")
	result, _ = out["result"].(map[string]interface{})
	if got := result["token"]; got != sseToken {
		t.Errorf("admin token = %v, want %q", got, sseToken)
	}
}

func TestGetMCPStatus_ScrubIsNotRoleBypass(t *testing.T) {
	// operator 同样属于"非 admin"，不应看到 token
	const sseToken = "beefbeefbeefbeefbeefbeefbeefbeef"
	app := NewApp()
	app.mcpManager = mcp.NewMCPServerManager(nil, app)
	app.mcpManager.SetToken(sseToken)

	const operatorToken = "operator-token-xyz"
	srv := NewHTTPServer(app, "127.0.0.1", 0, "master", []redc.HTTPUser{
		{Username: "bob", Role: "operator", Token: operatorToken},
	})
	ts := httptest.NewServer(srv.buildAPIMux())
	defer ts.Close()

	out := call(t, ts, operatorToken, "GetMCPStatus")
	result := out["result"].(map[string]interface{})
	if got := result["token"]; got != "" {
		t.Errorf("operator received token %q, want empty", got)
	}
}

func TestRoleSortingSanity(t *testing.T) {
	if !(RoleLevel("admin") > RoleLevel("operator") && RoleLevel("operator") > RoleLevel("viewer")) {
		t.Fatal("role ordering broken")
	}
	if fmt.Sprint(RoleLevel("unknown")) != "0" {
		t.Error("unknown role should be level 0")
	}
}
