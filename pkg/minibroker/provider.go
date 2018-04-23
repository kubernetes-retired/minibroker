package minibroker

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
)

type Provider interface {
	Bind(service corev1.Service, params map[string]interface{}, chartSecrets map[string]interface{}) (*Credentials, error)
}

type Credentials struct {
	Protocol string
	Username string
	Password string
	Host     string
	Port     int32
	Database string
}

// ToMap converts the credentials into the OSB API credentials response
// see https://github.com/openservicebrokerapi/servicebroker/blob/master/spec.md#device-object
// {
//   "credentials": {
//     "uri": "mysql://mysqluser:pass@mysqlhost:3306/dbname",
//     "username": "mysqluser",
//     "password": "pass",
//     "host": "mysqlhost",
//     "port": 3306,
//     "database": "dbname"
//     }
// }
func (c Credentials) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"uri":      c.URI(),
		"username": c.Username,
		"password": c.Password,
		"host":     c.Host,
		"port":     c.Port,
		"database": c.Database,
	}
}

func (c Credentials) URI() string {
	return fmt.Sprintf("%s://%s:%s@%s:%d/%s",
		c.Protocol, c.Username, c.Password, c.Host, c.Port, c.Database)
}

func buildHostFromService(service corev1.Service) string {
	return fmt.Sprintf("%s.%s.svc.cluster.local", service.Name, service.Namespace)
}
