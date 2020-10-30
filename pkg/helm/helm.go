/*
Copyright 2019 The Kubernetes Authors.

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

	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/helmpath"
	"helm.sh/helm/v3/pkg/repo"

	"github.com/kubernetes-sigs/minibroker/pkg/log"
)

const (
	stableURL = "https://charts.helm.sh/stable"
	// As old versions of Kubernetes had a limit on names of 63 characters, Helm uses 53, reserving
	// 10 characters for charts to add data.
	helmMaxNameLength = 53
)

// Client represents a Helm client to interact with the k8s cluster.
type Client struct {
	log              log.Verboser
	repositoryClient RepositoryInitializeDownloadLoader
	chartClient      *ChartClient

	settings  *cli.EnvSettings
	chartRepo *repo.ChartRepository
}

// NewDefaultClient creates a new Client with the default dependencies.
func NewDefaultClient() *Client {
	return NewClient(
		log.NewKlog(),
		NewDefaultRepositoryClient(),
		NewDefaultChartClient(),
	)
}

// NewClient creates a new client with explicit dependencies.
func NewClient(
	log log.Verboser,
	repositoryClient RepositoryInitializeDownloadLoader,
	chartClient *ChartClient,
) *Client {
	settings := &cli.EnvSettings{
		RegistryConfig:   helmpath.ConfigPath("registry.json"),
		RepositoryConfig: helmpath.ConfigPath("repositories.yaml"),
		RepositoryCache:  helmpath.CachePath("repository"),
	}
	return &Client{
		log:              log,
		repositoryClient: repositoryClient,
		chartClient:      chartClient,
		settings:         settings,
	}
}

// Initialize initializes a chart repository.
// TODO(f0rmiga): be able to handle multiple repositories. How can we handle charts with the same
// name across repositories?
// TODO(f0rmiga): add a readiness probe for this initialization process. A health endpoint would be
// enough.
func (c *Client) Initialize(repoURL string) error {
	c.log.V(3).Log("helm client: initializing")

	// TODO(f0rmiga): Allow private repos with authentication. Entry will need to contain the auth
	// configuration.
	chartCfg := repo.Entry{
		Name: "stable",
		URL:  repoURL,
	}
	if chartCfg.URL == "" {
		chartCfg.URL = stableURL
	}
	chartRepo, err := c.repositoryClient.Initialize(&chartCfg, getter.All(c.settings))
	if err != nil {
		return fmt.Errorf("failed to initialize helm client: %v", err)
	}

	c.log.V(3).Log("helm client: downloading index file")
	indexPath, err := c.repositoryClient.DownloadIndex(chartRepo)
	if err != nil {
		return fmt.Errorf("failed to initialize helm client: %v", err)
	}

	c.log.V(3).Log("helm client: loading repository")
	indexFile, err := c.repositoryClient.Load(indexPath)
	if err != nil {
		return fmt.Errorf("failed to initialize helm client: %v", err)
	}

	chartRepo.IndexFile = indexFile
	c.chartRepo = chartRepo

	c.log.V(3).Log("helm client: successfully initialized")

	return nil
}

// ListCharts lists the charts from the chart repository.
func (c *Client) ListCharts() map[string]repo.ChartVersions {
	c.log.V(4).Log("helm client: listing charts from %s", c.chartRepo.Config.URL)
	defer c.log.V(4).Log("helm client: listed charts from %s", c.chartRepo.Config.URL)
	return c.chartRepo.IndexFile.Entries
}

// GetChart gets a chart that exists in the chart repository. IndexFile.Get() cannot be used here
// since we filter by app version.
func (c *Client) GetChart(name, appVersion string) (*repo.ChartVersion, error) {
	c.log.V(4).Log("helm client: getting chart %s:%s", name, appVersion)

	charts := c.ListCharts()

	versions, ok := charts[name]
	if !ok {
		err := fmt.Errorf("chart not found: %s", name)
		c.log.V(4).Log("helm client: %v", err)
		return nil, fmt.Errorf("failed to get chart: %v", err)
	}

	for _, v := range versions {
		if v.AppVersion == appVersion {
			c.log.V(4).Log("helm client: got chart %s:%s", name, appVersion)
			return v, nil
		}
	}

	err := fmt.Errorf("chart app version not found for %q: %s", name, appVersion)
	c.log.V(4).Log("helm client: %v", err)
	return nil, fmt.Errorf("failed to get chart: %v", err)
}

// ChartClient returns the chart client for installing and uninstalling a chart.
func (c *Client) ChartClient() *ChartClient {
	return c.chartClient
}
