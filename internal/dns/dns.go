package dns

import (
	"net"
	"strings"
	"sync"

	"github.com/miekg/dns"
	"github.com/rs/zerolog/log"
	"go.uber.org/zap"
)

var logger *zap.SugaredLogger

func init() {
	rawLogger := zap.Must(zap.NewDevelopment())
	logger = rawLogger.Sugar()
	defer logger.Sync()
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
	remoteAddress := w.RemoteAddr()
	switch addr := remoteAddress.(type) {
	case *net.UDPAddr:
		logger.Debugw("got dns request", "remote address", addr.IP.To4())
	default:
		logger.Warn("unhandled remote address")
		handleRefused(w, r)
		return
	}

	m := new(dns.Msg)
	m.SetReply(r)
	m.Id = r.Id
	q := r.Question[0]
	switch q.Qtype {
	case dns.TypeA:
		fallthrough
	case dns.TypeAAAA:
		logger.Info("got A/AAAA record query type")

		// if not a request that we care about, send to upstream
		if isExternalRequest(q.Name) {
			m2, err := dns.Exchange(r, "8.8.8.8:53")
			if err != nil {
				log.Warn().Msg("sending upstream request")
				// TODO: deprecated function
				handleRefused(w, r)
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
					Rrtype: q.Qtype,
					Class:  dns.ClassINET,
					Ttl:    1500,
				},
				A: addr,
			}
			m.Answer = append(m.Answer, rr)
		}
		logger.Infow("returning response", "response", m)
		w.WriteMsg(m)
	default:
		logger.Warnw("unhandled query type", "qtype", r.Question[0].Qtype)
		handleRefused(w, r)
	}

}

func handleRefused(w dns.ResponseWriter, r *dns.Msg) {
	m := new(dns.Msg)
	m.SetRcode(r, dns.RcodeRefused)
	w.WriteMsg(m)
}

func RunServer(l *zap.SugaredLogger, ready *sync.WaitGroup, ipAddresses []net.IP, stop chan struct{}, complete *sync.WaitGroup) {
	logger = l

	complete.Add(1)
	defer complete.Done()
	logger.Info("running DNS server")

	handler := &dnsHandler{ipAddresses}

	addr := "0.0.0.0:53"
	server := &dns.Server{
		Addr:    addr,
		Net:     "udp",
		Handler: handler,
	}
	go server.ListenAndServe()
	defer server.Shutdown()

	logger.Infow("DNS server ready", "address", addr)
	ready.Done()

	logger.Info("waiting for shutdown signal")
	<-stop
	logger.Info("shutdown signal received")
}
