/*
 * Copyright 2019 The Service Manager Authors
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

package sbproxy

import (
	"sync"

	"github.com/Peripli/service-manager/pkg/types"

	"github.com/Peripli/service-broker-proxy/pkg/sbproxy/notifications/handlers"

	"fmt"

	"github.com/Peripli/service-broker-proxy/pkg/filter"
	"github.com/Peripli/service-broker-proxy/pkg/logging"
	"github.com/Peripli/service-manager/api/healthcheck"
	"github.com/Peripli/service-manager/pkg/health"
	"github.com/Peripli/service-manager/pkg/log"
	secfilters "github.com/Peripli/service-manager/pkg/security/filters"
	"github.com/Peripli/service-manager/pkg/util"

	"context"
	"time"

	"github.com/Peripli/service-broker-proxy/pkg/platform"
	"github.com/Peripli/service-broker-proxy/pkg/sbproxy/notifications"
	"github.com/Peripli/service-broker-proxy/pkg/sbproxy/reconcile"
	"github.com/Peripli/service-broker-proxy/pkg/sm"
	"github.com/Peripli/service-manager/api/filters"
	smosb "github.com/Peripli/service-manager/api/osb"
	"github.com/Peripli/service-manager/pkg/env"
	"github.com/Peripli/service-manager/pkg/server"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/spf13/pflag"
)

const (
	// BrokerPathParam for the broker id
	BrokerPathParam = "brokerID"

	// APIPrefix for the Proxy OSB API
	APIPrefix = "/v1/osb"

	// Path for the Proxy OSB API
	Path = APIPrefix + "/{" + BrokerPathParam + "}"
)

// SMProxyBuilder type is an extension point that allows adding additional filters, plugins and
// controllers before running SMProxy.
type SMProxyBuilder struct {
	*web.API

	ctx                   context.Context
	cfg                   *Settings
	group                 *sync.WaitGroup
	reconciler            *reconcile.Reconciler
	notificationsProducer *notifications.Producer
}

// SMProxy  struct
type SMProxy struct {
	*server.Server

	ctx                   context.Context
	group                 *sync.WaitGroup
	reconciler            *reconcile.Reconciler
	notificationsProducer *notifications.Producer
}

// DefaultEnv creates a default environment that can be used to boot up a Service Broker proxy
func DefaultEnv(additionalPFlags ...func(set *pflag.FlagSet)) (env.Environment, error) {
	set := pflag.NewFlagSet("Configuration Flags", pflag.ExitOnError)

	AddPFlags(set)
	for _, addFlags := range additionalPFlags {
		addFlags(set)
	}
	return env.New(set)
}

// New creates service broker proxy that is configured from the provided environment and platform client.
func New(ctx context.Context, cancel context.CancelFunc, settings *Settings, platformClient platform.Client) (*SMProxyBuilder, error) {
	if err := settings.Validate(); err != nil {
		return nil, fmt.Errorf("error validating settings: %s", err)
	}

	ctx = log.Configure(ctx, settings.Log)
	log.AddHook(&logging.ErrorLocationHook{})

	util.HandleInterrupts(ctx, cancel)

	api := &web.API{
		Controllers: []web.Controller{
			&smosb.Controller{
				BrokerFetcher: func(ctx context.Context, brokerID string) (*types.ServiceBroker, error) {
					return &types.ServiceBroker{
						Base: types.Base{
							ID: brokerID,
						},
						BrokerURL: fmt.Sprintf("%s%s/%s", settings.Sm.URL, settings.Sm.OSBAPIPath, brokerID),
						Credentials: &types.Credentials{
							Basic: &types.Basic{
								Username: settings.Sm.User,
								Password: settings.Sm.Password,
							},
						},
					}, nil
				},
			},
		},
		Filters: []web.Filter{
			&filters.Logging{},
			filter.NewBasicAuthnFilter(settings.Sm.User, settings.Sm.Password),
			secfilters.NewRequiredAuthnFilter(),
		},
		Registry: health.NewDefaultRegistry(),
	}

	smClient, err := sm.NewClient(settings.Sm)
	if err != nil {
		return nil, fmt.Errorf("error create service manager client: %s", err)
	}

	notificationsProducer, err := notifications.NewProducer(settings.Producer, settings.Sm)
	if err != nil {
		return nil, fmt.Errorf("error creating notifications producer: %s", err)
	}

	smPath := settings.Reconcile.URL + APIPrefix
	proxyPathPattern := settings.Reconcile.LegacyURL + APIPrefix + "/%s"

	resyncer := reconcile.NewResyncer(settings.Reconcile, platformClient, smClient, smPath, proxyPathPattern)
	consumer := &notifications.Consumer{
		Handlers: map[types.ObjectType]notifications.ResourceNotificationHandler{
			types.ServiceBrokerType: &handlers.BrokerResourceNotificationsHandler{
				BrokerClient:   platformClient.Broker(),
				CatalogFetcher: platformClient.CatalogFetcher(),
				ProxyPrefix:    settings.Reconcile.BrokerPrefix,
				SMPath:         smPath,
			},
			types.VisibilityType: &handlers.VisibilityResourceNotificationsHandler{
				VisibilityClient: platformClient.Visibility(),
				ProxyPrefix:      settings.Reconcile.BrokerPrefix,
			},
		},
	}
	reconciler := &reconcile.Reconciler{
		Resyncer: resyncer,
		Consumer: consumer,
	}
	var group sync.WaitGroup
	return &SMProxyBuilder{
		API:                   api,
		ctx:                   ctx,
		cfg:                   settings,
		group:                 &group,
		reconciler:            reconciler,
		notificationsProducer: notificationsProducer,
	}, nil
}

// Build builds the Service Manager
func (smb *SMProxyBuilder) Build() *SMProxy {
	smb.installHealth()

	srv := server.New(smb.cfg.Server, smb.API)
	srv.Use(filters.NewRecoveryMiddleware())

	return &SMProxy{
		Server:                srv,
		ctx:                   smb.ctx,
		group:                 smb.group,
		reconciler:            smb.reconciler,
		notificationsProducer: smb.notificationsProducer,
	}
}

func (smb *SMProxyBuilder) installHealth() {
	if len(smb.HealthIndicators) > 0 {
		smb.RegisterControllers(healthcheck.NewController(smb.HealthIndicators, smb.HealthAggregationPolicy))
	}
}

// Run starts the proxy
func (p *SMProxy) Run() {
	defer waitWithTimeout(p.ctx, p.group, p.Server.Config.ShutdownTimeout)

	messages := p.notificationsProducer.Start(p.ctx, p.group)
	p.reconciler.Reconcile(p.ctx, messages, p.group)

	log.C(p.ctx).Info("Running SBProxy...")
	p.Server.Run(p.ctx, p.group)

	p.group.Wait()
}

// waitWithTimeout waits for a WaitGroup to finish for a certain duration and times out afterwards
// WaitGroup parameter should be pointer or else the copy won't get notified about .Done() calls
func waitWithTimeout(ctx context.Context, group *sync.WaitGroup, timeout time.Duration) {
	c := make(chan struct{})
	go func() {
		defer close(c)
		group.Wait()
	}()
	select {
	case <-c:
		log.C(ctx).Debugf("Timeout WaitGroup %+v finished successfully", group)
	case <-time.After(timeout):
		log.C(ctx).Fatal("Shutdown took more than ", timeout)
		close(c)
	}
}
