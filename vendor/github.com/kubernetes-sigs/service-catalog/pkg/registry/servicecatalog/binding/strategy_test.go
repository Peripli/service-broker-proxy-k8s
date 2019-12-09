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

package binding

import (
	"fmt"
	"testing"

	sctestutil "github.com/kubernetes-sigs/service-catalog/test/util"
	utilfeature "k8s.io/apiserver/pkg/util/feature"

	servicecatalog "github.com/kubernetes-sigs/service-catalog/pkg/apis/servicecatalog/v1beta1"
	scfeatures "github.com/kubernetes-sigs/service-catalog/pkg/features"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func getTestInstanceCredential() *servicecatalog.ServiceBinding {
	return &servicecatalog.ServiceBinding{
		ObjectMeta: metav1.ObjectMeta{
			Generation: 1,
		},
		Spec: servicecatalog.ServiceBindingSpec{
			InstanceRef: servicecatalog.LocalObjectReference{
				Name: "some-string",
			},
		},
		Status: servicecatalog.ServiceBindingStatus{
			Conditions: []servicecatalog.ServiceBindingCondition{
				{
					Type:   servicecatalog.ServiceBindingConditionReady,
					Status: servicecatalog.ConditionTrue,
				},
			},
		},
	}
}

// TODO: Un-comment "spec-change" test case when there is a field
// in the spec to which the reconciler allows a change.

// TestInstanceCredentialUpdate tests that generation is incremented correctly when the
// spec of a ServiceBinding is updated.
func TestInstanceCredentialUpdate(t *testing.T) {
	cases := []struct {
		name                      string
		older                     *servicecatalog.ServiceBinding
		newer                     *servicecatalog.ServiceBinding
		shouldGenerationIncrement bool
	}{
		{
			name:  "no spec change",
			older: getTestInstanceCredential(),
			newer: getTestInstanceCredential(),
		},
		//		{
		//			name:  "spec change",
		//			older: getTestInstanceCredential(),
		//			newer: func() *v1beta1.ServiceBinding {
		//				ic := getTestInstanceCredential()
		//				ic.Spec.InstanceRef = servicecatalog.LocalObjectReference{
		//					Name: "new-string",
		//				}
		//				return ic
		//			},
		//			shouldGenerationIncrement: true,
		//		},
	}
	creatorUserName := "creator"
	createContext := sctestutil.ContextWithUserName(creatorUserName)
	for _, tc := range cases {
		bindingRESTStrategies.PrepareForUpdate(createContext, tc.newer, tc.older)

		expectedGeneration := tc.older.Generation
		if tc.shouldGenerationIncrement {
			expectedGeneration = expectedGeneration + 1
		}
		if e, a := expectedGeneration, tc.newer.Generation; e != a {
			t.Errorf("%v: expected %v, got %v for generation", tc.name, e, a)
		}
	}
}

// TestInstanceCredentialUserInfo tests that the user info is set properly
// as the user changes for different modifications of the instance credential.
func TestInstanceCredentialUserInfo(t *testing.T) {
	// Enable the OriginatingIdentity feature
	prevOrigIDEnablement := sctestutil.EnableOriginatingIdentity(t, true)
	defer utilfeature.DefaultMutableFeatureGate.Set(fmt.Sprintf("%v=%v", scfeatures.OriginatingIdentity, prevOrigIDEnablement))

	creatorUserName := "creator"
	createdInstanceCredential := getTestInstanceCredential()
	createContext := sctestutil.ContextWithUserName(creatorUserName)
	bindingRESTStrategies.PrepareForCreate(createContext, createdInstanceCredential)

	if e, a := creatorUserName, createdInstanceCredential.Spec.UserInfo.Username; e != a {
		t.Errorf("unexpected user info in created spec: expected %q, got %q", e, a)
	}

	// TODO: Un-comment the following portion of this test when there is a field
	// in the spec to which the reconciler allows a change.

	//  updaterUserName := "updater"
	//	updatedInstanceCredential := getTestInstanceCredential()
	//	updateContext := sctestutil.ContextWithUserName(updaterUserName)
	//	bindingRESTStrategies.PrepareForUpdate(updateContext, updatedInstanceCredential, createdInstanceCredential)

	//	if e, a := updaterUserName, updatedInstanceCredential.Spec.UserInfo.Username; e != a {
	//		t.Errorf("unexpected user info in updated spec: expected %q, got %q", e, a)
	//	}

	deleterUserName := "deleter"
	deletedInstanceCredential := getTestInstanceCredential()
	deleteContext := sctestutil.ContextWithUserName(deleterUserName)
	bindingRESTStrategies.CheckGracefulDelete(deleteContext, deletedInstanceCredential, nil)

	if e, a := deleterUserName, deletedInstanceCredential.Spec.UserInfo.Username; e != a {
		t.Errorf("unexpected user info in deleted spec: expected %q, got %q", e, a)
	}
}

// TestExternalIDSet checks that we set the ExternalID if the user doesn't provide it.
func TestExternalIDSet(t *testing.T) {
	createdInstanceCredential := getTestInstanceCredential()
	creatorUserName := "creator"
	createContext := sctestutil.ContextWithUserName(creatorUserName)
	bindingRESTStrategies.PrepareForCreate(createContext, createdInstanceCredential)

	if createdInstanceCredential.Spec.ExternalID == "" {
		t.Error("Expected an ExternalID to be set, but got none")
	}
}

// TestExternalIDUserProvided makes sure we don't modify a user-specified ExternalID.
func TestExternalIDUserProvided(t *testing.T) {
	userExternalID := "my-id"
	createdInstanceCredential := getTestInstanceCredential()
	createdInstanceCredential.Spec.ExternalID = userExternalID
	creatorUserName := "creator"
	createContext := sctestutil.ContextWithUserName(creatorUserName)
	bindingRESTStrategies.PrepareForCreate(createContext, createdInstanceCredential)

	if createdInstanceCredential.Spec.ExternalID != userExternalID {
		t.Errorf("Modified user provided ExternalID to %q", createdInstanceCredential.Spec.ExternalID)
	}
}
