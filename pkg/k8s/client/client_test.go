package client

import (
	"context"
	"errors"
	v1core "k8s.io/api/core/v1"
	"testing"

	"github.com/Peripli/service-broker-proxy/pkg/sbproxy"

	"github.com/Peripli/service-broker-proxy-k8s/pkg/k8s/api/apifakes"

	"github.com/Peripli/service-broker-proxy-k8s/pkg/k8s/config"
	"github.com/Peripli/service-broker-proxy/pkg/platform"

	"os"

	"github.com/kubernetes-sigs/service-catalog/pkg/apis/servicecatalog/v1beta1"
	servicecatalog "github.com/kubernetes-sigs/service-catalog/pkg/svcat/service-catalog"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
)

func TestClient(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Kubernetes Proxy Tests Suite")
}

var _ = Describe("Kubernetes Broker Proxy", func() {
	const fakeBrokerName = "fake-broker"
	const fakeBrokerUrl = "http://fake.broker.url"

	var (
		expectedError = errors.New("expected")
		clientConfig  *config.ClientConfiguration
		proxySettings *sbproxy.Settings
		settings      *config.Settings
		ctx           context.Context
		k8sApi        *apifakes.FakeKubernetesAPI
	)

	newDefaultPlatformClient := func() *PlatformClient {
		client, err := NewClient(settings)
		Expect(err).ToNot(HaveOccurred())
		client.platformAPI = k8sApi
		return client
	}

	BeforeSuite(func() {
		Expect(os.Setenv("KUBERNETES_SERVICE_HOST", "test")).ToNot(HaveOccurred())
		Expect(os.Setenv("KUBERNETES_SERVICE_PORT", "1234")).ToNot(HaveOccurred())
	})

	BeforeEach(func() {
		clientConfig = config.DefaultClientConfiguration()
		clientConfig.ClientSettings.NewClusterConfig = func(_ string) (*rest.Config, error) {
			return &rest.Config{
				Host:            "https://fakeme",
				BearerToken:     string("faketoken"),
				TLSClientConfig: rest.TLSClientConfig{},
			}, nil
		}
		clientConfig.Secret.Namespace = "secretNamespace"
		clientConfig.K8sClientCreateFunc = config.NewSvcatSDK

		proxySettings = sbproxy.DefaultSettings()
		proxySettings.Sm.User = "user"
		proxySettings.Sm.Password = "pass"
		proxySettings.Sm.URL = "url"
		proxySettings.Reconcile.LegacyURL = "legacy_url"
		proxySettings.Reconcile.URL = "reconcile_url"

		settings = &config.Settings{
			Settings: *proxySettings,
			K8S:      clientConfig,
		}
		ctx = context.TODO()
		k8sApi = &apifakes.FakeKubernetesAPI{}
	})

	Describe("New Client", func() {
		Context("With invalid config", func() {
			It("should return error", func() {
				settings.K8S = config.DefaultClientConfiguration()
				_, err := NewClient(settings)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("namespace of K8S secret configuration for broker registration missing"))
			})
		})

		Context("With invalid config", func() {
			It("should return error", func() {
				clientConfig.K8sClientCreateFunc = func(libraryConfig *config.LibraryConfig) (*servicecatalog.SDK, error) {
					return nil, expectedError
				}
				_, err := NewClient(settings)
				Expect(err).To(Equal(expectedError))
			})
		})

		Context("With valid config", func() {

			It("should handle broker operations", func() {
				client := newDefaultPlatformClient()
				Expect(client.Broker()).ToNot(BeNil())
			})

			It("should handle catalog fetch operations", func() {
				client := newDefaultPlatformClient()
				Expect(client.CatalogFetcher()).ToNot(BeNil())
			})

			It("should handle visibility operations", func() {
				client := newDefaultPlatformClient()
				Expect(client.Visibility()).ToNot(BeNil())
			})
		})
	})

	Describe("Cluster service broker", func() {
		Describe("Create a service broker", func() {

			Context("with no error", func() {
				It("returns broker", func() {
					platformClient := newDefaultPlatformClient()

					k8sApi.CreateClusterServiceBrokerStub = func(broker *v1beta1.ClusterServiceBroker) (*v1beta1.ClusterServiceBroker, error) {
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
						ID:        "id-in-sm",
						Name:      fakeBrokerName,
						BrokerURL: fakeBrokerUrl,
						Username:  "admin",
						Password:  "admin",
					}

					k8sApi.CreateSecretStub = func(secret2 *v1core.Secret) (secret *v1core.Secret, err error) {
						Expect(secret2.Name).To(Equal(requestBroker.ID))
						Expect(string(secret2.Data["username"])).To(Equal(requestBroker.Username))
						Expect(string(secret2.Data["password"])).To(Equal(requestBroker.Password))
						return secret2, nil
					}
					createdBroker, err := platformClient.CreateBroker(ctx, requestBroker)

					Expect(err).To(BeNil())
					Expect(createdBroker.GUID).To(Equal("1234"))
					Expect(createdBroker.Name).To(Equal(fakeBrokerName))
					Expect(createdBroker.BrokerURL).To(Equal(fakeBrokerUrl))
				})
			})

			Context("with an error", func() {
				It("returns error", func() {
					platformClient := newDefaultPlatformClient()

					k8sApi.CreateSecretStub = func(secret2 *v1core.Secret) (secret *v1core.Secret, err error) {
						return secret2, nil
					}
					k8sApi.CreateClusterServiceBrokerStub = func(broker *v1beta1.ClusterServiceBroker) (*v1beta1.ClusterServiceBroker, error) {
						return nil, errors.New("error from service-catalog")
					}

					requestBroker := &platform.CreateServiceBrokerRequest{}
					createdBroker, err := platformClient.CreateBroker(ctx, requestBroker)

					Expect(createdBroker).To(BeNil())
					Expect(err).To(Equal(errors.New("error from service-catalog")))
				})
			})
		})

		Describe("Delete a service broker", func() {
			Context("with no error", func() {
				It("returns no error", func() {
					platformClient := newDefaultPlatformClient()

					k8sApi.DeleteClusterServiceBrokerStub = func(name string, options *v1.DeleteOptions) error {
						return nil
					}

					requestBroker := &platform.DeleteServiceBrokerRequest{
						ID:   "id-in-sm",
						GUID: "1234",
						Name: fakeBrokerName,
					}

					err := platformClient.DeleteBroker(ctx, requestBroker)
					Expect(err).To(BeNil())
				})
			})

			Context("with an error", func() {
				It("returns the error", func() {
					platformClient := newDefaultPlatformClient()

					k8sApi.DeleteClusterServiceBrokerStub = func(name string, options *v1.DeleteOptions) error {
						return errors.New("error deleting clusterservicebroker")
					}

					requestBroker := &platform.DeleteServiceBrokerRequest{}

					err := platformClient.DeleteBroker(ctx, requestBroker)
					Expect(err).To(Equal(errors.New("error deleting clusterservicebroker")))
				})
			})
		})

		Describe("Get all service brokers", func() {
			Context("with no error", func() {
				It("returns brokers", func() {
					platformClient := newDefaultPlatformClient()

					k8sApi.RetrieveClusterServiceBrokersStub = func() (*v1beta1.ClusterServiceBrokerList, error) {
						brokers := make([]v1beta1.ClusterServiceBroker, 0)
						brokers = append(brokers, v1beta1.ClusterServiceBroker{
							ObjectMeta: v1.ObjectMeta{
								UID:  "1234",
								Name: fakeBrokerName,
							},
							Spec: v1beta1.ClusterServiceBrokerSpec{
								CommonServiceBrokerSpec: v1beta1.CommonServiceBrokerSpec{
									URL: fakeBrokerUrl,
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
					Expect(brokers[0].Name).To(Equal(fakeBrokerName))
					Expect(brokers[0].BrokerURL).To(Equal(fakeBrokerUrl))
				})
			})

			Context("when no service brokers are registered", func() {
				It("returns empty array", func() {
					platformClient := newDefaultPlatformClient()

					k8sApi.RetrieveClusterServiceBrokersStub = func() (*v1beta1.ClusterServiceBrokerList, error) {
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
					platformClient := newDefaultPlatformClient()

					k8sApi.RetrieveClusterServiceBrokersStub = func() (*v1beta1.ClusterServiceBrokerList, error) {
						return nil, errors.New("error getting clusterservicebrokers")
					}

					brokers, err := platformClient.GetBrokers(ctx)

					Expect(brokers).To(BeNil())
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("error getting clusterservicebrokers"))
				})
			})
		})

		Describe("Get service broker by name", func() {
			Context("with no error", func() {
				It("returns the service broker", func() {
					platformClient := newDefaultPlatformClient()

					k8sApi.RetrieveClusterServiceBrokerByNameStub = func(name string) (*v1beta1.ClusterServiceBroker, error) {
						return &v1beta1.ClusterServiceBroker{
							ObjectMeta: v1.ObjectMeta{
								UID:  "1234",
								Name: fakeBrokerName,
							},
							Spec: v1beta1.ClusterServiceBrokerSpec{
								CommonServiceBrokerSpec: v1beta1.CommonServiceBrokerSpec{
									URL: fakeBrokerUrl,
								},
							},
						}, nil
					}

					broker, err := platformClient.GetBrokerByName(ctx, fakeBrokerName)

					Expect(err).To(BeNil())
					Expect(broker).ToNot(BeNil())
					Expect(broker.Name).To(Equal(fakeBrokerName))
				})
			})

			Context("with an error", func() {
				It("returns the error", func() {
					platformClient := newDefaultPlatformClient()

					k8sApi.RetrieveClusterServiceBrokerByNameStub = func(name string) (*v1beta1.ClusterServiceBroker, error) {
						return nil, errors.New("error getting clusterservicebroker")
					}

					broker, err := platformClient.GetBrokerByName(ctx, "brokerName")

					Expect(broker).To(BeNil())
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("error getting clusterservicebroker"))
				})
			})
		})

		Describe("Update a service broker", func() {
			Context("with no errors", func() {
				It("returns updated broker", func() {
					platformClient := newDefaultPlatformClient()

					k8sApi.UpdateClusterServiceBrokerStub = func(broker *v1beta1.ClusterServiceBroker) (*v1beta1.ClusterServiceBroker, error) {
						// Return a new fake clusterservicebroker with the three attributes relevant for the OSBAPI guid, name and broker url.
						// UID and name cannot be modified, url can be modified
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
						ID:        "id-in-sm",
						GUID:      "1234",
						Name:      fakeBrokerName,
						BrokerURL: fakeBrokerUrl,
						Username:  "admin",
						Password:  "admin",
					}

					k8sApi.UpdateServiceBrokerCredentialsStub = func(secret2 *v1core.Secret) (secret *v1core.Secret, err error) {
						Expect(secret2.Name).To(Equal(requestBroker.ID))
						Expect(string(secret2.Data["username"])).To(Equal(requestBroker.Username))
						Expect(string(secret2.Data["password"])).To(Equal(requestBroker.Password))
						return secret2, nil
					}

					broker, err := platformClient.UpdateBroker(ctx, requestBroker)

					Expect(err).To(BeNil())
					Expect(broker.GUID).To(Equal("1234"))
					Expect(broker.Name).To(Equal(fakeBrokerName + "-updated"))
					Expect(broker.BrokerURL).To(Equal(fakeBrokerUrl + "-updated"))
				})
			})

			Context("with an error", func() {
				It("returns the error", func() {
					platformClient := newDefaultPlatformClient()

					k8sApi.UpdateServiceBrokerCredentialsStub = func(secret2 *v1core.Secret) (secret *v1core.Secret, err error) {
						return secret2, nil
					}
					k8sApi.UpdateClusterServiceBrokerStub = func(broker *v1beta1.ClusterServiceBroker) (*v1beta1.ClusterServiceBroker, error) {
						return nil, errors.New("error updating clusterservicebroker")
					}

					requestBroker := &platform.UpdateServiceBrokerRequest{}

					broker, err := platformClient.UpdateBroker(ctx, requestBroker)

					Expect(broker).To(BeNil())
					Expect(err).To(Equal(errors.New("error updating clusterservicebroker")))
				})
			})
		})

		Describe("Fetch the catalog information of a service broker", func() {
			Context("with no errors", func() {
				It("returns nil", func() {
					platformClient := newDefaultPlatformClient()

					requestBroker := &platform.UpdateServiceBrokerRequest{
						ID:        "id-in-sm",
						GUID:      "1234",
						Name:      fakeBrokerName,
						BrokerURL: fakeBrokerUrl,
						Username:  "admin",
						Password:  "admin",
					}

					k8sApi.UpdateServiceBrokerCredentialsStub = func(secret2 *v1core.Secret) (secret *v1core.Secret, err error) {
						Expect(secret2.Name).To(Equal(requestBroker.ID))
						Expect(string(secret2.Data["username"])).To(Equal(requestBroker.Username))
						Expect(string(secret2.Data["password"])).To(Equal(requestBroker.Password))
						return secret2, nil
					}
					k8sApi.SyncClusterServiceBrokerStub = func(name string, retries int) error {
						return nil
					}

					err := platformClient.Fetch(ctx, requestBroker)

					Expect(err).To(BeNil())
				})
			})

			Context("with an error", func() {
				It("returns the error", func() {
					platformClient := newDefaultPlatformClient()

					requestBroker := &platform.UpdateServiceBrokerRequest{}
					k8sApi.UpdateServiceBrokerCredentialsStub = func(secret2 *v1core.Secret) (secret *v1core.Secret, err error) {
						return secret2, nil
					}
					k8sApi.SyncClusterServiceBrokerStub = func(name string, retries int) error {
						return errors.New("error syncing service broker")
					}

					err := platformClient.Fetch(ctx, requestBroker)

					Expect(err).To(Equal(errors.New("error syncing service broker")))
				})
			})
		})

		Describe("EnableAccessForPlan", func() {
			It("should call Fetch", func() {
				platformClient := newDefaultPlatformClient()
				k8sApi.SyncClusterServiceBrokerStub = func(name string, retries int) error {
					return expectedError
				}
				Expect(platformClient.EnableAccessForPlan(ctx, &platform.ModifyPlanAccessRequest{})).To(Equal(expectedError))
			})
		})

		Describe("DisableAccessForPlan", func() {

			It("should call Fetch", func() {
				platformClient := newDefaultPlatformClient()
				k8sApi.SyncClusterServiceBrokerStub = func(name string, retries int) error {
					return expectedError
				}
				Expect(platformClient.DisableAccessForPlan(ctx, &platform.ModifyPlanAccessRequest{})).To(Equal(expectedError))
			})
		})

		Describe("Concurrent Modification", func() {
			It("visibility for the same broker", func() {
				NewClient(settings)
				platformClient, err := NewClient(settings)
				Expect(err).ToNot(HaveOccurred())
				scat := platformClient.platformAPI.(*ServiceCatalogAPI)
				scat.setBrokerInProgress("test")
				expectedError := platformClient.platformAPI.SyncClusterServiceBroker("test", 1)
				Expect(expectedError).NotTo(HaveOccurred())
				scat.unsetBrokerInProgress("test")
			})
		})
	})

	Describe("Namespace service broker", func() {
		BeforeEach(func() {
			settings.K8S.TargetNamespace = "test-namespace"
		})
		Describe("Create a service broker", func() {

			Context("with no error", func() {
				It("returns broker", func() {
					platformClient := newDefaultPlatformClient()

					k8sApi.CreateNamespaceServiceBrokerStub = func(broker *v1beta1.ServiceBroker, namespace string) (*v1beta1.ServiceBroker, error) {
						return &v1beta1.ServiceBroker{
							ObjectMeta: v1.ObjectMeta{
								UID:  "1234",
								Name: broker.Name,
							},
							Spec: v1beta1.ServiceBrokerSpec{
								CommonServiceBrokerSpec: v1beta1.CommonServiceBrokerSpec{
									URL: broker.Spec.URL,
								},
							},
						}, nil
					}

					requestBroker := &platform.CreateServiceBrokerRequest{
						ID:        "id-in-sm",
						Name:      fakeBrokerName,
						BrokerURL: fakeBrokerUrl,
						Username:  "admin",
						Password:  "admin",
					}

					k8sApi.CreateSecretStub = func(secret2 *v1core.Secret) (secret *v1core.Secret, err error) {
						Expect(secret2.Name).To(Equal(requestBroker.ID))
						Expect(string(secret2.Data["username"])).To(Equal(requestBroker.Username))
						Expect(string(secret2.Data["password"])).To(Equal(requestBroker.Password))
						return secret2, nil
					}
					createdBroker, err := platformClient.CreateBroker(ctx, requestBroker)

					Expect(err).To(BeNil())
					Expect(createdBroker.GUID).To(Equal("1234"))
					Expect(createdBroker.Name).To(Equal(fakeBrokerName))
					Expect(createdBroker.BrokerURL).To(Equal(fakeBrokerUrl))
				})
			})

			Context("with an error", func() {
				It("returns error", func() {
					platformClient := newDefaultPlatformClient()

					k8sApi.CreateSecretStub = func(secret2 *v1core.Secret) (secret *v1core.Secret, err error) {
						return secret2, nil
					}
					k8sApi.CreateNamespaceServiceBrokerStub = func(broker *v1beta1.ServiceBroker, namespace string) (*v1beta1.ServiceBroker, error) {
						return nil, errors.New("error from service-catalog")
					}

					requestBroker := &platform.CreateServiceBrokerRequest{}
					createdBroker, err := platformClient.CreateBroker(ctx, requestBroker)

					Expect(createdBroker).To(BeNil())
					Expect(err).To(Equal(errors.New("error from service-catalog")))
				})
			})
		})

		Describe("Delete a service broker", func() {
			Context("with no error", func() {
				It("returns no error", func() {
					platformClient := newDefaultPlatformClient()

					k8sApi.DeleteNamespaceServiceBrokerStub = func(name, namespace string, options *v1.DeleteOptions) error {
						return nil
					}

					requestBroker := &platform.DeleteServiceBrokerRequest{
						ID:   "id-in-sm",
						GUID: "1234",
						Name: fakeBrokerName,
					}

					err := platformClient.DeleteBroker(ctx, requestBroker)

					Expect(err).To(BeNil())
				})
			})

			Context("with an error", func() {
				It("returns the error", func() {
					platformClient := newDefaultPlatformClient()

					k8sApi.DeleteNamespaceServiceBrokerStub = func(name, namespace string, options *v1.DeleteOptions) error {
						return errors.New("error deleting servicebroker")
					}

					requestBroker := &platform.DeleteServiceBrokerRequest{}

					err := platformClient.DeleteBroker(ctx, requestBroker)

					Expect(err).To(Equal(errors.New("error deleting servicebroker")))
				})
			})
		})

		Describe("Get all service brokers", func() {
			Context("with no error", func() {
				It("returns brokers", func() {
					platformClient := newDefaultPlatformClient()

					k8sApi.RetrieveNamespaceServiceBrokersStub = func(namespace string) (*v1beta1.ServiceBrokerList, error) {
						brokers := make([]v1beta1.ServiceBroker, 0)
						brokers = append(brokers, v1beta1.ServiceBroker{
							ObjectMeta: v1.ObjectMeta{
								UID:  "1234",
								Name: fakeBrokerName,
							},
							Spec: v1beta1.ServiceBrokerSpec{
								CommonServiceBrokerSpec: v1beta1.CommonServiceBrokerSpec{
									URL: fakeBrokerUrl,
								},
							},
						})
						return &v1beta1.ServiceBrokerList{
							Items: brokers,
						}, nil
					}

					brokers, err := platformClient.GetBrokers(ctx)

					Expect(err).To(BeNil())
					Expect(brokers).ToNot(BeNil())
					Expect(len(brokers)).To(Equal(1))
					Expect(brokers[0].GUID).To(Equal("1234"))
					Expect(brokers[0].Name).To(Equal(fakeBrokerName))
					Expect(brokers[0].BrokerURL).To(Equal(fakeBrokerUrl))
				})
			})

			Context("when no service brokers are registered", func() {
				It("returns empty array", func() {
					platformClient := newDefaultPlatformClient()

					k8sApi.RetrieveNamespaceServiceBrokersStub = func(namespace string) (*v1beta1.ServiceBrokerList, error) {
						brokers := make([]v1beta1.ServiceBroker, 0)
						return &v1beta1.ServiceBrokerList{
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
					platformClient := newDefaultPlatformClient()

					k8sApi.RetrieveNamespaceServiceBrokersStub = func(namespace string) (*v1beta1.ServiceBrokerList, error) {
						return nil, errors.New("error getting servicebrokers")
					}

					brokers, err := platformClient.GetBrokers(ctx)

					Expect(brokers).To(BeNil())
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("error getting servicebrokers"))
				})
			})
		})

		Describe("Get service broker by name", func() {
			Context("with no error", func() {
				It("returns the service broker", func() {
					platformClient := newDefaultPlatformClient()

					k8sApi.RetrieveNamespaceServiceBrokerByNameStub = func(name, namespace string) (*v1beta1.ServiceBroker, error) {
						return &v1beta1.ServiceBroker{
							ObjectMeta: v1.ObjectMeta{
								UID:  "1234",
								Name: fakeBrokerName,
							},
							Spec: v1beta1.ServiceBrokerSpec{
								CommonServiceBrokerSpec: v1beta1.CommonServiceBrokerSpec{
									URL: fakeBrokerUrl,
								},
							},
						}, nil
					}

					broker, err := platformClient.GetBrokerByName(ctx, fakeBrokerName)

					Expect(err).To(BeNil())
					Expect(broker).ToNot(BeNil())
					Expect(broker.Name).To(Equal(fakeBrokerName))
				})
			})

			Context("with an error", func() {
				It("returns the error", func() {
					platformClient := newDefaultPlatformClient()

					k8sApi.RetrieveNamespaceServiceBrokerByNameStub = func(name, namespace string) (*v1beta1.ServiceBroker, error) {
						return nil, errors.New("error getting servicebroker")
					}

					broker, err := platformClient.GetBrokerByName(ctx, "brokerName")

					Expect(broker).To(BeNil())
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("error getting servicebroker"))
				})
			})
		})

		Describe("Update a service broker", func() {
			Context("with no errors", func() {
				It("returns updated broker", func() {
					platformClient := newDefaultPlatformClient()

					k8sApi.UpdateNamespaceServiceBrokerStub = func(broker *v1beta1.ServiceBroker, namespace string) (*v1beta1.ServiceBroker, error) {
						// Return a new fake clusterservicebroker with the three attributes relevant for the OSBAPI guid, name and broker url.
						// UID and name cannot be modified, url can be modified
						return &v1beta1.ServiceBroker{
							ObjectMeta: v1.ObjectMeta{
								Name: broker.Name + "-updated",
								UID:  "1234",
							},
							Spec: v1beta1.ServiceBrokerSpec{
								CommonServiceBrokerSpec: v1beta1.CommonServiceBrokerSpec{
									URL: broker.Spec.CommonServiceBrokerSpec.URL + "-updated",
								},
							},
						}, nil
					}

					requestBroker := &platform.UpdateServiceBrokerRequest{
						ID:        "id-in-sm",
						GUID:      "1234",
						Name:      fakeBrokerName,
						BrokerURL: fakeBrokerUrl,
						Username:  "admin",
						Password:  "admin",
					}

					k8sApi.UpdateServiceBrokerCredentialsStub = func(secret2 *v1core.Secret) (secret *v1core.Secret, err error) {
						Expect(secret2.Name).To(Equal(requestBroker.ID))
						Expect(string(secret2.Data["username"])).To(Equal(requestBroker.Username))
						Expect(string(secret2.Data["password"])).To(Equal(requestBroker.Password))
						return secret2, nil
					}

					broker, err := platformClient.UpdateBroker(ctx, requestBroker)

					Expect(err).To(BeNil())
					Expect(broker.GUID).To(Equal("1234"))
					Expect(broker.Name).To(Equal(fakeBrokerName + "-updated"))
					Expect(broker.BrokerURL).To(Equal(fakeBrokerUrl + "-updated"))
				})
			})

			Context("with an error", func() {
				It("returns the error", func() {
					platformClient := newDefaultPlatformClient()

					k8sApi.UpdateServiceBrokerCredentialsStub = func(secret2 *v1core.Secret) (secret *v1core.Secret, err error) {
						return secret2, nil
					}
					k8sApi.UpdateNamespaceServiceBrokerStub = func(broker *v1beta1.ServiceBroker, namespace string) (*v1beta1.ServiceBroker, error) {
						return nil, errors.New("error updating servicebroker")
					}

					requestBroker := &platform.UpdateServiceBrokerRequest{}

					broker, err := platformClient.UpdateBroker(ctx, requestBroker)

					Expect(broker).To(BeNil())
					Expect(err).To(Equal(errors.New("error updating servicebroker")))
				})
			})
		})

		Describe("Fetch the catalog information of a service broker", func() {
			Context("with no errors", func() {
				It("returns nil", func() {
					platformClient := newDefaultPlatformClient()

					requestBroker := &platform.UpdateServiceBrokerRequest{
						ID:        "id-in-sm",
						GUID:      "1234",
						Name:      fakeBrokerName,
						BrokerURL: fakeBrokerUrl,
						Username:  "admin",
						Password:  "admin",
					}

					k8sApi.UpdateServiceBrokerCredentialsStub = func(secret2 *v1core.Secret) (secret *v1core.Secret, err error) {
						Expect(secret2.Name).To(Equal(requestBroker.ID))
						Expect(string(secret2.Data["username"])).To(Equal(requestBroker.Username))
						Expect(string(secret2.Data["password"])).To(Equal(requestBroker.Password))
						return secret2, nil
					}
					k8sApi.SyncNamespaceServiceBrokerStub = func(name, namespace string, retries int) error {
						return nil
					}

					err := platformClient.Fetch(ctx, requestBroker)

					Expect(err).To(BeNil())
				})
			})

			Context("with an error", func() {
				It("returns the error", func() {
					platformClient := newDefaultPlatformClient()

					requestBroker := &platform.UpdateServiceBrokerRequest{}
					k8sApi.UpdateServiceBrokerCredentialsStub = func(secret2 *v1core.Secret) (secret *v1core.Secret, err error) {
						return secret2, nil
					}
					k8sApi.SyncNamespaceServiceBrokerStub = func(name, namespace string, retries int) error {
						return errors.New("error syncing service broker")
					}

					err := platformClient.Fetch(ctx, requestBroker)

					Expect(err).To(Equal(errors.New("error syncing service broker")))
				})
			})
		})

		Describe("EnableAccessForPlan", func() {
			It("should call Fetch", func() {
				platformClient := newDefaultPlatformClient()
				k8sApi.SyncNamespaceServiceBrokerStub = func(name, namespace string, retries int) error {
					return expectedError
				}
				Expect(platformClient.EnableAccessForPlan(ctx, &platform.ModifyPlanAccessRequest{})).To(Equal(expectedError))
			})
		})

		Describe("DisableAccessForPlan", func() {

			It("should call Fetch", func() {
				platformClient := newDefaultPlatformClient()
				k8sApi.SyncNamespaceServiceBrokerStub = func(name, namespace string, retries int) error {
					return expectedError
				}
				Expect(platformClient.DisableAccessForPlan(ctx, &platform.ModifyPlanAccessRequest{})).To(Equal(expectedError))
			})
		})

		Describe("Concurrent Modification", func() {
			It("visibility for the same broker", func() {
				NewClient(settings)
				platformClient, err := NewClient(settings)
				Expect(err).ToNot(HaveOccurred())
				scat := platformClient.platformAPI.(*ServiceCatalogAPI)
				scat.setBrokerInProgress("test")
				expectedError := platformClient.platformAPI.SyncNamespaceServiceBroker("test", "test-namespace", 1)
				Expect(expectedError).NotTo(HaveOccurred())
				scat.unsetBrokerInProgress("test")
			})
		})
	})

	Describe("GetVisibilitiesByBrokers", func() {
		It("returns no visibilities", func() {
			platformClient := newDefaultPlatformClient()
			visibilities, err := platformClient.GetVisibilitiesByBrokers(ctx, []string{})
			Expect(err).To(BeNil())
			Expect(visibilities).To(BeNil())
		})
	})

	Describe("VisibilityScopeLabelKey", func() {
		It("returns empty string", func() {
			Expect(newDefaultPlatformClient().VisibilityScopeLabelKey()).To(BeEmpty())
		})
	})

	Describe("Platform Broker Name", func() {
		It("returns lower case and replaces underscores to hyphens", func() {
			brokerNameWithUnderscoreAndCaps := "Fake_Broker-Name_1234"
			expectedBrokerName := "fake-broker-name-1234"
			platformClient := newDefaultPlatformClient()
			Expect(platformClient.GetBrokerPlatformName(brokerNameWithUnderscoreAndCaps)).To(Equal(expectedBrokerName))
		})
	})
})
