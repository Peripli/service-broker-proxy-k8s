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

package integration

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	scfeatures "github.com/kubernetes-incubator/service-catalog/pkg/features"
	osb "github.com/pmorie/go-open-service-broker-client/v2"
	fakeosb "github.com/pmorie/go-open-service-broker-client/v2/fake"
	utilfeature "k8s.io/apiserver/pkg/util/feature"

	// avoid error `servicecatalog/v1beta1 is not enabled`
	_ "github.com/kubernetes-incubator/service-catalog/pkg/apis/servicecatalog/install"

	"github.com/kubernetes-incubator/service-catalog/pkg/apis/servicecatalog/v1beta1"
	"github.com/kubernetes-incubator/service-catalog/test/util"
)

// TestCreateServiceInstanceNonExistentClusterServiceClassOrPlan tests that a ServiceInstance gets
// a Failed condition when the service class or service plan it references does not exist.
func TestCreateServiceInstanceNonExistentClusterServiceClassOrPlan(t *testing.T) {
	cases := []struct {
		name                string
		classExternalName   string
		classExternalID     string
		planExternalName    string
		planExternalID      string
		classK8sName        string
		planK8sName         string
		expectedErrorReason string
	}{
		{
			name:                "existent external class and plan name",
			classExternalName:   testClusterServiceClassName,
			planExternalName:    testClusterServicePlanName,
			expectedErrorReason: "",
		},
		{
			name:                "non-existent external class name",
			classExternalName:   "nothereclass",
			planExternalName:    testClusterServicePlanName,
			expectedErrorReason: "ReferencesNonexistentServiceClass",
		},
		{
			name:                "non-existent external plan name",
			classExternalName:   testClusterServiceClassName,
			planExternalName:    "nothereplan",
			expectedErrorReason: "ReferencesNonexistentServicePlan",
		},
		{
			name:                "non-existent external class and plan name",
			classExternalName:   "nothereclass",
			planExternalName:    "nothereplan",
			expectedErrorReason: "ReferencesNonexistentServiceClass",
		},
		{
			name:                "existent external class and plan id",
			classExternalID:     testClassExternalID,
			planExternalID:      testPlanExternalID,
			expectedErrorReason: "",
		},
		{
			name:                "non-existent external class id",
			classExternalID:     "nothereclass",
			planExternalID:      testPlanExternalID,
			expectedErrorReason: "ReferencesNonexistentServiceClass",
		},
		{
			name:                "non-existent external plan id",
			classExternalID:     testClassExternalID,
			planExternalID:      "nothereplan",
			expectedErrorReason: "ReferencesNonexistentServicePlan",
		},
		{
			name:                "non-existent external class and plan id",
			classExternalID:     "nothereclass",
			planExternalID:      "nothereplan",
			expectedErrorReason: "ReferencesNonexistentServiceClass",
		},
		{
			name:                "existent k8s class and plan name",
			classK8sName:        testClusterServiceClassGUID,
			planK8sName:         testPlanExternalID,
			expectedErrorReason: "",
		},
		{
			name:                "non-existent k8s class name",
			classK8sName:        "nothereclass",
			planK8sName:         testPlanExternalID,
			expectedErrorReason: "ReferencesNonexistentServiceClass",
		},
		{
			name:                "non-existent k8s plan name",
			classK8sName:        testClusterServiceClassGUID,
			planK8sName:         "nothereplan",
			expectedErrorReason: "ReferencesNonexistentServicePlan",
		},
		{
			name:                "non-existent k8s class and plan name",
			classK8sName:        "nothereclass",
			planK8sName:         "nothereplan",
			expectedErrorReason: "ReferencesNonexistentServiceClass",
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			ct := &controllerTest{
				t:      t,
				broker: getTestBroker(),
				instance: func() *v1beta1.ServiceInstance {
					i := getTestInstance()
					i.Spec.PlanReference.ClusterServiceClassExternalName = tc.classExternalName
					i.Spec.PlanReference.ClusterServicePlanExternalName = tc.planExternalName
					i.Spec.PlanReference.ClusterServiceClassExternalID = tc.classExternalID
					i.Spec.PlanReference.ClusterServicePlanExternalID = tc.planExternalID
					i.Spec.PlanReference.ClusterServiceClassName = tc.classK8sName
					i.Spec.PlanReference.ClusterServicePlanName = tc.planK8sName
					return i
				}(),
				skipVerifyingInstanceSuccess: tc.expectedErrorReason != "",
			}
			ct.run(func(ct *controllerTest) {
				status := v1beta1.ConditionTrue
				if tc.expectedErrorReason != "" {
					status = v1beta1.ConditionFalse
				}
				condition := v1beta1.ServiceInstanceCondition{
					Type: v1beta1.ServiceInstanceConditionReady,

					Status: status,
					Reason: tc.expectedErrorReason,
				}
				if err := util.WaitForInstanceCondition(ct.client, testNamespace, testInstanceName, condition); err != nil {
					t.Fatalf("error waiting for instance condition: %v", err)
				}
			})
		})
	}
}

// TestCreateServiceInstanceNonExistsentClusterServiceBroker tests creating a
// ServiceInstance whose broker does not exist.
func TestCreateServiceInstanceNonExistentClusterServiceBroker(t *testing.T) {
	ct := &controllerTest{
		t:                            t,
		instance:                     getTestInstance(),
		skipVerifyingInstanceSuccess: true,
		preCreateInstance: func(ct *controllerTest) {
			serviceClass := &v1beta1.ClusterServiceClass{
				ObjectMeta: metav1.ObjectMeta{Name: testClusterServiceClassGUID},
				Spec: v1beta1.ClusterServiceClassSpec{
					ClusterServiceBrokerName: testClusterServiceBrokerName,
					CommonServiceClassSpec: v1beta1.CommonServiceClassSpec{
						ExternalID:   testClusterServiceClassGUID,
						ExternalName: testClusterServiceClassName,
						Description:  "a test service",
						Bindable:     true,
					},
				},
			}
			if _, err := ct.client.ClusterServiceClasses().Create(serviceClass); err != nil {
				t.Fatalf("error creating ClusterServiceClass: %v", err)
			}

			if err := util.WaitForClusterServiceClassToExist(ct.client, testClusterServiceClassGUID); err != nil {
				t.Fatalf("error waiting for ClusterServiceClass to exist: %v", err)
			}

			servicePlan := &v1beta1.ClusterServicePlan{
				ObjectMeta: metav1.ObjectMeta{Name: testPlanExternalID},
				Spec: v1beta1.ClusterServicePlanSpec{
					ClusterServiceBrokerName: testClusterServiceBrokerName,
					CommonServicePlanSpec: v1beta1.CommonServicePlanSpec{
						ExternalID:   testPlanExternalID,
						ExternalName: testClusterServicePlanName,
						Description:  "a test plan",
					},
					ClusterServiceClassRef: v1beta1.ClusterObjectReference{
						Name: testClusterServiceClassGUID,
					},
				},
			}
			if _, err := ct.client.ClusterServicePlans().Create(servicePlan); err != nil {
				t.Fatalf("error creating ClusterServicePlan: %v", err)
			}
			if err := util.WaitForClusterServicePlanToExist(ct.client, testPlanExternalID); err != nil {
				t.Fatalf("error waiting for ClusterServicePlan to exist: %v", err)
			}
		},
	}
	ct.run(func(ct *controllerTest) {
		if err := util.WaitForInstanceCondition(ct.client, testNamespace, testInstanceName, v1beta1.ServiceInstanceCondition{
			Type:   v1beta1.ServiceInstanceConditionReady,
			Status: v1beta1.ConditionFalse,
			Reason: "ReferencesNonexistentBroker",
		}); err != nil {
			t.Fatalf("error waiting for instance reconciliation to fail: %v", err)
		}
	})
}

