package platform

// Client provides the logic for calling into the underlying platform and performing platform specific operations
//go:generate counterfeiter . Client
type Client interface {
	// GetBrokers obtains the registered brokers in the platform
	GetBrokers() ([]ServiceBroker, error)

	// CreateBroker registers a new broker at the platform
	CreateBroker(r *CreateServiceBrokerRequest) (*ServiceBroker, error)

	// DeleteBroker unregisters a broker from the platform
	DeleteBroker(r *DeleteServiceBrokerRequest) error

	// UpdateBroker updates a broker registration at the platform
	UpdateBroker(r *UpdateServiceBrokerRequest) (*ServiceBroker, error)
}
