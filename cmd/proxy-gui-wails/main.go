package main

import (
	"embed"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"runtime"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// Create an instance of the app structure
	app := NewApp()

	height := 270
	if runtime.GOOS == "windows" {
		height = 310 // windows cut part of the height, so we extend it
	}

	// Create application with options
	err := wails.Run(&options.App{
		Title:         "Tonutils Proxy",
		Width:         300,
		Height:        height,
		DisableResize: true,
		Mac: &mac.Options{
			Appearance: mac.NSAppearanceNameDarkAqua,
		},
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 255, G: 255, B: 255, A: 1},
		OnStartup:        app.startup,
		OnShutdown:       app.shutdown,
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
