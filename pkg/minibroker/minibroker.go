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

package minibroker

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"regexp"
	"strings"

	"github.com/Masterminds/semver"
	"github.com/davecgh/go-spew/spew"
	"github.com/golang/glog"
	"github.com/kubernetes-sigs/minibroker/pkg/helm"
	"github.com/kubernetes-sigs/minibroker/pkg/tiller"
	"github.com/pkg/errors"
	osb "github.com/pmorie/go-open-service-broker-client/v2"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/helm/pkg/repo"
)

const (
	InstanceLabel       = "minibroker.instance"
	ServiceKey          = "service-id"
	PlanKey             = "plan-id"
	ProvisionParamsKey  = "provision-params"
	ReleaseNamespaceKey = "release-namespace"
	HeritageLabel       = "heritage"
	ReleaseLabel        = "release"
	TillerHeritage      = "Tiller"
)

// ConfigMap keys for tracking the last operation
const (
	OperationNameKey        = "last-operation-name"
	OperationStateKey       = "last-operation-state"
	OperationDescriptionKey = "last-operation-description"
)

// Error code constants missing from go-open-service-broker-client
// See https://github.com/pmorie/go-open-service-broker-client/pull/136
const (
	ConcurrencyErrorMessage     = "ConcurrencyError"
	ConcurrencyErrorDescription = "Concurrent modification not supported"
)

// Last operation name prefixes for various operations
const (
	OperationPrefixProvision   = "provision-"
	OperationPrefixDeprovision = "deprovision-"
	OperationPrefixBind        = "bind-"
)

const (
	BindingKeyPrefix      = "binding-"
	BindingStateKeyPrefix = "binding-state-"
)

type Client struct {
	helm                      *helm.Client
	namespace                 string
	coreClient                kubernetes.Interface
	providers                 map[string]Provider
	serviceCatalogEnabledOnly bool
}

func NewClient(repoURL string, serviceCatalogEnabledOnly bool) *Client {
	return &Client{
		helm:                      helm.NewClient(repoURL),
		coreClient:                loadInClusterClient(),
		namespace:                 loadNamespace(),
		serviceCatalogEnabledOnly: serviceCatalogEnabledOnly,
		providers: map[string]Provider{
			"mysql":      MySQLProvider{},
			"mariadb":    MariadbProvider{},
			"postgresql": PostgresProvider{},
			"mongodb":    MongodbProvider{},
			"redis":      RedisProvider{},
		},
	}
}

func loadInClusterClient() kubernetes.Interface {
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	return clientset
}

func loadNamespace() string {
	if data, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace"); err == nil {
		if ns := strings.TrimSpace(string(data)); len(ns) > 0 {
			glog.Infof("namespace: %s", ns)
			return ns
		}
	}

	panic("could not detect current namespace")
}

func (c *Client) Init() error {
	return c.helm.Init()
}

func hasTag(tag string, list []string) bool {
	for _, listTag := range list {
		if listTag == tag {
			return true
		}
	}

	return false
}

func getTagIntersection(chartVersions repo.ChartVersions) []string {
	tagList := make([][]string, 0)

	for _, chartVersion := range chartVersions {
		tagList = append(tagList, chartVersion.Metadata.Keywords)
	}

	if len(tagList) == 0 {
		return []string{}
	}

	intersection := make([]string, 0)

	// There's only one chart version, so just return its tags
	if len(tagList) == 1 {
		for _, tag := range tagList[0] {
			intersection = append(intersection, tag)
		}

		return intersection
	}

Search:
	for _, searchTag := range tagList[0] {
		for _, other := range tagList[1:] {
			if !hasTag(searchTag, other) {
				// Stop searching for that tag if it isn't found in one of the charts
				continue Search
			}
		}

		// The tag has been found in all of the other keyword lists, so add it
		intersection = append(intersection, searchTag)
	}

	return intersection
}

func generateOperationName(prefix string) string {
	return fmt.Sprintf("%s%x", prefix, rand.Int31())
}

func (c *Client) getConfigMap(instanceID string) (*corev1.ConfigMap, error) {
	configMapInterface := c.coreClient.CoreV1().ConfigMaps(c.namespace)
	config, err := configMapInterface.Get(instanceID, metav1.GetOptions{})
	if err != nil {
		// Do not wrap the error to keep apierrors.IsNotFound() working correctly
		return nil, err
	}
	return config, nil
}

