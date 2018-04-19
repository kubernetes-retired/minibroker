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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/helm/pkg/repo"
)

const (
	InstanceLabel = "minibroker.instance"
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
	// TODO: The way I'm turning charts into plans is not reversible. Need a data store.
	chartVersion := strings.Replace(planID, serviceID+"-", "", 1)
	chartVersion = strings.Replace(chartVersion, "-", ".", -1)

	glog.Infof("provisioning %s/%s using stable helm chart %s@%s...", serviceID, planID, chartName, chartVersion)

	chartDef, err := c.helm.GetChart(chartName, chartVersion)
	if err != nil {
		return err
	}

	config := tiller.Config{
		Host: "localhost",
		Port: 44134,
	}
	tc, err := config.NewClient()
	if err != nil {
		return err
	}
	defer func() {
		err := tc.Close()
		if err != nil {
			log.Print(errors.Wrapf(err, "failed to disconnect tiller client"))
		}
	}()

	chart, err := helm.LoadChart(chartDef)
	if err != nil {
		return err
	}

	resp, err := tc.Create(chart, namespace)
	if err != nil {
		return err
	}

	// Label the secret with the instance id so that we don't need a datastore
	opts := metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(map[string]string{
			"heritage": "Tiller",
			"release":  resp.Release.Name,
		}).String(),
	}
	secrets, err := c.coreClient.CoreV1().Secrets(resp.Release.Namespace).List(opts)
	if err != nil {
		return err
	}

	for _, secret := range secrets.Items {
		secret.Labels[InstanceLabel] = instanceID
		_, err := c.coreClient.CoreV1().Secrets(secret.Namespace).Update(&secret)
		if err != nil {
			return errors.Wrapf(err, "failed to label %s/%s with %s", secret.Namespace, secret.Name, instanceID)
		}
	}

	glog.Infof("provision of %v@%v (%v@%v) complete\n%s\n",
		chartName, chartVersion, resp.Release.Name, resp.Release.Version, spew.Sdump(resp.Release.Manifest))

	return nil
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

func boolPtr(value bool) *bool {
	return &value
}
