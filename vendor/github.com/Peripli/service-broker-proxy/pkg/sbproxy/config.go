package sbproxy

import (
	"github.com/Peripli/service-broker-proxy/pkg/env"
	"github.com/Peripli/service-broker-proxy/pkg/osb"
	"github.com/Peripli/service-broker-proxy/pkg/sbproxy/server"
	"github.com/Peripli/service-broker-proxy/pkg/sm"
)

func NewConfigFromEnv(env env.Environment) (*Configuration, error) {
	appConfig, err := server.NewConfig(env)
	if err != nil {
		return nil, err
	}
	smConfig, err := sm.NewConfig(env)
	if err != nil {
		return nil, err
	}
	osbConfig, err := osb.NewConfig(smConfig)
	if err != nil {
		return nil, err
	}

	config := &Configuration{
		App: appConfig,
		Osb: osbConfig,
		Sm:  smConfig,
	}

	return config, nil
}

type Configuration struct {
	App *server.AppConfiguration
	Osb *osb.ClientConfiguration
	Sm  *sm.ClientConfiguration
}

func (c *Configuration) Validate() error {

	if err := c.App.Validate(); err != nil {
		return err
	}
	if err := c.Osb.Validate(); err != nil {
		return err
	}
	if err := c.Sm.Validate(); err != nil {
		return err
	}
	return nil
}
