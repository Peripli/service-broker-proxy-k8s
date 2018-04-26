package platform

type CreateServiceBrokerRequest struct {
	Name      string `json:"name"`
	BrokerURL string `json:"broker_url"`
}

type UpdateServiceBrokerRequest struct {
	Guid      string `json:"guid"`
	Name      string `json:"name"`
	BrokerURL string `json:"broker_url"`
}

type DeleteServiceBrokerRequest struct {
	Guid string `json:"guid"`
	Name string `json:"name"`
}

type ServiceBroker struct {
	Guid      string `json:"guid"`
	Name      string `json:"name"`
	BrokerURL string `json:"broker_url"`
}

type ServiceBrokerList struct {
	ServiceBrokers []ServiceBroker `json:"service_brokers"`
}
