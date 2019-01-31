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
	"encoding/json"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/Peripli/service-broker-proxy/pkg/platform"
	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/pkg/errors"
)

const (
	platformVisibilityCacheKey = "platform_visibilities"
	smPlansCacheKey            = "sm_plans"
)

// processVisibilities handles the reconsilation of the service visibilities.
// It gets the service visibilities from SM and the platform and runs the reconciliation
func (r *ReconciliationTask) processVisibilities() {
	logger := log.C(r.runContext)
	if r.platformClient.Visibility() == nil {
		logger.Debug("Platform client cannot handle visibilities. Visibility reconciliation will be skipped.")
		return
	}

	plans, err := r.getSMPlans()
	if err != nil {
		logger.WithError(err).Error("An error occurred while obtaining plans from Service Manager")
		return
	}
	var platformVisibilities []*platform.ServiceVisibilityEntity
	if r.options.VisibilityCache && r.areSMPlansSame(plans) {
		platformVisibilities = r.getPlatformVisibilitiesFromCache()
	}

	visibilityCacheUsed := platformVisibilities != nil
	if !visibilityCacheUsed {
		platformVisibilities, err = r.loadPlatformVisibilities(plans)
		if err != nil {
			logger.WithError(err).Error("An error occurred while loading visibilities from platform")
			return
		}
	}

	plansMap := smPlansToMap(plans)
	smVisibilities, err := r.getSMVisibilities(plansMap)
	if err != nil {
		logger.WithError(err).Error("An error occurred while obtaining SM visibilities")
		return
	}

	errorOccured := r.reconcileServiceVisibilities(platformVisibilities, smVisibilities)

	if r.options.VisibilityCache {
		if errorOccured {
			r.cache.Delete(platformVisibilityCacheKey)
			r.cache.Delete(smPlansCacheKey)
		} else {
			r.updateVisibilityCache(visibilityCacheUsed, plansMap, smVisibilities)
		}
	}
}

// updateVisibilityCache
func (r *ReconciliationTask) updateVisibilityCache(visibilityCacheUsed bool, plansMap map[string]*types.ServicePlan, visibilities []*platform.ServiceVisibilityEntity) {
	r.cache.Set(smPlansCacheKey, plansMap, r.options.CacheExpiration)
	cacheExpiration := r.options.CacheExpiration
	if visibilityCacheUsed {
		_, expiration, found := r.cache.GetWithExpiration(platformVisibilityCacheKey)
		if found {
			cacheExpiration = time.Until(expiration)
		}
	}
	r.cache.Set(platformVisibilityCacheKey, visibilities, cacheExpiration)
}

// areSMPlansSame checks if there are new or deleted plans in SM.
// Returns true if there are no new or deleted plans, false otherwise
func (r *ReconciliationTask) areSMPlansSame(plans []*types.ServicePlan) bool {
	cachedPlans, isPresent := r.cache.Get(smPlansCacheKey)
	if !isPresent {
		return false
	}
	cachedPlansMap, ok := cachedPlans.(map[string]*types.ServicePlan)
	if !ok {
		log.C(r.runContext).Error("SM plans cache is in invalid state! Clearing...")
		r.cache.Delete(smPlansCacheKey)
		return false
	}
	if len(cachedPlansMap) != len(plans) {
		return false
	}
	for _, plan := range plans {
		if cachedPlansMap[plan.ID] == nil {
			return false
		}
	}
	return true
}

func (r *ReconciliationTask) getPlatformVisibilitiesFromCache() []*platform.ServiceVisibilityEntity {
	platformVisibilities, found := r.cache.Get(platformVisibilityCacheKey)
	if !found {
		return nil
	}
	if result, ok := platformVisibilities.([]*platform.ServiceVisibilityEntity); ok {
		log.C(r.runContext).Debugf("ReconciliationTask fetched %d platform visibilities from cache", len(result))
		return result
	}
	log.C(r.runContext).Error("Platform visibilities cache is in invalid state! Clearing...")
	r.cache.Delete(platformVisibilityCacheKey)
	return nil
}

