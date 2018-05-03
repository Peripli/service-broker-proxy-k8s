package sbproxy

import (
	"sync"

	"strings"

	"github.com/Peripli/service-broker-proxy/pkg/platform"
	"github.com/Peripli/service-broker-proxy/pkg/sm"
	"github.com/sirupsen/logrus"
)

const ProxyBrokerPrefix = "sm-proxy-"

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
	r.group.Add(1)
	defer r.group.Done()
	r.run()
}

func (r SBProxyRegistration) run() {
	logrus.Debug("Running broker registration task...")
	registeredBrokers, err := r.platformClient.GetBrokers()
	if err != nil {
		logrus.WithError(err).Error("An error occurred while obtaining already registered brokers")
		return
	}

	brokersFromPlatform := make([]serviceBrokerReg, 0, len(registeredBrokers))
	for _, broker := range registeredBrokers {
		if !r.isBrokerProxy(broker) {
			logrus.WithFields(logFields(&broker)).Debug("Registration task SKIPPING registered broker as is not recognized to be proxy broker...")
			continue
		}

		if f, isFetcher := r.platformClient.(platform.Fetcher); isFetcher {
			if err := f.Fetch(&broker); err != nil {
				logrus.WithFields(logFields(&broker)).WithError(err).Error("Error during fetching catalog...")
			}
		}

		brokerReg := serviceBrokerReg{
			ServiceBroker: broker,
			SmID:          broker.BrokerURL[strings.LastIndex(broker.BrokerURL, "/")+1:],
		}
		brokersFromPlatform = append(brokersFromPlatform, brokerReg)
	}

	proxyBrokers, err := r.smClient.GetBrokers()
	if err != nil {
		logrus.WithError(err).Error("An error occurred while obtaining brokers that have to be registered at the platform")
		return
	}

	brokersFromSM := make([]serviceBrokerReg, 0, len(proxyBrokers))
	for _, broker := range proxyBrokers {
		brokerReg := serviceBrokerReg{
			ServiceBroker: broker,
			SmID:          broker.Guid,
		}
		brokersFromSM = append(brokersFromSM, brokerReg)
	}

	// register brokers that are present in SM and missing from platform
	updateBrokerRegistrations(r.createBrokerRegistration, brokersFromSM, brokersFromPlatform)

	// unregister brokers that are no longer in SM but are still in platform
	updateBrokerRegistrations(r.deleteBrokerRegistration, brokersFromPlatform, brokersFromSM)

}

func (r SBProxyRegistration) deleteBrokerRegistration(broker platform.ServiceBroker) {

	logrus.WithFields(logFields(&broker)).Info("Registration task will attempt to delete broker...")

	deleteRequest := &platform.DeleteServiceBrokerRequest{
		Guid: broker.Guid,
		Name: broker.Name,
	}
	if err := r.platformClient.DeleteBroker(deleteRequest); err != nil {
		logrus.WithFields(logFields(&broker)).WithError(err).Error("Error during broker deletion")
		//TODO how do we recover from that? Maybe atleast send email / slack notification?
	} else {
		logrus.WithFields(logFields(&broker)).Info("Registration task broker deletion successful")

	}
}

func (r SBProxyRegistration) updateBrokerRegistration(broker platform.ServiceBroker) {

	logrus.WithFields(logFields(&broker)).Info("Registration task will attempt to update broker...")

	updateRequest := &platform.UpdateServiceBrokerRequest{
		Guid:      broker.Guid,
		Name:      broker.Name,
		BrokerURL: broker.BrokerURL,
	}
	updatedBroker, err := r.platformClient.UpdateBroker(updateRequest)
	if err != nil {
		logrus.WithFields(logFields(&broker)).WithError(err).Error("Error during broker update")
		//TODO how do we recover from that? Maybe atleast send email / slack notification?
	} else {
		logrus.WithFields(logFields(updatedBroker)).Info("Registration task broker update successful")

	}
}

func (r SBProxyRegistration) createBrokerRegistration(broker platform.ServiceBroker) {
	logrus.WithFields(logFields(&broker)).Info("Registration task will attempt to create broker...")
	createRequest := &platform.CreateServiceBrokerRequest{
		Name:      ProxyBrokerPrefix + broker.Guid,
		BrokerURL: r.proxyPath + "/" + broker.Guid,
	}
	if _, err := r.platformClient.CreateBroker(createRequest); err != nil {
		logrus.WithFields(logFields(&broker)).WithError(err).Error("Error during broker creation")
		//TODO how do we recover from that? Maybe atleast send email / slack notification?
	} else {
		logrus.WithFields(logFields(&broker)).Info("Registration task broker creation successful")
	}
}

func (r SBProxyRegistration) isBrokerProxy(broker platform.ServiceBroker) bool {
	return strings.HasPrefix(broker.BrokerURL, r.proxyPath)
}

func updateBrokerRegistrations(updateOp func(broker platform.ServiceBroker), a, b []serviceBrokerReg) {
	mb := make(map[string]serviceBrokerReg)
	for _, broker := range b {
		mb[broker.SmID] = broker
	}
	for _, broker := range a {
		if _, ok := mb[broker.SmID]; !ok {
			//TODO at some point we will be hitting platform rate limits... how should we handle that?
			updateOp(broker.ServiceBroker)
		}
	}
}

func logFields(broker *platform.ServiceBroker) logrus.Fields {
	return logrus.Fields{
		"guid": broker.Guid,
		"name": broker.Name,
		"url":  broker.BrokerURL,
	}
}
