package main

import (
	"github.com/Peripli/service-broker-proxy/pkg/platform"
	"github.com/kubernetes-incubator/service-catalog/pkg/apis/servicecatalog/v1beta1"
	"github.com/kubernetes-incubator/service-catalog/pkg/client/clientset_generated/clientset"
	"github.com/kubernetes-incubator/service-catalog/pkg/svcat"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sclient "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// PlatformClient implements all broker specific functions, like create/update/delete/list a service broker
// in kubernetes.
type PlatformClient struct {
	app *svcat.App
}

var _ platform.Client = &PlatformClient{}
var _ platform.CatalogFetcher = &PlatformClient{}

var restInClusterConfig = rest.InClusterConfig

var createClusterServiceBroker = func(app *svcat.App, broker *v1beta1.ClusterServiceBroker) (*v1beta1.ClusterServiceBroker, error) {
	return app.ServiceCatalog().ClusterServiceBrokers().Create(broker)
}

var deleteClusterServiceBroker = func(app *svcat.App, name string, options *v1.DeleteOptions) error {
	return app.ServiceCatalog().ClusterServiceBrokers().Delete(name, options)
}

var retrieveClusterServiceBrokers = func(app *svcat.App) ([]v1beta1.ClusterServiceBroker, error) {
	return app.RetrieveBrokers()
}

var updateClusterServiceBroker = func(app *svcat.App, broker *v1beta1.ClusterServiceBroker) (*v1beta1.ClusterServiceBroker, error) {
	return app.ServiceCatalog().ClusterServiceBrokers().Update(broker)
}

var syncClusterServiceBroker = func(app *svcat.App, name string, retries int) error {
	return app.Sync(name, 3)
}

// NewClient can be used to create a service-catalog client to communicate with the kubernetes service-catalog.
func NewClient() (*PlatformClient, error) {
	config, err := restInClusterConfig()
	if err != nil {
		logrus.Fatalf("Failed to load client config: " + err.Error())
	}

	appClient, err := clientset.NewForConfig(config)
	if err != nil {
		logrus.Fatalf("Failed to create new ClientSet: " + err.Error())
	}

	k8sClient, err := k8sclient.NewForConfig(config)
	if err != nil {
		logrus.Fatalf("Failed to create new k8sClient: " + err.Error())
	}

	a, _ := svcat.NewApp(k8sClient, appClient, "")

	return &PlatformClient{
		a,
	}, nil
}

// GetBrokers returns all service-brokers currently registered in kubernetes service-catalog.
func (b PlatformClient) GetBrokers() ([]platform.ServiceBroker, error) {
	logrus.Debug("Getting all brokers registered in the k8s service-catalog...")
	brokers, err := retrieveClusterServiceBrokers(b.app)
	if err != nil {
		logrus.Error("Getting all brokers at the service catalog failed: " + err.Error())
		return nil, err
	}

	var clientBrokers = make([]platform.ServiceBroker, 0)
	for _, broker := range brokers {
		serviceBroker := platform.ServiceBroker{
			Guid:      string(broker.ObjectMeta.UID),
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
		},
	}

	csb, err := createClusterServiceBroker(b.app, request)
	if err != nil {
		logrus.Error("Registering new broker with name '" + r.Name + "' at the service catalog failed: " + err.Error())
		return nil, err
	}
	logrus.Debugf("New service broker successfully registered in k8s")

	return &platform.ServiceBroker{
		Guid:      string(csb.UID),
		Name:      r.Name,
		BrokerURL: r.BrokerURL,
	}, nil
}

// DeleteBroker deletes an existing broker in from kubernetes service-catalog.
func (b PlatformClient) DeleteBroker(r *platform.DeleteServiceBrokerRequest) error {
	logrus.Debugf("Deleting broker via k8s client with guid [%s] ", r.Guid)

	err := deleteClusterServiceBroker(b.app, r.Name, &v1.DeleteOptions{})
	if err != nil {
		logrus.Error("Deleting broker '" + r.Guid + "' at the service catalog failed: " + err.Error())
		return err
	}
	logrus.Debugf("Successfully deleted broker via k8s client with guid [%s] ", r.Guid)

	return nil
}

// UpdateBroker updates a service broker in the kubernetes service-catalog.
func (b PlatformClient) UpdateBroker(r *platform.UpdateServiceBrokerRequest) (*platform.ServiceBroker, error) {
	logrus.Debugf("Updating broker via k8s client with guid [%s] ", r.Guid)

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

	updatedBroker, err := updateClusterServiceBroker(b.app, broker)
	if err != nil {
		logrus.Error("Updating broker '" + r.Guid + "' at the service catalog failed: " + err.Error())
		return nil, err
	}
	logrus.Debugf("Successfully updated broker via k8s Client with guid [%s] ", r.Guid)

	return &platform.ServiceBroker{
		Guid:      string(updatedBroker.ObjectMeta.UID),
		Name:      updatedBroker.Name,
		BrokerURL: updatedBroker.Spec.URL,
	}, nil
}

// Fetch the new catalog information from reach service-broker registered in kubernetes,
// so that it is visible in the kubernetes service-catalog.
func (b PlatformClient) Fetch(serviceBroker *platform.ServiceBroker) error {
	logrus.Debugf("Updating catalog information of service-broker with guid [%s] ", serviceBroker.Guid)
	err := syncClusterServiceBroker(b.app, serviceBroker.Name, 3)
	if err != nil {
		logrus.Error("Syncing broker '" + serviceBroker.Guid + "' at the service catalog failed: " + err.Error())
	}
	return err
}
