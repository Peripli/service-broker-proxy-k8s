package platform

import "encoding/json"
import osbc "github.com/pmorie/go-open-service-broker-client/v2"

// CreateServiceBrokerRequest type used for requests by the platform client
type CreateServiceBrokerRequest struct {
	Name      string `json:"name"`
	BrokerURL string `json:"broker_url"`
}

// UpdateServiceBrokerRequest type used for requests by the platform client
type UpdateServiceBrokerRequest struct {
	GUID      string `json:"guid"`
	Name      string `json:"name"`
	BrokerURL string `json:"broker_url"`
}

// DeleteServiceBrokerRequest type used for requests by the platform client
type DeleteServiceBrokerRequest struct {
	GUID string `json:"guid"`
	Name string `json:"name"`
}

// ServiceBroker type for responses from the platform client
type ServiceBroker struct {
	GUID      string                     `json:"guid"`
	Name      string                     `json:"name"`
	BrokerURL string                     `json:"broker_url"`
	Catalog   *osbc.CatalogResponse      `json:"catalog"`
	Metadata  map[string]json.RawMessage `json:"metadata"`
}

// ServiceBrokerList type for responses from the platform client
type ServiceBrokerList struct {
	ServiceBrokers []ServiceBroker `json:"service_brokers"`
}
