package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"red-cloud/mod/gologger"
)

// CompactOptions controls LLM-based context compaction behavior.
type CompactOptions struct {
	// KeepRecentRounds is the number of recent interaction rounds to keep at full length.
	KeepRecentRounds int
	// ContextBudget is the target token count after compaction.
	ContextBudget int
	// MaxSummaryTokens is the max tokens for the summary of compacted history.
	MaxSummaryTokens int
}

// DefaultCompactOptions returns sensible defaults for context compaction.
func DefaultCompactOptions() CompactOptions {
	return CompactOptions{
		KeepRecentRounds: 4,
		ContextBudget:    108000,
		MaxSummaryTokens: 2000,
	}
}

// CompactWithLLM uses an AI call to summarize old conversation rounds, replacing them with
// a concise summary message. The recent rounds (tail) are kept intact.
//
// Returns the compacted message list. If the LLM call fails, falls back to string truncation.
func CompactWithLLM(ctx context.Context, client *Client, messages []Message, opts CompactOptions) []Message {
	if len(messages) <= 4 {
		return messages
	}

	estimated := EstimateTokens(messages)
	if estimated <= opts.ContextBudget {
		return messages
	}

	// Split messages into: system + head (to compact) + tail (to keep)
	systemMsg, head, tail := splitMessages(messages, opts.KeepRecentRounds)

	if len(head) == 0 {
		// Nothing to compact; only recent messages
		return messages
	}

	// Build a summarization prompt
	summary, err := summarizeWithLLM(ctx, client, head)
	if err != nil {
		gologger.Warning().Msgf("ai: LLM compaction failed, falling back to string truncation: %v", err)
		return fallbackCompact(messages, opts)
	}

	// Reconstruct: system + summary message + tail
	result := make([]Message, 0, 2+len(tail))
	result = append(result, systemMsg)
	result = append(result, Message{
		Role:    "user",
		Content: fmt.Sprintf("[Conversation History Summary]\n%s\n\n---\n(Earlier messages compacted. Continuing from recent context.)", summary),
	})
	result = append(result, tail...)

	newEstimate := EstimateTokens(result)
	gologger.Info().Msgf("ai: LLM compaction: %d → %d tokens (compacted %d messages → summary)", estimated, newEstimate, len(head))

	return result
}

// splitMessages separates messages into system, compactable head, and recent tail.
// Tail includes the last keepRounds worth of assistant/tool message pairs + user messages.
func splitMessages(messages []Message, keepRounds int) (system Message, head []Message, tail []Message) {
	if len(messages) == 0 {
		return Message{Role: "system", Content: ""}, nil, nil
	}

	system = messages[0]
	rest := messages[1:]

	if keepRounds <= 0 {
		keepRounds = 4
	}

	// Walk backward counting "rounds" (a round = assistant message, optionally followed by tool messages)
	roundCount := 0
	tailStart := len(rest)
	for i := len(rest) - 1; i >= 0; i-- {
		if rest[i].Role == "assistant" {
			roundCount++
			if roundCount >= keepRounds {
				tailStart = i
				break
			}
		}
	}

	// Ensure we don't orphan tool messages: if tailStart-1 is a user message, include it
	if tailStart > 0 && rest[tailStart-1].Role == "user" {
		tailStart--
	}

	head = rest[:tailStart]
	tail = rest[tailStart:]
	return
}

// summarizeWithLLM calls the AI to generate a concise summary of the conversation history.
func summarizeWithLLM(ctx context.Context, client *Client, messages []Message) (string, error) {
	// Build a transcript of the messages to summarize
	var transcript strings.Builder
	for _, m := range messages {
		switch m.Role {
		case "user":
			transcript.WriteString(fmt.Sprintf("[User]: %s\n\n", truncateForSummary(m.Content, 500)))
		case "assistant":
			content := truncateForSummary(m.Content, 300)
			if len(m.ToolCalls) > 0 {
				names := make([]string, len(m.ToolCalls))
				for i, tc := range m.ToolCalls {
					names[i] = tc.Function.Name
				}
				content += fmt.Sprintf(" [Tools called: %s]", strings.Join(names, ", "))
			}
			transcript.WriteString(fmt.Sprintf("[Assistant]: %s\n\n", content))
		case "tool":
			transcript.WriteString(fmt.Sprintf("[Tool %s]: %s\n\n", m.Name, truncateForSummary(m.Content, 200)))
		}
	}

	summaryPrompt := `Summarize the following multi-turn AI agent conversation into a concise summary (under 500 words).
Focus on:
1. What the user requested
2. Key actions taken (tools called, resources created/modified)
3. Important results and findings (IPs, errors, status)
4. Any unresolved issues or pending tasks

Conversation transcript:
` + transcript.String() + `

Provide ONLY the summary, no preamble.`

	summaryMessages := []Message{
		{Role: "system", Content: "You are a conversation summarizer. Output a concise factual summary."},
		{Role: "user", Content: summaryPrompt},
	}

	// Use a short timeout for the summary call
	sumCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	var result strings.Builder
	err := client.ChatStream(sumCtx, summaryMessages, func(chunk string) error {
		result.WriteString(chunk)
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("summary LLM call failed: %w", err)
	}

	summary := strings.TrimSpace(result.String())
	if summary == "" {
		return "", fmt.Errorf("empty summary from LLM")
	}

	return summary, nil
}

// fallbackCompact performs string-truncation compaction when LLM compaction fails.
func fallbackCompact(messages []Message, opts CompactOptions) []Message {
	if len(messages) <= 4 {
		return messages
	}

	const keepRecentFull = 6
	recentCount := 0
	boundary := len(messages)
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "assistant" || messages[i].Role == "tool" {
			recentCount++
			if recentCount >= keepRecentFull {
				boundary = i
				break
			}
		}
	}

	result := make([]Message, 0, len(messages))
	for i, m := range messages {
		if i == 0 && m.Role == "system" {
			result = append(result, m)
			continue
		}
		if m.Role == "user" {
			result = append(result, m)
			continue
		}
		if i >= boundary {
			result = append(result, m)
			continue
		}
		compressed := m
		switch m.Role {
		case "assistant":
			if len(m.Content) > 200 {
				compressed.Content = m.Content[:200] + "... (compacted)"
			}
			if len(m.ToolCalls) > 0 {
				names := make([]string, len(m.ToolCalls))
				for j, tc := range m.ToolCalls {
					names[j] = tc.Function.Name
				}
				compressed.Content += fmt.Sprintf(" [called: %s]", strings.Join(names, ", "))
			}
		case "tool":
			if len(m.Content) > 100 {
				compressed.Content = m.Content[:100] + "... (compacted)"
			}
		}
		result = append(result, compressed)
	}
	return result
}

func truncateForSummary(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

// SaveTranscript saves the full message history as JSON for audit trail before compaction.
func SaveTranscript(messages []Message, filePath string) error {
	data, err := json.MarshalIndent(messages, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal transcript: %w", err)
	}
	return os.WriteFile(filePath, data, 0644)
}
