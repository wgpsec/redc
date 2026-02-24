package mod

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"red-cloud/mod/cost"
	"red-cloud/mod/gologger"
	"red-cloud/pb"
	"red-cloud/utils"
	"sort"
	"sync"
	"time"

	bolt "go.etcd.io/bbolt"
	"google.golang.org/protobuf/proto"
)

// CustomDeploymentService 自定义部署服务
type CustomDeploymentService struct {
	templateMgr *TemplateManager
	validator   *ConfigValidator
	executor    *DeploymentExecutor
	configStore *ConfigStore
}
// EstimateCost 估算部署成本
// 集成现有的 cost.CostCalculator 根据部署配置估算成本
func (s *CustomDeploymentService) EstimateCost(config *DeploymentConfig, pricingService *cost.PricingService, costCalculator *cost.CostCalculator) (*CostEstimate, error) {
	if config == nil {
		return nil, fmt.Errorf("部署配置不能为空")
	}

	if pricingService == nil {
		return nil, fmt.Errorf("定价服务未初始化")
	}

	if costCalculator == nil {
		return nil, fmt.Errorf("成本计算器未初始化")
	}

	// 1. 验证配置
	validationResult, err := s.validator.ValidateDeploymentConfig(config)
	if err != nil {
		return nil, fmt.Errorf("验证配置失败: %w", err)
	}
	if !validationResult.Valid {
		return nil, fmt.Errorf("配置验证失败: %v", validationResult.Errors)
	}

	// 2. 获取模板路径
	templatePath, err := GetTemplatePath(config.TemplateName)
	if err != nil {
		return nil, fmt.Errorf("获取模板路径失败: %w", err)
	}

	// 3. 准备变量映射（合并所有配置参数）
	variables := make(map[string]string)

	// 添加核心配置参数
	variables["cloud_provider"] = config.Provider
	variables["region"] = config.Region
	variables["instance_type"] = config.InstanceType

	if config.Userdata != "" {
		variables["userdata"] = config.Userdata
	}

	// 添加其他自定义变量
	for k, v := range config.Variables {
		variables[k] = v
	}

	// 4. 解析模板获取资源规格
	resources, err := cost.ParseTemplate(templatePath, variables)
	if err != nil {
		return nil, fmt.Errorf("解析模板失败: %w", err)
	}

	// 5. 设置资源的 provider 和 region（如果模板解析没有设置）
	if resources.Provider == "" {
		resources.Provider = config.Provider
	}
	if resources.Region == "" {
		resources.Region = config.Region
	}

	// 确保每个资源都有 provider 和 region 信息
	for i := range resources.Resources {
		if resources.Resources[i].Provider == "" {
			resources.Resources[i].Provider = config.Provider
		}
		if resources.Resources[i].Region == "" {
			resources.Resources[i].Region = config.Region
		}
	}

	// 6. 使用 CostCalculator 计算成本
	estimate, err := costCalculator.CalculateCost(resources, pricingService)
	if err != nil {
		return nil, fmt.Errorf("计算成本失败: %w", err)
	}

	// 7. 转换为自定义部署的成本估算格式
	result := &CostEstimate{
		MonthlyCost: estimate.TotalMonthlyCost,
		Currency:    estimate.Currency,
		Details:     make(map[string]float64),
	}

	// 添加成本明细
	for _, breakdown := range estimate.Breakdown {
		if breakdown.Available && breakdown.TotalMonthly > 0 {
			key := fmt.Sprintf("%s (%s)", breakdown.ResourceName, breakdown.ResourceType)
			result.Details[key] = breakdown.TotalMonthly
		}
	}

	return result, nil
}

// NewCustomDeploymentService 创建自定义部署服务实例
func NewCustomDeploymentService() *CustomDeploymentService {
	return &CustomDeploymentService{
		templateMgr: NewTemplateManager(),
		validator:   NewConfigValidator(),
		executor:    NewDeploymentExecutor(),
		configStore: NewConfigStore(),
	}
}

// BaseTemplate 基础模板结构
type BaseTemplate struct {
	Name        string             `json:"name"`
	Description string             `json:"description"`
	Version     string             `json:"version"`
	Variables   []TemplateVariable `json:"variables"`
	Provider    string             `json:"provider"`  // 单一云厂商标识（新模板使用）
	Providers   []string           `json:"providers"` // 支持的云厂商列表（向后兼容）
	IsBase      bool               `json:"is_base"`
	User        string             `json:"user"`
	RedcModule  string             `json:"redc_module,omitempty"`
}

// TemplateVariable 模板变量定义
type TemplateVariable struct {
	Name         string              `json:"name"`
	Type         string              `json:"type"`
	Description  string              `json:"description"`
	Required     bool                `json:"required"`
	DefaultValue string              `json:"default_value,omitempty"`
	Validation   *VariableValidation `json:"validation,omitempty"`
}

// VariableValidation 变量验证规则
type VariableValidation struct {
	Pattern       string   `json:"pattern,omitempty"`
	MinLength     int      `json:"min_length,omitempty"`
	MaxLength     int      `json:"max_length,omitempty"`
	AllowedValues []string `json:"allowed_values,omitempty"`
}

// DeploymentConfig 部署配置
type DeploymentConfig struct {
	Name         string            `json:"name"`
	TemplateName string            `json:"template_name"`
	Provider     string            `json:"provider"`
	Region       string            `json:"region"`
	InstanceType string            `json:"instance_type"`
	Userdata     string            `json:"userdata,omitempty"`
	IsSpotInstance bool            `json:"is_spot_instance"`
	Variables    map[string]string `json:"variables"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
}

// ValidationResult 验证结果
type ValidationResult struct {
	Valid    bool                `json:"valid"`
	Errors   []ValidationError   `json:"errors,omitempty"`
	Warnings []ValidationWarning `json:"warnings,omitempty"`
}

// ValidationError 验证错误
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Code    string `json:"code"`
}
// Error implements the error interface for ValidationError
func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s (code: %s)", e.Field, e.Message, e.Code)
}


// ValidationWarning 验证警告
type ValidationWarning struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// 错误代码常量
const (
	ErrCodeRequired      = "REQUIRED"
	ErrCodeInvalidFormat = "INVALID_FORMAT"
	ErrCodeInvalidValue  = "INVALID_VALUE"
	ErrCodeNotSupported  = "NOT_SUPPORTED"
	ErrCodeNotAvailable  = "NOT_AVAILABLE"
)

// Region 地域信息
type Region struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

// InstanceType 实例规格信息
type InstanceType struct {
	Code        string  `json:"code"`
	Name        string  `json:"name"`
	CPU         int     `json:"cpu"`
	Memory      int     `json:"memory"`
	Description string  `json:"description"`
	Price       float64 `json:"price,omitempty"`
}

// CostEstimate 成本估算
type CostEstimate struct {
	MonthlyCost float64            `json:"monthly_cost"`
	Currency    string             `json:"currency"`
	Details     map[string]float64 `json:"details"`
}

// CustomDeployment 自定义部署记录
type CustomDeployment struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	TemplateName string                 `json:"template_name"`
	Config       *DeploymentConfig      `json:"config"`
	State        string                 `json:"state"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
	Outputs      map[string]interface{} `json:"outputs,omitempty"`
	ProjectID    string                 `json:"-"` // 不序列化到 JSON，仅用于数据库操作
}
// GetProviderRegions 获取云厂商支持的地域
func (s *CustomDeploymentService) GetProviderRegions(provider string) ([]Region, error) {
	return GetProviderRegions(provider)
}