// TestCreateServiceInstanceWithAuthError tests creating a SerivceInstance when
// the secret containing the broker authorization info cannot be found.
func TestCreateServiceInstanceWithAuthError(t *testing.T) {
	ct := &controllerTest{
		t: t,
		broker: func() *v1beta1.ClusterServiceBroker {
			b := getTestBroker()
			b.Spec.AuthInfo = &v1beta1.ClusterServiceBrokerAuthInfo{
				Basic: &v1beta1.ClusterBasicAuthConfig{
					SecretRef: &v1beta1.ObjectReference{
						Namespace: testNamespace,
						Name:      "secret-name",
					},
				},
			}
			return b
		}(),
		instance:                     getTestInstance(),
		skipVerifyingInstanceSuccess: true,
		preCreateBroker: func(ct *controllerTest) {
			prependGetSecretReaction(ct.kubeClient, "secret-name", map[string][]byte{
				"username": []byte("user"),
				"password": []byte("pass"),
			})
		},
		preCreateInstance: func(ct *controllerTest) {
			prependGetSecretNotFoundReaction(ct.kubeClient)
		},
	}
	ct.run(func(ct *controllerTest) {
		if err := util.WaitForInstanceCondition(ct.client, testNamespace, testInstanceName, v1beta1.ServiceInstanceCondition{
			Type:   v1beta1.ServiceInstanceConditionReady,
			Status: v1beta1.ConditionFalse,
			Reason: "ErrorGettingAuthCredentials",
		}); err != nil {
			t.Fatalf("error waiting for instance reconciliation to fail: %v", err)
		}
	})
}

// TestCreateServiceInstanceWithParameters tests creating a ServiceInstance
// with parameters.
func TestCreateServiceInstanceWithParameters(t *testing.T) {
	type secretDef struct {
		name string
		data map[string][]byte
	}
	cases := []struct {
		name           string
		params         map[string]interface{}
		paramsFrom     []v1beta1.ParametersFromSource
		secrets        []secretDef
		expectedParams map[string]interface{}
		expectedError  bool
	}{
		{
			name:           "no params",
			expectedParams: nil,
		},
		{
			name: "plain params",
			params: map[string]interface{}{
				"Name": "test-param",
				"Args": map[string]interface{}{
					"first":  "first-arg",
					"second": "second-arg",
				},
			},
			expectedParams: map[string]interface{}{
				"Name": "test-param",
				"Args": map[string]interface{}{
					"first":  "first-arg",
					"second": "second-arg",
				},
			},
		},
		{
			name: "secret params",
			paramsFrom: []v1beta1.ParametersFromSource{
				{
					SecretKeyRef: &v1beta1.SecretKeyReference{
						Name: "secret-name",
						Key:  "secret-key",
					},
				},
			},
			secrets: []secretDef{
				{
					name: "secret-name",
					data: map[string][]byte{
						"secret-key": []byte(`{"A":"B","C":{"D":"E","F":"G"}}`),
					},
				},
			},
			expectedParams: map[string]interface{}{
				"A": "B",
				"C": map[string]interface{}{
					"D": "E",
					"F": "G",
				},
			},
		},
		{
			name: "plain and secret params",
			params: map[string]interface{}{
				"Name": "test-param",
				"Args": map[string]interface{}{
					"first":  "first-arg",
					"second": "second-arg",
				},
			},
			paramsFrom: []v1beta1.ParametersFromSource{
				{
					SecretKeyRef: &v1beta1.SecretKeyReference{
						Name: "secret-name",
						Key:  "secret-key",
					},
				},
			},
			secrets: []secretDef{
				{
					name: "secret-name",
					data: map[string][]byte{
						"secret-key": []byte(`{"A":"B","C":{"D":"E","F":"G"}}`),
					},
				},
			},
			expectedParams: map[string]interface{}{
				"Name": "test-param",
				"Args": map[string]interface{}{
					"first":  "first-arg",
					"second": "second-arg",
				},
				"A": "B",
				"C": map[string]interface{}{
					"D": "E",
					"F": "G",
				},
			},
		},
		{
			name: "missing secret",
			paramsFrom: []v1beta1.ParametersFromSource{
				{
					SecretKeyRef: &v1beta1.SecretKeyReference{
						Name: "secret-name",
						Key:  "secret-key",
					},
				},
			},
			expectedError: true,
		},
		{
			name: "missing secret key",
			paramsFrom: []v1beta1.ParametersFromSource{
				{
					SecretKeyRef: &v1beta1.SecretKeyReference{
						Name: "secret-name",
						Key:  "other-secret-key",
					},
				},
			},
			secrets: []secretDef{
				{
					name: "secret-name",
					data: map[string][]byte{
						"secret-key": []byte(`bad`),
					},
				},
			},
			expectedError: true,
		},
		{
			name: "empty secret data",
			paramsFrom: []v1beta1.ParametersFromSource{
				{
					SecretKeyRef: &v1beta1.SecretKeyReference{
						Name: "secret-name",
						Key:  "secret-key",
					},
				},
			},
			secrets: []secretDef{
				{
					name: "secret-name",
					data: map[string][]byte{},
				},
			},
			expectedError: true,
		},
		{
			name: "bad secret data",
			paramsFrom: []v1beta1.ParametersFromSource{
				{
					SecretKeyRef: &v1beta1.SecretKeyReference{
						Name: "secret-name",
						Key:  "secret-key",
					},
				},
			},
			secrets: []secretDef{
				{
					name: "secret-name",
					data: map[string][]byte{
						"secret-key": []byte(`bad`),
					},
				},
			},
			expectedError: true,
		},
		{
			name: "no params in secret data",
			paramsFrom: []v1beta1.ParametersFromSource{
				{
					SecretKeyRef: &v1beta1.SecretKeyReference{
						Name: "secret-name",
						Key:  "secret-key",
					},
				},
			},
			secrets: []secretDef{
				{
					name: "secret-name",
					data: map[string][]byte{
						"secret-key": []byte(`{}`),
					},
				},
			},
			expectedParams: nil,
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			//t.Parallel()
			ct := &controllerTest{
				t:      t,
				broker: getTestBroker(),
				instance: func() *v1beta1.ServiceInstance {
					i := getTestInstance()
					if tc.params != nil {
						i.Spec.Parameters = convertParametersIntoRawExtension(t, tc.params)
					}
					i.Spec.ParametersFrom = tc.paramsFrom
					return i
				}(),
				skipVerifyingInstanceSuccess: tc.expectedError,
				setup: func(ct *controllerTest) {
					for _, secret := range tc.secrets {
						prependGetSecretReaction(ct.kubeClient, secret.name, secret.data)
					}
				},
			}
			ct.run(func(ct *controllerTest) {
				if tc.expectedError {
					if err := util.WaitForInstanceCondition(ct.client, testNamespace, testInstanceName, v1beta1.ServiceInstanceCondition{
						Type:   v1beta1.ServiceInstanceConditionReady,
						Status: v1beta1.ConditionFalse,
						Reason: "ErrorWithParameters",
					}); err != nil {
						t.Fatalf("error waiting for instance reconciliation to fail: %v", err)
					}
				} else {
					brokerAction := getLastBrokerAction(t, ct.osbClient, fakeosb.ProvisionInstance)
					if e, a := tc.expectedParams, brokerAction.Request.(*osb.ProvisionRequest).Parameters; !reflect.DeepEqual(e, a) {
						t.Fatalf("unexpected diff in provision parameters: expected %v, got %v", e, a)
					}
				}
			})
		})
	}
}

