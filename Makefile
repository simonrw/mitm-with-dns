all: transparent-endpoints init

transparent-endpoints: $(wildcard cmd/transparent-endpoints/*.go internal/dns/*.go internal/docker/*.go) init
	go build -o $@ cmd/transparent-endpoints/main.go

init: cmd/init/main.go
	go build -o $@ $<
