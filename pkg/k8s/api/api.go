package api

import (
	"github.com/kubernetes-sigs/service-catalog/pkg/apis/servicecatalog/v1beta1"
	v1core "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// KubernetesAPI interface for communicating with kubernetes cluster
//go:generate counterfeiter . KubernetesAPI
type KubernetesAPI interface {
	// CreateClusterServiceBroker creates cluster-wide visible service broker
	CreateClusterServiceBroker(broker *v1beta1.ClusterServiceBroker) (*v1beta1.ClusterServiceBroker, error)
	// DeleteClusterServiceBroker deletes cluster-wide visible service broker
	DeleteClusterServiceBroker(name string, options *v1.DeleteOptions) error
	// RetrieveClusterServiceBrokers gets all cluster-wide visible service brokers
	RetrieveClusterServiceBrokers() (*v1beta1.ClusterServiceBrokerList, error)
	// RetrieveClusterServiceBrokerByName gets cluster-wide visible service broker
	RetrieveClusterServiceBrokerByName(name string) (*v1beta1.ClusterServiceBroker, error)
	// UpdateClusterServiceBroker gets cluster-wide visible service broker
	UpdateClusterServiceBroker(broker *v1beta1.ClusterServiceBroker) (*v1beta1.ClusterServiceBroker, error)
	// SyncClusterServiceBroker synchronize a cluster-wide visible service broker
	SyncClusterServiceBroker(name string, retries int) error

	// CreateNamespaceServiceBroker creates namespace service broker
	CreateNamespaceServiceBroker(broker *v1beta1.ServiceBroker, namespace string) (*v1beta1.ServiceBroker, error)
	// DeleteNamespaceServiceBroker deletes a service broker in a namespace
	DeleteNamespaceServiceBroker(name string, namespace string, options *v1.DeleteOptions) error
	// RetrieveNamespaceServiceBrokers gets all service brokers in a namespace
	RetrieveNamespaceServiceBrokers(namespace string) (*v1beta1.ServiceBrokerList, error)
	// RetrieveNamespaceServiceBrokerByName gets a service broker in a namespace
	RetrieveNamespaceServiceBrokerByName(name, namespace string) (*v1beta1.ServiceBroker, error)
	// UpdateNamespaceServiceBroker updates a service broker in a namespace
	UpdateNamespaceServiceBroker(broker *v1beta1.ServiceBroker, namespace string) (*v1beta1.ServiceBroker, error)
	// SyncNamespaceServiceBroker synchronize a service broker in a namespace
	SyncNamespaceServiceBroker(name, namespace string, retries int) error

	// UpdateServiceBrokerCredentials updates broker's credentials secret
	UpdateServiceBrokerCredentials(secret *v1core.Secret) (*v1core.Secret, error)
	// CreateSecret creates a secret for broker's credentials
	CreateSecret(secret *v1core.Secret) (*v1core.Secret, error)
	// DeleteSecret deletes broker credentials secret
	DeleteSecret(namespace, name string) error
}
