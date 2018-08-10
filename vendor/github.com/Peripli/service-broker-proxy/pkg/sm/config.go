package sm

import (
	"time"

	"github.com/Peripli/service-manager/pkg/env"
	"github.com/pkg/errors"
)

// DefaultConfig builds a default Service Manager Config
func DefaultConfig() *Config {
	return &Config{
		User:              "",
		Password:          "",
		Host:              "",
		RequestTimeout:    5 * time.Second,
		CreateFunc:        NewClient,
		SkipSslValidation: false,
	}
}

// NewConfig builds a Service Manager Config from the provided Environment
func NewConfig(env env.Environment) (*Config, error) {
	config := struct {
		Sm *Config
	}{DefaultConfig()}

	if err := env.Unmarshal(&config); err != nil {
		return nil, errors.Wrap(err, "error unmarshaling SM configuration")
	}

	return config.Sm, nil
}

// Config type holds SM Client config properties
type Config struct {
	User              string
	Password          string
	Host              string
	OsbAPI            string
	RequestTimeout    time.Duration
	SkipSslValidation bool

	CreateFunc func(config *Config) (Client, error)
}

// Validate validates the configuration and returns appropriate errors in case it is invalid
func (c *Config) Validate() error {
	if len(c.User) == 0 {
		return errors.New("SM configuration User missing")
	}
	if len(c.Password) == 0 {
		return errors.New("SM configuration Password missing")
	}
	if len(c.Host) == 0 {
		return errors.New("SM configuration Host missing")
	}
	if len(c.OsbAPI) == 0 {
		return errors.New("SM configuration OSB API missing")
	}
	if c.RequestTimeout == 0 {
		return errors.New("SM configuration RequestTimeout missing")
	}
	if c.CreateFunc == nil {
		return errors.New("SM configuration CreateFunc missing")
	}
	return nil
}
