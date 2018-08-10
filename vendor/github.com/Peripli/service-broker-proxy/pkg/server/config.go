package server

import (
	"time"

	"github.com/Peripli/service-manager/pkg/env"
	"github.com/pkg/errors"
)

// DefaultConfig builds the default Config
func DefaultConfig() *Config {
	return &Config{
		Port:         0,
		LogLevel:     "debug",
		LogFormat:    "text",
		Timeout:      5 * time.Second,
		ResyncPeriod: 10 * time.Minute,
		TLSKey:       "",
		TLSCert:      "",
		Host:         "",
	}
}

// NewConfig builds an Config from the given Environment
func NewConfig(env env.Environment) (*Config, error) {
	config := struct {
		Server *Config
	}{Server: DefaultConfig()}

	if err := env.Unmarshal(&config); err != nil {
		return nil, errors.Wrap(err, "error unmarshaling app configuration")
	}
	return config.Server, nil
}

// Config type holds application config properties
type Config struct {
	Port         int
	LogLevel     string
	LogFormat    string
	Timeout      time.Duration
	ResyncPeriod time.Duration
	TLSKey       string
	TLSCert      string
	Host         string
}

// Validate validates the configuration and returns appropriate errors in case it is invalid
func (c *Config) Validate() error {
	if c.Port == 0 {
		return errors.New("application configuration Port missing")
	}
	if len(c.LogLevel) == 0 {
		return errors.New("application configuration LogLevel missing")
	}
	if len(c.LogFormat) == 0 {
		return errors.New("application configuration LogFormat missing")
	}
	if c.Timeout == 0 {
		return errors.New("application configuration RequestTimeout missing")
	}
	if c.ResyncPeriod == 0 {
		return errors.New("application configuration RequestTimeout missing")
	}
	if !tlsConfigOK(c.TLSCert, c.TLSKey) {
		return errors.New("application configuration both TLSCert and TLSKey must be provided to use TLS")
	}
	if len(c.Host) == 0 {
		return errors.New("application configuration Host missing")
	}
	return nil
}

func tlsConfigOK(TLSCert, TLSKey string) bool {
	return (TLSCert == "" && TLSKey == "") || (TLSCert != "" && TLSKey != "")
}
