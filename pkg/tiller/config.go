package tiller

import (
	"context"
	"fmt"
	"time"

	"github.com/golang/glog"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
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
	glog.Infof("connecting to tiller at %v ...", tillerHost)

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
	glog.Infoln("Connected!")
	return conn, nil
}

func (c Config) NewClient() (*Client, error) {
	conn, err := c.Connect()
	if err != nil {
		return nil, err
	}
	return NewClient(conn), nil
}
