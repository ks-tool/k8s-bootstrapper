/*
 Copyright (c) 2024 Alexey Shulutkov <github@shulutkov.ru>

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

package download

import (
	"archive/tar"
	"context"
	"path"
	"path/filepath"

	"github.com/ks-tool/k8s-bootstrapper/internal/config"
	"github.com/ks-tool/k8s-bootstrapper/pkg/fetch"
	"github.com/ks-tool/k8s-bootstrapper/pkg/flow"
)

var (
	Etcd = func(url string) flow.Action {
		return download("etcd", url,
			fetch.UnTar(config.DefaultBinDir, etcdFilter),
		)
	}
	KubeApiserver = func(url string) flow.Action {
		name := "kube-apiserver"
		return download(name, url, toFile(name))
	}
	KubeControllerManager = func(url string) flow.Action {
		name := "kube-controller-manager"
		return download(name, url, toFile(name))
	}
	KubeScheduler = func(url string) flow.Action {
		name := "kube-scheduler"
		return download(name, url, toFile(name))
	}
	Kubelet = func(url string) flow.Action {
		name := "kubelet"
		return download(name, url, toFile(name))
	}
	Coredns = func(url string) flow.Action {
		name := "coredns"
		return download(name, url,
			fetch.UnTar(config.DefaultBinDir),
		)
	}
)

func download(name, url string, writer fetch.Writer) flow.Action {
	action := func(ctx context.Context) (flow.StatusType, error) {
		if err := fetch.WithContext(ctx, writer, url); err != nil {
			return flow.StatusFailed, err
		}

		return flow.StatusSuccess, nil
	}

	return flow.NewAction(name, action)
}

func toFile(name string) fetch.Writer {
	return fetch.ToFile(filepath.Join(config.DefaultBinDir, name), 0755)
}

func etcdFilter(dst string, tr *tar.Reader, hdr *tar.Header) error {
	if hdr.Typeflag != tar.TypeReg {
		return nil
	}

	bn := path.Base(hdr.Name)
	if bn == "etcd" || bn == "etcdctl" {
		return fetch.ToFile(filepath.Join(dst, bn), 0755)(tr)
	}

	return nil
}
