package ai

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
)

// Skill represents a knowledge base document for IaC best practices.
type Skill struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
	Content     string   `json:"content,omitempty"`
}

// SkillIndex is a lightweight entry used for search without loading full content.
type SkillIndex struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
}

// SkillsEngine manages loading, indexing, and searching skills from a directory.
type SkillsEngine struct {
	mu       sync.RWMutex
	dir      string
	index    []SkillIndex
	loaded   bool
	builtins []Skill
}

// NewSkillsEngine creates a new engine. dir is the path to the skills directory.
// If dir is empty or doesn't exist, only built-in skills are available.
func NewSkillsEngine(dir string) *SkillsEngine {
	e := &SkillsEngine{
		dir:      dir,
		builtins: builtinSkills(),
	}
	return e
}

// ensureLoaded lazily builds the index on first access.
func (e *SkillsEngine) ensureLoaded() {
	e.mu.RLock()
	if e.loaded {
		e.mu.RUnlock()
		return
	}
	e.mu.RUnlock()

	e.mu.Lock()
	defer e.mu.Unlock()
	if e.loaded {
		return
	}

	e.index = make([]SkillIndex, 0)

	// Add built-in skills
	for _, s := range e.builtins {
		e.index = append(e.index, SkillIndex{
			ID:          s.ID,
			Name:        s.Name,
			Description: s.Description,
			Tags:        s.Tags,
		})
	}

	// Scan directory for custom skills
	if e.dir != "" {
		entries, err := os.ReadDir(e.dir)
		if err == nil {
			for _, entry := range entries {
				if !entry.IsDir() {
					continue
				}
				skillMD := filepath.Join(e.dir, entry.Name(), "SKILL.md")
				if _, err := os.Stat(skillMD); err != nil {
					continue
				}
				data, err := os.ReadFile(skillMD)
				if err != nil {
					continue
				}
				si := parseSkillFrontmatter(entry.Name(), string(data))
				e.index = append(e.index, si)
			}
		}
	}

	e.loaded = true
}

// List returns all available skill index entries. Optionally filter by keyword.
func (e *SkillsEngine) List(keyword string) []SkillIndex {
	e.ensureLoaded()
	e.mu.RLock()
	defer e.mu.RUnlock()

	if keyword == "" {
		result := make([]SkillIndex, len(e.index))
		copy(result, e.index)
		return result
	}

	kw := strings.ToLower(keyword)
	var matched []SkillIndex
	for _, si := range e.index {
		if strings.Contains(strings.ToLower(si.Name), kw) ||
			strings.Contains(strings.ToLower(si.Description), kw) ||
			strings.Contains(strings.ToLower(si.ID), kw) {
			matched = append(matched, si)
			continue
		}
		for _, tag := range si.Tags {
			if strings.Contains(strings.ToLower(tag), kw) {
				matched = append(matched, si)
				break
			}
		}
	}
	return matched
}

// Read returns the full content of a skill by ID.
func (e *SkillsEngine) Read(id string) (*Skill, error) {
	e.ensureLoaded()

	// Check built-ins first
	for _, s := range e.builtins {
		if s.ID == id {
			return &s, nil
		}
	}

	// Check custom directory
	if e.dir != "" {
		skillMD := filepath.Join(e.dir, id, "SKILL.md")
		data, err := os.ReadFile(skillMD)
		if err == nil {
			si := parseSkillFrontmatter(id, string(data))
			return &Skill{
				ID:          si.ID,
				Name:        si.Name,
				Description: si.Description,
				Tags:        si.Tags,
				Content:     string(data),
			}, nil
		}
	}

	return nil, fmt.Errorf("skill %q not found", id)
}

