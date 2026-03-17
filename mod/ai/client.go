package ai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"red-cloud/mod"
	"red-cloud/mod/gologger"
)

// Client represents an AI API client
type Client struct {
	Provider string
	APIKey   string
	BaseURL  string
	Model    string
	client   *http.Client
}

// NewClient creates a new AI client
func NewClient(provider, apiKey, baseURL, model string) *Client {
	proxyURL := mod.GetProxyURL()

	var transport *http.Transport
	if proxyURL != "" {
		if proxy, err := url.Parse(proxyURL); err == nil {
			transport = &http.Transport{
				Proxy: http.ProxyURL(proxy),
				DialContext: (&net.Dialer{
					Timeout:   30 * time.Second,
					KeepAlive: 30 * time.Second,
				}).DialContext,
				TLSHandshakeTimeout:   10 * time.Second,
				ResponseHeaderTimeout: 120 * time.Second,
				ExpectContinueTimeout: 1 * time.Second,
			}
		}
	}

	if transport == nil {
		transport = &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			TLSHandshakeTimeout:   10 * time.Second,
			ResponseHeaderTimeout: 120 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		}
	}

	return &Client{
		Provider: provider,
		APIKey:   apiKey,
		BaseURL:  strings.TrimSuffix(baseURL, "/"),
		Model:    model,
		client: &http.Client{
			Timeout:   0,
			Transport: transport,
		},
	}
}

// Message represents a chat message
type Message struct {
	Role       string          `json:"role"`
	Content    string          `json:"content,omitempty"`
	ToolCallID string          `json:"tool_call_id,omitempty"`
	ToolCalls  []ToolCall      `json:"tool_calls,omitempty"`
	Name       string          `json:"name,omitempty"`
}

// ToolCall represents an OpenAI tool call in a message
type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function ToolFunction `json:"function"`
}

// ToolFunction holds the name and arguments of a tool call
type ToolFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// ToolDefinition describes a tool for the OpenAI API
type ToolDefinition struct {
	Type     string           `json:"type"`
	Function ToolFunctionDef  `json:"function"`
}

// ToolFunctionDef describes the function within a tool definition
type ToolFunctionDef struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// ToolCallResponse is the full (non-streaming) response that may include tool_calls
type ToolCallResponse struct {
	Content   string     `json:"content"`
	ToolCalls []ToolCall `json:"tool_calls"`
	Usage     TokenUsage `json:"usage"`
}

// TokenUsage tracks API token consumption
type TokenUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// StreamCallback is called for each chunk of the stream
type StreamCallback func(chunk string) error

// ToolStreamCallback is called for each content chunk during streaming tool calls
type ToolStreamCallback func(chunk string) error

// ChatWithToolsStream sends a streaming chat request with tool definitions.
// Content chunks are sent to the callback in real-time. Returns the full response (content + tool_calls).
func (c *Client) ChatWithToolsStream(ctx context.Context, messages []Message, tools []ToolDefinition, callback ToolStreamCallback) (*ToolCallResponse, error) {
	reqBody := map[string]interface{}{
		"model":       c.Model,
		"messages":    messages,
		"tools":       tools,
		"tool_choice": "auto",
		"stream":      true,
		"stream_options": map[string]interface{}{
			"include_usage": true,
		},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.BaseURL+"/chat/completions", bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.APIKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var contentBuilder strings.Builder
	toolCallMap := make(map[int]*ToolCall) // index → accumulated tool call
	var usage TokenUsage

	reader := bufio.NewReader(resp.Body)
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("failed to read stream: %w", err)
		}

		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		if !bytes.HasPrefix(line, []byte("data: ")) {
			continue
		}
		data := bytes.TrimPrefix(line, []byte("data: "))
		if bytes.Equal(data, []byte("[DONE]")) {
			break
		}

		var chunk struct {
			Choices []struct {
				Delta struct {
					Content   string `json:"content"`
					ToolCalls []struct {
						Index    int    `json:"index"`
						ID       string `json:"id"`
						Type     string `json:"type"`
						Function struct {
							Name      string `json:"name"`
							Arguments string `json:"arguments"`
						} `json:"function"`
					} `json:"tool_calls"`
				} `json:"delta"`
			} `json:"choices"`
			Usage *struct {
				PromptTokens     int `json:"prompt_tokens"`
				CompletionTokens int `json:"completion_tokens"`
				TotalTokens      int `json:"total_tokens"`
			} `json:"usage"`
		}
		if err := json.Unmarshal(data, &chunk); err != nil {
			continue
		}

		// Capture usage from the final chunk (OpenAI sends it with stream_options)
		if chunk.Usage != nil {
			usage = TokenUsage{
				PromptTokens:     chunk.Usage.PromptTokens,
				CompletionTokens: chunk.Usage.CompletionTokens,
				TotalTokens:      chunk.Usage.TotalTokens,
			}
		}

		if len(chunk.Choices) == 0 {
			continue
		}

		delta := chunk.Choices[0].Delta

		// Stream content chunks to callback
		if delta.Content != "" {
			contentBuilder.WriteString(delta.Content)
			if callback != nil {
				_ = callback(delta.Content)
			}
		}

		// Accumulate tool calls
		for _, dtc := range delta.ToolCalls {
			tc, ok := toolCallMap[dtc.Index]
			if !ok {
				tc = &ToolCall{Type: "function"}
				toolCallMap[dtc.Index] = tc
			}
			if dtc.ID != "" {
				tc.ID = dtc.ID
			}
			if dtc.Function.Name != "" {
				tc.Function.Name = dtc.Function.Name
			}
			tc.Function.Arguments += dtc.Function.Arguments
		}
	}

	// Build sorted tool calls list
	var toolCalls []ToolCall
	for i := 0; i < len(toolCallMap); i++ {
		if tc, ok := toolCallMap[i]; ok {
			// Fix incomplete arguments from streaming (e.g., "{" without closing "}")
			args := strings.TrimSpace(tc.Function.Arguments)
			if args == "" || args == "{" {
				tc.Function.Arguments = "{}"
			}
			toolCalls = append(toolCalls, *tc)
		}
	}

	return &ToolCallResponse{
		Content:   contentBuilder.String(),
		ToolCalls: toolCalls,
		Usage:     usage,
	}, nil
}

