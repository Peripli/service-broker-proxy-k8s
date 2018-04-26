package sm

import (
	"encoding/base64"
	"fmt"
	"net/http"
)

// BasicAuthTransport intercepts that request that is being sent, adds basic authorization
// and delegates back to the original transport
//TODO Middleware ideas and resources
// TODO https://github.com/ernesto-jimenez/httplogger
//TODO https://github.com/h2non/gentleman#examples
type BasicAuthTransport struct {
	username string
	password string
	rt       http.RoundTripper
}

func (b BasicAuthTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", fmt.Sprintf("Basic %s",
		base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s",
			b.username, b.password)))))
	return b.rt.RoundTrip(req)
}
