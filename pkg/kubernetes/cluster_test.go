package kubernetes_test

import (
	"bytes"
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/kubernetes-sigs/minibroker/pkg/kubernetes"
)

var _ = Describe("Cluster", func() {
	It("should fail when reading the resolv.conf reader fails", func() {
		var resolvConf failReader
		clusterDomain, err := kubernetes.ClusterDomain(&resolvConf)
		Ω(clusterDomain).Should(BeZero())
		Ω(err).Should(MatchError("failed to get cluster domain: failed to read"))
	})

	It("should fail when the search path is missing", func() {
		var resolvConf bytes.Buffer
		fmt.Fprintln(&resolvConf, "nameserver 1.2.3.4")
		fmt.Fprintln(&resolvConf, "nameserver 4.3.2.1")
		clusterDomain, err := kubernetes.ClusterDomain(&resolvConf)
		Ω(clusterDomain).Should(BeZero())
		Ω(err).Should(MatchError("failed to get cluster domain: missing the search path from resolv.conf"))
	})

	It("should fail when the search path is missing a domain starting with svc.", func() {
		var resolvConf bytes.Buffer
		fmt.Fprintln(&resolvConf, "nameserver 1.2.3.4")
		fmt.Fprintln(&resolvConf, "nameserver 4.3.2.1")
		fmt.Fprintln(&resolvConf, "search kubecf.svc.cluster.local cluster.local")
		fmt.Fprintln(&resolvConf, "options ndots:5")
		clusterDomain, err := kubernetes.ClusterDomain(&resolvConf)
		Ω(clusterDomain).Should(BeZero())
		Ω(err).Should(MatchError("failed to get cluster domain: missing domain starting with 'svc.' in the search path"))
	})

	It("should succeed returning the cluster domain", func() {
		var resolvConf bytes.Buffer
		fmt.Fprintln(&resolvConf, "nameserver 1.2.3.4")
		fmt.Fprintln(&resolvConf, "nameserver 4.3.2.1")
		fmt.Fprintln(&resolvConf, "search kubecf.svc.cluster.local svc.cluster.local cluster.local")
		fmt.Fprintln(&resolvConf, "options ndots:5")
		clusterDomain, err := kubernetes.ClusterDomain(&resolvConf)
		Ω(clusterDomain).Should(Equal("cluster.local"))
		Ω(err).ShouldNot(HaveOccurred())
	})
})

type failReader struct{}

func (*failReader) Read(_ []byte) (int, error) {
	return 0, fmt.Errorf("failed to read")
}