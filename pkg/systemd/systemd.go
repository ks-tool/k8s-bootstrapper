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

package systemd

import (
	"bytes"
	"fmt"
	"os"
	"slices"
	"strings"

	"gopkg.in/ini.v1"
)

const (
	template = `[Unit]
Description=
Wants=network-online.target
After=network-online.target

[Service]
ExecStart=
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
`
	unitFilePath          = "/etc/systemd/system/"
	serviceUnitFileFormat = unitFilePath + "%s.service"
)

type SystemdUnit struct {
	Unit    UnitSection
	Service ServiceSection
	Install InstallSection
}

type UnitSection struct {
	Description   string
	Documentation string `ini:",omitempty"`
	After         string `ini:",omitempty"`
	Requires      string `ini:",omitempty"`
	Wants         string `ini:",omitempty"`
}

type ServiceSection struct {
	User            string `ini:",omitempty"`
	Type            string `ini:",omitempty"`
	Environment     string `ini:",omitempty"`
	EnvironmentFile string `ini:",omitempty"`
	ExecStart       string
	Restart         string `ini:",omitempty"`
	RestartSec      int    `ini:",omitempty"`
}

type InstallSection struct {
	WantedBy string
}

func init() {
	ini.PrettyFormat = false
}

func (u *SystemdUnit) SetServiceExecStart(path string, args map[string]string) {
	argv := make([]string, len(args)+1)
	argv[0] = path

	var i int
	tmp := make([]string, len(args))
	for arg := range args {
		tmp[i] = arg
		i++
	}

	slices.Sort(tmp)

	i = 1
	for _, arg := range tmp {
		argv[i] = fmt.Sprintf("--%s=%s", arg, args[arg])
		i++
	}

	u.Service.ExecStart = strings.Join(argv, " ")
}

func (u *SystemdUnit) String() string {
	iniData := ini.Empty()
	if err := iniData.ReflectFrom(u); err != nil {
		panic(err)
	}

	buf := new(bytes.Buffer)
	_, _ = iniData.WriteTo(buf)

	return buf.String()
}

func (u *SystemdUnit) WriteToUnit(serviceName string) error {
	if len(u.Service.ExecStart) == 0 {
		return fmt.Errorf("systemd-unit does not have ExecStart command")
	}

	iniData := ini.Empty()
	if err := iniData.ReflectFrom(u); err != nil {
		panic(err)
	}

	buf := new(bytes.Buffer)
	if _, err := iniData.WriteTo(buf); err != nil {
		return err
	}

	unitPath := fmt.Sprintf(serviceUnitFileFormat, serviceName)
	return os.WriteFile(unitPath, buf.Bytes(), 0644)
}

func NewSystemdUnit() *SystemdUnit {
	unit := new(SystemdUnit)
	_ = ini.MapTo(unit, []byte(template))

	return unit
}

func NewSystemdUnitFromUnitFile(name string) (*SystemdUnit, error) {
	unit := new(SystemdUnit)
	iniData, err := ini.Load(fmt.Sprintf(serviceUnitFileFormat, name))
	if err != nil {
		return nil, err
	}
	if err = ini.MapTo(unit, iniData); err != nil {
		return nil, err
	}

	return unit, nil
}