// ChatWithTools sends a non-streaming chat request with tool definitions, returns full response
func (c *Client) ChatWithTools(ctx context.Context, messages []Message, tools []ToolDefinition) (*ToolCallResponse, error) {
	reqBody := map[string]interface{}{
		"model":       c.Model,
		"messages":    messages,
		"tools":       tools,
		"tool_choice": "auto",
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.BaseURL+"/chat/completions", bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.APIKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var apiResp struct {
		Choices []struct {
			Message struct {
				Content   string     `json:"content"`
				ToolCalls []ToolCall `json:"tool_calls"`
			} `json:"message"`
		} `json:"choices"`
		Usage *struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	if len(apiResp.Choices) == 0 {
		return nil, fmt.Errorf("empty response from API")
	}

	msg := apiResp.Choices[0].Message
	result := &ToolCallResponse{
		Content:   msg.Content,
		ToolCalls: msg.ToolCalls,
	}
	if apiResp.Usage != nil {
		result.Usage = TokenUsage{
			PromptTokens:     apiResp.Usage.PromptTokens,
			CompletionTokens: apiResp.Usage.CompletionTokens,
			TotalTokens:      apiResp.Usage.TotalTokens,
		}
	}
	return result, nil
}

// ChatStream sends a chat request and streams the response
func (c *Client) ChatStream(ctx context.Context, messages []Message, callback StreamCallback) error {
	if c.Provider == "anthropic" {
		return c.chatStreamAnthropic(ctx, messages, callback)
	}
	return c.chatStreamOpenAI(ctx, messages, callback)
}

// chatStreamOpenAI handles OpenAI-compatible streaming
func (c *Client) chatStreamOpenAI(ctx context.Context, messages []Message, callback StreamCallback) error {
	reqBody := map[string]interface{}{
		"model":    c.Model,
		"messages": messages,
		"stream":   true,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.BaseURL+"/chat/completions", bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.APIKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	reader := bufio.NewReader(resp.Body)
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("failed to read stream: %w", err)
		}

		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}

		if !bytes.HasPrefix(line, []byte("data: ")) {
			continue
		}

		data := bytes.TrimPrefix(line, []byte("data: "))
		if bytes.Equal(data, []byte("[DONE]")) {
			break
		}

		var chunk struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
		}

		if err := json.Unmarshal(data, &chunk); err != nil {
			continue
		}

		if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
			content := chunk.Choices[0].Delta.Content
			gologger.Debug().Msgf("AI stream 收到内容: %s", content)
			if err := callback(content); err != nil {
				return err
			}
		}
	}

	return nil
}

// chatStreamAnthropic handles Anthropic-compatible streaming
func (c *Client) chatStreamAnthropic(ctx context.Context, messages []Message, callback StreamCallback) error {
	// Convert messages format for Anthropic
	var systemMsg string
	var userMessages []Message
	for _, msg := range messages {
		if msg.Role == "system" {
			systemMsg = msg.Content
		} else {
			userMessages = append(userMessages, msg)
		}
	}

	reqBody := map[string]interface{}{
		"model":      c.Model,
		"messages":   userMessages,
		"max_tokens": 8192,
		"stream":     true,
	}
	if systemMsg != "" {
		reqBody["system"] = systemMsg
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.BaseURL+"/v1/messages", bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	reader := bufio.NewReader(resp.Body)
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("failed to read stream: %w", err)
		}

		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}

		if !bytes.HasPrefix(line, []byte("data: ")) {
			continue
		}

		data := bytes.TrimPrefix(line, []byte("data: "))

		var event struct {
			Type  string `json:"type"`
			Delta struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"delta"`
		}

		if err := json.Unmarshal(data, &event); err != nil {
			continue
		}

		if event.Type == "content_block_delta" && event.Delta.Type == "text_delta" {
			if err := callback(event.Delta.Text); err != nil {
				return err
			}
		}
	}

	return nil
}

// EstimateTokens estimates token count for a message list.
// Uses heuristic: Chinese ~1.5 chars/token, ASCII ~4 chars/token, plus per-message overhead.
func EstimateTokens(messages []Message) int {
	total := 0
	for _, m := range messages {
		total += 4 // per-message overhead (role, separators)
		total += estimateContentTokens(m.Content)
		for _, tc := range m.ToolCalls {
			total += 4 // tool_call overhead
			total += estimateContentTokens(tc.Function.Name)
			total += estimateContentTokens(tc.Function.Arguments)
		}
	}
	return total
}

func estimateContentTokens(s string) int {
	if s == "" {
		return 0
	}
	chinese := 0
	ascii := 0
	for _, r := range s {
		if r > 127 {
			chinese++
		} else {
			ascii++
		}
	}
	// Chinese chars ~0.67 tokens each, ASCII ~0.25 tokens each
	tokens := (chinese*2 + 1) / 3
	tokens += (ascii + 3) / 4
	if tokens < 1 {
		tokens = 1
	}
	return tokens
}