// updateConfigMap will update the config map data for the given instance; it is
// expected that the config map already exists.
// Each value in data may be either a string (in which case it is set), or nil
// (in which case it is removed); any other value will panic.
func (c *Client) updateConfigMap(instanceID string, data map[string]interface{}) error {
	config, err := c.getConfigMap(instanceID)
	if err != nil {
		return err
	}
	for name, value := range data {
		if value == nil {
			delete(config.Data, name)
		} else if stringValue, ok := value.(string); ok {
			config.Data[name] = stringValue
		} else {
			panic(fmt.Sprintf("Invalid data (key %s), has value %+v", name, value))
		}
	}

	configMapInterface := c.coreClient.CoreV1().ConfigMaps(c.namespace)
	_, err = configMapInterface.Update(config)
	if err != nil {
		return errors.Wrapf(err, "Failed to update config for instance %q", instanceID)
	}
	return nil
}

func (c *Client) ListServices() ([]osb.Service, error) {
	glog.Info("Listing services...")
	var services []osb.Service

	charts, err := c.helm.ListCharts()
	if err != nil {
		return nil, err
	}

	for chart, chartVersions := range charts {
		if _, ok := c.providers[chart]; !ok && c.serviceCatalogEnabledOnly {
			continue
		}

		tags := getTagIntersection(chartVersions)

		svc := osb.Service{
			ID:          chart,
			Name:        chart,
			Description: "Helm Chart for " + chart,
			Bindable:    true,
			Plans:       make([]osb.Plan, 0, len(chartVersions)),
			Tags:        tags,
		}
		appVersions := map[string]*repo.ChartVersion{}
		for _, chartVersion := range chartVersions {
			if chartVersion.AppVersion == "" {
				continue
			}

			curV, err := semver.NewVersion(chartVersion.Version)
			if err != nil {
				fmt.Printf("Skipping %s@%s because %s is not a valid semver", chart, chartVersion.AppVersion, chartVersion.Version)
				continue
			}

			currentMax, ok := appVersions[chartVersion.AppVersion]
			if !ok {
				appVersions[chartVersion.AppVersion] = chartVersion
			} else {
				maxV, _ := semver.NewVersion(currentMax.Version)
				if curV.GreaterThan(maxV) {
					appVersions[chartVersion.AppVersion] = chartVersion
				} else {
					//fmt.Printf("Skipping %s@%s because %s<%s\n", chart, chartVersion.AppVersion, curV, maxV)
					continue
				}
			}
		}

		for _, chartVersion := range appVersions {
			planToken := fmt.Sprintf("%s@%s", chart, chartVersion.AppVersion)
			cleaner := regexp.MustCompile(`[^a-z0-9]`)
			planID := cleaner.ReplaceAllString(strings.ToLower(planToken), "-")
			planName := cleaner.ReplaceAllString(chartVersion.AppVersion, "-")
			plan := osb.Plan{
				ID:          planID,
				Name:        planName,
				Description: chartVersion.Description,
				Free:        boolPtr(true),
			}
			svc.Plans = append(svc.Plans, plan)
		}

		if len(svc.Plans) == 0 {
			continue
		}
		services = append(services, svc)
	}

	glog.Infoln("List complete")
	return services, nil
}

