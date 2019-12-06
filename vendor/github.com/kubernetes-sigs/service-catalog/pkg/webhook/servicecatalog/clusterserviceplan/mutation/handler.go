/*
Copyright 2019 The Kubernetes Authors.

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

package mutation

import (
	"context"
	"encoding/json"
	"net/http"

	sc "github.com/kubernetes-sigs/service-catalog/pkg/apis/servicecatalog/v1beta1"
	"github.com/kubernetes-sigs/service-catalog/pkg/util"
	webhookutil "github.com/kubernetes-sigs/service-catalog/pkg/webhookutil"

	admissionTypes "k8s.io/api/admission/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// CreateUpdateHandler handles ClusterServicePlann
type CreateUpdateHandler struct {
	decoder *admission.Decoder
}

var _ admission.Handler = &CreateUpdateHandler{}
var _ admission.DecoderInjector = &CreateUpdateHandler{}

// Handle handles admission requests.
func (h *CreateUpdateHandler) Handle(ctx context.Context, req admission.Request) admission.Response {
	traced := webhookutil.NewTracedLogger(req.UID)
	traced.Infof("Start handling mutation  operation: %s for %s: %q", req.Operation, req.Kind.Kind, req.Name)

	cb := &sc.ClusterServicePlan{}
	if err := webhookutil.MatchKinds(cb, req.Kind); err != nil {
		traced.Errorf("Error matching kinds: %v", err)
		return admission.Errored(http.StatusBadRequest, err)
	}

	if err := h.decoder.Decode(req, cb); err != nil {
		traced.Errorf("Could not decode request object: %v", err)
		return admission.Errored(http.StatusBadRequest, err)
	}

	mutated := cb.DeepCopy()
	switch req.Operation {
	case admissionTypes.Create:
		h.mutateOnCreate(ctx, mutated)
	case admissionTypes.Update:
		oldObj := &sc.ClusterServicePlan{}
		if err := h.decoder.DecodeRaw(req.OldObject, oldObj); err != nil {
			traced.Errorf("Could not decode request old object: %v", err)
			return admission.Errored(http.StatusBadRequest, err)
		}
		h.mutateOnUpdate(ctx, oldObj, mutated)
	default:
		traced.Infof("ClusterServicePlan mutation wehbook does not support action %q", req.Operation)
		return admission.Allowed("action not taken")
	}
	h.syncLabels(mutated)
	rawMutated, err := json.Marshal(mutated)
	if err != nil {
		traced.Errorf("Error marshaling mutated object: %v", err)
		return admission.Errored(http.StatusInternalServerError, err)
	}

	traced.Infof("Completed successfully mutation operation: %s for %s: %q", req.Operation, req.Kind.Kind, req.Name)
	return admission.PatchResponseFromRaw(req.Object.Raw, rawMutated)
}

// InjectDecoder injects the decoder
func (h *CreateUpdateHandler) InjectDecoder(d *admission.Decoder) error {
	h.decoder = d
	return nil
}

func (h *CreateUpdateHandler) mutateOnCreate(ctx context.Context, binding *sc.ClusterServicePlan) {
}

func (h *CreateUpdateHandler) mutateOnUpdate(ctx context.Context, oldClusterServicePlan, newClusterServicePlan *sc.ClusterServicePlan) {
	newClusterServicePlan.Spec.ClusterServiceClassRef = oldClusterServicePlan.Spec.ClusterServiceClassRef
	newClusterServicePlan.Spec.ClusterServiceBrokerName = oldClusterServicePlan.Spec.ClusterServiceBrokerName
}

func (h *CreateUpdateHandler) syncLabels(obj *sc.ClusterServicePlan) {
	if obj.Labels == nil {
		obj.Labels = make(map[string]string)
	}

	obj.Labels[sc.GroupName+"/"+sc.FilterSpecExternalID] = util.GenerateSHA(obj.Spec.ExternalID)
	obj.Labels[sc.GroupName+"/"+sc.FilterSpecExternalName] = util.GenerateSHA(obj.Spec.ExternalName)
	obj.Labels[sc.GroupName+"/"+sc.FilterSpecClusterServiceClassRefName] = util.GenerateSHA(obj.Spec.ClusterServiceClassRef.Name)
	obj.Labels[sc.GroupName+"/"+sc.FilterSpecClusterServiceBrokerName] = util.GenerateSHA(obj.Spec.ClusterServiceBrokerName)
}
