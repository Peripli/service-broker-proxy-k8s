package k8s

import (
	"errors"
	"time"

	"github.com/Peripli/service-manager/pkg/env"
	"github.com/kubernetes-incubator/service-catalog/pkg/client/clientset_generated/clientset"
	"github.com/kubernetes-incubator/service-catalog/pkg/svcat"
	"github.com/sirupsen/logrus"
	k8sclient "k8s.io/client-go/kubernetes"

	"github.com/spf13/pflag"
)

// HTTPClient configurations for the HTTPClient of the k8s library
type HTTPClient struct {
	Timeout time.Duration `mapstructure:"timeout"`
}

// LibraryConfig configurations for the k8s library
type LibraryConfig struct {
	*HTTPClient `mapstructure:"httpClient"`
}

// ClientConfiguration type holds config info for building the k8s client
type ClientConfiguration struct {
	*LibraryConfig      `mapstructure:"client"`
	K8sClientCreateFunc func(*LibraryConfig) (*svcat.App, error)
}

// Settings type wraps the K8S client configuration
type Settings struct {
	K8S *ClientConfiguration `mapstructure:"k8s"`
}

// newSvcatApp creates a service-catalog client from configuration
func newSvcatApp(libraryConfig *LibraryConfig) (*svcat.App, error) {
	config, err := restInClusterConfig()
	if err != nil {
		logrus.Error("Failed to load client config: " + err.Error())
		return nil, err
	}

	config.Timeout = libraryConfig.Timeout

	appClient, err := clientset.NewForConfig(config)
	if err != nil {
		logrus.Error("Failed to create new ClientSet: " + err.Error())
		return nil, err
	}

	k8sClient, err := k8sclient.NewForConfig(config)
	if err != nil {
		logrus.Error("Failed to create new k8sClient: " + err.Error())
		return nil, err
	}

	svcatApp, err := svcat.NewApp(k8sClient, appClient, "")
	if err != nil {
		logrus.Error("Failed to create new svcat application: " + err.Error())
		return nil, err
	}

	return svcatApp, nil
}

// defaultClientConfiguration creates a default config for the K8S client
func defaultClientConfiguration() *ClientConfiguration {
	return &ClientConfiguration{
		LibraryConfig: &LibraryConfig{
			&HTTPClient{Timeout: time.Second * 10},
		},
		K8sClientCreateFunc: newSvcatApp,
	}
}

// CreatePFlagsForK8SClient adds pflags relevant to the K8S client config
func CreatePFlagsForK8SClient(set *pflag.FlagSet) {
	env.CreatePFlags(set, &Settings{K8S: defaultClientConfiguration()})
}

// Validate validates the configuration and returns appropriate errors in case it is invalid
func (c *ClientConfiguration) Validate() error {
	if c.K8sClientCreateFunc == nil {
		return errors.New("K8S ClientCreateFunc missing")
	}
	if c.LibraryConfig == nil {
		return errors.New("K8S client configuration missing")
	}
	if c.LibraryConfig.HTTPClient != nil && c.LibraryConfig.HTTPClient.Timeout == 0 {
		return errors.New("K8S client configuration timeout missing")
	}
	return nil
}

// NewConfig creates ClientConfiguration from the provided environment
func NewConfig(env env.Environment) (*ClientConfiguration, error) {
	k8sSettings := &Settings{K8S: defaultClientConfiguration()}

	if err := env.Unmarshal(k8sSettings); err != nil {
		return nil, err
	}

	return k8sSettings.K8S, nil
}
