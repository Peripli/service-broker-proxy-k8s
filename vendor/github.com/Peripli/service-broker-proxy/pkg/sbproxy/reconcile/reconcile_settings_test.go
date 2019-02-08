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
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func validSettings() *Settings {
	settings := DefaultSettings()
	settings.URL = "http://localhost:8080"
	settings.Username = "user"
	settings.Password = "password"
	return settings
}

var _ = Describe("Reconcile", func() {

	Describe("Settings", func() {
		Describe("Validate", func() {
			Context("when all properties are set correctly", func() {
				It("no error is returned", func() {
					Expect(validSettings().Validate()).ShouldNot(HaveOccurred())
				})
			})

			Context("when URL is missing", func() {
				It("returns an error", func() {
					settings := validSettings()
					settings.URL = ""
					Expect(settings.Validate()).Should(HaveOccurred())
				})
			})

			Context("when Username is missing", func() {
				It("returns an error", func() {
					settings := validSettings()
					settings.Username = ""
					Expect(settings.Validate()).Should(HaveOccurred())
				})
			})

			Context("when Password is missing", func() {
				It("returns an error", func() {
					settings := validSettings()
					settings.Password = ""
					Expect(settings.Validate()).Should(HaveOccurred())
				})
			})

			Context("when CacheExpiration is less then 1 minute", func() {
				It("returns an error", func() {
					settings := validSettings()
					settings.CacheExpiration = time.Second
					Expect(settings.Validate()).Should(HaveOccurred())
				})
			})
		})
	})
})
