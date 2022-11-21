package main

import (
	"flag"
	"log"
	"tonutils-proxy/internal/proxy"
)

func main() {
	var addr = flag.String("addr", "127.0.0.1:8080", "The addr of the proxy.")
	var debug = flag.Bool("debug", false, "Show additional logs")
	flag.Parse()

	err := proxy.StartProxy(*addr, *debug, nil)
	if err != nil {
		log.Println("failed to start proxy:", err.Error())
		return
	}

	<-make(chan bool)
}
