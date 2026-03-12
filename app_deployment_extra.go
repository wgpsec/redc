package main

import (
	"fmt"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"red-cloud/i18n"
	redc "red-cloud/mod"
	"red-cloud/utils/sshutil"
	"time"
)

func (a *App) GetBaseTemplates() ([]*redc.BaseTemplate, error) {
	runtime.LogInfof(a.ctx, i18n.T("app_template_scan_start"))

	a.mu.Lock()
	templateMgr := a.templateManager
	a.mu.Unlock()

	if templateMgr == nil {
		runtime.LogErrorf(a.ctx, i18n.T("app_template_mgr_not_init"))
		return []*redc.BaseTemplate{}, nil // 返回空列表而不是错误
	}

	templates, err := templateMgr.ScanBaseTemplates()
	if err != nil {
		runtime.LogErrorf(a.ctx, i18n.Tf("app_template_scan_failed", err))
		return []*redc.BaseTemplate{}, nil // 返回空列表而不是错误
	}

	if templates == nil {
		templates = []*redc.BaseTemplate{}
	}

	runtime.LogInfof(a.ctx, i18n.Tf("app_template_scan_done", len(templates)))
	return templates, nil
}

func (a *App) GetTemplateMetadata(name string) (*redc.BaseTemplate, error) {
	a.mu.Lock()
	templateMgr := a.templateManager
	a.mu.Unlock()

	if templateMgr == nil {
		return nil, fmt.Errorf("%s", i18n.T("app_template_mgr_not_init"))
	}

	// 获取所有基础模板
	templates, err := templateMgr.ScanBaseTemplates()
	if err != nil {
		return nil, fmt.Errorf(i18n.Tf("app_template_scan_failed", err))
	}

	// 查找指定名称的模板
	for _, template := range templates {
		if template.Name == name {
			return template, nil
		}
	}

	return nil, fmt.Errorf(i18n.Tf("app_template_not_exist", name))
}

func (a *App) GetProviderRegions(provider string) ([]redc.Region, error) {
	a.mu.Lock()
	service := a.customDeploymentService
	a.mu.Unlock()

	if service == nil {
		return nil, fmt.Errorf("%s", i18n.T("app_deploy_service_not_init"))
	}

	regions, err := service.GetProviderRegions(provider)
	if err != nil {
		return nil, fmt.Errorf(i18n.Tf("app_region_load_failed", err))
	}

	return regions, nil
}

func (a *App) GetInstanceTypes(provider, region string) ([]redc.InstanceType, error) {
	a.mu.Lock()
	service := a.customDeploymentService
	a.mu.Unlock()

	if service == nil {
		return nil, fmt.Errorf("%s", i18n.T("app_deploy_service_not_init"))
	}

	types, err := service.GetInstanceTypes(provider, region)
	if err != nil {
		return nil, fmt.Errorf(i18n.Tf("app_instance_type_failed", err))
	}

	return types, nil
}

func (a *App) ValidateDeploymentConfig(config *redc.DeploymentConfig) (*redc.ValidationResult, error) {
	a.mu.Lock()
	service := a.customDeploymentService
	a.mu.Unlock()

	if service == nil {
		return nil, fmt.Errorf("%s", i18n.T("app_deploy_service_not_init"))
	}

	if config == nil {
		return nil, fmt.Errorf("%s", i18n.T("app_config_empty"))
	}

	validator := redc.NewConfigValidator()
	result, err := validator.ValidateDeploymentConfig(config)
	if err != nil {
		return nil, fmt.Errorf(i18n.Tf("app_validate_failed", err))
	}

	return result, nil
}

func (a *App) EstimateDeploymentCost(config *redc.DeploymentConfig) (*redc.CostEstimate, error) {
	a.mu.Lock()
	service := a.customDeploymentService
	pricingService := a.pricingService
	costCalculator := a.costCalculator
	a.mu.Unlock()

	if service == nil {
		return nil, fmt.Errorf("%s", i18n.T("app_deploy_service_not_init"))
	}

	if config == nil {
		return nil, fmt.Errorf("%s", i18n.T("app_config_empty"))
	}

	estimate, err := service.EstimateCost(config, pricingService, costCalculator)
	if err != nil {
		return nil, fmt.Errorf(i18n.Tf("app_estimate_failed", err))
	}

	return estimate, nil
}

