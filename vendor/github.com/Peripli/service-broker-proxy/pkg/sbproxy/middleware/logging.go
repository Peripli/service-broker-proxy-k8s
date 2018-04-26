package middleware

import (
	"net/http"

	"github.com/sirupsen/logrus"
)

func LogRequest() func(handler http.Handler) http.Handler {
	return func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			logrus.WithFields(map[string]interface{}{"path": r.RequestURI, "method": r.Method}).Info("Request: ")
			handler.ServeHTTP(w, r)
		})
	}
}
