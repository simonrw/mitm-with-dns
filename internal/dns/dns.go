package dns

import (
	"net"
	"strings"
	"sync"

	"github.com/miekg/dns"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var logger zerolog.Logger

func init() {
	logger = log.With().Str("module", "dns").Logger()
}

type dnsHandler struct {
	ipAddresses []net.IP
}

func isInternalRequest(name string) bool {
	strippedName := strings.TrimRight(name, ".")
	isAWS := strings.HasSuffix(strippedName, "amazonaws.com")
	isLocal := strippedName == "localhost" || strippedName == "localhost.localstack.cloud"
	return isAWS || isLocal
}

func isExternalRequest(name string) bool {
	return !isInternalRequest(name)
}

func (h *dnsHandler) ServeDNS(w dns.ResponseWriter, r *dns.Msg) {
	m := new(dns.Msg)
	m.SetReply(r)
	m.Id = r.Id
	q := r.Question[0]
	switch q.Qtype {
	case dns.TypeA:
		logger.Info().Msg("got A record query type")

		// if not a request that we care about, send to upstream
		if isExternalRequest(q.Name) {
			m2, err := dns.Exchange(r, "8.8.8.8:53")
			if err != nil {
				log.Warn().Msg("sending upstream request")
				// TODO: deprecated function
				dns.HandleFailed(w, r)
				return
			}
			w.WriteMsg(m2)
			return
		}

		for _, addr := range h.ipAddresses {
			if addr.IsLoopback() {
				continue
			}
			rr := &dns.A{
				Hdr: dns.RR_Header{
					Name:   q.Name,
					Rrtype: dns.TypeA,
					Class:  dns.ClassINET,
					Ttl:    1500,
				},
				A: addr,
			}
			m.Answer = append(m.Answer, rr)
		}
		logger.Info().Any("response", m).Msg("returning response")
		w.WriteMsg(m)
	case dns.TypeAAAA:
		logger.Info().Msg("got AAAA record query type")
		for _, addr := range h.ipAddresses {
			if addr.IsLoopback() {
				continue
			}
			rr := &dns.AAAA{
				Hdr: dns.RR_Header{
					Name:   q.Name,
					Rrtype: dns.TypeAAAA,
					Class:  dns.ClassINET,
					Ttl:    1500,
				},
				AAAA: addr,
			}
			m.Answer = append(m.Answer, rr)
			logger.Info().Any("response", m).Msg("returning response")
			w.WriteMsg(m)
		}
	default:
		log.Warn().Uint16("qtype", r.Question[0].Qtype).Msg("unhandled query type")
	}

	// return error
}

func RunServer(ready *sync.WaitGroup, ipAddresses []net.IP, stop chan struct{}, complete *sync.WaitGroup) {
	complete.Add(1)
	defer complete.Done()
	logger.Info().Msg("running DNS server")

	handler := &dnsHandler{ipAddresses}

	addr := "127.0.0.25:5300"
	server := &dns.Server{
		Addr:    addr,
		Net:     "udp",
		Handler: handler,
	}
	go server.ListenAndServe()
	defer server.Shutdown()

	logger.Info().Str("address", addr).Msg("DNS server ready")
	ready.Done()

	logger.Info().Msg("waiting for shutdown signal")
	<-stop
	logger.Info().Msg("shutdown signal received")
}
