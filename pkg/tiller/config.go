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
	"fmt"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	klog "k8s.io/klog/v2"
)

// maxMsgSize use 20MB as the default message size limit.
// grpc library default is 4MB
const maxMsgSize = 1024 * 1024 * 20

type Config struct {
	Host string
	Port int
}

func NewConfig(host string, port int) Config {
	return Config{host, port}
}

func (c Config) Connect() (*grpc.ClientConn, error) {
	tillerHost := fmt.Sprintf("%v:%v", c.Host, c.Port)
	klog.Infof("connecting to tiller at %v ...", tillerHost)

	opts := []grpc.DialOption{
		grpc.WithInsecure(),
		grpc.WithBlock(),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			// Send keepalive every 30 seconds to prevent the connection from
			// getting closed by upstreams
			Time: time.Duration(30) * time.Second,
		}),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(maxMsgSize)),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, err := grpc.DialContext(ctx, tillerHost, opts...)
	if err != nil {
		return conn, errors.Wrapf(err, "cannot connect to tiller at %v", tillerHost)
	}
	klog.Infoln("Connected!")
	return conn, nil
}

func (c Config) NewClient() (*Client, error) {
	conn, err := c.Connect()
	if err != nil {
		return nil, err
	}
	return NewClient(conn), nil
}