// Suggest returns recommended skill IDs based on context (target, tool usage, errors).
func (e *SkillsEngine) Suggest(context string, maxResults int) []SkillIndex {
	e.ensureLoaded()
	e.mu.RLock()
	defer e.mu.RUnlock()

	if maxResults <= 0 {
		maxResults = 5
	}

	ctxLower := strings.ToLower(context)
	ctxTokens := extractTokens(ctxLower)

	type scored struct {
		si    SkillIndex
		score int
	}
	var results []scored

	for _, si := range e.index {
		score := 0
		for _, tag := range si.Tags {
			tagLower := strings.ToLower(tag)
			if _, ok := ctxTokens[tagLower]; ok {
				score += 3
			} else if strings.Contains(ctxLower, tagLower) && len(tagLower) >= 3 {
				score += 1
			}
		}
		descLower := strings.ToLower(si.Description)
		for tok := range ctxTokens {
			if len(tok) >= 3 && strings.Contains(descLower, tok) {
				score += 1
			}
		}
		if score > 0 {
			results = append(results, scored{si: si, score: score})
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].score > results[j].score
	})

	if len(results) > maxResults {
		results = results[:maxResults]
	}

	out := make([]SkillIndex, len(results))
	for i, r := range results {
		out[i] = r.si
	}
	return out
}

// FormatSuggestions formats skill suggestions as a prompt injection block.
func FormatSuggestions(suggestions []SkillIndex) string {
	if len(suggestions) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("\n\n## Recommended Skills (auto-matched to context)\n")
	sb.WriteString("Load on demand when encountering related scenarios:\n")
	for _, si := range suggestions {
		desc := si.Description
		if len(desc) > 60 {
			desc = desc[:60] + "..."
		}
		sb.WriteString(fmt.Sprintf("- `read_skill(id=\"%s\")` — %s\n", si.ID, desc))
	}
	sb.WriteString("\nUse `list_skills(keyword=\"...\")` to search for more.\n")
	return sb.String()
}

// --- Internal helpers ---

func parseSkillFrontmatter(id, content string) SkillIndex {
	si := SkillIndex{ID: id, Name: id}

	if !strings.HasPrefix(content, "---") {
		return si
	}
	end := strings.Index(content[3:], "---")
	if end == -1 {
		return si
	}
	front := content[3 : 3+end]

	for _, line := range strings.Split(front, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "name:") {
			si.Name = strings.Trim(strings.TrimPrefix(line, "name:"), " \"'")
		} else if strings.HasPrefix(line, "description:") {
			si.Description = strings.Trim(strings.TrimPrefix(line, "description:"), " \"'")
		} else if strings.HasPrefix(line, "tags:") {
			tagStr := strings.TrimPrefix(line, "tags:")
			tagStr = strings.Trim(tagStr, " \"'")
			for _, t := range regexp.MustCompile(`[,，]`).Split(tagStr, -1) {
				t = strings.TrimSpace(t)
				if t != "" {
					si.Tags = append(si.Tags, t)
				}
			}
		}
	}
	return si
}

func extractTokens(s string) map[string]struct{} {
	re := regexp.MustCompile(`[a-z\p{Han}]{2,}`)
	matches := re.FindAllString(s, -1)
	tokens := make(map[string]struct{}, len(matches))
	for _, m := range matches {
		tokens[m] = struct{}{}
	}
	return tokens
}

