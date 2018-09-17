package platform

import "context"

// Client provides the logic for calling into the underlying platform and performing platform specific operations
//go:generate counterfeiter . Client
type Client interface {
	// GetBrokers obtains the registered brokers in the platform
	GetBrokers(ctx context.Context) ([]ServiceBroker, error)

	// CreateBroker registers a new broker at the platform
	CreateBroker(ctx context.Context, r *CreateServiceBrokerRequest) (*ServiceBroker, error)

	// DeleteBroker unregisters a broker from the platform
	DeleteBroker(ctx context.Context, r *DeleteServiceBrokerRequest) error

	// UpdateBroker updates a broker registration at the platform
	UpdateBroker(ctx context.Context, r *UpdateServiceBrokerRequest) (*ServiceBroker, error)
}
