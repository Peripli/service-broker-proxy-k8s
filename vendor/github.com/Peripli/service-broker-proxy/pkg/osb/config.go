// Package osb contains logic for building the Service Broker Proxy OSB API
package osb

import (
	"github.com/Peripli/service-broker-proxy/pkg/sm"
	"github.com/pkg/errors"
	osbc "github.com/pmorie/go-open-service-broker-client/v2"
)

const name = "sbproxy"

// ClientConfig type holds config info for building an OSB client
type ClientConfig struct {
	*osbc.ClientConfiguration

	CreateFunc func(config *osbc.ClientConfiguration) (osbc.Client, error)
}

// DefaultConfig returns default ClientConfig
func DefaultConfig() *ClientConfig {
	return &ClientConfig{
		ClientConfiguration: osbc.DefaultClientConfiguration(),
		CreateFunc:          osbc.NewClient,
	}
}

// NewConfig creates ClientConfig from the provided settings
func NewConfig(settings *sm.Config) (*ClientConfig, error) {
	clientConfig := DefaultConfig()
	clientConfig.Name = name

	if len(settings.Host) != 0 {
		clientConfig.URL = settings.Host + settings.OsbAPI
	}

	if settings.RequestTimeout != 0 {
		clientConfig.TimeoutSeconds = (int)(settings.RequestTimeout.Seconds())
	}

	if len(settings.User) != 0 && len(settings.Password) != 0 {
		clientConfig.AuthConfig = &osbc.AuthConfig{
			BasicAuthConfig: &osbc.BasicAuthConfig{
				Username: settings.User,
				Password: settings.Password,
			}}
	}

	clientConfig.Insecure = settings.SkipSslValidation

	return clientConfig, nil
}

// Validate validates the configuration and returns appropriate errors in case it is invalid
func (c *ClientConfig) Validate() error {
	if c.CreateFunc == nil {
		return errors.New("OSB client configuration CreateFunc missing")
	}
	if c.ClientConfiguration == nil {
		return errors.New("OSB client configuration missing")
	}
	if len(c.URL) == 0 {
		return errors.New("OSB client configuration URL missing")
	}
	if c.AuthConfig == nil {
		return errors.New("OSB client configuration AuthConfig missing")
	}
	if c.TimeoutSeconds == 0 {
		return errors.New("OSB client configuration RequestTimeout missing")
	}
	return nil
}
