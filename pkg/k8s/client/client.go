package client

import (
	"context"
	"fmt"
	"github.com/Peripli/service-broker-proxy-k8s/pkg/k8s/api"
	"github.com/Peripli/service-broker-proxy-k8s/pkg/k8s/config"
	servicecatalog "github.com/kubernetes-sigs/service-catalog/pkg/svcat/service-catalog"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"strings"
	"sync"

	"github.com/Peripli/service-broker-proxy/pkg/platform"
	"github.com/kubernetes-sigs/service-catalog/pkg/apis/servicecatalog/v1beta1"
	v1core "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const resyncBrokerRetryCount = 3

// NewDefaultKubernetesAPI returns default kubernetes api interface
func NewDefaultKubernetesAPI(cli *servicecatalog.SDK) *ServiceCatalogAPI {
	return &ServiceCatalogAPI{
		SDK:               cli,
		brokersInProgress: make(map[string]bool),
		lock:              &sync.Mutex{},
	}
}

// ServiceCatalogAPI uses service catalog SDK to interact with the kubernetes resources
type ServiceCatalogAPI struct {
	*servicecatalog.SDK
	brokersInProgress map[string]bool
	lock              *sync.Mutex
}

// CreateNamespaceServiceBroker creates namespace service broker
func (sca *ServiceCatalogAPI) CreateNamespaceServiceBroker(broker *v1beta1.ServiceBroker, namespace string) (*v1beta1.ServiceBroker, error) {
	return sca.ServiceCatalog().ServiceBrokers(namespace).Create(broker)
}

// CreateClusterServiceBroker creates a cluster service broker
func (sca *ServiceCatalogAPI) CreateClusterServiceBroker(broker *v1beta1.ClusterServiceBroker) (*v1beta1.ClusterServiceBroker, error) {
	return sca.ServiceCatalog().ClusterServiceBrokers().Create(broker)
}

// DeleteNamespaceServiceBroker deletes a service broker in a namespace
func (sca *ServiceCatalogAPI) DeleteNamespaceServiceBroker(name string, namespace string, options *v1.DeleteOptions) error {
	return sca.ServiceCatalog().ServiceBrokers(namespace).Delete(name, options)
}

// DeleteClusterServiceBroker deletes a cluster service broker
func (sca *ServiceCatalogAPI) DeleteClusterServiceBroker(name string, options *v1.DeleteOptions) error {
	return sca.ServiceCatalog().ClusterServiceBrokers().Delete(name, options)
}

// RetrieveNamespaceServiceBrokers gets all service brokers in a namespace
func (sca *ServiceCatalogAPI) RetrieveNamespaceServiceBrokers(namespace string) (*v1beta1.ServiceBrokerList, error) {
	return sca.ServiceCatalog().ServiceBrokers(namespace).List(v1.ListOptions{})
}

// RetrieveClusterServiceBrokers returns all cluster service brokers
func (sca *ServiceCatalogAPI) RetrieveClusterServiceBrokers() (*v1beta1.ClusterServiceBrokerList, error) {
	return sca.ServiceCatalog().ClusterServiceBrokers().List(v1.ListOptions{})
}

// RetrieveNamespaceServiceBrokerByName gets a service broker in a namespace
func (sca *ServiceCatalogAPI) RetrieveNamespaceServiceBrokerByName(name, namespace string) (*v1beta1.ServiceBroker, error) {
	return sca.ServiceCatalog().ServiceBrokers(namespace).Get(name, v1.GetOptions{})
}

// RetrieveClusterServiceBrokerByName returns a cluster service broker by name
func (sca *ServiceCatalogAPI) RetrieveClusterServiceBrokerByName(name string) (*v1beta1.ClusterServiceBroker, error) {
	return sca.ServiceCatalog().ClusterServiceBrokers().Get(name, v1.GetOptions{})
}

// UpdateNamespaceServiceBroker updates a service broker in a namespace
func (sca *ServiceCatalogAPI) UpdateNamespaceServiceBroker(broker *v1beta1.ServiceBroker, namespace string) (*v1beta1.ServiceBroker, error) {
	return sca.ServiceCatalog().ServiceBrokers(namespace).Update(broker)
}

// UpdateClusterServiceBroker updates a cluster service broker
func (sca *ServiceCatalogAPI) UpdateClusterServiceBroker(broker *v1beta1.ClusterServiceBroker) (*v1beta1.ClusterServiceBroker, error) {
	return sca.ServiceCatalog().ClusterServiceBrokers().Update(broker)
}

// SyncNamespaceServiceBroker synchronize a service broker in a namespace
func (sca *ServiceCatalogAPI) SyncNamespaceServiceBroker(name, namespace string, retries int) error {
	if sca.setBrokerInProgress(name) {
		defer sca.unsetBrokerInProgress(name)
		err := sca.Sync(name, servicecatalog.ScopeOptions{
			Scope:     servicecatalog.NamespaceScope,
			Namespace: namespace,
		}, retries)
		return err
	}
	return nil
}

// SyncClusterServiceBroker synchronizes a cluster service broker including its catalog
func (sca *ServiceCatalogAPI) SyncClusterServiceBroker(name string, retries int) error {
	if sca.setBrokerInProgress(name) {
		defer sca.unsetBrokerInProgress(name)
		err := sca.Sync(name, servicecatalog.ScopeOptions{
			Scope: servicecatalog.ClusterScope,
		}, retries)
		return err
	}
	return nil
}

// UpdateServiceBrokerCredentials updates broker's credentials secret
func (sca *ServiceCatalogAPI) UpdateServiceBrokerCredentials(secret *v1core.Secret) (*v1core.Secret, error) {
	_, err := sca.K8sClient.CoreV1().Secrets(secret.Namespace).Get(secret.Name, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return sca.CreateSecret(secret)
		}
		return nil, err
	}
	return sca.K8sClient.CoreV1().Secrets(secret.Namespace).Update(secret)
}

