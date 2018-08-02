package sm

import (
	"encoding/base64"
	"fmt"
	"net/http"
)

// BasicAuthTransport implements http.RoundTripper interface and intercepts that request that is being sent,
// adding basic authorization and delegates back to the original transport.
type BasicAuthTransport struct {
	username string
	password string
	rt       http.RoundTripper
}

// RoundTrip implements http.RoundTrip and adds basic authorization header before delegating to the
// underlying RoundTripper
func (b BasicAuthTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", fmt.Sprintf("Basic %s",
		base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s",
			b.username, b.password)))))
	return b.rt.RoundTrip(req)
}
