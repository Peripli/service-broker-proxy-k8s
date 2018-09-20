package sbproxy

import (
	"context"
	"github.com/Peripli/service-manager/pkg/log"
	"sync"

	"strings"

	"encoding/json"

	"github.com/Peripli/service-broker-proxy/pkg/platform"
	"github.com/Peripli/service-broker-proxy/pkg/sm"
	"github.com/pkg/errors"
	osbc "github.com/pmorie/go-open-service-broker-client/v2"
	"github.com/sirupsen/logrus"
)

// ProxyBrokerPrefix prefixes names of brokers registered at the platform
const ProxyBrokerPrefix = "sm-proxy-"

// ReconcileBrokersTask type represents a registration task that takes care of propagating broker creations
// and deletions to the platform. It reconciles the state of the proxy brokers in the platform to match
// the desired state provided by the Service Manager.
// TODO if the reg credentials are changed (the ones under cf.reg) we need to update the already registered brokers
type ReconcileBrokersTask struct {
	group          *sync.WaitGroup
	platformClient platform.Client
	smClient       sm.Client
	proxyPath      string
	ctx            context.Context
}

type serviceBrokerReg struct {
	platform.ServiceBroker
	SmID string
}

// NewTask builds a new ReconcileBrokersTask
func NewTask(ctx context.Context, group *sync.WaitGroup, platformClient platform.Client, smClient sm.Client, proxyPath string) *ReconcileBrokersTask {
	return &ReconcileBrokersTask{
		group:          group,
		platformClient: platformClient,
		smClient:       smClient,
		proxyPath:      proxyPath,
		ctx:            ctx,
	}
}

// Run executes the registration task that is responsible for reconciling the state of the proxy brokers at the
// platform with the brokers provided by the Service Manager
func (r ReconcileBrokersTask) Run() {
	logger := log.C(r.ctx)
	logger.Debug("STARTING scheduled reconciliation task...")

	r.group.Add(1)
	defer r.group.Done()
	r.run()

	logger.Debug("FINISHED scheduled reconciliation task...")
}

func (r ReconcileBrokersTask) run() {
	// get all the registered proxy brokers from the platform
	brokersFromPlatform, err := r.getBrokersFromPlatform()
	if err != nil {
		log.C(r.ctx).WithError(err).Error("An error occurred while obtaining already registered brokers")
		return
	}

	// get all the brokers that are in SM and for which a proxy broker should be present in the platform
	brokersFromSM, err := r.getBrokersFromSM()
	if err != nil {
		log.C(r.ctx).WithError(err).Error("An error occurred while obtaining brokers from Service Manager")
		return
	}

	// control logic - make sure current state matches desired state
	r.reconcileBrokers(brokersFromPlatform, brokersFromSM)
}

// reconcileBrokers attempts to reconcile the current brokers state in the platform (existingBrokers)
// to match the desired broker state coming from the Service Manager (payloadBrokers).
func (r ReconcileBrokersTask) reconcileBrokers(existingBrokers []serviceBrokerReg, payloadBrokers []serviceBrokerReg) {
	existingMap := convertBrokersRegListToMap(existingBrokers)
	for _, payloadBroker := range payloadBrokers {
		existingBroker := existingMap[payloadBroker.SmID]
		delete(existingMap, payloadBroker.SmID)
		if existingBroker == nil {
			r.createBrokerRegistration(&payloadBroker.ServiceBroker)
		} else {
			r.fetchBrokerCatalog(&existingBroker.ServiceBroker)
		}
		r.enableServiceAccessVisibilities(&payloadBroker.ServiceBroker)
	}

	for _, existingBroker := range existingMap {
		r.deleteBrokerRegistration(&existingBroker.ServiceBroker)
	}
}

func (r ReconcileBrokersTask) getBrokersFromPlatform() ([]serviceBrokerReg, error) {
	logger := log.C(r.ctx)
	logger.Debug("ReconcileBrokersTask task getting proxy brokers from platform...")
	registeredBrokers, err := r.platformClient.GetBrokers(r.ctx)
	if err != nil {
		return nil, errors.Wrap(err, "error getting brokers from platform")
	}

	brokersFromPlatform := make([]serviceBrokerReg, 0, len(registeredBrokers))
	for _, broker := range registeredBrokers {
		if !r.isProxyBroker(broker) {
			continue
		}

		logger.WithFields(logBroker(&broker)).Debug("ReconcileBrokersTask task FOUND registered proxy broker... ")
		brokerReg := serviceBrokerReg{
			ServiceBroker: broker,
			SmID:          broker.BrokerURL[strings.LastIndex(broker.BrokerURL, "/")+1:],
		}
		brokersFromPlatform = append(brokersFromPlatform, brokerReg)
	}
	logger.Debugf("ReconcileBrokersTask task SUCCESSFULLY retrieved %d proxy brokers from platform", len(brokersFromPlatform))
	return brokersFromPlatform, nil
}

