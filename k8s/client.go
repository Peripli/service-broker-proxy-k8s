package k8s

import (
	"github.com/Peripli/service-broker-proxy/pkg/platform"
	"github.com/kubernetes-incubator/service-catalog/pkg/apis/servicecatalog/v1beta1"
	"github.com/kubernetes-incubator/service-catalog/pkg/svcat/service-catalog"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
)

// PlatformClient implements all broker specific functions, like create/update/delete/list a service broker
// in kubernetes.
type PlatformClient struct {
	cli *servicecatalog.SDK
}

var _ platform.Client = &PlatformClient{}
var _ platform.CatalogFetcher = &PlatformClient{}

var restInClusterConfig = rest.InClusterConfig

var createClusterServiceBroker = func(cli *servicecatalog.SDK, broker *v1beta1.ClusterServiceBroker) (*v1beta1.ClusterServiceBroker, error) {
	return cli.ServiceCatalog().ClusterServiceBrokers().Create(broker)
}

var deleteClusterServiceBroker = func(cli *servicecatalog.SDK, name string, options *v1.DeleteOptions) error {
	return cli.ServiceCatalog().ClusterServiceBrokers().Delete(name, options)
}

var retrieveClusterServiceBrokers = func(cli *servicecatalog.SDK) ([]v1beta1.ClusterServiceBroker, error) {
	return cli.RetrieveBrokers()
}

var updateClusterServiceBroker = func(cli *servicecatalog.SDK, broker *v1beta1.ClusterServiceBroker) (*v1beta1.ClusterServiceBroker, error) {
	return cli.ServiceCatalog().ClusterServiceBrokers().Update(broker)
}

var syncClusterServiceBroker = func(cli *servicecatalog.SDK, name string, retries int) error {
	return cli.Sync(name, 3)
}

// NewClient create a client to communicate with the kubernetes service-catalog.
func NewClient(config *ClientConfiguration) (*PlatformClient, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	svcatSDK, err := config.K8sClientCreateFunc(config.LibraryConfig)
	if err != nil {
		return nil, err
	}
	return &PlatformClient{svcatSDK}, nil
}

// GetBrokers returns all service-brokers currently registered in kubernetes service-catalog.
func (b PlatformClient) GetBrokers() ([]platform.ServiceBroker, error) {
	logrus.Debug("Getting all brokers registered in the k8s service-catalog...")
	brokers, err := retrieveClusterServiceBrokers(b.cli)
	if err != nil {
		logrus.Error("Getting all brokers at the service catalog failed: " + err.Error())
		return nil, err
	}

	var clientBrokers = make([]platform.ServiceBroker, 0)
	for _, broker := range brokers {
		serviceBroker := platform.ServiceBroker{
			GUID:      string(broker.ObjectMeta.UID),
			Name:      broker.Name,
			BrokerURL: broker.Spec.URL,
		}
		clientBrokers = append(clientBrokers, serviceBroker)
	}
	logrus.Debugf("Successfully got %d brokers via k8s client", len(clientBrokers))

	return clientBrokers, nil
}

// CreateBroker registers a new broker in kubernetes service-catalog.
func (b PlatformClient) CreateBroker(r *platform.CreateServiceBrokerRequest) (*platform.ServiceBroker, error) {
	logrus.Debugf("Creating broker via k8s client with name [%s]...", r.Name)

	request := &v1beta1.ClusterServiceBroker{
		ObjectMeta: v1.ObjectMeta{
			Name: r.Name,
		},
		Spec: v1beta1.ClusterServiceBrokerSpec{
			CommonServiceBrokerSpec: v1beta1.CommonServiceBrokerSpec{
				URL:            r.BrokerURL,
				RelistBehavior: "Manual",
			},
			// TODO refer secret in broker
			// AuthInfo: &v1beta1.AuthInfo{
			// 	Basic: &v1beta1.ClusterBasicAuthConfig{
			// 		SecretRef: &
			// 	}
			// }
		},
	}

	csb, err := createClusterServiceBroker(b.cli, request)
	if err != nil {
		logrus.Error("Registering new broker with name '" + r.Name + "' at the service catalog failed: " + err.Error())
		return nil, err
	}
	logrus.Debugf("New service broker successfully registered in k8s")

	return &platform.ServiceBroker{
		GUID:      string(csb.UID),
		Name:      r.Name,
		BrokerURL: r.BrokerURL,
	}, nil
}

// DeleteBroker deletes an existing broker in from kubernetes service-catalog.
func (b PlatformClient) DeleteBroker(r *platform.DeleteServiceBrokerRequest) error {
	logrus.Debugf("Deleting broker via k8s client with guid [%s] ", r.GUID)

	err := deleteClusterServiceBroker(b.cli, r.Name, &v1.DeleteOptions{})
	if err != nil {
		logrus.Error("Deleting broker '" + r.GUID + "' at the service catalog failed: " + err.Error())
		return err
	}
	logrus.Debugf("Successfully deleted broker via k8s client with guid [%s] ", r.GUID)

	return nil
}

// UpdateBroker updates a service broker in the kubernetes service-catalog.
func (b PlatformClient) UpdateBroker(r *platform.UpdateServiceBrokerRequest) (*platform.ServiceBroker, error) {
	logrus.Debugf("Updating broker via k8s client with guid [%s] ", r.GUID)

	// Name and broker url are updateable
	broker := &v1beta1.ClusterServiceBroker{
		ObjectMeta: v1.ObjectMeta{
			Name: r.Name,
		},
		Spec: v1beta1.ClusterServiceBrokerSpec{
			CommonServiceBrokerSpec: v1beta1.CommonServiceBrokerSpec{
				URL: r.BrokerURL,
			},
		},
	}

	updatedBroker, err := updateClusterServiceBroker(b.cli, broker)
	if err != nil {
		logrus.Error("Updating broker '" + r.GUID + "' at the service catalog failed: " + err.Error())
		return nil, err
	}
	logrus.Debugf("Successfully updated broker via k8s Client with guid [%s] ", r.GUID)

	return &platform.ServiceBroker{
		GUID:      string(updatedBroker.ObjectMeta.UID),
		Name:      updatedBroker.Name,
		BrokerURL: updatedBroker.Spec.URL,
	}, nil
}

// Fetch the new catalog information from reach service-broker registered in kubernetes,
// so that it is visible in the kubernetes service-catalog.
func (b PlatformClient) Fetch(serviceBroker *platform.ServiceBroker) error {
	logrus.Debugf("Updating catalog information of service-broker with guid [%s] ", serviceBroker.GUID)
	err := syncClusterServiceBroker(b.cli, serviceBroker.Name, 3)
	if err != nil {
		logrus.Error("Syncing broker '" + serviceBroker.GUID + "' at the service catalog failed: " + err.Error())
	}
	return err
}
