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
	"github.com/Peripli/service-manager/pkg/log"

	"strings"

	"github.com/Peripli/service-broker-proxy/pkg/platform"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// ProxyBrokerPrefix prefixes names of brokers registered at the platform
const ProxyBrokerPrefix = "sm-proxy-"

// processBrokers handles the reconsilation of the service brokers.
// it gets the brokers from SM and the platform and runs the reconciliation
func (r *ReconciliationTask) processBrokers() {
	logger := log.C(r.runContext)
	if r.platformClient.Broker() == nil {
		logger.Debug("Platform client cannot handle brokers. Broker reconciliation will be skipped.")
		return
	}

	// get all the registered brokers from the platform
	brokersFromPlatform, err := r.getBrokersFromPlatform()
	if err != nil {
		logger.WithError(err).Error("An error occurred while obtaining already registered brokers")
		return
	}

	brokersFromSM, err := r.getBrokersFromSM()
	if err != nil {
		logger.WithError(err).Error("An error occurred while obtaining brokers from Service Manager")
		return
	}
	r.stats[smBrokersStats] = brokersFromSM

	// control logic - make sure current state matches desired state
	r.reconcileBrokers(brokersFromPlatform, brokersFromSM)
}

// reconcileBrokers attempts to reconcile the current brokers state in the platform (existingBrokers)
// to match the desired broker state coming from the Service Manager (payloadBrokers).
func (r *ReconciliationTask) reconcileBrokers(existingBrokers []platform.ServiceBroker, payloadBrokers []platform.ServiceBroker) {
	groupedExistingBrokers := groupBrokersByConditions(existingBrokers, mapByWholeURL, mapByURLSmIDSuffix(r.isProxyBroker))
	existingBrokersByURLMap := groupedExistingBrokers[0]
	existingBrokersByURLSmIDSuffixMap := groupedExistingBrokers[1]

	for _, payloadBroker := range payloadBrokers {
		existingBroker := existingBrokersByURLSmIDSuffixMap[payloadBroker.GUID]
		delete(existingBrokersByURLSmIDSuffixMap, payloadBroker.GUID)

		platformBroker, knownToPlatform := existingBrokersByURLMap[payloadBroker.BrokerURL]
		// Broker is registered in platform, but not already known to this broker proxy
		if knownToPlatform {
			r.updateBrokerRegistration(platformBroker.GUID, &payloadBroker)
		} else if existingBroker == nil {
			r.createBrokerRegistration(&payloadBroker)
		} else {
			r.fetchBrokerCatalog(existingBroker)
		}
	}

	for _, existingBroker := range existingBrokersByURLSmIDSuffixMap {
		r.deleteBrokerRegistration(existingBroker)
	}
}

func (r *ReconciliationTask) getBrokersFromPlatform() ([]platform.ServiceBroker, error) {
	logger := log.C(r.runContext)
	logger.Debug("ReconciliationTask task getting brokers from platform...")
	registeredBrokers, err := r.platformClient.Broker().GetBrokers(r.runContext)
	if err != nil {
		return nil, errors.Wrap(err, "error getting brokers from platform")
	}

	brokersFromPlatform := make([]platform.ServiceBroker, 0, len(registeredBrokers))
	for _, broker := range registeredBrokers {
		logger.WithFields(logBroker(&broker)).Debug("ReconciliationTask task FOUND registered broker... ")
		brokersFromPlatform = append(brokersFromPlatform, broker)
	}
	logger.Debugf("ReconciliationTask task SUCCESSFULLY retrieved %d brokers from platform", len(brokersFromPlatform))
	return brokersFromPlatform, nil
}

func (r *ReconciliationTask) getBrokersFromSM() ([]platform.ServiceBroker, error) {
	logger := log.C(r.runContext)
	logger.Debug("ReconciliationTask task getting brokers from Service Manager")

	proxyBrokers, err := r.smClient.GetBrokers(r.runContext)
	if err != nil {
		return nil, errors.Wrap(err, "error getting brokers from SM")
	}

	brokersFromSM := make([]platform.ServiceBroker, 0, len(proxyBrokers))
	for _, broker := range proxyBrokers {
		brokerReg := platform.ServiceBroker{
			GUID:             broker.ID,
			BrokerURL:        broker.BrokerURL,
			ServiceOfferings: broker.ServiceOfferings,
			Metadata:         broker.Metadata,
		}
		brokersFromSM = append(brokersFromSM, brokerReg)
	}
	logger.Debugf("ReconciliationTask task SUCCESSFULLY retrieved %d brokers from Service Manager", len(brokersFromSM))

	return brokersFromSM, nil
}

