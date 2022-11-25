package main

import (
	_ "embed"
	"os"
	"os/signal"
	"syscall"
	"tonutils-proxy/cmd/proxy-gui/ui"
	"tonutils-proxy/internal/proxy"
	"tonutils-proxy/internal/proxy/access"
)

func main() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL, syscall.SIGQUIT, syscall.SIGSEGV)

	ui.NewUI().Run(start)

	<-sigs
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
				proxyStarted = false
			}()
		}

		proxyStarted = true
	}
}
