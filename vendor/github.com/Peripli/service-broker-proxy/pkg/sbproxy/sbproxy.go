package sbproxy

import (
	"sync"

	"github.com/Peripli/service-broker-proxy/pkg/config"
	"github.com/Peripli/service-broker-proxy/pkg/platform"
	"github.com/Peripli/service-broker-proxy/pkg/server"
	"github.com/Peripli/service-broker-proxy/pkg/sm"
	"github.com/Peripli/service-manager/pkg/env"
	"github.com/pkg/errors"
	"github.com/robfig/cron"
	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"fmt"
)

const (
	// APIPrefix for the Proxy OSB API
	APIPrefix = "/v1/osb"
)

// SBProxy service broker proxy struct
type SBProxy struct {
	Server *server.Server

	cronScheduler *cron.Cron
	group         *sync.WaitGroup
}

// DefaultEnv creates a default environment that can be used to boot up a Service Manager
func DefaultEnv(additionalPFlags ...func(set *pflag.FlagSet)) env.Environment {
	set := env.EmptyFlagSet()

	env.CreatePFlagsForConfigFile(set)
	config.AddPFlags(set)

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
func New(env env.Environment, client platform.Client) (*SBProxy, error) {
	cronScheduler := cron.New()
	var group sync.WaitGroup

	cfg, err := config.New(env)
	if err != nil {
		return nil, err
	}

	proxyServer, err := server.New(cfg.Server, cfg.Osb)
	if err != nil {
		return nil, err
	}

	sbProxy := &SBProxy{
		Server:        proxyServer,
		cronScheduler: cronScheduler,
		group:         &group,
	}

	regJob, err := defaultRegJob(&group, client, cfg.Sm, cfg.Server.Host)
	if err != nil {
		return nil, err
	}

	resyncSchedule := "@every " + cfg.Server.ResyncPeriod.String()
	logrus.Info("Brokers and Access resync schedule: ", resyncSchedule)

	if err := cronScheduler.AddJob(resyncSchedule, regJob); err != nil {
		return nil, errors.Wrap(err, "error adding registration job")
	}

	return sbProxy, nil
}

// Run starts the proxy
func (p *SBProxy) Run() {
	p.cronScheduler.Start()
	defer p.cronScheduler.Stop()

	p.Server.Run(p.group)
}

func defaultRegJob(group *sync.WaitGroup, platformClient platform.Client, smConfig *sm.Config, proxyHost string) (cron.Job, error) {
	smClient, err := smConfig.CreateFunc(smConfig)
	if err != nil {
		return nil, err
	}
	regTask := NewTask(group, platformClient, smClient, proxyHost+APIPrefix)

	return regTask, nil
}
