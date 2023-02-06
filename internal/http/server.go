package http

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var logger zerolog.Logger

func init() {
	logger = log.With().Str("module", "http").Logger()
}

func createRouter() *http.ServeMux {
	router := http.NewServeMux()
	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "ok")
	})
	return router
}

func RunServer(ready *sync.WaitGroup, stop chan struct{}, finished *sync.WaitGroup) {
	logger.Info().Msg("running HTTP server")
	finished.Add(1)
	defer finished.Done()

	router := createRouter()
	httpServer := &http.Server{
		Addr:    "0.0.0.0:80",
		Handler: router,
	}
	httpsServer := &http.Server{
		Addr:    "0.0.0.0:443",
		Handler: router,
	}
	go func() {
		if err := httpServer.ListenAndServe(); err != nil {
			if err != http.ErrServerClosed {
				logger.Warn().Err(err).Msg("failed to start HTTP listener")
			}
		}
	}()
	go func() {
		if err := httpsServer.ListenAndServeTLS("./_wildcard.amazonaws.com+1.pem", "./_wildcard.amazonaws.com+1-key.pem"); err != nil {
			if err != http.ErrServerClosed {
				logger.Warn().Err(err).Msg("failed to start HTTPS listener")
			}
		}
	}()
	defer httpServer.Shutdown(context.TODO())

	logger.Info().Msg("http server running")
	ready.Done()

	<-stop
	logger.Info().Msg("http server closed")
}
