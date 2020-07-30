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

package minibroker

import (
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
)

const redisProtocolName = "redis"

type RedisProvider struct{}

func (p RedisProvider) Bind(services []corev1.Service, params map[string]interface{}, chartSecrets map[string]interface{}) (*Credentials, error) {
	var masterSvc *corev1.Service
	for _, svc := range services {
		if svc.Spec.Selector["role"] == "master" {
			masterSvc = &svc
			break
		}
	}
	if masterSvc == nil {
		return nil, errors.New("could not identify the master service")
	}

	if len(masterSvc.Spec.Ports) == 0 {
		return nil, errors.Errorf("no ports found")
	}
	svcPort := masterSvc.Spec.Ports[0]

	host := buildHostFromService(*masterSvc)

	var password string
	passwordVal, ok := chartSecrets["redis-password"]
	if !ok {
		return nil, errors.Errorf("redis-password not found in secret keys")
	}
	password = passwordVal.(string)

	creds := Credentials{
		Protocol: redisProtocolName,
		Port:     svcPort.Port,
		Host:     host,
		Password: password,
	}
	creds.URI = buildURI(creds)

	return &creds, nil
}