// CreateSecret creates a secret for broker's credentials
func (sca *ServiceCatalogAPI) CreateSecret(secret *v1core.Secret) (*v1core.Secret, error) {
	return sca.K8sClient.CoreV1().Secrets(secret.Namespace).Create(secret)
}

// DeleteSecret deletes broker credentials secret
func (sca *ServiceCatalogAPI) DeleteSecret(namespace, name string) error {
	return sca.K8sClient.CoreV1().Secrets(namespace).Delete(name, &v1.DeleteOptions{})
}

func (sca *ServiceCatalogAPI) setBrokerInProgress(name string) bool {
	sca.lock.Lock()
	defer sca.lock.Unlock()
	if _, ok := sca.brokersInProgress[name]; !ok {
		sca.brokersInProgress[name] = true
		return true
	}
	return false
}

func (sca *ServiceCatalogAPI) unsetBrokerInProgress(name string) {
	sca.lock.Lock()
	defer sca.lock.Unlock()
	delete(sca.brokersInProgress, name)
}

// PlatformClient implements all broker, visibility and catalog specific operations for kubernetes
type PlatformClient struct {
	platformAPI     api.KubernetesAPI
	secretNamespace string
	targetNamespace string
}

type brokersByUID map[types.UID]servicecatalog.Broker

var _ platform.Client = &PlatformClient{}

// NewClient create a client to communicate with the kubernetes service-catalog.
func NewClient(settings *config.Settings) (*PlatformClient, error) {
	if err := settings.Validate(); err != nil {
		return nil, err
	}
	svcatSDK, err := settings.K8S.K8sClientCreateFunc(settings.K8S.ClientSettings)
	if err != nil {
		return nil, err
	}
	return &PlatformClient{
		platformAPI:     NewDefaultKubernetesAPI(svcatSDK),
		secretNamespace: settings.K8S.Secret.Namespace,
		targetNamespace: settings.K8S.TargetNamespace,
	}, nil
}

// Broker returns the platform client which handles broker operations
func (pc *PlatformClient) Broker() platform.BrokerClient {
	return pc
}

