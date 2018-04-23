package platform

import (
	"fmt"
	"net/http"

	"time"

	"github.com/cloudfoundry-community/go-cfclient"
	"github.com/Peripli/service-broker-proxy/pkg/env"
)

type RegistrationDetails struct {
	User     string
	Password string
}

func (rd RegistrationDetails) String() string {
	return fmt.Sprintf("User: %s Host: %s", rd.User)
}

type PlatformClientConfiguration struct {
	*cfclient.Config
	cfClientCreateFunc func(*cfclient.Config) (*cfclient.Client, error)

	Reg *RegistrationDetails
}

func (c *PlatformClientConfiguration) Validate() error {
	//if c.cfClientCreateFunc == nil {
	//	return errors.New("CF ClientCreateFunc missing")
	//}
	//if c.Config == nil {
	//	return errors.New("CF client configuration missing")
	//}
	//if len(c.Config.ApiAddress) == 0 {
	//	return errors.New("CF client configuration ApiAddress missing")
	//}
	//if c.Reg == nil {
	//	return errors.New("CF client configuration Registration credentials missing")
	//}
	//if len(c.Reg.User) == 0 {
	//	return errors.New("CF client configuration Registration details user missing")
	//}
	//if len(c.Reg.Password) == 0 {
	//	return errors.New("CF client configuration Registration details password missing")
	//}
	return nil
}

type settings struct {
	Api            string
	ClientID       string
	ClientSecret   string
	Username       string
	Password       string
	SkipSSLVerify  bool
	TimeoutSeconds int
	Reg            *RegistrationDetails
}

func NewConfig(env env.Environment) (*PlatformClientConfiguration, error) {

	platformSettings := &struct {
		Cf *settings
	}{
		Cf: &settings{},
	}
	if err := env.Unmarshal(platformSettings); err != nil {
		return nil, err
	}

	clientConfig := cfclient.DefaultConfig()

	if len(platformSettings.Cf.Api) != 0 {
		clientConfig.ApiAddress = platformSettings.Cf.Api
	}
	if len(platformSettings.Cf.ClientID) != 0 {
		clientConfig.ClientID = platformSettings.Cf.ClientID
	}
	if len(platformSettings.Cf.ClientSecret) != 0 {
		clientConfig.ClientSecret = platformSettings.Cf.ClientSecret
	}
	if len(platformSettings.Cf.Username) != 0 {
		clientConfig.Username = platformSettings.Cf.Username
	}
	if len(platformSettings.Cf.Password) != 0 {
		clientConfig.Password = platformSettings.Cf.Password
	}
	if platformSettings.Cf.SkipSSLVerify {
		clientConfig.SkipSslValidation = platformSettings.Cf.SkipSSLVerify
	}
	if platformSettings.Cf.TimeoutSeconds != 0 {
		clientConfig.HttpClient = &http.Client{
			Timeout: time.Duration(platformSettings.Cf.TimeoutSeconds) * time.Second,
		}
	}
	return &PlatformClientConfiguration{
		Config:             clientConfig,
		Reg:                platformSettings.Cf.Reg,
		cfClientCreateFunc: cfclient.NewClient,
	}, nil
}