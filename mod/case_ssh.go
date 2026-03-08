package mod

import (
	"fmt"
	"path/filepath"
	"red-cloud/utils/sshutil"
	"strings"
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

// GetSSHConfig 统一获取 SSH 连接配置（单实例，取第一个 IP）
// 自动尝试多种常见的 Output Key (兼容 ecs_ip/public_ip, password/ecs_password，instance_ip)
func (c *Case) GetSSHConfig() (*sshutil.SSHConfig, error) {
	configs, err := c.GetSSHConfigs()
	if err != nil {
		return nil, err
	}
	if len(configs) == 0 {
		return nil, fmt.Errorf("无法获取 SSH 配置")
	}
	return configs[0], nil
}

// GetSSHConfigs 获取所有实例的 SSH 连接配置（支持多实例场景）
func (c *Case) GetSSHConfigs() ([]*sshutil.SSHConfig, error) {
	if c == nil {
		return nil, fmt.Errorf("case instance is nil")
	}

	// 1. 获取所有 IP（支持数组）
	ipKeys := []string{"public_ip", "ecs_ip", "ip", "main_ip", "instance_ip"}
	var ips []string
	for _, key := range ipKeys {
		list, err := c.GetInstanceInfoList(key)
		if err == nil && len(list) > 0 {
			ips = list
			break
		}
	}
	if len(ips) == 0 {
		return nil, fmt.Errorf("无法获取实例 IP (尝试了: %v)", ipKeys)
	}

	// 2. 获取所有密码（支持数组）
	pwdKeys := []string{"password", "ecs_password", "root_password", "ssh_password"}
	var pwds []string
	for _, key := range pwdKeys {
		list, _ := c.GetInstanceInfoList(key)
		if len(list) > 0 {
			pwds = list
			break
		}
	}

	// 3. 获取 SSH 私钥路径
	keyPathKeys := []string{"ssh_private_key_path", "ssh_key_path", "private_key_path", "key_path"}
	var keyPath string
	for _, key := range keyPathKeys {
		keyPath, _ = c.GetInstanceInfo(key)
		if keyPath != "" {
			break
		}
	}
	if keyPath != "" && c.Path != "" {
		if strings.HasPrefix(keyPath, "./") || strings.HasPrefix(keyPath, "../") {
			keyPath = filepath.Join(c.Path, keyPath)
		}
	}

	// 4. 获取 SSH 用户名
	userKeys := []string{"ssh_user", "user", "username"}
	var user string
	for _, key := range userKeys {
		user, _ = c.GetInstanceInfo(key)
		if user != "" {
			break
		}
	}
	if user == "" {
		user = "root"
	}

	// 5. 为每个 IP 构建配置
	configs := make([]*sshutil.SSHConfig, 0, len(ips))
	for i, ip := range ips {
		pwd := ""
		if i < len(pwds) {
			pwd = pwds[i]
		} else if len(pwds) == 1 {
			// 如果只有一个密码，所有实例共用
			pwd = pwds[0]
		}
		configs = append(configs, &sshutil.SSHConfig{
			Host:     ip,
			Port:     22,
			User:     user,
			Password: pwd,
			KeyPath:  keyPath,
			Timeout:  5 * time.Second,
		})
	}

	return configs, nil
}
func (c *Case) getSSHClient() (*sshutil.Client, error) {
	info, err := c.GetSSHConfig()
	if err != nil {
		return nil, err
	}
	return sshutil.NewClient(info)
}
