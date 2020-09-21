/*
Copyright 2020 The Kubernetes Authors.

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

package testutil

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"
)

type Helm struct {
	namespace string
}

func NewHelm(ns string) Helm {
	return Helm{
		namespace: ns,
	}
}

func (h Helm) Install(name, chart string) error {
	cmd := exec.Command(
		"helm", "install", name, chart,
		"--wait",
		"--timeout", "15m",
		"--namespace", h.namespace,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start a goroutine to print a waiting message every minute. It stops when
	// the 'cancel' function is called, which is after the Helm command exits
	// successfully or not.
	go func() {
		for i := 0; ; i++ {
			select {
			case <-ctx.Done():
				return
			default:
				// Print every minute but still keep the check for readiness
				// every second.
				if i%60 == 0 {
					fmt.Printf("Waiting for %q to be ready...", name)
				}
				time.Sleep(time.Second)
			}
		}
	}()

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install helm chart %q: %w", chart, err)
	}
	return nil
}

func (h Helm) Uninstall(name string) error {
	cmd := exec.Command("helm", "delete", name, "--namespace", h.namespace)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to uninstall helm release %q: %w", name, err)
	}
	return nil
}
