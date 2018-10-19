/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"errors"
	"reflect"
	"testing"
	"time"

	osb "github.com/pmorie/go-open-service-broker-client/v2"
	fakeosb "github.com/pmorie/go-open-service-broker-client/v2/fake"

	"github.com/kubernetes-incubator/service-catalog/pkg/apis/servicecatalog/v1beta1"
	"github.com/kubernetes-incubator/service-catalog/test/fake"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/diff"

	"strings"

	corev1 "k8s.io/api/core/v1"
	clientgotesting "k8s.io/client-go/testing"
)

// TestShouldReconcileClusterServiceBroker ensures that with the expected conditions the
// reconciler is reported as needing to run.
//
// The test cases are proving:
// - broker without ready condition will reconcile
// - broker with deletion timestamp set will reconcile
// - broker without ready condition, with status will reconcile
// - broker without ready condition, without status will reconcile
// - broker with status/ready, past relist interval will reconcile
// - broker with status/ready, within relist interval will NOT reconcile
// - broker with status/ready/checksum, will reconcile
func TestShouldReconcileClusterServiceBroker(t *testing.T) {
	// Anonymous struct fields:
	// name: short description of the test
	// broker: broker object to test
	// now: what time the interval is calculated with respect to interval
	// reconcile: whether or not the reconciler should run, the return of
	// shouldReconcileClusterServiceBroker
	cases := []struct {
		name      string
		broker    *v1beta1.ClusterServiceBroker
		now       time.Time
		reconcile bool
		err       error
	}{
		{
			name: "no status",
			broker: func() *v1beta1.ClusterServiceBroker {
				broker := getTestClusterServiceBroker()
				broker.Spec.RelistDuration = &metav1.Duration{Duration: 3 * time.Minute}
				return broker
			}(),
			now:       time.Now(),
			reconcile: true,
		},
		{
			name: "deletionTimestamp set",
			broker: func() *v1beta1.ClusterServiceBroker {
				broker := getTestClusterServiceBrokerWithStatus(v1beta1.ConditionTrue)
				broker.DeletionTimestamp = &metav1.Time{}
				broker.Spec.RelistDuration = &metav1.Duration{Duration: 3 * time.Hour}
				return broker
			}(),
			now:       time.Now(),
			reconcile: true,
		},
		{
			name: "no ready condition",
			broker: func() *v1beta1.ClusterServiceBroker {
				broker := getTestClusterServiceBroker()
				broker.Status = v1beta1.ClusterServiceBrokerStatus{
					CommonServiceBrokerStatus: v1beta1.CommonServiceBrokerStatus{
						Conditions: []v1beta1.ServiceBrokerCondition{
							{
								Type:   v1beta1.ServiceBrokerConditionType("NotARealCondition"),
								Status: v1beta1.ConditionTrue,
							},
						},
					},
				}
				broker.Spec.RelistDuration = &metav1.Duration{Duration: 3 * time.Minute}
				return broker
			}(),
			now:       time.Now(),
			reconcile: true,
		},
		{
			name: "not ready",
			broker: func() *v1beta1.ClusterServiceBroker {
				broker := getTestClusterServiceBrokerWithStatus(v1beta1.ConditionFalse)
				broker.Spec.RelistDuration = &metav1.Duration{Duration: 3 * time.Minute}
				return broker
			}(),
			now:       time.Now(),
			reconcile: true,
		},
		{
			name: "ready, interval elapsed",
			broker: func() *v1beta1.ClusterServiceBroker {
				broker := getTestClusterServiceBrokerWithStatus(v1beta1.ConditionTrue)
				broker.Spec.RelistDuration = &metav1.Duration{Duration: 3 * time.Minute}
				return broker
			}(),
			now:       time.Now(),
			reconcile: true,
		},
		{
			name: "good steady state - ready, interval not elapsed, but last state change was a long time ago",
			broker: func() *v1beta1.ClusterServiceBroker {
				lastTransitionTime := metav1.NewTime(time.Now().Add(-30 * time.Minute))
				lastRelistTime := metav1.NewTime(time.Now().Add(-2 * time.Minute))
				broker := getTestClusterServiceBrokerWithStatusAndTime(v1beta1.ConditionTrue, lastTransitionTime, lastRelistTime)
				broker.Spec.RelistDuration = &metav1.Duration{Duration: 3 * time.Minute}
				return broker
			}(),
			now:       time.Now(),
			reconcile: false,
		},
		{
			name: "good steady state - ready, interval has elapsed, last state change was a long time ago",
			broker: func() *v1beta1.ClusterServiceBroker {
				lastTransitionTime := metav1.NewTime(time.Now().Add(-30 * time.Minute))
				lastRelistTime := metav1.NewTime(time.Now().Add(-4 * time.Minute))
				broker := getTestClusterServiceBrokerWithStatusAndTime(v1beta1.ConditionTrue, lastTransitionTime, lastRelistTime)
				broker.Spec.RelistDuration = &metav1.Duration{Duration: 3 * time.Minute}
				return broker
			}(),
			now:       time.Now(),
			reconcile: true,
		},
		{
			name: "ready, interval not elapsed",
			broker: func() *v1beta1.ClusterServiceBroker {
				broker := getTestClusterServiceBrokerWithStatus(v1beta1.ConditionTrue)
				broker.Spec.RelistDuration = &metav1.Duration{Duration: 3 * time.Hour}
				return broker
			}(),
			now:       time.Now(),
			reconcile: false,
		},
		{
			name: "ready, interval not elapsed, spec changed",
			broker: func() *v1beta1.ClusterServiceBroker {
				broker := getTestClusterServiceBrokerWithStatus(v1beta1.ConditionTrue)
				broker.Generation = 2
				broker.Status.ReconciledGeneration = 1
				broker.Spec.RelistDuration = &metav1.Duration{Duration: 3 * time.Hour}
				return broker
			}(),
			now:       time.Now(),
			reconcile: true,
		},
		{
			name: "ready, duration behavior, nil duration, interval not elapsed",
			broker: func() *v1beta1.ClusterServiceBroker {
				t := metav1.NewTime(time.Now().Add(-23 * time.Hour))
				broker := getTestClusterServiceBrokerWithStatusAndTime(v1beta1.ConditionTrue, t, t)
				broker.Spec.RelistBehavior = v1beta1.ServiceBrokerRelistBehaviorDuration
				broker.Spec.RelistDuration = nil
				return broker
			}(),
			now:       time.Now(),
			reconcile: false,
		},
		{
			name: "ready, duration behavior, nil duration, interval elapsed",
			broker: func() *v1beta1.ClusterServiceBroker {
				t := metav1.NewTime(time.Now().Add(-25 * time.Hour))
				broker := getTestClusterServiceBrokerWithStatusAndTime(v1beta1.ConditionTrue, t, t)
				broker.Spec.RelistBehavior = v1beta1.ServiceBrokerRelistBehaviorDuration
				broker.Spec.RelistDuration = nil
				return broker
			}(),
			now:       time.Now(),
			reconcile: true,
		},
		{
			name: "ready, manual behavior",
			broker: func() *v1beta1.ClusterServiceBroker {
				broker := getTestClusterServiceBrokerWithStatus(v1beta1.ConditionTrue)
				broker.Spec.RelistBehavior = v1beta1.ServiceBrokerRelistBehaviorManual
				return broker
			}(),
			now:       time.Now(),
			reconcile: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var ltt *time.Time
			if len(tc.broker.Status.Conditions) != 0 {
				ltt = &tc.broker.Status.Conditions[0].LastTransitionTime.Time
			}

			if tc.broker.Spec.RelistDuration != nil {
				interval := tc.broker.Spec.RelistDuration.Duration
				lastRelistTime := tc.broker.Status.LastCatalogRetrievalTime
				t.Logf("now: %v, interval: %v, last transition time: %v, last relist time: %v", tc.now, interval, ltt, lastRelistTime)
			} else {
				t.Logf("broker.Spec.RelistDuration set to nil")
			}

			actual := shouldReconcileClusterServiceBroker(tc.broker, tc.now, 24*time.Hour)

			if e, a := tc.reconcile, actual; e != a {
				t.Errorf("unexpected result: %s", expectedGot(e, a))
			}
		})
	}
}

