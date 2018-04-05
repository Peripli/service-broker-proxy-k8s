package proxy

import (
	"net/http/httputil"
	"fmt"
	"net/http"
	"log"
)

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func printRequest(request *http.Request) {
	requestDump, _ := httputil.DumpRequest(request, true)
	log.Println("###\nReqeust\n" + string(requestDump))
}

func printResponse(request *http.Response) {
	responseDump, _ := httputil.DumpResponse(request, true)
	fmt.Println("###\nResponse\n" + string(responseDump))
}
