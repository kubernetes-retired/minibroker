package minibroker

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"

	"github.com/Masterminds/semver"
	"github.com/davecgh/go-spew/spew"
	"github.com/golang/glog"
	"github.com/osbkit/minibroker/pkg/helm"
	"github.com/osbkit/minibroker/pkg/tiller"
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
	InstanceLabel      = "minibroker.instance"
	ServiceKey         = "service-id"
	PlanKey            = "plan-id"
	ProvisionParamsKey = "provision-params"
	HeritageLabel      = "heritage"
	ReleaseLabel       = "release"
	TillerHeritage     = "Tiller"
)

type Client struct {
	helm       *helm.Client
	namespace  string
	coreClient kubernetes.Interface
	providers  map[string]Provider
}

func NewClient(repoURL string) *Client {
	return &Client{
		helm:       helm.NewClient(repoURL),
		coreClient: loadInClusterClient(),
		namespace:  loadNamespace(),
		providers: map[string]Provider{
			"mysql":      MySQLProvider{},
			"mariadb":    MariadbProvider{},
			"postgresql": PostgresProvider{},
			"mongodb":    MongodbProvider{},
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

func (c *Client) ListServices() ([]osb.Service, error) {
	glog.Info("Listing services...")
	var services []osb.Service

	charts, err := c.helm.ListCharts()
	if err != nil {
		return nil, err
	}

	for chart, chartVersions := range charts {
		svc := osb.Service{
			ID:          chart,
			Name:        chart,
			Description: "Helm Chart for " + chart,
			Bindable:    true,
			Plans:       make([]osb.Plan, 0, len(chartVersions)),
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

func (c *Client) Provision(instanceID, serviceID, planID, namespace string, provisionParams map[string]interface{}) error {
	chartName := serviceID
	// The way I'm turning charts into plans is not reversible
	chartVersion := strings.Replace(planID, serviceID+"-", "", 1)
	chartVersion = strings.Replace(chartVersion, "-", ".", -1)

	glog.Infof("provisioning %s/%s using stable helm chart %s@%s...", serviceID, planID, chartName, chartVersion)

	chartDef, err := c.helm.GetChart(chartName, chartVersion)
	if err != nil {
		return err
	}

	tc, close, err := c.connectTiller()
	if err != nil {
		return err
	}
	defer close()

	chart, err := helm.LoadChart(chartDef)
	if err != nil {
		return err
	}

	resp, err := tc.Create(chart, namespace, provisionParams)
	if err != nil {
		return err
	}

	// Store any required metadata necessary for bind and deprovision as labels on the resources itself
	glog.Info("Labeling chart resources with instance %q...", instanceID)
	filterByRelease := metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(map[string]string{
			HeritageLabel: TillerHeritage,
			ReleaseLabel:  resp.Release.Name,
		}).String(),
	}
	services, err := c.coreClient.CoreV1().Services(c.namespace).List(filterByRelease)
	if err != nil {
		return err
	}
	for _, service := range services.Items {
		err := c.labelService(service, instanceID, provisionParams)
		if err != nil {
			return err
		}
	}
	secrets, err := c.coreClient.CoreV1().Secrets(c.namespace).List(filterByRelease)
	if err != nil {
		return err
	}
	for _, secret := range secrets.Items {
		err := c.labelSecret(secret, instanceID)
		if err != nil {
			return err
		}
	}

	glog.Info("persisting the provisioning parameters...")
	paramsJson, err := json.Marshal(provisionParams)
	if err != nil {
		return errors.Wrapf(err, "could not marshall provisioning parameters %v", provisionParams)
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
			ProvisionParamsKey: string(paramsJson),
			ServiceKey:         serviceID,
			PlanKey:            planID,
		},
	}
	_, err = c.coreClient.CoreV1().ConfigMaps(config.Namespace).Create(&config)
	if err != nil {
		return errors.Wrapf(err, "could not persist the instance configmap for %q", instanceID)
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

	_, err = c.coreClient.CoreV1().Services(c.namespace).Patch(service.Name, types.StrategicMergePatchType, patch)
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

	_, err = c.coreClient.CoreV1().Secrets(c.namespace).Patch(secret.Name, types.StrategicMergePatchType, patch)
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

func (c *Client) Bind(instanceID, serviceID string, bindParams map[string]interface{}) (map[string]interface{}, error) {
	config, err := c.coreClient.CoreV1().ConfigMaps(c.namespace).Get(instanceID, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, osb.HTTPStatusCodeError{StatusCode: http.StatusNotFound}
		}
		return nil, err
	}

	var provisionParams map[string]interface{}
	err = json.Unmarshal([]byte(config.Data[ProvisionParamsKey]), &provisionParams)
	if err != nil {
		return nil, errors.Wrapf(err, "could not unmarshall provision parameters for instance %q", instanceID)
	}

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

	services, err := c.coreClient.CoreV1().Services(c.namespace).List(filterByInstance)
	if err != nil {
		return nil, err
	}
	if len(services.Items) == 0 {
		return nil, osb.HTTPStatusCodeError{StatusCode: http.StatusNotFound}
	}
	if len(services.Items) > 1 {
		return nil, errors.Errorf("more than one service labeled with %q", filterByInstance.LabelSelector)
	}
	service := services.Items[0]

	secrets, err := c.coreClient.CoreV1().Secrets(c.namespace).List(filterByInstance)
	if err != nil {
		return nil, err
	}
	if len(secrets.Items) == 0 {
		return nil, osb.HTTPStatusCodeError{StatusCode: http.StatusNotFound}
	}

	data := make(map[string]interface{})
	for _, secret := range secrets.Items {
		for key, value := range secret.Data {
			data[key] = string(value)
		}
	}

	// Apply additional provisioning logic for registered services
	provider, ok := c.providers[serviceID]
	if ok {
		creds, err := provider.Bind(service, params, data)
		if err != nil {
			return nil, errors.Wrapf(err, "unable to bind service %s/%s (%s)", service.Namespace, service.Name, instanceID)
		}
		for k, v := range creds.ToMap() {
			data[k] = v
		}
	}

	return data, nil
}

func (c *Client) Deprovision(instanceID string) error {
	filterByInstance := metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(map[string]string{
			InstanceLabel: instanceID,
		}).String(),
	}
	services, err := c.coreClient.CoreV1().Services(c.namespace).List(filterByInstance)
	if err != nil {
		return err
	}

	if len(services.Items) == 0 {
		return osb.HTTPStatusCodeError{StatusCode: http.StatusNotFound}
	}

	release, ok := services.Items[0].Labels[ReleaseLabel]
	if !ok {
		return errors.Errorf("service is missing the release label")
	}

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

func boolPtr(value bool) *bool {
	return &value
}