// TestReconcileClusterServiceBrokerExistingServiceClassAndServicePlan
// verifies a simple, successful run of reconcileClusterServiceBroker() when a
// ClusterServiceClass and plan already exist.  This test will cause
// reconcileBroker() to fetch the catalog from the ClusterServiceBroker,
// create a Service Class for the single service that it lists and reconcile
// the service class ensuring the name and id of the relisted service matches
// the existing entry and updates the service catalog. There will be two
// additional reconciles of plans before the final broker update
func TestReconcileClusterServiceBrokerExistingServiceClassAndServicePlan(t *testing.T) {
	fakeKubeClient, fakeCatalogClient, fakeClusterServiceBrokerClient, testController, sharedInformers := newTestController(t, getTestCatalogConfig())

	testClusterServiceClass := getTestClusterServiceClass()
	testClusterServicePlan := getTestClusterServicePlan()
	testClusterServicePlanNonbindable := getTestClusterServicePlanNonbindable()
	sharedInformers.ClusterServiceClasses().Informer().GetStore().Add(testClusterServiceClass)

	fakeCatalogClient.AddReactor("list", "clusterserviceclasses", func(action clientgotesting.Action) (bool, runtime.Object, error) {
		return true, &v1beta1.ClusterServiceClassList{
			Items: []v1beta1.ClusterServiceClass{
				*testClusterServiceClass,
			},
		}, nil
	})

	if err := reconcileClusterServiceBroker(t, testController, getTestClusterServiceBroker()); err != nil {
		t.Fatalf("This should not fail: %v", err)
	}

	brokerActions := fakeClusterServiceBrokerClient.Actions()
	assertNumberOfBrokerActions(t, brokerActions, 1)
	assertGetCatalog(t, brokerActions[0])

	listRestrictions := clientgotesting.ListRestrictions{
		Labels: labels.Everything(),
		Fields: fields.OneTermEqualSelector("spec.clusterServiceBrokerName", "test-clusterservicebroker"),
	}

	actions := fakeCatalogClient.Actions()
	assertNumberOfActions(t, actions, 6)
	assertList(t, actions[0], &v1beta1.ClusterServiceClass{}, listRestrictions)
	assertList(t, actions[1], &v1beta1.ClusterServicePlan{}, listRestrictions)
	assertUpdate(t, actions[2], testClusterServiceClass)
	assertCreate(t, actions[3], testClusterServicePlan)
	assertCreate(t, actions[4], testClusterServicePlanNonbindable)

	// 4 update action for broker status subresource
	updatedClusterServiceBroker := assertUpdateStatus(t, actions[5], getTestClusterServiceBroker())
	assertClusterServiceBrokerReadyTrue(t, updatedClusterServiceBroker)

	// verify no kube resources created
	kubeActions := fakeKubeClient.Actions()
	assertNumberOfActions(t, kubeActions, 0)
}

func TestReconcileClusterServiceBrokerRemovedClusterServiceClass(t *testing.T) {
	fakeKubeClient, fakeCatalogClient, fakeClusterServiceBrokerClient, testController, sharedInformers := newTestController(t, getTestCatalogConfig())

	testClusterServiceClass := getTestClusterServiceClass()
	testRemovedClusterServiceClass := getTestRemovedClusterServiceClass()
	testClusterServicePlan := getTestClusterServicePlan()
	testClusterServicePlanNonbindable := getTestClusterServicePlanNonbindable()
	sharedInformers.ClusterServiceClasses().Informer().GetStore().Add(testClusterServiceClass)
	sharedInformers.ClusterServiceClasses().Informer().GetStore().Add(testRemovedClusterServiceClass)

	fakeCatalogClient.AddReactor("list", "clusterserviceclasses", func(action clientgotesting.Action) (bool, runtime.Object, error) {
		return true, &v1beta1.ClusterServiceClassList{
			Items: []v1beta1.ClusterServiceClass{
				*testClusterServiceClass,
				*testRemovedClusterServiceClass,
			},
		}, nil
	})

	if err := reconcileClusterServiceBroker(t, testController, getTestClusterServiceBroker()); err != nil {
		t.Fatalf("This should not fail: %v", err)
	}

	brokerActions := fakeClusterServiceBrokerClient.Actions()
	assertNumberOfBrokerActions(t, brokerActions, 1)
	assertGetCatalog(t, brokerActions[0])

	listRestrictions := clientgotesting.ListRestrictions{
		Labels: labels.Everything(),
		Fields: fields.OneTermEqualSelector("spec.clusterServiceBrokerName", "test-clusterservicebroker"),
	}

	actions := fakeCatalogClient.Actions()
	assertNumberOfActions(t, actions, 7)
	assertList(t, actions[0], &v1beta1.ClusterServiceClass{}, listRestrictions)
	assertList(t, actions[1], &v1beta1.ClusterServicePlan{}, listRestrictions)
	assertUpdate(t, actions[2], testClusterServiceClass)
	assertUpdateStatus(t, actions[3], testRemovedClusterServiceClass)
	assertCreate(t, actions[4], testClusterServicePlan)
	assertCreate(t, actions[5], testClusterServicePlanNonbindable)

	updatedClusterServiceBroker := assertUpdateStatus(t, actions[6], getTestClusterServiceBroker())
	assertClusterServiceBrokerReadyTrue(t, updatedClusterServiceBroker)

	// verify no kube resources created
	kubeActions := fakeKubeClient.Actions()
	assertNumberOfActions(t, kubeActions, 0)
}

// TestReconcileClusterServiceBrokerRemovedAndRestoredClusterServiceClass
// validates where Service Catalog has a class and plan that is marked as
// RemovedFromBrokerCatalog but then the ServiceBroker adds the class and plan
// back into its getCatalog response.  This should result in the class's status
// and plan's status being updated resetting the RemovedFromBrokerCatalog to
// false.
func TestReconcileClusterServiceBrokerRemovedAndRestoredClusterServiceClass(t *testing.T) {
	fakeKubeClient, fakeCatalogClient, fakeClusterServiceBrokerClient, testController, sharedInformers := newTestController(t, getTestCatalogConfig())

	testClusterServiceClass := getTestClusterServiceClass()
	testClusterServicePlan := getTestClusterServicePlan()
	testClusterServicePlan.Status.RemovedFromBrokerCatalog = true
	testClusterServicePlanNonbindable := getTestClusterServicePlanNonbindable()
	testClusterServiceClass.Status.RemovedFromBrokerCatalog = true
	sharedInformers.ClusterServiceClasses().Informer().GetStore().Add(testClusterServiceClass)
	sharedInformers.ClusterServicePlans().Informer().GetStore().Add(testClusterServicePlan)

	fakeCatalogClient.AddReactor("list", "clusterserviceclasses", func(action clientgotesting.Action) (bool, runtime.Object, error) {
		return true, &v1beta1.ClusterServiceClassList{
			Items: []v1beta1.ClusterServiceClass{
				*testClusterServiceClass,
			},
		}, nil
	})
	fakeCatalogClient.AddReactor("list", "clusterserviceplans", func(action clientgotesting.Action) (bool, runtime.Object, error) {
		return true, &v1beta1.ClusterServicePlanList{
			Items: []v1beta1.ClusterServicePlan{
				*testClusterServicePlan,
			},
		}, nil
	})
	fakeCatalogClient.AddReactor("update", "clusterserviceclasses", func(action clientgotesting.Action) (bool, runtime.Object, error) {
		return true, testClusterServiceClass, nil
	})
	fakeCatalogClient.AddReactor("update", "clusterserviceplans", func(action clientgotesting.Action) (bool, runtime.Object, error) {
		return true, testClusterServicePlan, nil
	})

	if err := reconcileClusterServiceBroker(t, testController, getTestClusterServiceBroker()); err != nil {
		t.Fatalf("This should not fail: %v", err)
	}

	brokerActions := fakeClusterServiceBrokerClient.Actions()
	assertNumberOfBrokerActions(t, brokerActions, 1)
	assertGetCatalog(t, brokerActions[0])

	listRestrictions := clientgotesting.ListRestrictions{
		Labels: labels.Everything(),
		Fields: fields.OneTermEqualSelector("spec.clusterServiceBrokerName", "test-clusterservicebroker"),
	}

	actions := fakeCatalogClient.Actions()
	assertNumberOfActions(t, actions, 8)
	assertList(t, actions[0], &v1beta1.ClusterServiceClass{}, listRestrictions)
	assertList(t, actions[1], &v1beta1.ClusterServicePlan{}, listRestrictions)
	assertUpdate(t, actions[2], testClusterServiceClass)
	class := assertUpdateStatus(t, actions[3], testClusterServiceClass)
	assertClassRemovedFromBrokerCatalogFalse(t, class)
	assertUpdate(t, actions[4], testClusterServicePlan)
	plan := assertUpdateStatus(t, actions[5], testClusterServicePlan)
	assertPlanRemovedFromBrokerCatalogFalse(t, plan)
	assertCreate(t, actions[6], testClusterServicePlanNonbindable)
	updatedClusterServiceBroker := assertUpdateStatus(t, actions[7], getTestClusterServiceBroker())
	assertClusterServiceBrokerReadyTrue(t, updatedClusterServiceBroker)

	// verify no kube resources created
	kubeActions := fakeKubeClient.Actions()
	assertNumberOfActions(t, kubeActions, 0)
}

