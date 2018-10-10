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

package middleware

import (
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	"encoding/json"

	"github.com/Peripli/service-manager/pkg/util"
)

var _ = Describe("Basic Authentication wrapper", func() {
	const (
		validUsername = "validUsername"
		validPassword = "validPassword"
		invalidUser = "invalidUser"
		invalidPassword = "invalidPassword"
	)
	var (
		httpRecorder   *httptest.ResponseRecorder
		wrappedHandler http.Handler
	)

	newRequest := func(user, pass string) *http.Request {
		request, err := http.NewRequest("GET", "", nil)
		Expect(err).NotTo(HaveOccurred())
		request.SetBasicAuth(user, pass)
		return request
	}

	BeforeEach(func() {
		httpRecorder = httptest.NewRecorder()
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("{}"))
		})
		wrappedHandler = BasicAuth(validUsername, validPassword)(handler)
	})

	DescribeTable("when given a request with basic authorization",
		func(expectedStatus int, expectedError, username, password string) {

			request := newRequest(username, password)
			wrappedHandler.ServeHTTP(httpRecorder, request)

			Expect(httpRecorder.Code).To(Equal(expectedStatus))
			if expectedError != "" {
				var body util.HTTPError
				decoder := json.NewDecoder(httpRecorder.Body)
				err := decoder.Decode(&body)
				Expect(err).ToNot(HaveOccurred())
				Expect(body.ErrorType).To(Equal(expectedError))
				Expect(body.Description).To(Not(BeEmpty()))
			}
		},
		Entry("returns 401 for empty username", http.StatusUnauthorized, "Not Authorized", "", validPassword),
		Entry("returns 401 for empty password", http.StatusUnauthorized, "Not Authorized", validUsername, ""),
		Entry("returns 401 for invalid credentials", http.StatusUnauthorized, "Not Authorized", invalidUser, invalidPassword),
		Entry("returns 200 for valid credentials", http.StatusOK, "", validUsername, validPassword),
	)
})
