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

package testutil

import (
	"helm.sh/helm/v3/pkg/action"
	"k8s.io/cli-runtime/pkg/genericclioptions"

	"github.com/kubernetes-sigs/minibroker/pkg/helm"
)

type ConfigProvider interface {
	ConfigProvider(namespace string) (*action.Configuration, error)
}

type ConfigInitializer interface {
	ConfigInitializer(
		getter genericclioptions.RESTClientGetter,
		namespace string,
		helmDriver string,
		log action.DebugLog,
	) error
}

type ConfigInitializerProvider interface {
	ConfigInitializerProvider() (*action.Configuration, helm.ConfigInitializer)
}
