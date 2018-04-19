package minibroker

import (
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
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/helm/pkg/repo"
	"encoding/json"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
)

const (
	InstanceLabel        = "minibroker.instance"
	HeritageLabel        = "heritage"
	ReleaseLabel         = "release"
	TillerHeritage       = "Tiller"
)

type Client struct {
	helm       *helm.Client
	namespace  string
	coreClient kubernetes.Interface
}

func NewClient(repoURL string) *Client {
	return &Client{
		helm:       helm.NewClient(repoURL),
		coreClient: loadInClusterClient(),
		namespace:  loadNamespace(),
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

func (c *Client) Provision(instanceID, serviceID, planID, namespace string) error {
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

	resp, err := tc.Create(chart, namespace)
	if err != nil {
		return err
	}

	// Label the deployment with the instance id so that we don't need a datastore for deprovision
	opts := metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(map[string]string{
			HeritageLabel: TillerHeritage,
			ReleaseLabel:  resp.Release.Name,
		}).String(),
	}
	deploys, err := c.coreClient.AppsV1().Deployments(c.namespace).List(opts)
	if err != nil {
		return err
	}
	for _, deploy := range deploys.Items {
		err := c.labelDeployment(deploy, instanceID)
		if err != nil {
			return err
		}
	}

	// Label the secret with the instance id so that we don't need a datastore for bind
	secrets, err := c.coreClient.CoreV1().Secrets(c.namespace).List(opts)
	if err != nil {
		return err
	}
	for _, secret := range secrets.Items {
		err := c.labelSecret(secret, instanceID)
		if err != nil {
			return err
		}
	}

	glog.Infof("provision of %v@%v (%v@%v) complete\n%s\n",
		chartName, chartVersion, resp.Release.Name, resp.Release.Version, spew.Sdump(resp.Release.Manifest))

	return nil
}

func (c *Client) labelDeployment(deploy appsv1.Deployment, instanceID string) error {
	labeledDeploy := deploy.DeepCopy()
	labeledDeploy.Labels[InstanceLabel] = instanceID

	original, err := json.Marshal(deploy)
	if err != nil {
		return err
	}
	modified, err := json.Marshal(labeledDeploy)
	if err != nil {
		return err
	}
	patch, err := strategicpatch.CreateTwoWayMergePatch(original, modified, labeledDeploy)
	if err != nil {
		return err
	}

	_, err = c.coreClient.AppsV1().Deployments(c.namespace).Patch(deploy.Name, types.StrategicMergePatchType, patch)
	if err != nil {
		return errors.Wrapf(err, "failed to label deployment %s/%s with %s", deploy.Namespace, deploy.Name, instanceID)
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
		return errors.Wrapf(err, "failed to label secret %s/%s with %s", secret.Namespace, secret.Name, instanceID)
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

func (c *Client) Bind(instanceID string) (map[string]interface{}, error) {
	opts := metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(map[string]string{
			InstanceLabel: instanceID,
		}).String(),
	}
	secrets, err := c.coreClient.CoreV1().Secrets(c.namespace).List(opts)
	if err != nil {
		return nil, err
	}

	if len(secrets.Items) == 0 {
		return nil, osb.HTTPStatusCodeError{StatusCode: http.StatusNotFound}
	}

	creds := make(map[string]interface{})
	for _, secret := range secrets.Items {
		for key, value := range secret.Data {
			creds[key] = string(value)
		}
	}

	return creds, nil
}

func (c *Client) Deprovision(instanceID string) error {
	opts := metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(map[string]string{
			InstanceLabel: instanceID,
		}).String(),
	}
	deploys, err := c.coreClient.AppsV1().Deployments(c.namespace).List(opts)
	if err != nil {
		return err
	}

	if len(deploys.Items) == 0 {
		return osb.HTTPStatusCodeError{StatusCode: http.StatusNotFound}
	}

	release, ok := deploys.Items[0].Labels[ReleaseLabel]
	if !ok {
		return errors.Errorf("deployment is missing the release label")
	}

	tc, close, err := c.connectTiller()
	if err != nil {
		return err
	}
	defer close()

	_, err = tc.Delete(release)
	if err != nil {
		return err
	}

	glog.Infof("deprovision of %s complete", release)

	return nil
}

func boolPtr(value bool) *bool {
	return &value
}
