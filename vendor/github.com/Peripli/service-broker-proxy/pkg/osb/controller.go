package osb

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"time"

	smOsb "github.com/Peripli/service-manager/api/osb"
	"github.com/Peripli/service-manager/pkg/proxy"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// NewOsbAdapter creates an OSB business logic containing logic to proxy OSB calls
func NewOsbAdapter(config *ClientConfig) (smOsb.Adapter, error) {
	defaultTransport := http.DefaultTransport.(*http.Transport)
	t := &http.Transport{
		Proxy:                 defaultTransport.Proxy,
		TLSHandshakeTimeout:   defaultTransport.TLSHandshakeTimeout,
		ExpectContinueTimeout: defaultTransport.ExpectContinueTimeout,
	}
	t.TLSClientConfig = &tls.Config{
		InsecureSkipVerify: config.Insecure,
	}
	t.DialContext = (&net.Dialer{
		Timeout:   time.Duration(config.TimeoutSeconds),
		KeepAlive: 30 * time.Second,
		DualStack: true,
	}).DialContext

	t.TLSClientConfig = &tls.Config{InsecureSkipVerify: config.Insecure}
	proxier := proxy.NewReverseProxy(proxy.Options{
		Transport: t,
	})

	bl := &businessLogic{
		config:  config,
		proxier: proxier,
	}

	return bl, nil
}

type businessLogic struct {
	proxier *proxy.Proxy

	config *ClientConfig
}

func (b *businessLogic) Handler() web.HandlerFunc {
	return func(request *web.Request) (*web.Response, error) {
		target, err := osbClient(request, b.config)
		if err != nil {
			return nil, err
		}

		targetURL, _ := url.Parse(target.URL)
		targetURL.Path = request.Request.URL.Path

		reqBuilder := b.proxier.RequestBuilder().
			URL(targetURL).
			Auth(target.Username, target.Password)

		resp, err := b.proxier.ProxyRequest(request.Request, reqBuilder, request.Body)
		if err != nil {
			return nil, err
		}

		respBody, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}

		webResponse := &web.Response{
			StatusCode: resp.StatusCode,
			Header:     resp.Header,
			Body:       respBody,
		}

		return webResponse, nil
	}
}

func osbClient(request *web.Request, config *ClientConfig) (*Target, error) {
	brokerID, ok := request.PathParams["brokerID"]
	if !ok {
		errMsg := fmt.Sprintf("brokerId path parameter missing from %s", request.Host)
		logrus.WithError(errors.New(errMsg)).Error("Error building OSB client for proxy business logic")

		return nil, &util.HTTPError{
			StatusCode:  http.StatusBadRequest,
			Description: errMsg,
		}
	}

	target := &Target{
		URL:      config.URL + "/" + brokerID,
		Username: config.Username,
		Password: config.Password,
	}

	logrus.Debug("Building OSB client for broker with name: ", config.Name, " accesible at: ", target.URL)
	return target, nil
}
