/*
Copyright © 2024 Alexey Shulutkov <github@shulutkov.ru>

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

package flow

import "sync/atomic"

type StatusType int

const (
	StatusSkipped StatusType = iota - 1
	StatusPending
	StatusRunning
	StatusSuccess
	StatusFailed
	StatusHasFailed
	StatusUnknown
)

func (t StatusType) String() string {
	switch t {
	case StatusSkipped:
		return "skipped"
	case StatusPending:
		return "pending"
	case StatusRunning:
		return "running"
	case StatusSuccess:
		return "success"
	case StatusFailed:
		return "failed"
	case StatusHasFailed:
		return "has_failed"
	default:
		return "unknown"
	}
}

type status struct {
	v *atomic.Pointer[StatusType]
}

func newStatus() *status {
	return &status{v: new(atomic.Pointer[StatusType])}
}

func (s *status) set(t StatusType) {
	s.v.Store(&t)
}

func (s *status) get() StatusType {
	st := s.v.Load()
	if st == nil {
		return StatusPending
	}
	return *st
}
