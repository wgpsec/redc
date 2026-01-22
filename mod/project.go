package mod

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"red-cloud/mod/gologger"
	"strings"
	"sync"
	"text/tabwriter"
	"time"

	"github.com/hashicorp/terraform-exec/tfexec"
)

// Global mutex map to protect file operations per project
var (
	projectMutexes = make(map[string]*sync.Mutex)
	projectMapMu   sync.Mutex
)

// getProjectMutex returns a mutex for the given project name
func getProjectMutex(projectName string) *sync.Mutex {
	projectMapMu.Lock()
	defer projectMapMu.Unlock()

	if _, exists := projectMutexes[projectName]; !exists {
		projectMutexes[projectName] = &sync.Mutex{}
	}
	return projectMutexes[projectName]
}

// RedcProject 项目结构体
type RedcProject struct {
	ProjectName string  `json:"project_name"`
	ProjectPath string  `json:"project_path"`
	CreateTime  string  `json:"create_time"`
	User        string  `json:"user"`
	Case        []*Case `json:"case"`
}

// Case 项目信息
type Case struct {
	// Id uuid
	Id           string    `json:"id"`
	Name         string    `json:"name"`
	Type         string    `json:"type"`
	Module       string    `json:"module,omitempty"`
	Operator     string    `json:"operator"`
	Path         string    `json:"path"`
	Node         int       `json:"node"`
	CreateTime   string    `json:"create_time"`
	StateTime    string    `json:"state_time"`
	Parameter    []string  `json:"parameter"`
	State        CaseState `json:"state"`
	output       map[string]tfexec.OutputMeta
	saveHandler  func() error
	removeHandle func() error
}

type ChangeCommand struct {
	IsRemove bool
	Pars     map[string]string
}

func GetProjectCase(projectId string, caseID string, userName string) (*Case, error) {
	pro, err := ProjectParse(projectId, userName) // 注意：这里可能需要处理 global U 或者从配置读取
	if err != nil {
		gologger.Fatal().Msgf("项目解析失败: %s", err)
	}
	c, err := pro.GetCase(caseID)
	if err != nil {
		return nil, fmt.Errorf("操作失败: 找不到 ID 为「%s」的场景\n错误: %s", caseID, err)

	}
	return c, nil
}

// NewProjectConfig 创建项目配置文件
func NewProjectConfig(name string, user string) (*RedcProject, error) {
	path := filepath.Join(ProjectPath, name)
	// 创建项目目录
	if err := os.MkdirAll(path, 0755); err != nil {
		return nil, fmt.Errorf("创建项目目录失败: %w", err)
	}
	gologger.Info().Msgf("项目目录「%s」创建成功！", name)
	// 创建项目状态文件
	currentTime := time.Now().Format("2006-01-02 15:04:05")
	project := &RedcProject{
		ProjectName: name,
		ProjectPath: path,
		CreateTime:  currentTime,
		User:        user,
	}

	if err := project.SaveProject(); err != nil {
		// 如果保存失败，应该清理目录吗？视业务逻辑而定，这里暂时只返回错误
		return nil, fmt.Errorf("保存项目状态文件失败: %w", err)
	}
	gologger.Info().Msgf("项目状态文件「%s」创建成功！", ProjectFile)
	return project, nil

}

func ProjectParse(name string, user string) (*RedcProject, error) {
	// 尝试直接读取项目
	if p, err := ProjectByName(name); err == nil {
		// 项目鉴权
		if p.User != user && user != "system" {
			return nil, fmt.Errorf("当前用户「%s」无权限访问项目「%s」", user, name)
		}
		// 读取成功，直接返回
		return p, nil
	}
	path := filepath.Join(ProjectPath, name)
	// 检查目录是否存在，或者直接尝试创建
	if _, statErr := os.Stat(path); os.IsNotExist(statErr) {
		gologger.Info().Msgf("项目不存在，正在创建新项目: %s", name)
		return NewProjectConfig(name, user)
	} else if statErr != nil {
		// 目录存在但有其他错误（如权限不足）
		return nil, statErr
	}
	return NewProjectConfig(name, user)
}

