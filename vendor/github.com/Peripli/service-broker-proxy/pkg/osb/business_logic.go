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

// BusinessLogic provides an implementation of the pmorie/osb-broker-lib/pkg/broker/Interface interface.
type BusinessLogic struct {
	createFunc      osbc.CreateFunc
	osbClientConfig *osbc.ClientConfiguration
}

var _ broker.Interface = &BusinessLogic{}

// NewBusinessLogic creates an OSB business logic containing logic to proxy OSB calls
func NewBusinessLogic(config *ClientConfig) (*BusinessLogic, error) {
	return &BusinessLogic{
		osbClientConfig: config.ClientConfiguration,
		createFunc:      config.CreateFunc,
	}, nil
}

// GetCatalog implements pmorie/osb-broker-lib/pkg/broker/Interface.GetCatalog by
// proxying the call to an underlying OSB compliant API
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

// Provision implements pmorie/osb-broker-lib/pkg/broker/Interface.Provision by
// proxying the call to an underlying OSB compliant API
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

// Deprovision implements pmorie/osb-broker-lib/pkg/broker/Interface.Deprovision by
// proxying the call to an underlying OSB compliant API
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

// LastOperation implements pmorie/osb-broker-lib/pkg/broker/Interface.LastOperation by
// proxying the call to an underlying OSB compliant API
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

// Bind implements pmorie/osb-broker-lib/pkg/broker/Interface.Bind by
// proxying the call to an underlying OSB compliant API
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

// Unbind implements pmorie/osb-broker-lib/pkg/broker/Interface.Unbind by
// proxying the call to an underlying OSB compliant API
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

// Update implements pmorie/osb-broker-lib/pkg/broker/Interface.Update by
// proxying the call to an underlying OSB compliant API
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

// ValidateBrokerAPIVersion implements pmorie/osb-broker-lib/pkg/broker/Interface.ValidateBrokerAPIVersion by
// allowing all versions of the API
func (b *BusinessLogic) ValidateBrokerAPIVersion(version string) error {
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