func TestReconcileClusterServiceBrokerRemovedClusterServicePlan(t *testing.T) {
	fakeKubeClient, fakeCatalogClient, fakeClusterServiceBrokerClient, testController, sharedInformers := newTestController(t, getTestCatalogConfig())

	testClusterServiceClass := getTestClusterServiceClass()
	testClusterServicePlan := getTestClusterServicePlan()
	testClusterServicePlanNonbindable := getTestClusterServicePlanNonbindable()
	testRemovedClusterServicePlan := getTestRemovedClusterServicePlan()
	sharedInformers.ClusterServiceClasses().Informer().GetStore().Add(testClusterServiceClass)
	sharedInformers.ClusterServicePlans().Informer().GetStore().Add(testRemovedClusterServicePlan)

	fakeCatalogClient.AddReactor("list", "clusterserviceclasses", func(action clientgotesting.Action) (bool, runtime.Object, error) {
		return true, &v1beta1.ClusterServiceClassList{
			Items: []v1beta1.ClusterServiceClass{
				*testClusterServiceClass,
			},
		}, nil
	})
	fakeCatalogClient.AddReactor("list", "clusterserviceplans", func(action clientgotesting.Action) (bool, runtime.Object, error) {
		return true, &v1beta1.ClusterServicePlanList{
			Items: []v1beta1.ClusterServicePlan{
				*testRemovedClusterServicePlan,
			},
		}, nil
	})

	if err := reconcileClusterServiceBroker(t, testController, getTestClusterServiceBroker()); err != nil {
		t.Fatalf("This should not fail: %v", err)
	}

	brokerActions := fakeClusterServiceBrokerClient.Actions()
	assertNumberOfBrokerActions(t, brokerActions, 1)
	assertGetCatalog(t, brokerActions[0])

	listRestrictions := clientgotesting.ListRestrictions{
		Labels: labels.Everything(),
		Fields: fields.OneTermEqualSelector("spec.clusterServiceBrokerName", "test-clusterservicebroker"),
	}

	actions := fakeCatalogClient.Actions()
	assertNumberOfActions(t, actions, 7)
	assertList(t, actions[0], &v1beta1.ClusterServiceClass{}, listRestrictions)
	assertList(t, actions[1], &v1beta1.ClusterServicePlan{}, listRestrictions)
	assertUpdate(t, actions[2], testClusterServiceClass)
	assertCreate(t, actions[3], testClusterServicePlan)
	assertCreate(t, actions[4], testClusterServicePlanNonbindable)
	assertUpdateStatus(t, actions[5], testRemovedClusterServicePlan)

	updatedClusterServiceBroker := assertUpdateStatus(t, actions[6], getTestClusterServiceBroker())
	assertClusterServiceBrokerReadyTrue(t, updatedClusterServiceBroker)

	// verify no kube resources created
	kubeActions := fakeKubeClient.Actions()
	assertNumberOfActions(t, kubeActions, 0)
}

// TestReconcileClusterServiceBrokerExistingClusterServiceClassDifferentBroker simulates catalog
// refresh where broker lists a service which matches an existing, already
// cataloged service but the service points to a different ClusterServiceBroker.  Results in an error.
func TestReconcileClusterServiceBrokerExistingClusterServiceClassDifferentBroker(t *testing.T) {
	fakeKubeClient, fakeCatalogClient, fakeClusterServiceBrokerClient, testController, sharedInformers := newTestController(t, getTestCatalogConfig())

	testClusterServiceClass := getTestClusterServiceClass()
	testClusterServiceClass.Spec.ClusterServiceBrokerName = "notTheSame"

	sharedInformers.ClusterServiceClasses().Informer().GetStore().Add(testClusterServiceClass)

	if err := reconcileClusterServiceBroker(t, testController, getTestClusterServiceBroker()); err == nil {
		t.Fatal("The same service class should not belong to two different brokers.")
	}

	brokerActions := fakeClusterServiceBrokerClient.Actions()
	assertNumberOfBrokerActions(t, brokerActions, 1)
	assertGetCatalog(t, brokerActions[0])

	actions := fakeCatalogClient.Actions()
	assertNumberOfActions(t, actions, 3)

	listRestrictions := clientgotesting.ListRestrictions{
		Labels: labels.Everything(),
		Fields: fields.OneTermEqualSelector("spec.clusterServiceBrokerName", "test-clusterservicebroker"),
	}
	assertList(t, actions[0], &v1beta1.ClusterServiceClass{}, listRestrictions)
	assertList(t, actions[1], &v1beta1.ClusterServicePlan{}, listRestrictions)
	updatedClusterServiceBroker := assertUpdateStatus(t, actions[2], getTestClusterServiceBroker())
	assertClusterServiceBrokerReadyFalse(t, updatedClusterServiceBroker)

	// verify no kube resources created
	kubeActions := fakeKubeClient.Actions()
	assertNumberOfActions(t, kubeActions, 0)

	events := getRecordedEvents(testController)

	expectedEvent := warningEventBuilder(errorSyncingCatalogReason).msgf(
		"Error reconciling ClusterServiceClass (K8S: %q ExternalName: %q) (broker %q):",
		testClusterServiceClassGUID, testClusterServiceClassName, testClusterServiceBrokerName,
	).msgf(
		"ClusterServiceClass (K8S: %q ExternalName: %q) already exists for Broker %q",
		testClusterServiceClassGUID, testClusterServiceClassName, "notTheSame",
	)
	if err := checkEvents(events, expectedEvent.stringArr()); err != nil {
		t.Fatal(err)
	}
}

// TestReconcileClusterServiceBrokerExistingClusterServicePlanDifferentClass simulates catalog
// refresh where broker lists a service plan which matches an existing, already
// cataloged service plan but the plan points to a different ClusterServiceClass.  Results in an error.
func TestReconcileClusterServiceBrokerExistingClusterServicePlanDifferentClass(t *testing.T) {
	fakeKubeClient, fakeCatalogClient, fakeClusterServiceBrokerClient, testController, sharedInformers := newTestController(t, getTestCatalogConfig())

	testClusterServicePlan := getTestClusterServicePlan()
	testClusterServicePlan.Spec.ClusterServiceBrokerName = "notTheSame"
	testClusterServicePlan.Spec.ClusterServiceClassRef = v1beta1.ClusterObjectReference{
		Name: "notTheSameClass",
	}

	sharedInformers.ClusterServicePlans().Informer().GetStore().Add(testClusterServicePlan)

	if err := reconcileClusterServiceBroker(t, testController, getTestClusterServiceBroker()); err == nil {
		t.Fatal("The same service class should not belong to two different brokers.")
	}

	brokerActions := fakeClusterServiceBrokerClient.Actions()
	assertNumberOfBrokerActions(t, brokerActions, 1)
	assertGetCatalog(t, brokerActions[0])

	actions := fakeCatalogClient.Actions()
	assertNumberOfActions(t, actions, 4)

	listRestrictions := clientgotesting.ListRestrictions{
		Labels: labels.Everything(),
		Fields: fields.OneTermEqualSelector("spec.clusterServiceBrokerName", "test-clusterservicebroker"),
	}
	assertList(t, actions[0], &v1beta1.ClusterServiceClass{}, listRestrictions)
	assertList(t, actions[1], &v1beta1.ClusterServicePlan{}, listRestrictions)
	assertCreate(t, actions[2], getTestClusterServiceClass())
	updatedClusterServiceBroker := assertUpdateStatus(t, actions[3], getTestClusterServiceBroker())
	assertClusterServiceBrokerReadyFalse(t, updatedClusterServiceBroker)

	// verify no kube resources created
	kubeActions := fakeKubeClient.Actions()
	assertNumberOfActions(t, kubeActions, 0)

	events := getRecordedEvents(testController)

	expectedEvent := warningEventBuilder(errorSyncingCatalogReason).msgf(
		"Error reconciling ClusterServicePlan (K8S: %q ExternalName: %q):",
		testClusterServicePlanGUID, testClusterServicePlanName,
	).msgf(
		"ClusterServicePlan (K8S: %q ExternalName: %q) already exists for Broker %q",
		testClusterServicePlanGUID, testClusterServicePlanName, "notTheSame",
	)
	if err := checkEvents(events, expectedEvent.stringArr()); err != nil {
		t.Fatal(err)
	}
}

