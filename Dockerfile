#########################################################
# Build the sources and provide the result in a multi stage
# docker container.
#########################################################
FROM golang:1.10.0 AS build-env
ENV GOBIN /go/bin

RUN mkdir /go/src/app
ADD . /go/src/app
WORKDIR /go/src/app

# Install and update tools for static code checks and tests
RUN go get github.com/alecthomas/gometalinter && \
    go get github.com/GoASTScanner/gas/cmd/gas/...
RUN gometalinter --install --update

# Fetch dependencies
RUN go get -u github.com/golang/dep/...
RUN dep ensure

# Run static code checks and tests
RUN /go/bin/gometalinter  ./...
RUN /go/bin/gas -skip=vendor ./...
RUN go test

# Trigger the build
RUN go build -o /main .

#########################################################
# Build the runtime container
#########################################################
FROM alpine
WORKDIR /app
COPY --from=build-env /main /app/
ENTRYPOINT /app/main