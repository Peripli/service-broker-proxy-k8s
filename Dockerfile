#########################################################
# Build the sources and provide the result in a multi stage
# docker container.
#########################################################
FROM golang:1.10.0 AS build-env
ENV GOBIN /go/bin

RUN go get -u golang.org/x/lint/golint && \
    go get github.com/GoASTScanner/gas/cmd/gas/... && \
    go get github.com/alecthomas/gometalinter && \
    gometalinter --install --update && \
    go get -u github.com/golang/dep/...

RUN mkdir -p /go/src/github.com/Peripli/service-broker-proxy-k8s
    WORKDIR /go/src/github.com/Peripli/service-broker-proxy-k8s
    ADD . .

RUN dep ensure && \
    /go/bin/gometalinter --disable=gotype  ./... && \
    /go/bin/gas -skip=vendor ./... && \
    go test && \
    go build -o /main .

ENTRYPOINT /main

#########################################################
# Build the runtime container
#########################################################
FROM alpine
WORKDIR /app
COPY --from=build-env /main /app/
ENTRYPOINT /app/main