// TestReconcileClusterServiceBrokerDelete simulates a broker reconciliation where broker was marked for deletion.
// Results in service class and broker both being deleted.
func TestReconcileClusterServiceBrokerDelete(t *testing.T) {
	fakeKubeClient, fakeCatalogClient, fakeClusterServiceBrokerClient, testController, _ := newTestController(t, getTestCatalogConfig())

	testClusterServiceClass := getTestClusterServiceClass()
	testClusterServicePlan := getTestClusterServicePlan()

	broker := getTestClusterServiceBroker()
	broker.DeletionTimestamp = &metav1.Time{}
	broker.Finalizers = []string{v1beta1.FinalizerServiceCatalog}
	fakeCatalogClient.AddReactor("get", "clusterservicebrokers", func(action clientgotesting.Action) (bool, runtime.Object, error) {
		return true, broker, nil
	})
	fakeCatalogClient.AddReactor("list", "clusterserviceclasses", func(action clientgotesting.Action) (bool, runtime.Object, error) {
		return true, &v1beta1.ClusterServiceClassList{
			Items: []v1beta1.ClusterServiceClass{
				*testClusterServiceClass,
			},
		}, nil
	})
	fakeCatalogClient.AddReactor("list", "clusterserviceplans", func(action clientgotesting.Action) (bool, runtime.Object, error) {
		return true, &v1beta1.ClusterServicePlanList{
			Items: []v1beta1.ClusterServicePlan{
				*testClusterServicePlan,
			},
		}, nil
	})

	err := reconcileClusterServiceBroker(t, testController, broker)
	if err != nil {
		t.Fatalf("This should not fail : %v", err)
	}

	brokerActions := fakeClusterServiceBrokerClient.Actions()
	assertNumberOfBrokerActions(t, brokerActions, 0)

	// Verify no core kube actions occurred
	kubeActions := fakeKubeClient.Actions()
	assertNumberOfActions(t, kubeActions, 0)

	actions := fakeCatalogClient.Actions()
	// The four actions should be:
	// - list serviceplans
	// - delete serviceplans
	// - list serviceclasses
	// - delete serviceclass
	// - update the ready condition
	// - get the broker
	// - remove the finalizer
	assertNumberOfActions(t, actions, 7)

	listRestrictions := clientgotesting.ListRestrictions{
		Labels: labels.Everything(),
		Fields: fields.OneTermEqualSelector("spec.clusterServiceBrokerName", broker.Name),
	}
	assertList(t, actions[0], &v1beta1.ClusterServiceClass{}, listRestrictions)
	assertList(t, actions[1], &v1beta1.ClusterServicePlan{}, listRestrictions)
	assertDelete(t, actions[2], testClusterServicePlan)
	assertDelete(t, actions[3], testClusterServiceClass)
	updatedClusterServiceBroker := assertUpdateStatus(t, actions[4], broker)
	assertClusterServiceBrokerReadyFalse(t, updatedClusterServiceBroker)

	assertGet(t, actions[5], broker)

	updatedClusterServiceBroker = assertUpdateStatus(t, actions[6], broker)
	assertEmptyFinalizers(t, updatedClusterServiceBroker)

	events := getRecordedEvents(testController)

	expectedEvent := normalEventBuilder(successClusterServiceBrokerDeletedReason).msg(
		"The broker test-clusterservicebroker was deleted successfully.",
	)
	if err := checkEvents(events, expectedEvent.stringArr()); err != nil {
		t.Fatal(err)
	}
}

// TestReconcileClusterServiceBrokerErrorFetchingCatalog simulates broker reconciliation where
// OSB client responds with an error for getting the catalog which in turn causes
// reconcileClusterServiceBroker() to return an error.
func TestReconcileClusterServiceBrokerErrorFetchingCatalog(t *testing.T) {
	fakeKubeClient, fakeCatalogClient, fakeClusterServiceBrokerClient, testController, _ := newTestController(t, fakeosb.FakeClientConfiguration{
		CatalogReaction: &fakeosb.CatalogReaction{
			Error: errors.New("ooops"),
		},
	})

	broker := getTestClusterServiceBroker()

	if err := reconcileClusterServiceBroker(t, testController, broker); err == nil {
		t.Fatal("Should have failed to get the catalog.")
	}

	brokerActions := fakeClusterServiceBrokerClient.Actions()
	assertNumberOfBrokerActions(t, brokerActions, 1)
	assertGetCatalog(t, brokerActions[0])

	actions := fakeCatalogClient.Actions()
	assertNumberOfActions(t, actions, 2)

	updatedClusterServiceBroker := assertUpdateStatus(t, actions[0], broker)
	assertClusterServiceBrokerReadyFalse(t, updatedClusterServiceBroker)

	updatedClusterServiceBroker = assertUpdateStatus(t, actions[1], broker)
	assertClusterServiceBrokerOperationStartTimeSet(t, updatedClusterServiceBroker, true)

	assertNumberOfActions(t, fakeKubeClient.Actions(), 0)

	events := getRecordedEvents(testController)

	expectedEvent := warningEventBuilder(errorFetchingCatalogReason).msg("Error getting broker catalog:").msg("ooops")
	if err := checkEvents(events, expectedEvent.stringArr()); err != nil {
		t.Fatal(err)
	}
}

// TestReconcileClusterServiceBrokerZeroServices simulates broker reconciliation where
// OSB client responds with zero services which is valid
func TestReconcileClusterServiceBrokerZeroServices(t *testing.T) {
	fakeKubeClient, fakeCatalogClient, fakeClusterServiceBrokerClient, testController, _ := newTestController(t, fakeosb.FakeClientConfiguration{
		CatalogReaction: &fakeosb.CatalogReaction{
			Response: &osb.CatalogResponse{},
		},
	})

	// Broker's response to getCatalog is empty, there are no existing classes or plans,
	// reconcile will allow the empty services and just update the broker status

	broker := getTestClusterServiceBroker()

	fakeCatalogClient.AddReactor("list", "clusterserviceclasses", func(action clientgotesting.Action) (bool, runtime.Object, error) {
		return true, &v1beta1.ClusterServiceClassList{
			Items: []v1beta1.ClusterServiceClass{},
		}, nil
	})
	fakeCatalogClient.AddReactor("list", "clusterserviceplans", func(action clientgotesting.Action) (bool, runtime.Object, error) {
		return true, &v1beta1.ClusterServicePlanList{
			Items: []v1beta1.ClusterServicePlan{},
		}, nil
	})

	err := reconcileClusterServiceBroker(t, testController, broker)
	if err != nil {
		t.Fatalf("This should not fail : %v", err)
	}

	brokerActions := fakeClusterServiceBrokerClient.Actions()
	assertNumberOfBrokerActions(t, brokerActions, 1)
	assertGetCatalog(t, brokerActions[0])

	// Verify no core kube actions occurred
	kubeActions := fakeKubeClient.Actions()
	assertNumberOfActions(t, kubeActions, 0)

	actions := fakeCatalogClient.Actions()
	// The four actions should be:
	// - list serviceplans
	// - list serviceclasses
	// - update the broker status
	assertNumberOfActions(t, actions, 3)

	listRestrictions := clientgotesting.ListRestrictions{
		Labels: labels.Everything(),
		Fields: fields.OneTermEqualSelector("spec.clusterServiceBrokerName", broker.Name),
	}
	assertList(t, actions[0], &v1beta1.ClusterServiceClass{}, listRestrictions)
	assertList(t, actions[1], &v1beta1.ClusterServicePlan{}, listRestrictions)
	updatedClusterServiceBroker := assertUpdateStatus(t, actions[2], broker)
	assertClusterServiceBrokerReadyTrue(t, updatedClusterServiceBroker)

	events := getRecordedEvents(testController)
	expectedEvent := corev1.EventTypeNormal + " " + successFetchedCatalogReason + " " + successFetchedCatalogMessage
	if e, a := expectedEvent, events[0]; !strings.HasPrefix(a, e) {
		t.Fatalf("Received unexpected event, %s", expectedGot(e, a))
	}
}

