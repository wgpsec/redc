package mod

import "github.com/hashicorp/terraform-exec/tfexec"

const (
	StateRunning  string = "running"
	StateStopped  string = "stopped"
	StateError    string = "error"
	StateCreated  string = "created"
	StatePending  string = "pending"
	StateUnknown  string = "unknown"
	StateStarting string = "starting" // 正在启动
	StateStopping string = "stopping" // 正在停止
	StateRemoving    string = "removing"    // 正在删除
	StateTerminated  string = "terminated"  // 被云厂商回收（Spot 实例）
)

// RedcProject 项目结构体
type RedcProject struct {
	ProjectName string `json:"project_name"`
	ProjectPath string `json:"project_path"`
	CreateTime  string `json:"create_time"`
	User        string `json:"user"`
}

// Case 项目信息
type Case struct {
	// Id uuid
	Id           string   `json:"id"`
	Name         string   `json:"name"`
	Type         string   `json:"type"`
	Module       string   `json:"module,omitempty"`
	Operator     string   `json:"operator"`
	Path         string   `json:"path"`
	Node         int      `json:"node"`
	CreateTime   string   `json:"create_time"`
	StateTime    string   `json:"state_time"`
	Parameter    []string `json:"parameter"`
	State        string   `json:"state"`
	ProjectID    string   `json:"-"`
	Output       string
	output       map[string]tfexec.OutputMeta
	saveHandler  func() error
	removeHandle func() error
}