func (a *App) CreateCustomDeployment(config *redc.DeploymentConfig) (*redc.CustomDeployment, error) {
	a.mu.Lock()
	service := a.customDeploymentService
	project := a.project
	a.mu.Unlock()

	if service == nil {
		return nil, fmt.Errorf("%s", i18n.T("app_deploy_service_not_init"))
	}

	if project == nil {
		return nil, fmt.Errorf("%s", i18n.T("app_project_not_init"))
	}

	if config == nil {
		return nil, fmt.Errorf("%s", i18n.T("app_config_empty"))
	}

	deployment, err := service.CreateCustomDeployment(config, project.ProjectPath, project.ProjectName)
	if err != nil {
		return nil, fmt.Errorf(i18n.Tf("app_deploy_create_failed", err))
	}

	a.emitLog(i18n.Tf("app_deploy_custom_success", deployment.Name))
	a.emitRefresh()

	return deployment, nil
}

func (a *App) ListCustomDeployments() ([]*redc.CustomDeployment, error) {
	a.mu.Lock()
	service := a.customDeploymentService
	project := a.project
	a.mu.Unlock()

	if service == nil {
		return nil, fmt.Errorf("%s", i18n.T("app_deploy_service_not_init"))
	}

	if project == nil {
		return nil, fmt.Errorf("%s", i18n.T("app_project_not_init"))
	}

	deployments, err := service.ListCustomDeployments(project.ProjectName)
	if err != nil {
		return nil, fmt.Errorf(i18n.Tf("app_deploy_list_failed", err))
	}

	return deployments, nil
}

func (a *App) StartCustomDeployment(id string) error {
	a.mu.Lock()
	service := a.customDeploymentService
	project := a.project
	a.mu.Unlock()

	if service == nil {
		return fmt.Errorf("%s", i18n.T("app_deploy_service_not_init"))
	}

	if project == nil {
		return fmt.Errorf("%s", i18n.T("app_project_not_init"))
	}

	err := service.StartCustomDeployment(project.ProjectName, id, project.ProjectPath)
	if err != nil {
		return fmt.Errorf(i18n.Tf("app_deploy_start_failed", err))
	}

	a.emitLog(i18n.Tf("app_deploy_start_success", id))
	a.emitRefresh()

	return nil
}

func (a *App) StopCustomDeployment(id string) error {
	a.mu.Lock()
	service := a.customDeploymentService
	project := a.project
	a.mu.Unlock()

	if service == nil {
		return fmt.Errorf("%s", i18n.T("app_deploy_service_not_init"))
	}

	if project == nil {
		return fmt.Errorf("%s", i18n.T("app_project_not_init"))
	}

	err := service.StopCustomDeployment(project.ProjectName, id, project.ProjectPath)
	if err != nil {
		return fmt.Errorf(i18n.Tf("app_deploy_stop_failed", err))
	}

	a.emitLog(i18n.Tf("app_deploy_stop_success", id))
	a.emitRefresh()

	return nil
}

func (a *App) DeleteCustomDeployment(id string) error {
	a.mu.Lock()
	service := a.customDeploymentService
	project := a.project
	a.mu.Unlock()

	if service == nil {
		return fmt.Errorf("%s", i18n.T("app_deploy_service_not_init"))
	}

	if project == nil {
		return fmt.Errorf("%s", i18n.T("app_project_not_init"))
	}

	err := service.DeleteCustomDeployment(project.ProjectName, id, project.ProjectPath)
	if err != nil {
		return fmt.Errorf(i18n.Tf("app_deploy_delete_failed", err))
	}

	a.emitLog(i18n.Tf("app_deploy_delete_success", id))
	a.emitRefresh()

	return nil
}

