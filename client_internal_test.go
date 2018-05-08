package main

import (
	"testing"

	"github.com/Peripli/service-broker-proxy/pkg/platform"

	"fmt"
	"io"
	"os"

	. "github.com/onsi/ginkgo"

	"github.com/kubernetes-incubator/service-catalog/pkg/apis/servicecatalog/v1beta1"
	"github.com/kubernetes-incubator/service-catalog/pkg/svcat"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
)

var fs fileSystem = osFS{}

type fileSystem interface {
	Open(name string) (file, error)
	Stat(name string) (os.FileInfo, error)
}

type file interface {
	io.Closer
	io.Reader
	io.ReaderAt
	io.Seeker
	Stat() (os.FileInfo, error)
}

// osFS implements fileSystem using the local disk.
type osFS struct{}

func (osFS) Open(name string) (file, error) {
	fmt.Println("##### 1")
	if name == "/var/run/secrets/kubernetes.io/serviceaccount/token" {
		file, _ := os.Open("VERSION")
		return file, nil
	}
	return nil, nil
}
func (osFS) Stat(name string) (os.FileInfo, error) { return os.Stat(name) }

type MockSvcatApp struct {
}

func TestClient(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Kubernetes Client Tests Suite")
}

var _ = Describe("Kubernetes Broker Proxy", func() {
	Describe("Service Catalog", func() {

		BeforeSuite(func() {
			os.Setenv("KUBERNETES_SERVICE_HOST", "test")
			os.Setenv("KUBERNETES_SERVICE_PORT", "1234")
			restInClusterConfig = func() (*rest.Config, error) {
				return &rest.Config{
					Host:            "https://fakeme",
					BearerToken:     string("1234token"),
					TLSClientConfig: rest.TLSClientConfig{},
				}, nil
			}
		})

		Context("Create a service broker", func() {
			It("successfully", func() {
				platformClient, _ := NewClient()
				createClusterServiceBroker = func(app *svcat.App, broker *v1beta1.ClusterServiceBroker) (*v1beta1.ClusterServiceBroker, error) {
					return &v1beta1.ClusterServiceBroker{
						ObjectMeta: v1.ObjectMeta{
							Name: broker.Name,
						},
						Spec: v1beta1.ClusterServiceBrokerSpec{
							CommonServiceBrokerSpec: v1beta1.CommonServiceBrokerSpec{
								URL: broker.Spec.URL,
							},
						},
					}, nil
				}

				requestBroker := &platform.CreateServiceBrokerRequest{
					Name:      "test-broker",
					BrokerURL: "http://google.com",
				}
				createdBroker, err := platformClient.CreateBroker(requestBroker)

				Expect(createdBroker.Name).To(Equal("test-broker"))
				Expect(createdBroker.BrokerURL).To(Equal("http://google.com"))
				Expect(err).To(BeNil())
			})
		})

		Context("Deletes a service broker", func() {
			It("successfully", func() {
				platformClient, _ := NewClient()
				deleteClusterServiceBroker = func(app *svcat.App, name string, options *v1.DeleteOptions) error {
					return nil
				}

				requestBroker := &platform.DeleteServiceBrokerRequest{
					Guid: "1234",
					Name: "fake-broker",
				}

				err := platformClient.DeleteBroker(requestBroker)

				Expect(err).To(BeNil())
			})
		})

		Context("Gets all service brokers", func() {
			It("successfully", func() {
				platformClient, _ := NewClient()
				retrieveClusterServiceBrokers = func(app *svcat.App) ([]v1beta1.ClusterServiceBroker, error) {
					brokers := make([]v1beta1.ClusterServiceBroker, 0)
					brokers = append(brokers, v1beta1.ClusterServiceBroker{})
					return brokers, nil
				}

				brokers, err := platformClient.GetBrokers()
				Expect(err).To(BeNil())
				Expect(brokers).ToNot(BeNil())
			})
		})

		Context("Updates a service broker", func() {
			It("successfully", func() {
				platformClient, _ := NewClient()
				updateClusterServiceBroker = func(app *svcat.App, broker *v1beta1.ClusterServiceBroker) (*v1beta1.ClusterServiceBroker, error) {
					// Return a new fake clusterservicebroker with the three attributes relevant for the OSBAPI guid, name and broker url.
					// UID cannot be modified, name and url can be modified
					return &v1beta1.ClusterServiceBroker{
						ObjectMeta: v1.ObjectMeta{
							Name: broker.Name + "-updated",
							UID:  "1234",
						},
						Spec: v1beta1.ClusterServiceBrokerSpec{
							CommonServiceBrokerSpec: v1beta1.CommonServiceBrokerSpec{
								URL: broker.Spec.CommonServiceBrokerSpec.URL + "-updated",
							},
						},
					}, nil
				}

				requestBroker := &platform.UpdateServiceBrokerRequest{
					Guid:      "1234",
					Name:      "test-broker",
					BrokerURL: "http://google.com",
				}

				broker, err := platformClient.UpdateBroker(requestBroker)

				Expect(err).To(BeNil())
				Expect(broker.Guid).To(Equal("1234"))
				Expect(broker.Name).To(Equal("test-broker-updated"))
				Expect(broker.BrokerURL).To(Equal("http://google.com-updated"))
			})
		})

		Context("Fetches the catalog information of a service broker", func() {
			It("successfully", func() {
				platformClient, _ := NewClient()
				requestBroker := &platform.ServiceBroker{
					Guid:      "1234",
					Name:      "test-broker",
					BrokerURL: "http://google.com",
				}

				syncClusterServiceBroker = func(app *svcat.App, name string, retries int) error {
					return nil
				}

				err := platformClient.Fetch(requestBroker)

				Expect(err).To(BeNil())
			})
		})
	})
})
