/*
Copyright 2020 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package integration_test

import (
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/kubernetes-sigs/minibroker/tests/integration/testutil"
)

var _ = Describe("A consumer (wordpress)", func() {
	It("comes up and goes down", func() {
		releaseName := "wordpress"
		namespace := "minibroker-tests"

		pathToChart := os.Getenv("WORDPRESS_CHART")
		_, err := os.Stat(pathToChart)
		Expect(err).ToNot(HaveOccurred())

		h := testutil.NewHelm(namespace)

		err = h.Install(GinkgoWriter, GinkgoWriter, releaseName, pathToChart)
		Expect(err).ToNot(HaveOccurred())

		err = h.Uninstall(GinkgoWriter, GinkgoWriter, releaseName)
		Expect(err).ToNot(HaveOccurred())
	})
})
