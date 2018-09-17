package platform

import (
	"context"
	"encoding/json"
)

// ServiceAccess provides a way to add a hook for a platform specific way of enabling and disabling
// service and plan access.
//go:generate counterfeiter . ServiceAccess
type ServiceAccess interface {
	// EnableAccessForService enables the access to all plans of the service with the specified GUID
	// for the entities in the data
	EnableAccessForService(ctx context.Context, data json.RawMessage, serviceGUID string) error

	// EnableAccessForPlan enables the access to the plan with the specified GUID for
	// the entities in the data
	EnableAccessForPlan(ctx context.Context, data json.RawMessage, servicePlanGUID string) error

	// DisableAccessForService disables the access to all plans of the service with the specified GUID
	// for the entities in the data
	DisableAccessForService(ctx context.Context, data json.RawMessage, serviceGUID string) error

	// DisableAccessForPlan disables the access to the plan with the specified GUID for
	// the entities in the data
	DisableAccessForPlan(ctx context.Context, data json.RawMessage, servicePlanGUID string) error
}
