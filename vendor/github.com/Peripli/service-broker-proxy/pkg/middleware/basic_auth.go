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
	"github.com/Peripli/service-manager/pkg/log"
	"net/http"

	"github.com/Peripli/service-manager/pkg/util"
)

const (
	notAuthorized = "Not Authorized"
	errorMessage  = "Unauthorized resource access"
)

// BasicAuth is a middleware for basic authorization
func BasicAuth(username, password string) func(handler http.Handler) http.Handler {
	return func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !authorized(r, username, password) {
				logger := log.C(r.Context())
				logger.WithField("username", username).Debug(errorMessage)
				err := util.WriteJSON(w, http.StatusUnauthorized, &util.HTTPError{
					ErrorType:   notAuthorized,
					Description: errorMessage,
					StatusCode:  http.StatusUnauthorized,
				})
				if err != nil {
					logger.Error(err)
				}
				return
			}
			handler.ServeHTTP(w, r)
		})
	}
}

func authorized(r *http.Request, username, password string) bool {
	requestUsername, requestPassword, isOk := r.BasicAuth()
	return isOk && username == requestUsername && password == requestPassword
}
