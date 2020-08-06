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
	"strings"

	corev1 "k8s.io/api/core/v1"
)

// Provider is the interface for the Service Provider. Its methods wrap service-specific logic.
type Provider interface {
	Bind(
		service []corev1.Service,
		bindParams BindParams,
		provisionParams ProvisionParams,
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
	keyParts := strings.Split(key, ".")
	var part interface{} = o
	var ok bool
	for _, keyPart := range keyParts {
		switch p := part.(type) {
		case map[string]interface{}:
			part, ok = p[keyPart]
			if !ok {
				return nil, false
			}
		case Object:
			part, ok = p[keyPart]
			if !ok {
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

// DigStringOr wraps Object.DigString and returns defaultValue if any error is returned from
// Object.DigString.
func (o Object) DigStringOr(key string, defaultValue string) string {
	str, err := o.DigString(key)
	if err != nil {
		return defaultValue
	}
	return str
}

// BindParams is a new type that wraps Object.
type BindParams Object

// Dig wraps Object.Dig.
func (bp BindParams) Dig(key string) (interface{}, bool) {
	return Object(bp).Dig(key)
}

// DigString wraps Object.DigString.
func (bp BindParams) DigString(key string) (string, error) {
	return Object(bp).DigString(key)
}

// DigStringOr wraps Object.DigStringOr.
func (bp BindParams) DigStringOr(key string, defaultValue string) string {
	return Object(bp).DigStringOr(key, defaultValue)
}

// ProvisionParams is a new type that wraps Object.
type ProvisionParams Object

// Dig wraps Object.Dig.
func (pp ProvisionParams) Dig(key string) (interface{}, bool) {
	return Object(pp).Dig(key)
}

// DigString wraps Object.DigString.
func (pp ProvisionParams) DigString(key string) (string, error) {
	return Object(pp).DigString(key)
}

// DigStringOr wraps Object.DigStringOr.
func (pp ProvisionParams) DigStringOr(key string, defaultValue string) string {
	return Object(pp).DigStringOr(key, defaultValue)
}

func buildHostFromService(service corev1.Service) string {
	return fmt.Sprintf("%s.%s.svc", service.Name, service.Namespace)
}