// CatalogFetcher returns the platform client which handles catalog fetch operations
func (pc *PlatformClient) CatalogFetcher() platform.CatalogFetcher {
	return pc
}

// Visibility returns the platform client which handles visibility operations
func (pc *PlatformClient) Visibility() platform.VisibilityClient {
	return pc
}

// GetBrokers returns all service-brokers currently registered in kubernetes service-catalog.
func (pc *PlatformClient) GetBrokers(ctx context.Context) ([]*platform.ServiceBroker, error) {
	var clientBrokers = make([]*platform.ServiceBroker, 0)
	var brokers brokersByUID

	if pc.isClusterScoped() {
		clusterBrokers, err := pc.platformAPI.RetrieveClusterServiceBrokers()
		if err != nil {
			return nil, fmt.Errorf("unable to list cluster-scoped brokers (%s)", err)
		}

		brokers = clusterBrokersToBrokers(clusterBrokers)
	} else {
		namespaceBrokers, err := pc.platformAPI.RetrieveNamespaceServiceBrokers(pc.targetNamespace)
		if err != nil {
			return nil, fmt.Errorf("unable to list namespace-scoped brokers (%s)", err)
		}

		brokers = namespaceBrokersToBrokers(namespaceBrokers)
	}

	for uid, broker := range brokers {
		serviceBroker := &platform.ServiceBroker{
			GUID:      string(uid),
			Name:      broker.GetName(),
			BrokerURL: broker.GetURL(),
		}
		clientBrokers = append(clientBrokers, serviceBroker)
	}

	return clientBrokers, nil
}

// GetBrokerByName returns the service-broker with the specified name currently registered in kubernetes service-catalog with.
func (pc *PlatformClient) GetBrokerByName(ctx context.Context, name string) (*platform.ServiceBroker, error) {
	lowerCaseBrokerName := strings.ToLower(name)
	var broker servicecatalog.Broker
	var brokerUID types.UID

	if pc.isClusterScoped() {
		clusterBroker, err := pc.platformAPI.RetrieveClusterServiceBrokerByName(lowerCaseBrokerName)
		if err != nil {
			return nil, fmt.Errorf("unable to get cluster-scoped broker (%s)", err)
		}

		broker, brokerUID = clusterBroker, clusterBroker.GetUID()
	} else {
		namespaceBroker, err := pc.platformAPI.RetrieveNamespaceServiceBrokerByName(lowerCaseBrokerName, pc.targetNamespace)
		if err != nil {
			return nil, fmt.Errorf("unable to get namespace-scoped broker (%s)", err)
		}

		broker, brokerUID = namespaceBroker, namespaceBroker.GetUID()
	}

	return &platform.ServiceBroker{
		GUID:      string(brokerUID),
		Name:      broker.GetName(),
		BrokerURL: broker.GetURL(),
	}, nil
}

// CreateBroker registers a new broker in kubernetes service-catalog.
func (pc *PlatformClient) CreateBroker(ctx context.Context, r *platform.CreateServiceBrokerRequest) (*platform.ServiceBroker, error) {
	lowerCaseBrokerName := strings.ToLower(r.Name)
	if err := pc.updateBrokerPlatformSecret(r.ID, r.Username, r.Password); err != nil {
		return nil, err
	}
	var brokerUID types.UID

	if pc.isClusterScoped() {
		broker := newClusterServiceBroker(lowerCaseBrokerName, r.BrokerURL, &v1beta1.ObjectReference{
			Name:      r.ID,
			Namespace: pc.secretNamespace,
		})

		broker.Spec.CommonServiceBrokerSpec.RelistBehavior = "Manual"

		csb, err := pc.platformAPI.CreateClusterServiceBroker(broker)
		if err != nil {
			return nil, err
		}
		brokerUID = csb.GetUID()
	} else {
		broker := newNamespaceServiceBroker(lowerCaseBrokerName, r.BrokerURL, &v1beta1.LocalObjectReference{
			Name: r.ID,
		})
		broker.Spec.CommonServiceBrokerSpec.RelistBehavior = "Manual"

		csb, err := pc.platformAPI.CreateNamespaceServiceBroker(broker, pc.targetNamespace)
		if err != nil {
			return nil, err
		}
		brokerUID = csb.GetUID()
	}

	return &platform.ServiceBroker{
		GUID:      string(brokerUID),
		Name:      lowerCaseBrokerName,
		BrokerURL: r.BrokerURL,
	}, nil

}

