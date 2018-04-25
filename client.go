package main

import (
	"github.com/Peripli/service-broker-proxy/pkg/platform"
	"github.com/kubernetes-incubator/service-catalog/pkg/svcat"
	"flag"
	"github.com/kubernetes-incubator/service-catalog/pkg/client/clientset_generated/clientset"
	"github.com/sirupsen/logrus"
	"github.com/kubernetes-incubator/service-catalog/pkg/apis/servicecatalog/v1beta1"
	. "k8s.io/client-go/tools/clientcmd"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sclient "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/rest"
)

type PlatformClient struct {
	app *svcat.App
}

var _ platform.Client = &PlatformClient{}
var _ platform.Fetcher = &PlatformClient{}

func GetClientConfig(kubeconfig string) (*rest.Config, error) {
	if kubeconfig != "" {
		logrus.Println("Load configuration from kubeconfig")
		return clientcmd.BuildConfigFromFlags("", kubeconfig)
	}
	logrus.Println("Load 'inCluster' configuration ")
	return rest.InClusterConfig()
}

func NewClient() (platform.Client, error) {

	kubeconfig := flag.String(RecommendedConfigPathFlag, "", "Path to a kubeconfig file")
	flag.Parse()

	// Build the client config - optionally using a provided kubeconfig file.
	config, err := GetClientConfig(*kubeconfig)
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

func (b PlatformClient) DeleteBroker(r *platform.DeleteServiceBrokerRequest) error {
	logrus.Debugf("Deleting broker via k8s client with guid [%s] ", r.Guid)

	err := b.app.ServiceCatalog().ClusterServiceBrokers().Delete(r.Name, &v1.DeleteOptions{})
	if err != nil {
		return err
	}
	logrus.Debugf("Successfully deleted broker via k8s client with guid [%s] ", r.Guid)

	return nil
}

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

func (b PlatformClient) Fetch(serviceBroker *platform.ServiceBroker) error {
	return b.app.Sync(serviceBroker.Name, 3)
}
