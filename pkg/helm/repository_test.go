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
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/repo"

	"github.com/kubernetes-sigs/minibroker/pkg/helm"
	"github.com/kubernetes-sigs/minibroker/pkg/helm/mocks"
)

//go:generate mockgen -destination=./mocks/mock_repository.go -package=mocks github.com/kubernetes-sigs/minibroker/pkg/helm RepositoryInitializer,RepositoryDownloader,RepositoryLoader,RepositoryInitializeDownloadLoader,ChartRepo

var _ = Describe("Repository", func() {
	Context("RepositoryClient", func() {
		Describe("NewDefaultRepositoryClient", func() {
			It("should satisfy the RepositoryInitializeDownloadLoader interface", func() {
				var rc helm.RepositoryInitializeDownloadLoader = helm.NewDefaultRepositoryClient()
				Expect(rc).NotTo(BeNil())
			})
		})

		Describe("Initialize", func() {
			It("should fail when the internal newChartRepository fails", func() {
				cfg := &repo.Entry{Name: "foo"}
				providers := make(getter.Providers, 0)
				newChartRepository := func(arg0 *repo.Entry, arg1 getter.Providers) (*repo.ChartRepository, error) {
					Expect(arg0).To(Equal(cfg))
					Expect(arg1).To(Equal(providers))
					return nil, fmt.Errorf("some error")
				}
				rc := helm.NewRepositoryClient(newChartRepository, nil)
				chartRepo, err := rc.Initialize(cfg, providers)
				Expect(err).To(Equal(fmt.Errorf("failed to initialize repository \"foo\": some error")))
				Expect(chartRepo).To(BeNil())
			})

			It("should succeed when the internal newChartRepository succeeds", func() {
				cfg := &repo.Entry{}
				providers := make(getter.Providers, 0)
				expectedChartRepo := &repo.ChartRepository{}
				newChartRepository := func(arg0 *repo.Entry, arg1 getter.Providers) (*repo.ChartRepository, error) {
					Expect(arg0).To(Equal(cfg))
					Expect(arg1).To(Equal(providers))
					return expectedChartRepo, nil
				}
				rc := helm.NewRepositoryClient(newChartRepository, nil)
				chartRepo, err := rc.Initialize(cfg, providers)
				Expect(err).NotTo(HaveOccurred())
				Expect(chartRepo).To(Equal(expectedChartRepo))
			})
		})

		Describe("DownloadIndex", func() {
			var ctrl *gomock.Controller

			BeforeEach(func() {
				ctrl = gomock.NewController(GinkgoT())
			})

			AfterEach(func() {
				ctrl.Finish()
			})

			It("should fail when chartRepo.DownloadIndexFile fails", func() {
				expectedIndexPath := ""
				chartRepo := mocks.NewMockChartRepo(ctrl)
				chartRepo.EXPECT().
					DownloadIndexFile().
					Return("", fmt.Errorf("some error")).
					Times(1)
				rc := helm.NewRepositoryClient(nil, nil)
				indexPath, err := rc.DownloadIndex(chartRepo)
				Expect(err).To(Equal(fmt.Errorf("failed to download repository index: some error")))
				Expect(indexPath).To(Equal(expectedIndexPath))
			})

			It("should succeed when chartRepo.DownloadIndexFile succeeds", func() {
				expectedIndexPath := "some_index.yaml"
				chartRepo := mocks.NewMockChartRepo(ctrl)
				chartRepo.EXPECT().
					DownloadIndexFile().
					Return(expectedIndexPath, nil).
					Times(1)
				rc := helm.NewRepositoryClient(nil, nil)
				indexPath, err := rc.DownloadIndex(chartRepo)
				Expect(err).NotTo(HaveOccurred())
				Expect(indexPath).To(Equal(expectedIndexPath))
			})
		})

		Describe("Load", func() {
			It("should fail when the internal loadIndexFile fails", func() {
				path := "some_index.yaml"
				loadIndexFile := func(arg0 string) (*repo.IndexFile, error) {
					Expect(arg0).To(Equal(path))
					return nil, fmt.Errorf("some error")
				}
				rc := helm.NewRepositoryClient(nil, loadIndexFile)
				indexFile, err := rc.Load(path)
				Expect(err).To(Equal(fmt.Errorf("failed to load repository index \"some_index.yaml\": some error")))
				Expect(indexFile).To(BeNil())
			})

			It("should succeed when the internal loadIndexFile succeeds", func() {
				path := "some_index.yaml"
				expectedIndexFile := &repo.IndexFile{}
				loadIndexFile := func(arg0 string) (*repo.IndexFile, error) {
					Expect(arg0).To(Equal(path))
					return expectedIndexFile, nil
				}
				rc := helm.NewRepositoryClient(nil, loadIndexFile)
				indexFile, err := rc.Load(path)
				Expect(err).NotTo(HaveOccurred())
				Expect(indexFile).To(Equal(expectedIndexFile))
			})
		})
	})
})
