package main

import (
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"time"

	"red-cloud/i18n"
	"red-cloud/utils/sshutil"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"golang.org/x/crypto/ssh"
)

// ExecCommandResult 执行命令的结果
type ExecCommandResult struct {
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	ExitCode int    `json:"exitCode"`
	Success  bool   `json:"success"`
	Error    string `json:"error,omitempty"`
}

// FileTransferResult 文件传输结果
type FileTransferResult struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// PortForwardSession tracks an active SSH port forward
type PortForwardSession struct {
	ID         string
	CaseID     string
	LocalPort  int
	RemoteHost string
	RemotePort int
	listener   net.Listener
	client     *sshutil.Client
	stopCh     chan struct{}
}

type PortForwardInfo struct {
	ID         string `json:"id"`
	CaseID     string `json:"caseId"`
	LocalPort  int    `json:"localPort"`
	RemoteHost string `json:"remoteHost"`
	RemotePort int    `json:"remotePort"`
	Status     string `json:"status"`
}

var (
	terminalSessions   = make(map[string]*sshutil.TerminalSession)
	terminalSessionsMu sync.Mutex

	portForwardSessions   = make(map[string]*PortForwardSession)
	portForwardSessionsMu sync.Mutex
)

// ExecCommand 在指定场景或自定义部署上执行命令并返回结果
func (a *App) ExecCommand(caseID string, command string) ExecCommandResult {
	a.mu.Lock()
	project := a.project
	service := a.customDeploymentService
	a.mu.Unlock()

	result := ExecCommandResult{}

	if project == nil {
		result.Error = i18n.T("app_project_not_loaded")
		result.Success = false
		return result
	}

	c, caseErr := project.GetCase(caseID)
	if caseErr == nil {
		sshConfig, err := c.GetSSHConfig()
		if err != nil {
			result.Error = i18n.Tf("app_ssh_config_failed", err)
			result.Success = false
			return result
		}

		return a.execSSHCommand(sshConfig, command)
	}

	if service != nil {
		sshConfig, err := a.getDeploymentSSHConfig(caseID)
		if err == nil {
			return a.execSSHCommand(sshConfig, command)
		}
		result.Error = i18n.Tf("app_case_or_deploy_not_found", caseID)
		result.Success = false
		return result
	}

	result.Error = i18n.Tf("app_case_not_found", caseErr)
	result.Success = false
	return result
}

// execSSHCommand 执行 SSH 命令的通用方法
func (a *App) execSSHCommand(sshConfig *sshutil.SSHConfig, command string) ExecCommandResult {
	result := ExecCommandResult{}

	client, err := sshutil.NewClient(sshConfig)
	if err != nil {
		result.Error = i18n.Tf("app_ssh_connect_failed", err)
		result.Success = false
		return result
	}
	defer client.Close()

	var stdoutBuf, stderrBuf strings.Builder
	session, err := client.NewSession()
	if err != nil {
		result.Error = i18n.Tf("app_ssh_session_failed", err)
		result.Success = false
		return result
	}
	defer session.Close()

	session.Stdout = &stdoutBuf
	session.Stderr = &stderrBuf

	err = session.Run(command)
	result.Stdout = stdoutBuf.String()
	result.Stderr = stderrBuf.String()

	if err != nil {
		if exitErr, ok := err.(*ssh.ExitError); ok {
			result.ExitCode = exitErr.ExitStatus()
		} else {
			result.ExitCode = -1
			result.Error = err.Error()
		}
		result.Success = false
	} else {
		result.ExitCode = 0
		result.Success = true
	}

	return result
}

