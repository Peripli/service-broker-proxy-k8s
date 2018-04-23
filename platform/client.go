package platform

import (
	"github.com/Peripli/service-broker-proxy/pkg/platform"
	"github.com/kubernetes-incubator/service-catalog/pkg/svcat"
	"flag"
	"github.com/kubernetes-incubator/service-catalog/pkg/client/clientset_generated/clientset"
	"github.com/kubernetes-incubator/service-catalog/pkg/svcat/kube"
	"github.com/sirupsen/logrus"
	"github.com/kubernetes-incubator/service-catalog/pkg/apis/servicecatalog/v1beta1"

	"k8s.io/apimachinery/pkg/apis/meta/v1"
)

type PlatformClient struct {
	app *svcat.App
}

var _ platform.Client = &PlatformClient{}
var _ platform.Fetcher = &PlatformClient{}

func NewClient() (platform.Client, error) {
	kubeconfig := flag.String("kubeconfig", "", "Path to a kubeconfig file")
	flag.Parse()
	kube := kube.GetConfig("shoot-garden-cpet-peripli", *kubeconfig)
	restConfig, _ := kube.ClientConfig()
	appClient, _ := clientset.NewForConfig(restConfig)
	a, _ := svcat.NewApp(appClient, "shoot-garden-cpet-peripli")

	return &PlatformClient{
		a,
	}, nil
}

func (b PlatformClient) GetBrokers() ([]platform.ServiceBroker, error) {
	logrus.Debug("Getting brokers via k8s client...")
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
	logrus.Debugf("Successfully got %d brokers via CF client", len(clientBrokers))

	return clientBrokers, nil
}

func (b PlatformClient) CreateBroker(r *platform.CreateServiceBrokerRequest) (*platform.ServiceBroker, error) {
	logrus.Debugf("Creating broker via CF Client with name [%s]...", r.Name)

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
		logrus.Fatal("[client.go; RegisterBroker()] Registering a broker at the service catalog failed: " + err.Error())
		return nil, err
	}
	logrus.Println("[client.go; RegisterBroker()] New service broker successfully registered")
	return &platform.ServiceBroker{
		Guid:      string(csb.UID),
		Name:      r.Name,
		BrokerURL: r.BrokerURL,
	}, nil
}

func (b PlatformClient) DeleteBroker(r *platform.DeleteServiceBrokerRequest) error {
	logrus.Debugf("Deleting broker via CF Client with guid [%s] ", r.Guid)

	err := b.app.ServiceCatalog().ClusterServiceBrokers().Delete(r.Name, &v1.DeleteOptions{})
	if err != nil {
		return err
	}
	logrus.Debugf("Successfully deleted broker via CF Client with guid [%s] ", r.Guid)

	return nil
}

func (b PlatformClient) UpdateBroker(r *platform.UpdateServiceBrokerRequest) (*platform.ServiceBroker, error) {
	logrus.Debugf("Updating broker via CF Client with guid [%s] ", r.Guid)

	broker := &v1beta1.ClusterServiceBroker{
		ObjectMeta: v1.ObjectMeta{
			Name:      r.Name,
		},
	}

	updatedBroker, err := b.app.ServiceCatalog().ClusterServiceBrokers().Update(broker)
	if err != nil {
		return nil, err
	}
	logrus.Debugf("Successfully updated broker via CF Client with guid [%s] ", r.Guid)

	return &platform.ServiceBroker{
		Guid:      string(updatedBroker.ObjectMeta.UID),
		Name:      updatedBroker.Name,
		BrokerURL: updatedBroker.Spec.URL,
	}, nil
}

func (b PlatformClient) Fetch(serviceBroker *platform.ServiceBroker) error {
	return b.app.Sync(serviceBroker.Name, 3)
}
