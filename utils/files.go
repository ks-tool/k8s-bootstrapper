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

package utils

import (
	"fmt"
	"github.com/mitchellh/go-homedir"
	"os"
	"os/user"
	"strconv"
	"syscall"
)

func CurrentOwner(path string) (*user.User, *user.Group, error) {
	uid, gid, err := CurrentOwnerIds(path)
	if err != nil {
		return nil, nil, err
	}

	u, err := user.LookupId(strconv.Itoa(int(uid)))
	if err != nil {
		return nil, nil, err
	}

	g, err := user.LookupGroupId(strconv.Itoa(int(gid)))
	if err != nil {
		return nil, nil, err
	}

	return u, g, nil
}

func CurrentOwnerIds(path string) (uint32, uint32, error) {
	fs, err := os.Stat(path)
	if err != nil {
		return 0, 0, fmt.Errorf("cannot stat path: %q", err)
	}
	stat_t := fs.Sys().(*syscall.Stat_t)

	return stat_t.Uid, stat_t.Gid, nil
}

func Chown(path, owner, group string) error {
	if len(owner) == 0 && len(group) == 0 {
		return nil
	}

	path, err := homedir.Expand(path)
	if err != nil {
		return err
	}

	u, g, err := CurrentOwner(path)
	if err != nil {
		return err
	}

	var usr *user.User
	if len(owner) == 0 {
		usr = u
	} else {
		usr, err = LookupUser(owner)
		if err != nil {
			return err
		}
	}

	var grp *user.Group
	if len(group) == 0 {
		grp = g
	} else {
		grp, err = LookupGroup(group)
		if err != nil {
			return err
		}
	}

	if u.Uid == usr.Uid && g.Gid == grp.Gid {
		return nil
	}

	uid, err := strconv.Atoi(usr.Uid)
	if err != nil {
		return err
	}

	gid, err := strconv.Atoi(grp.Gid)
	if err != nil {
		return err
	}

	return os.Chown(path, uid, gid)
}