func TestReconcileClusterServiceBrokerWithAuth(t *testing.T) {
	basicAuthInfo := &v1beta1.ClusterServiceBrokerAuthInfo{
		Basic: &v1beta1.ClusterBasicAuthConfig{
			SecretRef: &v1beta1.ObjectReference{
				Namespace: "test-ns",
				Name:      "auth-secret",
			},
		},
	}
	bearerAuthInfo := &v1beta1.ClusterServiceBrokerAuthInfo{
		Bearer: &v1beta1.ClusterBearerTokenAuthConfig{
			SecretRef: &v1beta1.ObjectReference{
				Namespace: "test-ns",
				Name:      "auth-secret",
			},
		},
	}
	basicAuthSecret := &corev1.Secret{
		Data: map[string][]byte{
			v1beta1.BasicAuthUsernameKey: []byte("foo"),
			v1beta1.BasicAuthPasswordKey: []byte("bar"),
		},
	}
	bearerAuthSecret := &corev1.Secret{
		Data: map[string][]byte{
			v1beta1.BearerTokenKey: []byte("token"),
		},
	}

	// The test cases here are testing the correctness of authentication with broker
	//
	// Anonymous struct fields:
	// name: short description of the test
	// authInfo: broker auth configuration
	// secret: auth secret to be returned upon request from Service Catalog
	// shouldSucceed: whether authentication should succeed
	cases := []struct {
		name          string
		authInfo      *v1beta1.ClusterServiceBrokerAuthInfo
		secret        *corev1.Secret
		shouldSucceed bool
	}{
		{
			name:          "basic auth - normal",
			authInfo:      basicAuthInfo,
			secret:        basicAuthSecret,
			shouldSucceed: true,
		},
		{
			name:          "basic auth - invalid secret",
			authInfo:      basicAuthInfo,
			secret:        bearerAuthSecret,
			shouldSucceed: false,
		},
		{
			name:          "basic auth - secret not found",
			authInfo:      basicAuthInfo,
			secret:        nil,
			shouldSucceed: false,
		},
		{
			name:          "bearer auth - normal",
			authInfo:      bearerAuthInfo,
			secret:        bearerAuthSecret,
			shouldSucceed: true,
		},
		{
			name:          "bearer auth - invalid secret",
			authInfo:      bearerAuthInfo,
			secret:        basicAuthSecret,
			shouldSucceed: false,
		},
		{
			name:          "bearer auth - secret not found",
			authInfo:      bearerAuthInfo,
			secret:        nil,
			shouldSucceed: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			testReconcileClusterServiceBrokerWithAuth(t, tc.authInfo, tc.secret, tc.shouldSucceed)
		})
	}
}

func testReconcileClusterServiceBrokerWithAuth(t *testing.T, authInfo *v1beta1.ClusterServiceBrokerAuthInfo, secret *corev1.Secret, shouldSucceed bool) {
	fakeKubeClient, fakeCatalogClient, fakeClusterServiceBrokerClient, testController, _ := newTestController(t, fakeosb.FakeClientConfiguration{})

	broker := getTestClusterServiceBrokerWithAuth(authInfo)
	if secret != nil {
		addGetSecretReaction(fakeKubeClient, secret)
	} else {
		addGetSecretNotFoundReaction(fakeKubeClient)
	}
	testClusterServiceClass := getTestClusterServiceClass()
	fakeClusterServiceBrokerClient.CatalogReaction = &fakeosb.CatalogReaction{
		Response: &osb.CatalogResponse{
			Services: []osb.Service{
				{
					ID:   testClusterServiceClass.Spec.ExternalID,
					Name: testClusterServiceClass.Name,
				},
			},
		},
	}

	err := reconcileClusterServiceBroker(t, testController, broker)
	if shouldSucceed && err != nil {
		t.Fatal("Should have succeeded to get the catalog for the broker. got error: ", err)
	}

	brokerActions := fakeClusterServiceBrokerClient.Actions()
	if shouldSucceed {
		// GetCatalog
		assertNumberOfBrokerActions(t, brokerActions, 1)
		assertGetCatalog(t, brokerActions[0])
	} else {
		assertNumberOfBrokerActions(t, brokerActions, 0)
	}

	actions := fakeCatalogClient.Actions()
	if shouldSucceed {
		assertNumberOfActions(t, actions, 2)
		assertCreate(t, actions[0], testClusterServiceClass)
		updatedClusterServiceBroker := assertUpdateStatus(t, actions[1], broker)
		assertClusterServiceBrokerReadyTrue(t, updatedClusterServiceBroker)
	} else {
		assertNumberOfActions(t, actions, 1)
		updatedClusterServiceBroker := assertUpdateStatus(t, actions[0], broker)
		assertClusterServiceBrokerReadyFalse(t, updatedClusterServiceBroker)
	}

	// verify one kube action occurred
	kubeActions := fakeKubeClient.Actions()
	assertNumberOfActions(t, kubeActions, 1)

	getAction := kubeActions[0].(clientgotesting.GetAction)
	if e, a := "get", getAction.GetVerb(); e != a {
		t.Fatalf("Unexpected verb on action; %s", expectedGot(e, a))
	}
	if e, a := "secrets", getAction.GetResource().Resource; e != a {
		t.Fatalf("Unexpected resource on action; %s", expectedGot(e, a))
	}

	events := getRecordedEvents(testController)
	assertNumEvents(t, events, 1)

	var expectedEvent string
	if shouldSucceed {
		expectedEvent = corev1.EventTypeNormal + " " + successFetchedCatalogReason + " " + successFetchedCatalogMessage
	} else {
		expectedEvent = corev1.EventTypeWarning + " " + errorAuthCredentialsReason + " " + `Error getting broker auth credentials`
	}
	if e, a := expectedEvent, events[0]; !strings.HasPrefix(a, e) {
		t.Fatalf("Received unexpected event, %s", expectedGot(e, a))
	}
}

// TestReconcileClusterServiceBrokerWithReconcileError simulates broker reconciliation where
// creation of a service class causes an error which causes ReconcileClusterServiceBroker to
// return an error
func TestReconcileClusterServiceBrokerWithReconcileError(t *testing.T) {
	fakeKubeClient, fakeCatalogClient, fakeClusterServiceBrokerClient, testController, _ := newTestController(t, getTestCatalogConfig())

	broker := getTestClusterServiceBroker()

	fakeCatalogClient.AddReactor("create", "clusterserviceclasses", func(action clientgotesting.Action) (bool, runtime.Object, error) {
		return true, nil, errors.New("error creating serviceclass")
	})

	if err := reconcileClusterServiceBroker(t, testController, broker); err == nil {
		t.Fatal("There should have been an error.")
	}

	brokerActions := fakeClusterServiceBrokerClient.Actions()
	assertNumberOfBrokerActions(t, brokerActions, 1)
	assertGetCatalog(t, brokerActions[0])

	actions := fakeCatalogClient.Actions()
	assertNumberOfActions(t, actions, 4)

	listRestrictions := clientgotesting.ListRestrictions{
		Labels: labels.Everything(),
		Fields: fields.OneTermEqualSelector("spec.clusterServiceBrokerName", broker.Name),
	}
	assertList(t, actions[0], &v1beta1.ClusterServiceClass{}, listRestrictions)
	assertList(t, actions[1], &v1beta1.ClusterServicePlan{}, listRestrictions)

	// the two plans in the catalog as two separate actions

	createSCAction := actions[2].(clientgotesting.CreateAction)
	createdSC, ok := createSCAction.GetObject().(*v1beta1.ClusterServiceClass)
	if !ok {
		t.Fatalf("couldn't convert to a ClusterServiceClass: %+v", createSCAction.GetObject())
	}
	if e, a := getTestClusterServiceClass(), createdSC; !reflect.DeepEqual(e, a) {
		t.Fatalf("unexpected diff for created ClusterServiceClass: %v,\n\nEXPECTED: %+v\n\nACTUAL:  %+v", diff.ObjectReflectDiff(e, a), e, a)
	}
	updatedClusterServiceBroker := assertUpdateStatus(t, actions[3], broker)
	assertClusterServiceBrokerReadyFalse(t, updatedClusterServiceBroker)

	kubeActions := fakeKubeClient.Actions()
	assertNumberOfActions(t, kubeActions, 0)

	events := getRecordedEvents(testController)

	expectedEvent := warningEventBuilder(errorSyncingCatalogReason).msgf(
		"Error reconciling ClusterServiceClass (K8S: %q ExternalName: %q) (broker %q):",
		testClusterServiceClassGUID, testClusterServiceClassName, testClusterServiceBrokerName,
	).msg("error creating serviceclass")
	if err := checkEvents(events, expectedEvent.stringArr()); err != nil {
		t.Fatal(err)
	}
}

