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

package log

import klog "k8s.io/klog/v2"

const klogMaxLevel = 5

type Klog struct {
	levels []*klogLogger
}

func NewKlog() Verboser {
	levels := make([]*klogLogger, 0, klogMaxLevel+1)
	for level := 0; level <= 5; level++ {
		v := klog.V(klog.Level(level))
		levels = append(levels, &klogLogger{v})
	}
	return &Klog{levels}
}

func (l *Klog) V(level Level) Logger {
	if level > klogMaxLevel {
		return l.levels[klogMaxLevel]
	}
	return l.levels[level]
}

type klogLogger struct {
	v klog.Verbose
}

func (l *klogLogger) Get() Logger {
	if !l.v.Enabled() {
		return nil
	}
	return l
}

func (l *klogLogger) Log(format string, args ...interface{}) {
	l.v.Infof(format, args...)
}
