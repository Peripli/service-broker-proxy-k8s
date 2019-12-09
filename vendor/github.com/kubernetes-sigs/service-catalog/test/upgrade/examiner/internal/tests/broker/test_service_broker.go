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

package broker

import (
	sc "github.com/kubernetes-sigs/service-catalog/pkg/client/clientset_generated/clientset"
	"k8s.io/klog"
	"time"
)

const (
	serviceBrokerName                = "test-broker"
	successFetchedCatalogMessage     = "Successfully fetched catalog entries from broker."
	amountOfServiceClasses           = 24
	amountOfServicePlans             = 25
	serviceInstanceName              = "test-instance"
	successProvisionMessage          = "The instance was provisioned successfully"
	serviceBindingName               = "test-binding"
	successInjectedBindResultMessage = "Injected bind result"

	waitInterval    = 1 * time.Second
	timeoutInterval = 20 * time.Second
)

// ClientGetter is an interface to represent structs return kubernetes clientset
type ClientGetter interface {
	ServiceCatalogClient() sc.Interface
}

// TestBroker represents upgrade test for ServiceBroker
type TestBroker struct {
	client ClientGetter
}

// NewTestBroker is constructor for TestBroker
func NewTestBroker(cli ClientGetter) *TestBroker {
	return &TestBroker{cli}
}

// CreateResources prepares resources for upgrade test for ServiceBroker
func (tb *TestBroker) CreateResources(stop <-chan struct{}, namespace string) error {
	c := newCreator(tb.client, namespace)

	klog.Info("Start creation process")
	return c.execute()
}

// TestResources executes test for ServiceBroker and clean resource after finish
func (tb *TestBroker) TestResources(stop <-chan struct{}, namespace string) error {
	c := newTester(tb.client, namespace)

	klog.Info("Start test process")
	return c.execute()
}
