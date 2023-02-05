package dns

import (
	"github.com/miekg/dns"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var logger zerolog.Logger

func init() {
	logger = log.With().Str("module", "dns").Logger()
}

type server struct {
}

func handleRedirect(w dns.ResponseWriter, r *dns.Msg) {
}

func serve(net, nanme, secret string, soreuseport bool) {
	server := &dns.Server{
		Addr:       ":8053",
		Net:        net,
		TsigSecret: nil,
		ReusePort:  soreuseport,
	}
	if err := server.ListenAndServe(); err != nil {
		logger.Fatal().Msg("failed to set up server")
	}
}

type dnsHandler struct {
}

func (h *dnsHandler) ServeDNS(w dns.ResponseWriter, r *dns.Msg) {
}

func RunServer(ready chan struct{}, stop chan struct{}, complete chan struct{}) {
	logger.Info().Msg("running DNS server")

	handler := &dnsHandler{}

	server := &dns.Server{
		Net:     "udp",
		Handler: handler,
	}
	go server.ListenAndServe()
	defer server.Shutdown()

	logger.Info().Msg("DNS server ready")
	ready <- struct{}{}

	logger.Info().Msg("waiting for shutdown signal")
	<-stop
	logger.Info().Msg("shutdown signal received")
	complete <- struct{}{}
}
