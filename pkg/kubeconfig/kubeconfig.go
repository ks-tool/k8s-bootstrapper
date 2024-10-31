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
	"fmt"
	"os"
	"path/filepath"

	"github.com/ks-tool/k8s-bootstrapper/internal/config"
	"github.com/ks-tool/k8s-bootstrapper/pkg/pki"
	"github.com/ks-tool/k8s-bootstrapper/utils"

	"k8s.io/apimachinery/pkg/runtime"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	clientcmdapilatest "k8s.io/client-go/tools/clientcmd/api/latest"
)

const (
	clusterName = "kubernetes"
	confExt     = ".conf"
)

type KubeConfigSpec struct {
	Name      string
	OutputDir string
	Auth      *pki.CertRequest
	ServerURL string
}

func (s *KubeConfigSpec) authInfo() (*clientcmdapi.AuthInfo, error) {
	pk, err := s.Auth.PrivateKey()
	if err != nil {
		return nil, err
	}

	crt, err := pk.CertificateSign(s.Auth)
	if err != nil {
		return nil, err
	}

	return &clientcmdapi.AuthInfo{
		ClientCertificateData: crt.PEM(),
		ClientKeyData:         pk.PEM(),
	}, nil
}

func (s *KubeConfigSpec) Filepath() string {
	return filepath.Join(s.OutputDir, s.Name+confExt)
}

func (s *KubeConfigSpec) Create() error {
	authInfo, err := s.authInfo()
	if err != nil {
		return err
	}

	ca, err := s.Auth.LoadCA()
	if err != nil {
		return err
	}

	cluster := &clientcmdapi.Cluster{
		Server:                   s.ServerURL,
		CertificateAuthorityData: ca.Certificate().PEM(),
	}

	configData := clientcmdapi.NewConfig()
	configData.Clusters[clusterName] = cluster

	user := s.Auth.CommonName
	configData.AuthInfos[user] = authInfo

	context := fmt.Sprintf("%s@%s", user, clusterName)
	configData.CurrentContext = context
	configData.Contexts[context] = &clientcmdapi.Context{
		Cluster:  clusterName,
		AuthInfo: user,
	}

	configBytes, err := runtime.Encode(clientcmdapilatest.Codec, configData)
	if err != nil {
		return fmt.Errorf("encode kubeconfig failed: %s", err)
	}

	outputFile := s.Filepath()
	if err = os.WriteFile(outputFile, configBytes, 0640); err != nil {
		return fmt.Errorf("write kubeconfig file %q failed: %s", outputFile, err)
	}

	if err = utils.Chown(outputFile, "", config.DefaultGroupname); err != nil {
		return fmt.Errorf("chown kubeconfig file %q failed: %s", outputFile, err)
	}

	return nil
}
