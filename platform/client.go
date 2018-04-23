package platform

import (
	"github.com/Peripli/service-broker-proxy/pkg/platform"
	"github.com/kubernetes-incubator/service-catalog/pkg/svcat"
	"flag"
	"github.com/kubernetes-incubator/service-catalog/pkg/client/clientset_generated/clientset"
	"github.com/kubernetes-incubator/service-catalog/pkg/svcat/kube"
)

type PlatformClient struct {
	app 	 *svcat.App
}

var _ platform.Client = &PlatformClient{}
var _ platform.Fetcher = &PlatformClient{}

func NewClient(config *PlatformClientConfiguration) (platform.Client, error) {
	kubeconfig := flag.String("kubeconfig", "", "Path to a kubeconfig file")
	kube := kube.GetConfig("shoot-garden-cpet-peripli", *kubeconfig)
	restConfig, _ := kube.ClientConfig()
	appClient, _ := clientset.NewForConfig(restConfig)
	a, _ := svcat.NewApp(appClient, "shoot-garden-cpet-peripli")

	return &PlatformClient{
		a,
	}, nil
}

func (b PlatformClient) GetBrokers() ([]platform.ServiceBroker, error) {
	//logrus.Debug("Getting brokers via k8s client...")
	//
	//brokers, err := serviceCatalog.app.RetrieveBrokers()
	//if err != nil {
	//	return nil, err
	//}
	//
	//var clientBrokers []platform.ServiceBroker
	//for _, broker := range brokers {
	//	serviceBroker := platform.ServiceBroker{
	//		Guid:      broker.Guid,
	//		Name:      broker.Name,
	//		BrokerURL: broker.BrokerURL,
	//	}
	//	clientBrokers = append(clientBrokers, serviceBroker)
	//}
	//logrus.Debugf("Successfully got %d brokers via CF client", len(clientBrokers))

	//return clientBrokers, nil
	return nil, nil
}

func (b PlatformClient) CreateBroker(r *platform.CreateServiceBrokerRequest) (*platform.ServiceBroker, error) {
	//logrus.Debugf("Creating broker via CF Client with name [%s]...", r.Name)
	//
	//request := cfclient.CreateServiceBrokerRequest{
	//	Username:  b.reg.User,
	//	Password:  b.reg.Password,
	//	Name:      r.Name,
	//	BrokerURL: r.BrokerURL,
	//}
	//
	//broker, err := b.cfClient.CreateServiceBroker(request)
	//if err != nil {
	//	return nil, err
	//}
	//
	//response := &platform.ServiceBroker{
	//	Guid:      broker.Guid,
	//	Name:      broker.Name,
	//	BrokerURL: broker.BrokerURL,
	//}
	//logrus.Debugf("Successfully created broker via CF Client with name [%s]...", r.Name)

	//return response, nil
	return nil, nil
}

func (b PlatformClient) DeleteBroker(r *platform.DeleteServiceBrokerRequest) error {
	//logrus.Debugf("Deleting broker via CF Client with guid [%s] ", r.Guid)
	//
	//if err := b.cfClient.DeleteServiceBroker(r.Guid); err != nil {
	//	return err
	//}
	//logrus.Debugf("Successfully deleted broker via CF Client with guid [%s] ", r.Guid)
	//
	//return nil
	return nil
}

func (b PlatformClient) UpdateBroker(r *platform.UpdateServiceBrokerRequest) (*platform.ServiceBroker, error) {
	//logrus.Debugf("Updating broker with name [%s] and guid [%s]...", r.Name, r.Guid)
	//
	//request := cfclient.UpdateServiceBrokerRequest{
	//	Username:  b.reg.User,
	//	Password:  b.reg.Password,
	//	Name:      r.Name,
	//	BrokerURL: r.BrokerURL,
	//}
	//
	//broker, err := b.cfClient.UpdateServiceBroker(r.Guid, request)
	//if err != nil {
	//	return nil, err
	//}
	//response := &platform.ServiceBroker{
	//	Guid:      broker.Guid,
	//	Name:      broker.Name,
	//	BrokerURL: broker.BrokerURL,
	//}
	//logrus.Debugf("Successfully updated broker with name [%s] and guid [%s]...", r.Name, r.Guid)
	//
	//return response, nil
	return nil, nil
}

func (b PlatformClient) Fetch(serviceBroker *platform.ServiceBroker) error {
	//_, err := b.UpdateBroker(&platform.UpdateServiceBrokerRequest{
	//	Guid:      serviceBroker.Guid,
	//	Name:      serviceBroker.Name,
	//	BrokerURL: serviceBroker.BrokerURL,
	//})
	//return err
	return nil
}