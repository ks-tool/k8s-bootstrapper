/*
 Copyright (c) 2024 Alexey Shulutkov <github@shulutkov.ru>

 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this File except in compliance with the License.
 You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
*/

package fileproxy

import (
	"sync"
	"time"
)

type Semaphore struct {
	mu *sync.RWMutex
	m  map[string]struct{}
}

func NewSemaphore() *Semaphore {
	return &Semaphore{
		mu: &sync.RWMutex{},
		m:  make(map[string]struct{}),
	}
}

func (s *Semaphore) Acquire(key string) {
	if _, ok := s.m[key]; !ok {
		s.add(key)
		return
	}

	for {
		if ok := s.in(key); !ok {
			return
		}
		time.Sleep(500 * time.Millisecond)
	}
}

func (s *Semaphore) Release(key string) {
	s.mu.Lock()
	delete(s.m, key)
	s.mu.Unlock()
}

func (s *Semaphore) add(key string) {
	s.mu.Lock()
	s.m[key] = struct{}{}
	s.mu.Unlock()
}

func (s *Semaphore) in(key string) bool {
	s.mu.RLock()
	_, ok := s.m[key]
	s.mu.RUnlock()
	return ok
}
