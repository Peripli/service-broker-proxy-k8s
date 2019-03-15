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
	"context"
	"sync"
	"sync/atomic"

	"github.com/Peripli/service-broker-proxy/pkg/platform"
	"github.com/Peripli/service-broker-proxy/pkg/sm"
	"github.com/Peripli/service-manager/pkg/log"
	"github.com/gofrs/uuid"
	cache "github.com/patrickmn/go-cache"
	"github.com/pkg/errors"
)

const running int32 = 1
const notRunning int32 = 0

const smBrokersStats = "sm_brokers"

// ReconciliationTask type represents a registration task that takes care of propagating broker and visibility
// creations and deletions to the platform. It reconciles the state of the proxy brokers and visibilities
// in the platform to match the desired state provided by the Service Manager.
// TODO if the reg credentials are changed (the ones under cf.reg) we need to update the already registered brokers
type ReconciliationTask struct {
	options        *Settings
	group          *sync.WaitGroup
	platformClient platform.Client
	smClient       sm.Client
	proxyPath      string
	cache          *cache.Cache
	state          *int32

	globalContext context.Context
	runContext    context.Context

	stats map[string]interface{}
}

// NewTask builds a new ReconciliationTask
func NewTask(ctx context.Context,
	options *Settings,
	group *sync.WaitGroup,
	platformClient platform.Client,
	smClient sm.Client,
	proxyPath string,
	c *cache.Cache) *ReconciliationTask {
	return &ReconciliationTask{
		options:        options,
		group:          group,
		platformClient: platformClient,
		smClient:       smClient,
		proxyPath:      proxyPath,
		globalContext:  ctx,
		cache:          c,
		runContext:     nil,
		state:          new(int32),
	}
}

// Run executes the registration task that is responsible for reconciling the state of the proxy
// brokers and visibilities at the platform with the brokers provided by the Service Manager
func (r *ReconciliationTask) Run() {
	isAlreadyRunnning := !atomic.CompareAndSwapInt32(r.state, notRunning, running)
	if isAlreadyRunnning {
		log.C(r.globalContext).Warning("Reconciliation task cannot start. Another reconciliation task is already running")
		return
	}
	defer r.end()

	r.stats = make(map[string]interface{})

	newRunContext, taskID, err := r.generateRunContext()
	if err != nil {
		log.C(r.globalContext).WithError(err).Error("reconsilation task will not be sheduled")
		return
	}
	r.runContext = newRunContext
	log.C(r.globalContext).Debugf("STARTING scheduled reconciliation task %s...", taskID)

	r.group.Add(1)
	defer r.group.Done()
	r.run()
	log.C(r.globalContext).Debugf("FINISHED scheduled reconciliation task %s...", taskID)
}

func (r *ReconciliationTask) run() {
	r.processBrokers()
	r.processVisibilities()
}

func (r *ReconciliationTask) end() {
	atomic.StoreInt32(r.state, notRunning)
}

func (r *ReconciliationTask) generateRunContext() (context.Context, string, error) {
	correlationID, err := uuid.NewV4()
	if err != nil {
		return nil, "", errors.Wrap(err, "could not generate correlationID")
	}
	entry := log.C(r.globalContext).WithField(log.FieldCorrelationID, correlationID.String())
	return log.ContextWithLogger(r.globalContext, entry), correlationID.String(), nil
}

func (r *ReconciliationTask) stat(key string) interface{} {
	result, found := r.stats[key]
	if !found {
		return nil
	}
	return result
}
