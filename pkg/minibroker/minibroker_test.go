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
	"testing"

	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/repo"
)

func TestHasTag(t *testing.T) {
	tagTests := []struct {
		tag      string
		list     []string
		expected bool
	}{
		{"foo", []string{"foo", "bar"}, true},
		{"foo", []string{"bar", "baz"}, false},
		{"foo", []string{}, false},
	}

	for _, tt := range tagTests {
		actual := hasTag(tt.tag, tt.list)
		if actual != tt.expected {
			t.Errorf("hasTag(%s %v): expected %t, actual %t",
				tt.tag, tt.list, tt.expected, actual)
		}
	}
}

func TestGetTagIntersection(t *testing.T) {
	intersectionTests := []struct {
		charts   repo.ChartVersions
		expected []string
	}{
		{nil, []string{}},
		{
			repo.ChartVersions{
				&repo.ChartVersion{
					Metadata: &chart.Metadata{
						Keywords: []string{},
					},
				},
			},
			[]string{},
		},
		{
			repo.ChartVersions{
				&repo.ChartVersion{
					Metadata: &chart.Metadata{
						Keywords: []string{"foo", "bar"},
					},
				},
			},
			[]string{"foo", "bar"}},
		{
			repo.ChartVersions{
				&repo.ChartVersion{
					Metadata: &chart.Metadata{
						Keywords: []string{"foo", "bar"},
					},
				},
				&repo.ChartVersion{
					Metadata: &chart.Metadata{
						Keywords: []string{"baz", "foo"},
					},
				},
			},
			[]string{"foo"}},
	}

	for _, tt := range intersectionTests {
		actual := getTagIntersection(tt.charts)

		if len(actual) != len(tt.expected) {
			t.Errorf("getTagIntersection(%v): expected %v, actual %v",
				tt.charts, tt.expected, actual)

			break
		}

		for index, keyword := range actual {
			if keyword != tt.expected[index] {
				t.Errorf("getTagIntersection(%v): expected %v, actual %v",
					tt.charts, tt.expected, actual)
			}
		}
	}
}
