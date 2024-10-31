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
	"os"

	log "github.com/sirupsen/logrus"
)

type Flow struct {
	t   []Task
	log *Logger
}

func New() *Flow {
	return &Flow{
		t: make([]Task, 0),
		log: &Logger{
			Logger: &log.Logger{
				Out:          os.Stderr,
				Formatter:    new(plainFormatter),
				Hooks:        make(log.LevelHooks),
				Level:        log.InfoLevel,
				ExitFunc:     os.Exit,
				ReportCaller: false,
			},
		},
	}
}

func (f *Flow) SetLogLevel(lvl log.Level) {
	f.log.SetLevel(lvl)
}

func (f *Flow) AddTask(t Task) *Flow {
	if len(t.act) == 0 {
		panic(fmt.Sprintf("no actions defined for Task %q", t.name))
	}

	f.t = append(f.t, t)

	return f
}

func (f *Flow) Run(ctx context.Context) error {
	cctx := &Context{ctx, f}

	for _, t := range f.t {
		if err := t.run(cctx); err != nil {
			return err
		}
	}

	return nil
}
