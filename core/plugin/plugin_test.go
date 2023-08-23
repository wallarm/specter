package plugin

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Default registry", func() {
	BeforeEach(func() {
		Register(ptestType(), ptestPluginName, ptestNewImpl)
	})
	AfterEach(func() {
		defaultRegistry = NewRegistry()
	})
	It("lookup", func() {
		Expect(Lookup(ptestType())).To(BeTrue())
	})
	It("lookup factory", func() {
		Expect(LookupFactory(ptestNewErrType())).To(BeTrue())
	})
	It("new", func() {
		plugin, err := New(ptestType(), ptestPluginName)
		Expect(err).NotTo(HaveOccurred())
		Expect(plugin).NotTo(BeNil())
	})
	It("new factory", func() {
		pluginFactory, err := NewFactory(ptestNewErrType(), ptestPluginName)
		Expect(err).NotTo(HaveOccurred())
		Expect(pluginFactory).NotTo(BeNil())
	})
})

var _ = Describe("type helpers", func() {
	It("ptr type", func() {
		var plugin ptestPlugin
		Expect(PtrType(&plugin)).To(Equal(ptestType()))
	})
	It("factory plugin type ok", func() {
		factoryPlugin, ok := FactoryPluginType(ptestNewErrType())
		Expect(ok).To(BeTrue())
		Expect(factoryPlugin).To(Equal(ptestType()))
	})
})