// TestUpdateServiceInstanceNewDashboardResponse tests setting Dashboard URL when
// and update Instance request returns a new DashboardURL.
func TestUpdateServiceInstanceNewDashboardResponse(t *testing.T) {
	dashURL := testDashboardURL
	cases := []struct {
		name        string
		setup       func(t *controllerTest)
		osbResponse *osb.UpdateInstanceResponse
	}{
		{
			name: "alpha features enabled",
			setup: func(ct *controllerTest) {
				if err := utilfeature.DefaultFeatureGate.Set(fmt.Sprintf("%v=true", scfeatures.UpdateDashboardURL)); err != nil {
					t.Fatalf("Failed to enable updatable dashboard url feature: %v", err)
				}
			},
			osbResponse: &osb.UpdateInstanceResponse{
				DashboardURL: &dashURL,
			},
		},
		{
			name: "alpha feature disabled",
			setup: func(ct *controllerTest) {
				if err := utilfeature.DefaultFeatureGate.Set(fmt.Sprintf("%v=false", scfeatures.UpdateDashboardURL)); err != nil {
					t.Fatalf("Failed to enable updatable dashboard url feature: %v", err)
				}
			},
			osbResponse: &osb.UpdateInstanceResponse{
				DashboardURL: &dashURL,
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ct := &controllerTest{
				t:        t,
				broker:   getTestBroker(),
				instance: getTestInstance(),
				setup: func(ct *controllerTest) {
					ct.osbClient.UpdateInstanceReaction = &fakeosb.UpdateInstanceReaction{
						Response: tc.osbResponse,
					}
				},
			}
			ct.run(func(ct *controllerTest) {
				if utilfeature.DefaultFeatureGate.Enabled(scfeatures.UpdateDashboardURL) {
					if ct.instance.Status.DashboardURL != &dashURL {
						t.Fatalf("unexpected DashboardURL: expected %v got %v", dashURL, *tc.osbResponse.DashboardURL)
					}
				} else {
					if ct.instance.Status.DashboardURL != nil {
						t.Fatal("Dashboard URL should be nil")
					}
				}
			})
		})
	}
}

// TestUpdateServiceInstanceChangePlans tests changing plans for an existing
// ServiceInstance.
func TestUpdateServiceInstanceChangePlans(t *testing.T) {
	otherPlanName := "otherplanname"
	otherPlanID := "other-plan-id"
	cases := []struct {
		name                          string
		useExternalNames              bool
		dynamicUpdateInstanceReaction fakeosb.DynamicUpdateInstanceReaction
		asyncUpdateInstanceReaction   *fakeosb.UpdateInstanceReaction
	}{
		{
			name:             "external",
			useExternalNames: true,
		},
		{
			name:             "k8s",
			useExternalNames: false,
		},
		{
			name:             "external name with two update call failures",
			useExternalNames: true,
			dynamicUpdateInstanceReaction: fakeosb.DynamicUpdateInstanceReaction(
				getUpdateInstanceResponseByPollCountReactions(2, []fakeosb.UpdateInstanceReaction{
					fakeosb.UpdateInstanceReaction{
						Error: errors.New("fake update error"),
					},
					fakeosb.UpdateInstanceReaction{
						Response: &osb.UpdateInstanceResponse{},
					},
				})),
		},
		{
			name:             "external name with two update failures",
			useExternalNames: true,
			dynamicUpdateInstanceReaction: fakeosb.DynamicUpdateInstanceReaction(
				getUpdateInstanceResponseByPollCountReactions(2, []fakeosb.UpdateInstanceReaction{
					fakeosb.UpdateInstanceReaction{
						Error: osb.HTTPStatusCodeError{
							StatusCode:   http.StatusConflict,
							ErrorMessage: strPtr("OutOfQuota"),
							Description:  strPtr("You're out of quota!"),
						},
					},
					fakeosb.UpdateInstanceReaction{
						Response: &osb.UpdateInstanceResponse{},
					},
				})),
		},
		{
			name:             "external name update response async",
			useExternalNames: true,
			asyncUpdateInstanceReaction: &fakeosb.UpdateInstanceReaction{
				Response: &osb.UpdateInstanceResponse{
					Async: true,
				},
			},
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			//t.Parallel()
			ct := &controllerTest{
				t:      t,
				broker: getTestBroker(),
				instance: func() *v1beta1.ServiceInstance {
					i := getTestInstance()
					if !tc.useExternalNames {
						i.Spec.ClusterServiceClassExternalName = ""
						i.Spec.ClusterServicePlanExternalName = ""
						i.Spec.ClusterServiceClassName = testClusterServiceClassGUID
						i.Spec.ClusterServicePlanName = testPlanExternalID
					}
					return i
				}(),
				setup: func(ct *controllerTest) {
					if tc.dynamicUpdateInstanceReaction != nil {
						ct.osbClient.UpdateInstanceReaction = tc.dynamicUpdateInstanceReaction
					} else if tc.asyncUpdateInstanceReaction != nil {
						ct.osbClient.UpdateInstanceReaction = tc.asyncUpdateInstanceReaction
					}
					catalogResponse := ct.osbClient.CatalogReaction.(*fakeosb.CatalogReaction).Response
					catalogResponse.Services[0].PlanUpdatable = truePtr()
					catalogResponse.Services[0].Plans = append(catalogResponse.Services[0].Plans, osb.Plan{
						Name:        otherPlanName,
						Free:        truePtr(),
						ID:          otherPlanID,
						Description: "another test plan",
					})
				},
			}
			ct.run(func(ct *controllerTest) {
				if tc.useExternalNames {
					ct.instance.Spec.ClusterServicePlanExternalName = otherPlanName
				} else {
					ct.instance.Spec.ClusterServicePlanName = otherPlanID
				}

				updatedInstance, err := ct.client.ServiceInstances(testNamespace).Update(ct.instance)
				if err != nil {
					t.Fatalf("error updating Instance: %v", err)
				}
				ct.instance = updatedInstance

				if err := util.WaitForInstanceProcessedGeneration(ct.client, testNamespace, testInstanceName, ct.instance.Generation); err != nil {
					t.Fatalf("error waiting for instance to reconcile: %v", err)
				}

				if tc.asyncUpdateInstanceReaction != nil {
					// action sequence: GetCatalog, ProvisionInstance, UpdateInstance, PollLastOperation
					brokerAction := getLastBrokerAction(t, ct.osbClient, fakeosb.PollLastOperation)
					request := brokerAction.Request.(*osb.LastOperationRequest)
					if request.PlanID == nil {
						t.Fatalf("plan ID not sent in update instance request: request = %+v", request)
					}
					if e, a := otherPlanID, *request.PlanID; e != a {
						t.Fatalf("unexpected plan ID: expected %s, got %s", e, a)
					}
				} else {
					brokerAction := getLastBrokerAction(t, ct.osbClient, fakeosb.UpdateInstance)
					request := brokerAction.Request.(*osb.UpdateInstanceRequest)
					if request.PlanID == nil {
						t.Fatalf("plan ID not sent in update instance request: request = %+v", request)
					}
					if e, a := otherPlanID, *request.PlanID; e != a {
						t.Fatalf("unexpected plan ID: expected %s, got %s", e, a)
					}

				}
			})
		})
	}
}

