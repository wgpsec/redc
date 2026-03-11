package main

import (
	"fmt"
	"os/exec"
	"red-cloud/i18n"
	"runtime"
	"sync"
	"time"
)

type NotificationManager struct {
	enabled    bool
	mu         sync.RWMutex
	webhookMgr *WebhookManager
}

func NewNotificationManager() *NotificationManager {
	wm := NewWebhookManager()
	wm.LoadFromSettings()
	return &NotificationManager{
		enabled:    false,
		webhookMgr: wm,
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
				fmt.Println(i18n.Tf("notify_send_failed", r))
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
			fmt.Println(i18n.Tf("notify_unsupported_os", runtime.GOOS))
		}
	}()
}

func (nm *NotificationManager) sendMacOSNotification(title, message string) {
	script := fmt.Sprintf(`display notification "%s" with title "%s" sound name "Glass"`, message, title)
	cmd := exec.Command("osascript", "-e", script)
	if err := cmd.Run(); err != nil {
		fmt.Println(i18n.Tf("notify_macos_failed", err))
	}
}

func (nm *NotificationManager) sendWindowsNotification(title, message string) {
	powershell := fmt.Sprintf(`[Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType = WindowsRuntime] | Out-Null; [Windows.Data.Xml.Dom.XmlDocument, Windows.Data.Xml.Dom.XmlDocument, ContentType = WindowsRuntime] | Out-Null; $template = @"<toast><visual><binding template=""ToastGeneric""><text>%s</text><text>%s</text></binding></visual></toast>"; $xml = New-Object Windows.Data.Xml.Dom.XmlDocument; $xml.LoadXml($template); $toast = New-Object Windows.UI.Notifications.ToastNotification $xml; $notifier = [Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier("redc-gui"); $notifier.Show($toast)`, title, message)
	cmd := exec.Command("powershell", "-NoProfile", "-Command", powershell)
	if err := cmd.Run(); err != nil {
		fmt.Println(i18n.Tf("notify_windows_failed", err))
	}
}

func (nm *NotificationManager) sendLinuxNotification(title, message string) {
	if _, err := exec.LookPath("notify-send"); err != nil {
		fmt.Println(i18n.T("notify_linux_not_installed"))
		return
	}
	cmd := exec.Command("notify-send", "-a", "redc-gui", title, message)
	if err := cmd.Run(); err != nil {
		fmt.Println(i18n.Tf("notify_linux_failed", err))
	}
}

func (nm *NotificationManager) SendSceneStarted(sceneName string) {
	title := i18n.T("notify_scene_started")
	message := i18n.Tf("notify_scene_started_msg", sceneName)
	nm.Send(title, message)
	if nm.webhookMgr != nil {
		nm.webhookMgr.Send(title, message, "#36a64f")
	}
}

func (nm *NotificationManager) SendSceneStopped(sceneName string) {
	title := i18n.T("notify_scene_stopped")
	message := i18n.Tf("notify_scene_stopped_msg", sceneName)
	nm.Send(title, message)
	if nm.webhookMgr != nil {
		nm.webhookMgr.Send(title, message, "#ffa500")
	}
}

func (nm *NotificationManager) SendSceneFailed(sceneName, action string) {
	title := i18n.T("notify_scene_failed")
	message := i18n.Tf("notify_scene_failed_msg", sceneName, action)
	nm.Send(title, message)
	if nm.webhookMgr != nil {
		nm.webhookMgr.Send(title, message, "#ff0000")
	}
}

func (nm *NotificationManager) SendSpotTerminated(sceneName, ips string) {
	title := i18n.T("notify_spot_terminated")
	message := i18n.Tf("notify_spot_terminated_msg", sceneName, ips)
	nm.Send(title, message)
	if nm.webhookMgr != nil {
		nm.webhookMgr.Send(title, message, "#ff4500")
	}
}

func (nm *NotificationManager) SendSpotRecovered(sceneName string) {
	title := i18n.T("notify_spot_recovered")
	message := i18n.Tf("notify_spot_recovered_msg", sceneName)
	nm.Send(title, message)
	if nm.webhookMgr != nil {
		nm.webhookMgr.Send(title, message, "#36a64f")
	}
}

func (nm *NotificationManager) SendSpotRecoverFailed(sceneName string) {
	title := i18n.T("notify_spot_recover_failed")
	message := i18n.Tf("notify_spot_recover_failed_msg", sceneName)
	nm.Send(title, message)
	if nm.webhookMgr != nil {
		nm.webhookMgr.Send(title, message, "#ff0000")
	}
}
