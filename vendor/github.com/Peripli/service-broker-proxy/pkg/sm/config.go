package sm

import (
	"github.com/Peripli/service-broker-proxy/pkg/env"
	"github.com/pkg/errors"
)

type ClientConfiguration struct {
	User           string
	Password       string
	Host           string
	OsbApi         string
	TimeoutSeconds int
	CreateFunc     func(config *ClientConfiguration) (Client, error)
}

func (c *ClientConfiguration) Validate() error {
	if len(c.User) == 0 {
		return errors.New("SM configuration User missing")
	}
	if len(c.Password) == 0 {
		return errors.New("SM configuration Password missing")
	}
	if len(c.Host) == 0 {
		return errors.New("SM configuration Host missing")
	}
	if len(c.OsbApi) == 0 {
		return errors.New("SM configuration OSB API missing")
	}
	if c.TimeoutSeconds == 0 {
		return errors.New("SM configuration TimeoutSeconds missing")
	}
	if c.CreateFunc == nil {
		return errors.New("SM configuration CreateFunc missing")
	}
	return nil
}

func DefaultConfig() *ClientConfiguration {
	return &ClientConfiguration{
		User:           "admin",
		Password:       "admin",
		Host:           "",
		TimeoutSeconds: 10,
		CreateFunc:     NewClient,
	}
}

func NewConfig(env env.Environment) (*ClientConfiguration, error) {
	config := DefaultConfig()

	smConfig := &struct {
		Sm *ClientConfiguration
	}{
		Sm: config,
	}
	if err := env.Unmarshal(smConfig); err != nil {
		return nil, errors.Wrap(err, "error unmarshaling SM configuration")
	}

	return config, nil
}
