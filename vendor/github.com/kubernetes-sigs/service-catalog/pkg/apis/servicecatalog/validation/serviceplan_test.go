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

package validation

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubernetes-sigs/service-catalog/pkg/apis/servicecatalog"
)

func validClusterServicePlan() *servicecatalog.ClusterServicePlan {
	return &servicecatalog.ClusterServicePlan{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-clusterserviceplan",
		},
		Spec: servicecatalog.ClusterServicePlanSpec{
			CommonServicePlanSpec: servicecatalog.CommonServicePlanSpec{
				ExternalName: "test-clusterserviceplan",
				ExternalID:   "40d-0983-1b89",
				Description:  "plan description",
			},
			ClusterServiceBrokerName: "test-clusterservicebroker",
			ClusterServiceClassRef: servicecatalog.ClusterObjectReference{
				Name: "test-service-class",
			},
		},
	}
}

func validServicePlan() *servicecatalog.ServicePlan {
	return &servicecatalog.ServicePlan{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-clusterserviceplan",
			Namespace: "test-ns",
		},
		Spec: servicecatalog.ServicePlanSpec{
			CommonServicePlanSpec: servicecatalog.CommonServicePlanSpec{
				ExternalName: "test-clusterserviceplan",
				ExternalID:   "40d-0983-1b89",
				Description:  "plan description",
			},
			ServiceBrokerName: "test-clusterservicebroker",
			ServiceClassRef: servicecatalog.LocalObjectReference{
				Name: "test-service-class",
			},
		},
	}
}

func TestValidateClusterServicePlan(t *testing.T) {
	testCases := []struct {
		name               string
		clusterServicePlan *servicecatalog.ClusterServicePlan
		valid              bool
	}{
		{
			name:               "valid ClusterServicePlan",
			clusterServicePlan: validClusterServicePlan(),
			valid:              true,
		},
		{
			name: "valid ClusterServicePlan - period in externalName",
			clusterServicePlan: func() *servicecatalog.ClusterServicePlan {
				s := validClusterServicePlan()
				s.Spec.ExternalName = "test.plan"
				return s
			}(),
			valid: true,
		},
		{
			name: "missing name",
			clusterServicePlan: func() *servicecatalog.ClusterServicePlan {
				s := validClusterServicePlan()
				s.Name = ""
				return s
			}(),
			valid: false,
		},
		{
			name: "mixed case Name",
			clusterServicePlan: func() *servicecatalog.ClusterServicePlan {
				s := validClusterServicePlan()
				s.Name = "abcXYZ"
				return s
			}(),
			valid: true,
		},
		{
			name: "mixed case externalName",
			clusterServicePlan: func() *servicecatalog.ClusterServicePlan {
				s := validClusterServicePlan()
				s.Spec.ExternalName = "abcXYZ"
				return s
			}(),
			valid: true,
		},
		{
			name: "missing clusterServiceBrokerName",
			clusterServicePlan: func() *servicecatalog.ClusterServicePlan {
				s := validClusterServicePlan()
				s.Spec.ClusterServiceBrokerName = ""
				return s
			}(),
			valid: false,
		},
		{
			name: "missing externalName",
			clusterServicePlan: func() *servicecatalog.ClusterServicePlan {
				s := validClusterServicePlan()
				s.Spec.ExternalName = ""
				return s
			}(),
			valid: false,
		},
		{
			name: "missing external id",
			clusterServicePlan: func() *servicecatalog.ClusterServicePlan {
				s := validClusterServicePlan()
				s.Spec.ExternalID = ""
				return s
			}(),
			valid: false,
		},
		{
			// Note this is NOT due to the spec, this is due to
			// a Kubernetes limitation. So, technically this restriction
			// could cause a valid Broker to not work against Kube.
			name: "external id too long",
			clusterServicePlan: func() *servicecatalog.ClusterServicePlan {
				s := validClusterServicePlan()
				s.Spec.ExternalID = "1234567890123456789012345678901234567890123456789012345678901234"
				return s
			}(),
			valid: false,
		},
		{
			name: "missing description",
			clusterServicePlan: func() *servicecatalog.ClusterServicePlan {
				s := validClusterServicePlan()
				s.Spec.Description = ""
				return s
			}(),
			valid: false,
		},
		{
			name: "missing serviceclass reference",
			clusterServicePlan: func() *servicecatalog.ClusterServicePlan {
				s := validClusterServicePlan()
				s.Spec.ClusterServiceClassRef.Name = ""
				return s
			}(),
			valid: false,
		},
	}
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			errs := ValidateClusterServicePlan(tc.clusterServicePlan)
			t.Log(errs)
			if len(errs) != 0 && tc.valid {
				t.Errorf("unexpected error: %v", errs)
			} else if len(errs) == 0 && !tc.valid {
				t.Error("unexpected success")
			}
		})
	}
}

