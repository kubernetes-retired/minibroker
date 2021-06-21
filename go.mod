module github.com/kubernetes-sigs/minibroker

go 1.13

require (
	github.com/Masterminds/semver v1.4.0
	github.com/containers/libpod v1.9.3
	github.com/ghodss/yaml v1.0.1-0.20190212211648-25d852aebe32
	github.com/golang/mock v1.5.0
	github.com/onsi/ginkgo v1.15.0
	github.com/onsi/gomega v1.10.5
	github.com/pkg/errors v0.9.1
	github.com/pmorie/go-open-service-broker-client v0.0.0-20180304212357-e8aa16c90363
	github.com/pmorie/osb-broker-lib v0.0.0-20180516212803-87d71cfbf342
	github.com/prometheus/client_golang v1.6.0
	helm.sh/helm/v3 v3.2.3
	k8s.io/api v0.21.2
	k8s.io/apimachinery v0.21.2
	k8s.io/cli-runtime v0.18.0
	k8s.io/client-go v0.21.2
	k8s.io/klog/v2 v2.8.0
	rsc.io/letsencrypt v0.0.3 // indirect
)
