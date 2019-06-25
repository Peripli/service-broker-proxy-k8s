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

package reconcile_test

import (
	"context"

	"github.com/Peripli/service-broker-proxy/pkg/platform"
	"github.com/Peripli/service-broker-proxy/pkg/platform/platformfakes"
	"github.com/Peripli/service-broker-proxy/pkg/sbproxy/reconcile"
	"github.com/Peripli/service-broker-proxy/pkg/sm"
	"github.com/Peripli/service-broker-proxy/pkg/sm/smfakes"
	"github.com/Peripli/service-manager/pkg/types"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
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

		reconciler *reconcile.Reconciler

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

	flattenLabelsMap := func(labels types.Labels) []map[string]string {
		m := make([]map[string]string, len(labels), len(labels))
		for k, values := range labels {
			for i, value := range values {
				if m[i] == nil {
					m[i] = make(map[string]string)
				}
				m[i][k] = value
			}
		}

		return m
	}

	checkAccessArguments := func(data types.Labels, servicePlanGUID, platformBrokerName string, visibilities []*platform.Visibility) {
		maps := flattenLabelsMap(data)
		if len(maps) == 0 {
			visibility := &platform.Visibility{
				Public:             true,
				CatalogPlanID:      servicePlanGUID,
				Labels:             map[string]string{},
				PlatformBrokerName: platformBrokerName,
			}
			Expect(visibilities).To(ContainElement(visibility))
		} else {
			for _, m := range maps {
				visibility := &platform.Visibility{
					Public:             false,
					CatalogPlanID:      servicePlanGUID,
					Labels:             m,
					PlatformBrokerName: platformBrokerName,
				}
				Expect(visibilities).To(ContainElement(visibility))
			}
		}
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

		fakePlatformClient.BrokerReturns(fakePlatformBrokerClient)
		fakePlatformClient.VisibilityReturns(fakeVisibilityClient)
		fakePlatformClient.CatalogFetcherReturns(fakePlatformCatalogFetcher)

		reconciler = &reconcile.Reconciler{
			Resyncer: reconcile.NewResyncer(reconcile.DefaultSettings(), fakePlatformClient, fakeSMClient, fakeAppHost),
		}

		smPlan1 = &types.ServicePlan{
			Base: types.Base{
				ID: "smBroker1ServiceID1PlanID1",
			},
			CatalogID:         "smBroker1ServiceID1PlanID1",
			Name:              "smBroker1Service1Plan1",
			Description:       "description",
			ServiceOfferingID: "smBroker1ServiceID1",
		}

		smPlan2 = &types.ServicePlan{
			Base: types.Base{
				ID: "smBroker1ServiceID1PlanID2",
			},
			CatalogID:         "smBroker1ServiceID1PlanID2",
			Name:              "smBroker1Service1Plan2",
			Description:       "description",
			ServiceOfferingID: "smBroker1ServiceID1",
		}

		smPlan3 = &types.ServicePlan{
			Base: types.Base{
				ID: "smBroker2ServiceID1PlanID1",
			},
			CatalogID:         "smBroker2ServiceID1PlanID1",
			Name:              "smBroker2Service1Plan1",
			Description:       "description",
			ServiceOfferingID: "smBroker2ServiceID1",
		}

		smPlan4 = &types.ServicePlan{
			Base: types.Base{
				ID: "smBroker2ServiceID1PlanID1",
			},
			CatalogID:         "smBroker2ServiceID1PlanID1",
			Name:              "smBroker2Service1Plan1",
			Description:       "description",
			ServiceOfferingID: "smBroker2ServiceID2",
		}

		smPlan5 = &types.ServicePlan{
			Base: types.Base{
				ID: "smBroker2ServiceID1PlanID2",
			},
			CatalogID:         "smBroker2ServiceID1PlanID2",
			Name:              "smBroker2Service1Plan2",
			Description:       "description",
			ServiceOfferingID: "smBroker2ServiceID2",
		}

		smService1 = types.ServiceOffering{
			Base: types.Base{
				ID: "smBroker1ServiceID1",
			},
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
			Base: types.Base{
				ID: "smBroker2ServiceID1",
			},
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
			Base: types.Base{
				ID: "smBroker2ServiceID2",
			},
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
			Name:      "smBroker1",
			BrokerURL: "https://smBroker1.com",
			ServiceOfferings: []types.ServiceOffering{
				smService1,
				smService2,
			},
		}

		smbroker2 = sm.Broker{
			ID:        "smBrokerID2",
			Name:      "smBroker2",
			BrokerURL: "https://smBroker2.com",
			ServiceOfferings: []types.ServiceOffering{
				smService3,
			},
		}

		platformbroker1 = platform.ServiceBroker{
			GUID:      "platformBrokerID1",
			Name:      brokerProxyName(smbroker1.Name, smbroker1.ID),
			BrokerURL: fakeAppHost + "/" + smbroker1.ID,
		}

		platformbroker2 = platform.ServiceBroker{
			GUID:      "platformBrokerID2",
			Name:      brokerProxyName(smbroker2.Name, smbroker2.ID),
			BrokerURL: fakeAppHost + "/" + smbroker2.ID,
		}

		platformbrokerNonProxy = platform.ServiceBroker{
			GUID:      "platformBrokerID3",
			Name:      "platformBroker3",
			BrokerURL: "https://platformBroker3.com",
		}
	})

	type expectations struct {
		enablePlanVisibilityCalledFor  []*platform.Visibility
		disablePlanVisibilityCalledFor []*platform.Visibility
	}

	type testCase struct {
		stubs func()

		platformVisibilities func() ([]*platform.Visibility, error)
		smVisibilities       func() ([]*types.Visibility, error)
		smPlans              func() ([]*types.ServicePlan, error)
		smOfferings          func() ([]*types.ServiceOffering, error)

		expectations func() expectations
	}

	entries := []TableEntry{
		Entry("When no visibilities are present in platform and SM - should not enable access for plan", testCase{
			platformVisibilities: func() ([]*platform.Visibility, error) {
				return []*platform.Visibility{}, nil
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
			platformVisibilities: func() ([]*platform.Visibility, error) {
				return []*platform.Visibility{}, nil
			},
			smVisibilities: func() ([]*types.Visibility, error) {
				return []*types.Visibility{
					{
						Base: types.Base{
							Labels: types.Labels{
								"key": []string{"value0", "value1"},
							},
						},
						PlatformID:    "platformID",
						ServicePlanID: smbroker1.ServiceOfferings[0].Plans[0].CatalogID,
					},
				}, nil
			},
			smOfferings: stubGetSMOfferings,
			smPlans:     stubGetSMPlans,
			expectations: func() expectations {
				return expectations{
					enablePlanVisibilityCalledFor: []*platform.Visibility{
						{
							PlatformBrokerName: brokerProxyName(smbroker1.Name, smbroker1.ID),
							CatalogPlanID:      smbroker1.ServiceOfferings[0].Plans[0].CatalogID,
							Labels:             map[string]string{"key": "value0"},
						},
						{
							PlatformBrokerName: brokerProxyName(smbroker1.Name, smbroker1.ID),
							CatalogPlanID:      smbroker1.ServiceOfferings[0].Plans[0].CatalogID,
							Labels:             map[string]string{"key": "value1"},
						},
					},
					disablePlanVisibilityCalledFor: []*platform.Visibility{},
				}
			},
		}),

		Entry("When visibilities in platform and in SM are the same - should do nothing", testCase{
			platformVisibilities: func() ([]*platform.Visibility, error) {
				return []*platform.Visibility{
					{
						CatalogPlanID:      smbroker1.ServiceOfferings[0].Plans[0].CatalogID,
						Labels:             map[string]string{"key": "value0"},
						PlatformBrokerName: brokerProxyName(smbroker1.Name, smbroker1.ID),
					},
					{
						CatalogPlanID:      smbroker1.ServiceOfferings[0].Plans[0].CatalogID,
						Labels:             map[string]string{"key": "value1"},
						PlatformBrokerName: brokerProxyName(smbroker1.Name, smbroker1.ID),
					},
				}, nil
			},
			smVisibilities: func() ([]*types.Visibility, error) {
				return []*types.Visibility{
					{
						Base: types.Base{
							Labels: types.Labels{
								"key": []string{"value0", "value1"},
							},
						},
						PlatformID:    "platformID",
						ServicePlanID: smbroker1.ServiceOfferings[0].Plans[0].CatalogID,
					},
				}, nil
			},
			smPlans: stubGetSMPlans,
			expectations: func() expectations {
				return expectations{
					enablePlanVisibilityCalledFor:  []*platform.Visibility{},
					disablePlanVisibilityCalledFor: []*platform.Visibility{},
				}
			},
		}),

		Entry("When visibilities in platform and in SM are not the same - should reconcile", testCase{
			platformVisibilities: func() ([]*platform.Visibility, error) {
				return []*platform.Visibility{
					{
						CatalogPlanID:      smbroker1.ServiceOfferings[0].Plans[0].CatalogID,
						Labels:             map[string]string{"key": "value2"},
						PlatformBrokerName: brokerProxyName(smbroker1.Name, smbroker1.ID),
					},
					{
						CatalogPlanID:      smbroker1.ServiceOfferings[0].Plans[0].CatalogID,
						Labels:             map[string]string{"key": "value3"},
						PlatformBrokerName: brokerProxyName(smbroker1.Name, smbroker1.ID),
					},
				}, nil
			},
			smVisibilities: func() ([]*types.Visibility, error) {
				return []*types.Visibility{
					{
						Base: types.Base{
							Labels: types.Labels{
								"key": []string{"value0", "value1"},
							},
						},
						PlatformID:    "platformID",
						ServicePlanID: smbroker1.ServiceOfferings[0].Plans[0].CatalogID,
					},
				}, nil
			},
			smPlans: stubGetSMPlans,
			expectations: func() expectations {
				return expectations{
					enablePlanVisibilityCalledFor: []*platform.Visibility{
						{
							CatalogPlanID:      smbroker1.ServiceOfferings[0].Plans[0].CatalogID,
							Labels:             map[string]string{"key": "value0"},
							PlatformBrokerName: brokerProxyName(smbroker1.Name, smbroker1.ID),
						},
						{
							CatalogPlanID:      smbroker1.ServiceOfferings[0].Plans[0].CatalogID,
							Labels:             map[string]string{"key": "value1"},
							PlatformBrokerName: brokerProxyName(smbroker1.Name, smbroker1.ID),
						},
					},
					disablePlanVisibilityCalledFor: []*platform.Visibility{
						{
							CatalogPlanID:      smbroker1.ServiceOfferings[0].Plans[0].CatalogID,
							Labels:             map[string]string{"key": "value2"},
							PlatformBrokerName: brokerProxyName(smbroker1.Name, smbroker1.ID),
						},
						{
							CatalogPlanID:      smbroker1.ServiceOfferings[0].Plans[0].CatalogID,
							Labels:             map[string]string{"key": "value3"},
							PlatformBrokerName: brokerProxyName(smbroker1.Name, smbroker1.ID),
						},
					},
				}
			},
		}),

		Entry("When enable visibility returns error - should continue with reconciliation", testCase{
			stubs: func() {
				fakeVisibilityClient.EnableAccessForPlanReturnsOnCall(0, errors.New("Expected"))
			},
			platformVisibilities: func() ([]*platform.Visibility, error) {
				return []*platform.Visibility{}, nil
			},
			smVisibilities: func() ([]*types.Visibility, error) {
				return []*types.Visibility{
					{
						Base: types.Base{
							Labels: types.Labels{
								"key": []string{"value0"},
							},
						},
						PlatformID:    "platformID",
						ServicePlanID: smbroker1.ServiceOfferings[0].Plans[0].CatalogID,
					},
					{
						Base: types.Base{
							Labels: types.Labels{
								"key": []string{"value1"},
							},
						},
						PlatformID:    "platformID",
						ServicePlanID: smbroker1.ServiceOfferings[0].Plans[1].CatalogID,
					},
				}, nil
			},
			smPlans: stubGetSMPlans,
			expectations: func() expectations {
				return expectations{
					enablePlanVisibilityCalledFor: []*platform.Visibility{
						{
							CatalogPlanID:      smbroker1.ServiceOfferings[0].Plans[0].CatalogID,
							Labels:             map[string]string{"key": "value0"},
							PlatformBrokerName: brokerProxyName(smbroker1.Name, smbroker1.ID),
						},
						{
							CatalogPlanID:      smbroker1.ServiceOfferings[0].Plans[1].CatalogID,
							Labels:             map[string]string{"key": "value1"},
							PlatformBrokerName: brokerProxyName(smbroker1.Name, smbroker1.ID),
						},
					},
				}
			},
		}),

		Entry("When disable visibility returns error - should continue with reconciliation", testCase{
			stubs: func() {
				fakeVisibilityClient.DisableAccessForPlanReturnsOnCall(0, errors.New("Expected"))
			},
			platformVisibilities: func() ([]*platform.Visibility, error) {
				return []*platform.Visibility{
					{
						CatalogPlanID:      smbroker1.ServiceOfferings[0].Plans[0].CatalogID,
						Labels:             map[string]string{"key": "value0"},
						PlatformBrokerName: brokerProxyName(smbroker1.Name, smbroker1.ID),
					},
					{
						CatalogPlanID:      smbroker1.ServiceOfferings[0].Plans[1].CatalogID,
						Labels:             map[string]string{"key": "value1"},
						PlatformBrokerName: brokerProxyName(smbroker1.Name, smbroker1.ID),
					},
				}, nil
			},
			smVisibilities: func() ([]*types.Visibility, error) {
				return []*types.Visibility{}, nil
			},
			smPlans: stubGetSMPlans,
			expectations: func() expectations {
				return expectations{
					disablePlanVisibilityCalledFor: []*platform.Visibility{
						{
							CatalogPlanID:      smbroker1.ServiceOfferings[0].Plans[0].CatalogID,
							Labels:             map[string]string{"key": "value0"},
							PlatformBrokerName: brokerProxyName(smbroker1.Name, smbroker1.ID),
						},
						{
							CatalogPlanID:      smbroker1.ServiceOfferings[0].Plans[1].CatalogID,
							Labels:             map[string]string{"key": "value1"},
							PlatformBrokerName: brokerProxyName(smbroker1.Name, smbroker1.ID),
						},
					},
				}
			},
		}),

		Entry("When visibility from SM doesn't have scope label and scope is enabled - should not enable visibility", testCase{
			platformVisibilities: func() ([]*platform.Visibility, error) {
				return []*platform.Visibility{}, nil
			},
			smVisibilities: func() ([]*types.Visibility, error) {
				return []*types.Visibility{
					{
						Base: types.Base{
							Labels: types.Labels{
								"some key": []string{"some value"},
							},
						},
						PlatformID:    "platformID",
						ServicePlanID: smbroker1.ServiceOfferings[0].Plans[0].CatalogID,
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
			platformVisibilities: func() ([]*platform.Visibility, error) {
				return []*platform.Visibility{}, nil
			},
			smVisibilities: func() ([]*types.Visibility, error) {
				return []*types.Visibility{
					{
						Base: types.Base{
							Labels: types.Labels{
								"some key": []string{"some value"},
							},
						},
						PlatformID:    "platformID",
						ServicePlanID: smbroker1.ServiceOfferings[0].Plans[0].CatalogID,
					},
				}, nil
			},
			smPlans: stubGetSMPlans,
			expectations: func() expectations {
				return expectations{
					enablePlanVisibilityCalledFor: []*platform.Visibility{
						{
							Public:             true,
							CatalogPlanID:      smbroker1.ServiceOfferings[0].Plans[0].CatalogID,
							Labels:             map[string]string{},
							PlatformBrokerName: brokerProxyName(smbroker1.Name, smbroker1.ID),
						},
					},
				}
			},
		}),

		Entry("When visibilities in platform and in SM are both public - should reconcile", testCase{
			platformVisibilities: func() ([]*platform.Visibility, error) {
				return []*platform.Visibility{
					{
						Public:             true,
						CatalogPlanID:      smbroker1.ServiceOfferings[0].Plans[1].CatalogID,
						PlatformBrokerName: brokerProxyName(smbroker1.Name, smbroker1.ID),
					},
				}, nil
			},
			smVisibilities: func() ([]*types.Visibility, error) {
				return []*types.Visibility{
					{
						ServicePlanID: smbroker1.ServiceOfferings[0].Plans[0].CatalogID,
					},
				}, nil
			},
			smPlans: stubGetSMPlans,
			expectations: func() expectations {
				return expectations{
					enablePlanVisibilityCalledFor: []*platform.Visibility{
						{
							Public:             true,
							CatalogPlanID:      smbroker1.ServiceOfferings[0].Plans[0].CatalogID,
							Labels:             map[string]string{},
							PlatformBrokerName: brokerProxyName(smbroker1.Name, smbroker1.ID),
						},
					},
					disablePlanVisibilityCalledFor: []*platform.Visibility{
						{
							Public:             true,
							CatalogPlanID:      smbroker1.ServiceOfferings[0].Plans[1].CatalogID,
							Labels:             map[string]string{},
							PlatformBrokerName: brokerProxyName(smbroker1.Name, smbroker1.ID),
						},
					},
				}
			},
		}),

		Entry("When plans from SM could not be fetched - should not reconcile", testCase{
			platformVisibilities: func() ([]*platform.Visibility, error) {
				return []*platform.Visibility{
					{
						Public:             true,
						CatalogPlanID:      smbroker1.ServiceOfferings[0].Plans[1].CatalogID,
						PlatformBrokerName: brokerProxyName(smbroker1.Name, smbroker1.ID),
					},
				}, nil
			},
			smVisibilities: func() ([]*types.Visibility, error) {
				return []*types.Visibility{
					{
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
			platformVisibilities: func() ([]*platform.Visibility, error) {
				return []*platform.Visibility{
					{
						Public:             true,
						CatalogPlanID:      smbroker1.ServiceOfferings[0].Plans[1].CatalogID,
						PlatformBrokerName: brokerProxyName(smbroker1.Name, smbroker1.ID),
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
			stubs: func() {

			},
			platformVisibilities: func() ([]*platform.Visibility, error) {
				return nil, errors.New("Expected")
			},
			smVisibilities: func() ([]*types.Visibility, error) {
				return []*types.Visibility{
					{
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

	DescribeTable("Resync", func(t testCase) {
		setFakeBrokersClients()

		visibilities, smVisErr := t.smVisibilities()
		fakeSMClient.GetVisibilitiesReturns(visibilities, smVisErr)
		if t.smOfferings == nil {
			t.smOfferings = stubGetSMOfferings
		}

		serviceOfferings, smOfferingsErr := t.smOfferings()
		fakeSMClient.GetServiceOfferingsByBrokerIDsReturns(serviceOfferings, smOfferingsErr)
		fakeSMClient.GetPlansByServiceOfferingsStub = stubPlansByServices

		fakeVisibilityClient.GetVisibilitiesByBrokersReturns(t.platformVisibilities())

		fakeVisibilityClient.VisibilityScopeLabelKeyReturns("key")

		stubPlatformOpsToSucceed()

		if t.stubs != nil {
			t.stubs()
		}

		reconciler.Resyncer.Resync(context.TODO())

		Expect(fakeSMClient.GetBrokersCallCount()).To(Equal(1))
		expectedGetBrokersFromPlatformCallsCount := 1
		if smVisErr != nil || smOfferingsErr != nil {
			expectedGetBrokersFromPlatformCallsCount = 0
		}
		Expect(fakePlatformBrokerClient.GetBrokersCallCount()).To(Equal(expectedGetBrokersFromPlatformCallsCount))

		expected := t.expectations()

		Expect(fakeVisibilityClient.EnableAccessForPlanCallCount()).To(Equal(len(expected.enablePlanVisibilityCalledFor)))

		for index := range expected.enablePlanVisibilityCalledFor {
			_, request := fakeVisibilityClient.EnableAccessForPlanArgsForCall(index)
			checkAccessArguments(request.Labels, request.CatalogPlanID, request.BrokerName, expected.enablePlanVisibilityCalledFor)
		}

		Expect(fakeVisibilityClient.DisableAccessForPlanCallCount()).To(Equal(len(expected.disablePlanVisibilityCalledFor)))

		for index := range expected.disablePlanVisibilityCalledFor {
			_, request := fakeVisibilityClient.DisableAccessForPlanArgsForCall(index)
			checkAccessArguments(request.Labels, request.CatalogPlanID, request.BrokerName, expected.disablePlanVisibilityCalledFor)
		}

	}, entries...)

})
