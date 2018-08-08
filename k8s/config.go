package k8s

import (
	"errors"
	"time"

	"github.com/Peripli/service-manager/pkg/env"
	svcatclient "github.com/kubernetes-incubator/service-catalog/pkg/client/clientset_generated/clientset"
	"github.com/kubernetes-incubator/service-catalog/pkg/svcat/service-catalog"

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

// SecretRef reference to secret used for broker registration
type SecretRef struct {
	Namespace string
	Name      string
}

// RegistrationDetails type represents the credentials and secret name used to register a broker at the k8s cluster
type RegistrationDetails struct {
	User     string
	Password string
	Secret   *SecretRef
}

// ClientConfiguration type holds config info for building the k8s service catalog client
type ClientConfiguration struct {
	*LibraryConfig      `mapstructure:"client"`
	Reg                 *RegistrationDetails
	K8sClientCreateFunc func(*LibraryConfig) (*servicecatalog.SDK, error)
}

// Settings type wraps the K8S client configuration
type Settings struct {
	K8S *ClientConfiguration `mapstructure:"k8s"`
}

// newSvcatSDK creates a service-catalog client from configuration
func newSvcatSDK(libraryConfig *LibraryConfig) (*servicecatalog.SDK, error) {
	config, err := restInClusterConfig()
	if err != nil {
		logrus.Error("Failed to load client config: " + err.Error())
		return nil, err
	}

	config.Timeout = libraryConfig.Timeout

	svcatClient, err := svcatclient.NewForConfig(config)
	if err != nil {
		logrus.Error("Failed to create new svcat client: " + err.Error())
		return nil, err
	}

	k8sClient, err := k8sclient.NewForConfig(config)
	if err != nil {
		logrus.Error("Failed to create new k8sClient: " + err.Error())
		return nil, err
	}

	return &servicecatalog.SDK{
		K8sClient:            k8sClient,
		ServiceCatalogClient: svcatClient,
	}, nil
}

// defaultClientConfiguration creates a default config for the K8S client
func defaultClientConfiguration() *ClientConfiguration {
	return &ClientConfiguration{
		LibraryConfig: &LibraryConfig{
			&HTTPClient{Timeout: time.Second * 10},
		},
		Reg: &RegistrationDetails{
			Secret: &SecretRef{},
		},
		K8sClientCreateFunc: newSvcatSDK,
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
	if c.Reg == nil {
		return errors.New("K8S broker registration configuration missing")
	}
	if c.Reg.User == "" || c.Reg.Password == "" {
		return errors.New("K8S broker registration credentials missing")
	}
	if c.Reg.Secret == nil {
		return errors.New("K8S secret configuration for broker registration missing")
	}
	if c.Reg.Secret.Name == "" || c.Reg.Secret.Namespace == "" {
		return errors.New("Properties of K8S secret configuration for broker registration missing")
	}
	if c.LibraryConfig.HTTPClient == nil || c.LibraryConfig.HTTPClient.Timeout == 0 {
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
