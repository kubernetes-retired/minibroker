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
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/kubernetes-sigs/minibroker/pkg/broker"
	"github.com/kubernetes-sigs/minibroker/pkg/kubernetes"
	"github.com/pmorie/osb-broker-lib/pkg/metrics"
	prom "github.com/prometheus/client_golang/prometheus"
	klog "k8s.io/klog/v2"

	"github.com/pmorie/osb-broker-lib/pkg/rest"
	"github.com/pmorie/osb-broker-lib/pkg/server"
)

var (
	version   = "0.0.0"
	buildDate = ""
)

var options struct {
	broker.Options

	Port    int
	TLSCert string
	TLSKey  string
}

func main() {
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
	flag.StringVar(&options.ProvisioningSettingsPath, "provisioningSettings", "",
		"The path to the YAML file where the optional provisioning settings are stored")
	flag.StringVar(&options.ClusterDomain, "clusterDomain", "",
		"The k8s cluster domain - if not set, Minibroker infers from /etc/resolv.conf")
	flag.Parse()

	klogFlags := flag.NewFlagSet("klog", flag.ExitOnError)
	klog.InitFlags(klogFlags)
	// Sync the glog and klog flags.
	flag.CommandLine.VisitAll(func(f1 *flag.Flag) {
		f2 := klogFlags.Lookup(f1.Name)
		if f2 != nil {
			value := f1.Value.String()
			f2.Value.Set(value)
		}
	})
	defer klog.Flush()

	if options.ClusterDomain == "" {
		resolvConf, err := os.Open("/etc/resolv.conf")
		if err != nil {
			klog.Fatalln(err)
		}
		// An assurance for the future-proof copy-paste of this block! Yes, this
		// is not necessary here but shall be if other events happen in the
		// future.
		defer resolvConf.Close()

		if options.ClusterDomain, err = kubernetes.ClusterDomain(resolvConf); err != nil {
			klog.Fatalln(err)
		}
		resolvConf.Close()
	}

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
		printVersion()
		return nil
	}
	if (options.TLSCert != "" || options.TLSKey != "") &&
		(options.TLSCert == "" || options.TLSKey == "") {
		err := fmt.Errorf("failed to start Minibroker: to use TLS, both --tlsCert and --tlsKey must be used")
		return err
	}

	addr := ":" + strconv.Itoa(options.Port)

	options.Options.ConfigNamespace = os.Getenv("CONFIG_NAMESPACE")

	b, err := broker.NewBrokerFromOptions(options.Options)
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

	klog.V(1).Infof("starting broker!")

	if options.TLSCert == "" && options.TLSKey == "" {
		err = s.Run(ctx, addr)
	} else {
		err = s.RunTLS(ctx, addr, options.TLSCert, options.TLSKey)
	}
	return err
}

func cancelOnInterrupt(ctx context.Context, f context.CancelFunc) {
	term := make(chan os.Signal, 1)
	signal.Notify(term, os.Interrupt, syscall.SIGTERM)

	for {
		select {
		case <-term:
			klog.V(1).Infof("received SIGTERM, exiting gracefully...")
			f()
			os.Exit(0)
		case <-ctx.Done():
			os.Exit(0)
		}
	}
}

func printVersion() {
	v := map[string]interface{}{
		"version":    version,
		"build_date": buildDate,
	}
	encoder := json.NewEncoder(os.Stderr)
	encoder.SetIndent("", "  ")
	encoder.Encode(v)
}