// ProjectByName 读取项目配置
func ProjectByName(name string) (*RedcProject, error) {
	// Get the project-specific mutex and lock it for reading
	mu := getProjectMutex(name)
	mu.Lock()
	defer mu.Unlock()

	path := filepath.Join(ProjectPath, name, ProjectFile)
	data, err := os.ReadFile(path)
	if err != nil {
		gologger.Debug().Msgf("读取项目文件失败 [%s]: %v", name, err)
		return nil, err
	}

	var project RedcProject
	if err := json.Unmarshal(data, &project); err != nil {
		return nil, fmt.Errorf("解析项目配置失败: %w", err)
	}

	return &project, nil
}

// GetCase 支持通过 ID(精确/模糊) 或 Name(精确) 查找 Case
// 逻辑参考 Docker: 优先精确匹配，其次 ID 前缀匹配。如果 ID 前缀匹配到多个，则报错歧义。
func (p *RedcProject) GetCase(identifier string) (*Case, error) {
	var candidates []*Case

	// 遍历所有 Case
	for i := range p.Case {
		// 使用指针引用，避免大结构体复制，且允许返回原始切片中的地址
		c := p.Case[i]

		// 先绑定项目操作函数
		c.bindHandlers(p)

		// 1. 第一优先级：精确匹配 (ID 或 Name)
		// 如果输入的字符串完全等于 ID 或 Name，直接认定为目标
		if c.Id == identifier || c.Name == identifier {
			// 绑定 project 参数
			return c, nil
		}

		// 2. 第二优先级：ID 前缀模糊匹配 (Docker 风格)
		// 只有当 identifier 是 ID 的前缀时才算 (例如输入 "abc" 匹配 "abcde")
		// 注意：通常不对 Name 做前缀匹配，防止误操作，这里只针对 ID
		if strings.HasPrefix(c.Id, identifier) {
			candidates = append(candidates, c)
		}
	}

	// 3. 处理匹配结果
	if len(candidates) == 0 {
		return nil, fmt.Errorf("在项目 %s 中未找到 ID 或名称为 '%s' 的场景", p.ProjectName, identifier)
	}

	if len(candidates) == 1 {
		l := candidates[0]
		gologger.Debug().Msgf("关键词匹配「%s」%s %s", identifier, l.Name, l.Id)
		return l, nil
	}

	// 4. 歧义处理 (匹配到多个 ID 前缀)
	// 例如输入 "a1", 既匹配了 "a1b2..." 也匹配了 "a1c3..."
	return nil, fmt.Errorf("输入 '%s' 存在歧义，匹配到 %d 个场景 (请提供更完整的 ID)", identifier, len(candidates))
}

// HandleCase 删除指定uid的case
func (p *RedcProject) HandleCase(c *Case) error {
	uid := c.Id
	found := false
	for i, caseInfo := range p.Case {
		if caseInfo.Id == uid {
			// 执行删除逻辑：将 i 之后的所有元素前移
			p.Case = append(p.Case[:i], p.Case[i+1:]...)
			found = true
			break // 找到并删除后立即退出循环
		}
	}

	if !found {
		return fmt.Errorf("未找到 UID 为 %s 的 case，无需删除", uid)
	}

	// 3. 将修改后的 project 写回文件
	err := p.SaveProject()
	if err != nil {
		return fmt.Errorf("更新项目文件失败: %v", err)
	}

	return nil
}

func (p *RedcProject) AddCase(c *Case) error {
	p.Case = append(p.Case, c)
	return nil
}

