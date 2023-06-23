package main

import (
	"context"
	"fmt"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"github.com/xssnick/tonutils-proxy/proxy"
	"github.com/xssnick/tonutils-proxy/proxy/access"
	"log"
	"os"
	"os/exec"
	rt "runtime"
)

// App struct
type App struct {
	ctx context.Context

	statusUpd    chan proxy.State
	proxyStarted bool
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

func (a *App) StartProxy() {
	if !a.proxyStarted {
		a.proxyStarted = true
		go func() {
			_ = proxy.StartProxy("127.0.0.1:8080", false, a.statusUpd, false)
			err := access.SetProxy("127.0.0.1:8080")
			if err != nil {
				println(err.Error())
			}

			openbrowser("http://foundation.ton/")
		}()
	}
}

func (a *App) StopProxy() {
	_ = access.ClearProxy()
	os.Exit(0)
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