// GetInstanceTypes 获取实例规格列表
// 集成云厂商 SDK 获取实例规格列表，并实现本地缓存机制
func (s *CustomDeploymentService) GetInstanceTypes(provider, region string) ([]InstanceType, error) {
	return GetInstanceTypes(provider, region)
}

// CreateCustomDeployment 创建自定义部署
// 1. 复制模板到项目目录
// 2. 生成变量文件
// 3. 调用 Terraform init 和 plan
// 4. 创建部署记录
// 5. 保存部署记录到数据库
func (s *CustomDeploymentService) CreateCustomDeployment(config *DeploymentConfig, projectPath string, projectID string) (*CustomDeployment, error) {
	if config == nil {
		return nil, fmt.Errorf("部署配置不能为空")
	}
	
	if projectPath == "" {
		return nil, fmt.Errorf("项目路径不能为空")
	}

	if projectID == "" {
		return nil, fmt.Errorf("项目 ID 不能为空")
	}

	// 1. 验证配置
	validationResult, err := s.validator.ValidateDeploymentConfig(config)
	if err != nil {
		return nil, fmt.Errorf("验证配置失败: %w", err)
	}
	if !validationResult.Valid {
		return nil, fmt.Errorf("配置验证失败: %v", validationResult.Errors)
	}

	// 2. 为需要密码的云厂商自动生成密码（如果用户未提供）
	if config.Variables == nil {
		config.Variables = make(map[string]string)
	}
	if config.Variables["instance_password"] == "" {
		// 自动生成符合云厂商要求的密码
		generatedPassword := generateInstancePassword()
		config.Variables["instance_password"] = generatedPassword
		gologger.Info().Msgf("已自动生成实例密码")
	}

	// 为阿里云补充系统盘类型默认值（如果缺失）
	if config.Provider == "alicloud" && config.Variables["system_disk_category"] == "" {
		config.Variables["system_disk_category"] = "cloud_efficiency"
		gologger.Info().Msgf("已自动设置阿里云系统盘类型为 cloud_efficiency")
	}

	// 3. 生成部署 ID
	deploymentID := GenerateCaseID()
	
	// 4. 获取模板路径
	templatePath, err := GetTemplatePath(config.TemplateName)
	if err != nil {
		return nil, fmt.Errorf("获取模板路径失败: %w", err)
	}

	// 5. 创建部署目录
	deploymentPath := filepath.Join(projectPath, deploymentID)
	if err := os.MkdirAll(deploymentPath, 0755); err != nil {
		return nil, fmt.Errorf("创建部署目录失败: %w", err)
	}

	// 6. 复制模板到部署目录
	if err := copyTemplate(templatePath, deploymentPath); err != nil {
		// 清理失败的部署目录
		os.RemoveAll(deploymentPath)
		return nil, fmt.Errorf("复制模板失败: %w", err)
	}

	// 7. 生成并写入 provider 配置文件
	providerContent, err := s.executor.GenerateProviderBlock(config.Provider, config.Region)
	if err != nil {
		// 清理失败的部署目录
		os.RemoveAll(deploymentPath)
		return nil, fmt.Errorf("生成 provider 配置失败: %w", err)
	}
	
	providerPath := filepath.Join(deploymentPath, "provider.tf")
	if err := os.WriteFile(providerPath, []byte(providerContent), 0644); err != nil {
		// 清理失败的部署目录
		os.RemoveAll(deploymentPath)
		return nil, fmt.Errorf("写入 provider 配置失败: %w", err)
	}

	// 8. 生成变量文件
	tfvarsContent, err := s.executor.GenerateVariablesFile(config)
	if err != nil {
		// 清理失败的部署目录
		os.RemoveAll(deploymentPath)
		return nil, fmt.Errorf("生成变量文件失败: %w", err)
	}

	// 9. 写入变量文件
	tfvarsPath := filepath.Join(deploymentPath, "terraform.tfvars")
	if err := os.WriteFile(tfvarsPath, []byte(tfvarsContent), 0644); err != nil {
		// 清理失败的部署目录
		os.RemoveAll(deploymentPath)
		return nil, fmt.Errorf("写入变量文件失败: %w", err)
	}

	// 10. 调用 Terraform init
	if err := TfInit2(deploymentPath); err != nil {
		// 清理失败的部署目录
		os.RemoveAll(deploymentPath)
		return nil, fmt.Errorf("Terraform 初始化失败: %w", err)
	}

	// 11. 调用 Terraform plan
	if err := TfPlan(deploymentPath); err != nil {
		// 清理失败的部署目录
		os.RemoveAll(deploymentPath)
		return nil, fmt.Errorf("Terraform plan 失败: %w", err)
	}

	// 12. 创建部署记录
	now := time.Now()
	deployment := &CustomDeployment{
		ID:           deploymentID,
		Name:         config.Name,
		TemplateName: config.TemplateName,
		Config:       config,
		State:        StatePending,
		CreatedAt:    now,
		UpdatedAt:    now,
		Outputs:      make(map[string]interface{}),
		ProjectID:    projectID,
	}

	// 13. 保存部署记录到数据库
	if err := deployment.DBSave(); err != nil {
		// 清理失败的部署目录
		os.RemoveAll(deploymentPath)
		return nil, fmt.Errorf("保存部署记录失败: %w", err)
	}

	return deployment, nil
}
// ListCustomDeployments 列出自定义部署
// 从数据库查询所有自定义部署
func (s *CustomDeploymentService) ListCustomDeployments(projectID string) ([]*CustomDeployment, error) {
	if projectID == "" {
		return nil, fmt.Errorf("项目 ID 不能为空")
	}

	deployments, err := LoadProjectCustomDeployments(projectID)
	if err != nil {
		return nil, fmt.Errorf("加载部署列表失败: %w", err)
	}

	return deployments, nil
}

