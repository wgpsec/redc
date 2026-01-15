package compose

import (
	"fmt"
	"os"
	"path/filepath"
	"red-cloud/mod"
	"strings"

	"red-cloud/mod/gologger"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
)

// VerifyTemplates 静态校验：检查 Terraform 模版是否声明了所有即将注入的变量
// (此函数主体逻辑不变，仅 scanTfVariables 变了，为了完整性贴出)
func VerifyTemplates(ctx *ComposeContext) error {
	var totalErrors []string

	gologger.Info().Msg("🔍 正在预检模版配置...")

	checkedTemplates := make(map[string]map[string]bool)

	for _, name := range ctx.SortedSvcKeys {
		svc := ctx.RuntimeSvcs[name]
		templatePath := filepath.Join(mod.TemplateDir, svc.Spec.Image)

		// 1. 获取模版中声明的所有变量
		declaredVars, ok := checkedTemplates[templatePath]
		if !ok {
			var err error
			declaredVars, err = scanTfVariables(templatePath)
			if err != nil {
				return fmt.Errorf("解析模版 [%s] 失败: %v", svc.Spec.Image, err)
			}
			checkedTemplates[templatePath] = declaredVars
		}

		// 2. 计算 redc 打算注入的变量
		injectedVars := make(map[string]string)

		// A. 自动注入: provider_alias
		if pStr, ok := svc.Spec.Provider.(string); ok && pStr != "" && pStr != "default" {
			injectedVars["provider_alias"] = "Auto-injected (provider is set)"
		}

		// B. Environment
		for _, envStr := range svc.Spec.Environment {
			parts := strings.SplitN(envStr, "=", 2)
			if len(parts) >= 1 {
				key := strings.TrimSpace(parts[0])
				injectedVars[key] = fmt.Sprintf("YAML environment: %s", key)
			}
		}

		// C. Configs
		for _, cfgStr := range svc.Spec.Configs {
			parts := strings.SplitN(cfgStr, "=", 2)
			if len(parts) >= 1 {
				key := strings.TrimSpace(parts[0])
				injectedVars[key] = fmt.Sprintf("YAML configs: %s", key)
			}
		}

		// 3. 执行比对
		var missingVars []string
		for key, reason := range injectedVars {
			if !declaredVars[key] {
				missingVars = append(missingVars, fmt.Sprintf("  - %s (Source: %s)", key, reason))
			}
		}

		if len(missingVars) > 0 {
			msg := fmt.Sprintf("❌ 服务 [%s] (模版: %s) 缺失变量声明:\n%s",
				svc.Name, svc.Spec.Image, strings.Join(missingVars, "\n"))
			totalErrors = append(totalErrors, msg)
		}
	}

	if len(totalErrors) > 0 {
		return fmt.Errorf("模版校验失败，请在对应的 variables.tf 中添加缺失的变量:\n\n%s", strings.Join(totalErrors, "\n\n"))
	}

	gologger.Info().Msg("✅ 模版预检通过")
	return nil
}

// scanTfVariables 使用 hashicorp/hcl/v2 解析 TF 文件
func scanTfVariables(dir string) (map[string]bool, error) {
	vars := make(map[string]bool)
	parser := hclparse.NewParser()

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("模版目录不存在: %s", dir)
		}
		return nil, err
	}

	for _, entry := range entries {
		// 只处理 .tf 文件
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".tf") {
			continue
		}

		path := filepath.Join(dir, entry.Name())

		// 1. 解析文件到内存
		file, diags := parser.ParseHCLFile(path)
		if diags.HasErrors() {
			// 如果有严重语法错误，这里直接返回，有助于在 apply 前发现问题
			return nil, fmt.Errorf("文件 %s 存在语法错误: %s", entry.Name(), diags.Error())
		}

		// 2. 定义我们只关心的 Schema (只提取 variable 块)
		// variable "name" { ... }
		rootSchema := &hcl.BodySchema{
			Blocks: []hcl.BlockHeaderSchema{
				{
					Type:       "variable",
					LabelNames: []string{"name"}, // variable 后面跟的那个标签就是变量名
				},
			},
		}

		// 3. 部分解码 (PartialContent)
		// 这一步会忽略 resource, data, output 等块，只返回 variable
		content, _, diags := file.Body.PartialContent(rootSchema)
		if diags.HasErrors() {
			return nil, fmt.Errorf("解析 %s 结构失败: %s", entry.Name(), diags.Error())
		}

		// 4. 提取变量名
		for _, block := range content.Blocks {
			if block.Type == "variable" && len(block.Labels) > 0 {
				varName := block.Labels[0]
				vars[varName] = true
			}
		}
	}

	return vars, nil
}
