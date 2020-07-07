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

package nameutil_test

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/kubernetes-sigs/minibroker/pkg/nameutil"
)

var _ = Describe("Generator", func() {
	Context("NameGenerator", func() {
		Describe("NewDefaultNameGenerator", func() {
			It("should create a new Generator", func() {
				var generator nameutil.Generator = nameutil.NewDefaultNameGenerator()
				Expect(generator).NotTo(BeNil())
			})
		})

		Describe("Generate", func() {
			fakeTimeNow := func() time.Time {
				return time.Date(2001, time.September, 9, 1, 46, 40, 0, time.UTC)
			}

			It("should fail when rand.Read fails", func() {
				generator := nameutil.NewNameGenerator(
					fakeTimeNow,
					func([]byte) (int, error) {
						return 0, fmt.Errorf("failed rand.Read")
					},
				)

				name, err := generator.Generate("a-prefix-")
				Expect(name).To(Equal(""))
				Expect(err).To(MatchError("failed to generate a new name: failed rand.Read"))
			})

			It("should create a new name", func() {
				generator := nameutil.NewNameGenerator(
					fakeTimeNow,
					func(data []byte) (int, error) {
						for i, _ := range data {
							data[i] = byte(i + 1)
						}
						return len(data), nil
					},
				)

				name, err := generator.Generate("a-prefix-")
				Expect(name).To(Equal("a-prefix-000064a7b3b6e00d0102"))
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})
})
