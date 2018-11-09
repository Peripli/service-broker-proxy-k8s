package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/Peripli/service-broker-proxy-k8s/k8s"
	"github.com/Peripli/service-broker-proxy-k8s/load_plugin"
	"github.com/Peripli/service-broker-proxy/pkg/sbproxy"

	"github.com/spf13/pflag"
)

type ArrayValue []string

func (a *ArrayValue) String() string {
	return strings.Join(*a, ", ")
}

func (a *ArrayValue) Set(value string) error {
	*a = append(*a, value)
	return nil
}

func (a *ArrayValue) Type() string {
	return "string"
}

var pluginList ArrayValue

func main() {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	env := sbproxy.DefaultEnv(func(set *pflag.FlagSet) {
		k8s.CreatePFlagsForK8SClient(set)
		set.Var(&pluginList, "plugin", "list of plugins")
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
	if err = load_plugin.LoadPlugins(pluginList, proxyBuilder.API); err != nil {
		panic(fmt.Errorf("error loading plugins: %v", err))
	}
	proxy := proxyBuilder.Build()

	proxy.Run()
}