// Provision a new service instance.  Returns the async operation key (if
// acceptsIncomplete is set).
func (c *Client) Provision(instanceID, serviceID, planID, namespace string, acceptsIncomplete bool, provisionParams map[string]interface{}) (string, error) {
	chartName := serviceID
	// The way I'm turning charts into plans is not reversible
	chartVersion := strings.Replace(planID, serviceID+"-", "", 1)
	chartVersion = strings.Replace(chartVersion, "-", ".", -1)

	glog.Info("persisting the provisioning parameters...")
	paramsJSON, err := json.Marshal(provisionParams)
	if err != nil {
		return "", errors.Wrapf(err, "could not marshall provisioning parameters %v", provisionParams)
	}
	config := corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instanceID,
			Namespace: c.namespace,
			Labels: map[string]string{
				ServiceKey: serviceID,
				PlanKey:    planID,
			},
		},
		Data: map[string]string{
			ProvisionParamsKey: string(paramsJSON),
			ServiceKey:         serviceID,
			PlanKey:            planID,
		},
	}
	_, err = c.coreClient.CoreV1().ConfigMaps(config.Namespace).Create(&config)
	if err != nil {
		// TODO: compare provision parameters and ignore this call if it's the same
		if apierrors.IsAlreadyExists(err) {
			return "", osb.HTTPStatusCodeError{
				StatusCode:   http.StatusConflict,
				ErrorMessage: &[]string{ConcurrencyErrorMessage}[0],
				Description:  &[]string{ConcurrencyErrorDescription}[0],
			}
		}
		return "", errors.Wrapf(err, "could not persist the instance configmap for %q", instanceID)
	}

	if acceptsIncomplete {
		operationKey := generateOperationName(OperationPrefixProvision)
		err = c.updateConfigMap(instanceID, map[string]interface{}{
			OperationStateKey:       string(osb.StateInProgress),
			OperationNameKey:        operationKey,
			OperationDescriptionKey: fmt.Sprintf("provisioning service instance %q", instanceID),
		})
		if err != nil {
			return "", errors.Wrapf(err, "Failed to set operation key when provisioning instance %q", instanceID)
		}
		go func() {
			err = c.provisionSynchronously(instanceID, namespace, serviceID, planID, chartName, chartVersion, provisionParams)
			if err == nil {
				err = c.updateConfigMap(instanceID, map[string]interface{}{
					OperationStateKey:       string(osb.StateSucceeded),
					OperationDescriptionKey: fmt.Sprintf("service instance %q provisioned", instanceID),
				})
			} else {
				glog.Errorf("Failed to provision %q: %s", instanceID, err)
				err = c.updateConfigMap(instanceID, map[string]interface{}{
					OperationStateKey:       string(osb.StateFailed),
					OperationDescriptionKey: fmt.Sprintf("service instance %q failed to provision", instanceID),
				})
				if err != nil {
					glog.Errorf("Could not update operation state when provisioning asynchronously: %s", err)
				}
			}
		}()
		return operationKey, nil
	}

	err = c.provisionSynchronously(instanceID, namespace, serviceID, planID, chartName, chartVersion, provisionParams)
	if err != nil {
		return "", err
	}

	return "", nil
}

// provisionSynchronously will provision the service instance synchronously.
func (c *Client) provisionSynchronously(instanceID, namespace, serviceID, planID, chartName, chartVersion string, provisionParams map[string]interface{}) error {
	glog.Infof("provisioning %s/%s using stable helm chart %s@%s...", serviceID, planID, chartName, chartVersion)

	chartDef, err := c.helm.GetChart(chartName, chartVersion)
	if err != nil {
		return err
	}

	chartURL := chartDef.URLs[0]

	tc, close, err := c.connectTiller()
	if err != nil {
		return err
	}
	defer close()

	chart, err := helm.LoadChart(chartURL)
	if err != nil {
		return err
	}

	resp, err := tc.Create(chart, namespace, provisionParams)
	if err != nil {
		return err
	}

	// Store any required metadata necessary for bind and deprovision as labels on the resources itself
	glog.Infof("Labeling chart resources with instance %q...", instanceID)
	filterByRelease := metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(map[string]string{
			HeritageLabel: TillerHeritage,
			ReleaseLabel:  resp.Release.Name,
		}).String(),
	}
	services, err := c.coreClient.CoreV1().Services(namespace).List(filterByRelease)
	if err != nil {
		return err
	}
	for _, service := range services.Items {
		err := c.labelService(service, instanceID, provisionParams)
		if err != nil {
			return err
		}
	}
	secrets, err := c.coreClient.CoreV1().Secrets(namespace).List(filterByRelease)
	if err != nil {
		return err
	}
	for _, secret := range secrets.Items {
		err := c.labelSecret(secret, instanceID)
		if err != nil {
			return err
		}
	}

	err = c.updateConfigMap(instanceID, map[string]interface{}{
		ReleaseLabel:        resp.Release.Name,
		ReleaseNamespaceKey: resp.Release.Namespace,
	})
	if err != nil {
		return errors.Wrapf(err, "could not update the instance configmap for %q", instanceID)
	}

	glog.Infof("provision of %v@%v (%v@%v) complete\n%s\n",
		chartName, chartVersion, resp.Release.Name, resp.Release.Version, spew.Sdump(resp.Release.Manifest))

	return nil
}