// StartCustomDeployment 启动自定义部署
// 执行 Terraform apply 并更新部署状态
func (s *CustomDeploymentService) StartCustomDeployment(projectID, deploymentID, projectPath string) error {
	if projectID == "" {
		return fmt.Errorf("项目 ID 不能为空")
	}
	if deploymentID == "" {
		return fmt.Errorf("部署 ID 不能为空")
	}
	if projectPath == "" {
		return fmt.Errorf("项目路径不能为空")
	}

	// 加载部署记录
	deployment, err := LoadCustomDeployment(projectID, deploymentID)
	if err != nil {
		return fmt.Errorf("加载部署记录失败: %w", err)
	}

	// 更新状态为启动中
	deployment.State = StateStarting
	deployment.UpdatedAt = time.Now()
	if err := deployment.DBSave(); err != nil {
		return fmt.Errorf("更新部署状态失败: %w", err)
	}

	// 获取部署目录
	deploymentPath := filepath.Join(projectPath, deploymentID)

	// 删除旧的 plan 文件（如果存在），避免 "Saved plan is stale" 错误
	planFile := filepath.Join(deploymentPath, RedcPlanPath)
	if _, err := os.Stat(planFile); err == nil {
		if err := os.Remove(planFile); err != nil {
			gologger.Warning().Msgf("删除旧的 plan 文件失败: %v", err)
		}
	}

	// 重新生成 terraform.tfvars 文件，确保包含所有最新的变量（如 instance_name）
	if deployment.Config != nil {
		// 为需要密码的云厂商自动生成密码（如果用户未提供）
		if deployment.Config.Variables == nil {
			deployment.Config.Variables = make(map[string]string)
		}
		if deployment.Config.Variables["instance_password"] == "" {
			// 自动生成符合云厂商要求的密码
			generatedPassword := generateInstancePassword()
			deployment.Config.Variables["instance_password"] = generatedPassword
			gologger.Info().Msgf("已自动生成实例密码")
			// 更新数据库中的配置
			deployment.UpdatedAt = time.Now()
			if err := deployment.DBSave(); err != nil {
				gologger.Warning().Msgf("保存密码到数据库失败: %v", err)
			}
		}

		// 为阿里云补充系统盘类型默认值（如果缺失）
		if deployment.Config.Provider == "alicloud" && deployment.Config.Variables["system_disk_category"] == "" {
			deployment.Config.Variables["system_disk_category"] = "cloud_efficiency"
			gologger.Info().Msgf("已自动设置阿里云系统盘类型为 cloud_efficiency")
			deployment.UpdatedAt = time.Now()
			if err := deployment.DBSave(); err != nil {
				gologger.Warning().Msgf("保存系统盘类型到数据库失败: %v", err)
			}
		}

		tfvarsContent, err := s.executor.GenerateVariablesFile(deployment.Config)
		if err != nil {
			gologger.Warning().Msgf("重新生成变量文件失败: %v", err)
		} else {
			tfvarsPath := filepath.Join(deploymentPath, "terraform.tfvars")
			if err := os.WriteFile(tfvarsPath, []byte(tfvarsContent), 0644); err != nil {
				gologger.Warning().Msgf("写入变量文件失败: %v", err)
			} else {
				gologger.Info().Msgf("已更新 terraform.tfvars 文件")
			}
		}
	}

	// 执行 Terraform apply
	if err := TfApply(deploymentPath); err != nil {
		// 将错误信息保存到 outputs 中以便前端显示
		errorMsg := err.Error()
		deployment.Outputs = map[string]interface{}{
			"error_message": errorMsg,
		}
		
		// 自动清理已创建的资源（避免 EIP 等资源持续扣费）
		gologger.Warning().Msgf("部署失败，自动清理已创建的资源...")
		if destroyErr := TfDestroy(deploymentPath, nil); destroyErr != nil {
			gologger.Error().Msgf("自动清理资源失败: %v，用户可能需要手动清理", destroyErr)
		} else {
			gologger.Info().Msgf("已自动清理部署失败时创建的资源")
		}
		
		// 更新状态为错误
		deployment.State = StateError
		deployment.UpdatedAt = time.Now()
		deployment.DBSave() // 忽略保存错误
		return fmt.Errorf("Terraform apply 失败: %w", err)
	}

	// 读取 Terraform outputs
	outputs, err := TfOutput(deploymentPath)
	if err != nil {
		gologger.Warning().Msgf("读取 Terraform outputs 失败: %v", err)
		// 不中断流程，继续更新状态
	} else {
		// 将 outputs 转换为 map[string]interface{}
		outputMap := make(map[string]interface{})
		for key, output := range outputs {
			fmt.Printf("[DEBUG StartCustomDeployment] Output key=%s, Sensitive=%v, Type=%v, Value=%v\n", 
				key, output.Sensitive, output.Type, output.Value)
			
			// 需要 unmarshal Value (它是 json.RawMessage)
			var value interface{}
			if err := json.Unmarshal(output.Value, &value); err != nil {
				gologger.Warning().Msgf("解析 output %s 失败: %v", key, err)
				continue
			}
			outputMap[key] = value
		}
		deployment.Outputs = outputMap
		gologger.Info().Msgf("已读取 %d 个 Terraform outputs", len(outputMap))
		fmt.Printf("[DEBUG StartCustomDeployment] 最终 outputMap: %+v\n", outputMap)
	}

	// 更新状态为运行中
	deployment.State = StateRunning
	deployment.UpdatedAt = time.Now()
	if err := deployment.DBSave(); err != nil {
		return fmt.Errorf("更新部署状态失败: %w", err)
	}

	return nil
}

// StopCustomDeployment 停止自定义部署
// 执行 Terraform destroy 并更新部署状态
func (s *CustomDeploymentService) StopCustomDeployment(projectID, deploymentID, projectPath string) error {
	if projectID == "" {
		return fmt.Errorf("项目 ID 不能为空")
	}
	if deploymentID == "" {
		return fmt.Errorf("部署 ID 不能为空")
	}
	if projectPath == "" {
		return fmt.Errorf("项目路径不能为空")
	}

	// 加载部署记录
	deployment, err := LoadCustomDeployment(projectID, deploymentID)
	if err != nil {
		return fmt.Errorf("加载部署记录失败: %w", err)
	}

	// 更新状态为停止中
	deployment.State = StateStopping
	deployment.UpdatedAt = time.Now()
	if err := deployment.DBSave(); err != nil {
		return fmt.Errorf("更新部署状态失败: %w", err)
	}

	// 获取部署目录
	deploymentPath := filepath.Join(projectPath, deploymentID)

	// 执行 Terraform destroy
	if err := TfDestroy(deploymentPath, nil); err != nil {
		// 更新状态为错误
		deployment.State = StateError
		deployment.UpdatedAt = time.Now()
		deployment.DBSave() // 忽略保存错误
		return fmt.Errorf("Terraform destroy 失败: %w", err)
	}

	// 更新状态为已停止
	deployment.State = StateStopped
	deployment.UpdatedAt = time.Now()
	if err := deployment.DBSave(); err != nil {
		return fmt.Errorf("更新部署状态失败: %w", err)
	}

	return nil
}

