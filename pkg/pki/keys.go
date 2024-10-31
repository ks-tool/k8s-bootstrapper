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
	"encoding/pem"
	"fmt"
	"os"
	"strings"
)

type PrivateKey struct {
	key      *rsa.PrivateKey
	new      bool
	filepath string
}

type PublicKey struct {
	key *rsa.PublicKey
}

func NewPrivateKey() (*PrivateKey, error) {
	key, err := rsa.GenerateKey(rand.Reader, keySize)
	if err != nil {
		return nil, fmt.Errorf("could not generate private key: %v", err)
	}

	return &PrivateKey{key: key, new: true}, nil
}

func (key *PrivateKey) Public() *PublicKey {
	return &PublicKey{key: key.key.Public().(*rsa.PublicKey)}
}

func (key *PrivateKey) Filepath() string {
	return key.filepath
}

func (key *PrivateKey) CertificateSign(cr *CertRequest) (*Certificate, error) {
	cert := cr.x509Certificate()
	// Self-signed certificate
	ca := &CA{
		key:  key.key,
		cert: cert,
	}

	if !cert.IsCA {
		var err error
		ca, err = cr.LoadCA()
		if err != nil {
			return nil, fmt.Errorf("could not load CA: %v", err)
		}
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, cert, ca.cert, key.key.Public(), ca.key)
	if err != nil {
		return nil, fmt.Errorf("could not create certificate: %v", err)
	}

	return &Certificate{raw: certBytes}, nil
}

func (key *PrivateKey) PEM() []byte {
	block := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key.key)}
	return pem.EncodeToMemory(block)
}

func (key *PrivateKey) Save(keyFile string) error {
	if len(keyFile) == 0 {
		keyFile = key.filepath
	}

	if err := os.WriteFile(keyFile, key.PEM(), 0640); err != nil {
		return fmt.Errorf("could not write private key %q: %v", keyFile, err)
	}
	return nil
}

func (key *PrivateKey) IsNew() bool {
	return key.new
}

func (key *PrivateKey) PublicKeyFilepath() string {
	return strings.ReplaceAll(key.filepath, keyExt, pubExt)
}

func (key *PrivateKey) CertificateFilepath() string {
	return strings.ReplaceAll(key.filepath, keyExt, certExt)
}

func (key *PublicKey) PEM() []byte {
	der, err := x509.MarshalPKIXPublicKey(key.key)
	if err != nil {
		panic(err)
	}
	block := &pem.Block{Type: "PUBLIC KEY", Bytes: der}
	return pem.EncodeToMemory(block)
}

func (key *PublicKey) Save(keyFile string) error {
	if err := os.WriteFile(keyFile, key.PEM(), 0644); err != nil {
		return fmt.Errorf("could not write public key %q: %v", keyFile, err)
	}
	return nil
}