func (c *Client) labelService(service corev1.Service, instanceID string, params map[string]interface{}) error {
	labeledService := service.DeepCopy()
	labeledService.Labels[InstanceLabel] = instanceID

	original, err := json.Marshal(service)
	if err != nil {
		return err
	}
	modified, err := json.Marshal(labeledService)
	if err != nil {
		return err
	}
	patch, err := strategicpatch.CreateTwoWayMergePatch(original, modified, labeledService)
	if err != nil {
		return err
	}

	_, err = c.coreClient.CoreV1().Services(service.Namespace).Patch(service.Name, types.StrategicMergePatchType, patch)
	if err != nil {
		return errors.Wrapf(err, "failed to label service %s/%s with service metadata", service.Namespace, service.Name)
	}

	return nil
}

func (c *Client) labelSecret(secret corev1.Secret, instanceID string) error {
	labeledSecret := secret.DeepCopy()
	labeledSecret.Labels[InstanceLabel] = instanceID

	original, err := json.Marshal(secret)
	if err != nil {
		return err
	}
	modified, err := json.Marshal(labeledSecret)
	if err != nil {
		return err
	}
	patch, err := strategicpatch.CreateTwoWayMergePatch(original, modified, labeledSecret)
	if err != nil {
		return err
	}

	_, err = c.coreClient.CoreV1().Secrets(secret.Namespace).Patch(secret.Name, types.StrategicMergePatchType, patch)
	if err != nil {
		return errors.Wrapf(err, "failed to label secret %s/%s with service metadata", secret.Namespace, secret.Name)
	}

	return nil
}

func (c *Client) connectTiller() (*tiller.Client, func(), error) {
	config := tiller.Config{
		Host: "localhost",
		Port: 44134,
	}
	tc, err := config.NewClient()
	if err != nil {
		return nil, nil, err
	}
	close := func() {
		err := tc.Close()
		if err != nil {
			glog.Errorln(errors.Wrapf(err, "failed to disconnect tiller client"))
		}
	}

	return tc, close, nil
}

// Bind the given service instance (of the given service) asynchronously; the
// binding operation key is returned.
func (c *Client) Bind(instanceID, serviceID, bindingID string, acceptsIncomplete bool, bindParams map[string]interface{}) (string, error) {
	config, err := c.getConfigMap(instanceID)
	if err != nil {
		if apierrors.IsNotFound(err) {
			msg := fmt.Sprintf("could not find configmap %s/%s", c.namespace, instanceID)
			return "", osb.HTTPStatusCodeError{
				StatusCode:   http.StatusNotFound,
				ErrorMessage: &msg,
			}
		}
		return "", err
	}
	releaseNamespace := config.Data[ReleaseNamespaceKey]
	rawProvisionParams := config.Data[ProvisionParamsKey]
	operationName := generateOperationName(OperationPrefixBind)

	var provisionParams map[string]interface{}
	err = json.Unmarshal([]byte(rawProvisionParams), &provisionParams)
	if err != nil {
		return "", errors.Wrapf(err, "could not unmarshall provision parameters for instance %q", instanceID)
	}

	if acceptsIncomplete {
		go func() {
			_ = c.bindSynchronously(instanceID, serviceID, bindingID, releaseNamespace, bindParams, provisionParams)
		}()
		return operationName, nil
	}

	if err = c.bindSynchronously(instanceID, serviceID, bindingID, releaseNamespace, bindParams, provisionParams); err != nil {
		return "", err
	}
	return "", nil
}

