package main

import (
	"encoding/json"
	"strings"
	"testing"
)

// GetMCPStatus is role "viewer" but its payload carries the SSE bearer token.
// Before the scrub, any viewer could call it and read the token.
func TestScrubCredentialsFromHTTP_MCPStatus(t *testing.T) {
	status := MCPStatus{
		Running:         true,
		Mode:            "sse",
		Address:         "127.0.0.1:8899",
		ProtocolVersion: "2024-11-05",
		Token:           "5e1f0a9c4b7d2e6f8a3c1b0d9e4f2a7c",
	}

	got := scrubCredentialsFromHTTP(status)
	m, ok := got.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map, got %T", got)
	}
	if m["token"] != "" {
		t.Errorf("token = %q, want scrubbed", m["token"])
	}
	// 非敏感字段必须保留，否则前端状态显示会坏掉
	if m["address"] != "127.0.0.1:8899" {
		t.Errorf("address = %v, want preserved", m["address"])
	}
	if m["running"] != true {
		t.Errorf("running = %v, want true", m["running"])
	}
	if m["mode"] != "sse" {
		t.Errorf("mode = %v, want sse", m["mode"])
	}
}

func TestScrubCredentialsFromHTTP_Nested(t *testing.T) {
	v := map[string]interface{}{
		"providers": []interface{}{
			map[string]interface{}{
				"name":      "AWS",
				"accessKey": "AKIAIOSFODNN7EXAMPLE",
				"secretKey": "wJalrXUtnFEMI/K7MDENG",
				"region":    "us-east-1",
			},
		},
		"nested": map[string]interface{}{
			"deeper": map[string]interface{}{
				"apiKey":   "sk-ant-xxx",
				"password": "hunter2",
			},
		},
	}

	got := scrubCredentialsFromHTTP(v)
	data, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	s := string(data)
	for _, secret := range []string{"AKIAIOSFODNN7EXAMPLE", "wJalrXUtnFEMI", "sk-ant-xxx", "hunter2"} {
		if strings.Contains(s, secret) {
			t.Errorf("secret %q survived scrubbing: %s", secret, s)
		}
	}
	for _, keep := range []string{"AWS", "us-east-1", "deeper"} {
		if !strings.Contains(s, keep) {
			t.Errorf("non-secret %q was dropped: %s", keep, s)
		}
	}
	// 凭证字段非空（前端用非空判断"已配置"），所以是掩码而不是清空
	if !strings.Contains(s, `"accessKey":"***"`) {
		t.Errorf("accessKey should be masked, not blanked: %s", s)
	}
}

func TestScrubCredentialsFromHTTP_EmptyValuesStayEmpty(t *testing.T) {
	v := map[string]interface{}{"token": "", "address": "x"}
	got := scrubCredentialsFromHTTP(v).(map[string]interface{})
	if got["token"] != "" {
		t.Errorf("token = %v, want empty", got["token"])
	}
	if got["address"] != "x" {
		t.Errorf("address = %v, want x", got["address"])
	}
}

func TestScrubCredentialsFromHTTP_NonMapUntouched(t *testing.T) {
	if got := scrubCredentialsFromHTTP("plain"); got != "plain" {
		t.Errorf("got %v, want plain string unchanged", got)
	}
	if got := scrubCredentialsFromHTTP(42); got != float64(42) {
		t.Errorf("got %v (%T), want 42", got, got)
	}
}
