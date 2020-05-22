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
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"k8s.io/helm/pkg/chartutil"
	"k8s.io/helm/pkg/getter"
	"k8s.io/helm/pkg/helm/environment"
	"k8s.io/helm/pkg/helm/helmpath"
	"k8s.io/helm/pkg/proto/hapi/chart"
	"k8s.io/helm/pkg/repo"
	"k8s.io/klog"
)

const stableURL = "https://kubernetes-charts.storage.googleapis.com"

type Client struct {
	repoURL string
	home    helmpath.Home
	rf      *repo.RepoFile
}

func NewClient(repoURL string) *Client {
	if repoURL == "" {
		repoURL = stableURL
	}

	return &Client{repoURL: repoURL}
}

func (c *Client) Init() error {
	klog.V(5).Infof("helm client: initializing client")

	c.home = helmpath.Home(environment.DefaultHelmHome)
	klog.V(3).Infof("helm client: helm home: %s", c.home)
	f, err := repo.LoadRepositoriesFile(c.home.RepositoryFile())
	if err != nil {
		return err
	}

	klog.V(3).Infof("helm client: caching stable repository")
	cif := c.home.CacheIndex("stable")
	cr := repo.Entry{
		Name:  "stable",
		Cache: cif,
		URL:   c.repoURL,
	}

	var settings environment.EnvSettings
	r, err := repo.NewChartRepository(&cr, getter.All(settings))
	if err != nil {
		return err
	}

	if err := r.DownloadIndexFile(c.home.Cache()); err != nil {
		return errors.Wrapf(err, "Looks like %q is not a valid chart repository or cannot be reached", cr.URL)
	}

	f.Update(&cr)
	f.WriteFile(c.home.RepositoryFile(), 0644)

	// Load the repositories.yaml
	c.rf, err = repo.LoadRepositoriesFile(c.home.RepositoryFile())

	klog.V(5).Infof("helm client: initialized client")

	return err
}

func (c *Client) ListCharts() (map[string]repo.ChartVersions, error) {
	klog.V(4).Infof("helm client: listing charts")

	charts := map[string]repo.ChartVersions{}

	// TODO: handle non-unique names across repos
	for _, r := range c.rf.Repositories {
		n := r.Name
		f := c.home.CacheIndex(n)
		index, err := repo.LoadIndexFile(f)
		if err != nil {
			return nil, errors.Wrapf(err, "Could not load helm repository index at %s", f)
		}

		for chart, chartVersions := range index.Entries {
			charts[chart] = chartVersions
		}
	}

	klog.V(4).Infof("helm client: listed charts")

	return charts, nil
}

func (c *Client) GetChart(name, version string) (*repo.ChartVersion, error) {
	klog.V(4).Infof("helm client: getting chart %s:%s", name, version)

	charts, err := c.ListCharts()
	if err != nil {
		return nil, err
	}

	versions, ok := charts[name]
	if !ok {
		return nil, fmt.Errorf("chart not found: %s", name)
	}

	for _, v := range versions {
		if v.AppVersion == version {
			klog.V(4).Infof("helm client: got chart %s:%s", name, version)
			return v, nil
		}
	}

	return nil, fmt.Errorf("version not found: %s @ %s", name, version)
}

func LoadChart(chartURL string) (*chart.Chart, error) {
	klog.V(3).Infof("helm: loading chart %q", chartURL)

	resp, err := http.Get(chartURL)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to download chart from %s", chartURL)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err = errors.Errorf("got status code %d trying to download chart at %s", resp.StatusCode, chartURL)
		return nil, err
	}
	tmpDir, err := ioutil.TempDir("", "helm")
	if err != nil {
		return nil, errors.Wrap(err, "failed to create temp helm chart directory")
	}

	fullChartPath := filepath.Join(tmpDir, "chart")
	fd, err := os.Create(fullChartPath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create temp chart file")
	}
	defer func() {
		if err := fd.Close(); err != nil {
			klog.V(2).Infof("helm: failed to close file descriptor for chart at %q: %v", fullChartPath, err)
		}
	}()

	klog.V(3).Infof("helm: downloading chart %q to %q", chartURL, fullChartPath)
	if _, err := io.Copy(fd, resp.Body); err != nil {
		return nil, errors.Wrapf(err, "failed to download chart contents to %s", fullChartPath)
	}

	klog.V(3).Infof("helm: loading chart from disk %q", fullChartPath)
	chart, err := chartutil.Load(fullChartPath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load chart from disk")
	}

	klog.V(3).Infof("helm: successfully loaded chart downloaded from %q", chartURL)

	return chart, nil
}