func (a *App) getSSHConfig(caseID string) (*sshutil.SSHConfig, error) {
	a.mu.Lock()
	project := a.project
	service := a.customDeploymentService
	a.mu.Unlock()

	if project == nil {
		return nil, fmt.Errorf("%s", i18n.T("app_project_not_loaded"))
	}

	fmt.Printf("[DEBUG getSSHConfig] 尝试获取 SSH 配置，ID: %s\n", caseID)

	// 先尝试作为 Case 处理
	c, caseErr := project.GetCase(caseID)
	if caseErr == nil {
		fmt.Printf("[DEBUG getSSHConfig] 找到 Case: %s\n", caseID)
		// 是 Case
		return c.GetSSHConfig()
	}

	fmt.Printf("[DEBUG getSSHConfig] 不是 Case (错误: %v)，尝试作为部署处理\n", caseErr)

	// 不是 Case，尝试作为自定义部署处理
	if service != nil {
		sshConfig, err := a.getDeploymentSSHConfig(caseID)
		if err == nil {
			fmt.Printf("[DEBUG getSSHConfig] 成功从部署获取 SSH 配置\n")
			return sshConfig, nil
		}
		fmt.Printf("[DEBUG getSSHConfig] 从部署获取 SSH 配置失败: %v\n", err)
		// 返回更详细的错误信息
		return nil, fmt.Errorf("找不到场景或部署 '%s': Case错误=%v, 部署错误=%v", caseID, caseErr, err)
	}

	// 自定义部署服务未初始化
	fmt.Printf("[DEBUG getSSHConfig] 自定义部署服务未初始化\n")
	return nil, fmt.Errorf(i18n.Tf("app_case_not_found", caseErr))
}

