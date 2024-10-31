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

import (
	"context"
	"fmt"
)

type TaskErrors interface {
	GetActionStatus(name string) (StatusType, error)
	Successful() bool
}

type Task struct {
	status      *status
	exitOnError bool

	name string
	act  []*Action
}

func NewTask(name string) Task {
	return Task{
		status:      newStatus(),
		exitOnError: true,
		name:        name,
		act:         make([]*Action, 0),
	}
}

func (t *Task) NoExitOnError() {
	t.checkStatus()
	t.exitOnError = false
}

func (t *Task) AddAction(act Action) {
	t.checkStatus()

	if act.Fn == nil {
		panic(fmt.Sprintf("no action defined for %q", act.Name))
	}

	for _, a := range t.act {
		if a.Name == act.Name {
			panic(fmt.Sprintf("action %q already defined for task %q", act.Name, t.name))
		}
	}

	t.act = append(t.act, &act)
}

func (t *Task) Successful() bool {
	st := t.status.get()
	if st == StatusPending || st == StatusRunning {
		return false
	}

	for _, act := range t.act {
		if act.err != nil {
			return false
		}
	}

	return true
}

func (t *Task) GetActionStatus(name string) (StatusType, error) {
	for _, act := range t.act {
		if act.Name == name {
			return act.Status()
		}
	}

	return StatusUnknown, fmt.Errorf("action %q not found", name)
}

func (t *Task) run(ctx context.Context) error {
	t.status.set(StatusRunning)

	log := ctx.Value(LogKey).(*Logger)
	log.Plain("================= TASK: " + t.name + " =================")

	for _, act := range t.act {
		log.Infof("Run action: %s", act.Name)

		if err := act.Run(ctx); err != nil {
			log.Errorf("action %q failed: %v", act.Name, act.err)
			if t.exitOnError {
				t.status.set(StatusFailed)
				return err
			}

			t.status.set(StatusHasFailed)
		}
	}

	if t.status.get() != StatusHasFailed {
		t.status.set(StatusSuccess)
	}
	return nil
}

func (t *Task) checkStatus() {
	taskStatus := "done"
	st := t.status.get()
	switch st {
	case StatusPending:
		return
	case StatusRunning:
		taskStatus = "running"
	default:
	}

	panic(fmt.Sprintf("task %q already %s", t.name, taskStatus))
}