// ExecUserdata 在指定场景或自定义部署上执行 userdata 脚本
func (a *App) ExecUserdata(caseID string, script string) ExecCommandResult {
	a.mu.Lock()
	project := a.project
	service := a.customDeploymentService
	a.mu.Unlock()

	result := ExecCommandResult{}

	if project == nil {
		result.Error = i18n.T("app_project_not_loaded")
		result.Success = false
		return result
	}

	var sshConfig *sshutil.SSHConfig

	c, caseErr := project.GetCase(caseID)
	if caseErr == nil {
		var err error
		sshConfig, err = c.GetSSHConfig()
		if err != nil {
			result.Error = i18n.Tf("app_ssh_config_failed", err)
			result.Success = false
			return result
		}
	} else {
		if service != nil {
			var err error
			sshConfig, err = a.getDeploymentSSHConfig(caseID)
			if err != nil {
				result.Error = i18n.Tf("app_case_or_deploy_not_found", caseID)
				result.Success = false
				return result
			}
		} else {
			result.Error = i18n.Tf("app_case_not_found", caseErr)
			result.Success = false
			return result
		}
	}

	client, err := sshutil.NewClient(sshConfig)
	if err != nil {
		result.Error = i18n.Tf("app_ssh_connect_failed", err)
		result.Success = false
		return result
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		result.Error = i18n.Tf("app_ssh_session_failed", err)
		result.Success = false
		return result
	}

	var stdoutBuf, stderrBuf strings.Builder
	session.Stdout = &stdoutBuf
	session.Stderr = &stderrBuf

	shell := "/bin/bash"
	if strings.HasPrefix(script, "<powershell>") || strings.HasPrefix(script, "#!/usr/bin/env pwsh") {
		shell = "/usr/bin/env pwsh"
	}

	command := fmt.Sprintf("cat > /tmp/userdata_script.sh << 'EOFSCRIPT'\n%s\nEOFSCRIPT\nchmod +x /tmp/userdata_script.sh\n%s /tmp/userdata_script.sh", script, shell)

	err = session.Run(command)
	result.Stdout = stdoutBuf.String()
	result.Stderr = stderrBuf.String()

	if err != nil {
		if exitErr, ok := err.(*ssh.ExitError); ok {
			result.ExitCode = exitErr.ExitStatus()
		} else {
			result.ExitCode = -1
			result.Error = err.Error()
		}
		result.Success = false
	} else {
		result.ExitCode = 0
		result.Success = true
	}

	return result
}

// UploadUserdataScript 上传 userdata 脚本内容到远程服务器
func (a *App) UploadUserdataScript(caseID string, scriptContent string, fileName string) FileTransferResult {
	a.mu.Lock()
	project := a.project
	service := a.customDeploymentService
	a.mu.Unlock()

	result := FileTransferResult{}

	if project == nil {
		result.Error = i18n.T("app_project_not_loaded")
		return result
	}

	var sshConfig *sshutil.SSHConfig

	c, caseErr := project.GetCase(caseID)
	if caseErr == nil {
		var err error
		sshConfig, err = c.GetSSHConfig()
		if err != nil {
			result.Error = i18n.Tf("app_ssh_config_failed", err)
			return result
		}
	} else {
		if service != nil {
			var err error
			sshConfig, err = a.getDeploymentSSHConfig(caseID)
			if err != nil {
				result.Error = i18n.Tf("app_case_or_deploy_not_found", caseID)
				return result
			}
		} else {
			result.Error = i18n.Tf("app_case_not_found", caseErr)
			return result
		}
	}

	client, err := sshutil.NewClient(sshConfig)
	if err != nil {
		result.Error = i18n.Tf("app_ssh_connect_failed", err)
		return result
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		result.Error = i18n.Tf("app_ssh_session_failed", err)
		return result
	}
	defer session.Close()

	session.Stdin = strings.NewReader(scriptContent)

	remotePath := fmt.Sprintf("/tmp/%s", fileName)
	command := fmt.Sprintf("cat > %s && chmod +x %s", remotePath, remotePath)

	var stdoutBuf, stderrBuf strings.Builder
	session.Stdout = &stdoutBuf
	session.Stderr = &stderrBuf

	err = session.Run(command)

	if err != nil {
		result.Error = i18n.Tf("app_ssh_upload_failed", err, stderrBuf.String())
		return result
	}

	result.Success = true
	return result
}