// TestReconcileClusterServiceBrokerSuccessOnFinalRetry verifies that reconciliation can
// succeed on the last attempt before timing out of the retry loop
func TestReconcileClusterServiceBrokerSuccessOnFinalRetry(t *testing.T) {
	fakeKubeClient, fakeCatalogClient, fakeClusterServiceBrokerClient, testController, _ := newTestController(t, getTestCatalogConfig())

	testClusterServiceClass := getTestClusterServiceClass()

	broker := getTestClusterServiceBroker()
	// seven days ago, before the last refresh period
	startTime := metav1.NewTime(time.Now().Add(-7 * 24 * time.Hour))
	broker.Status.OperationStartTime = &startTime

	if err := reconcileClusterServiceBroker(t, testController, broker); err != nil {
		t.Fatalf("This should not fail : %v", err)
	}

	brokerActions := fakeClusterServiceBrokerClient.Actions()
	assertNumberOfBrokerActions(t, brokerActions, 1)
	assertGetCatalog(t, brokerActions[0])

	actions := fakeCatalogClient.Actions()
	assertNumberOfActions(t, actions, 7)

	listRestrictions := clientgotesting.ListRestrictions{
		Labels: labels.Everything(),
		Fields: fields.OneTermEqualSelector("spec.clusterServiceBrokerName", broker.Name),
	}

	// first action should be an update action to clear OperationStartTime
	updatedClusterServiceBroker := assertUpdateStatus(t, actions[0], getTestClusterServiceBroker())
	assertClusterServiceBrokerOperationStartTimeSet(t, updatedClusterServiceBroker, false)

	assertList(t, actions[1], &v1beta1.ClusterServiceClass{}, listRestrictions)
	assertList(t, actions[2], &v1beta1.ClusterServicePlan{}, listRestrictions)
	assertCreate(t, actions[3], testClusterServiceClass)
	assertCreate(t, actions[4], getTestClusterServicePlan())
	assertCreate(t, actions[5], getTestClusterServicePlanNonbindable())

	updatedClusterServiceBroker = assertUpdateStatus(t, actions[6], getTestClusterServiceBroker())
	assertClusterServiceBrokerReadyTrue(t, updatedClusterServiceBroker)

	// verify no kube resources created
	kubeActions := fakeKubeClient.Actions()
	assertNumberOfActions(t, kubeActions, 0)
}

// TestReconcileClusterServiceBrokerFailureOnFinalRetry verifies that reconciliation
// completes in the event of an error after the retry duration elapses.
func TestReconcileClusterServiceBrokerFailureOnFinalRetry(t *testing.T) {
	fakeKubeClient, fakeCatalogClient, fakeClusterServiceBrokerClient, testController, _ := newTestController(t, fakeosb.FakeClientConfiguration{
		CatalogReaction: &fakeosb.CatalogReaction{
			Error: errors.New("ooops"),
		},
	})

	broker := getTestClusterServiceBroker()
	startTime := metav1.NewTime(time.Now().Add(-7 * 24 * time.Hour))
	broker.Status.OperationStartTime = &startTime

	if err := reconcileClusterServiceBroker(t, testController, broker); err != nil {
		t.Fatalf("Should have return no error because the retry duration has elapsed: %v", err)
	}

	brokerActions := fakeClusterServiceBrokerClient.Actions()
	assertNumberOfBrokerActions(t, brokerActions, 1)
	assertGetCatalog(t, brokerActions[0])

	actions := fakeCatalogClient.Actions()
	assertNumberOfActions(t, actions, 2)

	updatedClusterServiceBroker := assertUpdateStatus(t, actions[0], broker)
	assertClusterServiceBrokerReadyFalse(t, updatedClusterServiceBroker)

	updatedClusterServiceBroker = assertUpdateStatus(t, actions[1], broker)
	assertClusterServiceBrokerCondition(t, updatedClusterServiceBroker, v1beta1.ServiceBrokerConditionFailed, v1beta1.ConditionTrue)
	assertClusterServiceBrokerOperationStartTimeSet(t, updatedClusterServiceBroker, false)

	assertNumberOfActions(t, fakeKubeClient.Actions(), 0)

	events := getRecordedEvents(testController)

	expectedEventPrefixes := []string{
		warningEventBuilder(errorFetchingCatalogReason).String(),
		warningEventBuilder(errorReconciliationRetryTimeoutReason).String(),
	}

	if err := checkEventPrefixes(events, expectedEventPrefixes); err != nil {
		t.Fatal(err)
	}
}

// TestReconcileClusterServiceBrokerWithStatusUpdateError verifies that the reconciler
// returns an error when there is a conflict updating the status of the resource.
// This is an otherwise successful scenario where the update to set the
// ready condition fails.
func TestReconcileClusterServiceBrokerWithStatusUpdateError(t *testing.T) {
	fakeKubeClient, fakeCatalogClient, fakeClusterServiceBrokerClient, testController, _ := newTestController(t, getTestCatalogConfig())

	testClusterServiceClass := getTestClusterServiceClass()

	broker := getTestClusterServiceBroker()

	fakeCatalogClient.AddReactor("update", "clusterservicebrokers", func(action clientgotesting.Action) (bool, runtime.Object, error) {
		return true, nil, errors.New("update error")
	})

	err := reconcileClusterServiceBroker(t, testController, broker)
	if err == nil {
		t.Fatalf("expected error from but got none")
	}
	if e, a := "update error", err.Error(); e != a {
		t.Fatalf("unexpected error returned: %s", expectedGot(e, a))
	}

	brokerActions := fakeClusterServiceBrokerClient.Actions()
	assertNumberOfBrokerActions(t, brokerActions, 1)
	assertGetCatalog(t, brokerActions[0])

	actions := fakeCatalogClient.Actions()
	assertNumberOfActions(t, actions, 6)

	listRestrictions := clientgotesting.ListRestrictions{
		Labels: labels.Everything(),
		Fields: fields.OneTermEqualSelector("spec.clusterServiceBrokerName", broker.Name),
	}

	assertList(t, actions[0], &v1beta1.ClusterServiceClass{}, listRestrictions)
	assertList(t, actions[1], &v1beta1.ClusterServicePlan{}, listRestrictions)
	assertCreate(t, actions[2], testClusterServiceClass)
	assertCreate(t, actions[3], getTestClusterServicePlan())
	assertCreate(t, actions[4], getTestClusterServicePlanNonbindable())

	// 4 update action for broker status subresource
	updatedClusterServiceBroker := assertUpdateStatus(t, actions[5], getTestClusterServiceBroker())
	assertClusterServiceBrokerReadyTrue(t, updatedClusterServiceBroker)

	// verify no kube resources created
	kubeActions := fakeKubeClient.Actions()
	assertNumberOfActions(t, kubeActions, 0)
}

// TestUpdateServiceBrokerCondition ensures that with specific conditions
// the broker correctly reflects the changes during updateServiceBrokerCondition().
//
// The test cases are proving:
// - broker transitions from unset status to not ready results in status change and new time
// - broker transitions from not ready to not ready results in no changes
// - broker transitions from not ready to not ready and with reason & msg updates results in no time change, but reflects new reason & msg
// - broker transitions from not ready to ready results in status change & new time
// - broker transitions from ready to ready results in no status change
// - broker transitions from ready to not ready results in status change & new time
// - condition reason & message should always be updated
func TestUpdateServiceBrokerCondition(t *testing.T) {
	// Anonymous struct fields:
	// name: short description of the test
	// input: broker object to test
	// status: new condition status
	// reason: condition reason
	// message: condition message
	// transitionTimeChanged: true if the test conditions should result in transition time change
	cases := []struct {
		name                  string
		input                 *v1beta1.ClusterServiceBroker
		status                v1beta1.ConditionStatus
		reason                string
		message               string
		transitionTimeChanged bool
	}{

		{
			name:                  "initially unset",
			input:                 getTestClusterServiceBroker(),
			status:                v1beta1.ConditionFalse,
			transitionTimeChanged: true,
		},
		{
			name:                  "not ready -> not ready",
			input:                 getTestClusterServiceBrokerWithStatus(v1beta1.ConditionFalse),
			status:                v1beta1.ConditionFalse,
			transitionTimeChanged: false,
		},
		{
			name:                  "not ready -> not ready with reason and message change",
			input:                 getTestClusterServiceBrokerWithStatus(v1beta1.ConditionFalse),
			status:                v1beta1.ConditionFalse,
			reason:                "foo",
			message:               "bar",
			transitionTimeChanged: false,
		},
		{
			name:                  "not ready -> ready",
			input:                 getTestClusterServiceBrokerWithStatus(v1beta1.ConditionFalse),
			status:                v1beta1.ConditionTrue,
			transitionTimeChanged: true,
		},
		{
			name:                  "ready -> ready",
			input:                 getTestClusterServiceBrokerWithStatus(v1beta1.ConditionTrue),
			status:                v1beta1.ConditionTrue,
			transitionTimeChanged: false,
		},
		{
			name:                  "ready -> not ready",
			input:                 getTestClusterServiceBrokerWithStatus(v1beta1.ConditionTrue),
			status:                v1beta1.ConditionFalse,
			transitionTimeChanged: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, fakeCatalogClient, _, testController, _ := newTestController(t, getTestCatalogConfig())

			inputClone := tc.input.DeepCopy()

			err := testController.updateClusterServiceBrokerCondition(tc.input, v1beta1.ServiceBrokerConditionReady, tc.status, tc.reason, tc.message)
			if err != nil {
				t.Fatalf("%v: error updating broker condition: %v", tc.name, err)
			}

			if !reflect.DeepEqual(tc.input, inputClone) {
				t.Fatalf("%v: updating broker condition mutated input: %s", tc.name, expectedGot(inputClone, tc.input))
			}

			actions := fakeCatalogClient.Actions()
			assertNumberOfActions(t, actions, 1)

			updatedClusterServiceBroker := assertUpdateStatus(t, actions[0], tc.input)

			updateActionObject, ok := updatedClusterServiceBroker.(*v1beta1.ClusterServiceBroker)
			if !ok {
				t.Fatalf("%v: couldn't convert to broker", tc.name)
			}

			var initialTs metav1.Time
			if len(inputClone.Status.Conditions) != 0 {
				initialTs = inputClone.Status.Conditions[0].LastTransitionTime
			}

			if e, a := 1, len(updateActionObject.Status.Conditions); e != a {
				t.Fatalf("%v: %s", tc.name, expectedGot(e, a))
			}

			outputCondition := updateActionObject.Status.Conditions[0]
			newTs := outputCondition.LastTransitionTime

			if tc.transitionTimeChanged && initialTs == newTs {
				t.Fatalf("%v: transition time didn't change when it should have", tc.name)
			} else if !tc.transitionTimeChanged && initialTs != newTs {
				t.Fatalf("%v: transition time changed when it shouldn't have", tc.name)
			}
			if e, a := tc.reason, outputCondition.Reason; e != "" && e != a {
				t.Fatalf("%v: condition reasons didn't match; %s", tc.name, expectedGot(e, a))
			}
			if e, a := tc.message, outputCondition.Message; e != "" && e != a {
				t.Fatalf("%v: condition message didn't match; %s", tc.name, expectedGot(e, a))
			}
		})
	}
}

