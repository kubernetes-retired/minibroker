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
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
)

type RabbitmqProvider struct{}

func (p RabbitmqProvider) Bind(services []corev1.Service, params map[string]interface{}, chartSecrets map[string]interface{}) (*Credentials, error) {
	service := services[0]

	var amqpPort *corev1.ServicePort = nil
	for _, port := range service.Spec.Ports {
		if port.Name == "amqp" {
			amqpPort = &port
			break
		}
	}
	if amqpPort == nil {
		return nil, errors.Errorf("no amqp port found")
	}

	var password string
	passwordVal, ok := chartSecrets["rabbitmq-password"]
	if !ok {
		return nil, errors.Errorf("password not found in secret keys")
	}
	password = passwordVal.(string)

	host := buildHostFromService(service)
	creds := Credentials{
		Protocol: amqpPort.Name,
		Port:     amqpPort.Port,
		Host:     host,
		Username: "user",
		Password: password,
		Database: "%2F",
	}
	creds.URI = buildURI(creds)

	return &creds, nil
}