func (r *ReconciliationTask) loadPlatformVisibilities(plans []*types.ServicePlan) ([]*platform.ServiceVisibilityEntity, error) {
	logger := log.C(r.runContext)
	logger.Debug("ReconciliationTask getting visibilities from platform")

	platformVisibilities, err := r.platformClient.Visibility().GetVisibilitiesByPlans(r.runContext, plans)
	if err != nil {
		return nil, err
	}
	logger.Debugf("ReconciliationTask successfully retrieved %d visibilities from platform", len(platformVisibilities))

	return platformVisibilities, nil
}

func (r *ReconciliationTask) getSMPlans() ([]*types.ServicePlan, error) {
	logger := log.C(r.runContext)
	logger.Debug("ReconciliationTask getting plans from Service Manager")

	plans, err := r.smClient.GetPlans(r.runContext)
	if err != nil {
		return nil, errors.Wrap(err, "error gettings plans from SM")
	}
	logger.Debugf("ReconciliationTask successfully retrieved %d plans from Service Manager", len(plans))

	return plans, nil
}

func (r *ReconciliationTask) getSMVisibilities(smPlansMap map[string]*types.ServicePlan) ([]*platform.ServiceVisibilityEntity, error) {
	logger := log.C(r.runContext)
	logger.Debug("ReconciliationTask getting visibilities from Service Manager")

	visibilities, err := r.smClient.GetVisibilities(r.runContext)
	if err != nil {
		return nil, err
	}
	logger.Debugf("ReconciliationTask successfully retrieved %d visibilities from Service Manager", len(visibilities))

	result := make([]*platform.ServiceVisibilityEntity, 0)
	for _, visibility := range visibilities {
		converted := r.convertSMVisibility(visibility, smPlansMap[visibility.ServicePlanID])
		result = append(result, converted...)
	}
	logger.Debugf("ReconciliationTask successfully converted %d SM visibilities to %d platform visibilities", len(visibilities), len(result))

	return result, nil
}

func (r *ReconciliationTask) convertSMVisibility(visibility *types.Visibility, smPlan *types.ServicePlan) []*platform.ServiceVisibilityEntity {
	scopeLabelKey := r.platformClient.Visibility().VisibilityScopeLabelKey()

	if visibility.PlatformID == "" || scopeLabelKey == "" {
		return []*platform.ServiceVisibilityEntity{
			{
				Public:        true,
				CatalogPlanID: smPlan.CatalogID,
				Labels:        map[string]string{},
			},
		}
	}

	scopes := visibility.Labels[scopeLabelKey]
	result := make([]*platform.ServiceVisibilityEntity, 0, len(scopes))
	for _, scope := range scopes {
		result = append(result, &platform.ServiceVisibilityEntity{
			Public:        false,
			CatalogPlanID: smPlan.CatalogID,
			Labels:        map[string]string{scopeLabelKey: scope},
		})
	}
	return result
}

func (r *ReconciliationTask) reconcileServiceVisibilities(platformVis, smVis []*platform.ServiceVisibilityEntity) bool {
	logger := log.C(r.runContext)
	logger.Debug("ReconciliationTask reconsiling platform and SM visibilities...")

	platformMap := r.convertVisListToMap(platformVis)
	visibilitiesToCreate := make([]*platform.ServiceVisibilityEntity, 0)
	for _, visibility := range smVis {
		key := r.getVisibilityKey(visibility)
		existingVis := platformMap[key]
		delete(platformMap, key)
		if existingVis == nil {
			visibilitiesToCreate = append(visibilitiesToCreate, visibility)
		}
	}

	logger.Debugf("ReconciliationTask %d visibilities will be removed from the platform", len(platformMap))
	if errorOccured := r.deleteVisibilities(platformMap); errorOccured != nil {
		logger.WithError(errorOccured).Error("ReconciliationTask - could not remove visibilities from platform")
		return true
	}

	logger.Debugf("ReconciliationTask %d visibilities will be created in the platform", len(visibilitiesToCreate))
	if errorOccured := r.createVisibilities(visibilitiesToCreate); errorOccured != nil {
		logger.WithError(errorOccured).Error("ReconciliationTask - could not create visibilities in platform")
		return true
	}

	return false
}

type visibilityProcessingState struct {
	Mutex          sync.Mutex
	Ctx            context.Context
	StopProcessing context.CancelFunc
	WaitGroup      sync.WaitGroup
	ErrorOccured   error
}

func (r *ReconciliationTask) newVisibilityProcessingState() *visibilityProcessingState {
	visibilitiesContext, cancel := context.WithCancel(r.runContext)
	return &visibilityProcessingState{
		Ctx:            visibilitiesContext,
		StopProcessing: cancel,
	}
}

