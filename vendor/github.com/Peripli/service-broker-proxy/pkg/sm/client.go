package sm

import (
	"net/http"

	"fmt"

	"time"

	"github.com/Peripli/service-broker-proxy/pkg/httputils"
	"github.com/Peripli/service-broker-proxy/pkg/platform"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const APIInternalBrokers = "%s/v1/service_brokers"

type Client interface {
	GetBrokers() ([]platform.ServiceBroker, error)
}

type serviceManagerClient struct {
	Config     *ClientConfiguration
	httpClient *http.Client
}

var _ Client = &serviceManagerClient{}

func NewClient(config *ClientConfiguration) (Client, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	httpClient := &http.Client{
		Timeout: time.Duration(config.TimeoutSeconds) * time.Second,
	}

	if config.User != "" && config.Password != "" {
		httpClient.Transport = BasicAuthTransport{
			username: config.User,
			password: config.Password,
			rt:       http.DefaultTransport,
		}
	}

	client := &serviceManagerClient{
		Config:     config,
		httpClient: httpClient,
	}

	return client, nil
}

//TODO paging
func (c *serviceManagerClient) GetBrokers() ([]platform.ServiceBroker, error) {
	logrus.Debugf("Getting brokers for proxy from Service Manager at %s", c.Config.Host)
	URL := fmt.Sprintf(APIInternalBrokers, c.Config.Host)
	response, err := httputils.SendRequest(c.httpClient, http.MethodGet, URL, nil, nil)
	if err != nil {
		return nil, errors.Wrap(err, "error getting brokers from Service Manager")
	}
	list := &BrokerList{}
	switch response.StatusCode {
	case http.StatusOK:
		if err = httputils.GetContent(list, response.Body); err != nil {
			return nil, errors.Wrapf(err, "error getting content from body of response with status %s", response.Status)
		}
	default:
		return nil, errors.WithStack(httputils.HandleResponseError(response))
	}

	return c.packResponse(list), nil
}

func (c *serviceManagerClient) packResponse(list *BrokerList) []platform.ServiceBroker {
	brokers := make([]platform.ServiceBroker, 0, len(list.Brokers))
	for _, broker := range list.Brokers {
		b := platform.ServiceBroker{
			Guid:      broker.ID,
			Name:      broker.Name,
			BrokerURL: broker.BrokerURL,
		}
		brokers = append(brokers, b)
	}
	return brokers
}