// TestUpdateServiceInstanceChangePlansToNonexistentPlan tests changing plans
// to a non-existent plan.
func TestUpdateServiceInstanceChangePlansToNonexistentPlan(t *testing.T) {
	cases := []struct {
		name             string
		useExternalNames bool
	}{
		{
			name:             "external",
			useExternalNames: true,
		},
		{
			name:             "k8s",
			useExternalNames: false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ct := &controllerTest{
				t:      t,
				broker: getTestBroker(),
				instance: func() *v1beta1.ServiceInstance {
					i := getTestInstance()
					if !tc.useExternalNames {
						i.Spec.ClusterServiceClassExternalName = ""
						i.Spec.ClusterServicePlanExternalName = ""
						i.Spec.ClusterServiceClassName = testClusterServiceClassGUID
						i.Spec.ClusterServicePlanName = testPlanExternalID
					}
					return i
				}(),
				setup: func(ct *controllerTest) {
					ct.osbClient.CatalogReaction.(*fakeosb.CatalogReaction).Response.Services[0].PlanUpdatable = truePtr()
				},
			}
			ct.run(func(ct *controllerTest) {
				if tc.useExternalNames {
					ct.instance.Spec.ClusterServicePlanExternalName = "other-plan-name"
				} else {
					ct.instance.Spec.ClusterServicePlanName = "other-plan-id"
				}

				if _, err := ct.client.ServiceInstances(testNamespace).Update(ct.instance); err != nil {
					t.Fatalf("error updating Instance: %v", err)
				}

				if err := util.WaitForInstanceCondition(ct.client, testNamespace, testInstanceName, v1beta1.ServiceInstanceCondition{
					Type:   v1beta1.ServiceInstanceConditionReady,
					Status: v1beta1.ConditionFalse,
					Reason: "ReferencesNonexistentServicePlan",
				}); err != nil {
					t.Fatalf("error waiting for instance reconciliation to fail: %v", err)
				}

			})
		})
	}
}

// TestUpdateServiceInstanceUpdateParameters tests updating the parameters
// of an existing ServiceInstance.
func TestUpdateServiceInstanceUpdateParameters(t *testing.T) {
	cases := []struct {
		name                        string
		createdWithParams           bool
		createdWithParamsFromSecret bool
		updateParams                bool
		updateParamsFromSecret      bool
		updateSecret                bool
		deleteParams                bool
		deleteParamsFromSecret      bool
	}{
		{
			name:              "add param",
			createdWithParams: false,
			updateParams:      true,
		},
		{
			name:              "update param",
			createdWithParams: true,
			updateParams:      true,
		},
		{
			name:              "delete param",
			createdWithParams: true,
			deleteParams:      true,
		},
		{
			name:                        "add param with secret",
			createdWithParams:           false,
			createdWithParamsFromSecret: true,
			updateParams:                true,
		},
		{
			name:                        "update param with secret",
			createdWithParams:           true,
			createdWithParamsFromSecret: true,
			updateParams:                true,
		},
		{
			name:                        "delete param with secret",
			createdWithParams:           true,
			createdWithParamsFromSecret: true,
			deleteParams:                true,
		},
		{
			name: "add secret param",
			createdWithParamsFromSecret: false,
			updateParamsFromSecret:      true,
		},
		{
			name: "update secret param",
			createdWithParamsFromSecret: true,
			updateParamsFromSecret:      true,
		},
		{
			name: "delete secret param",
			createdWithParamsFromSecret: true,
			deleteParamsFromSecret:      true,
		},
		{
			name:                        "add secret param with plain param",
			createdWithParams:           true,
			createdWithParamsFromSecret: false,
			updateParamsFromSecret:      true,
		},
		{
			name:                        "update secret param with plain param",
			createdWithParams:           true,
			createdWithParamsFromSecret: true,
			updateParamsFromSecret:      true,
		},
		{
			name:                        "delete secret param with plain param",
			createdWithParams:           true,
			createdWithParamsFromSecret: true,
			deleteParamsFromSecret:      true,
		},
		{
			name: "update secret",
			createdWithParamsFromSecret: true,
			updateSecret:                true,
		},
		{
			name:                        "update secret with plain param",
			createdWithParams:           true,
			createdWithParamsFromSecret: true,
			updateSecret:                true,
		},
		{
			name:                        "add plain and secret param",
			createdWithParams:           false,
			createdWithParamsFromSecret: false,
			updateParams:                true,
			updateParamsFromSecret:      true,
		},
		{
			name:                        "update plain and secret param",
			createdWithParams:           true,
			createdWithParamsFromSecret: true,
			updateParams:                true,
			updateParamsFromSecret:      true,
		},
		{
			name:                        "delete plain and secret param",
			createdWithParams:           true,
			createdWithParamsFromSecret: true,
			deleteParams:                true,
			deleteParamsFromSecret:      true,
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			//t.Parallel()
			ct := &controllerTest{
				t:      t,
				broker: getTestBroker(),
				instance: func() *v1beta1.ServiceInstance {
					i := getTestInstance()
					if tc.createdWithParams {
						i.Spec.Parameters = convertParametersIntoRawExtension(t,
							map[string]interface{}{
								"param-key": "param-value",
							})
					}
					if tc.createdWithParamsFromSecret {
						i.Spec.ParametersFrom = []v1beta1.ParametersFromSource{
							{
								SecretKeyRef: &v1beta1.SecretKeyReference{
									Name: "secret-name",
									Key:  "secret-key",
								},
							},
						}
					}
					return i
				}(),
				setup: func(ct *controllerTest) {
					prependGetSecretReaction(ct.kubeClient, "secret-name", map[string][]byte{
						"secret-key": []byte(`{"secret-param-key":"secret-param-value"}`),
					})
					prependGetSecretReaction(ct.kubeClient, "other-secret-name", map[string][]byte{
						"other-secret-key": []byte(`{"other-secret-param-key":"other-secret-param-value"}`),
					})
				},
			}
			ct.run(func(ct *controllerTest) {
				if tc.updateParams {
					ct.instance.Spec.Parameters = convertParametersIntoRawExtension(t,
						map[string]interface{}{
							"param-key": "new-param-value",
						})
				} else if tc.deleteParams {
					ct.instance.Spec.Parameters = nil
				}

				if tc.updateParamsFromSecret {
					ct.instance.Spec.ParametersFrom = []v1beta1.ParametersFromSource{
						{
							SecretKeyRef: &v1beta1.SecretKeyReference{
								Name: "other-secret-name",
								Key:  "other-secret-key",
							},
						},
					}
				} else if tc.deleteParamsFromSecret {
					ct.instance.Spec.ParametersFrom = nil
				}

				if tc.updateSecret {
					ct.kubeClient.Lock()
					prependGetSecretReaction(ct.kubeClient, "secret-name", map[string][]byte{
						"secret-key": []byte(`{"new-secret-param-key":"new-secret-param-value"}`),
					})
					ct.kubeClient.Unlock()
					ct.instance.Spec.UpdateRequests++
				}

				updatedInstance, err := ct.client.ServiceInstances(testNamespace).Update(ct.instance)
				if err != nil {
					t.Fatalf("error updating Instance: %v", err)
				}
				ct.instance = updatedInstance

				if err := util.WaitForInstanceProcessedGeneration(ct.client, testNamespace, testInstanceName, ct.instance.Generation); err != nil {
					t.Fatalf("error waiting for instance to reconcile: %v", err)
				}

				expectedParameters := make(map[string]interface{})

				if tc.updateParams {
					expectedParameters["param-key"] = "new-param-value"
				} else if tc.createdWithParams && !tc.deleteParams {
					expectedParameters["param-key"] = "param-value"
				}

				if tc.updateParamsFromSecret {
					expectedParameters["other-secret-param-key"] = "other-secret-param-value"
				} else if tc.updateSecret {
					expectedParameters["new-secret-param-key"] = "new-secret-param-value"
				} else if tc.createdWithParamsFromSecret && !tc.deleteParamsFromSecret {
					expectedParameters["secret-param-key"] = "secret-param-value"
				}

				brokerAction := getLastBrokerAction(t, ct.osbClient, fakeosb.UpdateInstance)
				request := brokerAction.Request.(*osb.UpdateInstanceRequest)
				if e, a := expectedParameters, request.Parameters; !reflect.DeepEqual(e, a) {
					t.Fatalf("unexpected parameters: expected %v, got %v", e, a)
				}
			})
		})
	}
}

