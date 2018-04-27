package minibroker

import (
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
)

type MariadbProvider struct{}

func (p MariadbProvider) Bind(services []corev1.Service, params map[string]interface{}, chartSecrets map[string]interface{}) (*Credentials, error) {
	service := services[0]
	if len(service.Spec.Ports) == 0 {
		return nil, errors.Errorf("no ports found")
	}
	svcPort := service.Spec.Ports[0]

	host := buildHostFromService(service)

	database := ""
	dbVal, ok := params["mariadbDatabase"]
	if ok {
		database = dbVal.(string)
	}

	var user, password string
	userVal, ok := params["mariadbUser"]
	if ok {
		user = userVal.(string)

		passwordVal, ok := chartSecrets["mariadb-password"]
		if !ok {
			return nil, errors.Errorf("mariadb-password not found in secret keys")
		}
		password = passwordVal.(string)
	} else {
		user = "root"

		rootPassword, ok := chartSecrets["mariadb-root-password"]
		if !ok {
			return nil, errors.Errorf("mariadb-root-password not found in secret keys")
		}
		password = rootPassword.(string)
	}

	creds := Credentials{
		Protocol: svcPort.Name,
		Port:     svcPort.Port,
		Host:     host,
		Username: user,
		Password: password,
		Database: database,
	}
	creds.URI = buildURI(creds)

	return &creds, nil
}