func TestReconcileClusterServicePlanFromClusterServiceBrokerCatalog(t *testing.T) {
	updatedPlan := func() *v1beta1.ClusterServicePlan {
		p := getTestClusterServicePlan()
		p.Spec.Description = "new-description"
		p.Spec.ExternalName = "new-value"
		p.Spec.Free = false
		p.Spec.ExternalMetadata = &runtime.RawExtension{Raw: []byte(`{"field1": "value1"}`)}
		p.Spec.ServiceInstanceCreateParameterSchema = &runtime.RawExtension{Raw: []byte(`{"field1": "value1"}`)}
		p.Spec.ServiceInstanceUpdateParameterSchema = &runtime.RawExtension{Raw: []byte(`{"field1": "value1"}`)}
		p.Spec.ServiceBindingCreateParameterSchema = &runtime.RawExtension{Raw: []byte(`{"field1": "value1"}`)}

		return p
	}

	cases := []struct {
		name                    string
		newServicePlan          *v1beta1.ClusterServicePlan
		existingServicePlan     *v1beta1.ClusterServicePlan
		listerServicePlan       *v1beta1.ClusterServicePlan
		shouldError             bool
		errText                 *string
		catalogClientPrepFunc   func(*fake.Clientset)
		catalogActionsCheckFunc func(t *testing.T, name string, actions []clientgotesting.Action)
	}{
		{
			name:           "new plan",
			newServicePlan: getTestClusterServicePlan(),
			shouldError:    false,
			catalogActionsCheckFunc: func(t *testing.T, name string, actions []clientgotesting.Action) {
				assertNumberOfActions(t, actions, 1)
				assertCreate(t, actions[0], getTestClusterServicePlan())
			},
		},
		{
			name:                "exists, but for a different broker",
			newServicePlan:      getTestClusterServicePlan(),
			existingServicePlan: getTestClusterServicePlan(),
			listerServicePlan: func() *v1beta1.ClusterServicePlan {
				p := getTestClusterServicePlan()
				p.Spec.ClusterServiceBrokerName = "something-else"
				return p
			}(),
			shouldError: true,
			errText:     strPtr(`ClusterServiceBroker "test-clusterservicebroker": ClusterServicePlan "test-clusterserviceplan" already exists for Broker "something-else"`),
		},
		{
			name:                "plan update",
			newServicePlan:      updatedPlan(),
			existingServicePlan: getTestClusterServicePlan(),
			shouldError:         false,
			catalogActionsCheckFunc: func(t *testing.T, name string, actions []clientgotesting.Action) {
				assertNumberOfActions(t, actions, 1)
				assertUpdate(t, actions[0], updatedPlan())
			},
		},
		{
			name:                "plan update - failure",
			newServicePlan:      updatedPlan(),
			existingServicePlan: getTestClusterServicePlan(),
			catalogClientPrepFunc: func(client *fake.Clientset) {
				client.AddReactor("update", "clusterserviceplans", func(action clientgotesting.Action) (bool, runtime.Object, error) {
					return true, nil, errors.New("oops")
				})
			},
			shouldError: true,
			errText:     strPtr("oops"),
		},
	}

	broker := getTestClusterServiceBroker()

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, fakeCatalogClient, _, testController, sharedInformers := newTestController(t, noFakeActions())
			if tc.catalogClientPrepFunc != nil {
				tc.catalogClientPrepFunc(fakeCatalogClient)
			}

			if tc.listerServicePlan != nil {
				sharedInformers.ClusterServicePlans().Informer().GetStore().Add(tc.listerServicePlan)
			}

			err := testController.reconcileClusterServicePlanFromClusterServiceBrokerCatalog(broker, tc.newServicePlan, tc.existingServicePlan)
			if err != nil {
				if !tc.shouldError {
					t.Fatalf("%v: unexpected error from method under test: %v", tc.name, err)
				} else if tc.errText != nil && *tc.errText != err.Error() {
					t.Fatalf("%v: unexpected error text from method under test; %s", tc.name, expectedGot(tc.errText, err.Error()))
				}
			}

			if tc.catalogActionsCheckFunc != nil {
				actions := fakeCatalogClient.Actions()
				tc.catalogActionsCheckFunc(t, tc.name, actions)
			}
		})
	}
}

func reconcileClusterServiceBroker(t *testing.T, testController *controller, broker *v1beta1.ClusterServiceBroker) error {
	clone := broker.DeepCopy()
	err := testController.reconcileClusterServiceBroker(broker)
	if !reflect.DeepEqual(broker, clone) {
		t.Errorf("reconcileClusterServiceBroker shouldn't mutate input, but it does: %s", expectedGot(clone, broker))
	}
	return err
}

// TestReconcileUpdatesManagedClassesAndPlans
// verifies that when an service classes and plans are updated during relist
// that they are flagged as service catalog managed.
func TestReconcileUpdatesManagedClassesAndPlans(t *testing.T) {
	_, fakeCatalogClient, _, testController, sharedInformers := newTestController(t, getTestCatalogConfig())

	testClusterServiceClass := getTestClusterServiceClass()
	testClusterServicePlan := getTestClusterServicePlan()

	sharedInformers.ClusterServiceClasses().Informer().GetStore().Add(testClusterServiceClass)
	sharedInformers.ClusterServicePlans().Informer().GetStore().Add(testClusterServicePlan)

	fakeCatalogClient.AddReactor("list", "clusterserviceclasses", func(action clientgotesting.Action) (bool, runtime.Object, error) {
		return true, &v1beta1.ClusterServiceClassList{
			Items: []v1beta1.ClusterServiceClass{
				*testClusterServiceClass,
			},
		}, nil
	})
	fakeCatalogClient.AddReactor("list", "clusterserviceplans", func(action clientgotesting.Action) (bool, runtime.Object, error) {
		return true, &v1beta1.ClusterServicePlanList{
			Items: []v1beta1.ClusterServicePlan{
				*testClusterServicePlan,
			},
		}, nil
	})

	if err := reconcileClusterServiceBroker(t, testController, getTestClusterServiceBroker()); err != nil {
		t.Fatalf("This should not fail: %v", err)
	}

	actions := fakeCatalogClient.Actions()

	c := assertUpdate(t, actions[2], testClusterServiceClass)
	updatedClass, ok := c.(metav1.Object)
	if !ok {
		t.Fatalf("could not cast %T to metav1.Object", c)
	}
	if !isServiceCatalogManagedResource(updatedClass) {
		t.Error("expected the class to have a service catalog controller reference")
	}

	p := assertUpdate(t, actions[3], testClusterServicePlan)
	updatedPlan, ok := p.(metav1.Object)
	if !ok {
		t.Fatalf("could not cast %T to metav1.Object", p)
	}
	if !isServiceCatalogManagedResource(updatedPlan) {
		t.Error("expected the plan to have a service catalog controller reference")
	}
}

