# go-pki

### Examples
```go
package main

import (
	"crypto/x509"
	"fmt"
	"log"
	"net"
	"os"
	
	"github.com/ks-tool/k8s-bootstrapper/pkg/pki"
)

func main() {
	const pkiDir = "/etc/kubernetes/pki"

	kubeCA := &pki.CertRequest{
		Name:                 "ca",
		CommonName:           "kubernetes",
		PrivateKeyAndCertDir: pkiDir,
	}

	kubeApiServer := &pki.CertRequest{
		Name:       "apiserver",
		CAName:     "ca",
		CommonName: "kube-apiserver",
		Usages:     []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		AltNames: pki.AltNames{
			DNSNames: []string{
				"master",
				"kubernetes",
				"kubernetes.default",
				"kubernetes.default.svc",
				"kubernetes.default.svc.cluster.local",
			},
			IPs: []net.IP{
				{10, 96, 0, 1},
				{192, 168, 96, 1},
			},
		},
		PrivateKeyAndCertDir: pkiDir,
	}

	kubeSA := &pki.PublicKeyRequest{
		Name:      "sa",
		OutputDir: pkiDir,
	}
	
	if err := createPrivateKeyAndCertificate(kubeCA); err != nil {
		log.Fatal(err)
	}

	if err := createPrivateKeyAndCertificate(kubeApiServer); err != nil {
		log.Fatal(err)
	}

	if err := createPrivateAndPublicKeys(kubeSA); err != nil {
		log.Fatal(err)
	}
}

func createPrivateKeyAndCertificate(cr *pki.CertRequest) error {
	pk, err := cr.PrivateKey()
	if err != nil {
		return err
	}

	if pk.IsNew() {
		if err = pk.Save(""); err != nil {
			return err
		}
	}

	certFile := pk.CertificateFilepath()
	if _, err = os.Stat(certFile); err == nil {
		if !pk.IsNew() {
			return fmt.Errorf("certificate file %s already exists", certFile)
		}
	} else if !os.IsNotExist(err) {
		return err
	}

	crt, err := pk.CertificateSign(cr)
	if err != nil {
		return fmt.Errorf("failed to sign certificate: %s", err)
	}

	return crt.Save(certFile)
}

func createPrivateAndPublicKeys(cr *pki.PublicKeyRequest) error {
	pk, err := cr.PrivateKey()
	if err != nil {
		return err
	}

	if pk.IsNew() {
		if err = pk.Save(""); err != nil {
			return err
		}
	}

	pubKey := pk.PublicKeyFilepath()
	if _, err = os.Stat(pubKey); err == nil {
		if !pk.IsNew() {
			return fmt.Errorf("public key file %s already exists", pubKey)
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	
	return pk.Public().Save(pubKey)
}
```