func (a *App) getDeploymentSSHConfig(deploymentID string) (*sshutil.SSHConfig, error) {
	a.mu.Lock()
	service := a.customDeploymentService
	project := a.project
	a.mu.Unlock()

	fmt.Printf("[DEBUG getDeploymentSSHConfig] 开始查找部署，ID: %s\n", deploymentID)

	if service == nil {
		fmt.Printf("[DEBUG getDeploymentSSHConfig] 自定义部署服务未初始化\n")
		return nil, fmt.Errorf("%s", i18n.T("app_deploy_service_not_init"))
	}

	if project == nil {
		fmt.Printf("[DEBUG getDeploymentSSHConfig] 项目未初始化\n")
		return nil, fmt.Errorf("%s", i18n.T("app_project_not_init"))
	}

	fmt.Printf("[DEBUG getDeploymentSSHConfig] 项目名称: %s\n", project.ProjectName)

	// 加载部署信息
	deployments, err := service.ListCustomDeployments(project.ProjectName)
	if err != nil {
		fmt.Printf("[DEBUG getDeploymentSSHConfig] 加载部署列表失败: %v\n", err)
		return nil, fmt.Errorf(i18n.Tf("app_deploy_load_failed", err))
	}

	fmt.Printf("[DEBUG getDeploymentSSHConfig] 查找部署 ID: %s, 总共有 %d 个部署\n", deploymentID, len(deployments))

	// 查找指定的部署
	var deployment *redc.CustomDeployment
	for i, d := range deployments {
		fmt.Printf("[DEBUG getDeploymentSSHConfig] [%d] 检查部署: ID=%s, Name=%s, State=%s, HasOutputs=%v\n",
			i, d.ID, d.Name, d.State, d.Outputs != nil)
		if d.ID == deploymentID {
			deployment = d
			fmt.Printf("[DEBUG getDeploymentSSHConfig] ✓ 找到匹配的部署！\n")
			break
		}
	}

	if deployment == nil {
		fmt.Printf("[DEBUG getDeploymentSSHConfig] ✗ 未找到部署: %s\n", deploymentID)
		return nil, fmt.Errorf(i18n.Tf("app_deploy_not_found", deploymentID))
	}

	fmt.Printf("[DEBUG getDeploymentSSHConfig] 找到部署: %s\n", deployment.Name)
	fmt.Printf("[DEBUG getDeploymentSSHConfig] Outputs 类型: %T\n", deployment.Outputs)
	fmt.Printf("[DEBUG getDeploymentSSHConfig] Outputs 内容: %+v\n", deployment.Outputs)

	// 解析 outputs
	var outputs map[string]interface{}
	if deployment.Outputs != nil {
		outputs = deployment.Outputs
		fmt.Printf("[DEBUG getDeploymentSSHConfig] Outputs 键列表: ")
		for key := range outputs {
			fmt.Printf("%s ", key)
		}
		fmt.Printf("\n")
	} else {
		fmt.Printf("[DEBUG getDeploymentSSHConfig] 部署没有 outputs 信息\n")
		return nil, fmt.Errorf("%s", i18n.T("app_deploy_no_outputs"))
	}

	// 从 outputs 获取 SSH 信息
	fmt.Printf("[DEBUG getDeploymentSSHConfig] 尝试获取 public_ip...\n")
	publicIPRaw, exists := outputs["public_ip"]
	fmt.Printf("[DEBUG getDeploymentSSHConfig] public_ip 存在: %v, 值: %v, 类型: %T\n", exists, publicIPRaw, publicIPRaw)

	publicIP, ok := publicIPRaw.(string)
	if !ok || publicIP == "" {
		fmt.Printf("[DEBUG getDeploymentSSHConfig] ✗ 未找到公网 IP 或类型转换失败\n")
		return nil, fmt.Errorf("未找到公网 IP，outputs: %v", outputs)
	}
	fmt.Printf("[DEBUG getDeploymentSSHConfig] ✓ 公网 IP: %s\n", publicIP)

	fmt.Printf("[DEBUG getDeploymentSSHConfig] 尝试获取 instance_password...\n")
	passwordRaw, exists := outputs["instance_password"]
	fmt.Printf("[DEBUG getDeploymentSSHConfig] instance_password 存在: %v, 值: %v, 类型: %T\n", exists, passwordRaw, passwordRaw)

	password, ok := passwordRaw.(string)
	if !ok || password == "" {
		fmt.Printf("[DEBUG getDeploymentSSHConfig] ✗ 未找到实例密码或类型转换失败\n")
		return nil, fmt.Errorf("未找到实例密码，outputs: %v", outputs)
	}
	fmt.Printf("[DEBUG getDeploymentSSHConfig] ✓ 实例密码: %s (长度: %d)\n", password, len(password))

	fmt.Printf("[DEBUG getDeploymentSSHConfig] SSH 配置: Host=%s, User=root\n", publicIP)

	// 构建 SSH 配置
	config := &sshutil.SSHConfig{
		Host:     publicIP,
		Port:     22,
		User:     "root",
		Password: password,
		Timeout:  30 * time.Second, // 30 秒超时
	}

	return config, nil
}

func (a *App) GetDeploymentHistory(id string) ([]*redc.DeploymentChangeHistory, error) {
	a.mu.Lock()
	service := a.customDeploymentService
	project := a.project
	a.mu.Unlock()

	if service == nil {
		return nil, fmt.Errorf("%s", i18n.T("app_deploy_service_not_init"))
	}

	if project == nil {
		return nil, fmt.Errorf("%s", i18n.T("app_project_not_init"))
	}

	history, err := service.GetDeploymentHistory(project.ProjectName, id)
	if err != nil {
		return nil, fmt.Errorf(i18n.Tf("app_deploy_history_failed", err))
	}

	return history, nil
}

func (a *App) BatchStartCustomDeployments(ids []string) []redc.BatchOperationResult {
	a.mu.Lock()
	service := a.customDeploymentService
	project := a.project
	a.mu.Unlock()

	if service == nil {
		return []redc.BatchOperationResult{{
			Success: false,
			Error:   "自定义部署服务未初始化",
		}}
	}

	if project == nil {
		return []redc.BatchOperationResult{{
			Success: false,
			Error:   "项目未初始化",
		}}
	}

	results := service.BatchStartDeployments(project.ProjectName, ids, project.ProjectPath)

	a.emitLog(i18n.Tf("app_batch_start", countSuccessful(results), countFailed(results)))
	a.emitRefresh()

	return results
}

