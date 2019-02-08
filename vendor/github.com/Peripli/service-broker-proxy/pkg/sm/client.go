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
	"fmt"
	"net/http"

	"github.com/Peripli/service-manager/pkg/types"

	"time"

	"context"

	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/pkg/errors"
)

// APIInternalBrokers is the SM API for obtaining the brokers for this proxy
const (
	APIInternalBrokers = "%s" + web.BrokersURL
	APIVisibilities    = "%s" + web.VisibilitiesURL
	APIPlans           = "%s" + web.ServicePlansURL
)

// Client provides the logic for calling into the Service Manager
//go:generate counterfeiter . Client
type Client interface {
	GetBrokers(ctx context.Context) ([]Broker, error)
	GetVisibilities(ctx context.Context) ([]*types.Visibility, error)
	GetPlans(ctx context.Context) ([]*types.ServicePlan, error)
}

// ServiceManagerClient allows consuming APIs from a Service Manager
type ServiceManagerClient struct {
	host       string
	httpClient *http.Client
}

// NewClient builds a new Service Manager Client from the provided configuration
func NewClient(config *Settings) (*ServiceManagerClient, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	httpClient := &http.Client{}
	httpClient.Timeout = time.Duration(config.RequestTimeout)
	tr := config.Transport

	if tr == nil {
		tr = &SkipSSLTransport{
			SkipSslValidation: config.SkipSSLValidation,
		}
	}

	httpClient.Transport = &BasicAuthTransport{
		Username: config.User,
		Password: config.Password,
		Rt:       tr,
	}

	return &ServiceManagerClient{
		host:       config.URL,
		httpClient: httpClient,
	}, nil
}

// GetBrokers calls the Service Manager in order to obtain all brokers that need to be registered
// in the service broker proxy
func (c *ServiceManagerClient) GetBrokers(ctx context.Context) ([]Broker, error) {
	log.C(ctx).Debugf("Getting brokers for proxy from Service Manager at %s", c.host)

	list := &Brokers{}
	err := c.call(ctx, fmt.Sprintf(APIInternalBrokers, c.host), list)
	if err != nil {
		return nil, errors.Wrap(err, "error getting brokers from Service Manager")
	}

	return list.Brokers, nil
}

// GetVisibilities returns plan visibilities from Service Manager
func (c *ServiceManagerClient) GetVisibilities(ctx context.Context) ([]*types.Visibility, error) {
	log.C(ctx).Debugf("Getting visibilities for proxy from Service Manager at %s", c.host)

	list := &types.Visibilities{}
	err := c.call(ctx, fmt.Sprintf(APIVisibilities, c.host), list)
	if err != nil {
		return nil, errors.Wrap(err, "error getting visibilities from Service Manager")
	}

	return list.Visibilities, nil
}

// GetPlans returns plans from Service Manager
func (c *ServiceManagerClient) GetPlans(ctx context.Context) ([]*types.ServicePlan, error) {
	log.C(ctx).Debugf("Getting service plans for proxy from Service Manager at %s", c.host)

	list := &struct {
		Plans []*types.ServicePlan `json:"service_plans"`
	}{}
	err := c.call(ctx, fmt.Sprintf(APIPlans, c.host), list)
	if err != nil {
		return nil, errors.Wrap(err, "error getting service plans from Service Manager")
	}

	return list.Plans, nil
}

func (c *ServiceManagerClient) call(ctx context.Context, url string, list interface{}) error {
	response, err := util.SendRequest(ctx, c.httpClient.Do, http.MethodGet, url, nil, nil)
	if err != nil {
		return err
	}

	if response.StatusCode != http.StatusOK {
		return errors.WithStack(util.HandleResponseError(response))
	}

	if err = util.BodyToObject(response.Body, list); err != nil {
		return errors.Wrapf(err, "error getting content from body of response with status %s", response.Status)
	}
	return nil
}