// bindSynchronously creates a new binding for the given service instance.  All
// results are only reported via the service instance configmap (under the
// appropriate key for the binding) for lookup by LastBindingOperationState().
func (c *Client) bindSynchronously(instanceID, serviceID, bindingID, releaseNamespace string, bindParams, provisionParams map[string]interface{}) error {

	// Wrap most of the code in an inner function to simplify error handling
	err := func() error {
		// Smoosh all the params together
		params := make(map[string]interface{}, len(bindParams)+len(provisionParams))
		for k, v := range provisionParams {
			params[k] = v
		}
		for k, v := range bindParams {
			params[k] = v
		}

		filterByInstance := metav1.ListOptions{
			LabelSelector: labels.SelectorFromSet(map[string]string{
				InstanceLabel: instanceID,
			}).String(),
		}

		services, err := c.coreClient.CoreV1().Services(releaseNamespace).List(filterByInstance)
		if err != nil {
			return err
		}
		if len(services.Items) == 0 {
			return osb.HTTPStatusCodeError{StatusCode: http.StatusNotFound}
		}

		secrets, err := c.coreClient.CoreV1().Secrets(releaseNamespace).List(filterByInstance)
		if err != nil {
			return err
		}
		if len(secrets.Items) == 0 {
			return osb.HTTPStatusCodeError{StatusCode: http.StatusNotFound}
		}

		data := make(map[string]interface{})
		for _, secret := range secrets.Items {
			for key, value := range secret.Data {
				data[key] = string(value)
			}
		}

		// Apply additional provisioning logic for Service Catalog Enabled services
		provider, ok := c.providers[serviceID]
		if ok {
			creds, err := provider.Bind(services.Items, params, data)
			if err != nil {
				return errors.Wrapf(err, "unable to bind instance %s", instanceID)
			}
			for k, v := range creds.ToMap() {
				data[k] = v
			}
		}

		// Record the result for later fetching
		bindingResponse := osb.GetBindingResponse{
			Credentials: data,
			Parameters:  bindParams,
		}
		bindingResponseJSON, err := json.Marshal(bindingResponse)
		if err != nil {
			return err
		}

		err = c.updateConfigMap(instanceID, map[string]interface{}{
			(BindingKeyPrefix + bindingID): string(bindingResponseJSON),
		})
		if err != nil {
			return err
		}

		return nil
	}()

	operationState := osb.LastOperationResponse{}
	if err == nil {
		operationState.State = osb.StateSucceeded
	} else {
		glog.Errorf("Error binding instance %s: %s", instanceID, err)
		operationState.State = osb.StateFailed
		operationState.Description = strPtr(fmt.Sprintf("Failed to bind instance %q", instanceID))
	}
	operationStateJSON, marshalError := json.Marshal(operationState)
	if marshalError != nil {
		glog.Errorf("Error serializing bind operation state: %s", marshalError)
		if err != nil {
			return err
		}
		return marshalError
	}
	updates := map[string]interface{}{
		(BindingStateKeyPrefix + bindingID): string(operationStateJSON),
	}
	updateError := c.updateConfigMap(instanceID, updates)
	if updateError != nil {
		glog.Errorf("Error updating bind status: %s", updateError)
		if err != nil {
			return err
		}
		return updateError
	}
	return nil
}

// Unbind a previously-bound instance binding.
func (c *Client) Unbind(instanceID, bindingID string) error {
	// The only clean up we need to do is to remove the binding information.
	data := map[string]interface{}{
		(BindingStateKeyPrefix + bindingID): nil,
		(BindingKeyPrefix + bindingID):      nil,
	}
	if err := c.updateConfigMap(instanceID, data); err != nil {
		return err
	}

	return nil
}

func (c *Client) GetBinding(instanceID, bindingID string) (*osb.GetBindingResponse, error) {
	config, err := c.getConfigMap(instanceID)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, osb.HTTPStatusCodeError{StatusCode: http.StatusNotFound}
		}
		return nil, errors.Wrapf(err, "failed to get service instance %q data", instanceID)
	}
	jsonData, ok := config.Data[BindingKeyPrefix+bindingID]
	if !ok {
		return nil, osb.HTTPStatusCodeError{StatusCode: http.StatusNotFound}
	}
	var data *osb.GetBindingResponse
	err = json.Unmarshal([]byte(jsonData), &data)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not decode binding data")
	}
	return data, nil
}

