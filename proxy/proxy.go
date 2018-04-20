package proxy

import (
	"log"
	"net/http"
	"strings"
	"time"
)

// Proxy is the service broker proxy. The Proxy listens to all incoming requests and proxies them to the service manager.
type Proxy struct {
	config   *Env
	url      string
	username string
	password string
}

// NewProxy creates a new instance of Proxy with the configuration from the environment variables stored in EnvConfig.
func NewProxy() Proxy {
	env := EnvConfig()
	return Proxy{
		&env,
		env.serviceManagerURL,
		env.serviceManagerUser,
		env.serviceManagerPassword,
	}
}

// Start starts the service broker proxy on port 8080. This function runs as long as the server is up and running.
func (proxy Proxy) Start() error {
	// Periodically check for service brokers
	go pingServiceManager(5)

	// Start the proxy
	http.HandleFunc("/", proxy.listen)
	return http.ListenAndServe(":8080", nil)
}

func (proxy Proxy) getBrokerIDFromURLFragment(urlFragment string) string {
	expectedFragmentPrefix := "/brokerproxy/"
	if strings.HasPrefix(urlFragment, expectedFragmentPrefix) {
		lenPrefix := len(expectedFragmentPrefix)
		lenFragment := len(urlFragment)
		return urlFragment[lenPrefix:lenFragment]
	}
	return ""
}

func (proxy Proxy) copyAndFilterHeaders(request *http.Request, newRequest *http.Request) {
	hopHeaders := []string{"host", "content-length", "connection", "keep-alive", "transfer-encoding", "upgrade"}
	for name, value := range request.Header {
		if !stringInSlice(name, hopHeaders) {
			valueString := strings.Join(value, " ")
			newRequest.Header.Add(name, valueString)
		}
	}
}

func (proxy Proxy) setBrokerHeaders(request *http.Request) {
	request.SetBasicAuth(proxy.username, proxy.password)
	if len(request.Header.Get("Accept-Encoding")) > 0 {
		request.Header.Set("Accept-Encoding", "gzip")
	}
}

func (proxy Proxy) listen(writer http.ResponseWriter, request *http.Request) {
	log.Println("[proxy.go; listen()] request url from env: " + request.URL.Path)

	brokerURLFragment := proxy.getBrokerIDFromURLFragment(request.URL.Path)
	log.Println("[proxy.go; listen()] broker url after getBrokerIdFromURLFragment: " + brokerURLFragment)

	brokerURL := proxy.url + "/v1/osb/" + brokerURLFragment
	log.Println("[proxy.go; listen()] broker url: " + brokerURL)

	//TODO error handling
	client := &http.Client{
		Timeout:   time.Duration(proxy.config.timeoutSeconds) * time.Second,
		Transport: &loggingRoundTripper{},
	}
	smRequest, _ := http.NewRequest(request.Method, brokerURL, request.Body)
	proxy.copyAndFilterHeaders(request, smRequest)
	proxy.setBrokerHeaders(smRequest)

	_, err := client.Do(smRequest)
	if err != nil {
		log.Fatal("Request to service manager failed: ", err.Error())
	}
}
