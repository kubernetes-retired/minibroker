package minibroker

import (
	"testing"

	"k8s.io/helm/pkg/proto/hapi/chart"
	"k8s.io/helm/pkg/repo"
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
