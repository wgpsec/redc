package main

import (
	"fmt"
	"github.com/hashicorp/terraform-json"
	"red-cloud/i18n"
	redc "red-cloud/mod"
	"red-cloud/mod/gologger"
	"sort"
	"time"
)

func (a *App) GetProvidersConfig(customPath string) (ProvidersConfigInfo, error) {
	conf, configPath, err := redc.ReadConfig(customPath)
	if err != nil {
		return ProvidersConfigInfo{}, err
	}

	providers := []ProviderCredential{
		{
			Name: "AWS",
			Fields: map[string]string{
				"accessKey": maskValue(conf.Providers.Aws.AccessKey),
				"secretKey": maskValue(conf.Providers.Aws.SecretKey),
				"region":    conf.Providers.Aws.Region,
			},
			HasSecrets: map[string]bool{
				"accessKey": conf.Providers.Aws.AccessKey != "",
				"secretKey": conf.Providers.Aws.SecretKey != "",
			},
		},
		{
			Name: "阿里云",
			Fields: map[string]string{
				"accessKey": maskValue(conf.Providers.Alicloud.AccessKey),
				"secretKey": maskValue(conf.Providers.Alicloud.SecretKey),
				"region":    conf.Providers.Alicloud.Region,
			},
			HasSecrets: map[string]bool{
				"accessKey": conf.Providers.Alicloud.AccessKey != "",
				"secretKey": conf.Providers.Alicloud.SecretKey != "",
			},
		},
		{
			Name: "腾讯云",
			Fields: map[string]string{
				"secretId":  maskValue(conf.Providers.Tencentcloud.SecretId),
				"secretKey": maskValue(conf.Providers.Tencentcloud.SecretKey),
				"region":    conf.Providers.Tencentcloud.Region,
			},
			HasSecrets: map[string]bool{
				"secretId":  conf.Providers.Tencentcloud.SecretId != "",
				"secretKey": conf.Providers.Tencentcloud.SecretKey != "",
			},
		},
		{
			Name: "火山引擎",
			Fields: map[string]string{
				"accessKey": maskValue(conf.Providers.Volcengine.AccessKey),
				"secretKey": maskValue(conf.Providers.Volcengine.SecretKey),
				"region":    conf.Providers.Volcengine.Region,
			},
			HasSecrets: map[string]bool{
				"accessKey": conf.Providers.Volcengine.AccessKey != "",
				"secretKey": conf.Providers.Volcengine.SecretKey != "",
			},
		},
		{
			Name: "华为云",
			Fields: map[string]string{
				"accessKey": maskValue(conf.Providers.Huaweicloud.AccessKey),
				"secretKey": maskValue(conf.Providers.Huaweicloud.SecretKey),
				"region":    conf.Providers.Huaweicloud.Region,
			},
			HasSecrets: map[string]bool{
				"accessKey": conf.Providers.Huaweicloud.AccessKey != "",
				"secretKey": conf.Providers.Huaweicloud.SecretKey != "",
			},
		},
		{
			Name: "Vultr",
			Fields: map[string]string{
				"apiKey": maskValue(conf.Providers.Vultr.ApiKey),
			},
			HasSecrets: map[string]bool{
				"apiKey": conf.Providers.Vultr.ApiKey != "",
			},
		},
		{
			Name: "Google Cloud",
			Fields: map[string]string{
				"credentials": conf.Providers.Google.Credentials,
				"project":     conf.Providers.Google.Project,
				"region":      conf.Providers.Google.Region,
			},
			HasSecrets: map[string]bool{},
		},
		{
			Name: "Azure",
			Fields: map[string]string{
				"clientId":       maskValue(conf.Providers.Azure.ClientId),
				"clientSecret":   maskValue(conf.Providers.Azure.ClientSecret),
				"subscriptionId": maskValue(conf.Providers.Azure.SubscriptionId),
				"tenantId":       maskValue(conf.Providers.Azure.TenantId),
			},
			HasSecrets: map[string]bool{
				"clientId":       conf.Providers.Azure.ClientId != "",
				"clientSecret":   conf.Providers.Azure.ClientSecret != "",
				"subscriptionId": conf.Providers.Azure.SubscriptionId != "",
				"tenantId":       conf.Providers.Azure.TenantId != "",
			},
		},
		{
			Name: "Oracle Cloud",
			Fields: map[string]string{
				"user":        maskValue(conf.Providers.Oracle.User),
				"tenancy":     maskValue(conf.Providers.Oracle.Tenancy),
				"fingerprint": maskValue(conf.Providers.Oracle.Fingerprint),
				"keyFile":     conf.Providers.Oracle.KeyFile,
				"region":      conf.Providers.Oracle.Region,
			},
			HasSecrets: map[string]bool{
				"user":        conf.Providers.Oracle.User != "",
				"tenancy":     conf.Providers.Oracle.Tenancy != "",
				"fingerprint": conf.Providers.Oracle.Fingerprint != "",
			},
		},
		{
			Name: "UCloud",
			Fields: map[string]string{
				"publicKey":  maskValue(conf.Providers.UCloud.PublicKey),
				"privateKey": maskValue(conf.Providers.UCloud.PrivateKey),
				"projectId":  conf.Providers.UCloud.ProjectId,
				"region":     conf.Providers.UCloud.Region,
			},
			HasSecrets: map[string]bool{
				"publicKey":  conf.Providers.UCloud.PublicKey != "",
				"privateKey": conf.Providers.UCloud.PrivateKey != "",
			},
		},
		{
			Name: "Ctyun",
			Fields: map[string]string{
				"accessKey": maskValue(conf.Providers.Ctyun.AccessKey),
				"secretKey": maskValue(conf.Providers.Ctyun.SecretKey),
				"region":    conf.Providers.Ctyun.Region,
			},
			HasSecrets: map[string]bool{
				"accessKey": conf.Providers.Ctyun.AccessKey != "",
				"secretKey": conf.Providers.Ctyun.SecretKey != "",
			},
		},
		{
			Name: "Cloudflare",
			Fields: map[string]string{
				"email":  conf.Cloudflare.Email,
				"apiKey": maskValue(conf.Cloudflare.APIKey),
			},
			HasSecrets: map[string]bool{
				"apiKey": conf.Cloudflare.APIKey != "",
			},
		},
	}

	return ProvidersConfigInfo{
		ConfigPath: configPath,
		Providers:  providers,
	}, nil
}

