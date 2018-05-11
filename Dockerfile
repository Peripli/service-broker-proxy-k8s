#########################################################
# The gometalinter currently does not work properly on
# alpine images, so we add an additional layer with a
# debian image just for the gometalinter.
# Github issue: https://github.com/alecthomas/gometalinter/issues/149
#########################################################
FROM golang:1.10.1 AS linter-env

# Install linters
RUN go get -u golang.org/x/lint/golint && \
    go get github.com/GoASTScanner/gas/cmd/gas/... && \
    go get github.com/alecthomas/gometalinter && \
    gometalinter --install --update

# Directory in workspace
RUN mkdir -p "/go/src/github.com/Peripli/service-broker-proxy-k8s"
COPY . "/go/src/github.com/Peripli/service-broker-proxy-k8s"

RUN /go/bin/gometalinter --deadline=300s --disable=gotype  /go/src/github.com/Peripli/service-broker-proxy-k8s

#########################################################
# Build the sources and provide the result in a multi stage
# docker container. The alpine build image has to match
# the alpine image in the referencing runtime container.
#########################################################
FROM golang:1.10.1-alpine3.7 AS build-env

# Directory in workspace
RUN mkdir -p "/go/src/github.com/Peripli/service-broker-proxy-k8s"
COPY . "/go/src/github.com/Peripli/service-broker-proxy-k8s"
WORKDIR "/go/src/github.com/Peripli/service-broker-proxy-k8s"

# Run tests and build the main (without any testing at the moment)
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go test -ginkgo.v && \
    go build -o /main .

#########################################################
# Build the runtime container
#########################################################
FROM alpine:3.7

# required to use x.509 certs (HTTPS)
RUN apk update && apk add ca-certificates

WORKDIR /app
COPY --from=build-env /main .
COPY application.yaml ./
ENTRYPOINT ./main
EXPOSE 8081
