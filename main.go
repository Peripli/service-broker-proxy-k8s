package main


import (
	"fmt"
	"github.com/kubernetes-incubator/service-catalog/pkg/svcat"
	"flag"
	"time"
	"github.com/kubernetes-incubator/service-catalog/pkg/apis/servicecatalog/v1beta1"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
)

func main() {
	brokerName := "my-super-broker"

	// When running as a pod in-cluster, a kubeconfig is not needed. Instead this will make use of the service account injected into the pod.
	// However, allow the use of a local kubeconfig as this can make local development & testing easier.
	kubeconfig := flag.String("kubeconfig", "", "Path to a kubeconfig file")

	flag.Parse()
	fmt.Println(*kubeconfig)

	// buggy code. Ensure that the env var "KUBERNETES_MASTER" is set. The code uses a deprecated
	// interface from
	//
	a, _ := svcat.NewApp(*kubeconfig, "shoot-garden-cpet-s3")

	fmt.Println("----------------------")
	fmt.Println("Current listed brokers")
	fmt.Println("----------------------")
	brokers, _ := a.RetrieveBrokers()
	for _, b := range brokers {
		fmt.Println(b.Name)
	}
	fmt.Println("----------------------\n\n")


	// Register a new Broker
	//
	request :=&v1beta1.ClusterServiceBroker{
		ObjectMeta: v1.ObjectMeta{
			Name:     brokerName,
			Namespace: "default",
		},
		Spec: v1beta1.ClusterServiceBrokerSpec{
			CommonServiceBrokerSpec: v1beta1.CommonServiceBrokerSpec{
				URL:            "http://example.com",
				RelistBehavior: "Manual",
			},
		},
	}

	fmt.Println("----------------------")
	a.ServiceCatalog().ClusterServiceBrokers().Create(request)
	fmt.Println("new Broker registered")
	fmt.Println("----------------------\n\n")

	fmt.Println("----------------------")
	fmt.Println("Current listed brokers")
	fmt.Println("----------------------")
	brokers, _ = a.RetrieveBrokers()
	for _, b := range brokers {
		fmt.Println(b.Name)
	}
	fmt.Println("----------------------\n\n")


	a.ServiceCatalog().ClusterServiceBrokers().Delete(brokerName, &v1.DeleteOptions{})
	// just wait some secinds until kubernetes has removed the ServiceBroker resource
	time.Sleep(2 * time.Second)
	fmt.Println("Broker deleted")
	fmt.Println("----------------------\n\n")


	fmt.Println("Current listed brokers")
	fmt.Println("----------------------")
	brokers, _ = a.RetrieveBrokers()
	for _, b := range brokers {
		fmt.Println(b.Name)
	}
	fmt.Println("----------------------\n\n")

}