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
	"net"
)

type Config struct {
	// NodeName field of the Node API object that will be created in this "init" or "join" operation.
	// This field is also used in the CommonName field of the kubelet's client certificate to the API server.
	// Defaults to the hostname of the node if not provided.
	NodeName string `json:"nodeName,omitempty"`

	// ImageRepository sets the container registry to pull images from.
	ImageRepository string `json:"imageRepository,omitempty"`

	// ControlPlain holds configuration for Kubernetes.
	ControlPlain ControlPlainSettings `json:"controlPlain,omitempty"`

	AssetsDir string `json:"assetsDir,omitempty"`
	ProxyPort int    `json:"proxyPort,omitempty"`
}

type ControlPlainSettings struct {
	// LocalAPIEndpoint represents the endpoint of the API server instance that's deployed on this control plane node.
	LocalAPIEndpoint APIEndpoint `json:"localAPIEndpoint,omitempty"`
	// EtcdVersion is the target version of the etcd.
	EtcdVersion string `json:"etcdVersion,omitempty"`
	// CorednsVersion is the target version of the coredns.
	CorednsVersion string `json:"corednsVersion,omitempty"`
	// KubernetesVersion is the target version of the control plane.
	KubernetesVersion string `json:"kubernetesVersion,omitempty"`
	// ServiceSubnet is the subnet used by k8s services. Defaults to "172.18.0.0/21".
	ServiceSubnet string `json:"serviceSubnet,omitempty"`
	// PodSubnet is the subnet used by pods. Defaults to "172.21.0.0/18".
	PodSubnet string `json:"podSubnet,omitempty"`
	// DNSDomain is the dns domain used by k8s services. Defaults to "cluster.local".
	DNSDomain string `json:"dnsDomain,omitempty"`
}

// APIEndpoint struct contains elements of API server instance deployed on a node.
type APIEndpoint struct {
	// AdvertiseAddress sets the IP address for the API server to advertise.
	AdvertiseAddress net.IP `json:"advertiseAddress,omitempty"`
	// BindPort sets the secure port for the API Server to bind to.
	// Defaults to 6443.
	BindPort int32 `json:"bindPort,omitempty"`
}

func (e APIEndpoint) URL() string {
	return fmt.Sprintf("https://%s:%d", e.AdvertiseAddress, e.BindPort)
}