// TestCreateServiceInstanceWithInvalidParameters tests creating a ServiceInstance
// with invalid parameters.
func TestCreateServiceInstanceWithInvalidParameters(t *testing.T) {
	ct := &controllerTest{
		t:      t,
		broker: getTestBroker(),
	}
	ct.run(func(ct *controllerTest) {
		instance := getTestInstance()
		instance.Spec.Parameters = convertParametersIntoRawExtension(t,
			map[string]interface{}{
				"Name": "test-param",
				"Args": map[string]interface{}{
					"first":  "first-arg",
					"second": "second-arg",
				},
			})
		instance.Spec.Parameters.Raw[0] = 0x21
		if _, err := ct.client.ServiceInstances(instance.Namespace).Create(instance); err == nil {
			t.Fatalf("expected instance to fail to be created due to invalid parameters")
		}
	})
}

// TimeoutError is an error sent back in a url.Error from the broker when
// the request has timed out at the network layer.
type TimeoutError string

// Timeout returns true since TimeoutError indicates that there was a timeout.
// This method is so that TimeoutError implements the url.timeout interface.
func (e TimeoutError) Timeout() bool {
	return true
}

// Error returns the TimeoutError as a string
func (e TimeoutError) Error() string {
	return string(e)
}

// TestCreateServiceInstanceWithProvisionFailure tests creating a ServiceInstance
// with various failure results in response to the provision request.
func TestCreateServiceInstanceWithProvisionFailure(t *testing.T) {
	cases := []struct {
		// name of the test
		name string
		// status code returned by failing provision calls
		statusCode int
		// non-HTTP error returned by failing provision calls
		nonHTTPResponseError error
		// expected reason used in the instance condition to indiciate that the provision failed
		provisionErrorReason string
		// expected reason used in the instance condiiton to indicate that the provision failed terminally
		failReason string
		// true if the failed provision is expected to trigger orphan mitigation
		triggersOrphanMitigation bool
	}{
		{
			name:                 "Status OK",
			statusCode:           http.StatusOK,
			provisionErrorReason: "ProvisionCallFailed",
		},
		{
			name:                     "Status Created",
			statusCode:               http.StatusCreated,
			provisionErrorReason:     "ProvisionCallFailed",
			triggersOrphanMitigation: true,
		},
		{
			name:                     "other 2xx",
			statusCode:               http.StatusNoContent,
			provisionErrorReason:     "ProvisionCallFailed",
			triggersOrphanMitigation: true,
		},
		{
			name:                 "3XX",
			statusCode:           300,
			provisionErrorReason: "ProvisionCallFailed",
		},
		{
			name:                     "Status Request Timeout",
			statusCode:               http.StatusRequestTimeout,
			provisionErrorReason:     "ProvisionCallFailed",
			triggersOrphanMitigation: false,
		},
		{
			name:                 "400",
			statusCode:           400,
			provisionErrorReason: "ProvisionCallFailed",
			failReason:           "ClusterServiceBrokerReturnedFailure",
		},
		{
			name:                 "other 4XX",
			statusCode:           403,
			provisionErrorReason: "ProvisionCallFailed",
		},
		{
			name:                     "5XX",
			statusCode:               500,
			provisionErrorReason:     "ProvisionCallFailed",
			triggersOrphanMitigation: true,
		},
		{
			name:                 "non-url transport error",
			nonHTTPResponseError: fmt.Errorf("non-url error"),
			provisionErrorReason: "ErrorCallingProvision",
		},
		{
			name: "non-timeout url error",
			nonHTTPResponseError: &url.Error{
				Op:  "Put",
				URL: "https://fakebroker.com/v2/service_instances/instance_id",
				Err: fmt.Errorf("non-timeout error"),
			},
			provisionErrorReason: "ErrorCallingProvision",
		},
		{
			name: "network timeout",
			nonHTTPResponseError: &url.Error{
				Op:  "Put",
				URL: "https://fakebroker.com/v2/service_instances/instance_id",
				Err: TimeoutError("timeout error"),
			},
			provisionErrorReason:     "ErrorCallingProvision",
			triggersOrphanMitigation: true,
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {

			provisionSuccessChan := make(chan bool, 2)
			deprovisionSuccessChan := make(chan bool, 2)
			deprovisionBlockChan := make(chan bool, 2)

			// Ensure that broker requests respond successfully after running
			// the core of the test so that the resource cleanup can proceed.
			defer func() {
				provisionSuccessChan <- true
				deprovisionSuccessChan <- true
				deprovisionBlockChan <- false
			}()

			//t.Parallel()
			ct := &controllerTest{
				t:                            t,
				broker:                       getTestBroker(),
				instance:                     getTestInstance(),
				skipVerifyingInstanceSuccess: true,
				setup: func(ct *controllerTest) {
					reactionError := tc.nonHTTPResponseError
					if reactionError == nil {
						reactionError = osb.HTTPStatusCodeError{
							StatusCode:   tc.statusCode,
							ErrorMessage: strPtr("error message"),
							Description:  strPtr("response description"),
						}
					}
					respondSuccessfullyToProvision := false
					ct.osbClient.ProvisionReaction = fakeosb.DynamicProvisionReaction(
						func(r *osb.ProvisionRequest) (*osb.ProvisionResponse, error) {
							select {
							case respondSuccessfullyToProvision = <-provisionSuccessChan:
							default:
							}
							if respondSuccessfullyToProvision {
								return &osb.ProvisionResponse{}, nil
							} else {
								return nil, reactionError
							}
						})
					respondSuccessfullyToDeprovision := false
					blockDeprovision := true
					ct.osbClient.DeprovisionReaction = fakeosb.DynamicDeprovisionReaction(
						func(r *osb.DeprovisionRequest) (*osb.DeprovisionResponse, error) {
							for blockDeprovision {
								blockDeprovision = <-deprovisionBlockChan
							}
							select {
							case respondSuccessfullyToDeprovision = <-deprovisionSuccessChan:
							default:
							}
							if respondSuccessfullyToDeprovision {
								return &osb.DeprovisionResponse{}, nil
							} else {
								return nil, osb.HTTPStatusCodeError{
									StatusCode:   500,
									ErrorMessage: strPtr("temporary deprovision error"),
								}
							}
						})
				},
			}
			ct.run(func(ct *controllerTest) {
				// Wait for the provision to fail
				condition := v1beta1.ServiceInstanceCondition{
					Type:   v1beta1.ServiceInstanceConditionReady,
					Status: v1beta1.ConditionFalse,
					Reason: tc.provisionErrorReason,
				}
				if tc.triggersOrphanMitigation {
					condition.Reason = "StartingInstanceOrphanMitigation"
				}
				if err := util.WaitForInstanceCondition(ct.client, testNamespace, testInstanceName, condition); err != nil {
					t.Fatalf("error waiting for provision to fail: %v", err)
				}

				// Assert that the latest generation has been observed
				instance, err := ct.client.ServiceInstances(testNamespace).Get(testInstanceName, metav1.GetOptions{})
				if err != nil {
					t.Fatalf("error getting instance: %v", err)
				}
				if e, a := int64(1), instance.Status.ObservedGeneration; e != a {
					t.Fatalf("unexpected observed generation: expected %v, got %v", e, a)
				}

				// If the provision failed with a terminating failure
				if tc.failReason != "" {
					util.AssertServiceInstanceCondition(t, instance, v1beta1.ServiceInstanceConditionFailed, v1beta1.ConditionTrue, tc.failReason)
					if e, a := 0, len(findBrokerActions(t, ct.osbClient, fakeosb.DeprovisionInstance)); e != a {
						t.Fatalf("unexpected calls to deprovision instance: expected %v, got %v", e, a)
					}
					return
				}

				// Assert that the orphan mitigation reason was set correctly
				if tc.triggersOrphanMitigation {
					util.AssertServiceInstanceCondition(t, instance, v1beta1.ServiceInstanceConditionOrphanMitigation, v1beta1.ConditionTrue, tc.provisionErrorReason)
					if !instance.Status.OrphanMitigationInProgress {
						t.Fatalf("expected orphan mitigation to be in progress")
					}
				} else {
					util.AssertServiceInstanceConditionFalseOrAbsent(t, instance, v1beta1.ServiceInstanceConditionOrphanMitigation)
					if instance.Status.OrphanMitigationInProgress {
						t.Fatalf("expected orphan mitigation to not be in progress")
					}
				}

				// Now that the provision error conditions have been asserted, allow the broker to send its response to the deprovision request
				deprovisionBlockChan <- false

				if tc.triggersOrphanMitigation {
					// Assert that the ready condition is set to Unknown when the deprovision request fails
					if err := util.WaitForInstanceCondition(ct.client, testNamespace, testInstanceName, v1beta1.ServiceInstanceCondition{
						Type:   v1beta1.ServiceInstanceConditionReady,
						Status: v1beta1.ConditionUnknown,
					}); err != nil {
						t.Fatalf("error waiting for instance deprovision to fail: %v", err)
					}
				}

				// Now that everything surround the failed provision has been asserted, allow provision requests
				// to succeed. Also, allow orphan mitigation to complete by allowing deprovision requests to succeed.
				provisionSuccessChan <- true
				deprovisionSuccessChan <- true

				// Wait for the instance to be provisioned successfully
				if err := util.WaitForInstanceCondition(ct.client, testNamespace, testInstanceName, v1beta1.ServiceInstanceCondition{
					Type:   v1beta1.ServiceInstanceConditionReady,
					Status: v1beta1.ConditionTrue,
				}); err != nil {
					t.Fatalf("error waiting for instance to be provisioned: %v", err)
				}

				// Assert that the observed generation is up-to-date, that orphan mitigation is not in progress,
				// and that the instance is not in a failed state.
				instance, err = ct.client.ServiceInstances(testNamespace).Get(testInstanceName, metav1.GetOptions{})
				if err != nil {
					t.Fatalf("error getting instance: %v", err)
				}
				if g, og := instance.Generation, instance.Status.ObservedGeneration; g != og {
					t.Fatalf("latest generation not observed: generation: %v, observed: %v", g, og)
				}
				if instance.Status.OrphanMitigationInProgress {
					t.Fatalf("unexpected orphan mitigation in progress")
				}
				util.AssertServiceInstanceConditionFalseOrAbsent(t, instance, v1beta1.ServiceInstanceConditionFailed)

				// Assert that the last broker action was a provision-instance request.
				getLastBrokerAction(t, ct.osbClient, fakeosb.ProvisionInstance)
			})
		})
	}
}

