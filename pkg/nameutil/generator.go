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

package nameutil

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"time"
)

//go:generate mockgen -destination=./mocks/mock_generator.go -package=mocks github.com/kubernetes-sigs/minibroker/pkg/nameutil Generator

// Generator is the interface that wraps the basic Generate method.
type Generator interface {
	Generate(prefix string) (generated string, err error)
}

// NameGenerator satisfies the Generator interface for generating names.
type NameGenerator struct {
	timeNow  func() time.Time
	randRead func([]byte) (int, error)
}

// NewDefaultNameGenerator creates a new NameGenerator with the default dependencies.
func NewDefaultNameGenerator() *NameGenerator {
	return NewNameGenerator(time.Now, rand.Read)
}

// NewNameGenerator creates a new NameGenerator.
func NewNameGenerator(
	timeNow func() time.Time,
	randRead func([]byte) (int, error),
) *NameGenerator {
	return &NameGenerator{
		timeNow:  timeNow,
		randRead: randRead,
	}
}

// Generate generates a new name with a prefix based on the UnixNano UTC timestamp plus 2 random
// bytes.
func (ng *NameGenerator) Generate(prefix string) (string, error) {
	b := make([]byte, 10)
	binary.LittleEndian.PutUint64(b[0:], uint64(ng.timeNow().UTC().UnixNano()))
	if _, err := ng.randRead(b[8:]); err != nil {
		return "", fmt.Errorf("failed to generate a new name: %v", err)
	}
	name := fmt.Sprintf("%s%x", prefix, b)
	return name, nil
}
