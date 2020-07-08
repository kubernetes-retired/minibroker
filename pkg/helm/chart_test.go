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
	"io"
	"net/http"
	"reflect"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/golang/mock/gomock"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/repo"

	"github.com/kubernetes-sigs/minibroker/pkg/helm"
	"github.com/kubernetes-sigs/minibroker/pkg/helm/mocks"
	"github.com/kubernetes-sigs/minibroker/pkg/log"
	nameutilmocks "github.com/kubernetes-sigs/minibroker/pkg/nameutil/mocks"
)

//go:generate mockgen -destination=./mocks/mock_testutil_chart.go -package=mocks github.com/kubernetes-sigs/minibroker/pkg/helm/testutil ChartInstallRunner,ChartUninstallRunner
//go:generate mockgen -destination=./mocks/mock_chart.go -package=mocks github.com/kubernetes-sigs/minibroker/pkg/helm ChartLoader,ChartHelmClientProvider
//go:generate mockgen -destination=./mocks/mock_http.go -package=mocks github.com/kubernetes-sigs/minibroker/pkg/helm HTTPGetter
//go:generate mockgen -destination=./mocks/mock_io.go -package=mocks io ReadCloser

var _ = Describe("Chart", func() {
	var ctrl *gomock.Controller

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Describe("ChartClient", func() {
		Describe("NewDefaultChartClient", func() {
			It("should return a new ChartClient", func() {
				client := helm.NewDefaultChartClient()
				Expect(client).NotTo(BeNil())
			})
		})

		Describe("Install", func() {
			It("should fail when the chartDef.URLs is empty", func() {
				client := helm.NewChartClient(log.NewNoop(), nil, nil, nil)
				chartDef := &repo.ChartVersion{
					Metadata: &chart.Metadata{Name: "foo"},
					URLs:     make([]string, 0),
				}
				release, err := client.Install(chartDef, "", nil)
				Expect(err).To(Equal(fmt.Errorf("failed to install chart: missing chart URL for \"foo\"")))
				Expect(release).To(BeNil())
			})

			It("should fail when loading the chart from the chart manager fails", func() {
				chartURL := "https://foo/bar.tar.gz"
				chartLoader := mocks.NewMockChartLoader(ctrl)
				chartLoader.EXPECT().
					Load(chartURL).
					Return(nil, fmt.Errorf("error from chart loader")).
					Times(1)
				client := helm.NewChartClient(log.NewNoop(), chartLoader, nil, nil)
				chartDef := &repo.ChartVersion{URLs: []string{chartURL}}
				release, err := client.Install(chartDef, "", nil)
				Expect(err).To(Equal(fmt.Errorf("failed to install chart: error from chart loader")))
				Expect(release).To(BeNil())
			})

			It("should fail when the name generator fails", func() {
				chartRequested := &chart.Chart{Metadata: &chart.Metadata{Deprecated: false}}
				chartLoader := mocks.NewMockChartLoader(ctrl)
				chartLoader.EXPECT().
					Load(gomock.Any()).
					Return(chartRequested, nil).
					Times(1)
				nameGenerator := nameutilmocks.NewMockGenerator(ctrl)
				nameGenerator.EXPECT().
					Generate("foo-").
					Return("", fmt.Errorf("error from name generator")).
					Times(1)
				client := helm.NewChartClient(log.NewNoop(), chartLoader, nameGenerator, nil)
				chartDef := &repo.ChartVersion{
					Metadata: &chart.Metadata{Name: "foo"},
					URLs:     []string{"https://foo/bar.tar.gz"},
				}
				release, err := client.Install(chartDef, "", nil)
				Expect(err).To(Equal(fmt.Errorf("failed to install chart: error from name generator")))
				Expect(release).To(BeNil())
			})

			It("should fail when the generated name length exceeds the maximum value", func() {
				chartRequested := &chart.Chart{Metadata: &chart.Metadata{Deprecated: false}}
				releaseName := strings.Repeat("x", 54)
				chartLoader := mocks.NewMockChartLoader(ctrl)
				chartLoader.EXPECT().
					Load(gomock.Any()).
					Return(chartRequested, nil).
					Times(1)
				nameGenerator := nameutilmocks.NewMockGenerator(ctrl)
				nameGenerator.EXPECT().
					Generate(gomock.Any()).
					Return(releaseName, nil).
					Times(1)
				client := helm.NewChartClient(log.NewNoop(), chartLoader, nameGenerator, nil)
				chartDef := &repo.ChartVersion{
					Metadata: &chart.Metadata{Name: "foo"},
					URLs:     []string{"https://foo/bar.tar.gz"},
				}
				release, err := client.Install(chartDef, "", nil)
				Expect(err).To(Equal(fmt.Errorf("failed to install chart: invalid release name %q: names cannot exceed 53 characters", releaseName)))
				Expect(release).To(BeNil())
			})

			It("should fail when getting the helm installer client fails", func() {
				releaseName := "foo-12345"
				namespace := "foo-namespace"
				chartRequested := &chart.Chart{Metadata: &chart.Metadata{Deprecated: false}}
				chartLoader := mocks.NewMockChartLoader(ctrl)
				chartLoader.EXPECT().
					Load(gomock.Any()).
					Return(chartRequested, nil).
					Times(1)
				nameGenerator := nameutilmocks.NewMockGenerator(ctrl)
				nameGenerator.EXPECT().
					Generate(gomock.Any()).
					Return(releaseName, nil).
					Times(1)
				chartHelmClientProvider := mocks.NewMockChartHelmClientProvider(ctrl)
				chartHelmClientProvider.EXPECT().
					ProvideInstaller(releaseName, namespace).
					Return(nil, fmt.Errorf("error from client provider")).
					Times(1)
				client := helm.NewChartClient(log.NewNoop(), chartLoader, nameGenerator, chartHelmClientProvider)
				chartDef := &repo.ChartVersion{
					Metadata: &chart.Metadata{Name: "foo"},
					URLs:     []string{"https://foo/bar.tar.gz"},
				}
				release, err := client.Install(chartDef, namespace, nil)
				Expect(err).To(Equal(fmt.Errorf("failed to install chart: error from client provider")))
				Expect(release).To(BeNil())
			})

			It("should fail when running the install client fails", func() {
				releaseName := "foo-12345"
				namespace := "foo-namespace"
				chartRequested := &chart.Chart{Metadata: &chart.Metadata{Deprecated: false}}
				values := map[string]interface{}{"bar": "baz"}
				chartLoader := mocks.NewMockChartLoader(ctrl)
				chartLoader.EXPECT().
					Load(gomock.Any()).
					Return(chartRequested, nil).
					Times(1)
				nameGenerator := nameutilmocks.NewMockGenerator(ctrl)
				nameGenerator.EXPECT().
					Generate(gomock.Any()).
					Return(releaseName, nil).
					Times(1)
				installRunner := mocks.NewMockChartInstallRunner(ctrl)
				installRunner.EXPECT().
					ChartInstallRunner(chartRequested, values).
					Return(nil, fmt.Errorf("error from client install runner")).
					Times(1)
				chartHelmClientProvider := mocks.NewMockChartHelmClientProvider(ctrl)
				chartHelmClientProvider.EXPECT().
					ProvideInstaller(releaseName, namespace).
					Return(installRunner.ChartInstallRunner, nil).
					Times(1)
				client := helm.NewChartClient(log.NewNoop(), chartLoader, nameGenerator, chartHelmClientProvider)
				chartDef := &repo.ChartVersion{
					Metadata: &chart.Metadata{Name: "foo"},
					URLs:     []string{"https://foo/bar.tar.gz"},
				}
				release, err := client.Install(chartDef, namespace, values)
				Expect(err).To(Equal(fmt.Errorf("failed to install chart: error from client install runner")))
				Expect(release).To(BeNil())
			})

			Describe("Succeeding", func() {
				tests := []struct {
					title      string
					deprecated bool
				}{
					{
						title:      "should install non-deprecated charts",
						deprecated: false,
					},
					{
						title:      "should install deprecated charts",
						deprecated: true,
					},
				}

				for _, t := range tests {
					tt := t
					It(tt.title, func() {
						releaseName := "foo-12345"
						expectedRelease := &release.Release{Name: releaseName}
						namespace := "foo-namespace"
						chartRequested := &chart.Chart{Metadata: &chart.Metadata{Deprecated: tt.deprecated}}
						values := map[string]interface{}{"bar": "baz"}
						chartLoader := mocks.NewMockChartLoader(ctrl)
						chartLoader.EXPECT().
							Load(gomock.Any()).
							Return(chartRequested, nil).
							Times(1)
						nameGenerator := nameutilmocks.NewMockGenerator(ctrl)
						nameGenerator.EXPECT().
							Generate(gomock.Any()).
							Return(releaseName, nil).
							Times(1)
						installRunner := mocks.NewMockChartInstallRunner(ctrl)
						installRunner.EXPECT().
							ChartInstallRunner(chartRequested, values).
							Return(expectedRelease, nil).
							Times(1)
						chartHelmClientProvider := mocks.NewMockChartHelmClientProvider(ctrl)
						chartHelmClientProvider.EXPECT().
							ProvideInstaller(releaseName, namespace).
							Return(installRunner.ChartInstallRunner, nil).
							Times(1)
						client := helm.NewChartClient(log.NewNoop(), chartLoader, nameGenerator, chartHelmClientProvider)
						chartDef := &repo.ChartVersion{
							Metadata: &chart.Metadata{Name: "foo"},
							URLs:     []string{"https://foo/bar.tar.gz"},
						}
						release, err := client.Install(chartDef, namespace, values)
						Expect(err).NotTo(HaveOccurred())
						Expect(release).To(Equal(expectedRelease))
					})
				}
			})
		})

		Describe("Uninstall", func() {
			It("should fail when getting the helm uninstaller client fails", func() {
				releaseName := "foo-12345"
				namespace := "foo-namespace"
				chartHelmClientProvider := mocks.NewMockChartHelmClientProvider(ctrl)
				chartHelmClientProvider.EXPECT().
					ProvideUninstaller(namespace).
					Return(nil, fmt.Errorf("error from client provider")).
					Times(1)
				client := helm.NewChartClient(log.NewNoop(), nil, nil, chartHelmClientProvider)
				err := client.Uninstall(releaseName, namespace)
				Expect(err).To(Equal(fmt.Errorf("failed to uninstall chart: error from client provider")))
			})

			It("should fail when running the uninstall client fails", func() {
				releaseName := "foo-12345"
				namespace := "foo-namespace"
				uninstallRunner := mocks.NewMockChartUninstallRunner(ctrl)
				uninstallRunner.EXPECT().
					ChartUninstallRunner(releaseName).
					Return(nil, fmt.Errorf("error from client uninstall runner")).
					Times(1)
				chartHelmClientProvider := mocks.NewMockChartHelmClientProvider(ctrl)
				chartHelmClientProvider.EXPECT().
					ProvideUninstaller(namespace).
					Return(uninstallRunner.ChartUninstallRunner, nil).
					Times(1)
				client := helm.NewChartClient(log.NewNoop(), nil, nil, chartHelmClientProvider)
				err := client.Uninstall(releaseName, namespace)
				Expect(err).To(Equal(fmt.Errorf("failed to uninstall chart: error from client uninstall runner")))
			})

			It("should succeed uninstalling", func() {
				releaseName := "foo-12345"
				namespace := "foo-namespace"
				uninstallRunner := mocks.NewMockChartUninstallRunner(ctrl)
				uninstallRunner.EXPECT().
					ChartUninstallRunner(releaseName).
					Return(&release.UninstallReleaseResponse{}, nil).
					Times(1)
				chartHelmClientProvider := mocks.NewMockChartHelmClientProvider(ctrl)
				chartHelmClientProvider.EXPECT().
					ProvideUninstaller(namespace).
					Return(uninstallRunner.ChartUninstallRunner, nil).
					Times(1)
				client := helm.NewChartClient(log.NewNoop(), nil, nil, chartHelmClientProvider)
				err := client.Uninstall(releaseName, namespace)
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})

	Describe("ChartManager", func() {
		Describe("NewDefaultChartManager", func() {
			It("should satisfy the ChartLoader interface", func() {
				var chartManager helm.ChartLoader = helm.NewDefaultChartManager()
				Expect(chartManager).NotTo(BeNil())
			})
		})

		Describe("Load", func() {
			It("should fail when downloading the chart fails", func() {
				chartURL := "https://foo/bar.tar.gz"
				httpGetter := mocks.NewMockHTTPGetter(ctrl)
				httpGetter.EXPECT().
					Get(chartURL).
					Return(nil, fmt.Errorf("http error")).
					Times(1)
				chartManager := helm.NewChartManager(httpGetter, nil)
				chart, err := chartManager.Load(chartURL)
				Expect(err).To(Equal(fmt.Errorf("failed to load chart: http error")))
				Expect(chart).To(BeNil())
			})

			It("should fail when loading the chart fails", func() {
				chartURL := "https://foo/bar.tar.gz"
				resBody := mocks.NewMockReadCloser(ctrl)
				resBody.EXPECT().
					Close().
					Times(1)
				httpRes := &http.Response{Body: resBody}
				httpGetter := mocks.NewMockHTTPGetter(ctrl)
				httpGetter.EXPECT().
					Get(chartURL).
					Return(httpRes, nil).
					Times(1)
				loadChartArchive := func(body io.Reader) (*chart.Chart, error) {
					Expect(body).To(Equal(resBody))
					return nil, fmt.Errorf("load chart archive error")
				}
				chartManager := helm.NewChartManager(httpGetter, loadChartArchive)
				chart, err := chartManager.Load(chartURL)
				Expect(err).To(Equal(fmt.Errorf("failed to load chart: load chart archive error")))
				Expect(chart).To(BeNil())
			})

			It("should load a chart", func() {
				chartURL := "https://foo/bar.tar.gz"
				resBody := mocks.NewMockReadCloser(ctrl)
				resBody.EXPECT().
					Close().
					Times(1)
				httpRes := &http.Response{Body: resBody}
				httpGetter := mocks.NewMockHTTPGetter(ctrl)
				httpGetter.EXPECT().
					Get(chartURL).
					Return(httpRes, nil).
					Times(1)
				expectedChart := &chart.Chart{
					Metadata: &chart.Metadata{
						Name: "foo",
					},
				}
				loadChartArchive := func(body io.Reader) (*chart.Chart, error) {
					Expect(body).To(Equal(resBody))
					return expectedChart, nil
				}
				chartManager := helm.NewChartManager(httpGetter, loadChartArchive)
				chart, err := chartManager.Load(chartURL)
				Expect(err).NotTo(HaveOccurred())
				Expect(chart).To(Equal(expectedChart))
			})
		})
	})

	Describe("ChartHelm", func() {
		Describe("NewDefaultChartHelm", func() {
			It("should satisfy the ChartHelmClientProvider interface", func() {
				var chartHelm helm.ChartHelmClientProvider = helm.NewDefaultChartHelm()
				Expect(chartHelm).NotTo(BeNil())
			})
		})

		Describe("ProvideInstaller", func() {
			It("should fail when config provider fails", func() {
				namespace := "foo-namespace"
				configProvider := mocks.NewMockConfigProvider(ctrl)
				configProvider.EXPECT().
					ConfigProvider(namespace).
					Return(nil, fmt.Errorf("error from config provider")).
					Times(1)
				chartHelm := helm.NewChartHelm(configProvider.ConfigProvider, nil, nil)
				installer, err := chartHelm.ProvideInstaller("", namespace)
				Expect(err).To(Equal(fmt.Errorf("failed to provide chart installer: error from config provider")))
				Expect(installer).To(BeNil())
			})

			It("should provide an install runner client", func() {
				releaseName := "foo-12345"
				namespace := "foo-namespace"
				cfg := &action.Configuration{}
				expectedInstaller := &action.Install{
					ReleaseName: releaseName,
					Namespace:   namespace,
				}
				configProvider := mocks.NewMockConfigProvider(ctrl)
				configProvider.EXPECT().
					ConfigProvider(namespace).
					Return(cfg, nil)
				actionNewInstall := func(arg0 *action.Configuration) *action.Install {
					Expect(arg0).To(Equal(cfg))
					return expectedInstaller
				}
				chartHelm := helm.NewChartHelm(configProvider.ConfigProvider, actionNewInstall, nil)
				installer, err := chartHelm.ProvideInstaller(releaseName, namespace)
				Expect(err).NotTo(HaveOccurred())
				Expect(
					reflect.ValueOf(installer).Pointer(),
				).To(Equal(
					reflect.ValueOf(expectedInstaller.Run).Pointer(),
				))
			})
		})

		Describe("ProvideUninstaller", func() {
			It("should fail when config provider fails", func() {
				namespace := "foo-namespace"
				configProvider := mocks.NewMockConfigProvider(ctrl)
				configProvider.EXPECT().
					ConfigProvider(namespace).
					Return(nil, fmt.Errorf("error from config provider"))
				chartHelm := helm.NewChartHelm(configProvider.ConfigProvider, nil, nil)
				uninstaller, err := chartHelm.ProvideUninstaller(namespace)
				Expect(err).To(Equal(fmt.Errorf("failed to provide chart uninstaller: error from config provider")))
				Expect(uninstaller).To(BeNil())
			})

			It("should provide an uninstall runner client", func() {
				namespace := "foo-namespace"
				cfg := &action.Configuration{}
				expectedUninstaller := &action.Uninstall{}
				configProvider := mocks.NewMockConfigProvider(ctrl)
				configProvider.EXPECT().
					ConfigProvider(namespace).
					Return(cfg, nil)
				actionNewUninstall := func(arg0 *action.Configuration) *action.Uninstall {
					Expect(arg0).To(Equal(cfg))
					return expectedUninstaller
				}
				chartHelm := helm.NewChartHelm(configProvider.ConfigProvider, nil, actionNewUninstall)
				uninstaller, err := chartHelm.ProvideUninstaller(namespace)
				Expect(err).NotTo(HaveOccurred())
				Expect(
					reflect.ValueOf(uninstaller).Pointer(),
				).To(Equal(
					reflect.ValueOf(expectedUninstaller.Run).Pointer(),
				))
			})
		})
	})
})
