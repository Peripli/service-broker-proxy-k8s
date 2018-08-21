package sm

import (
	"net/http"
	"crypto/tls"
)

// BasicAuthTransport implements http.RoundTripper interface and intercepts that request that is being sent,
// adding basic authorization and delegates back to the original transport.
type BasicAuthTransport struct {
	Username string
	Password string

	Rt http.RoundTripper
}

var _ http.RoundTripper = &BasicAuthTransport{}

// RoundTrip implements http.RoundTrip and adds basic authorization header before delegating to the
// underlying RoundTripper
func (b BasicAuthTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	if b.Username != "" && b.Password != "" {
		request.SetBasicAuth(b.Username, b.Password)
	}

	return b.Rt.RoundTrip(request)
}

// SkipSSLTransport implements http.RoundTripper and sets the SSL Validation to match the provided property
type SkipSSLTransport struct {
	SkipSslValidation bool
}

var _ http.RoundTripper = &SkipSSLTransport{}

// RoundTrip implements http.RoundTrip and adds skip SSL validation logic
func (b SkipSSLTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	defaultTransport := http.DefaultTransport.(*http.Transport)
	t := &http.Transport{
		Proxy:                 defaultTransport.Proxy,
		TLSHandshakeTimeout:   defaultTransport.TLSHandshakeTimeout,
		ExpectContinueTimeout: defaultTransport.ExpectContinueTimeout,
	}

	t.TLSClientConfig = &tls.Config{
		InsecureSkipVerify: b.SkipSslValidation,
	}
	return t.RoundTrip(request)
}