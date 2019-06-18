package reconcile

import (
	"context"

	"github.com/Peripli/service-broker-proxy/pkg/platform"
	"github.com/Peripli/service-broker-proxy/pkg/sm"

	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
)

// NewResyncer returns a resyncer that reconciles the state of the proxy brokers and visibilities
// in the platform to match the desired state provided by the Service Manager.
func NewResyncer(settings *Settings, platformClient platform.Client, smClient sm.Client, proxyPath string) Resyncer {
	return &resyncJob{
		options:        settings,
		platformClient: platformClient,
		smClient:       smClient,
		proxyPath:      proxyPath,
	}
}

type resyncJob struct {
	options        *Settings
	platformClient platform.Client
	smClient       sm.Client
	proxyPath      string
}

// Resync reconciles the state of the proxy brokers and visibilities at the platform
// with the brokers provided by the Service Manager
func (r *resyncJob) Resync(ctx context.Context) {
	resyncContext, taskID, err := createResyncContext(ctx)
	if err != nil {
		log.C(ctx).WithError(err).Error("could not create resync job context")
		return
	}
	log.C(ctx).Infof("STARTING resync job %s...", taskID)
	r.process(resyncContext)
	log.C(ctx).Infof("FINISHED resync job %s", taskID)
}

func createResyncContext(ctx context.Context) (context.Context, string, error) {
	correlationID, err := uuid.NewV4()
	if err != nil {
		return nil, "", errors.Wrap(err, "could not generate correlationID")
	}
	entry := log.C(ctx).WithField(log.FieldCorrelationID, correlationID.String())
	return log.ContextWithLogger(ctx, entry), correlationID.String(), nil
}

func (r *resyncJob) process(ctx context.Context) {
	logger := log.C(ctx)
	if r.platformClient.Broker() == nil || r.platformClient.Visibility() == nil {
		logger.Error("platform must be able to handle both brokers and visibilities. Cannot perform reconciliation")
		return
	}

	smBrokers, err := r.getBrokersFromSM(ctx)
	if err != nil {
		logger.WithError(err).Error("an error occurred while obtaining brokers from Service Manager")
		return
	}

	plansFromSM, err := r.getSMPlans(ctx, smBrokers)
	if err != nil {
		logger.Error(err)
		return
	}

	mappedPlans := mapPlansByBrokerPlanID(plansFromSM)
	smVisibilities, err := r.getVisibilitiesFromSM(ctx, mappedPlans, smBrokers) // fetch as soon as possible
	if err != nil {
		logger.WithError(err).Error("an error occurred while obtaining visibilities from Service Manager")
		return
	}

	// get all the registered brokers from the platform
	logger.Info("resyncJob getting brokers from platform...")
	brokersFromPlatform, err := r.platformClient.Broker().GetBrokers(ctx)
	if err != nil {
		logger.WithError(err).Error("an error occurred while obtaining brokers from Platform")
		return
	}
	logger.Infof("resyncJob SUCCESSFULLY retrieved %d brokers from platform", len(brokersFromPlatform))

	r.reconcileBrokers(ctx, brokersFromPlatform, smBrokers)
	r.reconcileVisibilities(ctx, smVisibilities, smBrokers, mappedPlans)
}

func (r *resyncJob) getSMPlans(ctx context.Context, smBrokers []platform.ServiceBroker) (map[string][]*types.ServicePlan, error) {
	smOfferings, err := r.getSMServiceOfferingsByBrokers(ctx, smBrokers)
	if err != nil {
		return nil, errors.Wrap(err, "an error occurred while obtaining service offerings from Service Manager")
	}
	smPlans, err := r.getSMPlansByBrokersAndOfferings(ctx, smOfferings)
	if err != nil {
		return nil, errors.Wrap(err, "an error occurred while obtaining plans from Service Manager")
	}
	return smPlans, nil
}
