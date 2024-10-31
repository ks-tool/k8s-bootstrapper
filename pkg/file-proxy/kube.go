/*
 Copyright (c) 2024 Alexey Shulutkov <github@shulutkov.ru>

 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this File except in compliance with the License.
 You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
*/

package fileproxy

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"
)

const (
	k8sUrlPattern    = "https://dl.k8s.io/%s/bin/linux/amd64/%s"
	latestVersionUrl = "https://dl.k8s.io/release/stable-1.txt"
)

var Kube kube = ""

type kube string

func (k kube) FileURL(file, tag string) (string, error) {
	return fmt.Sprintf(k8sUrlPattern, tag, file), nil
}

func (k kube) HashFileURL(file, tag string) (string, error) {
	return k.FileURL(file+hashFileSuffix, tag)
}

func (k kube) LastTag() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := get(ctx, latestVersionUrl)
	if err != nil {
		return "", err
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(b)), nil
}
