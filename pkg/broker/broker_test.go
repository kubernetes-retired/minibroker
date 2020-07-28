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

	"github.com/kubernetes-sigs/minibroker/pkg/broker"
	"github.com/kubernetes-sigs/minibroker/pkg/broker/mocks"
)

//go:generate mockgen -destination=./mocks/mock_broker.go -package=mocks github.com/kubernetes-sigs/minibroker/pkg/broker MinibrokerClient

var _ = Describe("Broker", func() {
	var (
		ctrl *gomock.Controller

		b                  *broker.Broker
		mbclient           *mocks.MockMinibrokerClient
		defaultChartValues broker.DefaultChartValues

		namespace = "namespace"
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())

		mbclient = mocks.NewMockMinibrokerClient(ctrl)
		defaultChartValues = broker.DefaultChartValues{}
	})

	JustBeforeEach(func() {
		b = broker.NewBroker(mbclient, namespace, defaultChartValues)
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
					Provision(gomock.Any(), gomock.Eq("redis"), gomock.Any(), gomock.Eq(namespace), gomock.Any(), gomock.Eq(provisionParams)).
					Return("foo", nil)

				b.Provision(provisionRequest, requestContext)
			})
		})

		Context("with default chart values", func() {
			BeforeEach(func() {
				defaultChartValues = broker.DefaultChartValues{
					Redis: map[string]interface{}{
						"rediskey": "redisvalue",
					},
				}
			})

			It("passes on default chart values", func() {
				mbclient.EXPECT().
					Provision(gomock.Any(), gomock.Eq("redis"), gomock.Any(), gomock.Eq(namespace), gomock.Any(), gomock.Eq(defaultChartValues.Redis)).
					Return("foo", nil)

				b.Provision(provisionRequest, requestContext)
			})
		})
	})
})
