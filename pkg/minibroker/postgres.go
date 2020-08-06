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

const postgresqlProtocolName = "postgresql"

type PostgresProvider struct{}

func (p PostgresProvider) Bind(services []corev1.Service, params map[string]interface{}, chartSecrets map[string]interface{}) (*Credentials, error) {
	service := services[0]
	if len(service.Spec.Ports) == 0 {
		return nil, errors.Errorf("no ports found")
	}
	svcPort := service.Spec.Ports[0]

	host := buildHostFromService(service)

	dbVal, ok := params["postgresqlDatabase"]
	if !ok {
		// Some older chart versions use postgresDatabase instead of postgresqlDatabase.
		dbVal, ok = params["postgresDatabase"]
		if !ok {
			dbVal = ""
		}
	}
	database, ok := dbVal.(string)
	if !ok {
		return nil, errors.Errorf("database not a string")
	}

	userVal, ok := params["postgresqlUsername"]
	if !ok {
		// Some older chart versions use postgresUsername instead of postgresqlUsername.
		userVal, ok = params["postgresUsername"]
		if !ok {
			userVal = "postgres"
		}
	}
	user, ok := userVal.(string)
	if !ok {
		return nil, errors.Errorf("username not a string")
	}

	var password string
	if user != "postgres" {
		// postgresql-postgres-password is used when postgresqlPostgresPassword is set and
		// postgresqlUsername is not 'postgres'.
		passwordVal, ok := chartSecrets["postgresql-postgres-password"]
		if !ok {
			passwordVal, ok = chartSecrets["postgresql-password"]
			if !ok {
				return nil, errors.Errorf("password not found in secret keys")
			}
		}
		password, ok = passwordVal.(string)
		if !ok {
			return nil, errors.Errorf("password not a string")
		}
	} else {
		passwordVal, ok := chartSecrets["postgresql-password"]
		if !ok {
			// Chart versions <2.0 use postgres-password instead of postgresql-password.
			// See https://github.com/kubernetes-sigs/minibroker/issues/17
			passwordVal, ok = chartSecrets["postgres-password"]
			if !ok {
				return nil, errors.Errorf("password not found in secret keys")
			}
		}
		password, ok = passwordVal.(string)
		if !ok {
			return nil, errors.Errorf("password not a string")
		}
	}

	creds := Credentials{
		Protocol: postgresqlProtocolName,
		Port:     svcPort.Port,
		Host:     host,
		Username: user,
		Password: password,
		Database: database,
	}
	creds.URI = buildURI(creds)

	return &creds, nil
}
