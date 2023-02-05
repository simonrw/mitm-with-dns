package dns

import (
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var logger zerolog.Logger

func init() {
	logger = log.With().Str("module", "dns").Logger()
}

type server struct {
}

func RunServer(ready chan struct{}) {
	logger.Info().Msg("running DNS server")
	logger.Info().Msg("DNS server ready")
	time.Sleep(5 * time.Second)
	ready <- struct{}{}
}