func (a *App) BatchStopCustomDeployments(ids []string) []redc.BatchOperationResult {
	a.mu.Lock()
	service := a.customDeploymentService
	project := a.project
	a.mu.Unlock()

	if service == nil {
		return []redc.BatchOperationResult{{
			Success: false,
			Error:   "自定义部署服务未初始化",
		}}
	}

	if project == nil {
		return []redc.BatchOperationResult{{
			Success: false,
			Error:   "项目未初始化",
		}}
	}

	results := service.BatchStopDeployments(project.ProjectName, ids, project.ProjectPath)

	a.emitLog(i18n.Tf("app_batch_stop", countSuccessful(results), countFailed(results)))
	a.emitRefresh()

	return results
}

func (a *App) BatchDeleteCustomDeployments(ids []string) []redc.BatchOperationResult {
	a.mu.Lock()
	service := a.customDeploymentService
	project := a.project
	a.mu.Unlock()

	if service == nil {
		return []redc.BatchOperationResult{{
			Success: false,
			Error:   "自定义部署服务未初始化",
		}}
	}

	if project == nil {
		return []redc.BatchOperationResult{{
			Success: false,
			Error:   "项目未初始化",
		}}
	}

	results := service.BatchDeleteDeployments(project.ProjectName, ids, project.ProjectPath)

	a.emitLog(i18n.Tf("app_batch_delete", countSuccessful(results), countFailed(results)))
	a.emitRefresh()

	return results
}

func countSuccessful(results []redc.BatchOperationResult) int {
	count := 0
	for _, r := range results {
		if r.Success {
			count++
		}
	}
	return count
}

func countFailed(results []redc.BatchOperationResult) int {
	count := 0
	for _, r := range results {
		if !r.Success {
			count++
		}
	}
	return count
}

func (a *App) SaveConfigTemplate(name string, config *redc.DeploymentConfig) error {
	a.mu.Lock()
	configStore := a.configStore
	a.mu.Unlock()

	if configStore == nil {
		return fmt.Errorf("%s", i18n.T("app_config_store_not_init"))
	}

	if name == "" {
		return fmt.Errorf("%s", i18n.T("app_config_name_empty"))
	}

	if config == nil {
		return fmt.Errorf("%s", i18n.T("app_config_empty"))
	}

	err := configStore.SaveConfigTemplate(name, config)
	if err != nil {
		return fmt.Errorf(i18n.Tf("app_config_save_failed", err))
	}

	a.emitLog(i18n.Tf("app_config_save_success", name))

	return nil
}

func (a *App) LoadConfigTemplate(name string) (*redc.DeploymentConfig, error) {
	a.mu.Lock()
	configStore := a.configStore
	a.mu.Unlock()

	if configStore == nil {
		return nil, fmt.Errorf("%s", i18n.T("app_config_store_not_init"))
	}

	if name == "" {
		return nil, fmt.Errorf("%s", i18n.T("app_config_name_empty"))
	}

	config, err := configStore.LoadConfigTemplate(name)
	if err != nil {
		return nil, fmt.Errorf(i18n.Tf("app_config_load_failed", err))
	}

	return config, nil
}

func (a *App) ListConfigTemplates() ([]string, error) {
	a.mu.Lock()
	configStore := a.configStore
	a.mu.Unlock()

	if configStore == nil {
		return nil, fmt.Errorf("%s", i18n.T("app_config_store_not_init"))
	}

	templates, err := configStore.ListConfigTemplates()
	if err != nil {
		return nil, fmt.Errorf(i18n.Tf("app_config_list_failed", err))
	}

	return templates, nil
}

