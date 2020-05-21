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

package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"path"
	"strconv"
	"syscall"

	"github.com/kubernetes-sigs/minibroker/pkg/broker"
	"github.com/pmorie/osb-broker-lib/pkg/metrics"
	prom "github.com/prometheus/client_golang/prometheus"
	"k8s.io/klog"

	"github.com/pmorie/osb-broker-lib/pkg/rest"
	"github.com/pmorie/osb-broker-lib/pkg/server"
)

var options struct {
	broker.Options

	Port    int
	TLSCert string
	TLSKey  string
}

func init() {
	klog.InitFlags(nil)

	flag.BoolVar(&options.ServiceCatalogEnabledOnly, "service-catalog-enabled-only", false,
		"Only list Service Catalog Enabled services")
	flag.IntVar(&options.Port, "port", 8005,
		"use '--port' option to specify the port for broker to listen on")
	flag.StringVar(&options.TLSCert, "tlsCert", "",
		"base-64 encoded PEM block to use as the certificate for TLS. If '--tlsCert' is used, then '--tlsKey' must also be used. If '--tlsCert' is not used, then TLS will not be used.")
	flag.StringVar(&options.TLSKey, "tlsKey", "",
		"base-64 encoded PEM block to use as the private key matching the TLS certificate. If '--tlsKey' is used, then '--tlsCert' must also be used")
	flag.StringVar(&options.CatalogPath, "catalogPath", "",
		"The path to the catalog")
	flag.StringVar(&options.HelmRepoURL, "helmUrl", "",
		"The url to the helm repo")
	flag.StringVar(&options.DefaultNamespace, "defaultNamespace", "",
		"The default namespace for brokers when the request doesn't specify")
	flag.Parse()
}

func main() {
	if err := run(); err != nil && err != context.Canceled && err != context.DeadlineExceeded {
		klog.Fatalln(err)
	}
}

func run() error {
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	go cancelOnInterrupt(ctx, cancelFunc)

	return runWithContext(ctx)
}

func runWithContext(ctx context.Context) error {
	if flag.Arg(0) == "version" {
		klog.V(0).Infof("%s/%s", path.Base(os.Args[0]), "0.1.0")
		return nil
	}
	if (options.TLSCert != "" || options.TLSKey != "") &&
		(options.TLSCert == "" || options.TLSKey == "") {
		klog.V(0).Infoln("To use TLS, both --tlsCert and --tlsKey must be used")
		return nil
	}

	addr := ":" + strconv.Itoa(options.Port)

	b, err := broker.NewBroker(options.Options)
	if err != nil {
		return err
	}

	// Prometheus metrics
	reg := prom.NewRegistry()
	osbMetrics := metrics.New()
	reg.MustRegister(osbMetrics)

	api, err := rest.NewAPISurface(b, osbMetrics)
	if err != nil {
		return err
	}

	s := server.New(api, reg)

	klog.V(1).Infof("Starting broker!")

	if options.TLSCert == "" && options.TLSKey == "" {
		err = s.Run(ctx, addr)
	} else {
		err = s.RunTLS(ctx, addr, options.TLSCert, options.TLSKey)
	}
	return err
}

func cancelOnInterrupt(ctx context.Context, f context.CancelFunc) {
	term := make(chan os.Signal)
	signal.Notify(term, os.Interrupt, syscall.SIGTERM)

	for {
		select {
		case <-term:
			klog.V(1).Infof("Received SIGTERM, exiting gracefully...")
			f()
			os.Exit(0)
		case <-ctx.Done():
			os.Exit(0)
		}
	}
}
