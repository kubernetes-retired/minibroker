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

	"github.com/golang/glog"
	"github.com/pkg/errors"
	"k8s.io/helm/pkg/chartutil"
	"k8s.io/helm/pkg/getter"
	"k8s.io/helm/pkg/helm/environment"
	"k8s.io/helm/pkg/helm/helmpath"
	"k8s.io/helm/pkg/proto/hapi/chart"
	"k8s.io/helm/pkg/repo"
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
	c.home = helmpath.Home(environment.DefaultHelmHome)
	glog.Infof("Helm Home: %s", c.home)
	f, err := repo.LoadRepositoriesFile(c.home.RepositoryFile())
	if err != nil {
		return err
	}

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
	return err
}

func (c *Client) ListCharts() (map[string]repo.ChartVersions, error) {
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

	return charts, nil
}

func (c *Client) GetChart(name, version string) (*repo.ChartVersion, error) {
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
			return v, nil
		}
	}

	return nil, fmt.Errorf("version not found: %s @ %s", name, version)
}

func LoadChart(chartURL string) (*chart.Chart, error) {
	glog.Infof("downloading chart from %s", chartURL)
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
			glog.Errorln(
				errors.Wrapf(err, "failed to close file descriptor for chart at %s", fullChartPath))
		}
	}()

	glog.Infof("copying chart to %s", fullChartPath)
	if _, err := io.Copy(fd, resp.Body); err != nil {
		return nil, errors.Wrapf(err, "failed to copy chart contents to %s", fullChartPath)
	}

	glog.Infof("loading chart from %s on disk", fullChartPath)
	chart, err := chartutil.Load(fullChartPath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load chart from disk")
	}

	return chart, nil
}
