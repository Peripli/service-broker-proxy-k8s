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
	"sync"
	"time"

	"github.com/Peripli/service-broker-proxy/pkg/platform"
	"github.com/pkg/errors"

	"github.com/Peripli/service-broker-proxy/pkg/platform/platformfakes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Reconcile", func() {

	Describe("Run", func() {
		var (
			stopRun        chan bool
			isRunning      chan bool
			wg             sync.WaitGroup
			platformClient platformfakes.FakeClient
			task           *ReconciliationTask
		)

		startRunAsync := func() {
			wg.Add(1)
			go func() {
				defer wg.Done()
				task.Run()
			}()
		}

		waitForChannel := func(c chan bool) error {
			select {
			case <-c:
				return nil
			case <-time.After(5 * time.Second):
				return errors.New("channel timed out")
			}
		}

		BeforeEach(func() {
			stopRun = make(chan bool, 1)
			isRunning = make(chan bool, 1)
			platformClient.BrokerStub = func() platform.BrokerClient {
				isRunning <- true
				Expect(waitForChannel(stopRun)).ToNot(HaveOccurred())
				return nil
			}
			task = NewTask(context.TODO(), nil, &sync.WaitGroup{}, &platformClient, nil, "", nil)
		})

		AfterEach(func() {
			wg.Wait()
		})

		Context("when another task is not yet finished", func() {

			It("should not be started", func() {
				startRunAsync()
				Expect(waitForChannel(isRunning)).ToNot(HaveOccurred())
				task.Run()
				stopRun <- true
			})
		})

		Context("when the first task has finished", func() {

			It("should finish the second one", func() {
				startRunAsync()
				stopRun <- true
				Expect(waitForChannel(isRunning)).ToNot(HaveOccurred())
				wg.Wait()

				startRunAsync()
				stopRun <- true
			})
		})

	})
})
