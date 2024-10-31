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
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
)

type CA struct {
	key  *rsa.PrivateKey
	cert *x509.Certificate
}

type Certificate struct {
	cert *x509.Certificate
	raw  []byte
}

func (ca *CA) Certificate() *Certificate {
	return &Certificate{cert: ca.cert}
}

func (crt *Certificate) PEM() []byte {
	bytes := crt.raw
	if bytes == nil {
		bytes = crt.cert.Raw
	}

	block := &pem.Block{Type: "CERTIFICATE", Bytes: bytes}
	return pem.EncodeToMemory(block)
}

func (crt *Certificate) Save(certFile string) error {
	if err := os.WriteFile(certFile, crt.PEM(), 0644); err != nil {
		return fmt.Errorf("could not write certificate %q: %v", certFile, err)
	}
	return nil
}
