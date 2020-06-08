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

//go:generate mockgen -destination=./mocks/mock_config.go -package=mocks github.com/kubernetes-sigs/minibroker/pkg/helm ConfigProvider,ConfigInitializer,ConfigInitializerProvider

const (
	driver = "secret"

	// Empty values for these mean the internal defaults will be used.
	defaultKubeConfig = ""
	defaultContext    = ""
)

// ConfigProvider is the interface that wraps the Provide method for the upstream Helm configuration
// provider.
type ConfigProvider interface {
	Provide(namespace string) (interface{}, error)
}

// Config satisfies the ConfigProvider interface and provides Helm configurations for interacting
// with a specific Kubernetes namespace.
type Config struct {
	log                       log.Verboser
	configInitializerProvider ConfigInitializerProvider
	kubeConfig                string
	context                   string
}

// NewDefaultConfig creates a new Config with the default dependencies.
func NewDefaultConfig() *Config {
	return NewConfig(
		log.NewKlog(),
		NewDefaultConfigInitializerProvider(),
		defaultKubeConfig,
		defaultContext,
	)
}

// NewConfig creates a new Config with the explicit dependencies.
func NewConfig(
	log log.Verboser,
	configInitializerProvider ConfigInitializerProvider,
	kubeConfig string,
	context string,
) *Config {
	return &Config{
		log:                       log,
		configInitializerProvider: configInitializerProvider,
		kubeConfig:                kubeConfig,
		context:                   context,
	}
}

// Provide provides a new Helm configuration that enables the Helm client to deploy resources to a
// specific namespace.
func (c *Config) Provide(namespace string) (interface{}, error) {
	restGetter := kube.GetConfig(c.kubeConfig, c.context, namespace)
	debug := func(string, ...interface{}) {}
	if l := c.log.V(4).Get(); l != nil {
		debug = func(format string, v ...interface{}) {
			l.Log("helm client: %s", fmt.Sprintf(format, v...))
		}
	}
	actionConfig := c.configInitializerProvider.Provide()
	if err := actionConfig.Init(restGetter, namespace, driver, debug); err != nil {
		return nil, fmt.Errorf("failed to provide action configuration: %v", err)
	}
	return actionConfig, nil
}

// ConfigInitializer is the interface that wraps the signature of the action.Configuration.Init
// method to avoid a hidden dependency call in the Config.Provide method.
type ConfigInitializer interface {
	Init(
		getter genericclioptions.RESTClientGetter,
		namespace string,
		helmDriver string,
		log action.DebugLog,
	) error
}

// ConfigInitializerProvider is the interface that wraps the basic Provide method for configuration
// initializers.
type ConfigInitializerProvider interface {
	Provide() ConfigInitializer
}

// configInitializerProvider is a private default implementation that satisfies the
// ConfigInitializerProvider interface.
type configInitializerProvider struct{}

// NewDefaultConfigInitializerProvider creates a new ConfigInitializerProvider.
func NewDefaultConfigInitializerProvider() ConfigInitializerProvider {
	return &configInitializerProvider{}
}

// Provide provides a new *action.Configuration wrapped as a ConfigInitializer interface.
func (*configInitializerProvider) Provide() ConfigInitializer {
	return &action.Configuration{}
}
