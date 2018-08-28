package osb

import (
	"net/http"
	"github.com/Peripli/service-manager/api/osb"
	"github.com/Peripli/service-manager/pkg/types"
)

// BrokerTransport implements osb.BrokerRoundTripper
type BrokerTransport struct {
	Username string
	Password string
	URL      string

	Tr http.RoundTripper
}

var _ osb.BrokerRoundTripper = &BrokerTransport{}

// Broker implements osb.BrokerRoundTripper and returns the coordinates of the broker with the specified id
func (b *BrokerTransport) Broker(brokerID string) (*types.Broker, error) {
	return &types.Broker{
		BrokerURL: b.URL + "/" + brokerID,
		Credentials: &types.Credentials{
			Basic: &types.Basic{
				Username: b.Username,
				Password: b.Password,
			},
		},
	}, nil
}

// RoundTrip implements http.RoundTripper by delegating to the provided RoundTripper
func (b *BrokerTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	return b.Tr.RoundTrip(request)
}