// builtinSkills returns the embedded IaC knowledge base skills.
func builtinSkills() []Skill {
	return []Skill{
		{
			ID:          "terraform-best-practices",
			Name:        "Terraform Best Practices",
			Description: "Terraform IaC best practices for cloud deployments: state management, modules, variables, security",
			Tags:        []string{"terraform", "iac", "best-practices", "state", "module"},
			Content: `# Terraform Best Practices

## State Management
- Always use remote state (S3+DynamoDB, OSS, COS)
- Enable state locking to prevent concurrent modifications
- Never commit .tfstate to version control
- Use separate state per environment (dev/staging/prod)

## Module Design
- Keep modules small and focused (single responsibility)
- Use variable validation blocks for input constraints
- Output essential values (IDs, IPs, endpoints)
- Pin module versions in production

## Security
- Never hardcode credentials in .tf files; use environment variables or vault
- Restrict security group ingress to necessary CIDRs
- Enable encryption at rest for storage resources
- Use IAM roles instead of access keys where possible
- Tag all resources for audit and cost tracking

## Variable Best Practices
- Set sensible defaults for optional variables
- Use ` + "`description`" + ` field for every variable
- Mark sensitive variables with ` + "`sensitive = true`" + `
- Use locals for computed values to avoid repetition

## Common Pitfalls
- Forgetting to run ` + "`terraform init`" + ` after adding providers
- Not using ` + "`-target`" + ` carefully (can cause drift)
- Ignoring plan output before apply
- Not handling resource dependencies explicitly when needed
`,
		},
		{
			ID:          "aws-security-hardening",
			Name:        "AWS Security Hardening",
			Description: "AWS security best practices: IAM, VPC, security groups, encryption, logging",
			Tags:        []string{"aws", "security", "iam", "vpc", "hardening"},
			Content: `# AWS Security Hardening for Red Team Infrastructure

## IAM
- Create dedicated IAM users for RedC operations (never use root)
- Apply least-privilege policies; use AWS managed policies as starting points
- Enable MFA on all IAM users
- Rotate access keys regularly (every 90 days)
- Use IAM roles for EC2 instances instead of embedded keys

## VPC & Network
- Deploy instances in a dedicated VPC (not default VPC)
- Use private subnets for internal resources; NAT gateway for outbound
- Security groups: restrict SSH (22) to your IP, not 0.0.0.0/0
- Use VPC Flow Logs for network audit trail

## Instance Security
- Use latest AMIs (Ubuntu 22.04 LTS recommended)
- Enable EBS encryption (default encryption per region)
- Use key pairs for SSH; disable password authentication
- Install security updates on first boot via user_data

## Cleanup
- Set auto-destroy schedules for temporary infrastructure
- Tag resources with "owner", "purpose", "expires" for governance
- Regularly audit running resources to avoid forgotten instances
- Use AWS Config rules to detect non-compliant resources
`,
		},
		{
			ID:          "multi-cloud-deployment",
			Name:        "Multi-Cloud Deployment",
			Description: "Guide for deploying across AWS, Azure, GCP, Alibaba Cloud, Tencent Cloud, Huawei Cloud",
			Tags:        []string{"multi-cloud", "aws", "azure", "gcp", "aliyun", "tencentcloud", "huaweicloud"},
			Content: `# Multi-Cloud Deployment Guide

## Provider Authentication
Each cloud provider requires specific credential setup:
- **AWS**: AWS_ACCESS_KEY_ID + AWS_SECRET_ACCESS_KEY (or ~/.aws/credentials)
- **Azure**: ARM_SUBSCRIPTION_ID + ARM_TENANT_ID + ARM_CLIENT_ID + ARM_CLIENT_SECRET
- **GCP**: GOOGLE_APPLICATION_CREDENTIALS (service account JSON file)
- **Alibaba Cloud**: ALICLOUD_ACCESS_KEY + ALICLOUD_SECRET_KEY
- **Tencent Cloud**: TENCENTCLOUD_SECRET_ID + TENCENTCLOUD_SECRET_KEY
- **Huawei Cloud**: HW_ACCESS_KEY + HW_SECRET_KEY
- **Volcengine**: VOLCENGINE_ACCESS_KEY + VOLCENGINE_SECRET_KEY

## Region Selection Strategy
- For red team ops: choose regions geographically close to target
- For high availability: distribute across 2+ regions
- For cost: us-east-1 (AWS), eastus (Azure), us-central1 (GCP) are typically cheapest
- Chinese clouds: cn-hangzhou (Alibaba), ap-guangzhou (Tencent) for mainland China

## Cross-Cloud Considerations
- Use consistent naming conventions across providers
- Standardize on Ubuntu 22.04 LTS for cross-cloud compatibility
- Use cloud-init/user_data for consistent post-deploy configuration
- Each provider has different instance type naming (t3.micro, B1s, e2-micro, ecs.t6-c1m1.large)

## Cost Control
- Use spot/preemptible instances for non-critical workloads
- Set billing alerts on all accounts
- Use RedC's scheduled task feature for auto-shutdown
- Monitor with ` + "`redc balance`" + ` / ` + "`get_balances`" + ` tool
`,
		},
		{
			ID:          "troubleshooting-guide",
			Name:        "Deployment Troubleshooting",
			Description: "Common Terraform deployment errors and solutions for RedC scenarios",
			Tags:        []string{"troubleshoot", "error", "debug", "terraform", "deploy"},
			Content: `# Deployment Troubleshooting Guide

## Terraform Init Errors
- "Failed to install provider": Check internet connectivity and proxy settings
- "Could not load plugin": Run ` + "`terraform init -upgrade`" + `
- "Backend initialization": Verify remote state bucket exists and credentials are correct

## Authentication Errors
- "NoCredentialProviders" (AWS): Check AWS_ACCESS_KEY_ID is set
- "AuthorizationFailed" (Azure): Verify subscription ID and service principal
- "googleapi: Error 403": Check service account permissions
- "InvalidAccessKeyId": Rotate expired access keys

## Resource Creation Failures
- "InstanceLimitExceeded": Request quota increase or use a different instance type
- "VPCLimitExceeded": Clean up unused VPCs or use a different region
- "InvalidParameterValue": Check instance type availability in the selected region
- "InsufficientInstanceCapacity": Try a different AZ or instance type

## Network Issues
- "timeout awaiting response": Check security groups allow outbound HTTPS (443)
- "SSH connection refused": Verify security group allows inbound SSH (22) from your IP
- "Connection timed out": Check if instance has a public IP and is in a public subnet

## State Issues
- "state lock": Another process is running; wait or force-unlock with ` + "`terraform force-unlock <ID>`" + `
- "Resource already exists": Import the existing resource with ` + "`terraform import`" + `
- "Unsupported attribute": Provider version mismatch; update provider version

## User Data / Provisioning
- Cloud-init failures: Check /var/log/cloud-init-output.log on the instance
- Script timeout: Increase timeout or break into smaller scripts
- Package install failures: Ensure outbound internet access is available
`,
		},
		{
			ID:          "cost-optimization",
			Name:        "Cloud Cost Optimization",
			Description: "Strategies for minimizing cloud costs in red team infrastructure deployments",
			Tags:        []string{"cost", "optimization", "pricing", "budget", "spot"},
			Content: `# Cloud Cost Optimization for Red Team Infrastructure

## Instance Right-Sizing
- Red team tools typically need 2 vCPU + 4GB RAM minimum
- Use t3.small (AWS), B2s (Azure), e2-small (GCP) as baseline
- ARM instances (t4g) are 20% cheaper but some tools require x86
- Monitor CPU/memory usage and downsize if consistently under 30%

## Time-Based Savings
- Destroy environments when not actively testing
- Use RedC's scheduled tasks for auto-start/stop
- Schedule deployments during off-peak hours for spot pricing
- Set maximum runtime limits to prevent forgotten instances

## Spot/Preemptible Instances
- 60-90% savings over on-demand pricing
- Best for: scanning, brute-force, data processing
- Not recommended for: long-running C2 servers, persistent implants
- AWS: use spot with ` + "`instance_interruption_behavior = \"stop\"`" + `

## Storage Optimization
- Use minimum disk size (18-20 GB for most red team scenarios)
- Use gp3 (AWS), Standard SSD (Azure) — cheaper than premium
- Clean up snapshots and unused volumes regularly
- Use ephemeral storage for temporary data

## Network Cost Reduction
- Data transfer costs are often overlooked
- Use same-region for multi-instance deployments
- Minimize cross-region traffic
- Use VPC endpoints for AWS service access (avoids NAT gateway costs)

## Monitoring
- Use ` + "`get_balances`" + ` to check account credits regularly
- Set billing alerts at 50%, 80%, 100% thresholds
- Use ` + "`get_predicted_monthly_cost`" + ` for cost estimation before deploy
`,
		},
	}
}
