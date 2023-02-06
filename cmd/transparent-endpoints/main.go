package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/simonrw/transparent-endpoints/internal/dns"
	"github.com/simonrw/transparent-endpoints/internal/docker"
)

var logger zerolog.Logger

func init() {
	logger = log.With().Str("module", "main").Logger()
}

func getIPAddresses() ([]net.IP, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("getting interfaces: %w", err)
	}
	var res []net.IP
	for _, i := range ifaces {
		addrs, err := i.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			default:
				continue
			}

			logger.Debug().Any("ip address", ip).Any("interface", i).Msg("found ip address")
			res = append(res, ip)

		}
	}

	return res, nil
}

func main() {
	imageNameFlag := flag.String("image", "", "name of the user image")
	flag.Parse()

	if *imageNameFlag == "" {
		logger.Fatal().Msg("no image name given")
	}

	ipAddresses, err := getIPAddresses()
	if err != nil {
		logger.Fatal().Err(err).Msg("getting IP addresses")
	}
	logger.Info().Any("ip addresses", ipAddresses).Msg("got IP addresses")

	// set up logging
	zerolog.SetGlobalLevel(zerolog.DebugLevel)

	ready := make(chan struct{}, 1)
	logger.Info().Msg("running DNS server in the background")

	stop := make(chan struct{})
	dnsComplete := make(chan struct{})
	go dns.RunServer(ready, ipAddresses, stop, dnsComplete)
	logger.Info().Msg("waiting for DNS server to be ready")
	<-ready

	logger.Info().Msg("running docker container")
	dockerComplete := make(chan struct{})
	go docker.Run(*imageNameFlag, ipAddresses, stop, dockerComplete)

	// handle ctrl-c
	sig := make(chan os.Signal)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	<-sig
	logger.Info().Msg("shutting down goroutines")
	for i := 0; i < 2; i++ {
		stop <- struct{}{}
	}

	// wait for goroutines to cleanup
	<-dnsComplete
	<-dockerComplete
	logger.Info().Msg("ending main goroutine")

}