func TestValidateServicePlan(t *testing.T) {
	testCases := []struct {
		name        string
		servicePlan *servicecatalog.ServicePlan
		valid       bool
	}{
		{
			name:        "valid ServicePlan",
			servicePlan: validServicePlan(),
			valid:       true,
		},
		{
			name: "valid ServicePlan - period in externalName",
			servicePlan: func() *servicecatalog.ServicePlan {
				s := validServicePlan()
				s.Spec.ExternalName = "test.plan"
				return s
			}(),
			valid: true,
		},
		{
			name: "missing name",
			servicePlan: func() *servicecatalog.ServicePlan {
				s := validServicePlan()
				s.Name = ""
				return s
			}(),
			valid: false,
		},
		{
			name: "mixed case Name",
			servicePlan: func() *servicecatalog.ServicePlan {
				s := validServicePlan()
				s.Name = "abcXYZ"
				return s
			}(),
			valid: true,
		},
		{
			name: "mixed case externalName",
			servicePlan: func() *servicecatalog.ServicePlan {
				s := validServicePlan()
				s.Spec.ExternalName = "abcXYZ"
				return s
			}(),
			valid: true,
		},
		{
			name: "missing clusterServiceBrokerName",
			servicePlan: func() *servicecatalog.ServicePlan {
				s := validServicePlan()
				s.Spec.ServiceBrokerName = ""
				return s
			}(),
			valid: false,
		},
		{
			name: "missing externalName",
			servicePlan: func() *servicecatalog.ServicePlan {
				s := validServicePlan()
				s.Spec.ExternalName = ""
				return s
			}(),
			valid: false,
		},
		{
			name: "missing external id",
			servicePlan: func() *servicecatalog.ServicePlan {
				s := validServicePlan()
				s.Spec.ExternalID = ""
				return s
			}(),
			valid: false,
		},
		{
			// Note this is NOT due to the spec, this is due to
			// a Kubernetes limitation. So, technically this restriction
			// could cause a valid Broker to not work against Kube.
			name: "external id too long",
			servicePlan: func() *servicecatalog.ServicePlan {
				s := validServicePlan()
				s.Spec.ExternalID = "1234567890123456789012345678901234567890123456789012345678901234"
				return s
			}(),
			valid: false,
		},
		{
			name: "missing description",
			servicePlan: func() *servicecatalog.ServicePlan {
				s := validServicePlan()
				s.Spec.Description = ""
				return s
			}(),
			valid: false,
		},
		{
			name: "missing serviceclass reference",
			servicePlan: func() *servicecatalog.ServicePlan {
				s := validServicePlan()
				s.Spec.ServiceClassRef.Name = ""
				return s
			}(),
			valid: false,
		},
		{
			name: "missing namespace",
			servicePlan: func() *servicecatalog.ServicePlan {
				s := validServicePlan()
				s.ObjectMeta = metav1.ObjectMeta{
					Name: "test-clusterserviceplan",
				}
				return s
			}(),
			valid: false,
		},
	}
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			errs := ValidateServicePlan(tc.servicePlan)
			t.Log(errs)
			if len(errs) != 0 && tc.valid {
				t.Errorf("unexpected error: %v", errs)
			} else if len(errs) == 0 && !tc.valid {
				t.Error("unexpected success")
			}
		})
	}
}

func TestValidateClusterServicePlanUpdate(t *testing.T) {
	testCases := []struct {
		name  string
		old   *servicecatalog.ClusterServicePlan
		new   *servicecatalog.ClusterServicePlan
		valid bool
	}{
		{
			name:  "valid servicePlan update same content",
			old:   validClusterServicePlan(),
			new:   validClusterServicePlan(),
			valid: true,
		},
		{
			name: "valid servicePlan update different content",
			old:  validClusterServicePlan(),
			new: func() *servicecatalog.ClusterServicePlan {
				s := validClusterServicePlan()
				s.Spec.Description = "a new description cause it changed"
				return s
			}(),
			valid: true,
		},
		{
			name: "servicePlan changing external ID",
			old:  validClusterServicePlan(),
			new: func() *servicecatalog.ClusterServicePlan {
				s := validClusterServicePlan()
				s.Spec.ExternalID = "something-else"
				return s
			}(),
			valid: false,
		},
	}
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			errs := ValidateClusterServicePlanUpdate(tc.new, tc.old)
			t.Log(errs)
			if len(errs) != 0 && tc.valid {
				t.Errorf("unexpected error: %v", errs)
			} else if len(errs) == 0 && !tc.valid {
				t.Error("unexpected success")
			}
		})
	}
}

func TestValidateServicePlanUpdate(t *testing.T) {
	testCases := []struct {
		name  string
		old   *servicecatalog.ServicePlan
		new   *servicecatalog.ServicePlan
		valid bool
	}{
		{
			name:  "valid servicePlan update same content",
			old:   validServicePlan(),
			new:   validServicePlan(),
			valid: true,
		},
		{
			name: "valid servicePlan update different content",
			old:  validServicePlan(),
			new: func() *servicecatalog.ServicePlan {
				s := validServicePlan()
				s.Spec.Description = "a new description cause it changed"
				return s
			}(),
			valid: true,
		},
		{
			name: "servicePlan changing external ID",
			old:  validServicePlan(),
			new: func() *servicecatalog.ServicePlan {
				s := validServicePlan()
				s.Spec.ExternalID = "something-else"
				return s
			}(),
			valid: false,
		},
	}
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			errs := ValidateServicePlanUpdate(tc.new, tc.old)
			t.Log(errs)
			if len(errs) != 0 && tc.valid {
				t.Errorf("unexpected error: %v", errs)
			} else if len(errs) == 0 && !tc.valid {
				t.Error("unexpected success")
			}
		})
	}
}
