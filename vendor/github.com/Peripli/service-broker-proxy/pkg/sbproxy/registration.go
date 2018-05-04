package sbproxy

import (
	"sync"

	"strings"

	"github.com/Peripli/service-broker-proxy/pkg/platform"
	"github.com/Peripli/service-broker-proxy/pkg/sm"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const ProxyBrokerPrefix = "sm-proxy-"

//TODO if the reg credentials are changed (the ones under cf.reg) we need to update the already registered brokers
type SBProxyRegistration struct {
	group          *sync.WaitGroup
	platformClient platform.Client
	smClient       sm.Client
	proxyPath      string
}

type serviceBrokerReg struct {
	platform.ServiceBroker
	SmID string
}

func NewTask(group *sync.WaitGroup, platformClient platform.Client, smClient sm.Client, proxyPath string) *SBProxyRegistration {
	return &SBProxyRegistration{
		group:          group,
		platformClient: platformClient,
		smClient:       smClient,
		proxyPath:      proxyPath,
	}
}

func (r SBProxyRegistration) Run() {
	logrus.Debug("STARTING scheduled registration task...")

	r.group.Add(1)
	defer r.group.Done()
	r.run()

	logrus.Debug("FINISHED scheduled registration task...")
}

func (r SBProxyRegistration) run() {

	// get all the registered proxy brokers from the platform
	brokersFromPlatform, err := r.getBrokersFromPlatform()
	if err != nil {
		logrus.WithError(err).Error("An error occurred while obtaining already registered brokers")
		return

	}

	// get all the brokers that are in SM and for which a proxy broker should be present in the platform
	brokersFromSM, err := r.getBrokersFromSM()
	if err != nil {
		logrus.WithError(err).Error("An error occurred while obtaining brokers from Service Manager")
		return
	}

	// register brokers that are present in SM and missing from platform
	updateBrokerRegistrations(r.createBrokerRegistration, brokersFromSM, brokersFromPlatform)

	// unregister proxy brokers that are no longer in SM but are still in platform
	unregisteredBrokers := updateBrokerRegistrations(r.deleteBrokerRegistration, brokersFromPlatform, brokersFromSM)

	// trigger a fetch of catalogs of proxy brokers that are registered at platform
	updateBrokerRegistrations(r.fetchBrokerCatalogs, brokersFromPlatform, unregisteredBrokers)

}

func (r SBProxyRegistration) getBrokersFromPlatform() ([]serviceBrokerReg, error) {
	logrus.Debug("Registration task getting proxy brokers from platform...")

	registeredBrokers, err := r.platformClient.GetBrokers()
	if err != nil {
		return nil, errors.Wrap(err, "error getting brokers from platform")
	}

	brokersFromPlatform := make([]serviceBrokerReg, 0, len(registeredBrokers))
	for _, broker := range registeredBrokers {
		if !r.isBrokerProxy(broker) {
			continue
		}

		logrus.WithFields(logFields(&broker)).Debug("Registration task FOUND registered proxy broker... ")
		brokerReg := serviceBrokerReg{
			ServiceBroker: broker,
			SmID:          broker.BrokerURL[strings.LastIndex(broker.BrokerURL, "/")+1:],
		}
		brokersFromPlatform = append(brokersFromPlatform, brokerReg)
	}
	logrus.Debugf("Registration task SUCCESSFULLY retrieved %d proxy brokers from platform", len(brokersFromPlatform))
	return brokersFromPlatform, nil
}

func (r SBProxyRegistration) getBrokersFromSM() ([]serviceBrokerReg, error) {
	logrus.Debug("Registration task getting brokers from Service Manager")

	proxyBrokers, err := r.smClient.GetBrokers()
	if err != nil {
		return nil, errors.Wrap(err, "error getting brokers from SM")
	}

	brokersFromSM := make([]serviceBrokerReg, 0, len(proxyBrokers))
	for _, broker := range proxyBrokers {
		brokerReg := serviceBrokerReg{
			ServiceBroker: broker,
			SmID:          broker.Guid,
		}
		brokersFromSM = append(brokersFromSM, brokerReg)
	}
	logrus.Debugf("Registration task SUCCESSFULLY retrieved %d brokers from Service Manager", len(brokersFromSM))

	return brokersFromSM, nil
}

func (r SBProxyRegistration) fetchBrokerCatalogs(broker platform.ServiceBroker) {
	logrus.WithFields(logFields(&broker)).Debugf("Registration task refetching catalog for broker")

	if f, isFetcher := r.platformClient.(platform.CatalogFetcher); isFetcher {
		if err := f.Fetch(&broker); err != nil {
			logrus.WithFields(logFields(&broker)).WithError(err).Error("Error during fetching catalog...")
		}
	}

	logrus.WithFields(logFields(&broker)).Debug("Registration task SUCCESSFULLY refetched catalog for broker")
}

func (r SBProxyRegistration) createBrokerRegistration(broker platform.ServiceBroker) {
	logrus.WithFields(logFields(&broker)).Info("Registration task attempting to create proxy for broker in platform...")

	createRequest := &platform.CreateServiceBrokerRequest{
		Name:      ProxyBrokerPrefix + broker.Guid,
		BrokerURL: r.proxyPath + "/" + broker.Guid,
	}

	if _, err := r.platformClient.CreateBroker(createRequest); err != nil {
		logrus.WithFields(logFields(&broker)).WithError(err).Error("Error during broker creation")
		//TODO how do we recover from that? Maybe atleast send email / slack notification?
	} else {
		logrus.WithFields(logFields(&broker)).Infof("Registration task SUCCESSFULLY created proxy for broker at platform under name [%s] accessible at [%s]", createRequest.Name, createRequest.BrokerURL)
	}
}

func (r SBProxyRegistration) deleteBrokerRegistration(broker platform.ServiceBroker) {
	logrus.WithFields(logFields(&broker)).Info("Registration task attempting to delete broker from platform...")

	deleteRequest := &platform.DeleteServiceBrokerRequest{
		Guid: broker.Guid,
		Name: broker.Name,
	}

	if err := r.platformClient.DeleteBroker(deleteRequest); err != nil {
		logrus.WithFields(logFields(&broker)).WithError(err).Error("Error during broker deletion")
		//TODO how do we recover from that? Maybe atleast send email / slack notification?
	} else {
		logrus.WithFields(logFields(&broker)).Infof("Registration task SUCCESSFULLY deleted proxy broker from platform with name [%s]", deleteRequest.Name)

	}
}

func (r SBProxyRegistration) isBrokerProxy(broker platform.ServiceBroker) bool {
	return strings.HasPrefix(broker.BrokerURL, r.proxyPath)
}

func updateBrokerRegistrations(updateOp func(broker platform.ServiceBroker), a, b []serviceBrokerReg) []serviceBrokerReg {
	affectedBrokers := make([]serviceBrokerReg, 0, 0)

	mb := make(map[string]serviceBrokerReg)
	for _, broker := range b {
		mb[broker.SmID] = broker
	}
	for _, broker := range a {
		if _, ok := mb[broker.SmID]; !ok {
			//TODO at some point we will be hitting platform rate limits... how should we handle that?
			updateOp(broker.ServiceBroker)
			affectedBrokers = append(affectedBrokers, broker)
		}
	}
	return affectedBrokers
}

func logFields(broker *platform.ServiceBroker) logrus.Fields {
	return logrus.Fields{
		"broker_guid": broker.Guid,
		"broker_name": broker.Name,
		"broker_url":  broker.BrokerURL,
	}
}
