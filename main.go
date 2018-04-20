package main

import (
	"log"

	"github.com/Peripli/service-broker-proxy-k8s/proxy"
)

func main() {
	log.Println("[main.go; main()] Staring Kubernetes Service Broker Proxy")

	p := proxy.NewProxy()
	err := p.Start()
	if err != nil {
		log.Fatal("[main.go; main()] Kubernetes Service Broker Proxy Error")
		log.Fatal(err)
		panic(err)
	}
}
