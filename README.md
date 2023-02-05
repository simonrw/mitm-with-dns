# MITM with DNS

This project is going to prove out custom HTTPS certificates and transparent DNS.

## Outline

* Run a docker container with the dns server configured to return `localhost` for `*.amazonaws.com`
* use `mkcert` to generate a certificate for `*.amazonaws.com` which is hosted by a local web server
* install the `mkcert` ca certificate into the container for request trust.


## Implementation


