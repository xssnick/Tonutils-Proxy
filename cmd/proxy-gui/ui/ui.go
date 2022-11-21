package ui

import (
	"embed"
	"fmt"
	"github.com/webview/webview"
	"os"
	"time"
	"tonutils-proxy/internal/proxy"
)

type UI struct {
}

//go:embed templates/*
var FS embed.FS

func NewUI() *UI {
	return &UI{}
}

func (ui *UI) Run(starter func(chan<- proxy.State) func()) {
	w := webview.New(true)
	defer w.Destroy()

	index, _ := FS.ReadFile("templates/index.html")

	w.SetTitle("TonUtils Proxy")
	w.SetSize(300, 300, webview.HintFixed)
	w.SetHtml(string(index))

	ch := make(chan proxy.State, 1)
	start := starter(ch)

	w.Bind("startProxy", func() {
		w.Dispatch(func() {
			w.Eval(fmt.Sprintf("document.getElementById('start').classList.add('disabled-button');"))
		})

		start()
	})

	w.Bind("exitProxy", func() {
		os.Exit(0)
	})

	go func() {
		go func() {
			time.Sleep(1 * time.Second)
			w.Dispatch(func() {
				w.Eval("document.addEventListener('contextmenu', event => event.preventDefault());")
			})
		}()

		for {
			state := <-ch
			if state.Stopped {
				w.Dispatch(func() {
					w.Eval(fmt.Sprintf("document.getElementById('start').classList.remove('disabled-button');"))
				})
			}

			color := "#a1a1a1"
			switch state.Type {
			case "loading":
				color = "greenyellow"
			case "error":
				color = "red"
			case "ready":
				color = "limegreen"
			}

			w.Dispatch(func() {
				w.Eval(fmt.Sprintf("var item = document.getElementById('status'); "+
					"item.innerText = '%s'; "+
					"item.style='color: %s';", state.State, color))
			})
		}
	}()

	w.Run()
}
