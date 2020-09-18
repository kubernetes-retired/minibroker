package kubernetes

import (
	"fmt"
	"io"
	"io/ioutil"
	"strings"
)

// ClusterDomain returns the k8s cluster domain extracted from the
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
