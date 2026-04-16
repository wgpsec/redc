# RedC AI Enhancements

This document describes the 5 major AI backend enhancements added to RedC GUI.

---

## 1. Provider Failover

**File:** `mod/ai/provider.go`

### Overview

ProviderManager manages multiple AI providers with automatic failover and retry logic. When the primary provider fails (rate limit, auth error, timeout), the system automatically switches to the next configured fallback provider.

### Architecture

```
Primary Provider → (error) → Classify Error → Transient? → Retry with Backoff (3s, 6s, 10s)
                                             → Permanent? → Failover to Next Provider
                                             → All exhausted? → Return Error
```

### Error Classification

- **Permanent errors** (immediate failover): `401`, `403`, `insufficient balance`, `invalid api key`, `billing`, `quota exceeded`
- **Transient errors** (retry first): `429`, `rate limit`, `overloaded`, `timeout`, `connection refused`, `500-504`, `broken pipe`

### Configuration

In the GUI's AI Settings, users can add fallback providers:

```json
{
  "aiConfig": {
    "provider": "openai",
    "apiKey": "sk-primary...",
    "baseUrl": "https://api.openai.com/v1",
    "model": "gpt-4o",
    "fallbackProviders": [
      {
        "name": "backup-deepseek",
        "provider": "openai",
        "apiKey": "sk-backup...",
        "baseUrl": "https://api.deepseek.com/v1",
        "model": "deepseek-chat"
      }
    ]
  }
}
```

### Key Types

- `ProviderConfig` — Single provider configuration (name, provider type, API key, base URL, model)
- `ProviderManager` — Manages provider list, current index, failover state
- `ExecuteWithRetry(maxRetries, fn)` — Generic retry+failover wrapper

---

## 2. Skills Knowledge Base

**File:** `mod/ai/skills.go`

### Overview

A knowledge base system that provides IaC best practices and cloud-specific guidance to the AI agent. Skills are automatically recommended based on the user's query context and can be loaded on-demand via MCP tools.

### Built-in Skills (5)

| ID | Name | Tags |
|----|------|------|
| `terraform-best-practices` | Terraform Best Practices | terraform, iac, state, module |
| `aws-security-hardening` | AWS Security Hardening | aws, security, iam, vpc |
| `multi-cloud-deployment` | Multi-Cloud Deployment | multi-cloud, aws, azure, gcp, aliyun |
| `troubleshooting-guide` | Deployment Troubleshooting | troubleshoot, error, debug, terraform |
| `cost-optimization` | Cloud Cost Optimization | cost, optimization, pricing, spot |

### Custom Skills

Users can add custom skills by creating directories under `~/redc/skills/`:

```
~/redc/skills/
  my-custom-skill/
    SKILL.md          # Must start with YAML frontmatter
```

**SKILL.md format:**
```markdown
---
name: My Custom Skill
description: Description of what this skill covers
tags: tag1, tag2, tag3
---

# Skill Content

Full knowledge base content here...
```

### MCP Tools

- `list_skills(keyword?)` — List available skills, optionally filtered by keyword
- `read_skill(id)` — Read the full content of a skill by ID

### Auto-Recommendation

When the agent loop starts, the system analyzes the user's last message, matches it against skill tags/descriptions, and injects up to 3 recommended skills into the system prompt:

```
## Recommended Skills (auto-matched to context)
- `read_skill(id="terraform-best-practices")` — Terraform IaC best practices...
- `read_skill(id="aws-security-hardening")` — AWS security best practices...
```

---

## 3. Safety Hooks

**File:** `mod/ai/hooks.go`

### Overview

A hook chain system that intercepts tool calls before and after execution, providing safety checks and annotations.

### Hook Types

- **PreToolHook** — Runs before tool execution. Can `Continue`, `Block`, or `Confirm` (ask user).
- **PostToolHook** — Runs after execution. Can annotate the result with additional context.

### Built-in Hooks

#### DangerousOpHook (Pre-hook)

Triggers user confirmation for destructive operations:

| Tool | Risk |
|------|------|
| `kill_case` | Force-kill (resources may not be cleaned) |
| `delete_template` | Permanent deletion |
| `compose_down` | Destroy all compose services |
| `stop_case` | Terraform destroy |

When triggered, the agent uses `ask_user` to get explicit confirmation before proceeding.

#### CostAnnotationPostHook (Post-hook)

Annotates results of cost-incurring tools (`start_case`, `plan_case`, `compose_up`) with a reminder to check balances.

#### Safety Refusal Detection

