package platform

// Fetcher provides a way to add a hook for platform specific way of refetching the service broker catalog on each
// run of the registration task. If the platform that this proxy represents already handles that, you don't
// have to implement this interface
type Fetcher interface {

	// Fetch contains the logic for platform specific catalog fetching for the provided service broker
	Fetch(serviceBroker *ServiceBroker) error
}