// UploadFile 上传文件到远程服务器（支持场景和自定义部署）
func (a *App) UploadFile(caseID string, localPath string, remotePath string) FileTransferResult {
	a.mu.Lock()
	project := a.project
	service := a.customDeploymentService
	a.mu.Unlock()

	result := FileTransferResult{}

	if project == nil {
		result.Error = i18n.T("app_project_not_loaded")
		return result
	}

	c, caseErr := project.GetCase(caseID)
	if caseErr == nil {
		sshConfig, err := c.GetSSHConfig()
		if err != nil {
			result.Error = i18n.Tf("app_ssh_config_failed", err)
			return result
		}

		return a.uploadFileSSH(sshConfig, localPath, remotePath)
	}

	if service != nil {
		sshConfig, err := a.getDeploymentSSHConfig(caseID)
		if err == nil {
			return a.uploadFileSSH(sshConfig, localPath, remotePath)
		}
		result.Error = i18n.Tf("app_case_or_deploy_not_found", caseID)
		return result
	}

	result.Error = i18n.Tf("app_case_not_found", caseErr)
	return result
}

// uploadFileSSH 上传文件的通用方法
func (a *App) uploadFileSSH(sshConfig *sshutil.SSHConfig, localPath string, remotePath string) FileTransferResult {
	result := FileTransferResult{}

	client, err := sshutil.NewClient(sshConfig)
	if err != nil {
		result.Error = i18n.Tf("app_ssh_connect_failed", err)
		return result
	}
	defer client.Close()

	if err := client.Upload(localPath, remotePath); err != nil {
		result.Error = i18n.Tf("app_ssh_upload_failed", err)
		return result
	}

	result.Success = true
	return result
}

// DownloadFile 从远程服务器下载文件（支持场景和自定义部署）
func (a *App) DownloadFile(caseID string, remotePath string, localPath string) FileTransferResult {
	a.mu.Lock()
	project := a.project
	service := a.customDeploymentService
	a.mu.Unlock()

	result := FileTransferResult{}

	if project == nil {
		result.Error = i18n.T("app_project_not_loaded")
		return result
	}

	c, caseErr := project.GetCase(caseID)
	if caseErr == nil {
		sshConfig, err := c.GetSSHConfig()
		if err != nil {
			result.Error = i18n.Tf("app_ssh_config_failed", err)
			return result
		}

		return a.downloadFileSSH(sshConfig, remotePath, localPath)
	}

	if service != nil {
		sshConfig, err := a.getDeploymentSSHConfig(caseID)
		if err == nil {
			return a.downloadFileSSH(sshConfig, remotePath, localPath)
		}
		result.Error = i18n.Tf("app_case_or_deploy_not_found", caseID)
		return result
	}

	result.Error = i18n.Tf("app_case_not_found", caseErr)
	return result
}

// downloadFileSSH 下载文件的通用方法
func (a *App) downloadFileSSH(sshConfig *sshutil.SSHConfig, remotePath string, localPath string) FileTransferResult {
	result := FileTransferResult{}

	client, err := sshutil.NewClient(sshConfig)
	if err != nil {
		result.Error = i18n.Tf("app_ssh_connect_failed", err)
		return result
	}
	defer client.Close()

	if err := client.Download(remotePath, localPath); err != nil {
		result.Error = i18n.Tf("app_ssh_download_failed", err)
		return result
	}

	result.Success = true
	return result
}

// SelectFile 打开文件选择对话框
func (a *App) SelectFile(title string) (string, error) {
	file, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{Title: title})
	return file, err
}

// SelectDirectory 打开目录选择对话框
func (a *App) SelectDirectory(title string) (string, error) {
	dir, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{Title: title})
	return dir, err
}

// SelectSaveFile 打开保存文件对话框
func (a *App) SelectSaveFile(title string, defaultFilename string) (string, error) {
	file, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           title,
		DefaultFilename: defaultFilename,
	})
	return file, err
}

// StartSSHTerminal 启动 SSH 终端会话
func (a *App) StartSSHTerminal(caseID string, rows, cols int) (string, error) {
	return a.StartSSHTerminalInstance(caseID, 0, rows, cols)
}

