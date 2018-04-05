#########################################################
# Build the sources and provide the result in a multi stage
# docker container. The alpine build image has to match
# the alpine image in the referencing runtime container.
#########################################################
FROM golang:1.10.1-alpine3.7 AS build-env

# Set all env variables needed for go
ENV GOBIN /go/bin
ENV GOPATH /go

# We need so that dep can fetch it's dependencies
RUN apk --no-cache add git

# Install linters
RUN go get -u golang.org/x/lint/golint && \
    go get github.com/GoASTScanner/gas/cmd/gas/... && \
    go get github.com/alecthomas/gometalinter && \
    gometalinter --install --update

# Directory in workspace
RUN mkdir -p "/go/src/github.com/Peripli/service-broker-proxy-k8s"
COPY . "/go/src/github.com/Peripli/service-broker-proxy-k8s"
WORKDIR "/go/src/github.com/Peripli/service-broker-proxy-k8s"

# Install dep, dependencies, lint, run tests and build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go get github.com/golang/dep/cmd/dep && \
    dep ensure -v && \
#    /go/bin/gometalinter --disable=gotype  ./... && \
    CGO_ENABLED=0 /go/bin/gometalinter --disable=gotype  ./...  && \
    go test && \
    go build -o /main .

#########################################################
# Build the runtime container
#########################################################
FROM alpine:3.7
WORKDIR /app
COPY --from=build-env /main /app/

ENTRYPOINT ./main
EXPOSE 8080
