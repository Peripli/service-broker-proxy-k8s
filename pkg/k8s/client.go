package k8s

import (
	"context"
	"fmt"

	"github.com/Peripli/service-broker-proxy/pkg/platform"
	"github.com/kubernetes-incubator/service-catalog/pkg/apis/servicecatalog/v1beta1"
	servicecatalog "github.com/kubernetes-incubator/service-catalog/pkg/svcat/service-catalog"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
)

// PlatformClient implements all broker specific functions, like create/update/delete/list a service broker
// in kubernetes.
type PlatformClient struct {
	cli          *servicecatalog.SDK
	regSecretRef *v1beta1.ObjectReference
}

var _ platform.Client = &PlatformClient{}

// restInClusterConfig function returns config object which uses the service account kubernetes gives to pods
var restInClusterConfig = rest.InClusterConfig

var createClusterServiceBroker = func(cli *servicecatalog.SDK, broker *v1beta1.ClusterServiceBroker) (*v1beta1.ClusterServiceBroker, error) {
	return cli.ServiceCatalog().ClusterServiceBrokers().Create(broker)
}

var deleteClusterServiceBroker = func(cli *servicecatalog.SDK, name string, options *v1.DeleteOptions) error {
	return cli.ServiceCatalog().ClusterServiceBrokers().Delete(name, options)
}

var retrieveClusterServiceBrokers = func(cli *servicecatalog.SDK) (*v1beta1.ClusterServiceBrokerList, error) {
	return cli.ServiceCatalog().ClusterServiceBrokers().List(v1.ListOptions{})
}

var updateClusterServiceBroker = func(cli *servicecatalog.SDK, broker *v1beta1.ClusterServiceBroker) (*v1beta1.ClusterServiceBroker, error) {
	return cli.ServiceCatalog().ClusterServiceBrokers().Update(broker)
}

var syncClusterServiceBroker = func(cli *servicecatalog.SDK, name string, retries int) error {
	return cli.Sync(name, servicecatalog.ScopeOptions{}, 3)
}

// NewClient create a client to communicate with the kubernetes service-catalog.
func NewClient(config *ClientConfiguration) (*PlatformClient, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	svcatSDK, err := config.K8sClientCreateFunc(config.Client)
	if err != nil {
		return nil, err
	}
	return &PlatformClient{
		cli: svcatSDK,
		regSecretRef: &v1beta1.ObjectReference{
			Namespace: config.Secret.Namespace,
			Name:      config.Secret.Name,
		},
	}, nil
}

// Broker returns the platform client which handles broker operations
func (b *PlatformClient) Broker() platform.BrokerClient {
	return b
}

// CatalogFetcher returns the platform client which handles catalog fetch operations
func (b *PlatformClient) CatalogFetcher() platform.CatalogFetcher {
	return b
}

// Visibility returns nil as the platform client cannot handle visibilities operations
func (b *PlatformClient) Visibility() platform.VisibilityClient {
	return nil
}

// GetBrokers returns all service-brokers currently registered in kubernetes service-catalog.
func (b *PlatformClient) GetBrokers(ctx context.Context) ([]platform.ServiceBroker, error) {
	brokers, err := retrieveClusterServiceBrokers(b.cli)
	if err != nil {
		return nil, fmt.Errorf("unable to list cluster-scoped brokers (%s)", err)
	}
	var clientBrokers = make([]platform.ServiceBroker, 0)
	for _, broker := range brokers.Items {
		serviceBroker := platform.ServiceBroker{
			GUID:      string(broker.ObjectMeta.UID),
			Name:      broker.Name,
			BrokerURL: broker.Spec.URL,
		}
		clientBrokers = append(clientBrokers, serviceBroker)
	}
	return clientBrokers, nil
}

// CreateBroker registers a new broker in kubernetes service-catalog.
func (b *PlatformClient) CreateBroker(ctx context.Context, r *platform.CreateServiceBrokerRequest) (*platform.ServiceBroker, error) {
	broker := newServiceBroker(r.Name, r.BrokerURL, b.regSecretRef)
	broker.Spec.CommonServiceBrokerSpec.RelistBehavior = "Manual"

	csb, err := createClusterServiceBroker(b.cli, broker)
	if err != nil {
		return nil, err
	}
	return &platform.ServiceBroker{
		GUID:      string(csb.UID),
		Name:      r.Name,
		BrokerURL: r.BrokerURL,
	}, nil
}

// DeleteBroker deletes an existing broker in from kubernetes service-catalog.
func (b *PlatformClient) DeleteBroker(ctx context.Context, r *platform.DeleteServiceBrokerRequest) error {
	return deleteClusterServiceBroker(b.cli, r.Name, &v1.DeleteOptions{})
}

// UpdateBroker updates a service broker in the kubernetes service-catalog.
func (b *PlatformClient) UpdateBroker(ctx context.Context, r *platform.UpdateServiceBrokerRequest) (*platform.ServiceBroker, error) {
	// Name and broker url are updateable
	broker := newServiceBroker(r.Name, r.BrokerURL, b.regSecretRef)

	updatedBroker, err := updateClusterServiceBroker(b.cli, broker)
	if err != nil {
		return nil, err
	}
	return &platform.ServiceBroker{
		GUID:      string(updatedBroker.ObjectMeta.UID),
		Name:      updatedBroker.Name,
		BrokerURL: updatedBroker.Spec.URL,
	}, nil
}

// Fetch the new catalog information from reach service-broker registered in kubernetes,
// so that it is visible in the kubernetes service-catalog.
func (b *PlatformClient) Fetch(ctx context.Context, serviceBroker *platform.ServiceBroker) error {
	return syncClusterServiceBroker(b.cli, serviceBroker.Name, 3)
}

func newServiceBroker(name string, url string, secret *v1beta1.ObjectReference) *v1beta1.ClusterServiceBroker {
	return &v1beta1.ClusterServiceBroker{
		ObjectMeta: v1.ObjectMeta{
			Name: name,
		},
		Spec: v1beta1.ClusterServiceBrokerSpec{
			CommonServiceBrokerSpec: v1beta1.CommonServiceBrokerSpec{
				URL: url,
			},
			AuthInfo: &v1beta1.ClusterServiceBrokerAuthInfo{
				Basic: &v1beta1.ClusterBasicAuthConfig{
					SecretRef: secret,
				},
			},
		},
	}
}
