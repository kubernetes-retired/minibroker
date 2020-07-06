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
	"context"
	"fmt"
	"path"
	"runtime"
	"time"

	apiv1beta1 "github.com/kubernetes-sigs/service-catalog/pkg/apis/servicecatalog/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/kubernetes-sigs/minibroker/tests/integration/testutil"
)

const (
	brokerName = "minibroker"
)

var (
	kubeClient         kubernetes.Interface
	svcat              *testutil.Svcat
	testDir            string
	brokerReadyTimeout time.Duration
	provisionTimeout   time.Duration
	bindTimeout        time.Duration
	assertTimeout      time.Duration
)

func init() {
	brokerReadyTimeout = testutil.MustTimeoutFromEnv("TEST_BROKER_READY_TIMEOUT")
	provisionTimeout = testutil.MustTimeoutFromEnv("TEST_PROVISION_TIMEOUT")
	bindTimeout = testutil.MustTimeoutFromEnv("TEST_BIND_TIMEOUT")
	assertTimeout = testutil.MustTimeoutFromEnv("TEST_ASSERT_TIMEOUT")
}

var _ = BeforeSuite(func() {
	var err error

	kubeClient, err = testutil.KubeClient()
	Expect(err).NotTo(HaveOccurred())

	svcat, err = testutil.NewSvcat(kubeClient, namespace)
	Expect(err).NotTo(HaveOccurred())

	_, err = svcat.WaitForBroker(brokerName, namespace, brokerReadyTimeout)
	Expect(err).NotTo(HaveOccurred())

	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		panic("failed to fetch runtime information for setting up tests")
	}
	testDir = path.Dir(filename)
})

