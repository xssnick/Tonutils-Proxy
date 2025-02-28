package main

import (
	"errors"
	"flag"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/xssnick/tonutils-proxy/proxy"
	"os"
)

var GitCommit string

func main() {
	var addr = flag.String("addr", "127.0.0.1:8080", "The addr of the proxy.")
	var verbosity = flag.Int("verbosity", 2, "Debug logs")
	var blockHttp = flag.Bool("no-http", false, "Block ordinary http requests")
	var tunnelConfig = flag.String("tunnel-config", "", "tunnel config path")
	var networkConfigPath = flag.String("global-config", "", "path to ton network config file")

	flag.Parse()

	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout}).Level(zerolog.InfoLevel)
	if *verbosity >= 3 {
		log.Logger = log.Logger.Level(zerolog.DebugLevel)
	}

	log.Info().Msg("Version:" + GitCommit)
	if *blockHttp {
		log.Info().Msg("Ordinary HTTP Will be blocked (flag --no-http set)")
	}

	_, err := proxy.StartProxy(*addr, *verbosity, nil, "CLI "+GitCommit, *blockHttp, *networkConfigPath, *tunnelConfig)
	if err != nil {
		if errors.Is(err, proxy.ErrGenerated) {
			return
		}

		log.Fatal().Err(err).Msg("failed to start proxy")
		return
	}

	<-make(chan bool)
}
