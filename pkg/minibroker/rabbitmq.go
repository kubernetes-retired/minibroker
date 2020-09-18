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

package minibroker

import (
	"fmt"
	"net/url"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
)

const (
	amqpProtocolName        = "amqp"
	defaultRabbitmqUsername = "user"
)

type RabbitmqProvider struct {
	hostBuilder
}

func (p RabbitmqProvider) Bind(
	services []corev1.Service,
	_ *BindParams,
	provisionParams *ProvisionParams,
	chartSecrets Object,
) (Object, error) {
	if len(services) == 0 {
		return nil, errors.Errorf("no services to process")
	}
	service := services[0]

	var svcPort *corev1.ServicePort
	for _, port := range service.Spec.Ports {
		if port.Name == amqpProtocolName {
			svcPort = &port
			break
		}
	}
	if svcPort == nil {
		return nil, errors.Errorf("no amqp port found")
	}

	user, err := provisionParams.DigStringOr("rabbitmq.username", defaultRabbitmqUsername)
	if err != nil {
		return nil, fmt.Errorf("failed to get username: %w", err)
	}

	password, err := chartSecrets.DigString("rabbitmq-password")
	if err != nil {
		return nil, fmt.Errorf("failed to get password: %w", err)
	}

	host := p.hostFromService(&service)
	creds := Object{
		"protocol": amqpProtocolName,
		"port":     svcPort.Port,
		"host":     host,
		"username": user,
		"password": password,
		"uri": (&url.URL{
			Scheme: amqpProtocolName,
			User:   url.UserPassword(user, password),
			Host:   fmt.Sprintf("%s:%d", host, svcPort.Port),
		}).String(),
	}

	return creds, nil
}
