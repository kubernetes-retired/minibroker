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
	"fmt"
	"os"
	"path/filepath"
	"time"

	servicecatalogv1beta1 "github.com/kubernetes-sigs/service-catalog/pkg/apis/servicecatalog/v1beta1"
	svcatclient "github.com/kubernetes-sigs/service-catalog/pkg/client/clientset_generated/clientset"
	"github.com/kubernetes-sigs/service-catalog/pkg/svcat"
	servicecatalog "github.com/kubernetes-sigs/service-catalog/pkg/svcat/service-catalog"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	interval = time.Second
	timeout  = time.Minute * 3
)

// KubeClient creates a new Kubernetes client using the default kubeconfig.
func KubeClient() (kubernetes.Interface, error) {
	config, err := kubeConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize kubernetes client: %v", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize kubernetes client: %v", err)
	}

	return clientset, nil
}

// svcatClient creates a new svcat client using the default kubeconfig.
func svcatClient() (*svcatclient.Clientset, error) {
	config, err := kubeConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize svcat client: %v", err)
	}

	clientset, err := svcatclient.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize svcat client: %v", err)
	}

	return clientset, nil
}

func kubeConfig() (*rest.Config, error) {
	location := os.Getenv("KUBECONFIG")
	if location == "" {
		location = filepath.Join(os.Getenv("HOME"), ".kube", "config")
	}

	config, err := clientcmd.BuildConfigFromFlags("", location)
	if err != nil {
		config, err = rest.InClusterConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to get kubernetes configuration: %v", err)
		}
	}

	return config, nil
}

// Svcat wraps the svcat functionality for easier use with the integration tests.
type Svcat struct {
	kubeClient kubernetes.Interface
	client     *svcatclient.Clientset
	app        *svcat.App
}

// NewSvcat constructs a new Svcat.
func NewSvcat(kubeClient kubernetes.Interface, namespace string) (*Svcat, error) {
	client, err := svcatClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create a new Svcat: %v", err)
	}
	app, err := svcat.NewApp(kubeClient, client, namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to create a new Svcat: %v", err)
	}
	sc := &Svcat{
		kubeClient: kubeClient,
		client:     client,
		app:        app,
	}
	return sc, nil
}

// WaitForBroker waits for the broker to be ready.
func (sc *Svcat) WaitForBroker(
	name string,
	namespace string,
) (servicecatalog.Broker, error) {
	opts := &servicecatalog.ScopeOptions{
		Scope:     servicecatalog.AllScope,
		Namespace: namespace,
	}
	broker, err := sc.app.WaitForBroker(name, opts, interval, &timeout)
	if err != nil {
		return nil, fmt.Errorf("failed to wait for broker: %v", err)
	}

	return broker, nil
}

// Provision provisions an instance.
func (sc *Svcat) Provision(
	namespace string,
	serviceName string,
	className string,
	planName string,
	params map[string]interface{},
) (*servicecatalogv1beta1.ServiceInstance, error) {
	scopeOpts := servicecatalog.ScopeOptions{
		Namespace: namespace,
		Scope:     servicecatalog.AllScope,
	}

	class, err := sc.app.RetrieveClassByID(className, scopeOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to provision instance: %v", err)
	}

	if class.IsClusterServiceClass() {
		scopeOpts.Scope = servicecatalog.ClusterScope
	} else {
		scopeOpts.Scope = servicecatalog.NamespaceScope
	}
	plan, err := sc.app.RetrievePlanByClassIDAndName(class.GetName(), planName, scopeOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to provision instance: %v", err)
	}

	provisionOpts := &servicecatalog.ProvisionOptions{
		Namespace: namespace,
		Params:    params,
	}
	instance, err := sc.app.Provision(serviceName, class.GetName(), plan.GetName(), class.IsClusterServiceClass(), provisionOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to provision instance: %v", err)
	}

	return instance, nil
}

// Deprovision deprovisions an instance.
func (sc *Svcat) Deprovision(instance *servicecatalogv1beta1.ServiceInstance) error {
	if err := sc.app.Deprovision(instance.Namespace, instance.Name); err != nil {
		return fmt.Errorf("failed to deprovision instance: %v", err)
	}
	return nil
}

// WaitProvisioning waits for an instance to be provisioned.
func (sc *Svcat) WaitProvisioning(instance *servicecatalogv1beta1.ServiceInstance) error {
	if _, err := sc.app.WaitForInstance(instance.Namespace, instance.Name, interval, &timeout); err != nil {
		return fmt.Errorf("failed to wait for instance to be provisioned: %v", err)
	}

	return nil
}

// Bind binds an instance.
func (sc *Svcat) Bind(instance *servicecatalogv1beta1.ServiceInstance) (*servicecatalogv1beta1.ServiceBinding, error) {
	binding, err := sc.app.Bind(instance.Namespace, "", "", instance.Name, "", nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to bind instance: %v", err)
	}

	return binding, nil
}

// Unbind unbinds an instance.
func (sc *Svcat) Unbind(instance *servicecatalogv1beta1.ServiceInstance) error {
	if _, err := sc.app.Unbind(instance.Namespace, instance.Name); err != nil {
		return fmt.Errorf("failed to unbind instance: %v", err)
	}
	return nil
}

// WaitBinding waits for a service binding to be ready.
func (sc *Svcat) WaitBinding(binding *servicecatalogv1beta1.ServiceBinding) error {
	if _, err := sc.app.WaitForBinding(binding.Namespace, binding.Name, interval, &timeout); err != nil {
		return fmt.Errorf("failed to wait for service binding: %v", err)
	}

	return nil
}
