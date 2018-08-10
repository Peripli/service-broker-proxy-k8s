package platform

import "encoding/json"

// ServiceAccess provides a way to add a hook for a platform specific way of enabling and disabling
// service and plan access.
//go:generate counterfeiter . ServiceAccess
type ServiceAccess interface {

	// EnableAccessForService enables the access to all plans of the service with the specified GUID
	// for the entities in the context
	EnableAccessForService(context json.RawMessage, serviceGUID string) error

	// EnableAccessForPlan enables the access to the plan with the specified GUID for
	// the entities in the context
	EnableAccessForPlan(context json.RawMessage, servicePlanGUID string) error

	// DisableAccessForService disables the access to all plans of the service with the specified GUID
	// for the entities in the context
	DisableAccessForService(context json.RawMessage, serviceGUID string) error

	// DisableAccessForPlan disables the access to the plan with the specified GUID for
	// the entities in the context
	DisableAccessForPlan(context json.RawMessage, servicePlanGUID string) error
}
