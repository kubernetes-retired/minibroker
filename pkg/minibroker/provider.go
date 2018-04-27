package minibroker

import (
	"encoding/json"
	"fmt"

	corev1 "k8s.io/api/core/v1"
)

type Provider interface {
	Bind(service []corev1.Service, params map[string]interface{}, chartSecrets map[string]interface{}) (*Credentials, error)
}

type Credentials struct {
	Protocol string
	URI      string `json:"uri,omitempty"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	Host     string `json:"host,omitempty"`
	Port     int32  `json:"port,omitempty"`
	Database string `json:"database,omitempty"`
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
	var result map[string]interface{}
	j, _ := json.Marshal(c)
	json.Unmarshal(j, &result)
	return result
}

func buildURI(c Credentials) string {
	if c.Database == "" {
		return fmt.Sprintf("%s://%s:%s@%s:%d",
			c.Protocol, c.Username, c.Password, c.Host, c.Port)
	}

	return fmt.Sprintf("%s://%s:%s@%s:%d/%s",
		c.Protocol, c.Username, c.Password, c.Host, c.Port, c.Database)
}

func buildHostFromService(service corev1.Service) string {
	return fmt.Sprintf("%s.%s.svc.cluster.local", service.Name, service.Namespace)
}
