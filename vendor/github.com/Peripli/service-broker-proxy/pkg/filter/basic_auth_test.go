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

package filter

import (
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/pkg/web"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"net/http"
)

var _ = Describe("Basic Authentication wrapper", func() {
	const (
		validUsername   = "validUsername"
		validPassword   = "validPassword"
		invalidUser     = "invalidUser"
		invalidPassword = "invalidPassword"
	)
	var (
		filter web.Filter
	)

	newRequest := func(user, pass string) *http.Request {
		request, err := http.NewRequest("GET", "", nil)
		Expect(err).NotTo(HaveOccurred())
		request.SetBasicAuth(user, pass)
		return request
	}

	BeforeEach(func() {
		filter = NewBasicAuthFilter(validUsername, validPassword)
	})

	DescribeTable("when given a request with basic authorization",
		func(expectedStatus int, expectsError bool, username, password string) {

			request := newRequest(username, password)
			response, err := filter.Run(&web.Request{Request: request}, testHandler())

			if expectsError {
				Expect(err).To(HaveOccurred())
				httpError := err.(*util.HTTPError)
				Expect(httpError.StatusCode).To(Equal(expectedStatus))
			} else {
				Expect(response.StatusCode).To(Equal(expectedStatus))
			}
		},
		Entry("returns 401 for empty username", http.StatusUnauthorized, true, "", validPassword),
		Entry("returns 401 for empty password", http.StatusUnauthorized, true, validUsername, ""),
		Entry("returns 401 for invalid credentials", http.StatusUnauthorized, true, invalidUser, invalidPassword),
		Entry("returns 200 for valid credentials", http.StatusOK, false, validUsername, validPassword),
	)
})

func testHandler() web.HandlerFunc {
	return web.HandlerFunc(func(request *web.Request) (*web.Response, error) {
		return &web.Response{
			StatusCode: 200,
			Body:       []byte(`{}`),
		}, nil
	})
}
