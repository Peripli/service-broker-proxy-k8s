package k8s

import (
	"context"
	"errors"

	"github.com/Peripli/service-broker-proxy/pkg/platform"

	"os"

	. "github.com/onsi/ginkgo"

	"github.com/kubernetes-incubator/service-catalog/pkg/apis/servicecatalog/v1beta1"
	"github.com/kubernetes-incubator/service-catalog/pkg/svcat/service-catalog"

	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
)

var _ = Describe("Kubernetes Broker Proxy", func() {
	var clientConfig *ClientConfiguration
	var ctx context.Context
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
		clientConfig = defaultClientConfiguration()
		clientConfig.Secret.Name = "secretName"
		clientConfig.Secret.Namespace = "secretNamespace"
		clientConfig.K8sClientCreateFunc = newSvcatSDK
		ctx = context.TODO()
	})

	Describe("New Client", func() {
		Context("With invalid config", func() {
			It("should return error", func() {
				config := defaultClientConfiguration()
				_, err := NewClient(config)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("Properties of K8S secret configuration for broker registration missing"))
			})
		})

		Context("With invalid config", func() {
			It("should return error", func() {
				config := *clientConfig // copy
				config.K8sClientCreateFunc = func(libraryConfig *LibraryConfig) (*servicecatalog.SDK, error) {
					return nil, errors.New("expected")
				}
				_, err := NewClient(&config)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("expected"))
			})
		})
	})

	Describe("Create a service broker", func() {

		Context("with no error", func() {
			It("returns broker", func() {
				platformClient, err := NewClient(clientConfig)
				Expect(err).ToNot(HaveOccurred())
				createClusterServiceBroker = func(cli *servicecatalog.SDK, broker *v1beta1.ClusterServiceBroker) (*v1beta1.ClusterServiceBroker, error) {
					return &v1beta1.ClusterServiceBroker{
						ObjectMeta: v1.ObjectMeta{
							UID:  "1234",
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
				createdBroker, err := platformClient.CreateBroker(ctx, requestBroker)

				Expect(createdBroker.GUID).To(Equal("1234"))
				Expect(createdBroker.Name).To(Equal("fake-broker"))
				Expect(createdBroker.BrokerURL).To(Equal("http://fake.broker.url"))
				Expect(err).To(BeNil())
			})
		})

		Context("with an error", func() {
			It("returns error", func() {
				platformClient, err := NewClient(clientConfig)
				Expect(err).ToNot(HaveOccurred())

				createClusterServiceBroker = func(cli *servicecatalog.SDK, broker *v1beta1.ClusterServiceBroker) (*v1beta1.ClusterServiceBroker, error) {
					return nil, errors.New("Error from service-catalog")
				}

				requestBroker := &platform.CreateServiceBrokerRequest{}
				createdBroker, err := platformClient.CreateBroker(ctx, requestBroker)

				Expect(createdBroker).To(BeNil())
				Expect(err).To(Equal(errors.New("Error from service-catalog")))
			})
		})
	})

	Describe("Delete a service broker", func() {
		Context("with no error", func() {
			It("returns no error", func() {
				platformClient, err := NewClient(clientConfig)
				Expect(err).ToNot(HaveOccurred())

				deleteClusterServiceBroker = func(cli *servicecatalog.SDK, name string, options *v1.DeleteOptions) error {
					return nil
				}

				requestBroker := &platform.DeleteServiceBrokerRequest{
					GUID: "1234",
					Name: "fake-broker",
				}

				err = platformClient.DeleteBroker(ctx, requestBroker)

				Expect(err).To(BeNil())
			})
		})

		Context("with an error", func() {
			It("returns the error", func() {
				platformClient, err := NewClient(clientConfig)
				Expect(err).ToNot(HaveOccurred())

				deleteClusterServiceBroker = func(cli *servicecatalog.SDK, name string, options *v1.DeleteOptions) error {
					return errors.New("Error deleting clusterservicebroker")
				}

				requestBroker := &platform.DeleteServiceBrokerRequest{}

				err = platformClient.DeleteBroker(ctx, requestBroker)

				Expect(err).To(Equal(errors.New("Error deleting clusterservicebroker")))
			})
		})
	})

	Describe("Get all service brokers", func() {
		Context("with no error", func() {
			It("returns brokers", func() {
				platformClient, err := NewClient(clientConfig)
				Expect(err).ToNot(HaveOccurred())

				retrieveClusterServiceBrokers = func(cli *servicecatalog.SDK) (*v1beta1.ClusterServiceBrokerList, error) {
					brokers := make([]v1beta1.ClusterServiceBroker, 0)
					brokers = append(brokers, v1beta1.ClusterServiceBroker{
						ObjectMeta: v1.ObjectMeta{
							UID:  "1234",
							Name: "fake-broker",
						},
						Spec: v1beta1.ClusterServiceBrokerSpec{
							CommonServiceBrokerSpec: v1beta1.CommonServiceBrokerSpec{
								URL: "http://fake.broker.url",
							},
						},
					})
					return &v1beta1.ClusterServiceBrokerList{
						Items: brokers,
					}, nil
				}

				brokers, err := platformClient.GetBrokers(ctx)

				Expect(err).To(BeNil())
				Expect(brokers).ToNot(BeNil())
				Expect(len(brokers)).To(Equal(1))
				Expect(brokers[0].GUID).To(Equal("1234"))
				Expect(brokers[0].Name).To(Equal("fake-broker"))
				Expect(brokers[0].BrokerURL).To(Equal("http://fake.broker.url"))
			})
		})

		Context("when no service brokers are registered", func() {
			It("returns empty array", func() {
				platformClient, err := NewClient(clientConfig)
				Expect(err).ToNot(HaveOccurred())

				retrieveClusterServiceBrokers = func(cli *servicecatalog.SDK) (*v1beta1.ClusterServiceBrokerList, error) {
					brokers := make([]v1beta1.ClusterServiceBroker, 0)
					return &v1beta1.ClusterServiceBrokerList{
						Items: brokers,
					}, nil
				}

				brokers, err := platformClient.GetBrokers(ctx)

				Expect(err).To(BeNil())
				Expect(brokers).ToNot(BeNil())
				Expect(len(brokers)).To(Equal(0))
			})
		})

		Context("with an error", func() {
			It("returns the error", func() {
				platformClient, err := NewClient(clientConfig)
				Expect(err).ToNot(HaveOccurred())

				retrieveClusterServiceBrokers = func(cli *servicecatalog.SDK) (*v1beta1.ClusterServiceBrokerList, error) {
					return nil, errors.New("Error getting clusterservicebrokers")
				}

				brokers, err := platformClient.GetBrokers(ctx)

				Expect(brokers).To(BeNil())
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Error getting clusterservicebrokers"))
			})
		})
	})

	Describe("Update a service broker", func() {
		Context("with no errors", func() {
			It("returns updated broker", func() {
				platformClient, err := NewClient(clientConfig)
				Expect(err).ToNot(HaveOccurred())

				updateClusterServiceBroker = func(cli *servicecatalog.SDK, broker *v1beta1.ClusterServiceBroker) (*v1beta1.ClusterServiceBroker, error) {
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
					GUID:      "1234",
					Name:      "fake-broker",
					BrokerURL: "http://fake.broker.url",
				}

				broker, err := platformClient.UpdateBroker(ctx, requestBroker)

				Expect(err).To(BeNil())
				Expect(broker.GUID).To(Equal("1234"))
				Expect(broker.Name).To(Equal("fake-broker-updated"))
				Expect(broker.BrokerURL).To(Equal("http://fake.broker.url-updated"))
			})
		})

		Context("with an error", func() {
			It("returns the error", func() {
				platformClient, err := NewClient(clientConfig)
				Expect(err).ToNot(HaveOccurred())

				updateClusterServiceBroker = func(cli *servicecatalog.SDK, broker *v1beta1.ClusterServiceBroker) (*v1beta1.ClusterServiceBroker, error) {
					return nil, errors.New("Error updating clusterservicebroker")
				}

				requestBroker := &platform.UpdateServiceBrokerRequest{}

				broker, err := platformClient.UpdateBroker(ctx, requestBroker)

				Expect(broker).To(BeNil())
				Expect(err).To(Equal(errors.New("Error updating clusterservicebroker")))
			})
		})
	})

	Describe("Fetch the catalog information of a service broker", func() {
		Context("with no errors", func() {
			It("returns nil", func() {
				platformClient, err := NewClient(clientConfig)
				Expect(err).ToNot(HaveOccurred())

				requestBroker := &platform.ServiceBroker{
					GUID:      "1234",
					Name:      "fake-broker",
					BrokerURL: "http://fake.broker.url",
				}

				syncClusterServiceBroker = func(cli *servicecatalog.SDK, name string, retries int) error {
					return nil
				}

				err = platformClient.Fetch(ctx, requestBroker)

				Expect(err).To(BeNil())
			})
		})

		Context("with an error", func() {
			It("returns the error", func() {
				platformClient, err := NewClient(clientConfig)
				Expect(err).ToNot(HaveOccurred())

				requestBroker := &platform.ServiceBroker{}
				syncClusterServiceBroker = func(cli *servicecatalog.SDK, name string, retries int) error {
					return errors.New("Error syncing service broker")
				}

				err = platformClient.Fetch(ctx, requestBroker)

				Expect(err).To(Equal(errors.New("Error syncing service broker")))
			})
		})
	})
})
