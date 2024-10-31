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
	"time"

	log "github.com/sirupsen/logrus"
)

type plainFormatter struct{}

func (f *plainFormatter) Format(entry *log.Entry) ([]byte, error) {
	sep := ": "
	if entry.Level != log.InfoLevel {
		sep += entry.Level.String() + sep
	}

	msg := []byte(entry.Time.Format(time.TimeOnly) + sep + entry.Message)

	return append(msg, '\n'), nil
}

type Logger struct{ *log.Logger }

func (l *Logger) Plain(msg string) {
	_, _ = l.Out.Write(append([]byte(msg), '\n'))
}
