package mod

import (
	"fmt"
	"red-cloud/utils/sshutil"
	"time"
)

// InstanceInfo 结构体
type InstanceInfo struct {
	ID       string
	IP       string
	User     string
	Password string
	KeyPath  string
	Port     int
}

// GetSSHConfig 统一获取 SSH 连接配置
// 自动尝试多种常见的 Output Key (兼容 ecs_ip/public_ip, password/ecs_password)
func (c *Case) GetSSHConfig() (*sshutil.SSHConfig, error) {
	if c == nil {
		return nil, fmt.Errorf("case instance is nil")
	}

	// 1. 尝试获取 IP
	// 优先顺序: public_ip -> ecs_ip -> ip
	ipKeys := []string{"public_ip", "ecs_ip", "ip", "main_ip"}
	var ip string
	var err error

	for _, key := range ipKeys {
		ip, err = c.GetInstanceInfo(key)
		if err == nil && ip != "" {
			break
		}
	}
	if ip == "" {
		return nil, fmt.Errorf("无法获取实例 IP (尝试了: %v)", ipKeys)
	}

	// 2. 尝试获取密码
	// 优先顺序: password -> ecs_password -> root_password
	pwdKeys := []string{"password", "ecs_password", "root_password"}
	var pwd string
	for _, key := range pwdKeys {
		pwd, _ = c.GetInstanceInfo(key) // 密码允许失败(可能是 Key 登录)，但这里为了简化暂不处理 KeyPath
		if pwd != "" {
			break
		}
	}

	// 3. 返回标准配置
	return &sshutil.SSHConfig{
		Host:     ip,
		Port:     22,
		User:     "root",
		Password: pwd,
		//KeyPath: "",
		Timeout: 5 * time.Second,
	}, nil
}
func (c *Case) getSSHClient() (*sshutil.Client, error) {
	info, err := c.GetSSHConfig()
	if err != nil {
		return nil, err
	}
	return sshutil.NewClient(info)
}
