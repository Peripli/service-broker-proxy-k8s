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
