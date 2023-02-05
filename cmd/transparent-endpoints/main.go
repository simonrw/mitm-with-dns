package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/simonrw/transparent-endpoints/internal/dns"
	"github.com/simonrw/transparent-endpoints/internal/docker"
)

var logger zerolog.Logger

func init() {
	logger = log.With().Str("module", "main").Logger()
}

func main() {
	imageNameFlag := flag.String("image", "", "name of the user image")
	flag.Parse()

	if *imageNameFlag == "" {
		logger.Fatal().Msg("no image name given")
	}

	// set up logging
	zerolog.SetGlobalLevel(zerolog.DebugLevel)

	ready := make(chan struct{}, 1)
	logger.Info().Msg("running DNS server in the background")

	stop := make(chan struct{})
	dnsComplete := make(chan struct{})
	go dns.RunServer(ready, stop, dnsComplete)
	logger.Info().Msg("waiting for DNS server to be ready")
	<-ready

	logger.Info().Msg("running docker container")
	dockerComplete := make(chan struct{})
	go docker.Run(*imageNameFlag, stop, dockerComplete)

	// handle ctrl-c
	sig := make(chan os.Signal)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	<-sig
	logger.Info().Msg("shutting down goroutines")
	for i := 0; i < 2; i++ {
		stop <- struct{}{}
	}

	// wait for goroutines to cleanup
	<-dnsComplete
	<-dockerComplete
	logger.Info().Msg("ending main goroutine")

}
