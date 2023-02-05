package main

import (
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
	// set up logging
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	ready := make(chan struct{}, 1)
	logger.Info().Msg("running DNS server in the background")
	go dns.RunServer(ready)
	logger.Info().Msg("waiting for DNS server to be ready")
	<-ready
	logger.Info().Msg("running docker container")
	docker.Run()
}
