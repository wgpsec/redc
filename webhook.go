package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	redc "red-cloud/mod"
	"red-cloud/mod/gologger"
	"strings"
	"sync"
	"time"
)

// WebhookConfig holds the configuration loaded from GUISettings
type WebhookConfig struct {
	Enabled         bool   `json:"enabled"`
	Slack           string `json:"slack"`
	Dingtalk        string `json:"dingtalk"`
	DingtalkSecret  string `json:"dingtalkSecret"`
	Feishu          string `json:"feishu"`
	FeishuSecret    string `json:"feishuSecret"`
	Discord         string `json:"discord"`
	Wecom           string `json:"wecom"`
}

type WebhookManager struct {
	mu     sync.RWMutex
	config WebhookConfig
	client *http.Client
}

func NewWebhookManager() *WebhookManager {
	return &WebhookManager{
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

func (wm *WebhookManager) LoadFromSettings() {
	settings, err := redc.LoadGUISettings()
	if err != nil || settings == nil {
		return
	}
	wm.mu.Lock()
	defer wm.mu.Unlock()
	wm.config = WebhookConfig{
		Enabled:        settings.WebhookEnabled,
		Slack:          settings.WebhookSlack,
		Dingtalk:       settings.WebhookDingtalk,
		DingtalkSecret: settings.WebhookDingtalkSecret,
		Feishu:         settings.WebhookFeishu,
		FeishuSecret:   settings.WebhookFeishuSecret,
		Discord:        settings.WebhookDiscord,
		Wecom:          settings.WebhookWecom,
	}
}

func (wm *WebhookManager) GetConfig() WebhookConfig {
	wm.mu.RLock()
	defer wm.mu.RUnlock()
	return wm.config
}

func (wm *WebhookManager) SetConfig(cfg WebhookConfig) {
	wm.mu.Lock()
	defer wm.mu.Unlock()
	wm.config = cfg
}

// Send pushes a notification to all configured webhook endpoints.
func (wm *WebhookManager) Send(title, message, color string) {
	cfg := wm.GetConfig()
	if !cfg.Enabled {
		return
	}
	go func() {
		if cfg.Slack != "" {
			if err := wm.sendSlack(cfg.Slack, title, message, color); err != nil {
				gologger.Error().Msgf("Webhook Slack failed: %v", err)
			}
		}
		if cfg.Dingtalk != "" {
			if err := wm.sendDingtalk(cfg.Dingtalk, cfg.DingtalkSecret, title, message); err != nil {
				gologger.Error().Msgf("Webhook DingTalk failed: %v", err)
			}
		}
		if cfg.Feishu != "" {
			if err := wm.sendFeishu(cfg.Feishu, cfg.FeishuSecret, title, message); err != nil {
				gologger.Error().Msgf("Webhook Feishu failed: %v", err)
			}
		}
		if cfg.Discord != "" {
			if err := wm.sendDiscord(cfg.Discord, title, message, color); err != nil {
				gologger.Error().Msgf("Webhook Discord failed: %v", err)
			}
		}
		if cfg.Wecom != "" {
			if err := wm.sendWecom(cfg.Wecom, title, message); err != nil {
				gologger.Error().Msgf("Webhook WeCom failed: %v", err)
			}
		}
	}()
}

// TestWebhook sends a test message to a single platform.
func (wm *WebhookManager) TestWebhook(platform, webhookURL, secret string) error {
	title := "RedC Webhook 测试"
	message := "如果您收到此消息，说明 Webhook 配置成功！\nIf you received this message, your webhook is configured correctly!"
	color := "#36a64f" // green

	switch platform {
	case "slack":
		return wm.sendSlack(webhookURL, title, message, color)
	case "dingtalk":
		return wm.sendDingtalk(webhookURL, secret, title, message)
	case "feishu":
		return wm.sendFeishu(webhookURL, secret, title, message)
	case "discord":
		return wm.sendDiscord(webhookURL, title, message, color)
	case "wecom":
		return wm.sendWecom(webhookURL, title, message)
	default:
		return fmt.Errorf("unsupported platform: %s", platform)
	}
}

// colorToDecimal converts hex color string to decimal int for Discord embeds.
func colorToDecimal(hex string) int {
	hex = strings.TrimPrefix(hex, "#")
	var c int
	fmt.Sscanf(hex, "%x", &c)
	return c
}

// --- Slack ---
func (wm *WebhookManager) sendSlack(webhookURL, title, message, color string) error {
	payload := map[string]interface{}{
		"attachments": []map[string]interface{}{
			{
				"color":  color,
				"title":  title,
				"text":   message,
				"footer": "RedC",
				"ts":     time.Now().Unix(),
			},
		},
	}
	return wm.postJSON(webhookURL, payload)
}

// --- DingTalk ---
func (wm *WebhookManager) sendDingtalk(webhookURL, secret, title, message string) error {
	if secret != "" {
		webhookURL = wm.dingtalkSign(webhookURL, secret)
	}
	payload := map[string]interface{}{
		"msgtype": "actionCard",
		"actionCard": map[string]interface{}{
			"title":          title,
			"text":           fmt.Sprintf("### %s\n\n%s\n\n---\n*RedC* | %s", title, message, time.Now().Format("2006-01-02 15:04:05")),
			"singleTitle":    "查看详情",
			"singleURL":      "https://github.com/wgpsec/redc",
		},
	}
	return wm.postJSON(webhookURL, payload)
}

func (wm *WebhookManager) dingtalkSign(webhookURL, secret string) string {
	timestamp := fmt.Sprintf("%d", time.Now().UnixMilli())
	stringToSign := timestamp + "\n" + secret
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(stringToSign))
	sign := url.QueryEscape(base64.StdEncoding.EncodeToString(h.Sum(nil)))
	sep := "&"
	if !strings.Contains(webhookURL, "?") {
		sep = "?"
	}
	return fmt.Sprintf("%s%stimestamp=%s&sign=%s", webhookURL, sep, timestamp, sign)
}

// --- Feishu ---
func (wm *WebhookManager) sendFeishu(webhookURL, secret, title, message string) error {
	payload := map[string]interface{}{
		"msg_type": "interactive",
		"card": map[string]interface{}{
			"header": map[string]interface{}{
				"title": map[string]string{
					"tag":     "plain_text",
					"content": title,
				},
				"template": "blue",
			},
			"elements": []map[string]interface{}{
				{
					"tag": "div",
					"text": map[string]string{
						"tag":     "lark_md",
						"content": message,
					},
				},
				{
					"tag": "note",
					"elements": []map[string]string{
						{"tag": "plain_text", "content": fmt.Sprintf("RedC | %s", time.Now().Format("2006-01-02 15:04:05"))},
					},
				},
			},
		},
	}

	if secret != "" {
		timestamp := fmt.Sprintf("%d", time.Now().Unix())
		stringToSign := timestamp + "\n" + secret
		h := hmac.New(sha256.New, []byte(stringToSign))
		sign := base64.StdEncoding.EncodeToString(h.Sum(nil))
		payload["timestamp"] = timestamp
		payload["sign"] = sign
	}

	return wm.postJSON(webhookURL, payload)
}

// --- Discord ---
func (wm *WebhookManager) sendDiscord(webhookURL, title, message, color string) error {
	payload := map[string]interface{}{
		"embeds": []map[string]interface{}{
			{
				"title":       title,
				"description": message,
				"color":       colorToDecimal(color),
				"footer": map[string]string{
					"text": "RedC",
				},
				"timestamp": time.Now().Format(time.RFC3339),
			},
		},
	}
	return wm.postJSON(webhookURL, payload)
}

// --- WeCom (企业微信) ---
func (wm *WebhookManager) sendWecom(webhookURL, title, message string) error {
	payload := map[string]interface{}{
		"msgtype": "markdown",
		"markdown": map[string]string{
			"content": fmt.Sprintf("## %s\n\n%s\n\n> RedC | %s", title, message, time.Now().Format("2006-01-02 15:04:05")),
		},
	}
	return wm.postJSON(webhookURL, payload)
}

// postJSON sends a JSON payload via HTTP POST.
func (wm *WebhookManager) postJSON(url string, payload interface{}) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	resp, err := wm.client.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("post: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var buf bytes.Buffer
		buf.ReadFrom(resp.Body)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, buf.String())
	}
	return nil
}