// TestReconcileMarksNewResourcesAsManaged
// verifies that when new service classes and plans are created during relist
// that they are flagged as service catalog managed.
func TestReconcileCreatesManagedClassesAndPlans(t *testing.T) {
	_, fakeCatalogClient, _, testController, _ := newTestController(t, getTestCatalogConfig())

	testClusterServiceClass := getTestClusterServiceClass()
	testClusterServicePlan := getTestClusterServicePlan()

	if err := reconcileClusterServiceBroker(t, testController, getTestClusterServiceBroker()); err != nil {
		t.Fatalf("This should not fail: %v", err)
	}

	actions := fakeCatalogClient.Actions()

	// Verify that the new class and plan are marked as managed
	c := assertCreate(t, actions[2], testClusterServiceClass)
	createdClass, ok := c.(metav1.Object)
	if !ok {
		t.Fatalf("could not cast %T to metav1.Object", c)
	}
	if !isServiceCatalogManagedResource(createdClass) {
		t.Error("expected the class to have a service catalog controller reference")
	}

	p := assertCreate(t, actions[3], testClusterServicePlan)
	createdPlan, ok := p.(metav1.Object)
	if !ok {
		t.Fatalf("could not cast %T to metav1.Object", p)
	}
	if !isServiceCatalogManagedResource(createdPlan) {
		t.Error("expected the plan to have a service catalog controller reference")
	}
}

// TestReconcileDoesNotUpdateUserDefinedClassesAndPlans
// verifies that user-defined classes and plans are not modified
// during relist.
func TestReconcileMarksExistingClassesAndPlansAsManaged(t *testing.T) {
	_, fakeCatalogClient, _, testController, sharedInformers := newTestController(t, getTestCatalogConfig())

	testClusterServiceClass := getTestClusterServiceClass()
	testClusterServicePlan := getTestClusterServicePlan()

	// Remove the controller ref, but keep the same names as resources returned during broker list
	testClusterServiceClass.ObjectMeta.OwnerReferences = nil
	testClusterServicePlan.ObjectMeta.OwnerReferences = nil

	sharedInformers.ClusterServiceClasses().Informer().GetStore().Add(testClusterServiceClass)
	sharedInformers.ClusterServicePlans().Informer().GetStore().Add(testClusterServicePlan)

	fakeCatalogClient.AddReactor("list", "clusterserviceclasses", func(action clientgotesting.Action) (bool, runtime.Object, error) {
		return true, &v1beta1.ClusterServiceClassList{
			Items: []v1beta1.ClusterServiceClass{
				*testClusterServiceClass,
			},
		}, nil
	})
	fakeCatalogClient.AddReactor("list", "clusterserviceplans", func(action clientgotesting.Action) (bool, runtime.Object, error) {
		return true, &v1beta1.ClusterServicePlanList{
			Items: []v1beta1.ClusterServicePlan{
				*testClusterServicePlan,
			},
		}, nil
	})

	if err := reconcileClusterServiceBroker(t, testController, getTestClusterServiceBroker()); err != nil {
		t.Fatalf("This should not fail: %v", err)
	}

	actions := fakeCatalogClient.Actions()

	// Verify that the existing class and plan are now marked as managed
	c := assertUpdate(t, actions[2], testClusterServiceClass)
	updatedClass, ok := c.(metav1.Object)
	if !ok {
		t.Fatalf("could not cast %T to metav1.Object", c)
	}
	if !isServiceCatalogManagedResource(updatedClass) {
		t.Error("expected the class to have a service catalog controller reference")
	}

	p := assertUpdate(t, actions[3], testClusterServicePlan)
	updatedPlan, ok := p.(metav1.Object)
	if !ok {
		t.Fatalf("could not cast %T to metav1.Object", p)
	}
	if !isServiceCatalogManagedResource(updatedPlan) {
		t.Error("expected the plan to have a service catalog controller reference")
	}
}

// TestReconcileDoesNotDeleteUserDefinedClassesAndPlans
// verifies that user-defined plans are not marked with RemovedFromBrokerCatalog during a list.
func TestReconcileDoesNotUpdateUserDefinedClassesAndPlans(t *testing.T) {
	_, fakeCatalogClient, _, testController, sharedInformers := newTestController(t, getTestCatalogConfig())

	testClusterServiceClass := getTestClusterServiceClass()
	testClusterServicePlan := getTestClusterServicePlan()

	// Flag the class and plan as user-defined with unique names not found in the broker catalog
	testClusterServiceClass.OwnerReferences = nil
	testClusterServiceClass.Name = "user-defined-class"
	testClusterServicePlan.OwnerReferences = nil
	testClusterServicePlan.Name = "user-defined-plan"

	sharedInformers.ClusterServiceClasses().Informer().GetStore().Add(testClusterServiceClass)
	sharedInformers.ClusterServicePlans().Informer().GetStore().Add(testClusterServicePlan)

	fakeCatalogClient.AddReactor("list", "clusterserviceclasses", func(action clientgotesting.Action) (bool, runtime.Object, error) {
		return true, &v1beta1.ClusterServiceClassList{
			Items: []v1beta1.ClusterServiceClass{
				*testClusterServiceClass,
			},
		}, nil
	})
	fakeCatalogClient.AddReactor("list", "clusterserviceplans", func(action clientgotesting.Action) (bool, runtime.Object, error) {
		return true, &v1beta1.ClusterServicePlanList{
			Items: []v1beta1.ClusterServicePlan{
				*testClusterServicePlan,
			},
		}, nil
	})

	if err := reconcileClusterServiceBroker(t, testController, getTestClusterServiceBroker()); err != nil {
		t.Fatalf("This should not fail: %v", err)
	}

	actions := fakeCatalogClient.Actions()

	// Verify none of the actions affected the user-defined class and plan
	for _, a := range actions {
		r := a.GetResource().Resource
		if a.GetVerb() == "update" &&
			(r == "clusterserviceclasses" || r == "clusterserviceplans") {
			t.Errorf("expected user-defined classes and plans to be ignored but found action %+v", a)
		}
	}
}

func TestIsServiceCatalogManagedResource(t *testing.T) {
	testcases := []struct {
		name     string
		resource metav1.Object
		want     bool
	}{
		{"unmanaged service class", &v1beta1.ServiceClass{}, false},
		{"unmanaged service plan", &v1beta1.ServicePlan{}, false},
		{"managed service class", &v1beta1.ServiceClass{ObjectMeta: metav1.ObjectMeta{OwnerReferences: []metav1.OwnerReference{
			{Controller: truePtr(), APIVersion: v1beta1.SchemeGroupVersion.String()}}}}, true},
		{"managed service plan", &v1beta1.ServicePlan{ObjectMeta: metav1.ObjectMeta{OwnerReferences: []metav1.OwnerReference{
			{Controller: truePtr(), APIVersion: v1beta1.SchemeGroupVersion.String()}}}}, true},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			got := isServiceCatalogManagedResource(tc.resource)
			if tc.want != got {
				t.Fatalf("WANT: %v, GOT: %v", tc.want, got)
			}
		})
	}
}

func TestMarkAsServiceCatalogManagedResource(t *testing.T) {
	testcases := []struct {
		name     string
		resource metav1.Object
	}{
		{"service class", &v1beta1.ServiceClass{}},
		{"service plan", &v1beta1.ServicePlan{}},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			broker := getTestClusterServiceBroker()
			markAsServiceCatalogManagedResource(tc.resource, broker)

			numOwners := len(tc.resource.GetOwnerReferences())
			if numOwners != 1 {
				t.Fatalf("Expected 1 owner reference, got %v", numOwners)
			}

			gotOwner := tc.resource.GetOwnerReferences()[0]

			gotIsController := gotOwner.Controller != nil && *gotOwner.Controller == true
			if !gotIsController {
				t.Errorf("Expected a controller reference, but Controller is false")
			}

			gotBlockOwnerDeletion := gotOwner.BlockOwnerDeletion != nil && *gotOwner.BlockOwnerDeletion == true
			if gotBlockOwnerDeletion {
				t.Errorf("Expected the controller reference to not modify deletion semantics, but BlockOwnerDeletion is true")
			}

			wantAPIVersion := v1beta1.SchemeGroupVersion.String()
			gotAPIVersion := gotOwner.APIVersion
			if wantAPIVersion != gotAPIVersion {
				t.Errorf("unexpected APIVersion. WANT: %q, GOT: %q", wantAPIVersion, gotAPIVersion)
			}

			// Also verify that our pair of functions work together
			if !isServiceCatalogManagedResource(tc.resource) {
				t.Fatal("expected isServiceCatalogManagedResource to return true")
			}
		})
	}
}