// DeleteCustomDeployment 删除自定义部署
// 执行 Terraform destroy（如果需要）并删除部署记录和文件
func (s *CustomDeploymentService) DeleteCustomDeployment(projectID, deploymentID, projectPath string) error {
	if projectID == "" {
		return fmt.Errorf("项目 ID 不能为空")
	}
	if deploymentID == "" {
		return fmt.Errorf("部署 ID 不能为空")
	}
	if projectPath == "" {
		return fmt.Errorf("项目路径不能为空")
	}

	// 加载部署记录
	deployment, err := LoadCustomDeployment(projectID, deploymentID)
	if err != nil {
		return fmt.Errorf("加载部署记录失败: %w", err)
	}

	// 更新状态为删除中
	deployment.State = StateRemoving
	deployment.UpdatedAt = time.Now()
	if err := deployment.DBSave(); err != nil {
		return fmt.Errorf("更新部署状态失败: %w", err)
	}

	// 获取部署目录
	deploymentPath := filepath.Join(projectPath, deploymentID)

	// 如果部署正在运行，先执行 destroy
	if deployment.State == StateRunning || deployment.State == StateStarting {
		if err := TfDestroy(deploymentPath, nil); err != nil {
			// Destroy 失败，更新状态为错误
			deployment.State = StateError
			deployment.UpdatedAt = time.Now()
			deployment.DBSave() // 忽略保存错误
			return fmt.Errorf("Terraform destroy 失败: %w", err)
		}
	}

	// 删除部署目录
	if err := os.RemoveAll(deploymentPath); err != nil {
		return fmt.Errorf("删除部署目录失败: %w", err)
	}

	// 从数据库删除部署记录
	if err := deployment.DBRemove(); err != nil {
		return fmt.Errorf("删除部署记录失败: %w", err)
	}

	return nil
}
// DeploymentChangeHistory 部署变更历史记录
type DeploymentChangeHistory struct {
	ID           string                 `json:"id"`
	DeploymentID string                 `json:"deployment_id"`
	ChangeType   string                 `json:"change_type"` // config_update, state_change, etc.
	OldValue     map[string]interface{} `json:"old_value,omitempty"`
	NewValue     map[string]interface{} `json:"new_value,omitempty"`
	Operator     string                 `json:"operator,omitempty"`
	Timestamp    time.Time              `json:"timestamp"`
	Description  string                 `json:"description,omitempty"`
	ProjectID    string                 `json:"-"`
}

// 变更类型常量
const (
	ChangeTypeConfigUpdate = "config_update"
	ChangeTypeStateChange  = "state_change"
	ChangeTypeCreate       = "create"
	ChangeTypeDelete       = "delete"
)

// RecordConfigChange 记录配置变更
func (s *CustomDeploymentService) RecordConfigChange(projectID, deploymentID string, oldConfig, newConfig *DeploymentConfig, operator string) error {
	if projectID == "" {
		return fmt.Errorf("项目 ID 不能为空")
	}
	if deploymentID == "" {
		return fmt.Errorf("部署 ID 不能为空")
	}

	// 创建变更记录
	change := &DeploymentChangeHistory{
		ID:           GenerateCaseID(),
		DeploymentID: deploymentID,
		ChangeType:   ChangeTypeConfigUpdate,
		OldValue:     configToMap(oldConfig),
		NewValue:     configToMap(newConfig),
		Operator:     operator,
		Timestamp:    time.Now(),
		Description:  "配置已更新",
		ProjectID:    projectID,
	}

	return change.DBSave()
}

// RecordStateChange 记录状态变更
func (s *CustomDeploymentService) RecordStateChange(projectID, deploymentID, oldState, newState, operator string) error {
	if projectID == "" {
		return fmt.Errorf("项目 ID 不能为空")
	}
	if deploymentID == "" {
		return fmt.Errorf("部署 ID 不能为空")
	}

	// 创建变更记录
	change := &DeploymentChangeHistory{
		ID:           GenerateCaseID(),
		DeploymentID: deploymentID,
		ChangeType:   ChangeTypeStateChange,
		OldValue:     map[string]interface{}{"state": oldState},
		NewValue:     map[string]interface{}{"state": newState},
		Operator:     operator,
		Timestamp:    time.Now(),
		Description:  fmt.Sprintf("状态从 %s 变更为 %s", oldState, newState),
		ProjectID:    projectID,
	}

	return change.DBSave()
}

// GetDeploymentHistory 获取部署的变更历史
func (s *CustomDeploymentService) GetDeploymentHistory(projectID, deploymentID string) ([]*DeploymentChangeHistory, error) {
	if projectID == "" {
		return nil, fmt.Errorf("项目 ID 不能为空")
	}
	if deploymentID == "" {
		return nil, fmt.Errorf("部署 ID 不能为空")
	}

	return LoadDeploymentHistory(projectID, deploymentID)
}

// configToMap 将 DeploymentConfig 转换为 map
func configToMap(config *DeploymentConfig) map[string]interface{} {
	if config == nil {
		return nil
	}

	return map[string]interface{}{
		"name":          config.Name,
		"template_name": config.TemplateName,
		"provider":      config.Provider,
		"region":        config.Region,
		"instance_type": config.InstanceType,
		"userdata":      config.Userdata,
		"variables":     config.Variables,
	}
}
// BatchOperationResult 批量操作结果
type BatchOperationResult struct {
	DeploymentID string `json:"deployment_id"`
	Success      bool   `json:"success"`
	Error        string `json:"error,omitempty"`
}

// BatchStartDeployments 批量启动部署
// 使用 goroutine 并发执行操作
func (s *CustomDeploymentService) BatchStartDeployments(projectID string, deploymentIDs []string, projectPath string) []BatchOperationResult {
	if len(deploymentIDs) == 0 {
		return []BatchOperationResult{}
	}

	results := make([]BatchOperationResult, len(deploymentIDs))
	var wg sync.WaitGroup
	var mu sync.Mutex

	for i, deploymentID := range deploymentIDs {
		wg.Add(1)
		go func(index int, id string) {
			defer wg.Done()

			result := BatchOperationResult{
				DeploymentID: id,
				Success:      true,
			}

			err := s.StartCustomDeployment(projectID, id, projectPath)
			if err != nil {
				result.Success = false
				result.Error = err.Error()
			}

			mu.Lock()
			results[index] = result
			mu.Unlock()
		}(i, deploymentID)
	}

	wg.Wait()
	return results
}