// CaseList 输出项目进程
func (p *RedcProject) CaseList() {
	// minwidth=0: 最小单元格宽度
	// tabwidth=8: tab 字符宽度
	// padding=3:  列之间至少保留 3 个空格（比原来的 1 个更清晰）
	// padchar=' ': 填充符
	// flags=0:    默认左对齐 (Docker 风格)，去掉 AlignRight
	w := tabwriter.NewWriter(os.Stdout, 0, 8, 3, ' ', 0)

	// 优化2: 表头全大写，符合 CLI 惯例
	// 并在每一列后明确加上 \t 进行分割
	fmt.Fprintln(w, "Case ID\tTYPE\tNAME\tOPERATOR\tCREATED\tSTATUS")

	for _, c := range p.Case {

		// ID过长显示前12位
		displayID := c.Id
		if len(c.Id) > 12 {
			displayID = c.Id[:12]
		}
		createTime := parseTime(c.StateTime) // 解析字符串时间
		var displayStatus string
		// 3. 这里使用 c.State 和新的常量
		switch c.State {
		case StateRunning:
			displayStatus = fmt.Sprintf("Up %s", humanDuration(createTime))

		case StateStopped:
			displayStatus = fmt.Sprintf("Exited (0) %s ago", humanDurationShort(createTime))

		case StateError:
			displayStatus = "Error"

		case StateCreated:
			displayStatus = "Created"

		// 如果之前的旧数据没有 State 字段，可能需要一个默认兜底
		case "":
			displayStatus = "Unknown"

		default:
			displayStatus = string(c.State)
		}

		// 使用 Fprintf 配合 \t 格式化输出
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			displayID,
			c.Type,
			c.Name,
			c.Operator,
			c.CreateTime,
			displayStatus,
		)

	}

	// 刷新缓冲区，确保输出
	if err := w.Flush(); err != nil {
		// 假设 gologger 是你的日志库
		// gologger.Fatal().Msgf("表格输出失败: %s", err.Error())
		fmt.Fprintf(os.Stderr, "表格输出失败: %s\n", err.Error())
	}
}

// SaveProject 将修改后的项目配置写回 JSON 文件
func (p *RedcProject) SaveProject() error {
	// Get the project-specific mutex and lock it
	mu := getProjectMutex(p.ProjectName)
	mu.Lock()
	defer mu.Unlock()

	dirPath := p.ProjectPath
	path := filepath.Join(ProjectPath, p.ProjectName, ProjectFile)

	// 2. 防御性编程：确保目录存在
	// 防止用户手动删除了目录，导致保存文件失败
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return fmt.Errorf("无法恢复项目目录: %w", err)
	}

	// Re-read the project file to get the latest state
	// This ensures we don't overwrite changes made by other concurrent operations
	var latestProject RedcProject
	if data, err := os.ReadFile(path); err == nil {
		// File exists, read it
		if err := json.Unmarshal(data, &latestProject); err == nil {
			// Successfully parsed, merge the cases
			// Create a map of current case IDs for fast lookup
			currentCases := make(map[string]*Case)
			for _, c := range p.Case {
				currentCases[c.Id] = c
			}

			// Update or keep cases from latest file
			mergedCases := make([]*Case, 0, len(latestProject.Case)+len(p.Case))
			seenIDs := make(map[string]bool)

			// First, add/update cases from the latest file with our current changes
			for _, latestCase := range latestProject.Case {
				if currentCase, exists := currentCases[latestCase.Id]; exists {
					// We have this case, use our version (it has the latest changes)
					mergedCases = append(mergedCases, currentCase)
					seenIDs[currentCase.Id] = true
				} else {
					// This case exists in file but not in our memory, keep it
					mergedCases = append(mergedCases, latestCase)
					seenIDs[latestCase.Id] = true
				}
			}

			// Then add any new cases that we have but weren't in the file
			for id, currentCase := range currentCases {
				if !seenIDs[id] {
					mergedCases = append(mergedCases, currentCase)
				}
			}

			// Update our case list with merged result
			p.Case = mergedCases
		}
	}

	// 2. 序列化数据
	// MarshalIndent 会生成带缩进的 JSON，方便人类阅读；如果追求体积小，可用 json.Marshal
	data, err := json.MarshalIndent(p, "", "    ")
	if err != nil {
		return fmt.Errorf("序列化失败: %v", err)
	}

	// 3. 写入文件
	// 0644 表示：所有者有读写权限，其他人只读
	err = os.WriteFile(path, data, 0644)
	if err != nil {
		return fmt.Errorf("写入文件失败: %v", err)
	}

	return nil
}

// 简单的时长计算，返回 "2 hours", "5 minutes" 等
func humanDurationShort(t time.Time) string {
	d := time.Since(t)
	if d.Seconds() < 60 {
		return fmt.Sprintf("%.0f seconds", d.Seconds())
	} else if d.Minutes() < 60 {
		return fmt.Sprintf("%.0f minutes", d.Minutes())
	} else if d.Hours() < 24 {
		return fmt.Sprintf("%.0f hours", d.Hours())
	} else {
		return fmt.Sprintf("%.0f days", d.Hours()/24)
	}
}
