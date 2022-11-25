package main

import (
	_ "embed"
	"fmt"
	"log"
	"os/exec"
	"runtime"
	"tonutils-proxy/cmd/proxy-gui/ui"
	"tonutils-proxy/internal/proxy"
	"tonutils-proxy/internal/proxy/access"
)

func main() {
	//	sigs := make(chan os.Signal, 1)
	//	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL, syscall.SIGQUIT, syscall.SIGSEGV)

	ui.NewUI().Run(start)

	if proxyStarted {
		_ = access.ClearProxy()
	}
}

var proxyStarted bool

func start(res chan<- proxy.State) func() {
	return func() {
		if !proxyStarted {
			go func() {
				_ = proxy.StartProxy("127.0.0.1:8080", false, res)
				err := access.SetProxy("127.0.0.1:8080")
				if err != nil {
					println(err.Error())
				}

				openbrowser("http://foundation.ton/")
			}()
		}

		proxyStarted = true
	}
}

func openbrowser(url string) {
	var err error

	switch runtime.GOOS {
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