// BatchStopDeployments 批量停止部署
// 使用 goroutine 并发执行操作
func (s *CustomDeploymentService) BatchStopDeployments(projectID string, deploymentIDs []string, projectPath string) []BatchOperationResult {
	if len(deploymentIDs) == 0 {
		return []BatchOperationResult{}
	}

	results := make([]BatchOperationResult, len(deploymentIDs))
	var wg sync.WaitGroup
	var mu sync.Mutex

	for i, deploymentID := range deploymentIDs {
		wg.Add(1)
		go func(index int, id string) {
			defer wg.Done()

			result := BatchOperationResult{
				DeploymentID: id,
				Success:      true,
			}

			err := s.StopCustomDeployment(projectID, id, projectPath)
			if err != nil {
				result.Success = false
				result.Error = err.Error()
			}

			mu.Lock()
			results[index] = result
			mu.Unlock()
		}(i, deploymentID)
	}

	wg.Wait()
	return results
}

// BatchDeleteDeployments 批量删除部署
// 使用 goroutine 并发执行操作
func (s *CustomDeploymentService) BatchDeleteDeployments(projectID string, deploymentIDs []string, projectPath string) []BatchOperationResult {
	if len(deploymentIDs) == 0 {
		return []BatchOperationResult{}
	}

	results := make([]BatchOperationResult, len(deploymentIDs))
	var wg sync.WaitGroup
	var mu sync.Mutex

	for i, deploymentID := range deploymentIDs {
		wg.Add(1)
		go func(index int, id string) {
			defer wg.Done()

			result := BatchOperationResult{
				DeploymentID: id,
				Success:      true,
			}

			err := s.DeleteCustomDeployment(projectID, id, projectPath)
			if err != nil {
				result.Success = false
				result.Error = err.Error()
			}

			mu.Lock()
			results[index] = result
			mu.Unlock()
		}(i, deploymentID)
	}

	wg.Wait()
	return results
}

// GetProviderRegions 从 regions.json 加载地域数据
func GetProviderRegions(provider string) ([]Region, error) {
	regions, err := loadRegionsData()
	if err != nil {
		return nil, err
	}

	// 如果指定了云厂商，返回该云厂商的地域列表
	if provider != "" {
		if providerRegions, ok := regions[provider]; ok {
			return providerRegions, nil
		}
		return nil, &ValidationError{
			Field:   "provider",
			Message: "不支持的云厂商: " + provider,
			Code:    ErrCodeNotSupported,
		}
	}

	// 如果没有指定云厂商，返回所有地域
	allRegions := []Region{}
	for _, regionList := range regions {
		allRegions = append(allRegions, regionList...)
	}
	return allRegions, nil
}

// loadRegionsData 加载地域数据
func loadRegionsData() (map[string][]Region, error) {
	// Try multiple possible paths for the regions.json file
	possiblePaths := []string{
		"mod/providers/regions.json",
		"providers/regions.json",
		"../mod/providers/regions.json",
	}

	var data []byte
	var err error
	var foundPath string

	for _, path := range possiblePaths {
		data, err = os.ReadFile(path)
		if err == nil {
			foundPath = path
			break
		}
	}

	if foundPath == "" {
		return nil, fmt.Errorf("读取地域配置文件失败: 未找到 regions.json 文件")
	}

	var regions map[string][]Region
	if err := json.Unmarshal(data, &regions); err != nil {
		return nil, fmt.Errorf("解析地域配置文件失败: %w", err)
	}

	return regions, nil
}

// GetInstanceTypes 获取实例规格列表（带缓存）
func GetInstanceTypes(provider, region string) ([]InstanceType, error) {
	// 1. 验证参数
	if provider == "" {
		return nil, &ValidationError{
			Field:   "provider",
			Message: "云厂商不能为空",
			Code:    ErrCodeRequired,
		}
	}
	if region == "" {
		return nil, &ValidationError{
			Field:   "region",
			Message: "地域不能为空",
			Code:    ErrCodeRequired,
		}
	}

	// 2. 验证地域是否对该云厂商有效
	regions, err := GetProviderRegions(provider)
	if err != nil {
		return nil, err
	}
	
	regionValid := false
	for _, r := range regions {
		if r.Code == region {
			regionValid = true
			break
		}
	}
	
	if !regionValid {
		return nil, &ValidationError{
			Field:   "region",
			Message: fmt.Sprintf("地域 '%s' 在云厂商 '%s' 中不可用", region, provider),
			Code:    ErrCodeNotAvailable,
		}
	}

	// 3. 从云厂商 API 获取实例规格
	types, err := fetchInstanceTypesFromProvider(provider, region)
	if err != nil {
		return nil, err
	}

	return types, nil
}

// fetchInstanceTypesFromProvider 从云厂商 API 获取实例规格
// fetchInstanceTypesFromProvider 从云厂商 API 获取实例规格
func fetchInstanceTypesFromProvider(provider, region string) ([]InstanceType, error) {
	// 首先尝试从真实 API 获取（如果凭证已配置）
	types, err := fetchInstanceTypesFromProviderAPI(provider, region)
	if err == nil && len(types) > 0 {
		return types, nil
	}
	
	// 如果 API 调用失败，使用静态数据作为后备
	switch provider {
	case "alicloud":
		return fetchAlicloudInstanceTypesStatic(region)
	case "tencentcloud":
		return fetchTencentcloudInstanceTypesStatic(region)
	case "aws":
		return fetchAWSInstanceTypesStatic(region)
	case "volcengine":
		return fetchVolcengineInstanceTypesStatic(region)
	case "huaweicloud":
		return fetchHuaweicloudInstanceTypesStatic(region)
	default:
		return nil, &ValidationError{
			Field:   "provider",
			Message: fmt.Sprintf("不支持的云厂商: %s", provider),
			Code:    ErrCodeNotSupported,
		}
	}
}