// StartSSHTerminalInstance 启动指定实例索引的 SSH 终端（支持多实例场景）
func (a *App) StartSSHTerminalInstance(caseID string, instanceIndex int, rows, cols int) (string, error) {
	a.mu.Lock()
	project := a.project
	a.mu.Unlock()

	if project == nil {
		return "", fmt.Errorf("%s", i18n.T("app_project_not_loaded"))
	}

	var sshConfig *sshutil.SSHConfig

	// 尝试作为 Case 获取多实例配置
	c, caseErr := project.GetCase(caseID)
	if caseErr == nil {
		configs, err := c.GetSSHConfigs()
		if err != nil {
			return "", err
		}
		if instanceIndex < 0 || instanceIndex >= len(configs) {
			return "", fmt.Errorf("实例索引 %d 超出范围 (共 %d 个实例)", instanceIndex, len(configs))
		}
		sshConfig = configs[instanceIndex]
	} else {
		// 非 Case（自定义部署等），使用原有逻辑
		var err error
		sshConfig, err = a.getSSHConfig(caseID)
		if err != nil {
			return "", err
		}
	}

	client, err := sshutil.NewClient(sshConfig)
	if err != nil {
		return "", fmt.Errorf(i18n.Tf("app_ssh_connect_failed", err))
	}

	session, err := client.NewTerminalSession(rows, cols)
	if err != nil {
		client.Close()
		return "", fmt.Errorf(i18n.Tf("app_terminal_session_failed", err))
	}

	sessionID := fmt.Sprintf("%s-%d", caseID, time.Now().Unix())

	terminalSessionsMu.Lock()
	terminalSessions[sessionID] = session
	terminalSessionsMu.Unlock()

	go a.readTerminalOutput(sessionID, session)

	return sessionID, nil
}

// readTerminalOutput 读取终端输出并发送到前端
func (a *App) readTerminalOutput(sessionID string, session *sshutil.TerminalSession) {
	buf := make([]byte, 4096)
	for {
		n, err := session.Read(buf)
		if err != nil {
			if err != io.EOF {
				runtime.EventsEmit(a.ctx, "terminal-error-"+sessionID, err.Error())
			}
			terminalSessionsMu.Lock()
			delete(terminalSessions, sessionID)
			terminalSessionsMu.Unlock()
			runtime.EventsEmit(a.ctx, "terminal-closed-"+sessionID, true)
			break
		}

		if n > 0 {
			runtime.EventsEmit(a.ctx, "terminal-output-"+sessionID, string(buf[:n]))
		}
	}
}

// WriteToTerminal 向终端写入数据
func (a *App) WriteToTerminal(sessionID string, data string) error {
	terminalSessionsMu.Lock()
	session, exists := terminalSessions[sessionID]
	terminalSessionsMu.Unlock()

	if !exists {
		return fmt.Errorf("%s", i18n.T("app_terminal_not_found"))
	}

	return session.Write([]byte(data))
}

// ResizeTerminal 调整终端大小
func (a *App) ResizeTerminal(sessionID string, rows, cols int) error {
	terminalSessionsMu.Lock()
	session, exists := terminalSessions[sessionID]
	terminalSessionsMu.Unlock()

	if !exists {
		return fmt.Errorf("%s", i18n.T("app_terminal_not_found"))
	}

	return session.Resize(rows, cols)
}

// CloseTerminal 关闭终端会话
func (a *App) CloseTerminal(sessionID string) error {
	terminalSessionsMu.Lock()
	session, exists := terminalSessions[sessionID]
	if exists {
		delete(terminalSessions, sessionID)
	}
	terminalSessionsMu.Unlock()

	if !exists {
		return nil
	}

	return session.Close()
}

// ListRemoteFiles 列出远程目录文件
func (a *App) ListRemoteFiles(caseID string, remotePath string) ([]sshutil.FileInfo, error) {
	sshConfig, err := a.getSSHConfig(caseID)
	if err != nil {
		return nil, err
	}

	client, err := sshutil.NewClient(sshConfig)
	if err != nil {
		return nil, fmt.Errorf(i18n.Tf("app_ssh_connect_failed", err))
	}
	defer client.Close()

	return client.ListFiles(remotePath)
}

