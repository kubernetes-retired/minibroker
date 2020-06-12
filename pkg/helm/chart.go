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
	"io"
	"net/http"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/repo"

	"github.com/kubernetes-sigs/minibroker/pkg/log"
	"github.com/kubernetes-sigs/minibroker/pkg/nameutil"
)

//go:generate mockgen -destination=./mocks/mock_chart.go -package=mocks github.com/kubernetes-sigs/minibroker/pkg/helm ChartLoader,ChartHelmClientProvider
//go:generate mockgen -destination=./mocks/mock_io.go -package=mocks io ReadCloser

// ChartInstaller is the interface that wraps the Install method.
type ChartInstaller interface {
	Install(
		chartDef *repo.ChartVersion,
		namespace string,
		values map[string]interface{},
	) (*release.Release, error)
}

// ChartUninstaller is the interface that wraps the Uninstall method.
type ChartUninstaller interface {
	Uninstall(releaseName, namespace string) error
}

// ChartInstallUninstaller wraps the ChartInstaller and ChartUninstaller interfaces.
type ChartInstallUninstaller interface {
	ChartInstaller
	ChartUninstaller
}

// ChartClient satisfies the ChartInstallUninstaller interface. It allows users of this client to
// install and uninstall charts.
type ChartClient struct {
	log                     log.Verboser
	chartLoader             ChartLoader
	nameGenerator           nameutil.Generator
	ChartHelmClientProvider ChartHelmClientProvider
}

// NewDefaultChartClient creates a new ChartClient with the default dependencies.
func NewDefaultChartClient() *ChartClient {
	return NewChartClient(
		log.NewKlog(),
		NewDefaultChartManager(),
		nameutil.NewDefaultNameGenerator(),
		NewDefaultChartHelm(),
	)
}

// NewChartClient creates a new ChartClient with the explicit dependencies.
func NewChartClient(
	log log.Verboser,
	chartLoader ChartLoader,
	nameGenerator nameutil.Generator,
	ChartHelmClientProvider ChartHelmClientProvider,
) *ChartClient {
	return &ChartClient{
		log:                     log,
		chartLoader:             chartLoader,
		nameGenerator:           nameGenerator,
		ChartHelmClientProvider: ChartHelmClientProvider,
	}
}

// Install installs a chart version into a specific namespace using the provided values.
func (cc *ChartClient) Install(
	chartDef *repo.ChartVersion,
	namespace string,
	values map[string]interface{},
) (*release.Release, error) {
	if len(chartDef.URLs) == 0 {
		err := fmt.Errorf("missing chart URL for %q", chartDef.Name)
		return nil, fmt.Errorf("failed to install chart: %v", err)
	}
	chartURL := chartDef.URLs[0]

	chartRequested, err := cc.chartLoader.Load(chartURL)
	if err != nil {
		return nil, fmt.Errorf("failed to install chart: %v", err)
	}

	if chartRequested.Metadata.Deprecated {
		if l := cc.log.V(3).Get(); l != nil {
			l.Log("minibroker: WARNING: the chart %s:%s is deprecated", chartDef.Name, chartDef.Version)
		}
	}

	// TODO(f0rmiga): ensure chart dependencies are fetched.

	releaseName, err := cc.nameGenerator.Generate(fmt.Sprintf("%s-", chartDef.Name))
	if err != nil {
		return nil, fmt.Errorf("failed to install chart: %v", err)
	}

	if len(releaseName) > helmMaxNameLength {
		err := fmt.Errorf(
			"invalid release name %q: names cannot exceed %d characters",
			releaseName,
			helmMaxNameLength)
		return nil, fmt.Errorf("failed to install chart: %v", err)
	}

	installer, err := cc.ChartHelmClientProvider.ProvideInstaller(releaseName, namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to install chart: %v", err)
	}

	rls, err := installer(chartRequested, values)
	if err != nil {
		return nil, fmt.Errorf("failed to install chart: %v", err)
	}

	return rls, nil
}

