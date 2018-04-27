package minibroker

import (
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
)

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
		Protocol: svcPort.Name,
		Port:     svcPort.Port,
		Host:     host,
		Password: password,
	}
	creds.URI = buildURI(creds)

	return &creds, nil
}
