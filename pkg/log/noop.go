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

func NewNoop() Verboser {
	return &noop{&noopLogger{}}
}

type noop struct {
	log *noopLogger
}

func (l *noop) V(Level) Logger {
	return l.log
}

type noopLogger struct{}

func (l *noopLogger) Get() Logger {
	return l
}

func (l *noopLogger) Log(format string, args ...interface{}) {
}
