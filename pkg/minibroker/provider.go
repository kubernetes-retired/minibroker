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
	"strings"

	corev1 "k8s.io/api/core/v1"
)

// Provider is the interface for the Service Provider. Its methods wrap service-specific logic.
type Provider interface {
	Bind(
		service []corev1.Service,
		bindParams *BindParams,
		provisionParams *ProvisionParams,
		chartSecrets Object,
	) (Object, error)
}

// Object is a wrapper around map[string]interface{} that implements methods for helping with
// digging and type asserting.
type Object map[string]interface{}

var (
	// ErrDigNotFound is the error for a key not found in the Object.
	ErrDigNotFound = fmt.Errorf("key not found")
	// ErrDigNotString is the error for a key that is not a string.
	ErrDigNotString = fmt.Errorf("key is not a string")
)

// Dig digs the Object based on the provided key.
// key must be in the format "foo.bar.baz". Each segment represents a level in the Object.
func (o Object) Dig(key string) (interface{}, bool) {
	if key == "" {
		return nil, false
	}
	keyParts := strings.Split(key, ".")
	var part interface{} = o
	var ok bool
	for _, keyPart := range keyParts {
		if keyPart == "" {
			return nil, false
		}
		switch p := part.(type) {
		case map[string]interface{}:
			if part, ok = p[keyPart]; !ok {
				return nil, false
			}
		case Object:
			if part, ok = p[keyPart]; !ok {
				return nil, false
			}
		default:
			return nil, false
		}
	}
	return part, ok
}

// DigString wraps Object.Dig and type-asserts the found key.
func (o Object) DigString(key string) (string, error) {
	val, ok := o.Dig(key)
	if !ok {
		return "", ErrDigNotFound
	}
	valStr, ok := val.(string)
	if !ok {
		return "", ErrDigNotString
	}
	return valStr, nil
}

// DigStringAlt digs for any of the given keys, returning the first found. It returns an error if
// none of the alternative keys are found.
func (o Object) DigStringAlt(altKeys []string) (string, error) {
	for _, altKey := range altKeys {
		valStr, err := o.DigString(altKey)
		if err == ErrDigNotFound {
			continue
		}
		if err != nil {
			return "", err
		}
		return valStr, nil
	}
	return "", ErrDigNotFound
}

// DigStringOr wraps Object.DigString and returns defaultValue if the value was not found.
func (o Object) DigStringOr(key string, defaultValue string) (string, error) {
	str, err := o.DigString(key)
	if err == ErrDigNotFound {
		return defaultValue, nil
	}
	if err != nil {
		return "", err
	}
	return str, nil
}

// DigStringAltOr wraps Object.DigStringAlt and returns defaultValue if none of the alternative
// keys are found.
func (o Object) DigStringAltOr(altKeys []string, defaultValue string) (string, error) {
	str, err := o.DigStringAlt(altKeys)
	if err == ErrDigNotFound {
		return defaultValue, nil
	}
	if err != nil {
		return "", err
	}
	return str, nil
}

// BindParams is a specialization of Object for binding parameters, ensuring type checking.
type BindParams struct {
	Object
}

// NewBindParams constructs a new BindParams.
func NewBindParams(m map[string]interface{}) *BindParams {
	return &BindParams{Object: m}
}

// ProvisionParams is a specialization of Object for provisioning parameters, ensuring type
// checking.
type ProvisionParams struct {
	Object
}

// NewProvisionParams constructs a new ProvisionParams.
func NewProvisionParams(m map[string]interface{}) *ProvisionParams {
	return &ProvisionParams{Object: m}
}

// hostBuilder provides the method for building the OSBAPI service's URI host
// from a k8s service.
type hostBuilder struct {
	clusterDomain string
}

// hostFromService builds the FQDN for the host using the service name and
// namespace, appending the cluster domain.
func (hb *hostBuilder) hostFromService(service *corev1.Service) string {
	return fmt.Sprintf("%s.%s.svc.%s", service.Name, service.Namespace, hb.clusterDomain)
}
