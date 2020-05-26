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

package integration_test

import (
	"fmt"

	apiv1beta1 "github.com/kubernetes-sigs/service-catalog/pkg/apis/servicecatalog/v1beta1"
	"k8s.io/client-go/kubernetes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/kubernetes-sigs/minibroker/tests/integration/testutil"
)

const (
	brokerName = "minibroker"
)

var (
	kubeClient kubernetes.Interface
	sc         *testutil.Svcat
)

var _ = BeforeSuite(func() {
	var err error

	kubeClient, err = testutil.KubeClient()
	Expect(err).NotTo(HaveOccurred())

	sc, err = testutil.NewSvcat(kubeClient, namespace)
	Expect(err).NotTo(HaveOccurred())

	_, err = sc.WaitForBroker(brokerName, namespace)
	Expect(err).NotTo(HaveOccurred())
})

var _ = Describe("classes", func() {
	classes := []struct {
		name   string
		plan   string
		params map[string]interface{}
		assert func(*apiv1beta1.ServiceInstance, *apiv1beta1.ServiceBinding)
	}{
		{
			name:   "mariadb",
			plan:   "10-3-22",
			params: map[string]interface{}{},
			assert: func(instance *apiv1beta1.ServiceInstance, binding *apiv1beta1.ServiceBinding) {},
		},
		{
			name:   "mongodb",
			plan:   "4-2-4",
			params: map[string]interface{}{},
			assert: func(instance *apiv1beta1.ServiceInstance, binding *apiv1beta1.ServiceBinding) {},
		},
		{
			name:   "mysql",
			plan:   "5-7-30",
			params: map[string]interface{}{},
			assert: func(instance *apiv1beta1.ServiceInstance, binding *apiv1beta1.ServiceBinding) {},
		},
		{
			name:   "postgresql",
			plan:   "11-7-0",
			params: map[string]interface{}{},
			assert: func(instance *apiv1beta1.ServiceInstance, binding *apiv1beta1.ServiceBinding) {},
		},
		{
			name:   "redis",
			plan:   "5-0-7",
			params: map[string]interface{}{},
			assert: func(instance *apiv1beta1.ServiceInstance, binding *apiv1beta1.ServiceBinding) {},
		},
	}

	for _, class := range classes {
		class := class
		Describe(class.name, func() {
			serviceName := fmt.Sprintf("%s-%s-test", class.name, class.plan)
			It(fmt.Sprintf("should setup, assert and tear-down %s/%s", class.name, class.plan), func() {
				By(fmt.Sprintf("provisioning %s", serviceName))
				instance, err := sc.Provision(namespace, serviceName, class.name, class.plan, class.params)
				Expect(err).NotTo(HaveOccurred())
				defer func() {
					By(fmt.Sprintf("deprovisioning %s", serviceName))
					err := sc.Deprovision(instance)
					Expect(err).NotTo(HaveOccurred())
				}()

				By(fmt.Sprintf("waiting for %s to be provisioned", serviceName))
				err = sc.WaitProvisioning(instance)
				Expect(err).NotTo(HaveOccurred())

				By(fmt.Sprintf("binding %s", serviceName))
				binding, err := sc.Bind(instance)
				Expect(err).NotTo(HaveOccurred())
				defer func() {
					By(fmt.Sprintf("unbinding %s", serviceName))
					err := sc.Unbind(instance)
					Expect(err).NotTo(HaveOccurred())
				}()

				By(fmt.Sprintf("waiting for %s binding", serviceName))
				err = sc.WaitBinding(binding)
				Expect(err).NotTo(HaveOccurred())

				By(fmt.Sprintf("asserting %s functionality", serviceName))
				class.assert(instance, binding)
			})
		})
	}
})