func (c *Client) Deprovision(instanceID string, acceptsIncomplete bool) (string, error) {
	config, err := c.coreClient.CoreV1().ConfigMaps(c.namespace).Get(instanceID, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return "", osb.HTTPStatusCodeError{StatusCode: http.StatusGone}
		}
		return "", err
	}
	release := config.Data[ReleaseLabel]

	if !acceptsIncomplete {
		err = c.deprovisionSynchronously(instanceID, release)
		if err != nil {
			return "", err
		}
		return "", nil
	}

	operationKey := generateOperationName(OperationPrefixDeprovision)
	err = c.updateConfigMap(instanceID, map[string]interface{}{
		OperationStateKey:       string(osb.StateInProgress),
		OperationNameKey:        operationKey,
		OperationDescriptionKey: fmt.Sprintf("deprovisioning service instance %q", instanceID),
	})
	if err != nil {
		return "", errors.Wrapf(err, "Failed to set operation key when deprovisioning instance %s", instanceID)
	}
	go func() {
		err = c.deprovisionSynchronously(instanceID, release)
		if err == nil {
			// After deprovisioning, there is no config map to update
			return
		}
		glog.Errorf("Failed to deprovision %q: %s", instanceID, err)
		err = c.updateConfigMap(instanceID, map[string]interface{}{
			OperationStateKey:       string(osb.StateFailed),
			OperationDescriptionKey: fmt.Sprintf("service instance %q failed to deprovision", instanceID),
		})
		if err != nil {
			glog.Errorf("Could not update operation state when deprovisioning asynchronously: %s", err)
		}
	}()
	return operationKey, nil
}

func (c *Client) deprovisionSynchronously(instanceID, release string) error {
	tc, close, err := c.connectTiller()
	if err != nil {
		return err
	}
	defer close()

	_, err = tc.Delete(release)
	if err != nil {
		return errors.Wrapf(err, "could not delete release %s", release)
	}

	glog.Infof("release %s deleted", release)

	err = c.coreClient.CoreV1().ConfigMaps(c.namespace).Delete(instanceID, &metav1.DeleteOptions{})
	if err != nil {
		return errors.Wrapf(err, "could not delete configmap %s/%s", c.namespace, instanceID)
	}

	glog.Infof("deprovision of %q is complete", instanceID)
	return nil
}

// LastOperationState returns the status of the last asynchronous operation.
func (c *Client) LastOperationState(instanceID string, operationKey osb.OperationKey) (*osb.LastOperationResponse, error) {
	config, err := c.coreClient.CoreV1().ConfigMaps(c.namespace).Get(instanceID, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			glog.V(5).Infof("last operation on missing instance \"%s\"", instanceID)
			return nil, osb.HTTPStatusCodeError{
				StatusCode: http.StatusGone,
			}
		}
		glog.Infof("could not get instance state of \"%s\": %s", instanceID, err)
		return nil, err
	}

	if config.Data[OperationNameKey] != string(operationKey) {
		// Got unexpected operation key.
		return nil, osb.HTTPStatusCodeError{
			StatusCode:   http.StatusBadRequest,
			ErrorMessage: strPtr(ConcurrencyErrorMessage),
			Description:  strPtr(ConcurrencyErrorDescription),
		}
	}

	description := config.Data[OperationDescriptionKey]
	return &osb.LastOperationResponse{
		State:       osb.LastOperationState(config.Data[OperationStateKey]),
		Description: &description,
	}, nil
}

func boolPtr(value bool) *bool {
	return &value
}

func strPtr(value string) *string {
	return &value
}

func (c *Client) LastBindingOperationState(instanceID, bindingID string) (*osb.LastOperationResponse, error) {
	config, err := c.getConfigMap(instanceID)
	if err != nil {
		if apierrors.IsNotFound(err) {
			glog.V(5).Infof(`last binding operation on missing instance "%s"`, instanceID)
			return nil, osb.HTTPStatusCodeError{
				StatusCode: http.StatusGone,
			}
		}
	}

	stateJSON, ok := config.Data[BindingStateKeyPrefix+bindingID]
	if !ok {
		glog.V(5).Infof(`last binding operation on missing binding "%s" of instance "%s"`, bindingID, instanceID)
		return nil, osb.HTTPStatusCodeError{
			StatusCode: http.StatusGone,
		}
	}

	var response *osb.LastOperationResponse
	err = json.Unmarshal([]byte(stateJSON), &response)
	if err != nil {
		return nil, errors.Wrapf(err, "Error unmarshalling binding state %s", stateJSON)
	}

	return response, nil
}
