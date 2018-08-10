package sm

import (
	"encoding/json"

	"github.com/Peripli/service-manager/pkg/types"
	osbc "github.com/pmorie/go-open-service-broker-client/v2"
)

// Brokers type used for responses from the Service Manager client
type Brokers struct {
	Brokers []Broker `json:"brokers"`
}

// Broker type used for responses from the Service Manager client
type Broker struct {
	*types.Broker
	Catalog  *osbc.CatalogResponse      `json:"catalog"`
	Metadata map[string]json.RawMessage `json:"metadata,omitempty"`
}
