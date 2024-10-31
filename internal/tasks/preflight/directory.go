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
	"context"
	"errors"
	"fmt"
	"os"
	"os/user"
	"strconv"

	"github.com/ks-tool/k8s-bootstrapper/internal/config"
	"github.com/ks-tool/k8s-bootstrapper/pkg/flow"
	"github.com/ks-tool/k8s-bootstrapper/utils"

	"github.com/mitchellh/go-homedir"
)

var (
	DirectoryKubernetes = actionDir(
		"mkdir kubernetes",
		Dir{Path: config.DefaultKubernetesDir})
	DirectoryKubernetesPKI = actionDir(
		"mkdir pki",
		Dir{Path: config.DefaultCertificatesDir, Owner: config.DefaultUsername, Group: config.DefaultGroupname})
	DirectoryKubelet = actionDir(
		"mkdir kubelet",
		Dir{Path: config.DefaultKubeletDir, Owner: "kubelet", Group: config.DefaultGroupname})
	DirectoryKubeProxy = actionDir(
		"mkdir kube-proxy",
		Dir{Path: config.DefaultKubeProxyDir})
	DirectoryAssets = func(cfg *config.Config) flow.Action {
		return actionDir(
			"mkdir assets",
			Dir{Path: cfg.AssetsDir})
	}
)

func actionDir(name string, d Dir) flow.Action { return flow.NewAction(name, d.MkdirAll) }

type Dir struct {
	Path  string
	Perm  os.FileMode
	Owner string
	Group string
}

func (d Dir) MkdirAll(ctx context.Context) (flow.StatusType, error) {
	path, err := homedir.Expand(d.Path)
	if err != nil {
		return flow.StatusFailed, fmt.Errorf("failed to expand path: %v", err)
	}

	log := ctx.Value(flow.LogKey).(*flow.Logger)
	log.Infof("mkdir: creating directory %s", path)

	fs, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			if err = os.MkdirAll(path, d.Perm); err != nil {
				return flow.StatusFailed, fmt.Errorf("mkdir: failed to create directory: %v", err)
			}

			log.Info("perm: change directory permissions")
			if st, err := d.chown(); err != nil {
				return flow.StatusFailed, fmt.Errorf("failed to chown directory: %v", err)
			} else if st == flow.StatusSkipped {
				log.Info("perm: successful")
			}

			return flow.StatusSuccess, nil
		}

		return flow.StatusFailed, fmt.Errorf("mkdir: could not stat path: %v", err)
	} else if !fs.IsDir() {
		return flow.StatusFailed, errors.New("mkdir: is not a directory")
	}

	return d.chown()
}

func (d Dir) chown() (flow.StatusType, error) {
	if len(d.Owner) == 0 && len(d.Group) == 0 {
		return flow.StatusSkipped, nil
	}

	path, err := homedir.Expand(d.Path)
	if err != nil {
		return flow.StatusFailed, fmt.Errorf("failed to expand path: %v", err)
	}

	u, g, err := utils.CurrentOwner(path)
	if err != nil {
		return flow.StatusFailed, err
	}

	var usr *user.User
	if len(d.Owner) == 0 {
		usr = u
	} else {
		usr, err = utils.LookupUser(d.Owner)
		if err != nil {
			return flow.StatusFailed, err
		}
	}

	var grp *user.Group
	if len(d.Group) == 0 {
		grp = g
	} else {
		grp, err = utils.LookupGroup(d.Group)
		if err != nil {
			return flow.StatusFailed, err
		}
	}

	if u.Uid == usr.Uid && g.Gid == grp.Gid {
		return flow.StatusSkipped, nil
	}

	uid, err := strconv.Atoi(usr.Uid)
	if err != nil {
		return flow.StatusFailed, err
	}

	gid, err := strconv.Atoi(grp.Gid)
	if err != nil {
		return flow.StatusFailed, err
	}

	if err = os.Chown(path, uid, gid); err != nil {
		return flow.StatusFailed, err
	}

	return flow.StatusSuccess, nil
}
