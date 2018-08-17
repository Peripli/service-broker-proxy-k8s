package main

import (
	"fmt"

	"github.com/Peripli/service-broker-proxy-k8s/k8s"
	"github.com/Peripli/service-broker-proxy/pkg/middleware"
	"github.com/Peripli/service-broker-proxy/pkg/sbproxy"
	"github.com/Peripli/service-manager/pkg/util"

	"github.com/spf13/pflag"
)

func main() {
	ctx, cancel := util.HandleInterrupts()
	defer cancel()

	env := sbproxy.DefaultEnv(func(set *pflag.FlagSet) {
		k8s.CreatePFlagsForK8SClient(set)
	})

	platformConfig, err := k8s.NewConfig(env)
	if err != nil {
		panic(fmt.Errorf("error loading config: %s", err))
	}

	platformClient, err := k8s.NewClient(platformConfig)
	if err != nil {
		panic(fmt.Errorf("error creating K8S client: %s", err))
	}

	proxyBuilder := sbproxy.New(ctx, env, platformClient)
	proxy := proxyBuilder.Build()

	proxy.Server.Use(middleware.BasicAuth(platformConfig.Reg.User, platformConfig.Reg.Password))

	proxy.Run()
}