func (a *App) DeleteConfigTemplate(name string) error {
	a.mu.Lock()
	configStore := a.configStore
	a.mu.Unlock()

	if configStore == nil {
		return fmt.Errorf("%s", i18n.T("app_config_store_not_init"))
	}

	if name == "" {
		return fmt.Errorf("%s", i18n.T("app_config_name_empty"))
	}

	err := configStore.DeleteConfigTemplate(name)
	if err != nil {
		return fmt.Errorf(i18n.Tf("app_config_delete_failed", err))
	}

	a.emitLog(i18n.Tf("app_config_delete_success", name))

	return nil
}

func (a *App) ExportConfigTemplate(name string, exportPath string) error {
	a.mu.Lock()
	configStore := a.configStore
	a.mu.Unlock()

	if configStore == nil {
		return fmt.Errorf("%s", i18n.T("app_config_store_not_init"))
	}

	if name == "" {
		return fmt.Errorf("%s", i18n.T("app_config_name_empty"))
	}

	if exportPath == "" {
		return fmt.Errorf("%s", i18n.T("app_config_export_path_empty"))
	}

	err := configStore.ExportConfigTemplate(name, exportPath)
	if err != nil {
		return fmt.Errorf(i18n.Tf("app_config_export_failed", err))
	}

	a.emitLog(i18n.Tf("app_config_export_success", name, exportPath))

	return nil
}

func (a *App) ImportConfigTemplate(name string, importPath string) error {
	a.mu.Lock()
	configStore := a.configStore
	a.mu.Unlock()

	if configStore == nil {
		return fmt.Errorf("%s", i18n.T("app_config_store_not_init"))
	}

	if name == "" {
		return fmt.Errorf("%s", i18n.T("app_config_name_empty"))
	}

	if importPath == "" {
		return fmt.Errorf("%s", i18n.T("app_config_import_path_empty"))
	}

	err := configStore.ImportConfigTemplate(name, importPath)
	if err != nil {
		return fmt.Errorf(i18n.Tf("app_config_import_failed", err))
	}

	a.emitLog(i18n.Tf("app_config_import_success", name, importPath))

	return nil
}

// CloneCustomDeployment clones an existing custom deployment with the same config
func (a *App) CloneCustomDeployment(deploymentID string, cloneName string) (*redc.CustomDeployment, error) {
	a.mu.Lock()
	service := a.customDeploymentService
	project := a.project
	a.mu.Unlock()

	if service == nil {
		return nil, fmt.Errorf("%s", i18n.T("app_deploy_service_not_init"))
	}
	if project == nil {
		return nil, fmt.Errorf("%s", i18n.T("app_project_not_init"))
	}

	// Load all deployments to find source
	deployments, err := service.ListCustomDeployments(project.ProjectName)
	if err != nil {
		return nil, fmt.Errorf(i18n.Tf("app_clone_load_failed", err))
	}

	var source *redc.CustomDeployment
	for _, d := range deployments {
		if d.ID == deploymentID {
			source = d
			break
		}
	}
	if source == nil {
		return nil, fmt.Errorf(i18n.Tf("app_clone_not_found", deploymentID))
	}

	// Deep copy config
	configName := cloneName
	if configName == "" {
		configName = source.Config.Name + "-clone"
	}
	cloneConfig := &redc.DeploymentConfig{
		Name:           configName,
		TemplateName:   source.Config.TemplateName,
		Provider:       source.Config.Provider,
		Region:         source.Config.Region,
		InstanceType:   source.Config.InstanceType,
		Userdata:       source.Config.Userdata,
		IsSpotInstance: source.Config.IsSpotInstance,
		Variables:      make(map[string]string),
	}
	for k, v := range source.Config.Variables {
		cloneConfig.Variables[k] = v
	}

	deployment, err := service.CreateCustomDeployment(cloneConfig, project.ProjectPath, project.ProjectName)
	if err != nil {
		return nil, fmt.Errorf(i18n.Tf("app_clone_failed", err))
	}

	a.emitLog(i18n.Tf("app_clone_success", deployment.Name, deployment.ID))
	a.emitRefresh()

	return deployment, nil
}