// CreateRemoteDirectory 创建远程目录
func (a *App) CreateRemoteDirectory(caseID string, remotePath string) error {
	sshConfig, err := a.getSSHConfig(caseID)
	if err != nil {
		return err
	}

	client, err := sshutil.NewClient(sshConfig)
	if err != nil {
		return fmt.Errorf(i18n.Tf("app_ssh_connect_failed", err))
	}
	defer client.Close()

	return client.CreateDirectory(remotePath)
}

// DeleteRemoteFile 删除远程文件或目录
func (a *App) DeleteRemoteFile(caseID string, remotePath string) error {
	if a.project == nil {
		return fmt.Errorf("%s", i18n.T("app_project_not_loaded"))
	}

	sshConfig, err := a.getSSHConfig(caseID)
	if err != nil {
		return err
	}

	client, err := sshutil.NewClient(sshConfig)
	if err != nil {
		return fmt.Errorf(i18n.Tf("app_ssh_connect_failed", err))
	}
	defer client.Close()

	return client.DeleteFile(remotePath)
}

// RenameRemoteFile 重命名远程文件或目录
func (a *App) RenameRemoteFile(caseID string, oldPath, newPath string) error {
	if a.project == nil {
		return fmt.Errorf("%s", i18n.T("app_project_not_loaded"))
	}

	sshConfig, err := a.getSSHConfig(caseID)
	if err != nil {
		return err
	}

	client, err := sshutil.NewClient(sshConfig)
	if err != nil {
		return fmt.Errorf(i18n.Tf("app_ssh_connect_failed", err))
	}
	defer client.Close()

	return client.RenameFile(oldPath, newPath)
}

// GetRemoteFileContent 获取远程文件内容（用于预览）
func (a *App) GetRemoteFileContent(caseID string, remotePath string) (string, error) {
	if a.project == nil {
		return "", fmt.Errorf("%s", i18n.T("app_project_not_loaded"))
	}

	sshConfig, err := a.getSSHConfig(caseID)
	if err != nil {
		return "", err
	}

	client, err := sshutil.NewClient(sshConfig)
	if err != nil {
		return "", fmt.Errorf(i18n.Tf("app_ssh_connect_failed", err))
	}
	defer client.Close()

	content, err := client.GetFileContent(remotePath, 1024*1024)
	if err != nil {
		return "", err
	}

	return string(content), nil
}

// WriteRemoteFileContent 写入远程文件内容
func (a *App) WriteRemoteFileContent(caseID string, remotePath string, content string) error {
	if a.project == nil {
		return fmt.Errorf("%s", i18n.T("app_project_not_loaded"))
	}

	sshConfig, err := a.getSSHConfig(caseID)
	if err != nil {
		return err
	}

	client, err := sshutil.NewClient(sshConfig)
	if err != nil {
		return fmt.Errorf(i18n.Tf("app_ssh_connect_failed", err))
	}
	defer client.Close()

	return client.WriteFileContent(remotePath, []byte(content))
}

