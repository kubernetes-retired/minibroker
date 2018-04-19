package tiller

import (
	"context"

	"github.com/davecgh/go-spew/spew"
	"github.com/golang/glog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"gopkg.in/yaml.v2"
	"k8s.io/helm/pkg/proto/hapi/chart"
	rls "k8s.io/helm/pkg/proto/hapi/services"
	"k8s.io/helm/pkg/version"
)

type Client struct {
	conn *grpc.ClientConn
}

func NewClient(conn *grpc.ClientConn) *Client {
	return &Client{conn: conn}
}

func (t *Client) Close() error {
	return t.conn.Close()
}

func (t *Client) Create(ch *chart.Chart, installNS string) (*rls.InstallReleaseResponse, error) {
	baseValues := map[string]interface{}{}
	valuesYaml, _ := yaml.Marshal(baseValues)

	rlsCl := rls.NewReleaseServiceClient(t.conn)
	ctx := newContext()
	req := &rls.InstallReleaseRequest{
		Chart:        ch,
		Namespace:    installNS,
		ReuseName:    true,
		DisableHooks: false,
		Values:       &chart.Config{Raw: string(valuesYaml)},
	}
	glog.Infof("installing release for chart %s\n%s", ch.Metadata.Name, spew.Sdump(req))
	return rlsCl.InstallRelease(ctx, req)
}

func (t *Client) Delete(relName string) (*rls.UninstallReleaseResponse, error) {
	rlsCl := rls.NewReleaseServiceClient(t.conn)
	ctx := newContext()
	req := &rls.UninstallReleaseRequest{
		Name:         relName,
		DisableHooks: false,
	}
	glog.Infof("uninstalling release %s", relName)
	return rlsCl.UninstallRelease(ctx, req)
}

// NewContext creates a versioned context.
func newContext() context.Context {
	md := metadata.Pairs("x-helm-api-client", version.GetVersion())
	return metadata.NewOutgoingContext(context.TODO(), md)
}