func (a *App) SaveProvidersConfig(providerName string, fields map[string]string, customPath string) error {
	conf, _, err := redc.ReadConfig(customPath)
	if err != nil {
		return err
	}

	switch providerName {
	case "AWS":
		if v, ok := fields["accessKey"]; ok && v != "" {
			conf.Providers.Aws.AccessKey = v
		}
		if v, ok := fields["secretKey"]; ok && v != "" {
			conf.Providers.Aws.SecretKey = v
		}
		if v, ok := fields["region"]; ok {
			conf.Providers.Aws.Region = v
		}
	case "阿里云":
		if v, ok := fields["accessKey"]; ok && v != "" {
			conf.Providers.Alicloud.AccessKey = v
		}
		if v, ok := fields["secretKey"]; ok && v != "" {
			conf.Providers.Alicloud.SecretKey = v
		}
		if v, ok := fields["region"]; ok {
			conf.Providers.Alicloud.Region = v
		}
	case "腾讯云":
		if v, ok := fields["secretId"]; ok && v != "" {
			conf.Providers.Tencentcloud.SecretId = v
		}
		if v, ok := fields["secretKey"]; ok && v != "" {
			conf.Providers.Tencentcloud.SecretKey = v
		}
		if v, ok := fields["region"]; ok {
			conf.Providers.Tencentcloud.Region = v
		}
	case "火山引擎":
		if v, ok := fields["accessKey"]; ok && v != "" {
			conf.Providers.Volcengine.AccessKey = v
		}
		if v, ok := fields["secretKey"]; ok && v != "" {
			conf.Providers.Volcengine.SecretKey = v
		}
		if v, ok := fields["region"]; ok {
			conf.Providers.Volcengine.Region = v
		}
	case "华为云":
		if v, ok := fields["accessKey"]; ok && v != "" {
			conf.Providers.Huaweicloud.AccessKey = v
		}
		if v, ok := fields["secretKey"]; ok && v != "" {
			conf.Providers.Huaweicloud.SecretKey = v
		}
		if v, ok := fields["region"]; ok {
			conf.Providers.Huaweicloud.Region = v
		}
	case "Vultr":
		if v, ok := fields["apiKey"]; ok && v != "" {
			conf.Providers.Vultr.ApiKey = v
		}
	case "Google Cloud":
		if v, ok := fields["credentials"]; ok && v != "" {
			conf.Providers.Google.Credentials = v
		}
		if v, ok := fields["project"]; ok {
			conf.Providers.Google.Project = v
		}
		if v, ok := fields["region"]; ok {
			conf.Providers.Google.Region = v
		}
	case "Azure":
		if v, ok := fields["clientId"]; ok && v != "" {
			conf.Providers.Azure.ClientId = v
		}
		if v, ok := fields["clientSecret"]; ok && v != "" {
			conf.Providers.Azure.ClientSecret = v
		}
		if v, ok := fields["subscriptionId"]; ok && v != "" {
			conf.Providers.Azure.SubscriptionId = v
		}
		if v, ok := fields["tenantId"]; ok && v != "" {
			conf.Providers.Azure.TenantId = v
		}
	case "Oracle Cloud":
		if v, ok := fields["user"]; ok && v != "" {
			conf.Providers.Oracle.User = v
		}
		if v, ok := fields["tenancy"]; ok && v != "" {
			conf.Providers.Oracle.Tenancy = v
		}
		if v, ok := fields["fingerprint"]; ok && v != "" {
			conf.Providers.Oracle.Fingerprint = v
		}
		if v, ok := fields["keyFile"]; ok {
			conf.Providers.Oracle.KeyFile = v
		}
		if v, ok := fields["region"]; ok {
			conf.Providers.Oracle.Region = v
		}
	case "UCloud":
		if v, ok := fields["publicKey"]; ok && v != "" {
			conf.Providers.UCloud.PublicKey = v
		}
		if v, ok := fields["privateKey"]; ok && v != "" {
			conf.Providers.UCloud.PrivateKey = v
		}
		if v, ok := fields["projectId"]; ok {
			conf.Providers.UCloud.ProjectId = v
		}
		if v, ok := fields["region"]; ok {
			conf.Providers.UCloud.Region = v
		}
	case "Ctyun":
		if v, ok := fields["accessKey"]; ok && v != "" {
			conf.Providers.Ctyun.AccessKey = v
		}
		if v, ok := fields["secretKey"]; ok && v != "" {
			conf.Providers.Ctyun.SecretKey = v
		}
		if v, ok := fields["region"]; ok {
			conf.Providers.Ctyun.Region = v
		}
	case "Cloudflare":
		if v, ok := fields["email"]; ok {
			conf.Cloudflare.Email = v
		}
		if v, ok := fields["apiKey"]; ok && v != "" {
			conf.Cloudflare.APIKey = v
		}
	default:
		return fmt.Errorf(i18n.Tf("app_unknown_provider", providerName))
	}

	if err := redc.SaveConfig(conf, customPath); err != nil {
		return err
	}

	a.emitLog(i18n.Tf("app_cred_updated", providerName))
	return nil
}

