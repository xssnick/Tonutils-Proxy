package main

import (
	"embed"
	"github.com/rs/zerolog/log"
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
	app, err := NewApp()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to init app")
	}

	height := 444
	if runtime.GOOS == "windows" {
		height += 40 // windows cut part of the height, so we extend it
	}

	// Create application with options
	err = wails.Run(&options.App{
		Title:         "TON Proxy",
		Width:         375,
		Height:        height,
		DisableResize: true,
		Mac: &mac.Options{
			Appearance: mac.NSAppearanceNameDarkAqua,
		},
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 0x23, G: 0x23, B: 0x28, A: 1},
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
