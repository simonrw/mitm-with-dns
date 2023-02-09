package http

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"go.uber.org/zap"
)

var logger *zap.SugaredLogger

func createRouter() *http.ServeMux {
	router := http.NewServeMux()
	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		logger := logger.With("path", r.URL.Path)
		logger.Debug("got request")
		fmt.Fprintf(w, "ok")
	})
	return router
}

func RunServer(l *zap.SugaredLogger, ready *sync.WaitGroup, stop chan struct{}, finished *sync.WaitGroup) {
	logger = l
	logger.Info("running HTTP server")
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
				logger.Warnf("failed to start http listener", "err", err)
			}
		}
	}()
	go func() {
		if err := httpsServer.ListenAndServeTLS("./_wildcard.amazonaws.com+1.pem", "./_wildcard.amazonaws.com+1-key.pem"); err != nil {
			if err != http.ErrServerClosed {
				logger.Warnf("failed to start https listener", "err", err)
			}
		}
	}()
	defer httpServer.Shutdown(context.TODO())

	logger.Info("http server running")
	ready.Done()

	<-stop
	logger.Info("http server closed")
}
