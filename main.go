package main

import (
	"github.com/Peripli/service-broker-proxy/pkg/sbproxy"
	"github.com/sirupsen/logrus"
	"github.com/Peripli/service-broker-proxy/pkg/env"
	"github.com/service-broker-proxy-k8s/platform"
)
const env_prefix = "PROXY"

func main() {

	env := env.Default(env_prefix)
	if err := env.Load(); err != nil {
		logrus.WithError(err).Fatal("Error loading environment")
	}

	proxyConfig, err := sbproxy.NewConfigFromEnv(env)
	if err != nil {
		logrus.WithError(err).Fatal("Error loading configuration")
	}

	platformClient, err := platform.NewClient()
	if err != nil {
		logrus.WithError(err).Fatal("Error creating platform client")
	}

	sbProxy, err := sbproxy.New(proxyConfig, platformClient)
	if err != nil {
		logrus.WithError(err).Fatal("Error creating SB Proxy")
	}

	//sbProxy.Use(middleware.BasicAuth(platformConfig.Reg.User, platformConfig.Reg.Password))

	sbProxy.Run()
}