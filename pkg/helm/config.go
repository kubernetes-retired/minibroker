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

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/kube"
	"k8s.io/cli-runtime/pkg/genericclioptions"

	"github.com/kubernetes-sigs/minibroker/pkg/log"
)

const (
	driver = "secret"

	// Empty values for these mean the internal defaults will be used.
	defaultKubeConfig = ""
	defaultContext    = ""
)

// ConfigProvider defines the signature for a function that provides *action.Configuration.
type ConfigProvider func(namespace string) (*action.Configuration, error)

// NewDefaultConfigProvider creates a new ConfigProvider using the default dependencies.
func NewDefaultConfigProvider() ConfigProvider {
	return NewConfigProvider(
		log.NewKlog(),
		DefaultConfigInitializerProvider,
		defaultKubeConfig,
		defaultContext,
	)
}

// NewConfigProvider creates a new ConfigProvider using the explicit dependencies.
func NewConfigProvider(
	log log.Verboser,
	configInitializerProvider ConfigInitializerProvider,
	kubeConfig string,
	context string,
) ConfigProvider {
	return func(namespace string) (*action.Configuration, error) {
		restGetter := kube.GetConfig(kubeConfig, context, namespace)
		debug := func(format string, v ...interface{}) {
			log.V(4).Log("helm client: "+format, v...)
		}
		cfg, init := configInitializerProvider()
		if err := init(restGetter, namespace, driver, debug); err != nil {
			return nil, fmt.Errorf("failed to provide action configuration: %v", err)
		}
		return cfg, nil
	}
}

// ConfigInitializer is the interface that wraps the signature of the action.Configuration.Init
// method to avoid a hidden dependency call in the Config.Provide method.
type ConfigInitializer func(
	getter genericclioptions.RESTClientGetter,
	namespace string,
	helmDriver string,
	log action.DebugLog,
) error

// ConfigInitializerProvider defines the signature for a function that provides an
// *action.Configuration and its Init method.
type ConfigInitializerProvider func() (*action.Configuration, ConfigInitializer)

// DefaultConfigInitializerProvider is the default implementation for ConfigInitializerProvider.
func DefaultConfigInitializerProvider() (*action.Configuration, ConfigInitializer) {
	cfg := &action.Configuration{}
	return cfg, cfg.Init
}
