package main

import (
	"context"
	"fmt"

	"github.com/Peripli/service-broker-proxy-k8s/k8s"
	"github.com/Peripli/service-broker-proxy-k8s/version"
	"github.com/Peripli/service-broker-proxy/pkg/sbproxy"

	"github.com/spf13/pflag"
)

func main() {
	version.Log()

	ctx, cancel := context.WithCancel(context.Background())
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

	proxyBuilder := sbproxy.New(ctx, cancel, env, platformClient)
	proxy := proxyBuilder.Build()

	proxy.Run()
}
