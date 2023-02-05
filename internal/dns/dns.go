package dns

import (
	"net"

	"github.com/miekg/dns"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var logger zerolog.Logger

func init() {
	logger = log.With().Str("module", "dns").Logger()
}

type dnsHandler struct{}

func (h *dnsHandler) ServeDNS(w dns.ResponseWriter, r *dns.Msg) {
	m := new(dns.Msg)
	m.Id = r.Id
	q := r.Question[0]
	switch q.Qtype {
	case dns.TypeA:
		logger.Info().Msg("got A record query type")
		a := net.ParseIP("7.7.7.7")
		rr := &dns.A{
			Hdr: dns.RR_Header{
				Name:   q.Name,
				Rrtype: dns.TypeA,
				Class:  dns.ClassINET,
				Ttl:    1500,
			},
			A: a,
		}
		m.Answer = append(m.Answer, rr)
		logger.Info().Any("response", m).Msg("returning response")
		w.WriteMsg(m)
	default:
		log.Warn().Uint16("qtype", r.Question[0].Qtype).Msg("unhandled query type")
	}

	// return error
}

func RunServer(ready chan struct{}, stop chan struct{}, complete chan struct{}) {
	logger.Info().Msg("running DNS server")

	handler := &dnsHandler{}

	server := &dns.Server{
		Addr:    "0.0.0.0:8053",
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
