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
	"os/exec"

	"github.com/pkg/errors"
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
	cmd := exec.Command("helm", "install", name, chart, "-n", h.namespace, "--wait")
	err := cmd.Run()
	if err != nil {
		return errors.Wrap(err, "failed to install helm chart '"+chart+"'")
	}
	return nil
}

func (h Helm) Uninstall(name string) error {
	cmd := exec.Command("helm", "delete", name, "-n", h.namespace)
	err := cmd.Run()
	if err != nil {
		return errors.Wrap(err, "failed to uninstall helm release '"+name+"'")
	}
	return nil
}