// DeleteBroker deletes an existing broker in from kubernetes service-catalog.
func (pc *PlatformClient) DeleteBroker(ctx context.Context, r *platform.DeleteServiceBrokerRequest) error {
	lowerCaseBrokerName := strings.ToLower(r.Name)
	if pc.isClusterScoped() {
		if err := pc.platformAPI.DeleteSecret(pc.secretNamespace, r.ID); err != nil {
			return fmt.Errorf("error deleting broker credentials secret: %v", err)
		}
		return pc.platformAPI.DeleteClusterServiceBroker(lowerCaseBrokerName, &v1.DeleteOptions{})
	}

	if err := pc.platformAPI.DeleteSecret(pc.targetNamespace, r.ID); err != nil {
		return fmt.Errorf("error deleting broker credentials secret in namespace %s: %v", pc.targetNamespace, err)
	}
	return pc.platformAPI.DeleteNamespaceServiceBroker(lowerCaseBrokerName, pc.targetNamespace, &v1.DeleteOptions{})

}

// UpdateBroker updates a service broker in the kubernetes service-catalog.
func (pc *PlatformClient) UpdateBroker(ctx context.Context, r *platform.UpdateServiceBrokerRequest) (*platform.ServiceBroker, error) {
	lowerCaseBrokerName := strings.ToLower(r.Name)
	if r.Username != "" && r.Password != "" {
		if err := pc.updateBrokerPlatformSecret(r.ID, r.Username, r.Password); err != nil {
			return nil, err
		}
	}

	var updatedBrokerUID types.UID
	var updatedBroker servicecatalog.Broker

	if pc.isClusterScoped() {
		// Only broker url and secret-references are updateable
		broker := newClusterServiceBroker(lowerCaseBrokerName, r.BrokerURL, &v1beta1.ObjectReference{
			Name:      r.ID,
			Namespace: pc.secretNamespace,
		})

		updatedClusterBroker, err := pc.platformAPI.UpdateClusterServiceBroker(broker)
		if err != nil {
			return nil, err
		}

		updatedBroker, updatedBrokerUID = updatedClusterBroker, updatedClusterBroker.GetUID()
	} else {
		// Only broker url and secret-references are updateable
		broker := newNamespaceServiceBroker(lowerCaseBrokerName, r.BrokerURL, &v1beta1.LocalObjectReference{
			Name: r.ID,
		})

		updatedNamespaceBroker, err := pc.platformAPI.UpdateNamespaceServiceBroker(broker, pc.targetNamespace)
		if err != nil {
			return nil, err
		}

		updatedBroker, updatedBrokerUID = updatedNamespaceBroker, updatedNamespaceBroker.GetUID()
	}

	return &platform.ServiceBroker{
		GUID:      string(updatedBrokerUID),
		Name:      updatedBroker.GetName(),
		BrokerURL: updatedBroker.GetURL(),
	}, nil
}

// Fetch the new catalog information from reach service-broker registered in kubernetes,
// so that it is visible in the kubernetes service-catalog.
func (pc *PlatformClient) Fetch(ctx context.Context, r *platform.UpdateServiceBrokerRequest) error {
	lowerCaseBrokerName := strings.ToLower(r.Name)
	if r.Username != "" && r.Password != "" {
		if err := pc.updateBrokerPlatformSecret(r.ID, r.Username, r.Password); err != nil {
			return err
		}
	}

	if pc.isClusterScoped() {
		return pc.platformAPI.SyncClusterServiceBroker(lowerCaseBrokerName, resyncBrokerRetryCount)
	}

	return pc.platformAPI.SyncNamespaceServiceBroker(lowerCaseBrokerName, pc.targetNamespace, resyncBrokerRetryCount)
}