var _ = Describe("classes", func() {
	classes := []struct {
		name   string
		plan   string
		params map[string]interface{}
		assert func(*apiv1beta1.ServiceInstance, *apiv1beta1.ServiceBinding)
	}{
		{
			name: "mariadb",
			plan: "10-3-22",
			params: map[string]interface{}{
				"db": map[string]interface{}{
					"name": "mydb",
					"user": "admin",
				},
			},
			assert: func(instance *apiv1beta1.ServiceInstance, binding *apiv1beta1.ServiceBinding) {
				By("rendering and loading the mariadb client template")
				tmplPath := path.Join(testDir, "resources", "mariadb_client.tmpl.yaml")
				values := map[string]interface{}{
					"DatabaseVersion": "10.3",
					"SecretName":      binding.Spec.SecretName,
					"Command": []string{
						"sh", "-c",
						"mysql" +
							" --host=${DATABASE_HOST}" +
							" --port=${DATABASE_PORT}" +
							" --user=${DATABASE_USER}" +
							" --password=${DATABASE_PASSWORD}" +
							" --database=${DATABASE_NAME}" +
							" --execute='SELECT 1'",
					},
				}
				obj, err := testutil.LoadKubeSpec(tmplPath, values)
				Expect(err).NotTo(HaveOccurred())

				By("creating the mariadb client resource")
				ctx := context.Background()
				pod, err := kubeClient.CoreV1().Pods(namespace).Create(ctx, obj.(*corev1.Pod), metav1.CreateOptions{})
				Expect(err).NotTo(HaveOccurred())
				defer func() {
					err := kubeClient.CoreV1().Pods(namespace).Delete(ctx, pod.Name, metav1.DeleteOptions{})
					Expect(err).NotTo(HaveOccurred())
				}()

				By("asserting the mariadb client completed successfully")
				timeLimit := time.Now().Add(assertTimeout)
				for {
					if time.Now().After(timeLimit) {
						Fail("assertion timed out")
					}

					pod, err = kubeClient.CoreV1().Pods(namespace).Get(ctx, pod.Name, metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())

					if pod.Status.Phase == corev1.PodFailed {
						Fail("the client failed to assert the database service")
					}

					if pod.Status.Phase == corev1.PodSucceeded {
						break
					}

					time.Sleep(time.Second)
				}
			},
		},
		{
			name:   "mongodb",
			plan:   "4-2-4",
			params: map[string]interface{}{},
			assert: func(instance *apiv1beta1.ServiceInstance, binding *apiv1beta1.ServiceBinding) {
				By("rendering and loading the mongodb client template")
				tmplPath := path.Join(testDir, "resources", "mongodb_client.tmpl.yaml")
				values := map[string]interface{}{
					"DatabaseVersion": "4.2.7",
					"SecretName":      binding.Spec.SecretName,
					"Command": []string{
						"sh", "-c",
						"mongo" +
							" ${DATABASE_URL}",
						" connectionStatus",
					},
				}
				obj, err := testutil.LoadKubeSpec(tmplPath, values)
				Expect(err).NotTo(HaveOccurred())

				By("creating the mongodb client resource")
				ctx := context.Background()
				pod, err := kubeClient.CoreV1().Pods(namespace).Create(ctx, obj.(*corev1.Pod), metav1.CreateOptions{})
				Expect(err).NotTo(HaveOccurred())
				defer func() {
					err := kubeClient.CoreV1().Pods(namespace).Delete(ctx, pod.Name, metav1.DeleteOptions{})
					Expect(err).NotTo(HaveOccurred())
				}()

				By("asserting the mongodb client completed successfully")
				timeLimit := time.Now().Add(assertTimeout)
				for {
					if time.Now().After(timeLimit) {
						Fail("assertion timed out")
					}

					pod, err = kubeClient.CoreV1().Pods(namespace).Get(ctx, pod.Name, metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())

					if pod.Status.Phase == corev1.PodFailed {
						Fail("the client failed to assert the database service")
					}

					if pod.Status.Phase == corev1.PodSucceeded {
						break
					}

					time.Sleep(time.Second)
				}
			},
		},
		{
			name:   "mysql",
			plan:   "5-7-30",
			params: map[string]interface{}{},
			assert: func(instance *apiv1beta1.ServiceInstance, binding *apiv1beta1.ServiceBinding) {
				By("rendering and loading the mysql client template")
				tmplPath := path.Join(testDir, "resources", "mysql_client.tmpl.yaml")
				values := map[string]interface{}{
					"DatabaseVersion": "5.7.30",
					"SecretName":      binding.Spec.SecretName,
					"Command": []string{
						"sh", "-c",
						"mysql" +
							" --host=${DATABASE_HOST}" +
							" --port=${DATABASE_PORT}" +
							" --user=${DATABASE_USER}" +
							" --password=${DATABASE_PASSWORD}" +
							" --execute='SELECT 1'",
					},
				}
				obj, err := testutil.LoadKubeSpec(tmplPath, values)
				Expect(err).NotTo(HaveOccurred())

				By("creating the mysql client resource")
				ctx := context.Background()
				pod, err := kubeClient.CoreV1().Pods(namespace).Create(ctx, obj.(*corev1.Pod), metav1.CreateOptions{})
				Expect(err).NotTo(HaveOccurred())
				defer func() {
					err := kubeClient.CoreV1().Pods(namespace).Delete(ctx, pod.Name, metav1.DeleteOptions{})
					Expect(err).NotTo(HaveOccurred())
				}()

				By("asserting the mysql client completed successfully")
				timeLimit := time.Now().Add(assertTimeout)
				for {
					if time.Now().After(timeLimit) {
						Fail("assertion timed out")
					}

					pod, err = kubeClient.CoreV1().Pods(namespace).Get(ctx, pod.Name, metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())

					if pod.Status.Phase == corev1.PodFailed {
						Fail("the client failed to assert the database service")
					}

					if pod.Status.Phase == corev1.PodSucceeded {
						break
					}

					time.Sleep(time.Second)
				}
			},
		},
		{
			name:   "postgresql",
			plan:   "11-7-0",
			params: map[string]interface{}{},
			assert: func(instance *apiv1beta1.ServiceInstance, binding *apiv1beta1.ServiceBinding) {
				By("rendering and loading the postgresql client template")
				tmplPath := path.Join(testDir, "resources", "postgresql_client.tmpl.yaml")
				values := map[string]interface{}{
					"DatabaseVersion": "11.7",
					"SecretName":      binding.Spec.SecretName,
					"Command": []string{
						"psql", "-c", "SELECT 1",
					},
				}
				obj, err := testutil.LoadKubeSpec(tmplPath, values)
				Expect(err).NotTo(HaveOccurred())

				By("creating the postgresql client resource")
				ctx := context.Background()
				pod, err := kubeClient.CoreV1().Pods(namespace).Create(ctx, obj.(*corev1.Pod), metav1.CreateOptions{})
				Expect(err).NotTo(HaveOccurred())
				defer func() {
					err := kubeClient.CoreV1().Pods(namespace).Delete(ctx, pod.Name, metav1.DeleteOptions{})
					Expect(err).NotTo(HaveOccurred())
				}()

				By("asserting the postgresql client completed successfully")
				timeLimit := time.Now().Add(assertTimeout)
				for {
					if time.Now().After(timeLimit) {
						Fail("assertion timed out")
					}

					pod, err = kubeClient.CoreV1().Pods(namespace).Get(ctx, pod.Name, metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())

					if pod.Status.Phase == corev1.PodFailed {
						Fail("the client failed to assert the database service")
					}

					if pod.Status.Phase == corev1.PodSucceeded {
						break
					}

					time.Sleep(time.Second)
				}
			},
		},
		{
			name:   "redis",
			plan:   "5-0-7",
			params: map[string]interface{}{},
			assert: func(instance *apiv1beta1.ServiceInstance, binding *apiv1beta1.ServiceBinding) {
				By("rendering and loading the redis client template")
				tmplPath := path.Join(testDir, "resources", "redis_client.tmpl.yaml")
				values := map[string]interface{}{
					"DatabaseVersion": "5.0.7",
					"SecretName":      binding.Spec.SecretName,
					"Command": []string{
						"sh", "-c",
						"redis-cli" +
							" -u ${DATABASE_URL}",
						" CLUSTER INFO",
					},
				}
				obj, err := testutil.LoadKubeSpec(tmplPath, values)
				Expect(err).NotTo(HaveOccurred())

				By("creating the redis client resource")
				ctx := context.Background()
				pod, err := kubeClient.CoreV1().Pods(namespace).Create(ctx, obj.(*corev1.Pod), metav1.CreateOptions{})
				Expect(err).NotTo(HaveOccurred())
				defer func() {
					err := kubeClient.CoreV1().Pods(namespace).Delete(ctx, pod.Name, metav1.DeleteOptions{})
					Expect(err).NotTo(HaveOccurred())
				}()

				By("asserting the redis client completed successfully")
				timeLimit := time.Now().Add(assertTimeout)
				for {
					if time.Now().After(timeLimit) {
						Fail("assertion timed out")
					}

					pod, err = kubeClient.CoreV1().Pods(namespace).Get(ctx, pod.Name, metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())

					if pod.Status.Phase == corev1.PodFailed {
						Fail("the client failed to assert the database service")
					}

					if pod.Status.Phase == corev1.PodSucceeded {
						break
					}

					time.Sleep(time.Second)
				}
			},
		},
	}

	for _, class := range classes {
		class := class
		Describe(class.name, func() {
			serviceName := fmt.Sprintf("%s-%s-test", class.name, class.plan)
			It(fmt.Sprintf("should setup, assert and tear-down %s/%s", class.name, class.plan), func() {
				By(fmt.Sprintf("provisioning %s", serviceName))
				instance, err := svcat.Provision(namespace, serviceName, class.name, class.plan, class.params)
				Expect(err).NotTo(HaveOccurred())
				defer func() {
					By(fmt.Sprintf("deprovisioning %s", serviceName))
					err := svcat.Deprovision(instance)
					Expect(err).NotTo(HaveOccurred())
				}()

				By(fmt.Sprintf("waiting for %s to be provisioned", serviceName))
				err = svcat.WaitProvisioning(instance, provisionTimeout)
				Expect(err).NotTo(HaveOccurred())

				By(fmt.Sprintf("binding %s", serviceName))
				binding, err := svcat.Bind(instance)
				Expect(err).NotTo(HaveOccurred())
				defer func() {
					By(fmt.Sprintf("unbinding %s", serviceName))
					err := svcat.Unbind(instance)
					Expect(err).NotTo(HaveOccurred())
				}()

				By(fmt.Sprintf("waiting for %s binding", serviceName))
				err = svcat.WaitBinding(binding, bindTimeout)
				Expect(err).NotTo(HaveOccurred())

				By(fmt.Sprintf("asserting %s functionality", serviceName))
				class.assert(instance, binding)
			})
		})
	}
})
