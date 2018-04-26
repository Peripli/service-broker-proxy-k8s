package osb

import (
	"github.com/Peripli/service-broker-proxy/pkg/env"
	"github.com/Peripli/service-broker-proxy/pkg/sm"
	"github.com/pkg/errors"
	osbc "github.com/pmorie/go-open-service-broker-client/v2"
)

type ClientConfiguration struct {
	*osbc.ClientConfiguration
	CreateFunc func(config *osbc.ClientConfiguration) (osbc.Client, error)
}

func NewConfig(env env.Environment) (*ClientConfiguration, error) {

	settings, err := sm.NewConfig(env)
	if err != nil {
		return nil, errors.Wrap(err, "error creating default SM config")
	}

	clientConfig := osbc.DefaultClientConfiguration()
	clientConfig.Name = "sm"

	if len(settings.Host) != 0 {
		clientConfig.URL = settings.Host + settings.OsbApi
	}

	if len(settings.User) != 0 && len(settings.Password) != 0 {
		clientConfig.AuthConfig = &osbc.AuthConfig{
			BasicAuthConfig: &osbc.BasicAuthConfig{
				Username: settings.User,
				Password: settings.Password,
			}}
	}
	if settings.TimeoutSeconds != 0 {
		clientConfig.TimeoutSeconds = settings.TimeoutSeconds
	}

	return &ClientConfiguration{
		ClientConfiguration: clientConfig,
		CreateFunc:          osbc.NewClient,
	}, nil
}

func (c *ClientConfiguration) Validate() error {
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
		return errors.New("OSB client configuration TimeoutSeconds missing")
	}
	return nil
}
