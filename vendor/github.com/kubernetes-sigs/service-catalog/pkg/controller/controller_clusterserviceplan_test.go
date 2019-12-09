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
	"testing"

	"github.com/kubernetes-sigs/service-catalog/pkg/apis/servicecatalog/v1beta1"
	"github.com/kubernetes-sigs/service-catalog/pkg/util"
	"github.com/kubernetes-sigs/service-catalog/test/fake"
	"k8s.io/apimachinery/pkg/runtime"

	"reflect"

	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	clientgotesting "k8s.io/client-go/testing"
)

func TestReconcileClusterServicePlanRemovedFromCatalog(t *testing.T) {
	getRemovedPlan := func() *v1beta1.ClusterServicePlan {
		p := getTestClusterServicePlan()
		p.Status.RemovedFromBrokerCatalog = true
		return p
	}

	cases := []struct {
		name                    string
		plan                    *v1beta1.ClusterServicePlan
		instances               []v1beta1.ServiceInstance
		catalogClientPrepFunc   func(*fake.Clientset)
		shouldError             bool
		errText                 *string
		catalogActionsCheckFunc func(t *testing.T, actions []clientgotesting.Action)
	}{
		{
			name:        "not removed from catalog",
			plan:        getTestClusterServicePlan(),
			shouldError: false,
		},
		{
			name:        "removed from catalog, instances left",
			plan:        getRemovedPlan(),
			instances:   []v1beta1.ServiceInstance{*getTestServiceInstance()},
			shouldError: false,
			catalogActionsCheckFunc: func(t *testing.T, actions []clientgotesting.Action) {
				listRestrictions := clientgotesting.ListRestrictions{
					Labels: labels.SelectorFromSet(labels.Set{
						v1beta1.GroupName + "/" + v1beta1.FilterSpecClusterServicePlanRefName: util.GenerateSHA("cspguid"),
					}),
					Fields: fields.Everything(),
				}

				assertNumberOfActions(t, actions, 1)
				assertList(t, actions[0], &v1beta1.ServiceInstance{}, listRestrictions)
			},
		},
		{
			name:        "removed from catalog, no instances left",
			plan:        getRemovedPlan(),
			instances:   nil,
			shouldError: false,
			catalogActionsCheckFunc: func(t *testing.T, actions []clientgotesting.Action) {
				listRestrictions := clientgotesting.ListRestrictions{
					Labels: labels.SelectorFromSet(labels.Set{
						v1beta1.GroupName + "/" + v1beta1.FilterSpecClusterServicePlanRefName: util.GenerateSHA("cspguid"),
					}),
					Fields: fields.Everything(),
				}

				assertNumberOfActions(t, actions, 2)
				assertList(t, actions[0], &v1beta1.ServiceInstance{}, listRestrictions)
				assertDelete(t, actions[1], getRemovedPlan())
			},
		},
		{
			name:        "removed from catalog, no instances left, delete fails",
			plan:        getRemovedPlan(),
			instances:   nil,
			shouldError: true,
			catalogClientPrepFunc: func(client *fake.Clientset) {
				client.AddReactor("delete", "clusterserviceplans", func(action clientgotesting.Action) (bool, runtime.Object, error) {
					return true, nil, errors.New("oops")
				})
			},
			errText: strPtr("oops"),
			catalogActionsCheckFunc: func(t *testing.T, actions []clientgotesting.Action) {
				listRestrictions := clientgotesting.ListRestrictions{
					Labels: labels.SelectorFromSet(labels.Set{
						v1beta1.GroupName + "/" + v1beta1.FilterSpecClusterServicePlanRefName: util.GenerateSHA("cspguid"),
					}),
					Fields: fields.Everything(),
				}

				assertNumberOfActions(t, actions, 2)
				assertList(t, actions[0], &v1beta1.ServiceInstance{}, listRestrictions)
				assertDelete(t, actions[1], getRemovedPlan())
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {

			_, fakeCatalogClient, _, testController, _ := newTestController(t, noFakeActions())

			fakeCatalogClient.AddReactor("list", "serviceinstances", func(action clientgotesting.Action) (bool, runtime.Object, error) {
				return true, &v1beta1.ServiceInstanceList{Items: tc.instances}, nil
			})

			if tc.catalogClientPrepFunc != nil {
				tc.catalogClientPrepFunc(fakeCatalogClient)
			}

			err := reconcileClusterServicePlan(t, testController, tc.plan)
			if err != nil {
				if !tc.shouldError {
					t.Fatalf("unexpected error from method under test: %v", err)
				} else if tc.errText != nil && *tc.errText != err.Error() {
					t.Fatalf("unexpected error text from method under test; expected %v, got %v", tc.errText, err.Error())
				}
			}

			actions := fakeCatalogClient.Actions()

			if tc.catalogActionsCheckFunc != nil {
				tc.catalogActionsCheckFunc(t, actions)
			} else {
				assertNumberOfActions(t, actions, 0)
			}
		})
	}
}

func reconcileClusterServicePlan(t *testing.T, testController *controller, clusterServicePlan *v1beta1.ClusterServicePlan) error {
	clone := clusterServicePlan.DeepCopy()
	err := testController.reconcileClusterServicePlan(clusterServicePlan)
	if !reflect.DeepEqual(clusterServicePlan, clone) {
		t.Errorf("reconcileClusterServicePlan shouldn't mutate input, but it does: %s", expectedGot(clone, clusterServicePlan))
	}
	return err
}