func TestCreateServiceInstanceFailsWithNonexistentPlan(t *testing.T) {
	ct := &controllerTest{
		t:                            t,
		broker:                       getTestBroker(),
		instance:                     getTestInstance(),
		skipVerifyingInstanceSuccess: true,
		preCreateInstance: func(ct *controllerTest) {
			otherPlanName := "otherplanname"
			otherPlanID := "other-plan-id"
			catalogResponse := ct.osbClient.CatalogReaction.(*fakeosb.CatalogReaction).Response
			catalogResponse.Services[0].PlanUpdatable = truePtr()
			catalogResponse.Services[0].Plans = []osb.Plan{
				{
					Name:        otherPlanName,
					Free:        truePtr(),
					ID:          otherPlanID,
					Description: "another test plan",
				},
			}

			ct.broker.Spec.RelistRequests++
			if _, err := ct.client.ClusterServiceBrokers().Update(ct.broker); err != nil {
				t.Fatalf("error updating Broker: %v", err)
			}
			if err := util.WaitForClusterServicePlanToExist(ct.client, otherPlanID); err != nil {
				t.Fatalf("error waiting for ClusterServiceClass to exist: %v", err)
			}
			if err := util.WaitForClusterServicePlanToNotExist(ct.client, testPlanExternalID); err != nil {
				t.Fatalf("error waiting for ClusterServiceClass to not exist: %v", err)
			}

		},
	}
	ct.run(func(ct *controllerTest) {
		condition := v1beta1.ServiceInstanceCondition{
			Type:   v1beta1.ServiceInstanceConditionReady,
			Status: v1beta1.ConditionFalse,
			Reason: "ReferencesNonexistentServicePlan",
		}
		if err := util.WaitForInstanceCondition(ct.client, testNamespace, testInstanceName, condition); err != nil {
			t.Fatalf("error waiting for instance condition: %v", err)
		}
	})
}

func TestCreateServiceInstanceAsynchronous(t *testing.T) {
	dashURL := testDashboardURL
	key := osb.OperationKey(testOperation)

	cases := []struct {
		name        string
		osbResponse *osb.ProvisionResponse
	}{
		{
			name: "asynchronous provision with operation key",
			osbResponse: &osb.ProvisionResponse{
				Async:        true,
				DashboardURL: &dashURL,
				OperationKey: &key,
			},
		},
		{
			name: "asynchronous provision without operation key",
			osbResponse: &osb.ProvisionResponse{
				Async:        true,
				DashboardURL: &dashURL,
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ct := &controllerTest{
				t:        t,
				broker:   getTestBroker(),
				instance: getTestInstance(),
				setup: func(ct *controllerTest) {
					ct.osbClient.ProvisionReaction = &fakeosb.ProvisionReaction{
						Response: tc.osbResponse,
					}
				},
			}
			ct.run(func(ct *controllerTest) {
				// the action sequence is GetCatalog, ProvisionInstance, PollLastOperation
				osbActions := ct.osbClient.Actions()
				if tc.osbResponse.OperationKey != nil {
					lastOpRequest := osbActions[len(osbActions)-1].Request.(*osb.LastOperationRequest)
					if lastOpRequest.OperationKey == nil {
						t.Fatal("OperationKey should not be nil")
					} else if e, a := key, *(osbActions[len(osbActions)-1].Request.(*osb.LastOperationRequest).OperationKey); e != a {
						t.Fatalf("unexpected OperationKey: expected %v, got %v", e, a)
					}
				} else {
					if a := osbActions[len(osbActions)-1].Request.(*osb.LastOperationRequest).OperationKey; a != nil {
						t.Fatalf("unexpected OperationKey: expected nil, got %v", a)
					}
				}

				condition := v1beta1.ServiceInstanceCondition{
					Type:   v1beta1.ServiceInstanceConditionReady,
					Status: v1beta1.ConditionTrue,
					Reason: "ProvisionedSuccessfully",
				}
				if err := util.WaitForInstanceCondition(ct.client, testNamespace, testInstanceName, condition); err != nil {
					t.Fatalf("error waiting for instance condition: %v", err)
				}
			})
		})
	}
}