// deleteVisibilities deletes visibilities from platform. Returns true if error has occurred
func (r *ReconciliationTask) deleteVisibilities(visibilities map[string]*platform.ServiceVisibilityEntity) error {
	state := r.newVisibilityProcessingState()
	defer state.StopProcessing()

	for _, visibility := range visibilities {
		execAsync(state, visibility, r.deleteVisibility)
	}
	return await(state)
}

// createVisibilities creates visibilities from platform. Returns true if error has occurred
func (r *ReconciliationTask) createVisibilities(visibilities []*platform.ServiceVisibilityEntity) error {
	state := r.newVisibilityProcessingState()
	defer state.StopProcessing()

	for _, visibility := range visibilities {
		execAsync(state, visibility, r.createVisibility)
	}
	return await(state)
}

func execAsync(state *visibilityProcessingState, visibility *platform.ServiceVisibilityEntity, f func(context.Context, *platform.ServiceVisibilityEntity) error) {
	state.WaitGroup.Add(1)

	go func() {
		defer state.WaitGroup.Done()
		err := f(state.Ctx, visibility)
		if err == nil {
			return
		}
		state.Mutex.Lock()
		defer state.Mutex.Unlock()
		if state.ErrorOccured == nil {
			state.ErrorOccured = err
		}
	}()
}

func await(state *visibilityProcessingState) error {
	state.WaitGroup.Wait()
	return state.ErrorOccured
}

// getVisibilityKey maps a generic visibility to a specific string. The string contains catalogID and scope for non-public plans
func (r *ReconciliationTask) getVisibilityKey(visibility *platform.ServiceVisibilityEntity) string {
	scopeLabelKey := r.platformClient.Visibility().VisibilityScopeLabelKey()

	const idSeparator = "|"
	if visibility.Public {
		return strings.Join([]string{"public", "", visibility.CatalogPlanID}, idSeparator)
	}
	return strings.Join([]string{"!public", visibility.Labels[scopeLabelKey], visibility.CatalogPlanID}, idSeparator)
}

func (r *ReconciliationTask) createVisibility(ctx context.Context, visibility *platform.ServiceVisibilityEntity) error {
	logger := log.C(r.runContext)
	logger.Debugf("Creating visibility for %s with labels %v", visibility.CatalogPlanID, visibility.Labels)

	json, err := marshalVisibilityLabels(logger, visibility.Labels)
	if err != nil {
		return err
	}
	if err = r.platformClient.Visibility().EnableAccessForPlan(ctx, json, visibility.CatalogPlanID); err != nil {
		logger.WithError(err).Errorf("Could not enable access for plan %s", visibility.CatalogPlanID)
		return err
	}
	return nil
}

func (r *ReconciliationTask) deleteVisibility(ctx context.Context, visibility *platform.ServiceVisibilityEntity) error {
	logger := log.C(r.runContext)
	logger.Debugf("Deleting visibility for %s with labels %v", visibility.CatalogPlanID, visibility.Labels)

	json, err := marshalVisibilityLabels(logger, visibility.Labels)
	if err != nil {
		return err
	}
	if err = r.platformClient.Visibility().DisableAccessForPlan(ctx, json, visibility.CatalogPlanID); err != nil {
		logger.WithError(err).Errorf("Could not disable access for plan %s", visibility.CatalogPlanID)
		return err
	}
	return nil
}

func marshalVisibilityLabels(logger *logrus.Entry, labels map[string]string) ([]byte, error) {
	json, err := json.Marshal(labels)
	if err != nil {
		logger.WithError(err).Error("Could not marshal labels to json")
	}
	return json, err
}

func (r *ReconciliationTask) convertVisListToMap(list []*platform.ServiceVisibilityEntity) map[string]*platform.ServiceVisibilityEntity {
	result := make(map[string]*platform.ServiceVisibilityEntity, len(list))
	for _, vis := range list {
		key := r.getVisibilityKey(vis)
		result[key] = vis
	}
	return result
}

func smPlansToMap(plans []*types.ServicePlan) map[string]*types.ServicePlan {
	plansMap := make(map[string]*types.ServicePlan, len(plans))
	for _, plan := range plans {
		plansMap[plan.ID] = plan
	}
	return plansMap
}
