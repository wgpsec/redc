package sshutil

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"red-cloud/mod/gologger"
	"time"

	"github.com/pkg/sftp"     // 需要: go get github.com/pkg/sftp
	"golang.org/x/crypto/ssh" // 需要: go get golang.org/x/crypto/ssh
	"golang.org/x/term"
)

// SSHConfig 定义连接参数
type SSHConfig struct {
	User       string
	Host       string
	Port       int
	Password   string
	KeyPath    string
	MaxRetries int
	Timeout    time.Duration
}

type Client struct {
	*ssh.Client
}

// NewClient 创建 SSH 连接
func NewClient(conf *SSHConfig) (*Client, error) {
	var authMethods []ssh.AuthMethod
	maxRetries := 3

	// 优先尝试密钥 (Requirement #3)
	if conf.KeyPath != "" {
		key, err := ioutil.ReadFile(conf.KeyPath)
		if err == nil {
			signer, err := ssh.ParsePrivateKey(key)
			if err == nil {
				authMethods = append(authMethods, ssh.PublicKeys(signer))
			}
		}
	}
	// 其次尝试密码
	if conf.Password != "" {
		authMethods = append(authMethods, ssh.Password(conf.Password))
	}
	gologger.Debug().Msgf("正在尝试SSH连接: ssh %s@%s %s\n", conf.User, conf.Host, conf.Password)
	// 如果没有设置重试，默认3次
	if conf.MaxRetries > 0 {
		maxRetries = 3
	}

	config := &ssh.ClientConfig{
		User:            conf.User,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // 红队场景通常忽略 HostKey 检查
		Timeout:         conf.Timeout,
		HostKeyAlgorithms: []string{
			ssh.KeyAlgoRSASHA256,
			ssh.KeyAlgoRSASHA512,
			ssh.KeyAlgoRSA,
			ssh.KeyAlgoECDSA256,
			ssh.KeyAlgoECDSA384,
			ssh.KeyAlgoECDSA521,
			ssh.KeyAlgoED25519,
		},
		Config: ssh.Config{
			Ciphers: []string{
				"aes128-gcm@openssh.com",
				"chacha20-poly1305@openssh.com",
				"aes128-ctr", "aes192-ctr", "aes256-ctr",
			},
			KeyExchanges: []string{
				"curve25519-sha256",
				"curve25519-sha256@libssh.org",
				"ecdh-sha2-nistp256",
				"diffie-hellman-group14-sha1",
				"diffie-hellman-group1-sha1",
			},
		},
	}
	var client *ssh.Client
	var err error
	addr := fmt.Sprintf("%s:%d", conf.Host, conf.Port)
	for i := 0; i < maxRetries; i++ {
		client, err = ssh.Dial("tcp", addr, config)
		if err == nil {
			// 连接成功，直接返回
			return &Client{client}, nil
		}
		//var netErr net.Error
		//if errors.As(err, &netErr) && netErr.Timeout() {
		//	gologger.Error().Msgf("SSH 网络连接超时")
		//}
		//if errors.Is(err, syscall.ECONNREFUSED) {
		//	gologger.Error().Msgf("连接被拒绝 (端口可能未开放或被防火墙拦截): %s\n", addr)
		//}
		// 如果不是最后一次尝试，则等待一段时间重试
		if i < maxRetries-1 {
			// 第二次尝试才进行提示
			if i >= 1 {
				gologger.Error().Msgf("SSH连接错误，正在重试 (%d/%d)... %s\n", i+1, maxRetries, err.Error())
			}
			time.Sleep(30 * time.Second)
		}

	}

	return &Client{client}, fmt.Errorf("ssh dial failed after %d attempts: %w", maxRetries, err)
}

// RunCommand 执行非交互式命令 (docker exec id cmd)
func (c *Client) RunCommand(cmd string) error {
	session, err := c.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	session.Stdout = os.Stdout
	session.Stderr = os.Stderr
	// 不需要 Stdin，除非是交互模式
	return session.Run(cmd)
}

