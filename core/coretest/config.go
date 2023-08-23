package coretest

import (
	"github.com/onsi/gomega"
	"github.com/wallarm/specter/core/config"
	"github.com/wallarm/specter/lib/ginkgoutil"
)

func Decode(data string, result interface{}) {
	conf := ginkgoutil.ParseYAML(data)
	err := config.Decode(conf, result)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
}

func DecodeAndValidate(data string, result interface{}) {
	Decode(data, result)
	err := config.Validate(result)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
}