// Uninstall uninstalls a release from a namespace.
func (cc *ChartClient) Uninstall(releaseName, namespace string) error {
	uninstaller, err := cc.ChartHelmClientProvider.ProvideUninstaller(namespace)
	if err != nil {
		return fmt.Errorf("failed to uninstall chart: %v", err)
	}

	if _, err := uninstaller(releaseName); err != nil {
		return fmt.Errorf("failed to uninstall chart: %v", err)
	}

	return nil
}

// ChartLoader is the interface that wraps the Load method.
type ChartLoader interface {
	Load(chartURL string) (*chart.Chart, error)
}

// ChartManager satisfies the ChartLoader interface.
type ChartManager struct {
	httpGetter       HTTPGetter
	loadChartArchive func(io.Reader) (*chart.Chart, error)
}

// NewDefaultChartManager creates a new ChartManager with the default dependencies.
func NewDefaultChartManager() *ChartManager {
	return NewChartManager(
		http.DefaultClient,
		loader.LoadArchive,
	)
}

// NewChartManager creates a new ChartManager with the explicit dependencies.
func NewChartManager(
	httpGetter HTTPGetter,
	loadChartArchive func(io.Reader) (*chart.Chart, error),
) *ChartManager {
	return &ChartManager{
		httpGetter:       httpGetter,
		loadChartArchive: loadChartArchive,
	}
}

// Load loads a chart from a URL.
// TODO(f0rmiga): implement caching for chart archives.
func (cm *ChartManager) Load(chartURL string) (*chart.Chart, error) {
	chartResp, err := cm.httpGetter.Get(chartURL)
	if err != nil {
		return nil, fmt.Errorf("failed to load chart: %v", err)
	}
	defer chartResp.Body.Close()

	chartRequested, err := cm.loadChartArchive(chartResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to load chart: %v", err)
	}

	return chartRequested, nil
}

// ChartHelmClientProvider is the interface that wraps the methods for providing Helm action clients
// for installing and uninstalling charts.
type ChartHelmClientProvider interface {
	ProvideInstaller(releaseName, namespace string) (ChartInstallRunner, error)
	ProvideUninstaller(namespace string) (ChartUninstallRunner, error)
}

// ChartHelm satisfies the ChartHelmClientProvider interface.
type ChartHelm struct {
	configProvider     ConfigProvider
	actionNewInstall   func(*action.Configuration) *action.Install
	actionNewUninstall func(*action.Configuration) *action.Uninstall
}

// NewDefaultChartHelm creates a new ChartHelm with the default dependencies.
func NewDefaultChartHelm() *ChartHelm {
	return NewChartHelm(
		NewDefaultConfigProvider(),
		action.NewInstall,
		action.NewUninstall,
	)
}

// NewChartHelm creates a new ChartHelm with the explicit dependencies.
func NewChartHelm(
	configProvider ConfigProvider,
	actionNewInstall func(*action.Configuration) *action.Install,
	actionNewUninstall func(*action.Configuration) *action.Uninstall,
) *ChartHelm {
	return &ChartHelm{
		configProvider:     configProvider,
		actionNewInstall:   actionNewInstall,
		actionNewUninstall: actionNewUninstall,
	}
}

// ProvideInstaller provides a Helm action client for installing charts.
func (ch *ChartHelm) ProvideInstaller(
	releaseName string,
	namespace string,
) (ChartInstallRunner, error) {
	cfg, err := ch.configProvider(namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to provide chart installer: %v", err)
	}
	client := ch.actionNewInstall(cfg)
	client.ReleaseName = releaseName
	client.Namespace = namespace
	client.Wait = true
	return client.Run, nil
}

// ProvideUninstaller provides a Helm action client for uninstalling charts.
func (ch *ChartHelm) ProvideUninstaller(namespace string) (ChartUninstallRunner, error) {
	cfg, err := ch.configProvider(namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to provide chart uninstaller: %v", err)
	}
	client := ch.actionNewUninstall(cfg)
	return client.Run, nil
}

// ChartInstallRunner defines the signature for a function that installs a chart.
type ChartInstallRunner func(*chart.Chart, map[string]interface{}) (*release.Release, error)

// ChartUninstallRunner defines the signature for a function that uninstalls a chart.
type ChartUninstallRunner func(string) (*release.UninstallReleaseResponse, error)
