package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/ton-blockchain/adnl-tunnel/config"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"github.com/xssnick/tonutils-proxy/proxy"
	"github.com/xssnick/tonutils-proxy/proxy/access"
	"log"
	"os"
	"os/exec"
	rt "runtime"
	"sync"
)

// App struct
type App struct {
	ctx context.Context

	proxy        *proxy.Proxy
	statusUpd    chan proxy.State
	proxyStarted bool

	tunnelConfig string
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.statusUpd = make(chan proxy.State, 1)

	go func() {
		for {
			state := <-a.statusUpd
			runtime.EventsEmit(a.ctx, "statusUpdate", state.Type, state.State)
		}
	}()
}

func (a *App) shutdown(ctx context.Context) {
	if a.proxyStarted {
		log.Println("Clearing proxy")
		_ = access.ClearProxy()
	}
}

var openOnce sync.Once

func (a *App) AddTunnel() {
	path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		DefaultDirectory: "",
		DefaultFilename:  "tunnel-config.json",
		Title:            "Open Tunnel Config",
		Filters: []runtime.FileFilter{
			{
				DisplayName: "tunnel-config.json",
				Pattern:     "*.json",
			},
		},
		ShowHiddenFiles:            false,
		CanCreateDirectories:       false,
		ResolvesAliases:            false,
		TreatPackagesAsDirectories: false,
	})
	if err != nil {
		println(err.Error())
	} else {
		if path != "" {
			data, err := os.ReadFile(path)
			if err != nil {
				_, _ = runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
					Type:          runtime.ErrorDialog,
					Title:         "Failed to read tunnel config",
					Message:       err.Error(),
					DefaultButton: "Ok",
				})
				return
			}

			if len(data) > 0 {
				var cfg config.ClientConfig
				if err = json.Unmarshal(data, &cfg); err != nil {
					_, _ = runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
						Type:          runtime.ErrorDialog,
						Title:         "Failed to parse tunnel config",
						Message:       err.Error(),
						DefaultButton: "Ok",
					})
					return
				}

				if len(cfg.OutGateway.Key) != 32 {
					_, _ = runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
						Type:    runtime.ErrorDialog,
						Title:   "Failed to parse tunnel config",
						Message: "Invalid config format",
					})
					return
				}
			}
		}

		a.tunnelConfig = path
		runtime.EventsEmit(a.ctx, "tunnelAdded", a.tunnelConfig != "")
	}
}

func (a *App) StartProxy() {
	if !a.proxyStarted {
		a.proxyStarted = true
		go func() {
			var err error
			a.proxy, err = proxy.StartProxy("127.0.0.1:8080", 3, a.statusUpd, "GUI 1.4", false, "", a.tunnelConfig)
			if err != nil {
				a.proxyStarted = false
				runtime.EventsEmit(a.ctx, "statusUpdate", "stopped", "stopped")

				_, _ = runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
					Type:    runtime.ErrorDialog,
					Title:   "Startup",
					Message: err.Error(),
				})
				println(err.Error())
			} else {
				err = access.SetProxy("127.0.0.1:8080")
				if err != nil {
					println(err.Error())
				} else {
					openOnce.Do(func() {
						openbrowser("http://foundation.ton/")
					})
				}
			}
		}()
	}
}

func (a *App) StopProxy() {
	runtime.EventsEmit(a.ctx, "statusUpdate", "loading", "stopping")

	_ = access.ClearProxy()
	if a.proxy != nil {
		a.proxy.Stop()
		a.proxy = nil
		a.proxyStarted = false
	}
	runtime.EventsEmit(a.ctx, "statusUpdate", "stopped", "stopped")

	// os.Exit(0)
}

func openbrowser(url string) {
	var err error

	switch rt.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}
	if err != nil {
		log.Println("cannot open browser:", err)
	}
}
