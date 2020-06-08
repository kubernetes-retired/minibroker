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

package helm_test

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/cli-runtime/pkg/genericclioptions"

	"helm.sh/helm/v3/pkg/action"

	"github.com/golang/mock/gomock"
	"github.com/kubernetes-sigs/minibroker/pkg/helm"
	"github.com/kubernetes-sigs/minibroker/pkg/helm/mocks"
	"github.com/kubernetes-sigs/minibroker/pkg/log"
)

var _ = Describe("Config", func() {
	Describe("Config", func() {
		Describe("NewDefaultConfig", func() {
			It("should satisfy the ConfigProvider interface", func() {
				var config helm.ConfigProvider = helm.NewDefaultConfig()
				Expect(config).NotTo(BeNil())
			})
		})

		Describe("Provide", func() {
			var ctrl *gomock.Controller

			BeforeEach(func() {
				ctrl = gomock.NewController(GinkgoT())
			})

			AfterEach(func() {
				ctrl.Finish()
			})

			It("should fail when actionConfig.Init fails", func() {
				actionConfig := mocks.NewMockConfigInitializer(ctrl)
				actionConfig.EXPECT().
					Init(gomock.Any(), "my-namespace", gomock.Any(), gomock.Any()).
					Return(fmt.Errorf("some error")).
					Times(1)
				configInitializerProvider := mocks.NewMockConfigInitializerProvider(ctrl)
				configInitializerProvider.EXPECT().
					Provide().
					Return(actionConfig).
					Times(1)
				config := helm.NewConfig(log.NewNoop(), configInitializerProvider, "", "")
				cfg, err := config.Provide("my-namespace")
				Expect(err).To(Equal(fmt.Errorf("failed to provide action configuration: some error")))
				Expect(cfg).To(BeNil())
			})

			It("should succeed", func() {
				actionConfig := mocks.NewMockConfigInitializer(ctrl)
				actionConfig.EXPECT().
					Init(gomock.Any(), "my-namespace", gomock.Any(), gomock.Any()).
					Do(func(
						_ genericclioptions.RESTClientGetter,
						_ string,
						_ string,
						log action.DebugLog,
					) error {
						log("whatever")
						return nil
					}).
					Return(nil).
					Times(1)
				configInitializerProvider := mocks.NewMockConfigInitializerProvider(ctrl)
				configInitializerProvider.EXPECT().
					Provide().
					Return(actionConfig).
					Times(1)
				config := helm.NewConfig(log.NewNoop(), configInitializerProvider, "", "")
				cfg, err := config.Provide("my-namespace")
				Expect(err).NotTo(HaveOccurred())
				Expect(cfg).To(Equal(actionConfig))
			})
		})
	})

	Describe("ConfigInitializerProvider", func() {
		Describe("NewDefaultConfigInitializerProvider", func() {
			It("should create a ConfigInitializerProvider", func() {
				configInitializerProvider := helm.NewDefaultConfigInitializerProvider()
				Expect(configInitializerProvider).NotTo(BeNil())
			})
		})

		Describe("Provide", func() {
			It("should provide a new pointer instance of action.Configuration", func() {
				configInitializerProvider := helm.NewDefaultConfigInitializerProvider()
				actionConfig := configInitializerProvider.Provide()
				Expect(actionConfig).NotTo(BeNil())
				Expect(*(actionConfig.(*action.Configuration))).To(Equal(action.Configuration{}))
			})
		})
	})
})
