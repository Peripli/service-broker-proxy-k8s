package proxy

import (
	"net/http/httputil"
	"fmt"
	"net/http"
	"log"
)

type loggingRoundTripper struct {

}

func (d *loggingRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	requestDump, _ := httputil.DumpRequest(request, true)
	log.Println("### Request ### \n" + string(requestDump))

	response, err := http.DefaultTransport.RoundTrip(request)

	responseDump, _ := httputil.DumpResponse(response, true)
	fmt.Println("### Response ###\n" + string(responseDump))

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
