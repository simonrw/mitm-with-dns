package http

import (
	"context"
	"net/http"
	"sync"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var logger zerolog.Logger

func init() {
	logger = log.With().Str("module", "http").Logger()
}

func RunServer(ready *sync.WaitGroup, stop chan struct{}, finished *sync.WaitGroup) {
	logger.Info().Msg("running HTTP server")
	finished.Add(1)
	defer finished.Done()

	server := &http.Server{
		Addr: "0.0.0.0:80",
	}
	go server.ListenAndServe()
	defer server.Shutdown(context.TODO())

	logger.Info().Msg("http server running")
	ready.Done()

	<-stop
	logger.Info().Msg("http server closed")
}