func (a *App) ListProfiles() ([]redc.ProfileInfo, error) {
	return redc.ListProfiles()
}

func (a *App) GetActiveProfile() (redc.ProfileInfo, error) {
	return redc.GetActiveProfile()
}

func (a *App) SetActiveProfile(profileID string) (redc.ProfileInfo, error) {
	return redc.SetActiveProfile(profileID)
}

func (a *App) CreateProfile(name string, configPath string, templateDir string) (redc.ProfileInfo, error) {
	return redc.CreateProfile(name, configPath, templateDir)
}

func (a *App) UpdateProfile(profileID string, name string, configPath string, templateDir string) (redc.ProfileInfo, error) {
	return redc.UpdateProfile(profileID, name, configPath, templateDir)
}

func (a *App) DeleteProfile(profileID string) error {
	return redc.DeleteProfile(profileID)
}

func (a *App) UpdateProfileAIConfig(profileID string, provider string, apiKey string, baseUrl string, model string) error {
	aiConfig := &redc.AIConfig{
		Provider: provider,
		APIKey:   apiKey,
		BaseURL:  baseUrl,
		Model:    model,
	}
	return redc.UpdateProfileAIConfig(profileID, aiConfig)
}

func (a *App) ListProjects() ([]ProjectInfo, error) {
	projects, err := redc.ListAllProjects()
	if err != nil {
		return nil, err
	}

	result := make([]ProjectInfo, 0, len(projects))
	for _, p := range projects {
		result = append(result, ProjectInfo{
			Name:       p.ProjectName,
			Path:       p.ProjectPath,
			CreateTime: p.CreateTime,
			User:       p.User,
		})
	}
	return result, nil
}

func (a *App) GetCurrentProject() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.project == nil {
		return ""
	}
	return a.project.ProjectName
}

func (a *App) SwitchProject(projectName string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Parse and load the new project
	p, err := redc.ProjectParse(projectName, redc.U)
	if err != nil {
		return fmt.Errorf(i18n.Tf("app_switch_project_failed", err))
	}

	// Update project reference
	a.project = p

	// Update log manager to use new project path
	a.logMgr = gologger.NewLogManager(p.ProjectPath)

	// Update global project variable
	redc.Project = projectName

	// Emit log and refresh
	a.emitLog(i18n.Tf("app_project_switched", projectName))
	a.emitRefresh()

	return nil
}

