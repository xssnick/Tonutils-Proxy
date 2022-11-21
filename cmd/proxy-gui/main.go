package main

import (
	_ "embed"
	"tonutils-proxy/cmd/proxy-gui/ui"
	"tonutils-proxy/internal/proxy"
)

func main() {
	ui.NewUI().Run(start)
}

var proxyStarted bool

func start(res chan<- proxy.State) func() {
	return func() {
		if !proxyStarted {
			go func() {
				_ = proxy.StartProxy("127.0.0.1:8080", false, res)
				proxyStarted = false
			}()
		}

		proxyStarted = true
	}
}
