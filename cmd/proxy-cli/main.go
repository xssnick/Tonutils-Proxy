package main

import (
	"flag"
	"github.com/xssnick/tonutils-proxy/proxy"
	"log"
)

var GitCommit string

func main() {
	var addr = flag.String("addr", "127.0.0.1:8080", "The addr of the proxy.")
	var debug = flag.Bool("debug", false, "Show additional logs")
	var blockHttp = flag.Bool("no-http", false, "Block ordinary http requests")
	flag.Parse()

	log.Println("Version:", GitCommit)
	if *blockHttp {
		log.Println("Ordinary HTTP Will be blocked (flag --no-http set)")
	}

	_, err := proxy.StartProxy(*addr, *debug, nil, "CLI "+GitCommit, *blockHttp)
	if err != nil {
		log.Println("failed to start proxy:", err.Error())
		return
	}

	<-make(chan bool)
}
