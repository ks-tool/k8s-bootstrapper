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

package fetch

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type Writer func(r io.Reader) error
type TarFilter func(dst string, tr *tar.Reader, th *tar.Header) error

// ToFile saves the response body to a file.
func ToFile(dst string, perm os.FileMode) Writer {
	return func(r io.Reader) error {
		fi, err := os.OpenFile(dst, os.O_RDWR|os.O_CREATE|os.O_TRUNC, perm)
		if err != nil {
			return err
		}
		defer func() {
			if e := fi.Close(); e != nil {
				err = fmt.Errorf("file %q closing failed: %v", dst, e)
			}
		}()

		buf := make([]byte, 5*1024*1024)
		_, err = io.CopyBuffer(fi, r, buf)

		return err
	}
}

func unTar(dst string, tr *tar.Reader, hdr *tar.Header) error {
	target := filepath.Join(dst, hdr.Name)

	switch hdr.Typeflag {
	case tar.TypeDir:
		if err := os.MkdirAll(target, 0755); err != nil {
			return err
		}

	case tar.TypeReg:
		if err := ToFile(target, os.FileMode(hdr.Mode))(tr); err != nil {
			return err
		}
	}

	return nil
}

// UnTar unpacks files from tar.gz archive
func UnTar(dst string, filter ...TarFilter) Writer {
	if filter == nil {
		filter = []TarFilter{unTar}
	}

	return func(r io.Reader) error {
		gzr, err := gzip.NewReader(r)
		if err != nil {
			return err
		}
		defer func() {
			_ = gzr.Close()
		}()

		tr := tar.NewReader(gzr)
		for {
			hdr, err := tr.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				return err
			}
			if hdr == nil {
				continue
			}

			for _, f := range filter {
				if err = f(dst, tr, hdr); err != nil {
					return err
				}
			}
		}
		return nil
	}
}

func JSONUnmarshal(v any) Writer {
	return func(r io.Reader) error {
		return json.NewDecoder(r).Decode(v)
	}
}
