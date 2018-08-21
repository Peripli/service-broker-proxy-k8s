package sm

import (
	"time"

	"github.com/Peripli/service-manager/pkg/env"
	"github.com/pkg/errors"
)

// DefaultSettings builds a default Service Manager Settings
func DefaultSettings() *Settings {
	return &Settings{
		User:              "",
		Password:          "",
		Host:              "",
		RequestTimeout:    5 * time.Second,
		CreateFunc:        NewClient,
		SkipSSLValidation: false,
	}
}

// NewSettings builds a Service Manager Settings from the provided Environment
func NewSettings(env env.Environment) (*Settings, error) {
	config := struct {
		Sm *Settings
	}{DefaultSettings()}

	if err := env.Unmarshal(&config); err != nil {
		return nil, errors.Wrap(err, "error unmarshaling SM configuration")
	}

	return config.Sm, nil
}

// Settings type holds SM Client config properties
type Settings struct {
	User              string
	Password          string
	Host              string
	OsbAPI            string        `mapstructure:"osb_api"`
	RequestTimeout    time.Duration `mapstructure:"request_timeout"`
	ResyncPeriod      time.Duration `mapstructure:"resync_period"`
	SkipSSLValidation bool          `mapstructure:"skip_ssl_validation"`

	CreateFunc func(config *Settings) (Client, error)
}

// Validate validates the configuration and returns appropriate errors in case it is invalid
func (c *Settings) Validate() error {
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
	if c.ResyncPeriod == 0 {
		return errors.New("SM configuration RequestTimeout missing")
	}
	if c.CreateFunc == nil {
		return errors.New("SM configuration CreateFunc missing")
	}
	return nil
}
