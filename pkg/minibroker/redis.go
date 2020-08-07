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
	"fmt"
	"net/url"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
)

const redisProtocolName = "redis"

type RedisProvider struct{}

func (p RedisProvider) Bind(
	services []corev1.Service,
	_ *BindParams,
	_ *ProvisionParams,
	chartSecrets Object,
) (Object, error) {
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

	password, err := chartSecrets.DigString("redis-password")
	if err != nil {
		switch err {
		case ErrDigNotFound:
			return nil, fmt.Errorf("password not found in secret keys")
		case ErrDigNotString:
			return nil, fmt.Errorf("password not a string")
		default:
			return nil, err
		}
	}

	creds := Object{
		"protocol": redisProtocolName,
		"port":     svcPort.Port,
		"host":     host,
		"password": password,
		"uri": (&url.URL{
			Scheme: redisProtocolName,
			User:   url.UserPassword("", password),
			Host:   fmt.Sprintf("%s:%d", host, svcPort.Port),
		}).String(),
	}

	return creds, nil
}