`DetectSafetyRefusal()` checks if AI output contains safety refusal patterns (e.g., "I cannot", "As an AI"). This can be used for detection and retry.

### Integration

Hooks are initialized in `runAgentLoop()` and applied during the serial tool execution path. The parallel execution path bypasses hooks since it only runs read-only tools.

---

## 4. LLM Context Compaction

**File:** `mod/ai/compact.go`

### Overview

Replaces the previous string-truncation approach (`content[:200]+"..."`) with LLM-powered conversation summarization. When the context window budget is exceeded, old conversation rounds are summarized by an AI call into a concise history block.

### How It Works

1. **Detect**: `EstimateTokens(messages) > contextBudget` triggers compaction
2. **Split**: Messages are divided into system prompt, compactable head, and recent tail
3. **Summarize**: The head is sent to the LLM with a summarization prompt
4. **Reconstruct**: `[system] + [summary as user message] + [recent tail]`
5. **Fallback**: If the LLM call fails, falls back to string truncation

### Safe Tail Splitting

The tail-splitting logic ensures:
- System prompt is always preserved
- The last N rounds (default 4) are kept at full fidelity
- Tool messages are never orphaned from their parent assistant message
- User messages adjacent to the tail boundary are included

### Configuration

```go
CompactOptions{
    KeepRecentRounds: 4,      // Keep last 4 assistant rounds at full length
    ContextBudget:    108000, // Target token count after compaction
    MaxSummaryTokens: 2000,   // Max tokens for the summary
}
```

### Transcript Audit Trail

`SaveTranscript(messages, filePath)` can be called before compaction to preserve the full conversation for audit purposes.

---

## 5. Multi-round Orchestrator

**File:** `app_ai_orchestrator.go`

### Overview

A multi-round orchestration engine that breaks complex deployment objectives into iterative rounds, each evaluated by a Judge agent. Inspired by tchkiller's orchestrator+judge pattern, adapted for IaC deployment.

### Architecture

```
Objective → [Round 1] → Judge → Feedback → [Round 2] → Judge → ... → Complete
                ↓                                ↓
          Agent Loop                        Agent Loop
          (tool calls)                      (tool calls + prior knowledge)
```

### Phases per Round

1. **Planning** — Build system prompt with objective + cross-round knowledge
2. **Executing** — Run agent loop with tool calls (reuses existing MCP tools)
3. **Judging** — Evaluate round output against objective with structured criteria

### Judge Evaluation

The Judge produces a structured JSON evaluation:

```json
{
  "complete": false,
  "confidence": 0.6,
  "feedback": "Infrastructure created but software not yet deployed",
  "missing_areas": ["software installation", "service verification"],
  "next_steps": ["Install nginx on the new instance", "Verify service is accessible"],
  "evidence_summary": "EC2 instance created at 1.2.3.4, SSH accessible"
}
```

Completion criteria: `complete == true && confidence >= 0.8`

### Cross-Round Knowledge Injection

Each round receives:
- **Evidence log**: Cumulative findings from all prior rounds
- **Failure history**: What went wrong previously (to avoid repeating mistakes)
- **Judge feedback**: Specific improvement suggestions from the last evaluation

### Wails Binding

```go
// Frontend can call:
app.OrchestratorStream(conversationId, OrchestratorConfig{
    MaxRounds:   5,
    Objective:   "Deploy nginx on AWS with HTTPS",
    AutoApprove: false,
}, messages)
```

### Frontend Events

| Event | Data |
|-------|------|
| `ai-orchestrator-status` | `{round, totalRounds, phase, detail}` |
| `ai-orchestrator-judge` | `{round, evaluation}` |
| `ai-chat-chunk` | Standard streaming chunks |
| `ai-chat-complete` | Standard completion with usage |

---

## File Summary

| File | Lines | Purpose |
|------|-------|---------|
| `mod/ai/provider.go` | ~225 | Provider failover + retry |
| `mod/ai/skills.go` | ~280 | Skills knowledge base engine |
| `mod/ai/hooks.go` | ~165 | Safety hook chain |
| `mod/ai/compact.go` | ~225 | LLM context compaction |
| `app_ai_orchestrator.go` | ~480 | Multi-round orchestrator |

## Modified Files

| File | Changes |
|------|---------|
| `mod/profile.go` | Added `FallbackProviders` to `AIConfig`, `FallbackProvider` struct |
| `mod/mcp/mcp.go` | Added `list_skills`, `read_skill` tools + implementations |
| `app_ai_chat.go` | Integrated ProviderManager, skills suggestions, hooks, LLM compaction |
