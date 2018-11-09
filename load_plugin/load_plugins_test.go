package load_plugin


import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Load Plugins", func() {

	Describe("Fetch", func() {
		Context("when the call to UpdateBroker is successful", func() {

			It("returns no error for empty list", func() {
				err := LoadPlugins(make([]string, 0), nil)

				Expect(err).ShouldNot(HaveOccurred())
			})
		})

	})
})
