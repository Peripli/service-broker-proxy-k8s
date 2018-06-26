package middleware

import (
	"net/http"

	"github.com/Peripli/service-broker-proxy/pkg/httputils"
	"github.com/sirupsen/logrus"
)

const (
	notAuthorized = "Not Authorized"
	errorMessage  = "Unauthorized resource access"
)

func BasicAuth(username, password string) func(handler http.Handler) http.Handler {
	return func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !authorized(r, username, password) {
				logrus.WithField("username", username).Debug(errorMessage)
				httputils.WriteResponse(w, http.StatusUnauthorized, httputils.HTTPErrorResponse{
					ErrorKey:     notAuthorized,
					ErrorMessage: errorMessage,
				},
				)
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
