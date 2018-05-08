package main

import (
	"github.com/Peripli/service-broker-proxy/pkg/env"
	"github.com/Peripli/service-broker-proxy/pkg/sbproxy"
	"github.com/sirupsen/logrus"
)

const envPrefix = "PROXY"

func main() {

	env := env.Default(envPrefix)
	if err := env.Load(); err != nil {
		logrus.WithError(err).Fatal("Error loading environment")
	}

	proxyConfig, err := sbproxy.NewConfigFromEnv(env)
	if err != nil {
		logrus.WithError(err).Fatal("Error loading configuration")
	}

	platformClient, err := NewClient()
	if err != nil {
		logrus.WithError(err).Fatal("Error creating platform client")
	}

	sbProxy, err := sbproxy.New(proxyConfig, platformClient)
	if err != nil {
		logrus.WithError(err).Fatal("Error creating SB Proxy")
	}

	sbProxy.Run()
}
