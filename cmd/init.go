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

package cmd

import (
	"os"

	"github.com/ks-tool/k8s-bootstrapper/internal/tasks/download"
	"github.com/ks-tool/k8s-bootstrapper/internal/tasks/kubeconfig"
	"github.com/ks-tool/k8s-bootstrapper/internal/tasks/pki"
	"github.com/ks-tool/k8s-bootstrapper/internal/tasks/preflight"
	"github.com/ks-tool/k8s-bootstrapper/internal/tasks/systemd"
	"github.com/ks-tool/k8s-bootstrapper/pkg/flow"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Run this command in order to set up the Kubernetes control plane",
	Run: func(cmd *cobra.Command, args []string) {
		logger := logrus.New()
		cfg, err := readConfig(cmd)
		if err != nil {
			logger.Fatal(err)
		}

		initFlow := flow.New()

		preflightTask := flow.NewTask("preflight")
		preflightTask.AddAction(preflight.GroupKubernetes)
		preflightTask.AddAction(preflight.UserKubernetes)
		preflightTask.AddAction(preflight.UserEtcd)
		preflightTask.AddAction(preflight.UserKubeApiserver)
		preflightTask.AddAction(preflight.UserKubeControllerManager)
		preflightTask.AddAction(preflight.UserCoredns)
		preflightTask.AddAction(preflight.DirectoryKubernetes)
		preflightTask.AddAction(preflight.DirectoryKubernetesPKI)
		preflightTask.AddAction(preflight.DirectoryAssets(cfg))
		initFlow.AddTask(preflightTask)

		downloadTask := flow.NewTask("download")
		urlPfx := newProxyUrl(cfg)
		downloadTask.AddAction(download.Etcd(urlPfx.etcd()))
		downloadTask.AddAction(download.KubeApiserver(urlPfx.kube("kube-apiserver")))
		downloadTask.AddAction(download.KubeControllerManager(urlPfx.kube("kube-controller-manager")))
		downloadTask.AddAction(download.KubeScheduler(urlPfx.kube("kube-scheduler")))
		downloadTask.AddAction(download.Coredns(urlPfx.coredns()))
		initFlow.AddTask(downloadTask)

		pkiTask := flow.NewTask("pki")
		pkiTask.AddAction(pki.CA)
		pkiTask.AddAction(pki.KubeApiserver(cfg))
		pkiTask.AddAction(pki.FrontProxyCA)
		pkiTask.AddAction(pki.FrontProxyClient)
		pkiTask.AddAction(pki.SA)
		initFlow.AddTask(pkiTask)

		kubeconfigTask := flow.NewTask("kubeconfig")
		serverUrl := cfg.ControlPlain.LocalAPIEndpoint.URL()
		kubeconfigTask.AddAction(kubeconfig.Admin(serverUrl))
		kubeconfigTask.AddAction(kubeconfig.SuperAdmin(serverUrl))
		kubeconfigTask.AddAction(kubeconfig.ControllerManager(serverUrl))
		kubeconfigTask.AddAction(kubeconfig.Scheduler(serverUrl))
		kubeconfigTask.AddAction(kubeconfig.Coredns(serverUrl))
		initFlow.AddTask(kubeconfigTask)

		systemdTask := flow.NewTask("systemd")
		etcd := systemd.Etcd
		systemdTask.AddAction(etcd)
		apiserver := systemd.KubeApiserver(cfg)
		systemdTask.AddAction(apiserver)
		manager := systemd.KubeControllerManager
		systemdTask.AddAction(manager)
		scheduler := systemd.KubeScheduler
		systemdTask.AddAction(scheduler)
		coredns := systemd.Coredns
		systemdTask.AddAction(coredns)
		systemdTask.AddAction(systemd.DaemonReload)
		units := []string{etcd.Name, apiserver.Name, manager.Name, scheduler.Name, coredns.Name}
		systemdTask.AddAction(systemd.Enable(units...))
		systemdTask.AddAction(systemd.Start(units...))
		initFlow.AddTask(systemdTask)

		if err := initFlow.Run(cmd.Context()); err != nil {
			cmd.PrintErr(err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
