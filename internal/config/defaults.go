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

package config

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
)

const (
	// DefaultServiceDNSDomain defines default cluster-internal domain name for Services and Pods
	DefaultServiceDNSDomain = "cluster.local"
	// DefaultServicesSubnet defines default service subnet range
	DefaultServicesSubnet = "172.18.0.0/21"
	// DefaultPodSubnet defines default pod subnet range
	DefaultPodSubnet = "172.21.0.0/18"
	// DefaultKubernetesLastStableVersionURL defines the URL of the latest stable version of Kubernetes
	DefaultKubernetesLastStableVersionURL = "https://dl.k8s.io/release/stable-1.txt"
	// DefaultKubernetesDir is the directory Kubernetes owns for storing various configuration files
	DefaultKubernetesDir = "/etc/kubernetes"
	// DefaultCertificatesDir defines default certificate directory
	DefaultCertificatesDir = DefaultKubernetesDir + "/pki"
	// DefaultImageRepository defines default image registry
	DefaultImageRepository = "registry.k8s.io"
	// DefaultEtcdHomeDir defines default location etcd homedir
	DefaultEtcdHomeDir = "/var/lib/etcd"
	// DefaultEtcdDataDir defines default location of etcd where will save data to
	DefaultEtcdDataDir = DefaultEtcdHomeDir + "/data"
	// DefaultEtcdVersion defines default etcd version
	DefaultEtcdVersion = "v3.5.16"
	// DefaultAssetsDir defines default location of downloaded files
	DefaultAssetsDir = "~/kubernetes"
	// DefaultKubeAPIServerPort is the default port for the apiserver.
	DefaultKubeAPIServerPort = 6443
	// DefaultKubeletDir specifies the directory where the kubelet runtime information is stored.
	DefaultKubeletDir = "/var/lib/kubelet"
	// DefaultKubeProxyDir specifies the directory where the kube-proxy runtime information is stored.
	DefaultKubeProxyDir = "/var/lib/kube-proxy"
	// DefaultBinDir defines default location of binary files
	DefaultBinDir = "/usr/local/bin"

	DefaultCorednsVersion   = "v1.11.3"
	DefaultAssetsServerPort = 18080

	DefaultUsername  = "kubernetes"
	DefaultGroupname = "kubernetes"
	DefaultCAName    = "ca"
)

const (
	// LabelNodeRoleControlPlane specifies that a node hosts control-plane components
	LabelNodeRoleControlPlane = "node-role.kubernetes.io/control-plane"
	// ClusterAdminsGroupAndClusterRoleBinding is the name of the Group used for kubeadm generated cluster
	// admin credentials and the name of the ClusterRoleBinding that binds the same Group to the "cluster-admin"
	// built-in ClusterRole.
	ClusterAdminsGroupAndClusterRoleBinding = "ks-tool:cluster-admins"
	// ControllerManagerUser defines the well-known user the controller-manager should be authenticated as
	ControllerManagerUser = "system:kube-controller-manager"
	// SchedulerUser defines the well-known user the scheduler should be authenticated as
	SchedulerUser = "system:kube-scheduler"
	// CorednsUser defines the well-known user the coredns should be authenticated as
	CorednsUser = "system:coredns"
	// NodesUserPrefix defines the username prefix as requested by the Node authorizer.
	NodesUserPrefix = "system:node:"
	// SystemPrivilegedGroup defines the well-known group for the apiservers. This group is also superuser by default
	// (i.e. bound to the cluster-admin ClusterRole)
	SystemPrivilegedGroup = "system:masters"
	// NodesGroup defines the well-known group for all nodes.
	NodesGroup = "system:nodes"
	// NodeBootstrapTokenAuthGroup specifies which group a Node Bootstrap Token should be authenticated in
	NodeBootstrapTokenAuthGroup = "system:bootstrappers:kubeadm:default-node-token"
	// KubeProxyClusterRoleName sets the name for the kube-proxy ClusterRole
	KubeProxyClusterRoleName = "system:node-proxier"
	// NodeBootstrapperClusterRoleName defines the name of the auto-bootstrapped ClusterRole for letting someone post a CSR
	NodeBootstrapperClusterRoleName = "system:node-bootstrapper"
	// CSRAutoApprovalClusterRoleName defines the name of the auto-bootstrapped ClusterRole for making the csrapprover controller auto-approve the CSR
	// Starting from v1.8, CSRAutoApprovalClusterRoleName is automatically created by the API server on startup
	CSRAutoApprovalClusterRoleName = "system:certificates.k8s.io:certificatesigningrequests:nodeclient"
	// NodeSelfCSRAutoApprovalClusterRoleName is a role defined in default 1.8 RBAC policies for automatic CSR approvals for automatically rotated node certificates
	NodeSelfCSRAutoApprovalClusterRoleName = "system:certificates.k8s.io:certificatesigningrequests:selfnodeclient"
	// NodesClusterRoleBinding defines the well-known ClusterRoleBinding which binds the too permissive system:node
	// ClusterRole to the system:nodes group. Since kubeadm is using the Node Authorizer, this ClusterRoleBinding's
	// system:nodes group subject is removed if present.
	NodesClusterRoleBinding = "system:node"
)

