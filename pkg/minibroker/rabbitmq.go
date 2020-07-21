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

const amqpProtocolName = "amqp"

type RabbitmqProvider struct{}

func (p RabbitmqProvider) Bind(services []corev1.Service, params map[string]interface{}, chartSecrets map[string]interface{}) (*Credentials, error) {
	if len(services) == 0 {
		return nil, errors.Errorf("no services to process")
	}
	service := services[0]

	var amqpPort *corev1.ServicePort
	for _, port := range service.Spec.Ports {
		if port.Name == amqpProtocolName {
			amqpPort = &port
			break
		}
	}
	if amqpPort == nil {
		return nil, errors.Errorf("no amqp port found")
	}

	passwordVal, ok := chartSecrets["rabbitmq-password"]
	if !ok {
		return nil, errors.Errorf("password not found in secret keys")
	}
	password, ok := passwordVal.(string)
	if !ok {
		return nil, errors.Errorf("invalid password type")
	}

	host := buildHostFromService(service)
	creds := Credentials{
		Protocol: amqpProtocolName,
		Port:     amqpPort.Port,
		Host:     host,
		Username: "user",
		Password: password,
		Database: "/",
	}
	creds.URI = buildRabbitmqURI(creds)

	return &creds, nil
}

func buildRabbitmqURI(c Credentials) string {
	return fmt.Sprintf("%s://%s:%s@%s:%d/%s",
		c.Protocol, c.Username, c.Password, c.Host, c.Port, url.QueryEscape(c.Database))
}
