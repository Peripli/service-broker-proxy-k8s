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
	"net/url"
	"strings"

	"github.com/Peripli/service-manager/pkg/types"

	"time"

	"context"

	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/pkg/errors"
)

const (
	// APIInternalBrokers is the SM API for obtaining the brokers for this proxy
	APIInternalBrokers = "%s" + web.ServiceBrokersURL

	// APIVisibilities is the SM API for obtaining plan visibilities
	APIVisibilities = "%s" + web.VisibilitiesURL

	// APIPlans is the SM API for obtaining service plans
	APIPlans = "%s" + web.ServicePlansURL

	// APIServiceOfferings is the SM API for obtaining service offerings
	APIServiceOfferings = "%s" + web.ServiceOfferingsURL
)

// Client provides the logic for calling into the Service Manager
//go:generate counterfeiter . Client
type Client interface {
	GetBrokers(ctx context.Context) ([]*types.ServiceBroker, error)
	GetVisibilities(ctx context.Context) ([]*types.Visibility, error)
	GetPlans(ctx context.Context) ([]*types.ServicePlan, error)
	GetServiceOfferingsByBrokerIDs(ctx context.Context, brokerIDs []string) ([]*types.ServiceOffering, error)
	GetPlansByServiceOfferings(ctx context.Context, sos []*types.ServiceOffering) ([]*types.ServicePlan, error)
}

// ServiceManagerClient allows consuming Service Manager APIs
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
func (c *ServiceManagerClient) GetBrokers(ctx context.Context) ([]*types.ServiceBroker, error) {
	log.C(ctx).Debugf("Getting brokers for proxy from Service Manager at %s", c.host)

	result := make([]*types.ServiceBroker, 0)
	err := c.call(ctx, fmt.Sprintf(APIInternalBrokers, c.host), nil, &result)
	if err != nil {
		return nil, errors.Wrap(err, "error getting brokers from Service Manager")
	}

	return result, nil
}

// GetVisibilities returns plan visibilities from Service Manager
func (c *ServiceManagerClient) GetVisibilities(ctx context.Context) ([]*types.Visibility, error) {
	log.C(ctx).Debugf("Getting visibilities for proxy from Service Manager at %s", c.host)

	result := make([]*types.Visibility, 0)
	err := c.call(ctx, fmt.Sprintf(APIVisibilities, c.host), nil, &result)
	if err != nil {
		return nil, errors.Wrap(err, "error getting visibilities from Service Manager")
	}

	return result, nil
}

// GetPlans returns plans from Service Manager
func (c *ServiceManagerClient) GetPlans(ctx context.Context) ([]*types.ServicePlan, error) {
	log.C(ctx).Debugf("Getting service plans for proxy from Service Manager at %s", c.host)

	result := make([]*types.ServicePlan, 0)
	err := c.call(ctx, fmt.Sprintf(APIPlans, c.host), nil, &result)
	if err != nil {
		return nil, errors.Wrap(err, "error getting service plans from Service Manager")
	}

	return result, nil
}

// GetServiceOfferingsByBrokerIDs returns plans from Service Manager
func (c *ServiceManagerClient) GetServiceOfferingsByBrokerIDs(ctx context.Context, brokerIDs []string) ([]*types.ServiceOffering, error) {
	log.C(ctx).Debugf("Getting service offerings from Service Manager at %s", c.host)

	fieldQuery := fmt.Sprintf("broker_id in ('%s')", strings.Join(brokerIDs, "','"))
	params := map[string]string{
		"fieldQuery": fieldQuery,
	}

	result := make([]*types.ServiceOffering, 0)
	err := c.call(ctx, fmt.Sprintf(APIServiceOfferings, c.host), params, &result)
	if err != nil {
		return nil, errors.Wrap(err, "error getting service offerings from Service Manager")
	}

	return result, nil
}

// GetPlansByServiceOfferings returns plans from Service Manager
func (c *ServiceManagerClient) GetPlansByServiceOfferings(ctx context.Context, sos []*types.ServiceOffering) ([]*types.ServicePlan, error) {
	log.C(ctx).Debugf("Getting service plans from Service Manager at %s", c.host)

	soIDs := ""
	for _, so := range sos {
		soIDs += ",'" + so.ID + "'"
	}
	soIDs = soIDs[1:]

	fieldQuery := fmt.Sprintf("service_offering_id in (%s)", soIDs)
	params := map[string]string{
		"fieldQuery": fieldQuery,
	}

	result := make([]*types.ServicePlan, 0)
	err := c.call(ctx, fmt.Sprintf(APIPlans, c.host), params, &result)
	if err != nil {
		return nil, errors.Wrap(err, "error getting service plans from Service Manager")
	}

	return result, nil
}

func (c *ServiceManagerClient) call(ctx context.Context, smURL string, params map[string]string, list interface{}) error {
	fullURL, err := url.Parse(smURL)
	if err != nil {
		return err
	}
	q := fullURL.Query()
	for k, v := range params {
		q.Add(k,v)
	}
	fullURL.RawQuery = q.Encode()
	return util.ListAll(ctx, c.httpClient.Do, fullURL.String(), list)
}
