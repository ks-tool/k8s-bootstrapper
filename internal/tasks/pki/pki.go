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

package pki

import (
	"context"
	"crypto/x509"
	"fmt"
	"net"
	"os"

	"github.com/ks-tool/k8s-bootstrapper/internal/config"
	"github.com/ks-tool/k8s-bootstrapper/pkg/flow"
	"github.com/ks-tool/k8s-bootstrapper/pkg/pki"
	"github.com/ks-tool/k8s-bootstrapper/utils"
)

var (
	CA = genCert(&pki.CertRequest{
		Name:        config.DefaultCAName,
		CommonName:  "kubernetes",
		PkiDir:      config.DefaultCertificatesDir,
		Description: "Generate the self-signed Kubernetes CA to provision identities for other Kubernetes components",
	})
	KubeApiserver = func(cfg *config.Config) flow.Action {
		clusterIP, err := utils.GetIndexedIPFromCIDR(config.DefaultServicesSubnet, 1)
		if err != nil {
			panic(fmt.Sprintf("failed to get cluster IP address: %v", err))
		}

		publicIP, err := utils.GetOutboundIP()
		if err != nil {
			panic(fmt.Sprintf("failed to get public IP address: %v", err))
		}

		return genCert(&pki.CertRequest{
			Name:       "apiserver",
			CAName:     config.DefaultCAName,
			CommonName: "kube-apiserver",
			Usages:     []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
			AltNames: pki.AltNames{
				DNSNames: []string{
					cfg.NodeName,
					"kubernetes",
					"kubernetes.default",
					"kubernetes.default.svc",
					fmt.Sprintf("kubernetes.default.svc.%s", cfg.ControlPlain.DNSDomain),
				},
				IPs: []net.IP{
					publicIP,
					cfg.ControlPlain.LocalAPIEndpoint.AdvertiseAddress,
					clusterIP,
				},
			},
			PkiDir:      config.DefaultCertificatesDir,
			Description: "Generate the certificate for serving the Kubernetes API",
		})
	}
	FrontProxyCA = genCert(&pki.CertRequest{
		Name:        "front-proxy-ca",
		CommonName:  "front-proxy-ca",
		PkiDir:      config.DefaultCertificatesDir,
		Description: "Generate the self-signed CA to provision identities for front proxy",
	})
	FrontProxyClient = genCert(&pki.CertRequest{
		Name:        "front-proxy-client",
		CAName:      "front-proxy-ca",
		CommonName:  "front-proxy-client",
		Usages:      []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		PkiDir:      config.DefaultCertificatesDir,
		Description: "Generate the certificate for the front proxy client",
	})
	SA = genKey(&pki.PublicKeyRequest{
		Name:        "sa",
		PkiDir:      config.DefaultCertificatesDir,
		Description: "Generate a private key for signing service account tokens along with its public key",
	})
)

func genCert(req *pki.CertRequest) flow.Action {
	action := func(ctx context.Context) (flow.StatusType, error) {
		log := ctx.Value(flow.LogKey).(*flow.Logger)
		log.Infof("generate private key and certificate %s", req.Name)

		pk, err := req.PrivateKey()
		if err != nil {
			return flow.StatusFailed, err
		}

		if pk.IsNew() {
			if err = pk.Save(""); err != nil {
				return flow.StatusFailed, err
			}
		}

		certFile := pk.CertificateFilepath()
		if _, err = os.Stat(certFile); err == nil {
			if !pk.IsNew() {
				return flow.StatusSkipped, nil
			}
		} else if !os.IsNotExist(err) {
			return flow.StatusFailed, err
		}

		crt, err := pk.CertificateSign(req)
		if err != nil {
			return flow.StatusFailed, fmt.Errorf("failed to sign certificate: %s", err)
		}

		if err = crt.Save(certFile); err != nil {
			return flow.StatusFailed, err
		}

		return flow.StatusSuccess, nil
	}

	return flow.NewAction(req.Name, action)
}

func genKey(req *pki.PublicKeyRequest) flow.Action {
	action := func(ctx context.Context) (flow.StatusType, error) {
		pk, err := req.PrivateKey()
		if err != nil {
			return flow.StatusFailed, err
		}

		if pk.IsNew() {
			if err = pk.Save(""); err != nil {
				return flow.StatusFailed, err
			}
		}

		pubKey := pk.PublicKeyFilepath()
		if _, err = os.Stat(pubKey); err == nil {
			if !pk.IsNew() {
				return flow.StatusSkipped, nil
			}
		} else if !os.IsNotExist(err) {
			return flow.StatusFailed, err
		}

		if err = pk.Public().Save(pubKey); err != nil {
			return flow.StatusFailed, err
		}

		return flow.StatusSuccess, nil
	}

	return flow.NewAction(req.Name+":"+req.Name, action)
}
