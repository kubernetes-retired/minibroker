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

const (
	mongodbProtocolName = "mongodb"
	rootMongodbUsername = "root"
)

type MongodbProvider struct{}

func (p MongodbProvider) Bind(
	services []corev1.Service,
	_ *BindParams,
	provisionParams *ProvisionParams,
	chartSecrets Object,
) (Object, error) {
	service := services[0]
	if len(service.Spec.Ports) == 0 {
		return nil, errors.Errorf("no ports found")
	}
	svcPort := service.Spec.Ports[0]

	host := buildHostFromService(service)

	database := provisionParams.DigStringOr("mongodbDatabase", "")
	user := provisionParams.DigStringOr("mongodbUsername", rootMongodbUsername)

	var passwordKey string
	if user == rootMongodbUsername {
		passwordKey = "mongodb-root-password"
	} else {
		passwordKey = "mongodb-password"
	}
	password, err := chartSecrets.DigString(passwordKey)
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
		"protocol": mongodbProtocolName,
		"port":     svcPort.Port,
		"host":     host,
		"username": user,
		"password": password,
		"database": database,
		"uri": (&url.URL{
			Scheme: mongodbProtocolName,
			User:   url.UserPassword(user, password),
			Host:   fmt.Sprintf("%s:%d", host, svcPort.Port),
			Path:   database,
		}).String(),
	}

	return creds, nil
}
