package sbproxy

import (
	"github.com/Peripli/service-broker-proxy/pkg/logging"
	"github.com/Peripli/service-manager/pkg/log"
	"sync"

	"fmt"

	"context"
	"time"

	"github.com/Peripli/service-broker-proxy/pkg/osb"
	"github.com/Peripli/service-broker-proxy/pkg/platform"
	"github.com/Peripli/service-broker-proxy/pkg/sm"
	"github.com/Peripli/service-manager/api/filters"
	smosb "github.com/Peripli/service-manager/api/osb"
	"github.com/Peripli/service-manager/pkg/env"
	"github.com/Peripli/service-manager/pkg/server"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/robfig/cron"
	"github.com/spf13/pflag"
	"github.com/Peripli/service-broker-proxy/pkg/sbproxy/reconcile"
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
	*cron.Cron

	ctx   context.Context
	cfg   *Settings
	group *sync.WaitGroup
}

// SMProxy  struct
type SMProxy struct {
	*server.Server

	scheduler *cron.Cron
	ctx       context.Context
	group     *sync.WaitGroup
}

// DefaultEnv creates a default environment that can be used to boot up a Service Broker proxy
func DefaultEnv(additionalPFlags ...func(set *pflag.FlagSet)) env.Environment {
	set := pflag.NewFlagSet("Configuration Flags", pflag.ExitOnError)

	AddPFlags(set)
	for _, addFlags := range additionalPFlags {
		addFlags(set)
	}
	environment, err := env.New(set)
	if err != nil {
		panic(fmt.Errorf("error loading environment: %s", err))
	}
	return environment
}

// New creates service broker proxy that is configured from the provided environment and platform client.
func New(ctx context.Context, env env.Environment, platformClient platform.Client) *SMProxyBuilder {
	cronScheduler := cron.New()
	var group sync.WaitGroup

	cfg, err := NewSettings(env)
	if err != nil {
		panic(err)
	}

	if err := cfg.Validate(); err != nil {
		panic(err)
	}

	ctx = log.Configure(ctx, cfg.Log)
	log.AddHook(&logging.ErrorLocationHook{})

	api := &web.API{
		Controllers: []web.Controller{
			smosb.NewController(&osb.BrokerDetailsFetcher{
				URL:      cfg.Sm.URL + cfg.Sm.OSBAPIPath,
				Username: cfg.Sm.User,
				Password: cfg.Sm.Password,
			}, &sm.SkipSSLTransport{
				SkipSslValidation: cfg.Sm.SkipSSLValidation,
			}),
		},
		Filters: []web.Filter{
			&filters.Logging{},
		},
	}

	sbProxy := &SMProxyBuilder{
		API:   api,
		Cron:  cronScheduler,
		ctx:   ctx,
		cfg:   cfg,
		group: &group,
	}

	smClient, _ := sm.NewClient(cfg.Sm)
	if err != nil {
		panic(err)
	}

	regJob := reconcile.NewTask(ctx, &group, platformClient, smClient, cfg.Reconcile.URL+APIPrefix)

	resyncSchedule := "@every " + cfg.Sm.ResyncPeriod.String()
	log.C(ctx).Info("Brokers and Access resync schedule: ", resyncSchedule)

	if err := cronScheduler.AddJob(resyncSchedule, regJob); err != nil {
		panic(err)
	}

	return sbProxy
}

// Build builds the Service Manager
func (smb *SMProxyBuilder) Build() *SMProxy {
	srv := server.New(smb.cfg.Server, smb.API)
	srv.Use(filters.NewRecoveryMiddleware())

	return &SMProxy{
		Server:    srv,
		scheduler: smb.Cron,
		ctx:       smb.ctx,
		group:     smb.group,
	}
}

// Run starts the proxy
func (p *SMProxy) Run() {
	p.scheduler.Start()
	defer p.scheduler.Stop()
	defer waitWithTimeout(p.ctx, p.group, p.Server.Config.ShutdownTimeout)

	log.C(p.ctx).Info("Running SBProxy...")

	p.Server.Run(p.ctx)
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
		log.C(ctx).Debug(fmt.Sprintf("Timeout WaitGroup %+v finished successfully", group))
	case <-time.After(timeout):
		log.C(ctx).Fatal("Shutdown took more than ", timeout)
		close(c)
	}
}
