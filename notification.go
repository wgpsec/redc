package main

import (
	"fmt"
	"os/exec"
	"runtime"
	"sync"
	"time"
)

type NotificationManager struct {
	enabled bool
	mu      sync.RWMutex
}

func NewNotificationManager() *NotificationManager {
	return &NotificationManager{
		enabled: false,
	}
}

func (nm *NotificationManager) SetEnabled(enabled bool) {
	nm.mu.Lock()
	defer nm.mu.Unlock()
	nm.enabled = enabled
}

func (nm *NotificationManager) IsEnabled() bool {
	nm.mu.RLock()
	defer nm.mu.RUnlock()
	return nm.enabled
}

func (nm *NotificationManager) Send(title, message string) {
	if !nm.IsEnabled() {
		return
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("通知发送失败: %v\n", r)
			}
		}()

		time.Sleep(100 * time.Millisecond)

		switch runtime.GOOS {
		case "darwin":
			nm.sendMacOSNotification(title, message)
		case "windows":
			nm.sendWindowsNotification(title, message)
		case "linux":
			nm.sendLinuxNotification(title, message)
		default:
			fmt.Printf("不支持的平台: %s\n", runtime.GOOS)
		}
	}()
}

func (nm *NotificationManager) sendMacOSNotification(title, message string) {
	script := fmt.Sprintf(`display notification "%s" with title "%s" sound name "Glass"`, message, title)
	cmd := exec.Command("osascript", "-e", script)
	if err := cmd.Run(); err != nil {
		fmt.Printf("macOS 通知发送失败: %v\n", err)
	}
}

func (nm *NotificationManager) sendWindowsNotification(title, message string) {
	powershell := fmt.Sprintf(`[Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType = WindowsRuntime] | Out-Null; [Windows.Data.Xml.Dom.XmlDocument, Windows.Data.Xml.Dom.XmlDocument, ContentType = WindowsRuntime] | Out-Null; $template = @"<toast><visual><binding template=""ToastGeneric""><text>%s</text><text>%s</text></binding></visual></toast>"; $xml = New-Object Windows.Data.Xml.Dom.XmlDocument; $xml.LoadXml($template); $toast = New-Object Windows.UI.Notifications.ToastNotification $xml; $notifier = [Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier("redc-gui"); $notifier.Show($toast)`, title, message)
	cmd := exec.Command("powershell", "-NoProfile", "-Command", powershell)
	if err := cmd.Run(); err != nil {
		fmt.Printf("Windows 通知发送失败: %v\n", err)
	}
}

func (nm *NotificationManager) sendLinuxNotification(title, message string) {
	if _, err := exec.LookPath("notify-send"); err != nil {
		fmt.Printf("notify-send 未安装，无法发送 Linux 通知\n")
		return
	}
	cmd := exec.Command("notify-send", "-a", "redc-gui", title, message)
	if err := cmd.Run(); err != nil {
		fmt.Printf("Linux 通知发送失败: %v\n", err)
	}
}

func (nm *NotificationManager) SendSceneStarted(sceneName string) {
	title := "场景已启动"
	message := fmt.Sprintf("场景「%s」已成功启动", sceneName)
	nm.Send(title, message)
}

func (nm *NotificationManager) SendSceneStopped(sceneName string) {
	title := "场景已停止"
	message := fmt.Sprintf("场景「%s」已成功停止", sceneName)
	nm.Send(title, message)
}

func (nm *NotificationManager) SendSceneFailed(sceneName, action string) {
	title := "场景操作失败"
	message := fmt.Sprintf("场景「%s」%s失败", sceneName, action)
	nm.Send(title, message)
}