func SetDefaults(cfg *Config) error {
	if len(cfg.ImageRepository) == 0 {
		cfg.ImageRepository = DefaultImageRepository
	}
	if len(cfg.ControlPlain.EtcdVersion) == 0 {
		cfg.ControlPlain.EtcdVersion = DefaultEtcdVersion
	}
	if len(cfg.ControlPlain.KubernetesVersion) == 0 {
		resp, err := http.Get(DefaultKubernetesLastStableVersionURL)
		if err != nil {
			return err
		}
		if resp.StatusCode != 200 {
			return fmt.Errorf("fetch last stable version of Kubernetes failed: expected 200 response code, got %d", resp.StatusCode)
		}
		b, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("fetch last stable version of Kubernetes failed: %s", err)
		}
		cfg.ControlPlain.KubernetesVersion = strings.TrimSpace(string(b))
	}
	if len(cfg.ControlPlain.CorednsVersion) == 0 {
		cfg.ControlPlain.CorednsVersion = DefaultCorednsVersion
	}

	if len(cfg.ControlPlain.PodSubnet) == 0 {
		cfg.ControlPlain.PodSubnet = DefaultPodSubnet
	}
	if len(cfg.ControlPlain.ServiceSubnet) == 0 {
		cfg.ControlPlain.ServiceSubnet = DefaultServicesSubnet
	}
	if len(cfg.ControlPlain.DNSDomain) == 0 {
		cfg.ControlPlain.DNSDomain = DefaultServiceDNSDomain
	}
	if cfg.ProxyPort == 0 {
		cfg.ProxyPort = DefaultAssetsServerPort
	}
	if len(cfg.AssetsDir) == 0 {
		cfg.AssetsDir = DefaultAssetsDir
	}

	if len(cfg.ControlPlain.LocalAPIEndpoint.AdvertiseAddress) == 0 {
		hostIPs, err := net.LookupIP("ya.ru")
		if err != nil {
			return err
		}

		conn, err := net.Dial("udp", fmt.Sprintf("%s:80", hostIPs[0]))
		if err != nil {
			return err
		}
		cfg.ControlPlain.LocalAPIEndpoint.AdvertiseAddress = conn.LocalAddr().(*net.UDPAddr).IP
	}
	if cfg.ControlPlain.LocalAPIEndpoint.BindPort < 1 {
		cfg.ControlPlain.LocalAPIEndpoint.BindPort = DefaultKubeAPIServerPort
	}

	if len(cfg.NodeName) == 0 {
		hostname, err := os.Hostname()
		if err != nil {
			return err
		}

		cfg.NodeName = strings.SplitN(hostname, ".", 2)[0]
	}

	return nil
}
