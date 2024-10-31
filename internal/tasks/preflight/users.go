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

package preflight

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/user"
	"strconv"
	"strings"

	"github.com/ks-tool/k8s-bootstrapper/internal/config"
	"github.com/ks-tool/k8s-bootstrapper/pkg/flow"
)

var (
	GroupKubernetes = actionUserGroup(
		"groupadd kubernetes",
		Group{Name: config.DefaultGroupname})
	UserKubernetes = actionUserGroup(
		"useradd kubernetes",
		User{Name: config.DefaultUsername, Group: config.DefaultGroupname})
	UserKubelet = actionUserGroup(
		"useradd kubelet",
		User{Name: "kubelet", Group: config.DefaultGroupname})
	UserEtcd = actionUserGroup(
		"useradd etcd",
		User{Name: "etcd", Group: config.DefaultGroupname, HomeDir: config.DefaultEtcdHomeDir, CreateHomeDir: true})
	UserKubeApiserver = actionUserGroup(
		"useradd kube-apiserver",
		User{Name: "kube-apiserver", Group: config.DefaultGroupname})
	UserKubeControllerManager = actionUserGroup(
		"useradd kube-controller-manager",
		User{Name: "kube-controller-manager", Group: config.DefaultGroupname})
	UserCoredns = actionUserGroup(
		"useradd coredns",
		User{Name: "coredns", Group: config.DefaultGroupname})
)

const (
	shadow = "/etc/shadow"
	users  = "/etc/passwd"
	groups = "/etc/group"

	sep = ":"
)

func actionUserGroup(name string, m UserGroupManager) flow.Action { return flow.NewAction(name, m.Add) }

type UserGroupManager interface {
	Add(context.Context) (flow.StatusType, error)
}

type User struct {
	Name          string
	Group         string
	Comment       string
	HomeDir       string
	Shell         string
	CreateHomeDir bool
}

type Group struct {
	Name string
}

func (u User) Validate() error {
	if len(u.Name) == 0 {
		return errors.New("empty username")
	}
	if strings.Contains(u.Name, sep) || strings.Contains(u.Name, "/") {
		return errors.New("invalid username")
	}
	if strings.Contains(u.Comment, sep) {
		return errors.New("invalid comment")
	}
	if strings.Contains(u.HomeDir, sep) {
		return errors.New("invalid homedir")
	}
	if strings.Contains(u.Shell, sep) {
		return errors.New("invalid shell")
	}

	return nil
}

func (u User) Exist() (bool, error) {
	var e user.UnknownUserError
	if _, err := user.Lookup(u.Name); err != nil && !errors.As(err, &e) {
		return false, fmt.Errorf("lookup user failed: %v", err)
	} else if err == nil {
		return true, nil
	}

	return false, nil
}

func (u User) Add(ctx context.Context) (flow.StatusType, error) {
	log := ctx.Value(flow.LogKey).(*flow.Logger)
	log.Infof("creating user %s", u.Name)

	if err := u.Validate(); err != nil {
		return flow.StatusFailed, fmt.Errorf("invalid user definition: %v", err)
	}

	if ok, err := u.Exist(); err != nil {
		return flow.StatusFailed, err
	} else if ok {
		return flow.StatusSkipped, nil
	}

	grp, err := user.LookupGroup(u.Group)
	if err != nil {
		return flow.StatusFailed, fmt.Errorf("lookup group failed: %v", err)
	}

	id, err := getLastId(users)
	if err != nil {
		return flow.StatusFailed, fmt.Errorf("get last uid failed: %v", err)
	}

	if len(u.HomeDir) == 0 {
		u.HomeDir = "/home/" + u.Name
	}

	usr := []string{
		u.Name,
		"x",
		strconv.Itoa(id),
		grp.Gid,
		"",
		u.HomeDir,
		u.Shell,
	}

	newUser := []byte(strings.Join(usr, sep))
	if err = appendToFile(users, newUser); err != nil {
		return flow.StatusFailed, fmt.Errorf("write user failed: %v", err)
	}

	newUserShadow := []byte(u.Name + ":*:20012:0:99999:7:::")
	if err = appendToFile(shadow, newUserShadow); err != nil {
		return flow.StatusFailed, fmt.Errorf("write shadow failed: %v", err)
	}

	if u.CreateHomeDir {
		homeDir := Dir{
			Path:  u.HomeDir,
			Perm:  0700,
			Owner: u.Name,
			Group: u.Group,
		}

		return homeDir.MkdirAll(ctx)
	}

	return flow.StatusSuccess, nil
}

func (g Group) Validate() error {
	if len(g.Name) == 0 {
		return errors.New("empty groupname")
	}
	if strings.Contains(g.Name, sep) || strings.Contains(g.Name, "/") {
		return errors.New("invalid groupname")
	}

	return nil
}

func (g Group) Exist() (bool, error) {
	var e user.UnknownGroupError
	if _, err := user.LookupGroup(g.Name); err != nil && !errors.As(err, &e) {
		return false, fmt.Errorf("lookup group failed: %v", err)
	} else if err == nil {
		return true, nil
	}

	return false, nil
}

func (g Group) Add(ctx context.Context) (flow.StatusType, error) {
	log := ctx.Value(flow.LogKey).(*flow.Logger)
	log.Infof("creating group %s", g.Name)

	if err := g.Validate(); err != nil {
		return flow.StatusFailed, fmt.Errorf("invalid user definition: %v", err)
	}

	if ok, err := g.Exist(); err != nil {
		return flow.StatusFailed, err
	} else if ok {
		return flow.StatusSkipped, nil
	}

	id, err := getLastId(groups)
	if err != nil {
		return flow.StatusFailed, fmt.Errorf("get last gid failed: %v", err)
	}

	grp := []string{
		g.Name,
		"x",
		strconv.Itoa(id),
	}
	newGroup := []byte(strings.Join(grp, sep))
	if err = appendToFile(groups, newGroup); err != nil {
		return flow.StatusFailed, fmt.Errorf("write group failed: %v", err)
	}

	return flow.StatusSuccess, nil
}

func appendToFile(path string, data []byte) error {
	fs, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("cannot stat file: %v", err)
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, fs.Mode())
	if err != nil {
		return fmt.Errorf("cannot open file: %v", err)
	}
	defer func() {
		if e := f.Close(); e != nil {
			err = fmt.Errorf("close file filed: %v", e)
		}
	}()

	if _, err = f.Write(data); err != nil {
		return fmt.Errorf("cannot write to file: %v", err)
	}

	return err
}

func getLastId(filename string) (int, error) {
	fi, err := os.Open(filename)
	if err != nil {
		return 0, err
	}
	defer func() { _ = fi.Close() }()

	lastId := 1000
	scanner := bufio.NewScanner(fi)
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) == 0 || line[0] == '#' ||
			strings.HasPrefix(line, "nobody:") ||
			strings.HasPrefix(line, "nogroups:") {
			continue
		}

		parts := strings.SplitN(line, sep, 3)
		currId, err := strconv.ParseInt(parts[2], 10, 64)
		if err != nil {
			return 0, err
		}
		if lastId < int(currId) {
			lastId = int(currId)
		}
	}

	if err = scanner.Err(); err != nil {
		return 0, err
	}

	return lastId, nil
}
