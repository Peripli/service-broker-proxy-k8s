package main

import (
	"errors"
	"testing"

	"github.com/Peripli/service-broker-proxy/pkg/platform"

	"os"

	. "github.com/onsi/ginkgo"

	"github.com/kubernetes-incubator/service-catalog/pkg/apis/servicecatalog/v1beta1"
	"github.com/kubernetes-incubator/service-catalog/pkg/svcat"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
)

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
					BearerToken:     string("faketoken"),
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
					Name:      "fake-broker",
					BrokerURL: "http://fake.broker.url",
				}
				createdBroker, err := platformClient.CreateBroker(requestBroker)

				Expect(createdBroker.Name).To(Equal("fake-broker"))
				Expect(createdBroker.BrokerURL).To(Equal("http://fake.broker.url"))
				Expect(err).To(BeNil())
			})

			It("with an error", func() {
				platformClient, _ := NewClient()
				createClusterServiceBroker = func(app *svcat.App, broker *v1beta1.ClusterServiceBroker) (*v1beta1.ClusterServiceBroker, error) {
					return nil, errors.New("Error from service-catalog")
				}

				requestBroker := &platform.CreateServiceBrokerRequest{}
				createdBroker, err := platformClient.CreateBroker(requestBroker)

				Expect(createdBroker).To(BeNil())
				Expect(err).To(Equal(errors.New("Error from service-catalog")))
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

			It("with an error", func() {
				platformClient, _ := NewClient()
				deleteClusterServiceBroker = func(app *svcat.App, name string, options *v1.DeleteOptions) error {
					return errors.New("Error deleting clusterservicebroker")
				}

				requestBroker := &platform.DeleteServiceBrokerRequest{}

				err := platformClient.DeleteBroker(requestBroker)

				Expect(err).To(Equal(errors.New("Error deleting clusterservicebroker")))
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

			It("with an error", func() {
				platformClient, _ := NewClient()
				retrieveClusterServiceBrokers = func(app *svcat.App) ([]v1beta1.ClusterServiceBroker, error) {
					return nil, errors.New("Error getting clusterservicebrokers")
				}

				brokers, err := platformClient.GetBrokers()

				Expect(brokers).To(BeNil())
				Expect(err).To(Equal(errors.New("Error getting clusterservicebrokers")))
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
					Name:      "fake-broker",
					BrokerURL: "http://fake.broker.url",
				}

				broker, err := platformClient.UpdateBroker(requestBroker)

				Expect(err).To(BeNil())
				Expect(broker.Guid).To(Equal("1234"))
				Expect(broker.Name).To(Equal("fake-broker-updated"))
				Expect(broker.BrokerURL).To(Equal("http://fake.broker.url-updated"))
			})

			It("with an error", func() {
				platformClient, _ := NewClient()
				updateClusterServiceBroker = func(app *svcat.App, broker *v1beta1.ClusterServiceBroker) (*v1beta1.ClusterServiceBroker, error) {
					return nil, errors.New("Error updating clusterservicebroker")
				}

				requestBroker := &platform.UpdateServiceBrokerRequest{}

				broker, err := platformClient.UpdateBroker(requestBroker)

				Expect(broker).To(BeNil())
				Expect(err).To(Equal(errors.New("Error updating clusterservicebroker")))
			})
		})

		Context("Fetches the catalog information of a service broker", func() {
			It("successfully", func() {
				platformClient, _ := NewClient()
				requestBroker := &platform.ServiceBroker{
					Guid:      "1234",
					Name:      "fake-broker",
					BrokerURL: "http://fake.broker.url",
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
