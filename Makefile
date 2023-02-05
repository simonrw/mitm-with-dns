TEST_IMAGE_NAME := mitm-with-dns

all: transparent-endpoints init

transparent-endpoints: $(wildcard cmd/transparent-endpoints/*.go internal/dns/*.go internal/docker/*.go) init Makefile
	CGO_ENABLED=0 go build -o $@ cmd/transparent-endpoints/main.go

init: cmd/init/main.go Makefile
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o $@ $<

.PHONY: run

run: transparent-endpoints init _wildcard.amazonaws.com.pem
	$(MAKE) dockerbuild
	./transparent-endpoints -image $(TEST_IMAGE_NAME)

_wildcard.amazonaws.com.pem:
	mkcert '*.amazonaws.com'

.PHONY: dockerbuild
dockerbuild:
	$(MAKE) -C userimage build IMAGENAME=$(TEST_IMAGE_NAME)