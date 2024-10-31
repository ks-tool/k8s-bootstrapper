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

package utils

import (
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"strings"
)

func GetIndexedIPFromCIDR(s string, idx int64) (net.IP, error) {
	_, cidr, err := net.ParseCIDR(s)
	if err != nil {
		return nil, err
	}

	baseIP := big.NewInt(0).SetBytes(cidr.IP.To16())
	r := big.NewInt(0).Add(baseIP, big.NewInt(idx)).Bytes()
	r = append(make([]byte, 16), r...)

	return r[len(r)-16:], nil
}

func GetOutboundIP() (net.IP, error) {
	const url2ip = "https://2ip.ru"

	req, err := http.NewRequest("GET", url2ip, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "curl/8.7.1")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return net.ParseIP(strings.TrimSpace(string(b))), nil
}