// fetchAlicloudInstanceTypesStatic 获取阿里云实例规格（静态数据）
func fetchAlicloudInstanceTypesStatic(region string) ([]InstanceType, error) {
	// 返回常用的阿里云实例规格列表
	// 在实际生产环境中，这里应该调用阿里云 SDK 的 DescribeInstanceTypes API
	// 由于需要 AccessKey 和 SecretKey，这里先返回静态数据
	types := []InstanceType{
		{Code: "ecs.t6-c1m1.large", Name: "突发性能实例 t6", CPU: 2, Memory: 2048, Description: "2核2GB"},
		{Code: "ecs.t6-c1m2.large", Name: "突发性能实例 t6", CPU: 2, Memory: 4096, Description: "2核4GB"},
		{Code: "ecs.c7.large", Name: "计算型 c7", CPU: 2, Memory: 4096, Description: "2核4GB"},
		{Code: "ecs.c7.xlarge", Name: "计算型 c7", CPU: 4, Memory: 8192, Description: "4核8GB"},
		{Code: "ecs.c7.2xlarge", Name: "计算型 c7", CPU: 8, Memory: 16384, Description: "8核16GB"},
		{Code: "ecs.g7.large", Name: "通用型 g7", CPU: 2, Memory: 8192, Description: "2核8GB"},
		{Code: "ecs.g7.xlarge", Name: "通用型 g7", CPU: 4, Memory: 16384, Description: "4核16GB"},
		{Code: "ecs.g7.2xlarge", Name: "通用型 g7", CPU: 8, Memory: 32768, Description: "8核32GB"},
		{Code: "ecs.r7.large", Name: "内存型 r7", CPU: 2, Memory: 16384, Description: "2核16GB"},
		{Code: "ecs.r7.xlarge", Name: "内存型 r7", CPU: 4, Memory: 32768, Description: "4核32GB"},
	}

	return types, nil
}

// fetchTencentcloudInstanceTypesStatic 获取腾讯云实例规格（静态数据）
func fetchTencentcloudInstanceTypesStatic(region string) ([]InstanceType, error) {
	// 返回常用的腾讯云实例规格列表
	// 在实际生产环境中，这里应该调用腾讯云 SDK 的 DescribeInstanceTypeConfigs API
	types := []InstanceType{
		{Code: "S6.MEDIUM2", Name: "标准型 S6", CPU: 2, Memory: 2048, Description: "2核2GB"},
		{Code: "S6.MEDIUM4", Name: "标准型 S6", CPU: 2, Memory: 4096, Description: "2核4GB"},
		{Code: "S6.LARGE8", Name: "标准型 S6", CPU: 4, Memory: 8192, Description: "4核8GB"},
		{Code: "S6.2XLARGE16", Name: "标准型 S6", CPU: 8, Memory: 16384, Description: "8核16GB"},
		{Code: "C6.LARGE8", Name: "计算型 C6", CPU: 4, Memory: 8192, Description: "4核8GB"},
		{Code: "C6.2XLARGE16", Name: "计算型 C6", CPU: 8, Memory: 16384, Description: "8核16GB"},
		{Code: "M6.MEDIUM8", Name: "内存型 M6", CPU: 2, Memory: 8192, Description: "2核8GB"},
		{Code: "M6.LARGE16", Name: "内存型 M6", CPU: 4, Memory: 16384, Description: "4核16GB"},
	}

	return types, nil
}

// fetchAWSInstanceTypesStatic 获取 AWS 实例规格（静态数据）
func fetchAWSInstanceTypesStatic(region string) ([]InstanceType, error) {
	// 返回常用的 AWS 实例规格列表
	types := []InstanceType{
		{Code: "t3.micro", Name: "T3 Micro", CPU: 2, Memory: 1024, Description: "2核1GB"},
		{Code: "t3.small", Name: "T3 Small", CPU: 2, Memory: 2048, Description: "2核2GB"},
		{Code: "t3.medium", Name: "T3 Medium", CPU: 2, Memory: 4096, Description: "2核4GB"},
		{Code: "t3.large", Name: "T3 Large", CPU: 2, Memory: 8192, Description: "2核8GB"},
		{Code: "m5.large", Name: "M5 Large", CPU: 2, Memory: 8192, Description: "2核8GB"},
		{Code: "m5.xlarge", Name: "M5 XLarge", CPU: 4, Memory: 16384, Description: "4核16GB"},
		{Code: "c5.large", Name: "C5 Large", CPU: 2, Memory: 4096, Description: "2核4GB"},
		{Code: "c5.xlarge", Name: "C5 XLarge", CPU: 4, Memory: 8192, Description: "4核8GB"},
	}

	return types, nil
}

// fetchVolcengineInstanceTypesStatic 获取火山引擎实例规格（静态数据）
func fetchVolcengineInstanceTypesStatic(region string) ([]InstanceType, error) {
	// 火山引擎不同区域支持的实例规格不同
	// 这里维护常见区域的实例规格映射
	// 在实际生产环境中，应该调用火山引擎 SDK 的 DescribeInstanceTypes API
	
	regionInstanceTypes := map[string][]InstanceType{
		"cn-beijing": {
			{Code: "ecs.g3i.large", Name: "通用型 g3i", CPU: 2, Memory: 8192, Description: "2核8GB"},
			{Code: "ecs.g3i.xlarge", Name: "通用型 g3i", CPU: 4, Memory: 16384, Description: "4核16GB"},
			{Code: "ecs.g3i.2xlarge", Name: "通用型 g3i", CPU: 8, Memory: 32768, Description: "8核32GB"},
			{Code: "ecs.g2i.large", Name: "通用型 g2i", CPU: 2, Memory: 8192, Description: "2核8GB"},
			{Code: "ecs.g2i.xlarge", Name: "通用型 g2i", CPU: 4, Memory: 16384, Description: "4核16GB"},
			{Code: "ecs.c4i.large", Name: "计算型 c4i", CPU: 2, Memory: 4096, Description: "2核4GB"},
			{Code: "ecs.c4i.xlarge", Name: "计算型 c4i", CPU: 4, Memory: 8192, Description: "4核8GB"},
			{Code: "ecs.c4i.2xlarge", Name: "计算型 c4i", CPU: 8, Memory: 16384, Description: "8核16GB"},
			{Code: "ecs.c4i.4xlarge", Name: "计算型 c4i", CPU: 16, Memory: 32768, Description: "16核32GB"},
			{Code: "ecs.r4i.large", Name: "内存型 r4i", CPU: 2, Memory: 16384, Description: "2核16GB"},
			{Code: "ecs.r4i.xlarge", Name: "内存型 r4i", CPU: 4, Memory: 32768, Description: "4核32GB"},
		},
		"cn-shanghai": {
			{Code: "ecs.g3i.large", Name: "通用型 g3i", CPU: 2, Memory: 8192, Description: "2核8GB"},
			{Code: "ecs.g3i.xlarge", Name: "通用型 g3i", CPU: 4, Memory: 16384, Description: "4核16GB"},
			{Code: "ecs.g3i.2xlarge", Name: "通用型 g3i", CPU: 8, Memory: 32768, Description: "8核32GB"},
			{Code: "ecs.c3i.large", Name: "计算型 c3i", CPU: 2, Memory: 4096, Description: "2核4GB"},
			{Code: "ecs.c3i.xlarge", Name: "计算型 c3i", CPU: 4, Memory: 8192, Description: "4核8GB"},
		},
		"cn-guangzhou": {
			{Code: "ecs.g3i.large", Name: "通用型 g3i", CPU: 2, Memory: 8192, Description: "2核8GB"},
			{Code: "ecs.g3i.xlarge", Name: "通用型 g3i", CPU: 4, Memory: 16384, Description: "4核16GB"},
			{Code: "ecs.c3i.large", Name: "计算型 c3i", CPU: 2, Memory: 4096, Description: "2核4GB"},
		},
	}
	
	// 查找该区域的实例规格
	types, exists := regionInstanceTypes[region]
	if !exists {
		// 如果区域不在映射表中，返回通用的实例规格列表
		types = []InstanceType{
			{Code: "ecs.g3i.large", Name: "通用型 g3i", CPU: 2, Memory: 8192, Description: "2核8GB"},
			{Code: "ecs.g3i.xlarge", Name: "通用型 g3i", CPU: 4, Memory: 16384, Description: "4核16GB"},
		}
	}

	return types, nil
}

