package minibroker

import (
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
)

type MySQLProvider struct{}

func (p MySQLProvider) Bind(service corev1.Service, params map[string]interface{}, chartSecrets map[string]interface{}) (*Credentials, error) {
	if len(service.Spec.Ports) == 0 {
		return nil, errors.Errorf("no ports found")
	}
	svcPort := service.Spec.Ports[0]

	rootPwd, ok := chartSecrets["mysql-root-password"]
	if !ok {
		return nil, errors.Errorf("mysql-root-password not found in secret keys")
	}

	db, ok := params["mysqlDatabase"]
	if !ok {
		// The database name may not be populated, it's okay to return an empty string
		db = ""
	}

	creds := Credentials{
		Protocol: svcPort.Name,
		Port:     svcPort.Port,
		Host:     buildHostFromService(service),
		Username: "root",
		Password: rootPwd.(string),
		Database: db.(string),
	}

	return &creds, nil
}