func (r *ReconciliationTask) fetchBrokerCatalog(broker *platform.ServiceBroker) {
	if f, isFetcher := r.platformClient.(platform.CatalogFetcher); isFetcher {
		logger := log.C(r.runContext)
		logger.WithFields(logBroker(broker)).Debugf("ReconciliationTask task refetching catalog for broker")
		if err := f.Fetch(r.runContext, broker); err != nil {
			logger.WithFields(logBroker(broker)).WithError(err).Error("Error during fetching catalog...")
		} else {
			logger.WithFields(logBroker(broker)).Debug("ReconciliationTask task SUCCESSFULLY refetched catalog for broker")
		}
	}
}

func (r *ReconciliationTask) createBrokerRegistration(broker *platform.ServiceBroker) {
	logger := log.C(r.runContext)
	logger.WithFields(logBroker(broker)).Info("ReconciliationTask task attempting to create proxy for broker in platform...")

	createRequest := &platform.CreateServiceBrokerRequest{
		Name:      ProxyBrokerPrefix + broker.GUID,
		BrokerURL: r.proxyPath + "/" + broker.GUID,
	}

	if b, err := r.platformClient.Broker().CreateBroker(r.runContext, createRequest); err != nil {
		logger.WithFields(logBroker(broker)).WithError(err).Error("Error during broker creation")
	} else {
		logger.WithFields(logBroker(b)).Infof("ReconciliationTask task SUCCESSFULLY created proxy for broker at platform under name [%s] accessible at [%s]", createRequest.Name, createRequest.BrokerURL)
	}
}

func (r *ReconciliationTask) updateBrokerRegistration(brokerGUID string, broker *platform.ServiceBroker) {
	logger := log.C(r.runContext)
	logger.WithFields(logBroker(broker)).Info("ReconciliationTask task attempting to update broker registration in platform...")

	updateRequest := &platform.UpdateServiceBrokerRequest{
		GUID:      brokerGUID,
		Name:      ProxyBrokerPrefix + broker.GUID,
		BrokerURL: r.proxyPath + "/" + broker.GUID,
	}

	if b, err := r.platformClient.Broker().UpdateBroker(r.runContext, updateRequest); err != nil {
		logger.WithFields(logBroker(broker)).WithError(err).Error("Error during broker update")
	} else {
		logger.WithFields(logBroker(b)).Infof("ReconciliationTask task SUCCESSFULLY updated broker registration at platform under name [%s] accessible at [%s]", updateRequest.Name, updateRequest.BrokerURL)
	}
}

func (r *ReconciliationTask) deleteBrokerRegistration(broker *platform.ServiceBroker) {
	logger := log.C(r.runContext)
	logger.WithFields(logBroker(broker)).Info("ReconciliationTask task attempting to delete broker from platform...")

	deleteRequest := &platform.DeleteServiceBrokerRequest{
		GUID: broker.GUID,
		Name: broker.Name,
	}

	if err := r.platformClient.Broker().DeleteBroker(r.runContext, deleteRequest); err != nil {
		logger.WithFields(logBroker(broker)).WithError(err).Error("Error during broker deletion")
	} else {
		logger.WithFields(logBroker(broker)).Infof("ReconciliationTask task SUCCESSFULLY deleted proxy broker from platform with name [%s]", deleteRequest.Name)
	}
}

func (r *ReconciliationTask) isProxyBroker(broker *platform.ServiceBroker) bool {
	return strings.HasPrefix(broker.BrokerURL, r.proxyPath)
}

func logBroker(broker *platform.ServiceBroker) logrus.Fields {
	return logrus.Fields{
		"broker_guid": broker.GUID,
		"broker_name": broker.Name,
		"broker_url":  broker.BrokerURL,
	}
}

type filterBrokerPredicate func(broker *platform.ServiceBroker) bool

func mapByURLSmIDSuffix(predicate filterBrokerPredicate) func(platform.ServiceBroker) (string, bool) {
	return func(broker platform.ServiceBroker) (string, bool) {
		if !predicate(&broker) {
			return "", false
		}
		return broker.BrokerURL[strings.LastIndex(broker.BrokerURL, "/")+1:], true
	}
}

func mapByWholeURL(broker platform.ServiceBroker) (string, bool) {
	return broker.BrokerURL, true
}

func groupBrokersByConditions(brokerList []platform.ServiceBroker, groupBy ...func(platform.ServiceBroker) (string, bool)) []map[string]*platform.ServiceBroker {
	brokerMaps := make([]map[string]*platform.ServiceBroker, len(groupBy))
	for i := range groupBy {
		brokerMaps[i] = make(map[string]*platform.ServiceBroker)
	}

	for i, broker := range brokerList {
		for groupIndex, groupFunc := range groupBy {
			group, mapped := groupFunc(broker)
			if mapped {
				brokerMaps[groupIndex][group] = &brokerList[i]
			}
		}
	}

	return brokerMaps
}