// fetchHuaweicloudInstanceTypesStatic 获取华为云实例规格（静态数据）
func fetchHuaweicloudInstanceTypesStatic(region string) ([]InstanceType, error) {
	// 返回常用的华为云实例规格列表
	types := []InstanceType{
		{Code: "s6.small.1", Name: "通用计算型 s6", CPU: 1, Memory: 1024, Description: "1核1GB"},
		{Code: "s6.medium.2", Name: "通用计算型 s6", CPU: 1, Memory: 2048, Description: "1核2GB"},
		{Code: "s6.large.2", Name: "通用计算型 s6", CPU: 2, Memory: 4096, Description: "2核4GB"},
		{Code: "s6.xlarge.2", Name: "通用计算型 s6", CPU: 4, Memory: 8192, Description: "4核8GB"},
		{Code: "c6.large.2", Name: "计算型 c6", CPU: 2, Memory: 4096, Description: "2核4GB"},
		{Code: "c6.xlarge.2", Name: "计算型 c6", CPU: 4, Memory: 8192, Description: "4核8GB"},
	}

	return types, nil
}


// copyTemplate 复制模板到目标目录
func copyTemplate(src, dst string) error {
	return utils.Dir(src, dst)
}

// ==========================================
// CustomDeployment 数据库操作
// ==========================================

// toProto 将 CustomDeployment 转换为 protobuf 消息
func (d *CustomDeployment) toProto() (*pb.CustomDeployment, error) {
	// 序列化 Config
	configBytes, err := json.Marshal(d.Config)
	if err != nil {
		return nil, fmt.Errorf("序列化配置失败: %w", err)
	}

	// 序列化 Outputs
	outputsBytes, err := json.Marshal(d.Outputs)
	if err != nil {
		return nil, fmt.Errorf("序列化输出失败: %w", err)
	}

	return &pb.CustomDeployment{
		Id:           d.ID,
		Name:         d.Name,
		TemplateName: d.TemplateName,
		ConfigJson:   string(configBytes),
		State:        d.State,
		CreatedAt:    d.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    d.UpdatedAt.Format(time.RFC3339),
		OutputsJson:  string(outputsBytes),
		ProjectId:    d.ProjectID,
	}, nil
}

// customDeploymentFromProto 将 protobuf 消息转换为 CustomDeployment
func customDeploymentFromProto(p *pb.CustomDeployment) (*CustomDeployment, error) {
	d := &CustomDeployment{
		ID:           p.Id,
		Name:         p.Name,
		TemplateName: p.TemplateName,
		State:        p.State,
		ProjectID:    p.ProjectId,
	}

	// 解析 Config
	if len(p.ConfigJson) > 0 {
		var config DeploymentConfig
		if err := json.Unmarshal([]byte(p.ConfigJson), &config); err != nil {
			return nil, fmt.Errorf("解析配置失败: %w", err)
		}
		d.Config = &config
	}

	// 解析 Outputs
	if len(p.OutputsJson) > 0 {
		var outputs map[string]interface{}
		if err := json.Unmarshal([]byte(p.OutputsJson), &outputs); err != nil {
			return nil, fmt.Errorf("解析输出失败: %w", err)
		}
		d.Outputs = outputs
		fmt.Printf("[DEBUG customDeploymentFromProto] 从数据库加载 Outputs: %+v\n", outputs)
	}

	// 解析时间
	if p.CreatedAt != "" {
		createdAt, err := time.Parse(time.RFC3339, p.CreatedAt)
		if err == nil {
			d.CreatedAt = createdAt
		}
	}
	if p.UpdatedAt != "" {
		updatedAt, err := time.Parse(time.RFC3339, p.UpdatedAt)
		if err == nil {
			d.UpdatedAt = updatedAt
		}
	}

	return d, nil
}

// DBSave 将自定义部署记录保存到数据库
func (d *CustomDeployment) DBSave() error {
	if d.ProjectID == "" {
		return fmt.Errorf("严重错误: CustomDeployment %s 丢失了 ProjectID，无法保存", d.ID)
	}

	return dbExec(func(tx *bolt.Tx) error {
		// 每个项目有独立的 Bucket，例如 "CustomDeployments_default"
		bucketName := []byte(fmt.Sprintf("CustomDeployments_%s", d.ProjectID))

		b, err := tx.CreateBucketIfNotExists(bucketName)
		if err != nil {
			return fmt.Errorf("创建 bucket 失败: %w", err)
		}

		// 序列化
		pbData, err := d.toProto()
		if err != nil {
			return err
		}

		data, err := proto.Marshal(pbData)
		if err != nil {
			return fmt.Errorf("序列化失败: %w", err)
		}

		// 写入 (Key=DeploymentID)
		return b.Put([]byte(d.ID), data)
	})
}

// DBRemove 从数据库中删除自定义部署记录
func (d *CustomDeployment) DBRemove() error {
	if d.ProjectID == "" {
		return fmt.Errorf("严重错误: CustomDeployment %s 丢失了 ProjectID，无法删除", d.ID)
	}

	return dbExec(func(tx *bolt.Tx) error {
		bucketName := []byte(fmt.Sprintf("CustomDeployments_%s", d.ProjectID))
		b := tx.Bucket(bucketName)
		if b == nil {
			return nil // 桶不存在，也就是数据本来就没有，视为成功
		}
		return b.Delete([]byte(d.ID))
	})
}

