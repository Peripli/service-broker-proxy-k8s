package config

import (
	"github.com/Peripli/service-broker-proxy/pkg/osb"
	"github.com/Peripli/service-broker-proxy/pkg/server"
	"github.com/Peripli/service-broker-proxy/pkg/sm"
	"github.com/Peripli/service-manager/pkg/env"
	"github.com/spf13/pflag"
)

// New builds an sbproxy.Config from the specified Environment
func New(env env.Environment) (*Config, error) {
	serverConfig, err := server.NewConfig(env)
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

	config := &Config{
		Server: serverConfig,
		Osb:    osbConfig,
		Sm:     smConfig,
	}

	return config, nil
}

// AddPFlags adds the pflags needed for the the proxy default config to the provided flag set.
func AddPFlags(set *pflag.FlagSet) {
	defaultCfg := &Config{
		Server: server.DefaultConfig(),
		Sm:     sm.DefaultConfig(),
		Osb:    osb.DefaultConfig(),
	}

	env.CreatePFlags(set, defaultCfg)
}

// Config type holds all config properties for the sbproxy
type Config struct {
	Server *server.Config
	Sm     *sm.Config
	Osb    *osb.ClientConfig `structs:"-"`
}

// Validate validates the configuration and returns appropriate errors in case it is invalid
func (c *Config) Validate() error {
	validatable := []interface{ Validate() error }{c.Server, c.Osb, c.Sm}

	for _, item := range validatable {
		if err := item.Validate(); err != nil {
			return err
		}
	}
	return nil
}
