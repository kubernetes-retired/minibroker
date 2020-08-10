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

import "testing"

func TestObjectDig(t *testing.T) {
	tests := []struct {
		obj         Object
		key         string
		expectedVal interface{}
		expectedOk  bool
	}{
		{
			Object{"foo": "baz"},
			"bar",
			nil,
			false,
		},
		{
			Object{"foo": Object{"bar": "baz"}},
			"foo.foo",
			nil,
			false,
		},
		{
			Object{"foo": Object{"bar": "baz"}},
			"foo.bar.bar",
			nil,
			false,
		},
		{
			Object{"foo": Object{"bar": "baz"}},
			"",
			nil,
			false,
		},
		{
			Object{"foo": Object{"": "baz"}},
			"foo.",
			nil,
			false,
		},
		{
			Object{"foo": Object{"bar": "baz"}},
			"foo.",
			nil,
			false,
		},
		{
			Object{"foo": Object{"bar": "baz"}},
			"foo..bar",
			nil,
			false,
		},
		{
			Object{"": Object{"bar": "baz"}},
			".bar",
			nil,
			false,
		},
		{
			Object{"foo": "baz"},
			"foo",
			"baz",
			true,
		},
		{
			Object{"foo": Object{"bar": "baz"}},
			"foo.bar",
			"baz",
			true,
		},
	}

	for _, tt := range tests {
		val, ok := tt.obj.Dig(tt.key)
		if ok != tt.expectedOk {
			t.Errorf("Object.Dig(%s): expected ok %v, actual ok %v", tt.key, tt.expectedOk, ok)
		}
		if val != tt.expectedVal {
			t.Errorf("Object.Dig(%s): expected val %v, actual val %v", tt.key, tt.expectedVal, val)
		}
	}
}

func TestObjectDigString(t *testing.T) {
	tests := []struct {
		obj         Object
		key         string
		expectedVal string
		expectedErr error
	}{
		{
			Object{"foo": "baz"},
			"bar",
			"",
			ErrDigNotFound,
		},
		{
			Object{"foo": 3},
			"foo",
			"",
			ErrDigNotString,
		},
		{
			Object{"foo": Object{"bar": "baz"}},
			"foo.bar",
			"baz",
			nil,
		},
	}

	for _, tt := range tests {
		val, err := tt.obj.DigString(tt.key)
		if err != tt.expectedErr {
			t.Errorf("Object.DigString(%s): expected err %v, actual err %v", tt.key, tt.expectedErr, err)
		}
		if val != tt.expectedVal {
			t.Errorf("Object.DigString(%s): expected val %v, actual val %v", tt.key, tt.expectedVal, val)
		}
	}
}

func TestObjectDigStringAlt(t *testing.T) {
	tests := []struct {
		obj         Object
		altKeys     []string
		expectedVal string
		expectedErr error
	}{
		{
			Object{"foo": "baz"},
			[]string{"bar", "baz"},
			"",
			ErrDigNotFound,
		},
		{
			Object{"foo": 3},
			[]string{"bar", "foo"},
			"",
			ErrDigNotString,
		},
		{
			Object{"foo": Object{"bar": "baz"}},
			[]string{"foo", "foo.bar"},
			"",
			ErrDigNotString,
		},
		{
			Object{"foo": Object{"bar": "baz"}},
			[]string{"foo.foo", "foo.bar"},
			"baz",
			nil,
		},
	}

	for _, tt := range tests {
		val, err := tt.obj.DigStringAlt(tt.altKeys)
		if err != tt.expectedErr {
			t.Errorf("Object.DigStringAlt(%v): expected err %v, actual err %v", tt.altKeys, tt.expectedErr, err)
		}
		if val != tt.expectedVal {
			t.Errorf("Object.DigStringAlt(%v): expected val %v, actual val %v", tt.altKeys, tt.expectedVal, val)
		}
	}
}

func TestObjectDigStringOr(t *testing.T) {
	tests := []struct {
		obj         Object
		key         string
		defaultVal  string
		expectedVal string
		expectedErr error
	}{
		{
			Object{"foo": 1},
			"foo",
			"default",
			"",
			ErrDigNotString,
		},
		{
			Object{"foo": "baz"},
			"bar",
			"default",
			"default",
			nil,
		},
		{
			Object{"foo": "baz"},
			"foo",
			"default",
			"baz",
			nil,
		},
	}

	for _, tt := range tests {
		val, err := tt.obj.DigStringOr(tt.key, tt.defaultVal)
		if err != tt.expectedErr {
			t.Errorf("Object.DigStringOr(%s): expected err %v, actual err %v", tt.key, tt.expectedErr, err)
		}
		if val != tt.expectedVal {
			t.Errorf("Object.DigStringOr(%s): expected val %v, actual val %v", tt.key, tt.expectedVal, val)
		}
	}
}

func TestObjectDigStringAltOr(t *testing.T) {
	tests := []struct {
		obj         Object
		altKeys     []string
		defaultVal  string
		expectedVal string
		expectedErr error
	}{
		{
			Object{"foo": 1},
			[]string{"bar", "foo"},
			"default",
			"",
			ErrDigNotString,
		},
		{
			Object{"foo": "baz"},
			[]string{"bar", "baz"},
			"default",
			"default",
			nil,
		},
		{
			Object{"foo": "baz"},
			[]string{"bar", "foo"},
			"default",
			"baz",
			nil,
		},
	}

	for _, tt := range tests {
		val, err := tt.obj.DigStringAltOr(tt.altKeys, tt.defaultVal)
		if err != tt.expectedErr {
			t.Errorf("Object.DigStringAltOr(%v): expected err %v, actual err %v", tt.altKeys, tt.expectedErr, err)
		}
		if val != tt.expectedVal {
			t.Errorf("Object.DigStringAltOr(%v): expected val %v, actual val %v", tt.altKeys, tt.expectedVal, val)
		}
	}
}