// RunInteractiveShell 启动交互式 Shell (docker exec -it id /bin/bash) (Requirement #4)
func (c *Client) RunInteractiveShell(cmd string) error {
	session, err := c.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	fd := int(os.Stdin.Fd())
	oldState, err := term.MakeRaw(fd) // 设置本地终端为 Raw 模式
	if err != nil {
		return err
	}
	defer term.Restore(fd, oldState)

	// 获取终端大小
	w, h, _ := term.GetSize(fd)
	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}

	// 请求 PTY
	if err := session.RequestPty("xterm", h, w, modes); err != nil {
		return err
	}

	session.Stdout = os.Stdout
	session.Stderr = os.Stderr
	session.Stdin = os.Stdin

	if cmd == "" {
		cmd = "/bin/bash" // 默认 Shell
	}

	if err := session.Start(cmd); err != nil {
		return err
	}
	return session.Wait()
}

// Upload 上传文件 (Local -> Remote)
func (c *Client) Upload(localPath, remotePath string) error {
	sftpClient, err := sftp.NewClient(c.Client)
	if err != nil {
		return fmt.Errorf("SFTP建立失败: %v", err)
	}
	defer sftpClient.Close()

	// 1. 打开本地源文件
	srcFile, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("无法读取本地文件: %v", err)
	}
	defer srcFile.Close()

	// 获取本地文件名
	localStat, err := srcFile.Stat()
	if err != nil {
		return err
	}
	fileName := localStat.Name()

	// 2. 智能处理远程路径逻辑
	// 尝试获取远程路径状态
	remoteStat, err := sftpClient.Stat(remotePath)
	if err == nil {
		// 如果远程路径存在，且是一个目录 (例如 /tmp)
		if remoteStat.IsDir() {
			// 将文件名拼接到目录后面 -> /tmp/tool
			remotePath = path.Join(remotePath, fileName)
		}
	} else {
		// 如果远程路径不存在，检查其父目录是否存在
		parentDir := path.Dir(remotePath)
		if _, err := sftpClient.Stat(parentDir); err != nil {
			// 父目录也不存在，尝试创建 (mkdir -p)
			if err := sftpClient.MkdirAll(parentDir); err != nil {
				return fmt.Errorf("无法创建远程目录结构: %v", err)
			}
		}
	}

	// 3. 创建远程文件
	// 注意：使用 path 包处理远程路径，确保使用 '/' 分隔符
	dstFile, err := sftpClient.Create(remotePath)
	if err != nil {
		return fmt.Errorf("无法创建远程文件 [%s]: %v", remotePath, err)
	}
	defer dstFile.Close()

	// 4. 开始传输
	// 修改权限与本地一致 (可选，但推荐)
	dstFile.Chmod(localStat.Mode())

	_, err = io.Copy(dstFile, srcFile)
	return err
}

// Download 下载文件 (Remote -> Local)
func (c *Client) Download(remotePath, localPath string) error {
	sftpClient, err := sftp.NewClient(c.Client)
	if err != nil {
		return fmt.Errorf("SFTP建立失败: %v", err)
	}
	defer sftpClient.Close()

	// 1. 打开远程源文件
	srcFile, err := sftpClient.Open(remotePath)
	if err != nil {
		return fmt.Errorf("无法读取远程文件: %v", err)
	}
	defer srcFile.Close()

	// 获取远程文件名
	remoteStat, err := srcFile.Stat()
	if err != nil {
		return err
	}
	fileName := remoteStat.Name()

	// 2. 智能处理本地路径逻辑
	localFileInfo, err := os.Stat(localPath)
	if err == nil {
		// 如果本地路径存在且是目录 (例如下载到 ./)
		if localFileInfo.IsDir() {
			localPath = filepath.Join(localPath, fileName)
		}
	} else {
		// 如果本地路径不存在，确保父文件夹存在
		parentDir := filepath.Dir(localPath)
		os.MkdirAll(parentDir, 0755)
	}

	// 3. 创建本地文件
	dstFile, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("无法创建本地文件: %v", err)
	}
	defer dstFile.Close()

	// 4. 传输
	dstFile.Chmod(remoteStat.Mode())
	_, err = io.Copy(dstFile, srcFile)
	return err
}

// RunCommandWithLogger 支持自定义 Writer 的执行方法
func (c *Client) RunCommandWithLogger(cmd string, writer io.Writer) error {
	session, err := c.Client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	session.Stdout = writer
	session.Stderr = writer

	return session.Run(cmd)
}
