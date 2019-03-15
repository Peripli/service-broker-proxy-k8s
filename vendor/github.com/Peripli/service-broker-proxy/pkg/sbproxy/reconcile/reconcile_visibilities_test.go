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
	"sync"
	"time"

	"github.com/Peripli/service-broker-proxy/pkg/platform"
	"github.com/Peripli/service-broker-proxy/pkg/platform/platformfakes"
	"github.com/Peripli/service-broker-proxy/pkg/sm"
	"github.com/Peripli/service-broker-proxy/pkg/sm/smfakes"
	"github.com/Peripli/service-manager/pkg/types"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	cache "github.com/patrickmn/go-cache"
	"github.com/pkg/errors"
)

var _ = Describe("Reconcile visibilities", func() {
	const fakeAppHost = "https://smproxy.com"

	var (
		fakeSMClient *smfakes.FakeClient

		fakePlatformClient         *platformfakes.FakeClient
		fakePlatformCatalogFetcher *platformfakes.FakeCatalogFetcher
		fakePlatformBrokerClient   *platformfakes.FakeBrokerClient
		fakeVisibilityClient       *platformfakes.FakeVisibilityClient

		visibilityCache *cache.Cache

		waitGroup *sync.WaitGroup

		reconciliationTask *ReconciliationTask

		smbroker1 sm.Broker
		smbroker2 sm.Broker

		smService1 types.ServiceOffering
		smService2 types.ServiceOffering
		smService3 types.ServiceOffering

		smPlan1 *types.ServicePlan
		smPlan2 *types.ServicePlan
		smPlan3 *types.ServicePlan
		smPlan4 *types.ServicePlan
		smPlan5 *types.ServicePlan

		platformbroker1        platform.ServiceBroker
		platformbroker2        platform.ServiceBroker
		platformbrokerNonProxy platform.ServiceBroker
	)

	stubGetSMPlans := func() ([]*types.ServicePlan, error) {
		return []*types.ServicePlan{
			smbroker1.ServiceOfferings[0].Plans[0],
			smbroker1.ServiceOfferings[0].Plans[1],
		}, nil
	}

	stubGetSMOfferings := func() ([]*types.ServiceOffering, error) {
		return []*types.ServiceOffering{
			&smbroker1.ServiceOfferings[0],
			&smbroker1.ServiceOfferings[1],
			&smbroker2.ServiceOfferings[0],
		}, nil
	}

	stubCreateBrokerToSucceed := func(ctx context.Context, r *platform.CreateServiceBrokerRequest) (*platform.ServiceBroker, error) {
		return &platform.ServiceBroker{
			GUID:      r.Name,
			Name:      r.Name,
			BrokerURL: r.BrokerURL,
		}, nil
	}

	stubPlatformOpsToSucceed := func() {
		fakePlatformBrokerClient.CreateBrokerStub = stubCreateBrokerToSucceed
		fakePlatformBrokerClient.DeleteBrokerReturns(nil)
		fakePlatformCatalogFetcher.FetchReturns(nil)
	}

	stubPlansByServices := func(_ context.Context, offerings []*types.ServiceOffering) ([]*types.ServicePlan, error) {
		result := make([]*types.ServicePlan, 0)
		for _, offering := range offerings {
			if smPlan1.ServiceOfferingID == offering.ID {
				result = append(result, smPlan1)
			}
			if smPlan2.ServiceOfferingID == offering.ID {
				result = append(result, smPlan2)
			}
			if smPlan2.ServiceOfferingID == offering.ID {
				result = append(result, smPlan3)
			}
		}
		return result, nil
	}

	checkAccessArguments := func(data json.RawMessage, servicePlanGUID, platformBrokerName string, visibilities []*platform.ServiceVisibilityEntity) {
		var labels map[string]string
		err := json.Unmarshal(data, &labels)
		Expect(err).To(Not(HaveOccurred()))
		if labels == nil {
			labels = map[string]string{}
		}
		visibility := &platform.ServiceVisibilityEntity{
			Public:             len(labels) == 0,
			CatalogPlanID:      servicePlanGUID,
			Labels:             labels,
			PlatformBrokerName: platformBrokerName,
		}
		Expect(visibilities).To(ContainElement(visibility))
	}

	setFakeBrokersClients := func() {
		fakeSMClient.GetBrokersReturns([]sm.Broker{
			smbroker1,
			smbroker2,
		}, nil)
		fakePlatformBrokerClient.GetBrokersReturns([]platform.ServiceBroker{
			platformbroker1,
			platformbroker2,
			platformbrokerNonProxy,
		}, nil)
	}

	BeforeEach(func() {
		fakeSMClient = &smfakes.FakeClient{}
		fakePlatformClient = &platformfakes.FakeClient{}
		fakePlatformBrokerClient = &platformfakes.FakeBrokerClient{}
		fakePlatformCatalogFetcher = &platformfakes.FakeCatalogFetcher{}
		fakeVisibilityClient = &platformfakes.FakeVisibilityClient{}

		visibilityCache = cache.New(5*time.Minute, 10*time.Minute)
		waitGroup = &sync.WaitGroup{}

		fakePlatformClient.BrokerReturns(fakePlatformBrokerClient)
		fakePlatformClient.VisibilityReturns(fakeVisibilityClient)
		fakePlatformClient.CatalogFetcherReturns(fakePlatformCatalogFetcher)

		reconciliationTask = NewTask(
			context.TODO(), DefaultSettings(), waitGroup, fakePlatformClient, fakeSMClient,
			fakeAppHost, visibilityCache)

		smPlan1 = &types.ServicePlan{
			ID:                "smBroker1ServiceID1PlanID1",
			CatalogID:         "smBroker1ServiceID1PlanID1",
			Name:              "smBroker1Service1Plan1",
			Description:       "description",
			ServiceOfferingID: "smBroker1ServiceID1",
		}

		smPlan2 = &types.ServicePlan{
			ID:                "smBroker1ServiceID1PlanID2",
			CatalogID:         "smBroker1ServiceID1PlanID2",
			Name:              "smBroker1Service1Plan2",
			Description:       "description",
			ServiceOfferingID: "smBroker1ServiceID1",
		}

		smPlan3 = &types.ServicePlan{
			ID:                "smBroker2ServiceID1PlanID1",
			CatalogID:         "smBroker2ServiceID1PlanID1",
			Name:              "smBroker2Service1Plan1",
			Description:       "description",
			ServiceOfferingID: "smBroker2ServiceID1",
		}

		smPlan4 = &types.ServicePlan{
			ID:                "smBroker2ServiceID1PlanID1",
			CatalogID:         "smBroker2ServiceID1PlanID1",
			Name:              "smBroker2Service1Plan1",
			Description:       "description",
			ServiceOfferingID: "smBroker2ServiceID2",
		}

		smPlan5 = &types.ServicePlan{
			ID:                "smBroker2ServiceID1PlanID2",
			CatalogID:         "smBroker2ServiceID1PlanID2",
			Name:              "smBroker2Service1Plan2",
			Description:       "description",
			ServiceOfferingID: "smBroker2ServiceID2",
		}

		smService1 = types.ServiceOffering{
			ID:                  "smBroker1ServiceID1",
			Name:                "smBroker1Service1",
			Description:         "description",
			Bindable:            true,
			BindingsRetrievable: true,
			Plans: []*types.ServicePlan{
				smPlan1,
				smPlan2,
			},
			BrokerID: "smBrokerID1",
		}

		smService2 = types.ServiceOffering{
			ID:                  "smBroker2ServiceID1",
			Name:                "smBroker2Service2",
			Description:         "description",
			Bindable:            true,
			BindingsRetrievable: true,
			Plans: []*types.ServicePlan{
				smPlan3,
			},
			BrokerID: "smBrokerID1",
		}

		smService3 = types.ServiceOffering{
			ID:                  "smBroker2ServiceID2",
			Name:                "smBroker2Service3",
			Description:         "description",
			Bindable:            true,
			BindingsRetrievable: true,
			Plans: []*types.ServicePlan{
				smPlan4,
				smPlan5,
			},
			BrokerID: "smBrokerID2",
		}

		smbroker1 = sm.Broker{
			ID:        "smBrokerID1",
			BrokerURL: "https://smBroker1.com",
			ServiceOfferings: []types.ServiceOffering{
				smService1,
				smService2,
			},
		}

		smbroker2 = sm.Broker{
			ID:        "smBrokerID2",
			BrokerURL: "https://smBroker2.com",
			ServiceOfferings: []types.ServiceOffering{
				smService3,
			},
		}

		platformbroker1 = platform.ServiceBroker{
			GUID:      "platformBrokerID1",
			Name:      ProxyBrokerPrefix + "smBrokerID1",
			BrokerURL: fakeAppHost + "/" + smbroker1.ID,
		}

		platformbroker2 = platform.ServiceBroker{
			GUID:      "platformBrokerID2",
			Name:      ProxyBrokerPrefix + "smBrokerID2",
			BrokerURL: fakeAppHost + "/" + smbroker2.ID,
		}

		platformbrokerNonProxy = platform.ServiceBroker{
			GUID:      "platformBrokerID3",
			Name:      "platformBroker3",
			BrokerURL: "https://platformBroker3.com",
		}
	})

	type expectations struct {
		enablePlanVisibilityCalledFor  []*platform.ServiceVisibilityEntity
		disablePlanVisibilityCalledFor []*platform.ServiceVisibilityEntity
	}

	type testCase struct {
		stubs func()

		platformVisibilities    func() ([]*platform.ServiceVisibilityEntity, error)
		smVisibilities          func() ([]*types.Visibility, error)
		smPlans                 func() ([]*types.ServicePlan, error)
		smOfferings             func() ([]*types.ServiceOffering, error)
		convertedSMVisibilities func() []*platform.ServiceVisibilityEntity

		expectations func() expectations
	}

	entries := []TableEntry{
		Entry("When no visibilities are present in platform and SM - should not enable access for plan", testCase{
			platformVisibilities: func() ([]*platform.ServiceVisibilityEntity, error) {
				return []*platform.ServiceVisibilityEntity{}, nil
			},
			smVisibilities: func() ([]*types.Visibility, error) {
				return []*types.Visibility{}, nil
			},
			smPlans: stubGetSMPlans,
			expectations: func() expectations {
				return expectations{}
			},
		}),

		Entry("When no visibilities are present in platform and there are some in SM - should reconcile", testCase{
			platformVisibilities: func() ([]*platform.ServiceVisibilityEntity, error) {
				return []*platform.ServiceVisibilityEntity{}, nil
			},
			smVisibilities: func() ([]*types.Visibility, error) {
				return []*types.Visibility{
					&types.Visibility{
						PlatformID:    "platformID",
						ServicePlanID: smbroker1.ServiceOfferings[0].Plans[0].CatalogID,
						Labels: types.Labels{
							"key": []string{"value0", "value1"},
						},
					},
				}, nil
			},
			smOfferings: stubGetSMOfferings,
			smPlans:     stubGetSMPlans,
			expectations: func() expectations {
				return expectations{
					enablePlanVisibilityCalledFor: []*platform.ServiceVisibilityEntity{
						&platform.ServiceVisibilityEntity{
							PlatformBrokerName: ProxyBrokerPrefix + smbroker1.ID,
							CatalogPlanID:      smbroker1.ServiceOfferings[0].Plans[0].CatalogID,
							Labels:             map[string]string{"key": "value0"},
						},
						&platform.ServiceVisibilityEntity{
							PlatformBrokerName: ProxyBrokerPrefix + smbroker1.ID,
							CatalogPlanID:      smbroker1.ServiceOfferings[0].Plans[0].CatalogID,
							Labels:             map[string]string{"key": "value1"},
						},
					},
					disablePlanVisibilityCalledFor: []*platform.ServiceVisibilityEntity{},
				}
			},
		}),

		Entry("When visibilities in platform and in SM are the same - should do nothing", testCase{
			platformVisibilities: func() ([]*platform.ServiceVisibilityEntity, error) {
				return []*platform.ServiceVisibilityEntity{
					&platform.ServiceVisibilityEntity{
						CatalogPlanID:      smbroker1.ServiceOfferings[0].Plans[0].CatalogID,
						Labels:             map[string]string{"key": "value0"},
						PlatformBrokerName: ProxyBrokerPrefix + smbroker1.ID,
					},
					&platform.ServiceVisibilityEntity{
						CatalogPlanID:      smbroker1.ServiceOfferings[0].Plans[0].CatalogID,
						Labels:             map[string]string{"key": "value1"},
						PlatformBrokerName: ProxyBrokerPrefix + smbroker1.ID,
					},
				}, nil
			},
			smVisibilities: func() ([]*types.Visibility, error) {
				return []*types.Visibility{
					&types.Visibility{
						PlatformID:    "platformID",
						ServicePlanID: smbroker1.ServiceOfferings[0].Plans[0].CatalogID,
						Labels: types.Labels{
							"key": []string{"value0", "value1"},
						},
					},
				}, nil
			},
			smPlans: stubGetSMPlans,
			expectations: func() expectations {
				return expectations{
					enablePlanVisibilityCalledFor:  []*platform.ServiceVisibilityEntity{},
					disablePlanVisibilityCalledFor: []*platform.ServiceVisibilityEntity{},
				}
			},
		}),

		Entry("When visibilities in platform and in SM are not the same - should reconcile", testCase{
			platformVisibilities: func() ([]*platform.ServiceVisibilityEntity, error) {
				return []*platform.ServiceVisibilityEntity{
					&platform.ServiceVisibilityEntity{
						CatalogPlanID:      smbroker1.ServiceOfferings[0].Plans[0].CatalogID,
						Labels:             map[string]string{"key": "value2"},
						PlatformBrokerName: ProxyBrokerPrefix + smbroker1.ID,
					},
					&platform.ServiceVisibilityEntity{
						CatalogPlanID:      smbroker1.ServiceOfferings[0].Plans[0].CatalogID,
						Labels:             map[string]string{"key": "value3"},
						PlatformBrokerName: ProxyBrokerPrefix + smbroker1.ID,
					},
				}, nil
			},
			smVisibilities: func() ([]*types.Visibility, error) {
				return []*types.Visibility{
					&types.Visibility{
						PlatformID:    "platformID",
						ServicePlanID: smbroker1.ServiceOfferings[0].Plans[0].CatalogID,
						Labels: types.Labels{
							"key": []string{"value0", "value1"},
						},
					},
				}, nil
			},
			smPlans: stubGetSMPlans,
			expectations: func() expectations {
				return expectations{
					enablePlanVisibilityCalledFor: []*platform.ServiceVisibilityEntity{
						&platform.ServiceVisibilityEntity{
							CatalogPlanID:      smbroker1.ServiceOfferings[0].Plans[0].CatalogID,
							Labels:             map[string]string{"key": "value0"},
							PlatformBrokerName: ProxyBrokerPrefix + smbroker1.ID,
						},
						&platform.ServiceVisibilityEntity{
							CatalogPlanID:      smbroker1.ServiceOfferings[0].Plans[0].CatalogID,
							Labels:             map[string]string{"key": "value1"},
							PlatformBrokerName: ProxyBrokerPrefix + smbroker1.ID,
						},
					},
					disablePlanVisibilityCalledFor: []*platform.ServiceVisibilityEntity{
						&platform.ServiceVisibilityEntity{
							CatalogPlanID:      smbroker1.ServiceOfferings[0].Plans[0].CatalogID,
							Labels:             map[string]string{"key": "value2"},
							PlatformBrokerName: ProxyBrokerPrefix + smbroker1.ID,
						},
						&platform.ServiceVisibilityEntity{
							CatalogPlanID:      smbroker1.ServiceOfferings[0].Plans[0].CatalogID,
							Labels:             map[string]string{"key": "value3"},
							PlatformBrokerName: ProxyBrokerPrefix + smbroker1.ID,
						},
					},
				}
			},
		}),

		Entry("When enable visibility returns error - should continue with reconciliation", testCase{
			stubs: func() {
				fakeVisibilityClient.EnableAccessForPlanReturnsOnCall(0, errors.New("Expected"))
			},
			platformVisibilities: func() ([]*platform.ServiceVisibilityEntity, error) {
				return []*platform.ServiceVisibilityEntity{}, nil
			},
			smVisibilities: func() ([]*types.Visibility, error) {
				return []*types.Visibility{
					&types.Visibility{
						PlatformID:    "platformID",
						ServicePlanID: smbroker1.ServiceOfferings[0].Plans[0].CatalogID,
						Labels: types.Labels{
							"key": []string{"value0"},
						},
					},
					&types.Visibility{
						PlatformID:    "platformID",
						ServicePlanID: smbroker1.ServiceOfferings[0].Plans[1].CatalogID,
						Labels: types.Labels{
							"key": []string{"value1"},
						},
					},
				}, nil
			},
			smPlans: stubGetSMPlans,
			expectations: func() expectations {
				return expectations{
					enablePlanVisibilityCalledFor: []*platform.ServiceVisibilityEntity{
						&platform.ServiceVisibilityEntity{
							CatalogPlanID:      smbroker1.ServiceOfferings[0].Plans[0].CatalogID,
							Labels:             map[string]string{"key": "value0"},
							PlatformBrokerName: ProxyBrokerPrefix + smbroker1.ID,
						},
						&platform.ServiceVisibilityEntity{
							CatalogPlanID:      smbroker1.ServiceOfferings[0].Plans[1].CatalogID,
							Labels:             map[string]string{"key": "value1"},
							PlatformBrokerName: ProxyBrokerPrefix + smbroker1.ID,
						},
					},
				}
			},
		}),

		Entry("When disable visibility returns error - should continue with reconciliation", testCase{
			stubs: func() {
				fakeVisibilityClient.DisableAccessForPlanReturnsOnCall(0, errors.New("Expected"))
			},
			platformVisibilities: func() ([]*platform.ServiceVisibilityEntity, error) {
				return []*platform.ServiceVisibilityEntity{
					&platform.ServiceVisibilityEntity{
						CatalogPlanID:      smbroker1.ServiceOfferings[0].Plans[0].CatalogID,
						Labels:             map[string]string{"key": "value0"},
						PlatformBrokerName: ProxyBrokerPrefix + smbroker1.ID,
					},
					&platform.ServiceVisibilityEntity{
						CatalogPlanID:      smbroker1.ServiceOfferings[0].Plans[1].CatalogID,
						Labels:             map[string]string{"key": "value1"},
						PlatformBrokerName: ProxyBrokerPrefix + smbroker1.ID,
					},
				}, nil
			},
			smVisibilities: func() ([]*types.Visibility, error) {
				return []*types.Visibility{}, nil
			},
			smPlans: stubGetSMPlans,
			expectations: func() expectations {
				return expectations{
					disablePlanVisibilityCalledFor: []*platform.ServiceVisibilityEntity{
						&platform.ServiceVisibilityEntity{
							CatalogPlanID:      smbroker1.ServiceOfferings[0].Plans[0].CatalogID,
							Labels:             map[string]string{"key": "value0"},
							PlatformBrokerName: ProxyBrokerPrefix + smbroker1.ID,
						},
						&platform.ServiceVisibilityEntity{
							CatalogPlanID:      smbroker1.ServiceOfferings[0].Plans[1].CatalogID,
							Labels:             map[string]string{"key": "value1"},
							PlatformBrokerName: ProxyBrokerPrefix + smbroker1.ID,
						},
					},
				}
			},
		}),

		Entry("When visibility from SM doesn't have scope label and scope is enabled - should not enable visibility", testCase{
			platformVisibilities: func() ([]*platform.ServiceVisibilityEntity, error) {
				return []*platform.ServiceVisibilityEntity{}, nil
			},
			smVisibilities: func() ([]*types.Visibility, error) {
				return []*types.Visibility{
					&types.Visibility{
						PlatformID:    "platformID",
						ServicePlanID: smbroker1.ServiceOfferings[0].Plans[0].CatalogID,
						Labels: types.Labels{
							"some key": []string{"some value"},
						},
					},
				}, nil
			},
			smPlans: stubGetSMPlans,
			expectations: func() expectations {
				return expectations{}
			},
		}),

		Entry("When visibility from SM doesn't have scope label and scope is disabled - should enable visibility", testCase{
			stubs: func() {
				fakeVisibilityClient.VisibilityScopeLabelKeyReturns("")
			},
			platformVisibilities: func() ([]*platform.ServiceVisibilityEntity, error) {
				return []*platform.ServiceVisibilityEntity{}, nil
			},
			smVisibilities: func() ([]*types.Visibility, error) {
				return []*types.Visibility{
					&types.Visibility{
						PlatformID:    "platformID",
						ServicePlanID: smbroker1.ServiceOfferings[0].Plans[0].CatalogID,
						Labels: types.Labels{
							"some key": []string{"some value"},
						},
					},
				}, nil
			},
			smPlans: stubGetSMPlans,
			expectations: func() expectations {
				return expectations{
					enablePlanVisibilityCalledFor: []*platform.ServiceVisibilityEntity{
						&platform.ServiceVisibilityEntity{
							Public:             true,
							CatalogPlanID:      smbroker1.ServiceOfferings[0].Plans[0].CatalogID,
							Labels:             map[string]string{},
							PlatformBrokerName: ProxyBrokerPrefix + smbroker1.ID,
						},
					},
				}
			},
		}),

		Entry("When visibilities in platform and in SM are both public - should reconcile", testCase{
			platformVisibilities: func() ([]*platform.ServiceVisibilityEntity, error) {
				return []*platform.ServiceVisibilityEntity{
					&platform.ServiceVisibilityEntity{
						Public:             true,
						CatalogPlanID:      smbroker1.ServiceOfferings[0].Plans[1].CatalogID,
						PlatformBrokerName: ProxyBrokerPrefix + smbroker1.ID,
					},
				}, nil
			},
			smVisibilities: func() ([]*types.Visibility, error) {
				return []*types.Visibility{
					&types.Visibility{
						ServicePlanID: smbroker1.ServiceOfferings[0].Plans[0].CatalogID,
					},
				}, nil
			},
			smPlans: stubGetSMPlans,
			expectations: func() expectations {
				return expectations{
					enablePlanVisibilityCalledFor: []*platform.ServiceVisibilityEntity{
						&platform.ServiceVisibilityEntity{
							Public:             true,
							CatalogPlanID:      smbroker1.ServiceOfferings[0].Plans[0].CatalogID,
							Labels:             map[string]string{},
							PlatformBrokerName: ProxyBrokerPrefix + smbroker1.ID,
						},
					},
					disablePlanVisibilityCalledFor: []*platform.ServiceVisibilityEntity{
						&platform.ServiceVisibilityEntity{
							Public:             true,
							CatalogPlanID:      smbroker1.ServiceOfferings[0].Plans[1].CatalogID,
							Labels:             map[string]string{},
							PlatformBrokerName: ProxyBrokerPrefix + smbroker1.ID,
						},
					},
				}
			},
		}),

		Entry("When plans from SM could not be fetched - should not reconcile", testCase{
			platformVisibilities: func() ([]*platform.ServiceVisibilityEntity, error) {
				return []*platform.ServiceVisibilityEntity{
					&platform.ServiceVisibilityEntity{
						Public:             true,
						CatalogPlanID:      smbroker1.ServiceOfferings[0].Plans[1].CatalogID,
						PlatformBrokerName: ProxyBrokerPrefix + smbroker1.ID,
					},
				}, nil
			},
			smVisibilities: func() ([]*types.Visibility, error) {
				return []*types.Visibility{
					&types.Visibility{
						ServicePlanID: smbroker1.ServiceOfferings[0].Plans[0].CatalogID,
					},
				}, nil
			},
			smOfferings: func() ([]*types.ServiceOffering, error) { return nil, errors.New("Expected") },
			smPlans:     func() ([]*types.ServicePlan, error) { return nil, errors.New("Expected") },
			expectations: func() expectations {
				return expectations{}
			},
		}),

		Entry("When visibilities from SM cannot be fetched - no reconcilation is done", testCase{
			platformVisibilities: func() ([]*platform.ServiceVisibilityEntity, error) {
				return []*platform.ServiceVisibilityEntity{
					&platform.ServiceVisibilityEntity{
						Public:             true,
						CatalogPlanID:      smbroker1.ServiceOfferings[0].Plans[1].CatalogID,
						PlatformBrokerName: ProxyBrokerPrefix + smbroker1.ID,
					},
				}, nil
			},
			smVisibilities: func() ([]*types.Visibility, error) {
				return nil, errors.New("Expected")
			},
			smPlans: stubGetSMPlans,
			expectations: func() expectations {
				return expectations{}
			},
		}),

		Entry("When visibilities from platform cannot be fetched - no reconcilation is done", testCase{
			platformVisibilities: func() ([]*platform.ServiceVisibilityEntity, error) {
				return nil, errors.New("Expected")
			},
			smVisibilities: func() ([]*types.Visibility, error) {
				return []*types.Visibility{
					&types.Visibility{
						ServicePlanID: smbroker1.ServiceOfferings[0].Plans[0].CatalogID,
					},
				}, nil
			},
			smPlans: stubGetSMPlans,
			expectations: func() expectations {
				return expectations{}
			},
		}),
	}

	DescribeTable("Run", func(t testCase) {
		setFakeBrokersClients()

		fakeSMClient.GetVisibilitiesReturns(t.smVisibilities())
		if t.smOfferings == nil {
			t.smOfferings = stubGetSMOfferings
		}
		fakeSMClient.GetServiceOfferingsByBrokerIDsReturns(t.smOfferings())
		fakeSMClient.GetPlansByServiceOfferingsStub = stubPlansByServices

		fakeVisibilityClient.GetVisibilitiesByBrokersReturns(t.platformVisibilities())

		fakeVisibilityClient.VisibilityScopeLabelKeyReturns("key")

		stubPlatformOpsToSucceed()

		if t.stubs != nil {
			t.stubs()
		}

		reconciliationTask.Run()

		Expect(fakeSMClient.GetBrokersCallCount()).To(Equal(1))
		Expect(fakePlatformBrokerClient.GetBrokersCallCount()).To(Equal(1))

		expected := t.expectations()

		Expect(fakeVisibilityClient.EnableAccessForPlanCallCount()).To(Equal(len(expected.enablePlanVisibilityCalledFor)))

		for index := range expected.enablePlanVisibilityCalledFor {
			_, data, servicePlanGUID, platformBrokerName := fakeVisibilityClient.EnableAccessForPlanArgsForCall(index)
			checkAccessArguments(data, servicePlanGUID, platformBrokerName, expected.enablePlanVisibilityCalledFor)
		}

		Expect(fakeVisibilityClient.DisableAccessForPlanCallCount()).To(Equal(len(expected.disablePlanVisibilityCalledFor)))

		for index := range expected.disablePlanVisibilityCalledFor {
			_, data, servicePlanGUID, platformBrokerName := fakeVisibilityClient.DisableAccessForPlanArgsForCall(index)
			checkAccessArguments(data, servicePlanGUID, platformBrokerName, expected.disablePlanVisibilityCalledFor)
		}

	}, entries...)

	Describe("Run cache", func() {

		setVisibilityClients := func() {
			fakeSMClient.GetVisibilitiesReturns([]*types.Visibility{}, nil)
			fakeSMClient.GetPlansReturns(stubGetSMPlans())

			fakeVisibilityClient.GetVisibilitiesByBrokersReturns([]*platform.ServiceVisibilityEntity{}, nil)
			fakeVisibilityClient.VisibilityScopeLabelKeyReturns("key")
		}

		setFakes := func() {
			setFakeBrokersClients()
			setVisibilityClients()
			stubPlatformOpsToSucceed()
		}

		assertCallCounts := func(nonCachedCallsCounts, platformVisibilityCallCount int) {
			Expect(fakeSMClient.GetBrokersCallCount()).To(Equal(nonCachedCallsCounts))
			Expect(fakePlatformBrokerClient.GetBrokersCallCount()).To(Equal(nonCachedCallsCounts))
			Expect(fakeVisibilityClient.GetVisibilitiesByBrokersCallCount()).To(Equal(platformVisibilityCallCount))
			Expect(fakeSMClient.GetVisibilitiesCallCount()).To(Equal(nonCachedCallsCounts))
		}

		BeforeEach(func() {
			setFakes()
			reconciliationTask.Run()
			assertCallCounts(1, 1)
		})

		Context("when visibility cache is invalid", func() {
			It("should call platform", func() {
				visibilityCache.Replace(platformVisibilityCacheKey, nil, time.Minute)
				reconciliationTask.Run()
				assertCallCounts(2, 2)
			})
		})

		Context("when visibility cache has expired", func() {
			It("should call platform", func() {
				visibilities, found := visibilityCache.Get(platformVisibilityCacheKey)
				Expect(found).To(BeTrue())
				visibilityCache.Set(platformVisibilityCacheKey, visibilities, time.Nanosecond)
				time.Sleep(time.Nanosecond)
				reconciliationTask.Run()
				assertCallCounts(2, 2)
			})
		})

		Context("when plan cache is invalid", func() {
			It("should call platform", func() {
				visibilityCache.Replace(smPlansCacheKey, nil, time.Minute)
				reconciliationTask.Run()
				assertCallCounts(2, 2)
			})
		})

		Context("when plan cache has expired", func() {
			It("should call platform", func() {
				plans, found := visibilityCache.Get(smPlansCacheKey)
				Expect(found).To(BeTrue())
				visibilityCache.Set(smPlansCacheKey, plans, time.Nanosecond)
				time.Sleep(time.Nanosecond)
				reconciliationTask.Run()
				assertCallCounts(2, 2)
			})
		})

		Context("when there are no changes in SM plans", func() {
			It("should use cache", func() {
				reconciliationTask.Run()
				assertCallCounts(2, 1)
			})
		})

		Context("when there are changes in SM plans", func() {
			Context("and plans are not the same count", func() {
				It("should not use cache", func() {
					fakeSMClient.GetServiceOfferingsByBrokerIDsReturns([]*types.ServiceOffering{
						&smbroker1.ServiceOfferings[0],
					}, nil)
					fakeSMClient.GetPlansByServiceOfferingsReturns([]*types.ServicePlan{
						smbroker1.ServiceOfferings[0].Plans[0],
					}, nil)
					reconciliationTask.Run()
					assertCallCounts(2, 2)
				})
			})

			Context("and plans are the same count but different", func() {
				It("should not use cache", func() {
					fakeSMClient.GetServiceOfferingsByBrokerIDsReturns([]*types.ServiceOffering{
						&smbroker1.ServiceOfferings[0],
						&smbroker1.ServiceOfferings[1],
					}, nil)
					fakeSMClient.GetPlansByServiceOfferingsReturns([]*types.ServicePlan{
						smbroker1.ServiceOfferings[0].Plans[0],
						smbroker1.ServiceOfferings[1].Plans[0],
					}, nil)

					reconciliationTask.Run()
					assertCallCounts(2, 2)
				})
			})
		})
	})

})
