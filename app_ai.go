package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"red-cloud/i18n"
	redc "red-cloud/mod"
	"red-cloud/mod/ai"
	"red-cloud/mod/gologger"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// TemplateRecommendation represents a template recommendation result
type TemplateRecommendation struct {
	Template    string   `json:"template"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Match       int      `json:"match"`
	Tags        []string `json:"tags"`
	Provider    string   `json:"provider"`
	Version     string   `json:"version"`
	Installed   bool     `json:"installed"`
}

// RecommendTemplates searches and recommends templates based on user query
func (a *App) RecommendTemplates(query string) ([]TemplateRecommendation, error) {
	if strings.TrimSpace(query) == "" {
		return nil, fmt.Errorf("%s", i18n.T("app_search_keyword_empty"))
	}

	opts := redc.PullOptions{
		RegistryURL: "https://redc.wgpsec.org",
		Timeout:     30 * time.Second,
	}

	results, err := redc.Search(context.Background(), query, opts)
	if err != nil {
		return nil, fmt.Errorf(i18n.Tf("app_search_failed", err))
	}

	localTemplates, _ := redc.ListLocalTemplates()
	installedMap := make(map[string]bool)
	for _, t := range localTemplates {
		installedMap[t.Name] = true
	}

	recommendations := make([]TemplateRecommendation, 0, len(results))
	for _, r := range results {
		maxScore := 1000
		if len(results) > 0 && results[0].Score > 0 {
			maxScore = results[0].Score
		}
		matchPercent := 50
		if r.Score >= maxScore {
			matchPercent = 95
		} else if r.Score > 0 {
			matchPercent = 50 + (r.Score*45)/maxScore
		}
		if matchPercent > 100 {
			matchPercent = 100
		}

		tags := []string{r.Provider}
		if r.Author != "" {
			tags = append(tags, r.Author)
		}

		name := r.Key
		if parts := strings.Split(r.Key, "/"); len(parts) == 2 {
			name = parts[1]
		}

		recommendations = append(recommendations, TemplateRecommendation{
			Template:    r.Key,
			Name:        name,
			Description: r.Description,
			Match:       matchPercent,
			Tags:        tags,
			Provider:    r.Provider,
			Version:     r.Version,
			Installed:   installedMap[r.Key],
		})
	}

	if len(recommendations) > 10 {
		recommendations = recommendations[:10]
	}

	return recommendations, nil
}

// AIRecommendTemplates uses AI to recommend templates based on user query with streaming
func (a *App) AIRecommendTemplates(query string) error {
	if strings.TrimSpace(query) == "" {
		return fmt.Errorf("%s", i18n.T("app_search_keyword_empty"))
	}

	profile, err := redc.GetActiveProfile()
	if err != nil || profile.AIConfig == nil {
		return fmt.Errorf("%s", i18n.T("app_ai_not_configured"))
	}

	uiLang := a.GetLanguage()
	langPrompt := "请用中文回复"
	if uiLang == "en" {
		langPrompt = "Please reply in English"
	}

	aiConfig := profile.AIConfig
	if aiConfig.APIKey == "" || aiConfig.BaseURL == "" || aiConfig.Model == "" {
		return fmt.Errorf("%s", i18n.T("app_ai_config_incomplete"))
	}

	localTemplates, _ := redc.ListLocalTemplates()
	templateList := make([]string, 0, len(localTemplates))
	for _, t := range localTemplates {
		templateList = append(templateList, fmt.Sprintf("- %s: %s", t.Name, t.Description))
	}

	client := ai.NewClient(aiConfig.Provider, aiConfig.APIKey, aiConfig.BaseURL, aiConfig.Model)

	systemPrompt := `你是一个云资源场景推荐助手。用户会描述他们的需求，你需要根据可用的模板列表推荐最合适的场景。

可用的模板列表：
` + strings.Join(templateList, "\n") + `

请根据用户需求，推荐最合适的模板，并说明推荐理由。如果没有完全匹配的模板，可以推荐相近的模板并说明如何调整使用。

` + langPrompt + `，用简洁、友好的语言回复，直接给出推荐结果和理由。`

	messages := []ai.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: query},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	err = client.ChatStream(ctx, messages, func(chunk string) error {
		runtime.EventsEmit(a.ctx, "ai-recommend-chunk", chunk)
		return nil
	})

	if err != nil {
		return fmt.Errorf(i18n.Tf("app_ai_recommend_failed", err))
	}

	runtime.EventsEmit(a.ctx, "ai-recommend-complete", true)
	return nil
}

// AICostOptimization uses AI to analyze running cases and provide cost optimization suggestions
func (a *App) AICostOptimization() error {
	profile, err := redc.GetActiveProfile()
	if err != nil || profile.AIConfig == nil {
		return fmt.Errorf("%s", i18n.T("app_ai_not_configured"))
	}

	aiConfig := profile.AIConfig
	if aiConfig.APIKey == "" || aiConfig.BaseURL == "" || aiConfig.Model == "" {
		return fmt.Errorf("%s", i18n.T("app_ai_config_incomplete"))
	}

	a.mu.Lock()
	project := a.project
	pricingService := a.pricingService
	costCalculator := a.costCalculator
	logMgr := a.logMgr
	a.mu.Unlock()

	if project == nil {
		return fmt.Errorf("%s", i18n.T("app_project_not_loaded"))
	}

	if pricingService == nil || costCalculator == nil {
		return fmt.Errorf("%s", i18n.T("app_cost_estimate_not_init"))
	}

	cases, err := redc.LoadProjectCases(project.ProjectName)
	if err != nil {
		return fmt.Errorf(i18n.Tf("app_case_load_failed", err))
	}

	if logMgr != nil {
		if logger, logErr := logMgr.NewServiceLogger("cost-optimization"); logErr == nil {
			logger.Write([]byte(fmt.Sprintf("[INFO] Starting AI cost optimization analysis\n")))
			logger.Write([]byte(fmt.Sprintf("[INFO] Total cases: %d\n", len(cases))))
			logger.Close()
		}
	}

	var caseInfoList []string
	runningCount := 0

	for _, c := range cases {
		if c.State != redc.StateRunning {
			continue
		}
		runningCount++

		if logMgr != nil {
			if logger, logErr := logMgr.NewServiceLogger("cost-optimization"); logErr == nil {
				logger.Write([]byte(fmt.Sprintf("[INFO] Processing case: %s (path: %s)\n", c.Name, c.Path)))
				logger.Close()
			}
		}

		if c.Path == "" {
			if logMgr != nil {
				if logger, logErr := logMgr.NewServiceLogger("cost-optimization"); logErr == nil {
					logger.Write([]byte(fmt.Sprintf("[WARN] Case %s has empty path, skipping\n", c.Name)))
					logger.Close()
				}
			}
			caseInfo := fmt.Sprintf(`- **%s**
  - 模板: %s
  - 状态: 运行中
  - 说明: 场景路径为空
  - 建议: 请检查场景配置`, c.Name, c.Module)
			caseInfoList = append(caseInfoList, caseInfo)
			continue
		}

		state, err := redc.TfStatus(c.Path)
		if err != nil {
			if logMgr != nil {
				if logger, logErr := logMgr.NewServiceLogger("cost-optimization"); logErr == nil {
					logger.Write([]byte(fmt.Sprintf("[ERROR] Failed to get terraform state for %s: %v\n", c.Name, err)))
					logger.Close()
				}
			}
			caseInfo := fmt.Sprintf(`- **%s**
  - 模板: %s
  - 状态: 运行中
  - 说明: 状态获取失败 (%v)
  - 建议: 请检查 Terraform 是否正确安装，场景是否已完成部署`, c.Name, c.Module, err)
			caseInfoList = append(caseInfoList, caseInfo)
			continue
		}

		if state == nil || state.Values == nil {
			if logMgr != nil {
				if logger, logErr := logMgr.NewServiceLogger("cost-optimization"); logErr == nil {
					logger.Write([]byte(fmt.Sprintf("[WARN] Case %s has nil state or values, skipping\n", c.Name)))
					logger.Close()
				}
			}
			caseInfo := fmt.Sprintf(`- **%s**
  - 模板: %s
  - 状态: 运行中
  - 说明: 状态数据为空
  - 建议: 该场景可能尚未创建资源`, c.Name, c.Module)
			caseInfoList = append(caseInfoList, caseInfo)
			continue
		}

		resources := extractResourcesFromState(state)
		if resources == nil || len(resources.Resources) == 0 {
			if logMgr != nil {
				if logger, logErr := logMgr.NewServiceLogger("cost-optimization"); logErr == nil {
					logger.Write([]byte(fmt.Sprintf("[WARN] Case %s has no resources\n", c.Name)))
					logger.Close()
				}
			}
			caseInfo := fmt.Sprintf(`- **%s**
  - 模板: %s
  - 状态: 运行中
  - 说明: 未找到资源信息
  - 建议: 该场景可能尚未创建资源，或资源已被销毁`, c.Name, c.Module)
			caseInfoList = append(caseInfoList, caseInfo)
			continue
		}

		if logMgr != nil {
			if logger, logErr := logMgr.NewServiceLogger("cost-optimization"); logErr == nil {
				logger.Write([]byte(fmt.Sprintf("[INFO] Case %s has %d resources\n", c.Name, len(resources.Resources))))
				logger.Close()
			}
		}

		estimate, err := costCalculator.CalculateCost(resources, pricingService)
		if err != nil {
			if logMgr != nil {
				if logger, logErr := logMgr.NewServiceLogger("cost-optimization"); logErr == nil {
					logger.Write([]byte(fmt.Sprintf("[ERROR] Failed to calculate cost for %s: %v\n", c.Name, err)))
					logger.Close()
				}
			}
			var resourceList []string
			for _, r := range resources.Resources {
				resourceList = append(resourceList, fmt.Sprintf("  - %s (%s)", r.Name, r.Type))
			}
			caseInfo := fmt.Sprintf(`- **%s**
  - 模板: %s
  - 状态: 运行中
  - 资源数量: %d
  - 资源列表:
%s
  - 说明: 成本计算失败 (%v)
  - 建议: 请检查定价数据是否可用`, c.Name, c.Module, len(resources.Resources), strings.Join(resourceList, "\n"), err)
			caseInfoList = append(caseInfoList, caseInfo)
			continue
		}

		if logMgr != nil {
			if logger, logErr := logMgr.NewServiceLogger("cost-optimization"); logErr == nil {
				logger.Write([]byte(fmt.Sprintf("[INFO] Cost calculated for %s: ¥%.2f/month\n", c.Name, estimate.TotalMonthlyCost)))
				logger.Close()
			}
		}

		var resourceDetails []string
		for _, rb := range estimate.Breakdown {
			if rb.TotalMonthly > 0 {
				resourceDetails = append(resourceDetails, fmt.Sprintf("  - %s (%s): ¥%.2f/月",
					rb.ResourceName, rb.ResourceType, rb.TotalMonthly))
			} else if !rb.Available {
				resourceDetails = append(resourceDetails, fmt.Sprintf("  - %s (%s): 定价不可用",
					rb.ResourceName, rb.ResourceType))
			}
		}

		provider := "未知"
		if len(estimate.Breakdown) > 0 {
			provider = estimate.Breakdown[0].Provider
		}

		caseInfo := fmt.Sprintf(`- **%s**
  - 模板: %s
  - 云服务商: %s
  - 月度成本: ¥%.2f
  - 资源数量: %d
  - 资源详情:
%s`, c.Name, c.Module, provider, estimate.TotalMonthlyCost, len(estimate.Breakdown), strings.Join(resourceDetails, "\n"))

		caseInfoList = append(caseInfoList, caseInfo)
	}

	if runningCount == 0 {
		return fmt.Errorf("%s", i18n.T("app_no_running_case"))
	}

	uiLang := a.GetLanguage()
	langPrompt := "请用中文回复"
	if uiLang == "en" {
		langPrompt = "Please reply in English"
	}

	client := ai.NewClient(aiConfig.Provider, aiConfig.APIKey, aiConfig.BaseURL, aiConfig.Model)

	systemPrompt := `你是一个云成本优化专家。用户会提供当前运行中的云资源场景及其成本信息，你需要分析并提供成本优化建议。

**重要说明**：
- 某些场景可能因为状态文件问题无法获取完整信息
- 对于信息不完整的场景，请基于已知信息提供方向性建议
- 对于有完整成本信息的场景，请提供详细的优化建议

**分析维度**：
1. **实例规格优化**：是否可以降低配置或使用更经济的实例类型
2. **使用模式优化**：是否可以使用竞价实例、预留实例、定时开关机等策略
3. **资源利用率**：识别可能的资源浪费（如过度配置、闲置资源）
4. **存储优化**：存储类型是否合理，是否有优化空间
5. **网络优化**：带宽配置是否合理

**输出格式**：
对每个场景，请提供：
- 当前状态分析
- 具体的优化建议（可操作的）
- 预计可节省的成本（如果有成本数据）
- 优化的优先级（高/中/低）

**特殊情况处理**：
- 如果场景状态文件读取失败，建议检查部署状态
- 如果无法获取成本信息，提供通用的优化方向
- 如果资源信息不完整，基于模板类型给出建议

` + langPrompt + `，用清晰、专业的语言回复，给出实用的建议。`

	casesInfo := strings.Join(caseInfoList, "\n\n")
	userPrompt := fmt.Sprintf(`请分析以下 %d 个运行中的云资源场景，并提供成本优化建议：

%s

请为每个场景提供详细的优化建议。`, runningCount, casesInfo)

	messages := []ai.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	err = client.ChatStream(ctx, messages, func(chunk string) error {
		runtime.EventsEmit(a.ctx, "ai-cost-chunk", chunk)
		return nil
	})

	if err != nil {
		return fmt.Errorf(i18n.Tf("app_ai_cost_analysis_failed", err))
	}

	runtime.EventsEmit(a.ctx, "ai-cost-complete", true)
	return nil
}

// AnalyzeDeploymentError uses AI to analyze deployment errors and provide solutions
func (a *App) AnalyzeDeploymentError(deploymentID, errorMessage, provider, templateName string) error {
	gologger.Info().Msgf("开始 AI 分析部署错误: deploymentID=%s, provider=%s, template=%s", deploymentID, provider, templateName)

	profile, err := redc.GetActiveProfile()
	if err != nil || profile.AIConfig == nil {
		gologger.Error().Msgf("AI 配置获取失败: %v", err)
		return fmt.Errorf("%s", i18n.T("app_ai_not_configured"))
	}

	aiConfig := profile.AIConfig
	if aiConfig.APIKey == "" || aiConfig.BaseURL == "" || aiConfig.Model == "" {
		return fmt.Errorf("%s", i18n.T("app_ai_config_incomplete"))
	}

	uiLang := a.GetLanguage()
	langPrompt := "请用中文回复"
	if uiLang == "en" {
		langPrompt = "Please reply in English"
	}

	client := ai.NewClient(aiConfig.Provider, aiConfig.APIKey, aiConfig.BaseURL, aiConfig.Model)

	systemPrompt := `你是一个云资源部署专家助手。用户会提供一个部署失败的错误信息，你需要分析错误原因并提供解决方案。

请分析以下部署错误：

- 云服务商: ` + provider + `
- 模板名称: ` + templateName + `
- 错误信息:
` + errorMessage + `

请按以下格式回复：
1. 错误原因分析
2. 解决方案建议
3. 如果需要，提供具体的配置修改建议

` + langPrompt + `，用简洁、专业的语言回复，直接给出分析结果和解决方案。`

	messages := []ai.Message{{Role: "system", Content: systemPrompt}}

	gologger.Info().Msgf("AI 分析: 准备调用流式 API，错误信息长度: %d", len(errorMessage))

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	err = client.ChatStream(ctx, messages, func(chunk string) error {
		gologger.Debug().Msgf("AI 分析收到 chunk: %s", chunk)
		runtime.EventsEmit(a.ctx, "ai-deployment-error-chunk", map[string]string{
			"deploymentId": deploymentID,
			"chunk":        chunk,
		})
		return nil
	})

	if err != nil {
		gologger.Error().Msgf("AI 分析失败: %v", err)
		return fmt.Errorf(i18n.Tf("app_ai_analysis_failed", err))
	}

	gologger.Info().Msgf("AI 分析完成")
	runtime.EventsEmit(a.ctx, "ai-deployment-error-complete", map[string]interface{}{
		"deploymentId": deploymentID,
		"success":      true,
	})
	return nil
}

// AnalyzeCaseError uses AI to analyze case (predefined scenario) creation errors and provide solutions
func (a *App) AnalyzeCaseError(caseName, errorMessage, provider, templateName string) error {
	gologger.Info().Msgf("开始 AI 分析场景错误: caseName=%s, provider=%s, template=%s", caseName, provider, templateName)

	profile, err := redc.GetActiveProfile()
	if err != nil || profile.AIConfig == nil {
		gologger.Error().Msgf("AI 配置获取失败: %v", err)
		return fmt.Errorf("%s", i18n.T("app_ai_not_configured"))
	}

	aiConfig := profile.AIConfig
	if aiConfig.APIKey == "" || aiConfig.BaseURL == "" || aiConfig.Model == "" {
		return fmt.Errorf("%s", i18n.T("app_ai_config_incomplete"))
	}

	uiLang := a.GetLanguage()
	langPrompt := "请用中文回复"
	if uiLang == "en" {
		langPrompt = "Please reply in English"
	}

	client := ai.NewClient(aiConfig.Provider, aiConfig.APIKey, aiConfig.BaseURL, aiConfig.Model)

	systemPrompt := `你是一个云资源部署专家助手。用户会提供一个部署失败的错误信息，你需要分析错误原因并提供解决方案。

请分析以下部署错误：

- 云服务商: ` + provider + `
- 模板名称: ` + templateName + `
- 场景名称: ` + caseName + `
- 错误信息:
` + errorMessage + `

请按以下格式回复：
1. 错误原因分析
2. 解决方案建议
3. 如果需要，提供修正后的配置示例

注意：` + langPrompt + `。`

	messages := []ai.Message{{Role: "system", Content: systemPrompt}}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	err = client.ChatStream(ctx, messages, func(chunk string) error {
		runtime.EventsEmit(a.ctx, "ai-case-error-chunk", map[string]interface{}{
			"caseId": caseName,
			"chunk":  chunk,
		})
		return nil
	})

	if err != nil {
		runtime.EventsEmit(a.ctx, "ai-case-error-complete", map[string]interface{}{
			"caseId":  caseName,
			"success": false,
		})
		return fmt.Errorf(i18n.Tf("app_ai_analysis_failed", err))
	}

	runtime.EventsEmit(a.ctx, "ai-case-error-complete", map[string]interface{}{
		"caseId":  caseName,
		"success": true,
	})
	return nil
}