func (a *App) CreateProject(name string) error {
	_, err := redc.NewProjectConfig(name, redc.U)
	if err != nil {
		return fmt.Errorf(i18n.Tf("app_create_project_failed", err))
	}
	a.emitLog(i18n.Tf("app_project_created", name))
	return nil
}

func (a *App) ListCases() ([]CaseInfo, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.project == nil {
		if a.initError != "" {
			return nil, fmt.Errorf(a.initError)
		}
		return nil, fmt.Errorf("%s", i18n.T("app_project_not_loaded"))
	}

	cases, err := redc.LoadProjectCases(a.project.ProjectName)
	if err != nil {
		return nil, err
	}

	result := make([]CaseInfo, 0, len(cases))
	for _, c := range cases {
		result = append(result, CaseInfo{
			ID:         c.Id,
			Name:       c.Name,
			Type:       c.Type,
			State:      c.State,
			StateTime:  c.StateTime,
			CreateTime: c.CreateTime,
			Operator:   c.Operator,
		})
	}
	return result, nil
}

func (a *App) GetResourceSummary() ([]ResourceSummary, error) {
	a.mu.Lock()
	project := a.project
	a.mu.Unlock()

	if project == nil {
		if a.initError != "" {
			return nil, fmt.Errorf(a.initError)
		}
		return nil, fmt.Errorf("%s", i18n.T("app_project_not_loaded"))
	}

	cases, err := redc.LoadProjectCases(project.ProjectName)
	if err != nil {
		return nil, err
	}

	counts := make(map[string]int)
	for _, c := range cases {
		if c.State != redc.StateRunning {
			continue
		}
		if c.Path == "" {
			continue
		}
		state, err := redc.TfStatus(c.Path)
		if err != nil || state == nil || state.Values == nil {
			continue
		}
		addModuleResources(counts, state.Values.RootModule)
	}

	result := make([]ResourceSummary, 0, len(counts))
	for typ, count := range counts {
		result = append(result, ResourceSummary{Type: typ, Count: count})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Type < result[j].Type })
	return result, nil
}

func addModuleResources(counts map[string]int, module *tfjson.StateModule) {
	if module == nil {
		return
	}
	for _, res := range module.Resources {
		if res.Type != "" {
			counts[res.Type]++
		}
	}
	for _, child := range module.ChildModules {
		addModuleResources(counts, child)
	}
}

