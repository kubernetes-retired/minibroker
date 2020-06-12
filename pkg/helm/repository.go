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

package helm

import (
	"fmt"

	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/repo"
)

//go:generate mockgen -destination=./mocks/mock_repository.go -package=mocks github.com/kubernetes-sigs/minibroker/pkg/helm RepositoryInitializer,RepositoryDownloader,RepositoryLoader,RepositoryInitializeDownloadLoader,ChartRepo

// RepositoryInitializer is the interface that wraps the Initialize method for initializing a
// repo.ChartRepository.
type RepositoryInitializer interface {
	Initialize(*repo.Entry, getter.Providers) (*repo.ChartRepository, error)
}

// RepositoryDownloader is the interface that wraps the DownloadIndex method for downloading a
// ChartRepo.
type RepositoryDownloader interface {
	DownloadIndex(ChartRepo) (string, error)
}

// RepositoryLoader is the interface that wraps the chart repository Load method.
type RepositoryLoader interface {
	Load(path string) (*repo.IndexFile, error)
}

// RepositoryInitializeDownloadLoader wraps all the repository interfaces.
type RepositoryInitializeDownloadLoader interface {
	RepositoryInitializer
	RepositoryDownloader
	RepositoryLoader
}

// ChartRepo is the interface that wraps the DownloadIndexFile method. It exists to be mocked on the
// Downloader.Download.
type ChartRepo interface {
	DownloadIndexFile() (string, error)
}

// RepositoryClient satisfies the RepositoryInitializeDownloadLoader interface.
type RepositoryClient struct {
	newChartRepository func(*repo.Entry, getter.Providers) (*repo.ChartRepository, error)
	loadIndexFile      func(string) (*repo.IndexFile, error)
}

// NewDefaultRepositoryClient creates a new RepositoryClient with the default dependencies.
func NewDefaultRepositoryClient() *RepositoryClient {
	return NewRepositoryClient(repo.NewChartRepository, repo.LoadIndexFile)
}

// NewRepositoryClient creates a new RepositoryClient with the explicit dependencies.
func NewRepositoryClient(
	newChartRepository func(*repo.Entry, getter.Providers) (*repo.ChartRepository, error),
	loadIndexFile func(string) (*repo.IndexFile, error),
) *RepositoryClient {
	return &RepositoryClient{
		newChartRepository: newChartRepository,
		loadIndexFile:      loadIndexFile,
	}
}

// Initialize initializes a repo.ChartRepository.
func (rc *RepositoryClient) Initialize(
	cfg *repo.Entry,
	providers getter.Providers,
) (*repo.ChartRepository, error) {
	chartRepo, err := rc.newChartRepository(cfg, providers)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize repository %q: %v", cfg.Name, err)
	}
	return chartRepo, nil
}

// DownloadIndex downloads a chart repository index and returns its path.
func (*RepositoryClient) DownloadIndex(chartRepo ChartRepo) (string, error) {
	indexPath, err := chartRepo.DownloadIndexFile()
	if err != nil {
		return "", fmt.Errorf("failed to download repository index: %v", err)
	}
	return indexPath, nil
}

// Load loads a chart repository index file using the path on disk.
func (rc *RepositoryClient) Load(path string) (*repo.IndexFile, error) {
	index, err := rc.loadIndexFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to load repository index %q: %v", path, err)
	}
	return index, nil
}
