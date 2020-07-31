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
	"github.com/golang/mock/gomock"
	osb "github.com/pmorie/go-open-service-broker-client/v2"
	osbbroker "github.com/pmorie/osb-broker-lib/pkg/broker"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/kubernetes-sigs/minibroker/pkg/broker"
	"github.com/kubernetes-sigs/minibroker/pkg/broker/mocks"
)

//go:generate mockgen -destination=./mocks/mock_broker.go -package=mocks github.com/kubernetes-sigs/minibroker/pkg/broker MinibrokerClient

const (
	overrideParamsYaml = `provisioning:
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

		overrideChartParams = &broker.OverrideChartParams{}
		namespace           = "namespace"
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mbclient = mocks.NewMockMinibrokerClient(ctrl)
	})

	JustBeforeEach(func() {
		b = broker.NewBroker(mbclient, namespace, overrideChartParams)
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Describe("Provision", func() {
		var (
			provisionParams = map[string]interface{}{
				"key": "value",
			}
			provisionRequest = &osb.ProvisionRequest{
				ServiceID:  "redis",
				Parameters: provisionParams,
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
				overrideChartParams = &broker.OverrideChartParams{}
				err := overrideChartParams.LoadYaml([]byte(overrideParamsYaml))
				Expect(err).ToNot(HaveOccurred())
			})

			It("passes on default chart values", func() {
				services := []string{"mariadb", "mongodb", "mysql", "postgresql", "rabbitmq", "redis"}

				for _, service := range services {
					provisionRequest.ServiceID = service
					expectedValues, found := overrideChartParams.ForService(service)
					Expect(found).To(BeTrue())

					mbclient.EXPECT().
						Provision(gomock.Any(), gomock.Eq(service), gomock.Any(), gomock.Eq(namespace), gomock.Any(), gomock.Eq(expectedValues))

					b.Provision(provisionRequest, requestContext)
				}
			})
		})
	})
})

var _ = Describe("OverrideChartParams", func() {
	Describe("LoadYaml", func() {
		var (
			ocp = &broker.OverrideChartParams{}
		)

		It("Loads valid data", func() {
			yaml := []byte(`rabbitmq:
  rabbitmqdata: thevalue`)

			err := ocp.LoadYaml(yaml)

			Expect(err).ToNot(HaveOccurred())
			p, _ := ocp.ForService("rabbitmq")
			Expect(p["rabbitmqdata"]).To(Equal("thevalue"))
		})

		It("returns an error on unknown fields", func() {
			yaml := []byte(`unknownservice:
  key: value`)

			err := ocp.LoadYaml(yaml)

			Expect(err).To(HaveOccurred())
		})
	})
})