// StartPortForward starts an SSH local port forward (like ssh -L localPort:remoteHost:remotePort)
func (a *App) StartPortForward(caseID string, localPort int, remoteHost string, remotePort int) (PortForwardInfo, error) {
	sshConfig, err := a.getSSHConfig(caseID)
	if err != nil {
		return PortForwardInfo{}, fmt.Errorf("获取SSH配置失败: %w", err)
	}

	client, err := sshutil.NewClient(sshConfig)
	if err != nil {
		return PortForwardInfo{}, fmt.Errorf("SSH连接失败: %w", err)
	}

	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", localPort))
	if err != nil {
		client.Close()
		return PortForwardInfo{}, fmt.Errorf("本地端口 %d 监听失败: %w", localPort, err)
	}

	id := fmt.Sprintf("pf-%s-%d-%d", caseID, localPort, time.Now().Unix())
	stopCh := make(chan struct{})

	session := &PortForwardSession{
		ID:         id,
		CaseID:     caseID,
		LocalPort:  localPort,
		RemoteHost: remoteHost,
		RemotePort: remotePort,
		listener:   listener,
		client:     client,
		stopCh:     stopCh,
	}

	portForwardSessionsMu.Lock()
	portForwardSessions[id] = session
	portForwardSessionsMu.Unlock()

	go func() {
		defer func() {
			listener.Close()
			client.Close()
			portForwardSessionsMu.Lock()
			delete(portForwardSessions, id)
			portForwardSessionsMu.Unlock()
			runtime.EventsEmit(a.ctx, "port-forward-closed", id)
		}()

		for {
			localConn, err := listener.Accept()
			if err != nil {
				select {
				case <-stopCh:
					return
				default:
					return
				}
			}

			remoteAddr := fmt.Sprintf("%s:%d", remoteHost, remotePort)
			remoteConn, err := client.Client.Dial("tcp", remoteAddr)
			if err != nil {
				localConn.Close()
				continue
			}

			// Bidirectional copy
			go func() {
				defer localConn.Close()
				defer remoteConn.Close()
				done := make(chan struct{}, 2)
				go func() { io.Copy(remoteConn, localConn); done <- struct{}{} }()
				go func() { io.Copy(localConn, remoteConn); done <- struct{}{} }()
				<-done
			}()
		}
	}()

	return PortForwardInfo{
		ID:         id,
		CaseID:     caseID,
		LocalPort:  localPort,
		RemoteHost: remoteHost,
		RemotePort: remotePort,
		Status:     "active",
	}, nil
}

// StopPortForward stops a port forward session
func (a *App) StopPortForward(id string) error {
	portForwardSessionsMu.Lock()
	session, ok := portForwardSessions[id]
	portForwardSessionsMu.Unlock()

	if !ok {
		return fmt.Errorf("端口转发会话未找到: %s", id)
	}

	close(session.stopCh)
	session.listener.Close()
	return nil
}

// ListPortForwards returns all active port forward sessions
func (a *App) ListPortForwards() []PortForwardInfo {
	portForwardSessionsMu.Lock()
	defer portForwardSessionsMu.Unlock()

	var result []PortForwardInfo
	for _, s := range portForwardSessions {
		result = append(result, PortForwardInfo{
			ID:         s.ID,
			CaseID:     s.CaseID,
			LocalPort:  s.LocalPort,
			RemoteHost: s.RemoteHost,
			RemotePort: s.RemotePort,
			Status:     "active",
		})
	}
	return result
}

// GetSSHInfoForCase returns SSH connection info for display (host, port, user)
func (a *App) GetSSHInfoForCase(caseID string) (map[string]interface{}, error) {
	sshConfig, err := a.getSSHConfig(caseID)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"host": sshConfig.Host,
		"port": sshConfig.Port,
		"user": sshConfig.User,
	}, nil
}

// GetSSHInfosForCase 获取场景的所有实例 SSH 信息（支持多实例）
func (a *App) GetSSHInfosForCase(caseID string) ([]map[string]interface{}, error) {
	a.mu.Lock()
	project := a.project
	a.mu.Unlock()

	if project == nil {
		return nil, fmt.Errorf("%s", i18n.T("app_project_not_loaded"))
	}

	// 尝试作为 Case 处理
	c, err := project.GetCase(caseID)
	if err != nil {
		// 不是 Case，按单实例处理（自定义部署等）
		config, err2 := a.getSSHConfig(caseID)
		if err2 != nil {
			return nil, err2
		}
		return []map[string]interface{}{{
			"host": config.Host,
			"port": config.Port,
			"user": config.User,
		}}, nil
	}

	// 是 Case，获取所有实例
	configs, err := c.GetSSHConfigs()
	if err != nil {
		return nil, err
	}
	result := make([]map[string]interface{}, 0, len(configs))
	for _, cfg := range configs {
		result = append(result, map[string]interface{}{
			"host": cfg.Host,
			"port": cfg.Port,
			"user": cfg.User,
		})
	}
	return result, nil
}
