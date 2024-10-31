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
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/ks-tool/k8s-bootstrapper/internal/config"
	"github.com/ks-tool/k8s-bootstrapper/pkg/flow"
	"github.com/ks-tool/k8s-bootstrapper/pkg/systemd"
)

var (
	caCert = filepath.Join(config.DefaultCertificatesDir, "ca.crt")
	caKey  = filepath.Join(config.DefaultCertificatesDir, "ca.key")
	saKey  = filepath.Join(config.DefaultCertificatesDir, "sa.key")

	Etcd = gen("etcd", map[string]string{
		"name":     "controller",
		"data-dir": config.DefaultEtcdDataDir,
	})
	KubeApiserver = func(cfg *config.Config) flow.Action {
		apiEndpoint := cfg.ControlPlain.LocalAPIEndpoint
		proxyClientCertFile := filepath.Join(config.DefaultCertificatesDir, "front-proxy-client.crt")
		proxyClientKeyFile := filepath.Join(config.DefaultCertificatesDir, "front-proxy-client.key")
		saIssuer := fmt.Sprintf("https://kubernetes.default.svc.%s", cfg.ControlPlain.DNSDomain)

		return gen("kube-apiserver", map[string]string{
			"advertise-address":                  apiEndpoint.AdvertiseAddress.String(),
			"allow-privileged":                   "true",
			"authorization-mode":                 "Node,RBAC",
			"client-ca-file":                     caCert,
			"enable-admission-plugins":           "NodeRestriction",
			"enable-bootstrap-token-auth":        "false",
			"etcd-servers":                       "http://127.0.0.1:2379",
			"proxy-client-cert-file":             proxyClientCertFile,
			"proxy-client-key-file":              proxyClientKeyFile,
			"requestheader-allowed-names":        "front-proxy-client",
			"requestheader-client-ca-file":       filepath.Join(config.DefaultCertificatesDir, "front-proxy-ca.crt"),
			"requestheader-extra-headers-prefix": "X-Remote-Extra-",
			"requestheader-group-headers":        "X-Remote-Group",
			"requestheader-username-headers":     "X-Remote-User",
			"secure-port":                        fmt.Sprintf("%d", apiEndpoint.BindPort),
			"service-account-issuer":             saIssuer,
			"service-account-key-file":           filepath.Join(config.DefaultCertificatesDir, "sa.pub"),
			"service-account-signing-key-file":   saKey,
			"service-cluster-ip-range":           cfg.ControlPlain.ServiceSubnet,
			"tls-cert-file":                      filepath.Join(config.DefaultCertificatesDir, "apiserver.crt"),
			"tls-private-key-file":               filepath.Join(config.DefaultCertificatesDir, "apiserver.key"),
		})
	}
	KubeControllerManager = func() flow.Action {
		kubeconfig := filepath.Join(config.DefaultKubernetesDir, "controller-manager.conf")
		return gen("kube-controller-manager", map[string]string{
			"authentication-kubeconfig":        kubeconfig,
			"authorization-kubeconfig":         kubeconfig,
			"client-ca-file":                   caCert,
			"cluster-name":                     "kubernetes",
			"cluster-signing-cert-file":        caCert,
			"cluster-signing-key-file":         caKey,
			"controllers":                      "*,bootstrapsigner,tokencleaner",
			"kubeconfig":                       kubeconfig,
			"root-ca-file":                     caCert,
			"service-account-private-key-file": saKey,
			"use-service-account-credentials":  "true",
			"bind-address":                     "127.0.0.1",
		})
	}()
	KubeScheduler = func() flow.Action {
		kubeconfig := filepath.Join(config.DefaultKubernetesDir, "scheduler.conf")
		return gen("kube-scheduler", map[string]string{
			"authentication-kubeconfig": kubeconfig,
			"authorization-kubeconfig":  kubeconfig,
			"kubeconfig":                kubeconfig,
			"bind-address":              "127.0.0.1",
		})
	}()
	Kubelet = gen("kubelet", map[string]string{
		"kubeconfig":    filepath.Join(config.DefaultKubernetesDir, "kubelet.conf"),
		"config":        filepath.Join(config.DefaultKubeletDir, "config.yaml"),
		"register-node": "true",
	})
	Coredns = gen("coredns", map[string]string{
		"config": filepath.Join(config.DefaultKubernetesDir, "Corefile"),
	})
	DaemonReload = func() flow.Action {
		action := func(ctx context.Context) (flow.StatusType, error) {
			out, err := exec.Command("systemctl", "daemon-reload").CombinedOutput()
			if err != nil {
				return flow.StatusFailed, fmt.Errorf("failed to reload systemd daemon reload: %s: %v", out, err)
			}

			return flow.StatusSuccess, nil
		}

		return flow.NewAction("daemon-reload", action)
	}()
	Enable = func(units ...string) flow.Action {
		action := func(ctx context.Context) (flow.StatusType, error) {
			for _, unit := range units {
				out, err := exec.Command("systemctl", "enable", unit).CombinedOutput()
				if err != nil {
					return flow.StatusFailed,
						fmt.Errorf("failed to enable systemd unit %q: %s: %v", unit, out, err)
				}
			}

			return flow.StatusSuccess, nil
		}

		return flow.NewAction("enable units", action)
	}
	Start = func(units ...string) flow.Action {
		action := func(ctx context.Context) (flow.StatusType, error) {
			for _, unit := range units {
				out, err := exec.Command("systemctl", "start", unit).CombinedOutput()
				if err != nil {
					return flow.StatusFailed,
						fmt.Errorf("failed to start systemd unit %q: %s: %v", unit, out, err)
				}
			}

			return flow.StatusSuccess, nil
		}

		return flow.NewAction("start units", action)
	}
)

func gen(name string, args map[string]string) flow.Action {
	path := filepath.Join(config.DefaultBinDir, name)
	action := func(ctx context.Context) (flow.StatusType, error) {
		oldSysd, err := systemd.NewSystemdUnitFromUnitFile(name)
		if err != nil && !os.IsNotExist(err) {
			return flow.StatusFailed, err
		}

		sysd := systemd.NewSystemdUnit()
		sysd.SetServiceExecStart(path, args)

		if err == nil {
			if sysd.String() == oldSysd.String() {
				return flow.StatusSkipped, nil
			}
		}

		if err = sysd.WriteToUnit(name); err != nil {
			return flow.StatusFailed, err
		}

		return flow.StatusSuccess, nil
	}

	return flow.NewAction(name, action)
}
