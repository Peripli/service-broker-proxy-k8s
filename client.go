package main

import (
	"github.com/Peripli/service-broker-proxy/pkg/platform"
	"github.com/kubernetes-incubator/service-catalog/pkg/svcat"
	"flag"
	"github.com/kubernetes-incubator/service-catalog/pkg/client/clientset_generated/clientset"
	"github.com/sirupsen/logrus"
	"github.com/kubernetes-incubator/service-catalog/pkg/apis/servicecatalog/v1beta1"
	"k8s.io/client-go/tools/clientcmd"
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

func getClientConfig(kubeconfig string) (*rest.Config, error) {
	if kubeconfig != "" {
		logrus.Println("Load configuration from kubeconfig")
		return clientcmd.BuildConfigFromFlags("", kubeconfig)
	}
	logrus.Println("Load 'inCluster' configuration ")
	return rest.InClusterConfig()
}

// NewClient can be used to create a service-catalog client to communicate with the kubernetes service-catalog.
func NewClient() (*PlatformClient, error) {

	kubeconfig := flag.String(clientcmd.RecommendedConfigPathFlag, "", "Path to a kubeconfig file")
	flag.Parse()

	// Build the client config - optionally using a provided kubeconfig file.
	config, err := getClientConfig(*kubeconfig)
	if err != nil {
		logrus.Fatalf("Failed to load client config: %v", err)
	}

	appClient, err := clientset.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	k8sClient, err := k8sclient.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	a, _ := svcat.NewApp(k8sClient, appClient, "")

	return &PlatformClient{
		a,
	}, nil
}
// GetBrokers returns all service-brokers currently registered in kubernetes
func (b PlatformClient) GetBrokers() ([]platform.ServiceBroker, error) {
	logrus.Debug("Getting all brokers registered in the k8s service-catalog...")
	brokers, err := b.app.RetrieveBrokers()
	if err != nil {
		return nil, err
	}

	var clientBrokers []platform.ServiceBroker
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

// CreateBroker registers a new broker in kubernetes
func (b PlatformClient) CreateBroker(r *platform.CreateServiceBrokerRequest) (*platform.ServiceBroker, error) {
	logrus.Debugf("Creating broker via k8s client with name [%s]...", r.Name)

	request := &v1beta1.ClusterServiceBroker{
		ObjectMeta: v1.ObjectMeta{
			Name:      r.Name,
		},
		Spec: v1beta1.ClusterServiceBrokerSpec{
			CommonServiceBrokerSpec: v1beta1.CommonServiceBrokerSpec{
				URL:            r.BrokerURL,
				RelistBehavior: "Manual",
			},
		},
	}

	csb, err := b.app.ServiceCatalog().ClusterServiceBrokers().Create(request)
	if err != nil {
		logrus.Warn("Registering a broker at the service catalog failed: " + err.Error())
		return nil, err
	}
	logrus.Println("New service broker successfully registered in k8s")
	return &platform.ServiceBroker{
		Guid:      string(csb.UID),
		Name:      r.Name,
		BrokerURL: r.BrokerURL,
	}, nil
}

// DeleteBroker deletes an existing broker in from kubernetes
func (b PlatformClient) DeleteBroker(r *platform.DeleteServiceBrokerRequest) error {
	logrus.Debugf("Deleting broker via k8s client with guid [%s] ", r.Guid)

	err := b.app.ServiceCatalog().ClusterServiceBrokers().Delete(r.Name, &v1.DeleteOptions{})
	if err != nil {
		return err
	}
	logrus.Debugf("Successfully deleted broker via k8s client with guid [%s] ", r.Guid)

	return nil
}

// UpdateBroker updates a broker in kubernetes
func (b PlatformClient) UpdateBroker(r *platform.UpdateServiceBrokerRequest) (*platform.ServiceBroker, error) {
	logrus.Debugf("Updating broker via k8s client with guid [%s] ", r.Guid)

	broker := &v1beta1.ClusterServiceBroker{
		ObjectMeta: v1.ObjectMeta{
			Name:      r.Name,
		},
	}

	updatedBroker, err := b.app.ServiceCatalog().ClusterServiceBrokers().Update(broker)
	if err != nil {
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
// so that it is visible in the kubernetes service-catalog
func (b PlatformClient) Fetch(serviceBroker *platform.ServiceBroker) error {
	return b.app.Sync(serviceBroker.Name, 3)
}

