package osb

import (
	"net/http"

	"github.com/pmorie/osb-broker-lib/pkg/broker"

	"github.com/pkg/errors"

	"fmt"

	"github.com/gorilla/mux"
	osbc "github.com/pmorie/go-open-service-broker-client/v2"
	"github.com/sirupsen/logrus"
)

// BusinessLogic provides an implementation of the osb.BusinessLogic interface.
type BusinessLogic struct {
	createFunc      osbc.CreateFunc
	osbClientConfig *osbc.ClientConfiguration
}

var _ broker.Interface = &BusinessLogic{}

func NewBusinessLogic(config *ClientConfiguration) (*BusinessLogic, error) {
	return &BusinessLogic{
		osbClientConfig: config.ClientConfiguration,
		createFunc:      config.CreateFunc,
	}, nil
}

func (b *BusinessLogic) GetCatalog(c *broker.RequestContext) (*broker.CatalogResponse, error) {
	client, err := osbClient(c.Request, *b.osbClientConfig, b.createFunc)
	if err != nil {
		return nil, err
	}
	response, err := client.GetCatalog()
	if err != nil {
		return nil, err
	}

	return &broker.CatalogResponse{
		CatalogResponse: *response,
	}, nil
}

func (b *BusinessLogic) Provision(request *osbc.ProvisionRequest, c *broker.RequestContext) (*broker.ProvisionResponse, error) {
	client, err := osbClient(c.Request, *b.osbClientConfig, b.createFunc)
	if err != nil {
		return nil, err
	}
	response, err := client.ProvisionInstance(request)
	if err != nil {
		return nil, err
	}

	return &broker.ProvisionResponse{
		ProvisionResponse: *response,
	}, nil
}

func (b *BusinessLogic) Deprovision(request *osbc.DeprovisionRequest, c *broker.RequestContext) (*broker.DeprovisionResponse, error) {
	client, err := osbClient(c.Request, *b.osbClientConfig, b.createFunc)
	if err != nil {
		return nil, err
	}
	response, err := client.DeprovisionInstance(request)
	if err != nil {
		return nil, err
	}

	return &broker.DeprovisionResponse{
		DeprovisionResponse: *response,
	}, nil
}

func (b *BusinessLogic) LastOperation(request *osbc.LastOperationRequest, c *broker.RequestContext) (*broker.LastOperationResponse, error) {
	client, err := osbClient(c.Request, *b.osbClientConfig, b.createFunc)
	if err != nil {
		return nil, err
	}
	response, err := client.PollLastOperation(request)
	if err != nil {
		return nil, err
	}

	return &broker.LastOperationResponse{
		LastOperationResponse: *response,
	}, nil
}

func (b *BusinessLogic) Bind(request *osbc.BindRequest, c *broker.RequestContext) (*broker.BindResponse, error) {
	client, err := osbClient(c.Request, *b.osbClientConfig, b.createFunc)
	if err != nil {
		return nil, err
	}
	response, err := client.Bind(request)
	if err != nil {
		return nil, err
	}

	return &broker.BindResponse{
		BindResponse: *response,
	}, nil

}
func (b *BusinessLogic) Unbind(request *osbc.UnbindRequest, c *broker.RequestContext) (*broker.UnbindResponse, error) {
	client, err := osbClient(c.Request, *b.osbClientConfig, b.createFunc)
	if err != nil {
		return nil, err
	}

	response, err := client.Unbind(request)
	if err != nil {
		return nil, err
	}

	return &broker.UnbindResponse{
		UnbindResponse: *response,
	}, nil
}

func (b *BusinessLogic) Update(request *osbc.UpdateInstanceRequest, c *broker.RequestContext) (*broker.UpdateInstanceResponse, error) {
	client, err := osbClient(c.Request, *b.osbClientConfig, b.createFunc)
	if err != nil {
		return nil, err
	}
	response, err := client.UpdateInstance(request)
	if err != nil {
		return nil, err
	}

	return &broker.UpdateInstanceResponse{
		UpdateInstanceResponse: *response,
	}, nil
}

func (b *BusinessLogic) ValidateBrokerAPIVersion(version string) error {
	if version == "" {
		return errors.New("X-Broker-API-Version Header not set")
	}

	apiVersionHeader := b.osbClientConfig.APIVersion.HeaderValue()
	if apiVersionHeader != version {
		return errors.New("X-Broker-API-Version Header must be " + apiVersionHeader + " but was " + version)
	}
	return nil
}

func osbClient(request *http.Request, config osbc.ClientConfiguration, createFunc osbc.CreateFunc) (osbc.Client, error) {
	vars := mux.Vars(request)
	brokerID, ok := vars["brokerID"]
	if !ok {
		errMsg := fmt.Sprintf("brokerId path parameter missing from %s", request.Host)
		logrus.WithError(errors.New(errMsg)).Error("Error building OSB client for proxy business logic")

		return nil, osbc.HTTPStatusCodeError{
			StatusCode:  http.StatusBadRequest,
			Description: &errMsg,
		}
	}
	config.URL = config.URL + "/" + brokerID
	config.Name = config.Name + "-" + brokerID
	logrus.Debug("Building OSB client for broker with name: ", config.Name, " accesible at: ", config.URL)
	return createFunc(&config)
}