func (a *App) GetBalances(providers []string) ([]BalanceInfo, error) {
	if len(providers) == 0 {
		providers = []string{"aliyun", "tencentcloud", "volcengine", "huaweicloud", "ucloud", "vultr", "aws"}
	}

	conf, _, err := redc.ReadConfig(redc.ActiveConfigPath)
	if err != nil {
		return nil, err
	}

	results := make([]BalanceInfo, 0, len(providers))
	for _, p := range providers {
		result := BalanceInfo{
			Provider:  p,
			Amount:    "-",
			Currency:  "-",
			UpdatedAt: time.Now().Format("2006-01-02 15:04:05"),
		}
		switch p {
		case "aliyun":
			if conf.Providers.Alicloud.AccessKey == "" || conf.Providers.Alicloud.SecretKey == "" {
				result.Error = i18n.T("app_cred_aliyun_missing")
			} else {
				amount, currency, err := redc.QueryAliyunBalance(conf.Providers.Alicloud.AccessKey, conf.Providers.Alicloud.SecretKey, conf.Providers.Alicloud.Region)
				if err != nil {
					result.Error = err.Error()
				} else {
					result.Amount = amount
					result.Currency = currency
				}
			}
		case "tencentcloud":
			if conf.Providers.Tencentcloud.SecretId == "" || conf.Providers.Tencentcloud.SecretKey == "" {
				result.Error = i18n.T("app_cred_tencent_missing")
			} else {
				amount, currency, err := redc.QueryTencentBalance(conf.Providers.Tencentcloud.SecretId, conf.Providers.Tencentcloud.SecretKey, conf.Providers.Tencentcloud.Region)
				if err != nil {
					result.Error = err.Error()
				} else {
					result.Amount = amount
					result.Currency = currency
				}
			}
		case "volcengine":
			if conf.Providers.Volcengine.AccessKey == "" || conf.Providers.Volcengine.SecretKey == "" {
				result.Error = i18n.T("app_cred_volcengine_missing")
			} else {
				amount, currency, err := redc.QueryVolcengineBalance(conf.Providers.Volcengine.AccessKey, conf.Providers.Volcengine.SecretKey, conf.Providers.Volcengine.Region)
				if err != nil {
					result.Error = err.Error()
				} else {
					result.Amount = amount
					result.Currency = currency
				}
			}
		case "huaweicloud":
			if conf.Providers.Huaweicloud.AccessKey == "" || conf.Providers.Huaweicloud.SecretKey == "" {
				result.Error = i18n.T("app_cred_huawei_missing")
			} else {
				amount, currency, err := redc.QueryHuaweiBalance(conf.Providers.Huaweicloud.AccessKey, conf.Providers.Huaweicloud.SecretKey, conf.Providers.Huaweicloud.Region)
				if err != nil {
					result.Error = err.Error()
				} else {
					result.Amount = amount
					result.Currency = currency
				}
			}
		case "ucloud":
			if conf.Providers.UCloud.PublicKey == "" || conf.Providers.UCloud.PrivateKey == "" {
				result.Error = i18n.T("app_cred_ucloud_missing")
			} else {
				amount, currency, err := redc.QueryUCloudBalance(conf.Providers.UCloud.PublicKey, conf.Providers.UCloud.PrivateKey, conf.Providers.UCloud.Region)
				if err != nil {
					result.Error = err.Error()
				} else {
					result.Amount = amount
					result.Currency = currency
				}
			}
		case "vultr":
			if conf.Providers.Vultr.ApiKey == "" {
				result.Error = i18n.T("app_cred_vultr_missing")
			} else {
				amount, currency, err := redc.QueryVultrBalance(conf.Providers.Vultr.ApiKey)
				if err != nil {
					result.Error = err.Error()
				} else {
					result.Amount = amount
					result.Currency = currency
				}
			}
		case "aws":
			if conf.Providers.Aws.AccessKey == "" || conf.Providers.Aws.SecretKey == "" {
				result.Error = i18n.T("app_cred_aws_missing")
			} else {
				amount, currency, err := redc.QueryAWSBill(conf.Providers.Aws.AccessKey, conf.Providers.Aws.SecretKey, conf.Providers.Aws.Region)
				if err != nil {
					result.Error = err.Error()
				} else {
					result.Amount = amount
					result.Currency = currency
				}
			}
		default:
			result.Error = i18n.T("app_provider_unsupported")
		}
		results = append(results, result)
	}
	return results, nil
}

func (a *App) GetBills(providers []string) ([]BillInfo, error) {
	if len(providers) == 0 {
		providers = []string{"aws", "vultr"}
	}

	conf, _, err := redc.ReadConfig(redc.ActiveConfigPath)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	startDate := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).Format("2006-01-02")
	endDate := now.Format("2006-01-02")
	month := now.Format("2006-01")

	results := make([]BillInfo, 0, len(providers))
	for _, p := range providers {
		result := BillInfo{
			Provider:  p,
			Month:     month,
			StartDate: startDate,
			EndDate:   endDate,
		}

		switch p {
		case "aws":
			if conf.Providers.Aws.AccessKey == "" || conf.Providers.Aws.SecretKey == "" {
				result.Error = i18n.T("app_cred_aws_missing")
			} else {
				amount, currency, err := redc.QueryAWSBill(conf.Providers.Aws.AccessKey, conf.Providers.Aws.SecretKey, conf.Providers.Aws.Region)
				if err != nil {
					result.Error = err.Error()
				} else {
					result.TotalAmount = amount
					result.Currency = currency
				}
			}
		case "gcp":
			if conf.Providers.Google.Credentials == "" {
				result.Error = i18n.T("app_cred_gcp_missing")
			} else {
				amount, currency, err := redc.QueryGCPBillFromConfig(conf.Providers.Google.Credentials, conf.Providers.Google.Project, conf.Providers.Google.Region)
				if err != nil {
					result.Error = err.Error()
				} else {
					result.TotalAmount = amount
					result.Currency = currency
				}
			}
		case "vultr":
			if conf.Providers.Vultr.ApiKey == "" {
				result.Error = i18n.T("app_cred_vultr_missing")
			} else {
				amount, currency, err := redc.QueryVultrBill(conf.Providers.Vultr.ApiKey)
				if err != nil {
					result.Error = err.Error()
				} else {
					result.TotalAmount = amount
					result.Currency = currency
				}
			}
		default:
			result.Error = i18n.T("app_provider_unsupported")
		}
		results = append(results, result)
	}
	return results, nil
}
