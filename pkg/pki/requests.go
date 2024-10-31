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
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	day      = 24 * time.Hour
	year     = 365 * day
	tenYears = 10*year + 3*day

	keySize = 2048
	keyExt  = ".key"
	certExt = ".crt"
	pubExt  = ".pub"

	etcdPrefixName = "etcd-"
)

type CertRequest struct {
	Name         string
	CAName       string
	CommonName   string
	Organization []string
	Usages       []x509.ExtKeyUsage
	AltNames     AltNames

	PkiDir      string
	Description string
}

type AltNames struct {
	DNSNames []string
	IPs      []net.IP
}

type PublicKeyRequest struct {
	Name string

	PkiDir      string
	Description string
}

func (r *CertRequest) PrivateKey() (*PrivateKey, error) {
	return privateKey(r.keyCertFile(keyExt))
}

func (r *CertRequest) keyCertFile(ext string) string {
	name := r.Name
	if strings.HasPrefix(name, etcdPrefixName) {
		name = strings.TrimPrefix(name, etcdPrefixName)
	}
	return filepath.Join(r.PkiDir, name+ext)
}

func (r *CertRequest) caFile(ext string) string {
	name := r.CAName
	if strings.HasPrefix(name, etcdPrefixName) {
		name = filepath.Join(strings.TrimPrefix(name, etcdPrefixName), "etcd")
	}
	return filepath.Join(r.PkiDir, name+ext)
}

func (r *CertRequest) x509Certificate() *x509.Certificate {
	isCA := len(r.CAName) == 0
	d := year

	keyUsage := x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature
	if isCA {
		keyUsage |= x509.KeyUsageCertSign
		d = tenYears
	}

	serial, _ := rand.Int(rand.Reader, new(big.Int).SetInt64(math.MaxInt64-1))
	notBefore := time.Now().UTC()
	notAfter := notBefore.Add(d)

	return &x509.Certificate{
		Subject: pkix.Name{
			CommonName:   r.CommonName,
			Organization: r.Organization,
		},
		DNSNames:              r.AltNames.DNSNames,
		IPAddresses:           r.AltNames.IPs,
		SerialNumber:          serial,
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              keyUsage,
		ExtKeyUsage:           r.Usages,
		BasicConstraintsValid: true,
		IsCA:                  isCA,
	}
}

func (r *CertRequest) LoadCA() (*CA, error) {
	pk, err := loadPrivateKeyFromDisk(r.caFile(keyExt))
	if err != nil {
		return nil, err
	}

	crt, err := loadCertificateFromDisk(r.caFile(certExt))
	if err != nil {
		return nil, err
	}

	return &CA{key: pk, cert: crt}, nil
}

func (r *PublicKeyRequest) PrivateKey() (*PrivateKey, error) {
	return privateKey(r.privatePublicKeyFile(keyExt))
}

func (r *PublicKeyRequest) privatePublicKeyFile(ext string) string {
	return filepath.Join(r.PkiDir, r.Name+ext)
}

/*
func (r *PublicKeyRequest) Run() *playbook.Status {
	privateKeyFile := r.privatePublicKeyFile(keyExt)
	pk, err := privateKey(privateKeyFile)
	if err != nil {
		return playbook.NewErrorStatus(err)
	}

	if pk.new {
		if err = pk.Save(privateKeyFile); err != nil {
			return playbook.NewErrorStatus(err)
		}
		if err = utils.Chown(r.Chown, privateKeyFile); err != nil {
			return playbook.NewErrorStatus(err)
		}
	}

	publicKeyFile := r.privatePublicKeyFile(pubExt)
	if _, err = os.Stat(publicKeyFile); err == nil {
		return playbook.NewSkippedStatus("Private and public keys already exists")
	} else if !os.IsNotExist(err) {
		return playbook.NewErrorStatus(fmt.Errorf("could not stat %q: %v", publicKeyFile, err))
	}

	if err = pk.Public().Save(publicKeyFile); err != nil {
		return playbook.NewErrorStatus(err)
	}
	if err = utils.Chown(r.Chown, publicKeyFile); err != nil {
		return playbook.NewErrorStatus(err)
	}

	return playbook.NewSuccessStatus("New private and public key has been created")
}
*/

func privateKey(keyFile string) (*PrivateKey, error) {
	if _, err := os.Stat(keyFile); os.IsNotExist(err) {
		pk, err := NewPrivateKey()
		if err != nil {
			return nil, err
		}
		pk.filepath = keyFile
		return pk, nil

	} else if err != nil {
		return nil, err
	}

	pk, err := loadPrivateKeyFromDisk(keyFile)
	if err != nil {
		return nil, err
	}

	return &PrivateKey{key: pk, filepath: keyFile}, nil
}

func loadPrivateKeyFromDisk(keyFile string) (*rsa.PrivateKey, error) {
	const errPfx = "load private key from disk"
	b, err := os.ReadFile(keyFile)
	if err != nil {
		return nil, fmt.Errorf("%s: read pem from file %q failed: %v", errPfx, keyFile, err)
	}

	block, _ := pem.Decode(b)
	pk, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("%s: parse %q failed: %v", errPfx, keyFile, err)
	}

	return pk, nil
}

func loadCertificateFromDisk(certFile string) (*x509.Certificate, error) {
	const errPfx = "load certificate from disk"
	b, err := os.ReadFile(certFile)
	if err != nil {
		return nil, fmt.Errorf("%s: read pem from file %q failed: %v", errPfx, certFile, err)
	}

	block, _ := pem.Decode(b)
	crt, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("%s: parse %q failed: %v", errPfx, certFile, err)
	}

	return crt, nil
}