func (r ReconcileBrokersTask) getBrokersFromSM() ([]serviceBrokerReg, error) {
	logger := log.C(r.ctx)
	logger.Debug("ReconcileBrokersTask task getting brokers from Service Manager")

	proxyBrokers, err := r.smClient.GetBrokers(r.ctx)
	if err != nil {
		return nil, errors.Wrap(err, "error getting brokers from SM")
	}

	brokersFromSM := make([]serviceBrokerReg, 0, len(proxyBrokers))
	for _, broker := range proxyBrokers {
		brokerReg := serviceBrokerReg{
			ServiceBroker: broker,
			SmID:          broker.GUID,
		}
		brokersFromSM = append(brokersFromSM, brokerReg)
	}
	logger.Debugf("ReconcileBrokersTask task SUCCESSFULLY retrieved %d brokers from Service Manager", len(brokersFromSM))

	return brokersFromSM, nil
}

func (r ReconcileBrokersTask) fetchBrokerCatalog(broker *platform.ServiceBroker) {
	if f, isFetcher := r.platformClient.(platform.CatalogFetcher); isFetcher {
		logger := log.C(r.ctx)
		logger.WithFields(logBroker(broker)).Debugf("ReconcileBrokersTask task refetching catalog for broker")
		if err := f.Fetch(r.ctx, broker); err != nil {
			logger.WithFields(logBroker(broker)).WithError(err).Error("Error during fetching catalog...")
		} else {
			logger.WithFields(logBroker(broker)).Debug("ReconcileBrokersTask task SUCCESSFULLY refetched catalog for broker")
		}
	}
}

func (r ReconcileBrokersTask) createBrokerRegistration(broker *platform.ServiceBroker) {
	logger := log.C(r.ctx)
	logger.WithFields(logBroker(broker)).Info("ReconcileBrokersTask task attempting to create proxy for broker in platform...")

	createRequest := &platform.CreateServiceBrokerRequest{
		Name:      ProxyBrokerPrefix + broker.GUID,
		BrokerURL: r.proxyPath + "/" + broker.GUID,
	}

	if _, err := r.platformClient.CreateBroker(r.ctx, createRequest); err != nil {
		logger.WithFields(logBroker(broker)).WithError(err).Error("Error during broker creation")
	} else {
		logger.WithFields(logBroker(broker)).Infof("ReconcileBrokersTask task SUCCESSFULLY created proxy for broker at platform under name [%s] accessible at [%s]", createRequest.Name, createRequest.BrokerURL)
	}
}

func (r ReconcileBrokersTask) deleteBrokerRegistration(broker *platform.ServiceBroker) {
	logger := log.C(r.ctx)
	logger.WithFields(logBroker(broker)).Info("ReconcileBrokersTask task attempting to delete broker from platform...")

	deleteRequest := &platform.DeleteServiceBrokerRequest{
		GUID: broker.GUID,
		Name: broker.Name,
	}

	if err := r.platformClient.DeleteBroker(r.ctx, deleteRequest); err != nil {
		logger.WithFields(logBroker(broker)).WithError(err).Error("Error during broker deletion")
	} else {
		logger.WithFields(logBroker(broker)).Infof("ReconcileBrokersTask task SUCCESSFULLY deleted proxy broker from platform with name [%s]", deleteRequest.Name)
	}
}

func (r ReconcileBrokersTask) enableServiceAccessVisibilities(broker *platform.ServiceBroker) {
	if f, isEnabler := r.platformClient.(platform.ServiceAccess); isEnabler {
		emptyContext := emptyContext()
		logger := log.C(r.ctx)
		logger.WithFields(logBroker(broker)).Info("ReconcileBrokersTask task attempting to enable service access for broker...")

		catalog := broker.Catalog
		if catalog == nil {
			logger.WithFields(logBroker(broker)).Error("Error enabling service access due to missing catalog details")
			return
		}

		for _, service := range catalog.Services {
			logger.WithFields(logService(service)).Debug("ReconcileBrokersTask task attempting to enable service access for service...")
			if err := f.EnableAccessForService(r.ctx, emptyContext, service.ID); err != nil {
				logger.WithFields(logService(service)).WithError(err).Errorf("Error enabling service access for service with ID=%s...", service.ID)
			}
			logger.WithFields(logService(service)).Debug("ReconcileBrokersTask task finished enabling service access for service...")
		}
		logger.WithFields(logBroker(broker)).Infof("ReconcileBrokersTask task finished enabling service access for broker")
	}
}

func (r ReconcileBrokersTask) isProxyBroker(broker platform.ServiceBroker) bool {
	return strings.HasPrefix(broker.BrokerURL, r.proxyPath)
}

func logBroker(broker *platform.ServiceBroker) logrus.Fields {
	return logrus.Fields{
		"broker_guid": broker.GUID,
		"broker_name": broker.Name,
		"broker_url":  broker.BrokerURL,
	}
}

func logService(service osbc.Service) logrus.Fields {
	return logrus.Fields{
		"service_guid": service.ID,
		"service_name": service.Name,
	}
}

func emptyContext() json.RawMessage {
	return json.RawMessage(`{}`)
}

func convertBrokersRegListToMap(brokerList []serviceBrokerReg) map[string]*serviceBrokerReg {
	brokerRegMap := make(map[string]*serviceBrokerReg, len(brokerList))

	for i := range brokerList {
		brokerRegMap[brokerList[i].SmID] = &brokerList[i]
	}

	return brokerRegMap
}
