package docker

import (
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var logger zerolog.Logger

func init() {
	logger = log.With().Str("module", "docker").Logger()
}

func Run(stop chan struct{}, complete chan struct{}) {
	logger.Info().Msg("running docker container")
	logger.Info().Msg("waiting for shutdown signal")
	<-stop
	logger.Info().Msg("shutting down docker container")
	complete <- struct{}{}
}
