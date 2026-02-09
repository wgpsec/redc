package main

import (
	"embed"
	"runtime"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// Create an instance of the app structure
	app := NewApp()

	// Create application with options
	appOptions := &options.App{
		Title:  "RedC - 红队基础设施管理",
		Width:  1600,
		Height: 1000,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 250, G: 251, B: 252, A: 1}, // 匹配 App.svelte 的 bg-[#fafbfc]
		OnStartup:        app.startup,
		Bind: []interface{}{
			app,
		},
		Mac: &mac.Options{
			TitleBar: mac.TitleBarHiddenInset(),
		},
		Windows: &windows.Options{
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
			DisableWindowIcon:    false,
			CustomTheme: &windows.ThemeSettings{
				DarkModeTitleBar:   windows.RGB(32, 32, 32),
				DarkModeTitleText:  windows.RGB(255, 255, 255),
				DarkModeBorder:     windows.RGB(32, 32, 32),
				LightModeTitleBar:  windows.RGB(250, 251, 252),
				LightModeTitleText: windows.RGB(0, 0, 0),
				LightModeBorder:    windows.RGB(250, 251, 252),
			},
		},
	}

	// 只在 Windows 上启用无边框模式
	if runtime.GOOS == "windows" {
		appOptions.Frameless = true
	}

	err := wails.Run(appOptions)

	if err != nil {
		println("Error:", err.Error())
	}
}
