package k8s

import (
	// "errors"

	// "github.com/Peripli/service-broker-proxy/pkg/platform"

	// "os"

	. "github.com/onsi/ginkgo"

	// "github.com/kubernetes-incubator/service-catalog/pkg/apis/servicecatalog/v1beta1"
	// "github.com/kubernetes-incubator/service-catalog/pkg/svcat/service-catalog"

	. "github.com/onsi/gomega"
	// "k8s.io/apimachinery/pkg/apis/meta/v1"
	// "k8s.io/client-go/rest"
)

var _ = Describe("Kubernetes Broker Proxy", func() {
	Describe("Config", func() {
		Describe("Validation", func() {
			var config *ClientConfiguration

			BeforeEach(func() {
				config = defaultClientConfiguration()
				config.Reg.User = "abc"
				config.Reg.Password = "abc"
				config.Reg.Secret.Name = "abc"
				config.Reg.Secret.Namespace = "abc"
			})

			Context("when all properties available", func() {
				It("should return nil", func() {
					err := config.Validate()
					Expect(err).ToNot(HaveOccurred())
				})
			})

			Context("when ClientCreateFunc is missing", func() {
				It("should fail", func() {
					config.K8sClientCreateFunc = nil
					err := config.Validate()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(Equal("K8S ClientCreateFunc missing"))
				})
			})

			Context("when LibraryConfig is missing", func() {
				It("should fail", func() {
					config.LibraryConfig = nil
					err := config.Validate()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(Equal("K8S client configuration missing"))
				})
			})

			Context("when Reg is missing", func() {
				It("should fail", func() {
					config.Reg = nil
					err := config.Validate()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(Equal("K8S broker registration configuration missing"))
				})
			})

			Context("when Reg user is missing", func() {
				It("should fail", func() {
					config.Reg.User = ""
					err := config.Validate()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(Equal("K8S broker registration credentials missing"))
				})
			})

			Context("when Reg password is missing", func() {
				It("should fail", func() {
					config.Reg.Password = ""
					err := config.Validate()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(Equal("K8S broker registration credentials missing"))
				})
			})

			Context("when Reg secret is missing", func() {
				It("should fail", func() {
					config.Reg.Secret = nil
					err := config.Validate()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(Equal("K8S secret configuration for broker registration missing"))
				})
			})

			Context("when Reg secret Name is missing", func() {
				It("should fail", func() {
					config.Reg.Secret.Name = ""
					err := config.Validate()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(Equal("Properties of K8S secret configuration for broker registration missing"))
				})
			})

			Context("when Reg secret Namespace is missing", func() {
				It("should fail", func() {
					config.Reg.Secret.Namespace = ""
					err := config.Validate()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(Equal("Properties of K8S secret configuration for broker registration missing"))
				})
			})

			Context("when LibraryConfig.HTTPClient is missing", func() {
				It("should fail", func() {
					config.LibraryConfig.HTTPClient = nil
					err := config.Validate()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(Equal("K8S client configuration timeout missing"))
				})
			})

			Context("when LibraryConfig.Timeout is missing", func() {
				It("should fail", func() {
					config.LibraryConfig.Timeout = 0
					err := config.Validate()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(Equal("K8S client configuration timeout missing"))
				})
			})
		})
	})
})
