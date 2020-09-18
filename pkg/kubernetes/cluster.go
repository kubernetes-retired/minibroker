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

package kubernetes

import (
	"fmt"
	"io"
	"io/ioutil"
	"strings"
)

// ClusterDomain returns the k8s cluster domain extracted from
// /etc/resolv.conf.
func ClusterDomain(resolvConf io.Reader) (string, error) {
	data, err := ioutil.ReadAll(resolvConf)
	if err != nil {
		return "", fmt.Errorf("failed to get cluster domain: %w", err)
	}

	lines := strings.Split(string(data), "\n")
	var searchLine string
	for _, line := range lines {
		if strings.HasPrefix(line, "search") {
			searchLine = line
		}
	}

	if searchLine == "" {
		err := fmt.Errorf("missing the search path from resolv.conf")
		return "", fmt.Errorf("failed to get cluster domain: %w", err)
	}

	domains := strings.Split(searchLine, " ")
	for i := 1; i < len(domains); i++ {
		if strings.HasPrefix(domains[i], "svc.") {
			return strings.TrimPrefix(domains[i], "svc."), nil
		}
	}

	err = fmt.Errorf("missing domain starting with 'svc.' in the search path")
	return "", fmt.Errorf("failed to get cluster domain: %w", err)
}
