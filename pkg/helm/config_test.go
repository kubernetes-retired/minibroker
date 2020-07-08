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
	"reflect"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/cli-runtime/pkg/genericclioptions"

	"helm.sh/helm/v3/pkg/action"

	"github.com/kubernetes-sigs/minibroker/pkg/helm"
	"github.com/kubernetes-sigs/minibroker/pkg/helm/mocks"
	"github.com/kubernetes-sigs/minibroker/pkg/log"
)

//go:generate mockgen -destination=./mocks/mock_config.go -package=mocks github.com/kubernetes-sigs/minibroker/pkg/helm/testutil ConfigProvider,ConfigInitializer,ConfigInitializerProvider

var _ = Describe("Config", func() {
	Describe("Config", func() {
		Describe("NewDefaultConfigProvider", func() {
			It("should return a new ConfigProvider", func() {
				var config helm.ConfigProvider = helm.NewDefaultConfigProvider()
				Expect(config).NotTo(BeNil())
			})
		})

		Describe("ConfigProvider", func() {
			var ctrl *gomock.Controller

			BeforeEach(func() {
				ctrl = gomock.NewController(GinkgoT())
			})

			AfterEach(func() {
				ctrl.Finish()
			})

			It("should fail when ConfigInitializer fails", func() {
				namespace := "my-namespace"
				configInitializer := mocks.NewMockConfigInitializer(ctrl)
				configInitializer.EXPECT().
					ConfigInitializer(gomock.Any(), namespace, gomock.Any(), gomock.Any()).
					Return(fmt.Errorf("some error")).
					Times(1)
				configInitializerProvider := mocks.NewMockConfigInitializerProvider(ctrl)
				configInitializerProvider.EXPECT().
					ConfigInitializerProvider().
					Return(nil, configInitializer.ConfigInitializer).
					Times(1)

				configProvider := helm.NewConfigProvider(
					log.NewNoop(),
					configInitializerProvider.ConfigInitializerProvider,
					"",
					"",
				)
				cfg, err := configProvider(namespace)
				Expect(err).To(Equal(fmt.Errorf("failed to provide action configuration: some error")))
				Expect(cfg).To(BeNil())
			})

			It("should succeed", func() {
				namespace := "my-namespace"
				actionConfig := &action.Configuration{}
				configInitializer := mocks.NewMockConfigInitializer(ctrl)
				configInitializer.EXPECT().
					ConfigInitializer(gomock.Any(), namespace, gomock.Any(), gomock.Any()).
					Do(func(
						_ genericclioptions.RESTClientGetter,
						_ string,
						_ string,
						log action.DebugLog,
					) error {
						log("whatever")
						return nil
					}).
					Times(1)
				configInitializerProvider := mocks.NewMockConfigInitializerProvider(ctrl)
				configInitializerProvider.EXPECT().
					ConfigInitializerProvider().
					Return(actionConfig, configInitializer.ConfigInitializer).
					Times(1)

				configProvider := helm.NewConfigProvider(
					log.NewNoop(),
					configInitializerProvider.ConfigInitializerProvider,
					"",
					"",
				)
				cfg, err := configProvider(namespace)
				Expect(err).NotTo(HaveOccurred())
				Expect(cfg).To(Equal(actionConfig))
			})
		})
	})

	Describe("DefaultConfigInitializerProvider", func() {
		It("should return a new action.Configuration and its Init method", func() {
			config, initializer := helm.DefaultConfigInitializerProvider()
			Expect(config).NotTo(BeNil())
			Expect(
				reflect.ValueOf(initializer).Pointer(),
			).To(Equal(
				reflect.ValueOf(config.Init).Pointer(),
			))
		})
	})
})