func TestDeleteServiceInstance(t *testing.T) {
	key := osb.OperationKey(testOperation)

	cases := []struct {
		name                         string
		skipVerifyingInstanceSuccess bool
		binding                      *v1beta1.ServiceBinding
		setup                        func(*controllerTest)
		testFunction                 func(t *controllerTest)
	}{
		{
			name:    "synchronous deprovision",
			binding: getTestBinding(),
			setup: func(ct *controllerTest) {
				ct.osbClient.DeprovisionReaction = &fakeosb.DeprovisionReaction{
					Response: &osb.DeprovisionResponse{},
				}
			},
		},
		{
			name: "synchronous deprovision, no binding",
			setup: func(ct *controllerTest) {
				ct.osbClient.DeprovisionReaction = &fakeosb.DeprovisionReaction{
					Response: &osb.DeprovisionResponse{},
				}
			},
		},
		{
			name:    "asynchronous deprovision with operation key",
			binding: getTestBinding(),
			setup: func(ct *controllerTest) {
				ct.osbClient.DeprovisionReaction = &fakeosb.DeprovisionReaction{
					Response: &osb.DeprovisionResponse{
						Async:        true,
						OperationKey: &key,
					},
				}
			},
		},
		{
			name: "asynchronous deprovision with operation key, no binding",
			setup: func(ct *controllerTest) {
				ct.osbClient.DeprovisionReaction = &fakeosb.DeprovisionReaction{
					Response: &osb.DeprovisionResponse{
						Async:        true,
						OperationKey: &key,
					},
				}
			},
		},
		{
			name:    "asynchronous deprovision without operation key",
			binding: getTestBinding(),
			setup: func(ct *controllerTest) {
				ct.osbClient.DeprovisionReaction = &fakeosb.DeprovisionReaction{
					Response: &osb.DeprovisionResponse{
						Async: true,
					},
				}
			},
		},
		{
			name: "asynchronous deprovision without operation key, no binding",
			setup: func(ct *controllerTest) {
				ct.osbClient.DeprovisionReaction = &fakeosb.DeprovisionReaction{
					Response: &osb.DeprovisionResponse{
						Async: true,
					},
				}
			},
		},
		{
			name:    "deprovision instance with binding not deleted",
			binding: getTestBinding(),
			setup: func(ct *controllerTest) {
				ct.osbClient.DeprovisionReaction = &fakeosb.DeprovisionReaction{
					Response: &osb.DeprovisionResponse{},
				}
			},
			testFunction: func(ct *controllerTest) {
				if err := ct.client.ServiceInstances(ct.instance.Namespace).Delete(ct.instance.Name, &metav1.DeleteOptions{}); err != nil {
					ct.t.Fatalf("instance delete should have been accepted: %v", err)
				}

				condition := v1beta1.ServiceInstanceCondition{
					Type:   v1beta1.ServiceInstanceConditionReady,
					Status: v1beta1.ConditionFalse,
					Reason: "DeprovisionBlockedByExistingCredentials",
				}
				if err := util.WaitForInstanceCondition(ct.client, testNamespace, testInstanceName, condition); err != nil {
					ct.t.Fatalf("error waiting for instance condition: %v", err)
				}
				// instance can't be deleted later, as we've
				// already started deleting it now.  Instance is
				// still in "deleted" mode at the end, the
				// reconciler will pick it up and delete
				// it. Thus we should null it out before the
				// test runner goes and tries to do automated
				// cleanup.
				ct.instance = nil
			},
		},
		{
			name: "deprovision instance after in progress provision",
			skipVerifyingInstanceSuccess: true,
			setup: func(ct *controllerTest) {
				ct.osbClient.PollLastOperationReaction = fakeosb.DynamicPollLastOperationReaction(
					getLastOperationResponseByPollCountReactions(2, []fakeosb.PollLastOperationReaction{
						fakeosb.PollLastOperationReaction{
							Response: &osb.LastOperationResponse{
								State: osb.StateInProgress,
							},
						},
						fakeosb.PollLastOperationReaction{
							Response: &osb.LastOperationResponse{
								State: osb.StateSucceeded,
							},
						},
					}))
				ct.osbClient.ProvisionReaction = &fakeosb.ProvisionReaction{
					Response: &osb.ProvisionResponse{
						Async: true,
					},
				}
				ct.osbClient.DeprovisionReaction = &fakeosb.DeprovisionReaction{
					Response: &osb.DeprovisionResponse{},
				}
			},
			testFunction: func(ct *controllerTest) {
				verifyCondition := v1beta1.ServiceInstanceCondition{
					Type:   v1beta1.ServiceInstanceConditionReady,
					Status: v1beta1.ConditionTrue,
					Reason: "ProvisionedSuccessfully",
				}
				if err := util.WaitForInstanceCondition(ct.client, testNamespace, testInstanceName, verifyCondition); err != nil {
					t.Fatalf("error waiting for instance condition: %v", err)
				}
			},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			//t.Parallel()
			ct := &controllerTest{
				t:                            t,
				broker:                       getTestBroker(),
				binding:                      tc.binding,
				instance:                     getTestInstance(),
				skipVerifyingInstanceSuccess: tc.skipVerifyingInstanceSuccess,
				setup: tc.setup,
			}
			ct.run(tc.testFunction)
		})
	}
}

