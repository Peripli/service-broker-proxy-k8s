package proxy

import (
	"log"
	"net/http"
	"net/http/httputil"
	"time"
)

type loggingRoundTripper struct {
}

func (d *loggingRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	requestDump, _ := httputil.DumpRequest(request, true)
	log.Println("### Request ### \n" + string(requestDump))

	response, err := http.DefaultTransport.RoundTrip(request)

	responseDump, _ := httputil.DumpResponse(response, true)
	log.Println("### Response ###\n" + string(responseDump))

	return response, err
}

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func pingServiceManager(seconds int) {
	d := time.Duration(seconds) * time.Second
	for range time.Tick(d) {
		log.Println("Request service manager")
	}
}
