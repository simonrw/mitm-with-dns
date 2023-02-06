FROM golang as builder

ENV CGO_ENABLED=0

WORKDIR /app

COPY go.mod ./
COPY go.sum ./

RUN go mod download

COPY cmd ./cmd
COPY internal ./internal
COPY Makefile ./

RUN make -j

FROM alpine

WORKDIR /app

COPY --from=builder /app/transparent-endpoints ./transparent-endpoints
COPY --from=builder /app/init ./init
COPY *.pem ./
COPY ca ./ca


ENTRYPOINT ["/app/transparent-endpoints"]
