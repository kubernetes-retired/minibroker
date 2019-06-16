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

func (t *Client) Create(ch *chart.Chart, installNS string, values map[string]interface{}) (*rls.InstallReleaseResponse, error) {
	valuesYaml, _ := yaml.Marshal(values)

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
		Purge:        true,
	}
	glog.Infof("uninstalling release %s", relName)
	return rlsCl.UninstallRelease(ctx, req)
}

// NewContext creates a versioned context.
func newContext() context.Context {
	md := metadata.Pairs("x-helm-api-client", version.GetVersion())
	return metadata.NewOutgoingContext(context.TODO(), md)
}