func TestPollServiceInstanceLastOperationSuccess(t *testing.T) {
	cases := []struct {
		name                         string
		setup                        func(t *controllerTest)
		skipVerifyingInstanceSuccess bool
		verifyCondition              *v1beta1.ServiceInstanceCondition
		preDeleteBroker              func(t *controllerTest)
		preCreateInstance            func(t *controllerTest)
		postCreateInstance           func(t *controllerTest)
	}{
		{
			name: "async provisioning with last operation response state in progress",
			setup: func(ct *controllerTest) {
				ct.osbClient.ProvisionReaction = &fakeosb.ProvisionReaction{
					Response: &osb.ProvisionResponse{
						Async: true,
					},
				}
				ct.osbClient.PollLastOperationReaction = fakeosb.DynamicPollLastOperationReaction(
					getLastOperationResponseByPollCountStates(2, []osb.LastOperationState{osb.StateInProgress, osb.StateSucceeded}))
			},
			skipVerifyingInstanceSuccess: true,
			verifyCondition: &v1beta1.ServiceInstanceCondition{
				Type:   v1beta1.ServiceInstanceConditionReady,
				Status: v1beta1.ConditionTrue,
				Reason: "ProvisionedSuccessfully",
			},
		},
		{
			name: "async provisioning with last operation response state succeeded",
			setup: func(ct *controllerTest) {
				ct.osbClient.ProvisionReaction = &fakeosb.ProvisionReaction{
					Response: &osb.ProvisionResponse{
						Async: true,
					},
				}
				ct.osbClient.PollLastOperationReaction = &fakeosb.PollLastOperationReaction{
					Response: &osb.LastOperationResponse{
						State:       osb.StateSucceeded,
						Description: strPtr("testDescription"),
					},
				}
			},
			verifyCondition: &v1beta1.ServiceInstanceCondition{
				Type:   v1beta1.ServiceInstanceConditionReady,
				Status: v1beta1.ConditionTrue,
				Reason: "ProvisionedSuccessfully",
			},
		},
		// response errors
		{
			name: "async provisioning with error on first poll",
			setup: func(ct *controllerTest) {
				ct.osbClient.ProvisionReaction = &fakeosb.ProvisionReaction{
					Response: &osb.ProvisionResponse{
						Async: true,
					},
				}
				ct.osbClient.PollLastOperationReaction = fakeosb.DynamicPollLastOperationReaction(
					getLastOperationResponseByPollCountReactions(2, []fakeosb.PollLastOperationReaction{
						fakeosb.PollLastOperationReaction{
							Error: errors.New("some error"),
						},
						fakeosb.PollLastOperationReaction{
							Response: &osb.LastOperationResponse{
								State: osb.StateSucceeded,
							},
						},
					}))
			},
			skipVerifyingInstanceSuccess: true,
			verifyCondition: &v1beta1.ServiceInstanceCondition{
				Type:   v1beta1.ServiceInstanceConditionReady,
				Status: v1beta1.ConditionTrue,
				Reason: "ProvisionedSuccessfully",
			},
		},
		{
			name: "async provisioning with error on second poll",
			setup: func(ct *controllerTest) {
				ct.osbClient.ProvisionReaction = &fakeosb.ProvisionReaction{
					Response: &osb.ProvisionResponse{
						Async: true,
					},
				}
				ct.osbClient.PollLastOperationReaction = fakeosb.DynamicPollLastOperationReaction(
					getLastOperationResponseByPollCountReactions(1, []fakeosb.PollLastOperationReaction{
						fakeosb.PollLastOperationReaction{
							Response: &osb.LastOperationResponse{
								State: osb.StateInProgress,
							},
						},
						fakeosb.PollLastOperationReaction{
							Error: errors.New("some error"),
						},
						fakeosb.PollLastOperationReaction{
							Response: &osb.LastOperationResponse{
								State: osb.StateSucceeded,
							},
						},
					}))
			},
			skipVerifyingInstanceSuccess: true,
			verifyCondition: &v1beta1.ServiceInstanceCondition{
				Type:   v1beta1.ServiceInstanceConditionReady,
				Status: v1beta1.ConditionTrue,
				Reason: "ProvisionedSuccessfully",
			},
		},
		{
			name: "async last operation response successful with originating identity",
			setup: func(ct *controllerTest) {
				if err := utilfeature.DefaultFeatureGate.Set(fmt.Sprintf("%v=true", scfeatures.OriginatingIdentity)); err != nil {
					t.Fatalf("Failed to enable originating identity feature: %v", err)
				}

				ct.osbClient.ProvisionReaction = &fakeosb.ProvisionReaction{
					Response: &osb.ProvisionResponse{
						Async: true,
					},
				}
				ct.osbClient.PollLastOperationReaction = &fakeosb.PollLastOperationReaction{
					Response: &osb.LastOperationResponse{
						State:       osb.StateSucceeded,
						Description: strPtr("testDescription"),
					},
				}
			},
			verifyCondition: &v1beta1.ServiceInstanceCondition{
				Type:   v1beta1.ServiceInstanceConditionReady,
				Status: v1beta1.ConditionTrue,
				Reason: "ProvisionedSuccessfully",
			},
			preCreateInstance: func(ct *controllerTest) {
				catalogClient, err := changeUsernameForCatalogClient(ct.catalogClient, ct.catalogClientConfig, "instance-creator")
				if err != nil {
					t.Fatalf("could not change the username for the catalog client: %v", err)
				}
				ct.catalogClient = catalogClient
				ct.client = catalogClient.ServicecatalogV1beta1()

			},
			postCreateInstance: func(ct *controllerTest) {
				verifyUsernameInLastBrokerAction(ct.t, ct.osbClient, fakeosb.PollLastOperation, "instance-creator")
			},
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			//t.Parallel()
			ct := &controllerTest{
				t:                            t,
				broker:                       getTestBroker(),
				instance:                     getTestInstance(),
				skipVerifyingInstanceSuccess: tc.skipVerifyingInstanceSuccess,
				setup:              tc.setup,
				preDeleteBroker:    tc.preDeleteBroker,
				preCreateInstance:  tc.preCreateInstance,
				postCreateInstance: tc.postCreateInstance,
			}
			ct.run(func(ct *controllerTest) {
				if tc.verifyCondition != nil {
					if err := util.WaitForInstanceCondition(ct.client, testNamespace, testInstanceName, *tc.verifyCondition); err != nil {
						t.Fatalf("error waiting for instance condition: %v", err)
					}
				}
			})
		})
	}
}

// TestPollServiceInstanceLastOperationFailure checks that async operation is correctly
// retried after the initial operation fails
func TestPollServiceInstanceLastOperationFailure(t *testing.T) {
	cases := []struct {
		name                         string
		setup                        func(t *controllerTest)
		skipVerifyingInstanceSuccess bool
		failureCondition             *v1beta1.ServiceInstanceCondition
		successCondition             *v1beta1.ServiceInstanceCondition
	}{
		{
			name: "async provisioning with last operation response state failed",
			setup: func(ct *controllerTest) {
				ct.osbClient.ProvisionReaction = &fakeosb.ProvisionReaction{
					Response: &osb.ProvisionResponse{
						Async: true,
					},
				}
				ct.osbClient.PollLastOperationReaction = fakeosb.DynamicPollLastOperationReaction(
					getLastOperationResponseByPollCountStates(2,
						[]osb.LastOperationState{
							osb.StateFailed,
							osb.StateSucceeded,
						}))
			},
			skipVerifyingInstanceSuccess: false,
			failureCondition: &v1beta1.ServiceInstanceCondition{
				Type:   v1beta1.ServiceInstanceConditionReady,
				Status: v1beta1.ConditionFalse,
				Reason: "ProvisionCallFailed",
			},
			successCondition: &v1beta1.ServiceInstanceCondition{
				Type:   v1beta1.ServiceInstanceConditionReady,
				Status: v1beta1.ConditionTrue,
				Reason: "ProvisionedSuccessfully",
			},
		},
		// response errors
		{
			name: "async provisioning with last operation response state failed eventually",
			setup: func(ct *controllerTest) {
				ct.osbClient.ProvisionReaction = &fakeosb.ProvisionReaction{
					Response: &osb.ProvisionResponse{
						Async: true,
					},
				}
				ct.osbClient.PollLastOperationReaction = fakeosb.DynamicPollLastOperationReaction(
					getLastOperationResponseByPollCountStates(1,
						[]osb.LastOperationState{
							osb.StateInProgress,
							osb.StateFailed,
							osb.StateInProgress,
							osb.StateSucceeded,
						}))
			},
			skipVerifyingInstanceSuccess: false,
			failureCondition: &v1beta1.ServiceInstanceCondition{
				Type:   v1beta1.ServiceInstanceConditionReady,
				Status: v1beta1.ConditionFalse,
				Reason: "ProvisionCallFailed",
			},
			successCondition: &v1beta1.ServiceInstanceCondition{
				Type:   v1beta1.ServiceInstanceConditionReady,
				Status: v1beta1.ConditionTrue,
				Reason: "ProvisionedSuccessfully",
			},
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			//t.Parallel()
			ct := &controllerTest{
				t:                            t,
				broker:                       getTestBroker(),
				instance:                     getTestInstance(),
				skipVerifyingInstanceSuccess: tc.skipVerifyingInstanceSuccess,
				setup: tc.setup,
			}
			ct.run(func(ct *controllerTest) {
				if err := util.WaitForInstanceCondition(ct.client, testNamespace, testInstanceName, *tc.successCondition); err != nil {
					t.Fatalf("error waiting for instance condition: %v", err)
				}
			})
		})
	}
}

// TestRetryAsyncDeprovision tests whether asynchronous deprovisioning retries
// by attempting a new deprovision after failing.
func TestRetryAsyncDeprovision(t *testing.T) {
	hasPollFailed := false
	ct := &controllerTest{
		t:        t,
		broker:   getTestBroker(),
		instance: getTestInstance(),
		setup: func(ct *controllerTest) {
			ct.osbClient.DeprovisionReaction = fakeosb.DynamicDeprovisionReaction(
				func(_ *osb.DeprovisionRequest) (*osb.DeprovisionResponse, error) {
					response := &osb.DeprovisionResponse{Async: true}
					if hasPollFailed {
						response.Async = false
					}
					return response, nil
				})

			ct.osbClient.PollLastOperationReaction = fakeosb.DynamicPollLastOperationReaction(
				func(_ *osb.LastOperationRequest) (*osb.LastOperationResponse, error) {
					hasPollFailed = true
					return &osb.LastOperationResponse{
						State: osb.StateFailed,
					}, nil
				})
		},
	}
	ct.run(func(_ *controllerTest) {})
}
