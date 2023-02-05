package main

import "github.com/simonrw/transparent-endpoints/internal/dns"
import "github.com/simonrw/transparent-endpoints/internal/docker"

func main() {
	go dns.RunServer()
	docker.Run()
}
