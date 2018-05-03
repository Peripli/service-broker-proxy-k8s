package platform

// Client configuration provides the logic for configuring and creating a client to be used to perform platform specific operations
type ClientConfiguration interface {

	// CreateFunc creates a new platform client that will be used to communicate with the platform
	CreateFunc() (Client, error)

	// Validate validates the configuration properties
	Validate() error
}