// LoadCustomDeployment 加载指定的自定义部署记录
func LoadCustomDeployment(projectName, deploymentID string) (*CustomDeployment, error) {
	var deployment *CustomDeployment

	err := dbExec(func(tx *bolt.Tx) error {
		bucketName := []byte(fmt.Sprintf("CustomDeployments_%s", projectName))
		b := tx.Bucket(bucketName)
		if b == nil {
			return fmt.Errorf("项目 %s 没有自定义部署记录", projectName)
		}

		data := b.Get([]byte(deploymentID))
		if data == nil {
			return fmt.Errorf("部署记录不存在: %s", deploymentID)
		}

		var p pb.CustomDeployment
		if err := proto.Unmarshal(data, &p); err != nil {
			return fmt.Errorf("反序列化失败: %w", err)
		}

		d, err := customDeploymentFromProto(&p)
		if err != nil {
			return err
		}

		deployment = d
		return nil
	})

	return deployment, err
}

// LoadProjectCustomDeployments 加载指定项目下的所有自定义部署记录
func LoadProjectCustomDeployments(projectName string) ([]*CustomDeployment, error) {
	var deployments []*CustomDeployment

	err := dbExec(func(tx *bolt.Tx) error {
		bucketName := []byte(fmt.Sprintf("CustomDeployments_%s", projectName))
		b := tx.Bucket(bucketName)
		if b == nil {
			return nil // 没数据，返回空切片
		}

		// 遍历桶内所有数据
		return b.ForEach(func(k, v []byte) error {
			var p pb.CustomDeployment
			// 反序列化 Proto
			if err := proto.Unmarshal(v, &p); err == nil {
				// 转为业务对象
				d, err := customDeploymentFromProto(&p)
				if err == nil {
					deployments = append(deployments, d)
				}
			}
			return nil
		})
	})

	// 按创建时间降序排序（最新的在前面）
	sort.Slice(deployments, func(i, j int) bool {
		return deployments[i].CreatedAt.After(deployments[j].CreatedAt)
	})

	return deployments, err
}
// ==========================================
// DeploymentChangeHistory 数据库操作
// ==========================================

// toProto 将 DeploymentChangeHistory 转换为 protobuf 消息
func (h *DeploymentChangeHistory) toProto() (*pb.DeploymentChangeHistory, error) {
	// 序列化 OldValue
	oldValueBytes, err := json.Marshal(h.OldValue)
	if err != nil {
		return nil, fmt.Errorf("序列化旧值失败: %w", err)
	}

	// 序列化 NewValue
	newValueBytes, err := json.Marshal(h.NewValue)
	if err != nil {
		return nil, fmt.Errorf("序列化新值失败: %w", err)
	}

	return &pb.DeploymentChangeHistory{
		Id:           h.ID,
		DeploymentId: h.DeploymentID,
		ChangeType:   h.ChangeType,
		OldValueJson: string(oldValueBytes),
		NewValueJson: string(newValueBytes),
		Operator:     h.Operator,
		Timestamp:    h.Timestamp.Format(time.RFC3339),
		Description:  h.Description,
		ProjectId:    h.ProjectID,
	}, nil
}

// deploymentChangeHistoryFromProto 将 protobuf 消息转换为 DeploymentChangeHistory
func deploymentChangeHistoryFromProto(p *pb.DeploymentChangeHistory) (*DeploymentChangeHistory, error) {
	h := &DeploymentChangeHistory{
		ID:           p.Id,
		DeploymentID: p.DeploymentId,
		ChangeType:   p.ChangeType,
		Operator:     p.Operator,
		Description:  p.Description,
		ProjectID:    p.ProjectId,
	}

	// 解析 OldValue
	if len(p.OldValueJson) > 0 {
		var oldValue map[string]interface{}
		if err := json.Unmarshal([]byte(p.OldValueJson), &oldValue); err != nil {
			return nil, fmt.Errorf("解析旧值失败: %w", err)
		}
		h.OldValue = oldValue
	}

	// 解析 NewValue
	if len(p.NewValueJson) > 0 {
		var newValue map[string]interface{}
		if err := json.Unmarshal([]byte(p.NewValueJson), &newValue); err != nil {
			return nil, fmt.Errorf("解析新值失败: %w", err)
		}
		h.NewValue = newValue
	}

	// 解析时间
	if p.Timestamp != "" {
		timestamp, err := time.Parse(time.RFC3339, p.Timestamp)
		if err == nil {
			h.Timestamp = timestamp
		}
	}

	return h, nil
}

// DBSave 将变更历史记录保存到数据库
func (h *DeploymentChangeHistory) DBSave() error {
	if h.ProjectID == "" {
		return fmt.Errorf("严重错误: DeploymentChangeHistory %s 丢失了 ProjectID，无法保存", h.ID)
	}

	return dbExec(func(tx *bolt.Tx) error {
		// 每个项目有独立的 Bucket，例如 "DeploymentHistory_default"
		bucketName := []byte(fmt.Sprintf("DeploymentHistory_%s", h.ProjectID))

		b, err := tx.CreateBucketIfNotExists(bucketName)
		if err != nil {
			return fmt.Errorf("创建 bucket 失败: %w", err)
		}

		// 序列化
		pbData, err := h.toProto()
		if err != nil {
			return err
		}

		data, err := proto.Marshal(pbData)
		if err != nil {
			return fmt.Errorf("序列化失败: %w", err)
		}

		// 写入 (Key=HistoryID)
		return b.Put([]byte(h.ID), data)
	})
}

// LoadDeploymentHistory 加载指定部署的变更历史
func LoadDeploymentHistory(projectName, deploymentID string) ([]*DeploymentChangeHistory, error) {
	var history []*DeploymentChangeHistory

	err := dbExec(func(tx *bolt.Tx) error {
		bucketName := []byte(fmt.Sprintf("DeploymentHistory_%s", projectName))
		b := tx.Bucket(bucketName)
		if b == nil {
			return nil // 没数据，返回空切片
		}

		// 遍历桶内所有数据
		return b.ForEach(func(k, v []byte) error {
			var p pb.DeploymentChangeHistory
			// 反序列化 Proto
			if err := proto.Unmarshal(v, &p); err == nil {
				// 只返回指定部署的历史记录
				if p.DeploymentId == deploymentID {
					// 转为业务对象
					h, err := deploymentChangeHistoryFromProto(&p)
					if err == nil {
						history = append(history, h)
					}
				}
			}
			return nil
		})
	})

	// 按时间降序排序（最新的在前面）
	sort.Slice(history, func(i, j int) bool {
		return history[i].Timestamp.After(history[j].Timestamp)
	})

	return history, err
}
