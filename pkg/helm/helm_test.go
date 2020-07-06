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

	"github.com/golang/mock/gomock"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/repo"

	"github.com/kubernetes-sigs/minibroker/pkg/helm"
	"github.com/kubernetes-sigs/minibroker/pkg/helm/mocks"
	"github.com/kubernetes-sigs/minibroker/pkg/log"
)

var _ = Describe("Helm", func() {
	Context("Client", func() {
		Describe("NewDefaultClient", func() {
			It("should create a new Client", func() {
				client := helm.NewDefaultClient()
				Expect(client).NotTo(BeNil())
			})
		})

		Describe("Initialize", func() {
			var ctrl *gomock.Controller

			BeforeEach(func() {
				ctrl = gomock.NewController(GinkgoT())
			})

			AfterEach(func() {
				ctrl.Finish()
			})

			It("should fail when repoInitializer.Initialize fails", func() {
				repoClient := mocks.NewMockRepositoryInitializeDownloadLoader(ctrl)
				repoClient.EXPECT().
					Initialize(gomock.Any(), gomock.Any()).
					Return(nil, fmt.Errorf("amazing repoInitializer failure")).
					Times(1)

				client := helm.NewClient(
					log.NewNoop(),
					repoClient,
					nil,
				)
				err := client.Initialize("")
				Expect(err).To(Equal(fmt.Errorf("failed to initialize helm client: amazing repoInitializer failure")))
			})

			It("should fail when repoDownloader.DownloadIndex fails", func() {
				repoClient := mocks.NewMockRepositoryInitializeDownloadLoader(ctrl)
				chartRepo := &repo.ChartRepository{}
				repoClient.EXPECT().
					Initialize(gomock.Any(), gomock.Any()).
					Return(chartRepo, nil).
					Times(1)
				repoClient.EXPECT().
					DownloadIndex(chartRepo).
					Return("", fmt.Errorf("awesome repoDownloader error")).
					Times(1)

				client := helm.NewClient(
					log.NewNoop(),
					repoClient,
					nil,
				)
				err := client.Initialize("")
				Expect(err).To(Equal(fmt.Errorf("failed to initialize helm client: awesome repoDownloader error")))
			})

			It("should fail when repoLoader.Load fails", func() {
				repoClient := mocks.NewMockRepositoryInitializeDownloadLoader(ctrl)
				chartRepo := &repo.ChartRepository{}
				repoClient.EXPECT().
					Initialize(gomock.Any(), gomock.Any()).
					Return(chartRepo, nil).
					Times(1)
				indexPath := "some_path.yaml"
				repoClient.EXPECT().
					DownloadIndex(chartRepo).
					Return(indexPath, nil).
					Times(1)
				repoClient.EXPECT().
					Load(indexPath).
					Return(nil, fmt.Errorf("marvelous repoLoader fault")).
					Times(1)

				client := helm.NewClient(
					log.NewNoop(),
					repoClient,
					nil,
				)
				err := client.Initialize("")
				Expect(err).To(Equal(fmt.Errorf("failed to initialize helm client: marvelous repoLoader fault")))
			})

			It("should succeed", func() {
				repoClient := mocks.NewMockRepositoryInitializeDownloadLoader(ctrl)
				chartRepo := &repo.ChartRepository{}
				repoClient.EXPECT().
					Initialize(gomock.Any(), gomock.Any()).
					Return(chartRepo, nil).
					Times(1)
				indexPath := "some_path.yaml"
				repoClient.EXPECT().
					DownloadIndex(chartRepo).
					Return(indexPath, nil).
					Times(1)
				repoClient.EXPECT().
					Load(indexPath).
					Return(&repo.IndexFile{}, nil).
					Times(1)

				client := helm.NewClient(
					log.NewNoop(),
					repoClient,
					nil,
				)
				err := client.Initialize("")
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Describe("ListCharts", func() {
			It("should return the chart entries", func() {
				client := helm.NewClient(log.NewNoop(), nil, nil)
				expectedCharts := map[string]repo.ChartVersions{
					"foo": make(repo.ChartVersions, 0),
					"bar": make(repo.ChartVersions, 0),
				}
				chartRepo := &repo.ChartRepository{
					Config:    &repo.Entry{URL: "https://repository"},
					IndexFile: &repo.IndexFile{Entries: expectedCharts},
				}
				client.SetChartRepo(chartRepo)
				charts := client.ListCharts()
				Expect(charts).To(Equal(expectedCharts))
			})
		})

		Describe("GetChart", func() {
			It("should fail when the chart doesn't exist", func() {
				client := helm.NewClient(log.NewNoop(), nil, nil)
				charts := map[string]repo.ChartVersions{"foo": make(repo.ChartVersions, 0)}
				chartRepo := &repo.ChartRepository{
					Config:    &repo.Entry{URL: "https://repository"},
					IndexFile: &repo.IndexFile{Entries: charts},
				}
				client.SetChartRepo(chartRepo)
				chart, err := client.GetChart("bar", "")
				Expect(err).To(Equal(fmt.Errorf("failed to get chart: chart not found: bar")))
				Expect(chart).To(BeNil())
			})

			It("should fail when the chart version doesn't exist", func() {
				client := helm.NewClient(log.NewNoop(), nil, nil)
				charts := map[string]repo.ChartVersions{"bar": make(repo.ChartVersions, 0)}
				chartRepo := &repo.ChartRepository{
					Config:    &repo.Entry{URL: "https://repository"},
					IndexFile: &repo.IndexFile{Entries: charts},
				}
				client.SetChartRepo(chartRepo)
				chart, err := client.GetChart("bar", "1.2.3")
				Expect(err).To(Equal(fmt.Errorf("failed to get chart: chart app version not found for \"bar\": 1.2.3")))
				Expect(chart).To(BeNil())
			})

			It("should succeed returning the requested chart", func() {
				client := helm.NewClient(log.NewNoop(), nil, nil)
				chartMetadata := &chart.Metadata{AppVersion: "1.2.3"}
				expectedChart := &repo.ChartVersion{Metadata: chartMetadata}
				versions := repo.ChartVersions{expectedChart}
				charts := map[string]repo.ChartVersions{"bar": versions}
				chartRepo := &repo.ChartRepository{
					Config:    &repo.Entry{URL: "https://repository"},
					IndexFile: &repo.IndexFile{Entries: charts},
				}
				client.SetChartRepo(chartRepo)
				chart, err := client.GetChart("bar", "1.2.3")
				Expect(err).NotTo(HaveOccurred())
				Expect(chart).To(Equal(expectedChart))
			})
		})

		Describe("ChartClient", func() {
			It("should return the expected chart client", func() {
				chartClient := helm.NewDefaultChartClient()
				client := helm.NewClient(nil, nil, chartClient)
				Expect(client.ChartClient()).To(Equal(chartClient))
			})
		})
	})
})
