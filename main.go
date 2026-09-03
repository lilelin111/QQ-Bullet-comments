package main

import (
	"context"
	"embed"
	"os"
	"temp-project/store"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func NewApp() *App {
	return &App{}
}

func main() {
	app := NewApp()
	err := wails.Run(&options.App{
		Title:     "QQ弹幕",
		Width:     1024,
		Height:    768,
		MinHeight: 800,
		MinWidth:  600,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 123, G: 104, B: 238, A: 1},
		OnStartup: func(ctx context.Context) {
			app.startup(ctx)
		},
		Bind: []interface{}{
			app,
		},
	})
	if err != nil {
		println("Error:", err.Error())
	}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	exePath, err := os.Executable()
	if err != nil {
		println("获取程序路径失败:", err.Error())
		return
	}
	if err := store.CreateDesktopShortcut(exePath); err != nil {
		println("创建桌面快捷方式失败：:", err.Error())
	}

}
