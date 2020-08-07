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

package broker_test

import (
	"github.com/ghodss/yaml"
	"github.com/golang/mock/gomock"
	osb "github.com/pmorie/go-open-service-broker-client/v2"
	osbbroker "github.com/pmorie/osb-broker-lib/pkg/broker"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/kubernetes-sigs/minibroker/pkg/broker"
	"github.com/kubernetes-sigs/minibroker/pkg/broker/mocks"
	"github.com/kubernetes-sigs/minibroker/pkg/minibroker"
)

//go:generate mockgen -destination=./mocks/mock_broker.go -package=mocks github.com/kubernetes-sigs/minibroker/pkg/broker MinibrokerClient

const (
	overrideParamsYaml = `
mariadb:
  overrideParams:
    mariadb: value
mongodb:
  overrideParams:
    mongodb: value
mysql:
  overrideParams:
    mysql: value
postgresql:
  overrideParams:
    postgresql: value
rabbitmq:
  overrideParams:
    rabbitmq: value
redis:
  overrideParams:
    redis: value
`
)

var _ = Describe("Broker", func() {
	var (
		ctrl *gomock.Controller

		b        *broker.Broker
		mbclient *mocks.MockMinibrokerClient

		provisioningSettings = &broker.ProvisioningSettings{}
		namespace            = "namespace"
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mbclient = mocks.NewMockMinibrokerClient(ctrl)
	})

	JustBeforeEach(func() {
		b = broker.NewBroker(mbclient, namespace, provisioningSettings)
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Describe("Provision", func() {
		var (
			provisionParams = minibroker.NewProvisionParams(map[string]interface{}{
				"key": "value",
			})
			provisionRequest = &osb.ProvisionRequest{
				ServiceID:  "redis",
				Parameters: provisionParams.Object,
			}
			requestContext = &osbbroker.RequestContext{}
		)

		Context("without default chart values", func() {
			It("passes on unaltered provision params", func() {
				mbclient.EXPECT().
					Provision(gomock.Any(), gomock.Eq("redis"), gomock.Any(), gomock.Eq(namespace), gomock.Any(), gomock.Eq(provisionParams))

				b.Provision(provisionRequest, requestContext)
			})
		})

		Context("with default chart values", func() {
			BeforeEach(func() {
				provisioningSettings = &broker.ProvisioningSettings{}
				err := provisioningSettings.LoadYaml([]byte(overrideParamsYaml))
				Expect(err).ToNot(HaveOccurred())
			})

			It("passes on default chart values", func() {
				services := []string{"mariadb", "mongodb", "mysql", "postgresql", "rabbitmq", "redis"}

				for _, service := range services {
					provisionRequest.ServiceID = service
					provisioningSettings, found := provisioningSettings.ForService(service)
					Expect(found).To(BeTrue())
					params := minibroker.NewProvisionParams(provisioningSettings.OverrideParams)

					mbclient.EXPECT().
						Provision(gomock.Any(), gomock.Eq(service), gomock.Any(), gomock.Eq(namespace), gomock.Any(), gomock.Eq(params))

					b.Provision(provisionRequest, requestContext)
				}
			})
		})
	})
})

var _ = Describe("OverrideChartParams", func() {
	Describe("LoadYaml", func() {
		var (
			ocp = &broker.ProvisioningSettings{}
		)

		It("Loads valid data", func() {
			yamlStr, _ := yaml.Marshal(map[string]interface{}{
				"rabbitmq": map[string]interface{}{
					"overrideParams": map[string]interface{}{
						"rabbitmqdata": "thevalue",
					},
				},
			})

			err := ocp.LoadYaml(yamlStr)

			Expect(err).ToNot(HaveOccurred())
			p, _ := ocp.ForService("rabbitmq")
			Expect(p.OverrideParams["rabbitmqdata"]).To(Equal("thevalue"))
		})

		It("returns an error on unknown fields", func() {
			yamlStr, _ := yaml.Marshal(map[string]interface{}{
				"unknownservice": map[string]interface{}{
					"overrideParams": map[string]interface{}{
						"key": "value",
					},
				},
			})

			err := ocp.LoadYaml(yamlStr)

			Expect(err).To(HaveOccurred())
		})
	})
})
