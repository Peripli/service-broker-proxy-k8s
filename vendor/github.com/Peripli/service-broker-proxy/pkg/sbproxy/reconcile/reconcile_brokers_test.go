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
	"fmt"
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
)

var _ = Describe("Reconcile brokers", func() {
	const fakeAppHost = "https://smproxy.com"

	var (
		fakeSMClient *smfakes.FakeClient

		fakePlatformCatalogFetcher *platformfakes.FakeCatalogFetcher
		fakePlatformBrokerClient   *platformfakes.FakeBrokerClient

		waitGroup *sync.WaitGroup

		reconciliationTask *ReconciliationTask

		smbroker1 sm.Broker
		smbroker2 sm.Broker

		platformbroker1        platform.ServiceBroker
		platformbroker2        platform.ServiceBroker
		platformbrokerNonProxy platform.ServiceBroker
	)

	stubCreateBrokerToSucceed := func(ctx context.Context, r *platform.CreateServiceBrokerRequest) (*platform.ServiceBroker, error) {
		return &platform.ServiceBroker{
			GUID:      r.Name,
			Name:      r.Name,
			BrokerURL: r.BrokerURL,
		}, nil
	}

	stubCreateBrokerToReturnError := func(ctx context.Context, r *platform.CreateServiceBrokerRequest) (*platform.ServiceBroker, error) {
		return nil, fmt.Errorf("error")
	}

	stubPlatformOpsToSucceed := func() {
		fakePlatformBrokerClient.CreateBrokerStub = stubCreateBrokerToSucceed
		fakePlatformBrokerClient.DeleteBrokerReturns(nil)
		fakePlatformCatalogFetcher.FetchReturns(nil)
	}

	BeforeEach(func() {
		fakeSMClient = &smfakes.FakeClient{}
		fakePlatformClient := &platformfakes.FakeClient{}

		fakePlatformBrokerClient = &platformfakes.FakeBrokerClient{}
		fakePlatformCatalogFetcher = &platformfakes.FakeCatalogFetcher{}

		fakePlatformClient.BrokerReturns(fakePlatformBrokerClient)
		fakePlatformClient.CatalogFetcherReturns(fakePlatformCatalogFetcher)
		fakePlatformClient.VisibilityReturns(nil)

		visibilityCache := cache.New(5*time.Minute, 10*time.Minute)
		waitGroup = &sync.WaitGroup{}

		platformClient := struct {
			*platformfakes.FakeCatalogFetcher
			*platformfakes.FakeClient
		}{
			FakeCatalogFetcher: fakePlatformCatalogFetcher,
			FakeClient:         fakePlatformClient,
		}

		reconciliationTask = NewTask(
			context.TODO(), DefaultSettings(), waitGroup, platformClient, fakeSMClient,
			fakeAppHost, visibilityCache)

		smbroker1 = sm.Broker{
			ID:        "smBrokerID1",
			BrokerURL: "https://smBroker1.com",
			ServiceOfferings: []types.ServiceOffering{
				{
					ID:                  "smBroker1ServiceID1",
					Name:                "smBroker1Service1",
					Description:         "description",
					Bindable:            true,
					BindingsRetrievable: true,
					Plans: []*types.ServicePlan{
						{
							ID:          "smBroker1ServiceID1PlanID1",
							Name:        "smBroker1Service1Plan1",
							Description: "description",
						},
						{
							ID:          "smBroker1ServiceID1PlanID2",
							Name:        "smBroker1Service1Plan2",
							Description: "description",
						},
					},
				},
			},
		}

		smbroker2 = sm.Broker{
			ID:        "smBrokerID2",
			BrokerURL: "https://smBroker2.com",
			ServiceOfferings: []types.ServiceOffering{
				{
					ID:                  "smBroker2ServiceID1",
					Name:                "smBroker2Service1",
					Description:         "description",
					Bindable:            true,
					BindingsRetrievable: true,
					Plans: []*types.ServicePlan{
						{
							ID:          "smBroker2ServiceID1PlanID1",
							Name:        "smBroker2Service1Plan1",
							Description: "description",
						},
						{
							ID:          "smBroker2ServiceID1PlanID2",
							Name:        "smBroker2Service1Plan2",
							Description: "description",
						},
					},
				},
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
		reconcileCreateCalledFor  []platform.ServiceBroker
		reconcileDeleteCalledFor  []platform.ServiceBroker
		reconcileCatalogCalledFor []platform.ServiceBroker
	}

	type testCase struct {
		stubs           func()
		platformBrokers func() ([]platform.ServiceBroker, error)
		smBrokers       func() ([]sm.Broker, error)

		expectations func() expectations
	}

	entries := []TableEntry{
		Entry("When fetching brokers from SM fails no reconcilation should be done", testCase{
			stubs: func() {

			},
			platformBrokers: func() ([]platform.ServiceBroker, error) {
				return []platform.ServiceBroker{}, nil
			},
			smBrokers: func() ([]sm.Broker, error) {
				return nil, fmt.Errorf("error fetching brokers")
			},
			expectations: func() expectations {
				return expectations{
					reconcileCreateCalledFor:  []platform.ServiceBroker{},
					reconcileDeleteCalledFor:  []platform.ServiceBroker{},
					reconcileCatalogCalledFor: []platform.ServiceBroker{},
				}
			},
		}),

		Entry("When fetching brokers from platform fails no reconcilation should be done", testCase{
			stubs: func() {

			},
			platformBrokers: func() ([]platform.ServiceBroker, error) {
				return nil, fmt.Errorf("error fetching brokers")
			},
			smBrokers: func() ([]sm.Broker, error) {
				return []sm.Broker{}, nil
			},
			expectations: func() expectations {
				return expectations{
					reconcileCreateCalledFor:  []platform.ServiceBroker{},
					reconcileDeleteCalledFor:  []platform.ServiceBroker{},
					reconcileCatalogCalledFor: []platform.ServiceBroker{},
				}
			},
		}),

		Entry("When platform broker op fails reconcilation continues with the next broker", testCase{
			stubs: func() {
				fakePlatformBrokerClient.DeleteBrokerReturns(fmt.Errorf("error"))
				fakePlatformCatalogFetcher.FetchReturns(fmt.Errorf("error"))
				fakePlatformBrokerClient.CreateBrokerStub = stubCreateBrokerToReturnError
			},
			platformBrokers: func() ([]platform.ServiceBroker, error) {
				return []platform.ServiceBroker{
					platformbroker2,
				}, nil
			},
			smBrokers: func() ([]sm.Broker, error) {
				return []sm.Broker{
					smbroker1,
				}, nil
			},
			expectations: func() expectations {
				return expectations{
					reconcileCreateCalledFor: []platform.ServiceBroker{
						platformbroker1,
					},
					reconcileDeleteCalledFor: []platform.ServiceBroker{
						platformbroker2,
					},
					reconcileCatalogCalledFor: []platform.ServiceBroker{},
				}
			},
		}),

		Entry("When broker from SM has no catalog reconcilation continues with the next broker", testCase{
			stubs: func() {
				stubPlatformOpsToSucceed()
			},
			platformBrokers: func() ([]platform.ServiceBroker, error) {
				return []platform.ServiceBroker{
					platformbroker1,
					platformbroker2,
				}, nil
			},
			smBrokers: func() ([]sm.Broker, error) {
				smbroker1.ServiceOfferings = nil
				return []sm.Broker{
					smbroker1,
					smbroker2,
				}, nil
			},
			expectations: func() expectations {
				return expectations{
					reconcileCreateCalledFor: []platform.ServiceBroker{},
					reconcileDeleteCalledFor: []platform.ServiceBroker{},
					reconcileCatalogCalledFor: []platform.ServiceBroker{
						platformbroker1,
						platformbroker2,
					},
				}
			},
		}),

		Entry("When broker is in SM and is missing from platform it should be created", testCase{
			stubs: func() {
				stubPlatformOpsToSucceed()
			},
			platformBrokers: func() ([]platform.ServiceBroker, error) {
				return []platform.ServiceBroker{}, nil
			},
			smBrokers: func() ([]sm.Broker, error) {
				return []sm.Broker{
					smbroker1,
					smbroker2,
				}, nil
			},
			expectations: func() expectations {
				return expectations{
					reconcileCreateCalledFor: []platform.ServiceBroker{
						platformbroker1,
						platformbroker2,
					},
					reconcileDeleteCalledFor:  []platform.ServiceBroker{},
					reconcileCatalogCalledFor: []platform.ServiceBroker{},
				}
			},
		}),

		Entry("When broker is in SM and is also in platform it should be catalog refetched and access enabled", testCase{
			stubs: func() {
				stubPlatformOpsToSucceed()
			},
			platformBrokers: func() ([]platform.ServiceBroker, error) {
				return []platform.ServiceBroker{
					platformbroker1,
				}, nil
			},
			smBrokers: func() ([]sm.Broker, error) {
				return []sm.Broker{
					smbroker1,
				}, nil
			},
			expectations: func() expectations {
				return expectations{
					reconcileCreateCalledFor: []platform.ServiceBroker{},
					reconcileDeleteCalledFor: []platform.ServiceBroker{},
					reconcileCatalogCalledFor: []platform.ServiceBroker{
						platformbroker1,
					},
				}
			},
		}),

		Entry("When broker is missing from SM but is in platform it should be deleted", testCase{
			stubs: func() {
				stubPlatformOpsToSucceed()
			},
			platformBrokers: func() ([]platform.ServiceBroker, error) {
				return []platform.ServiceBroker{
					platformbroker1,
				}, nil
			},
			smBrokers: func() ([]sm.Broker, error) {
				return []sm.Broker{}, nil
			},
			expectations: func() expectations {
				return expectations{
					reconcileCreateCalledFor: []platform.ServiceBroker{},
					reconcileDeleteCalledFor: []platform.ServiceBroker{
						platformbroker1,
					},
					reconcileCatalogCalledFor: []platform.ServiceBroker{},
				}
			},
		}),

		Entry("When broker is missing from SM but is in platform that is not represented by the proxy should be ignored", testCase{
			stubs: func() {
				stubPlatformOpsToSucceed()
			},
			platformBrokers: func() ([]platform.ServiceBroker, error) {
				return []platform.ServiceBroker{
					platformbrokerNonProxy,
				}, nil
			},
			smBrokers: func() ([]sm.Broker, error) {
				return []sm.Broker{}, nil
			},
			expectations: func() expectations {
				return expectations{
					reconcileCreateCalledFor:  []platform.ServiceBroker{},
					reconcileDeleteCalledFor:  []platform.ServiceBroker{},
					reconcileCatalogCalledFor: []platform.ServiceBroker{},
				}
			},
		}),
	}

	DescribeTable("Run", func(t testCase) {
		smBrokers, err1 := t.smBrokers()
		platformBrokers, err2 := t.platformBrokers()

		fakeSMClient.GetBrokersReturns(smBrokers, err1)
		fakePlatformBrokerClient.GetBrokersReturns(platformBrokers, err2)
		t.stubs()

		reconciliationTask.Run()

		if err1 != nil {
			Expect(len(fakePlatformBrokerClient.Invocations())).To(Equal(1))
			Expect(len(fakePlatformCatalogFetcher.Invocations())).To(Equal(0))
			Expect(fakeSMClient.GetBrokersCallCount()).To(Equal(1))
			return
		}

		if err2 != nil {
			Expect(len(fakePlatformBrokerClient.Invocations())).To(Equal(1))
			Expect(len(fakePlatformCatalogFetcher.Invocations())).To(Equal(0))
			Expect(fakeSMClient.GetBrokersCallCount()).To(Equal(0))
			return
		}

		Expect(fakeSMClient.GetBrokersCallCount()).To(Equal(1))
		Expect(fakePlatformBrokerClient.GetBrokersCallCount()).To(Equal(1))

		expected := t.expectations()
		Expect(fakePlatformBrokerClient.CreateBrokerCallCount()).To(Equal(len(expected.reconcileCreateCalledFor)))
		for index, broker := range expected.reconcileCreateCalledFor {
			_, request := fakePlatformBrokerClient.CreateBrokerArgsForCall(index)
			Expect(request).To(Equal(&platform.CreateServiceBrokerRequest{
				Name:      broker.Name,
				BrokerURL: broker.BrokerURL,
			}))
		}

		Expect(fakePlatformCatalogFetcher.FetchCallCount()).To(Equal(len(expected.reconcileCatalogCalledFor)))
		for index, broker := range expected.reconcileCatalogCalledFor {
			_, serviceBroker := fakePlatformCatalogFetcher.FetchArgsForCall(index)
			Expect(serviceBroker).To(Equal(&broker))
		}

		Expect(fakePlatformBrokerClient.DeleteBrokerCallCount()).To(Equal(len(expected.reconcileDeleteCalledFor)))
		for index, broker := range expected.reconcileDeleteCalledFor {
			_, request := fakePlatformBrokerClient.DeleteBrokerArgsForCall(index)
			Expect(request).To(Equal(&platform.DeleteServiceBrokerRequest{
				GUID: broker.GUID,
				Name: broker.Name,
			}))
		}
	}, entries...)

})
