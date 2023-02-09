package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/simonrw/transparent-endpoints/internal/dns"
	"github.com/simonrw/transparent-endpoints/internal/docker"
	"github.com/simonrw/transparent-endpoints/internal/http"
	"go.uber.org/zap"
)

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

			logger.Debugw("found ip address", "ip address", ip, "interface", i)
			res = append(res, ip)

		}
	}

	return res, nil
}

var logger *zap.SugaredLogger

func main() {
	rawLogger := zap.Must(zap.NewDevelopment())
	logger = rawLogger.Sugar()
	defer logger.Sync()

	imageNameFlag := flag.String("image", "", "name of the user image")
	flag.Parse()

	if *imageNameFlag == "" {
		logger.Fatal("no image name given")
	}

	ipAddresses, err := getIPAddresses()
	if err != nil {
		logger.Fatalw("getting IP addresses", "error", err)
	}
	logger.Infow("got ip addresses", "ip addresses", ipAddresses)

	var ready sync.WaitGroup

	logger.Info("running DNS server in the background")

	numGoroutines := 3
	stop := make(chan struct{}, numGoroutines)
	var finished sync.WaitGroup
	ready.Add(1)
	go dns.RunServer(logger, &ready, ipAddresses, stop, &finished)

	// start http server
	ready.Add(1)
	go http.RunServer(logger, &ready, stop, &finished)

	logger.Info("waiting for servers to be ready")
	ready.Wait()

	logger.Info("running docker container")
	containerExited := make(chan struct{})
	go docker.Run(logger, *imageNameFlag, ipAddresses, stop, &finished, containerExited)

	// handle ctrl-c
	sig := make(chan os.Signal)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	shutdown := func() {
		logger.Info("shutting down goroutines")
		for i := 0; i < numGoroutines; i++ {
			stop <- struct{}{}
		}
	}

	select {
	case <-sig:
		logger.Debug("ctrl-c")
		shutdown()
	case <-containerExited:
		logger.Debug("container exited early")
		shutdown()
	}

	// wait for goroutines to cleanup
	logger.Info("waiting for goroutines to shut down")
	finished.Wait()
	logger.Info("ending main goroutine")
}
