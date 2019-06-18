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

package reconcile

import (
	"fmt"
	"github.com/pkg/errors"
)

// DefaultProxyBrokerPrefix prefix for brokers registered by the proxy
const DefaultProxyBrokerPrefix = "sm-"

// Settings type represents the sbproxy settings
type Settings struct {
	URL      string `mapstructure:"url"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`

	BrokerPrefix string `mapstructure:"broker_prefix"`
}

// DefaultSettings creates default proxy settings
func DefaultSettings() *Settings {
	return &Settings{
		URL:          "",
		Username:     "",
		Password:     "",
		BrokerPrefix: DefaultProxyBrokerPrefix,
	}
}

// Validate validates that the configuration contains all mandatory properties
func (c *Settings) Validate() error {
	if c.URL == "" {
		return fmt.Errorf("validate settings: missing url")
	}
	if len(c.Username) == 0 {
		return errors.New("validate settings: missing username")
	}
	if len(c.Password) == 0 {
		return errors.New("validate settings: missing password")
	}
	return nil
}
