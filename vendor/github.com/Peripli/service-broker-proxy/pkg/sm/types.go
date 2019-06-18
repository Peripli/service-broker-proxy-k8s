/*
 * Copyright 2018 The Service Manager Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package sm

import (
	"encoding/json"

	"github.com/Peripli/service-manager/pkg/types"
)

// Brokers type used for responses from the Service Manager client
type Brokers struct {
	Brokers []Broker `json:"service_brokers"`
}

// Broker type used for responses from the Service Manager client
type Broker struct {
	ID               string                     `json:"id"`
	Name             string                     `json:"name"`
	BrokerURL        string                     `json:"broker_url"`
	ServiceOfferings []types.ServiceOffering    `json:"services"`
	Metadata         map[string]json.RawMessage `json:"metadata,omitempty"`
}
