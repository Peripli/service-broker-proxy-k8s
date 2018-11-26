#########################################################
# Build the sources and provide the result in a multi stage
# docker container. The alpine build image has to match
# the alpine image in the referencing runtime container.
#########################################################
FROM golang:1.11.2-alpine3.8 AS builder

# We need so that dep can fetch it's dependencies
# RUN apk --no-cache add git
# RUN go get github.com/golang/dep/cmd/dep

# Directory in workspace
WORKDIR "/go/src/github.com/Peripli/service-broker-proxy-k8s"

# Copy dep files only and ensure dependencies are satisfied
# COPY Gopkg.lock Gopkg.toml ./
# RUN dep ensure -vendor-only -v

# Copy and build source code
COPY . ./
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -o /main main.go

########################################################
# Build the runtime container
########################################################
FROM alpine:3.8

# required to use x.509 certs (HTTPS)
RUN apk add --no-cache ca-certificates

WORKDIR /app

# Copy the executable file
COPY --from=builder /main /app/

EXPOSE 8081
ENTRYPOINT [ "./main" ]