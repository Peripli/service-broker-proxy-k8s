// Package osb contains logic for building the Service Broker Proxy OSB API
package osb

import (
	"github.com/Peripli/service-broker-proxy/pkg/sm"
	"github.com/pkg/errors"
)

const name = "sbproxy"

type Target struct {
	// URL is the URL to use to contact the broker.
	URL string
	// Username is the basic auth username.
	Username string
	// Password is the basic auth password.
	Password string
	// URL is the URL to use to contact the broker.
}

// ClientConfig type holds config info for building an OSB client
type ClientConfig struct {
	Name string
	// Username is the basic auth username.
	Username string
	// Password is the basic auth password.
	Password string
	// URL is the URL to use to contact the broker.
	URL string
	// Insecure represents whether the 'InsecureSkipVerify' TLS configuration
	// field should be set.  If the TLSConfig field is set and this field is
	// set to true, it overrides the value in the TLSConfig field.
	Insecure bool
	// TimeoutSeconds is the length of the timeout of any request to the
	// broker, in seconds.
	TimeoutSeconds int

	// CreateFunc func(config *osbc.ClientConfiguration) (osbc.Client, error)
}

// DefaultConfig returns default ClientConfig
func DefaultConfig() *ClientConfig {
	return &ClientConfig{
		Name: name,
	}
}

// NewConfig creates ClientConfig from the provided settings
func NewConfig(settings *sm.Config) (*ClientConfig, error) {
	clientConfig := DefaultConfig()

	if len(settings.Host) != 0 {
		clientConfig.URL = settings.Host + settings.OsbAPI
	}

	if settings.RequestTimeout != 0 {
		clientConfig.TimeoutSeconds = (int)(settings.RequestTimeout.Seconds())
	}

	if len(settings.User) != 0 && len(settings.Password) != 0 {
		clientConfig.Username = settings.User
		clientConfig.Password = settings.Password
	}

	clientConfig.Insecure = settings.SkipSslValidation

	return clientConfig, nil
}

// Validate validates the configuration and returns appropriate errors in case it is invalid
func (c *ClientConfig) Validate() error {
	if len(c.URL) == 0 {
		return errors.New("OSB client configuration URL missing")
	}
	if c.Username == "" || c.Password == "" {
		return errors.New("OSB client configuration Username/Password missing")
	}
	if c.TimeoutSeconds == 0 {
		return errors.New("OSB client configuration RequestTimeout missing")
	}
	return nil
}