func (pc *PlatformClient) updateBrokerPlatformSecret(name, username, password string) error {
	var secretNamespace string
	if pc.isClusterScoped() {
		secretNamespace = pc.secretNamespace
	} else {
		secretNamespace = pc.targetNamespace
	}

	secret := newServiceBrokerCredentialsSecret(secretNamespace, name, username, password)
	_, err := pc.platformAPI.UpdateServiceBrokerCredentials(secret)
	if err != nil {
		return fmt.Errorf("error updating broker credentials secret in namespace %s: %v", secretNamespace, err)
	}

	return nil
}

func clusterBrokersToBrokers(clusterBrokers *v1beta1.ClusterServiceBrokerList) brokersByUID {
	brokers := make(brokersByUID, len(clusterBrokers.Items))

	for _, clusterBroker := range clusterBrokers.Items {
		brokers[clusterBroker.ObjectMeta.UID] = &clusterBroker
	}

	return brokers
}

func namespaceBrokersToBrokers(namespaceBrokers *v1beta1.ServiceBrokerList) brokersByUID {
	brokers := make(brokersByUID, len(namespaceBrokers.Items))

	for _, clusterBroker := range namespaceBrokers.Items {
		brokers[clusterBroker.ObjectMeta.UID] = &clusterBroker
	}

	return brokers
}

func newClusterServiceBroker(name string, url string, secret *v1beta1.ObjectReference) *v1beta1.ClusterServiceBroker {
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

func newNamespaceServiceBroker(name string, url string, secret *v1beta1.LocalObjectReference) *v1beta1.ServiceBroker {
	return &v1beta1.ServiceBroker{
		ObjectMeta: v1.ObjectMeta{
			Name: name,
		},
		Spec: v1beta1.ServiceBrokerSpec{
			CommonServiceBrokerSpec: v1beta1.CommonServiceBrokerSpec{
				URL: url,
			},
			AuthInfo: &v1beta1.ServiceBrokerAuthInfo{
				Basic: &v1beta1.BasicAuthConfig{
					SecretRef: secret,
				},
			},
		},
	}
}

func newServiceBrokerCredentialsSecret(namespace, name, username, password string) *v1core.Secret {
	return &v1core.Secret{
		ObjectMeta: v1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
		Data: map[string][]byte{
			"password": []byte(password),
			"username": []byte(username),
		},
	}
}

func (pc *PlatformClient) isClusterScoped() bool {
	return len(pc.targetNamespace) == 0
}

// GetVisibilitiesByBrokers get currently available visibilities in the platform for specific broker names
func (pc *PlatformClient) GetVisibilitiesByBrokers(ctx context.Context, brokers []string) ([]*platform.Visibility, error) {
	// This will cause all brokers to re-fetch their catalogs
	return nil, nil
}

// VisibilityScopeLabelKey returns a specific label key which should be used when converting SM visibilities to platform.Visibilities
func (pc *PlatformClient) VisibilityScopeLabelKey() string {
	return ""
}

// EnableAccessForPlan enables the access for the specified plan
func (pc *PlatformClient) EnableAccessForPlan(ctx context.Context, request *platform.ModifyPlanAccessRequest) error {
	if pc.isClusterScoped() {
		return pc.platformAPI.SyncClusterServiceBroker(request.BrokerName, resyncBrokerRetryCount)
	}

	return pc.platformAPI.SyncNamespaceServiceBroker(request.BrokerName, pc.targetNamespace, resyncBrokerRetryCount)
}

// DisableAccessForPlan disables the access for the specified plan
func (pc *PlatformClient) DisableAccessForPlan(ctx context.Context, request *platform.ModifyPlanAccessRequest) error {
	if pc.isClusterScoped() {
		return pc.platformAPI.SyncClusterServiceBroker(request.BrokerName, resyncBrokerRetryCount)
	}

	return pc.platformAPI.SyncNamespaceServiceBroker(request.BrokerName, pc.targetNamespace, resyncBrokerRetryCount)
}
