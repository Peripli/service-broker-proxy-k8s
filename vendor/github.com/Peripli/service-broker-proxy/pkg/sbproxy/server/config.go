package server

import (
	"github.com/Peripli/service-broker-proxy/pkg/env"
	"github.com/pkg/errors"
)

type AppConfiguration struct {
	Port       int
	LogLevel   string
	LogFormat  string
	TimeoutSec int
	TLSKey     string
	TLSCert    string
	Host       string
}

type Settings struct {
	App *AppConfiguration
}

func (c *AppConfiguration) Validate() error {
	if c.Port == 0 {
		return errors.New("application configuration Port missing")
	}
	if len(c.LogLevel) == 0 {
		return errors.New("application configuration LogLevel missing")
	}
	if len(c.LogFormat) == 0 {
		return errors.New("application configuration LogFormat missing")
	}
	if c.TimeoutSec == 0 {
		return errors.New("application configuration TimeoutSec missing")
	}
	if (c.TLSCert != "" || c.TLSKey != "") &&
		(c.TLSCert == "" || c.TLSKey == "") {
		return errors.New("application configuration both TLSCert and TLSKey must be provided to use TLS")
	}

	if len(c.Host) == 0 {
		return errors.New("application configuration Host missing")
	}
	return nil
}

func DefaultConfig() *AppConfiguration {
	return &AppConfiguration{
		Port:       8080,
		LogLevel:   "debug",
		LogFormat:  "text",
		TimeoutSec: 15,
		TLSKey:     "",
		TLSCert:    "",
		Host:       "",
	}
}

func NewConfig(env env.Environment) (*AppConfiguration, error) {
	config := DefaultConfig()
	appConfig := &Settings{
		App: config,
	}

	if err := env.Unmarshal(appConfig); err != nil {
		return nil, errors.Wrap(err, "error unmarshaling app configuration")
	}
	return config, nil
}
