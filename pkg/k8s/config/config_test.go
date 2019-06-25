package config

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestClient(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Kubernetes Proxy Config Tests Suite")
}

var _ = Describe("Kubernetes Broker Proxy", func() {
	Describe("Config", func() {
		Describe("Validation", func() {
			var config *ClientConfiguration

			BeforeEach(func() {
				config = DefaultClientConfiguration()
				config.Secret.Name = "abc"
				config.Secret.Namespace = "abc"
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
					config.Client = nil
					err := config.Validate()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(Equal("K8S client configuration missing"))
				})
			})

			Context("when LibraryConfig.Timeout is missing", func() {
				It("should fail", func() {
					config.Client.Timeout = 0
					err := config.Validate()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(Equal("K8S client configuration timeout missing"))
				})
			})

			Context("when LibraryConfig.NewClusterConfig is missing", func() {
				It("should fail", func() {
					config.Client.NewClusterConfig = nil
					err := config.Validate()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(Equal("K8S client cluster configuration missing"))
				})
			})

			Context("when Secret is missing", func() {
				It("should fail", func() {
					config.Secret = nil
					err := config.Validate()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(Equal("K8S broker secret missing"))
				})
			})

			Context("when secret Name is missing", func() {
				It("should fail", func() {
					config.Secret.Name = ""
					err := config.Validate()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(Equal("properties of K8S secret configuration for broker registration missing"))
				})
			})

			Context("when secret Namespace is missing", func() {
				It("should fail", func() {
					config.Secret.Namespace = ""
					err := config.Validate()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(Equal("properties of K8S secret configuration for broker registration missing"))
				})
			})

		})
	})
})
