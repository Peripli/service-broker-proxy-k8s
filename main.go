package main

import (
	"github.com/Peripli/service-broker-proxy-k8s/proxy"
	"log"
)

func main() {
	log.Println("Staring Kubernetes Service Broker Proxy")

	p := proxy.New()
	err := p.Start()
	if err != nil {
		log.Fatal("Kubernetes Service Broker Proxy Error")
		log.Fatal(err)
		panic(err)
	}
}
