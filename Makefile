LS_IMAGE_NAME := ls
TEST_IMAGE_NAME := mitm-with-dns
__UNAME := $(shell uname -m)

all: transparent-endpoints init

transparent-endpoints: $(wildcard cmd/transparent-endpoints/*.go internal/dns/*.go internal/docker/*.go) init Makefile
	CGO_ENABLED=0 go build -o $@ cmd/transparent-endpoints/main.go

init: cmd/init/main.go Makefile
ifeq ($(__UNAME),x86_64)
	GOOS=linux CGO_ENABLED=0 go build -o $@ $<
else
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o $@ $<
endif

.PHONY: run
run: transparent-endpoints init _wildcard.amazonaws.com.pem dockerbuild
	$(MAKE) dockerbuild
	./transparent-endpoints -image $(TEST_IMAGE_NAME)

.PHONY: dockerrun
dockerrun: build
	docker run --rm -it --name ls -v /var/run/docker.sock:/var/run/docker.sock $(LS_IMAGE_NAME) -image $(TEST_IMAGE_NAME)

_wildcard.amazonaws.com.pem:
	mkcert '*.amazonaws.com'

.PHONY: dockerbuild
dockerbuild:
	$(MAKE) -C userimage build IMAGENAME=$(TEST_IMAGE_NAME)

build:
	docker build -t $(LS_IMAGE_NAME) .
