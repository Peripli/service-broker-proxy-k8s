#########################################################
# Build the sources and provide the result in a multi stage
# docker container. The alpine build image has to match
# the alpine image in the referencing runtime container.
#########################################################
FROM golang:1.10.1-alpine3.7 AS build-env

# We need so that dep can fetch it's dependencies
RUN apk --no-cache add git


# Directory in workspace
RUN mkdir -p "/go/src/github.com/Peripli/service-broker-proxy-k8s"
COPY . "/go/src/github.com/Peripli/service-broker-proxy-k8s"
WORKDIR "/go/src/github.com/Peripli/service-broker-proxy-k8s"

# Install dep, dependencies and build the main (without any testing at the moment)
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go get github.com/golang/dep/cmd/dep && \
    rm -rf vendor && \
    dep ensure -vendor-only -v && \
    go test && \
    go build -o /main .

#########################################################
# Build the runtime container
#########################################################
FROM alpine:3.7

# required to use x.509 certs (HTTPS)
RUN apk update && apk add ca-certificates

# ENV KUBERNETES_MASTER https://api.s3.cpet.k8s.sapcloud.io
# ENV KUBECONFIG /app/kubeconfig.yaml

WORKDIR /app
COPY --from=build-env /main .
COPY application.yaml ./
ENTRYPOINT ./main
EXPOSE 8081
