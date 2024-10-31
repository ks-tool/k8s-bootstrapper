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

package kubeconfig

import (
	"context"
	"crypto/x509"
	"fmt"
	"os"

	"github.com/ks-tool/k8s-bootstrapper/internal/config"
	"github.com/ks-tool/k8s-bootstrapper/pkg/flow"
	"github.com/ks-tool/k8s-bootstrapper/pkg/kubeconfig"
	"github.com/ks-tool/k8s-bootstrapper/pkg/pki"
)

var (
	Admin = func(serverUrl string) flow.Action {
		return gen(kubeconfig.KubeConfigSpec{
			Name: "admin",
			Auth: &pki.CertRequest{
				Organization: []string{config.ClusterAdminsGroupAndClusterRoleBinding},
				Usages:       []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
			},
			ServerURL: serverUrl,
		})
	}
	SuperAdmin = func(serverUrl string) flow.Action {
		return gen(kubeconfig.KubeConfigSpec{
			Name: "super-admin",
			Auth: &pki.CertRequest{
				Organization: []string{config.SystemPrivilegedGroup},
				Usages:       []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
			},
			ServerURL: serverUrl,
		})
	}
	Kubelet = func(serverUrl, nodeName string) flow.Action {
		return gen(kubeconfig.KubeConfigSpec{
			Name: "kubelet",
			Auth: &pki.CertRequest{
				CommonName:   config.NodesUserPrefix + nodeName,
				Organization: []string{config.NodesGroup},
				Usages:       []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
			},
			ServerURL: serverUrl,
		})
	}
	ControllerManager = func(serverUrl string) flow.Action {
		return gen(kubeconfig.KubeConfigSpec{
			Name: "controller-manager",
			Auth: &pki.CertRequest{
				CommonName: config.ControllerManagerUser,
				Usages:     []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
			},
			ServerURL: serverUrl,
		})
	}
	Scheduler = func(serverUrl string) flow.Action {
		return gen(kubeconfig.KubeConfigSpec{
			Name: "scheduler",
			Auth: &pki.CertRequest{
				CommonName: config.SchedulerUser,
				Usages:     []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
			},
			ServerURL: serverUrl,
		})
	}
	Coredns = func(serverUrl string) flow.Action {
		return gen(kubeconfig.KubeConfigSpec{
			Name: "coredns",
			Auth: &pki.CertRequest{
				CommonName: config.CorednsUser,
				Usages:     []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
			},
			ServerURL: serverUrl,
		})
	}
)

func gen(spec kubeconfig.KubeConfigSpec) flow.Action {
	spec.OutputDir = config.DefaultKubernetesDir
	spec.Auth.CAName = config.DefaultCAName
	spec.Auth.CommonName = fmt.Sprintf("kubernetes-%s", spec.Name)
	spec.Auth.PkiDir = config.DefaultCertificatesDir

	action := func(ctx context.Context) (flow.StatusType, error) {
		outputFile := spec.Filepath()
		log := ctx.Value(flow.LogKey).(*flow.Logger)
		log.Infof("kubeconfig: creating file %s", outputFile)

		if _, err := os.Stat(outputFile); err != nil && !os.IsNotExist(err) {
			log.Errorf("kubeconfig: stat file failed: %v", err)
			return flow.StatusFailed, err
		} else if err == nil {
			return flow.StatusSkipped, nil
		}

		if err := spec.Create(); err != nil {
			return flow.StatusSuccess, err
		}

		return flow.StatusSuccess, nil
	}

	return flow.NewAction(spec.Name, action)
}
