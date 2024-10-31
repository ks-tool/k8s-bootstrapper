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
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/ks-tool/k8s-bootstrapper/internal/config"
)

const (
	hashFileSuffix = ".sha256"
)

var (
	ErrIsNotRegularFile = errors.New("not a regular File")
	ErrNotImplemented   = errors.New("not implemented")
	ErrFileNotFound     = errors.New("file not found")
)

type Endpoint interface {
	FileURL(string, string) (string, error)
	HashFileURL(string, string) (string, error)
	LastTag() (string, error)
}

type httpError struct {
	status int
	error  error
}

func (e *httpError) Error() string {
	return fmt.Sprintf("%d: %s", e.status, e.error.Error())
}

type Proxy struct {
	dir string
	sem *Semaphore
}

func NewProxy(cfg *config.Config) *Proxy {
	return &Proxy{
		dir: cfg.AssetsDir,
		sem: NewSemaphore(),
	}
}

// Handler handle request url /binary-name[/version]
// /coredns/v1.10.0 -> pattern - /coredns/
// /etcd/v3.14.5 -> pattern - /etcd/
// /kubectl/v1.31.1 -> pattern - /
func (p *Proxy) Handler(endp Endpoint) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet, http.MethodHead:
		default:
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}

		reqPath := path.Clean(r.URL.Path)
		if len(reqPath) > 0 && reqPath[0] == '/' {
			reqPath = reqPath[1:]
		}

		if len(reqPath) == 0 || reqPath[0] == '.' || strings.HasSuffix(reqPath, hashFileSuffix) {
			http.NotFound(w, nil)
			return
		}

		reqPathParts := strings.Split(reqPath, "/")
		filename := reqPathParts[0]
		version := reqPathParts[1]

		reqFileLen := len(reqPathParts[1:])
		if reqFileLen == 1 {
			latestVersion, err := endp.LastTag()
			if err != nil {
				httpErrorWriter(w, err)
				return
			}

			redirectPath := path.Join(filename, latestVersion)
			http.Redirect(w, r, redirectPath, http.StatusTemporaryRedirect)
			return
		} else if reqFileLen > 2 {
			http.NotFound(w, nil)
			return
		}

		reqFileRelPath := filepath.Join(version, filename)

		filePath := filepath.Join(p.dir, reqFileRelPath)
		hashFilePath := filePath + hashFileSuffix

		reqFileIsExist, err := fileIsExist(filePath)
		if err != nil {
			httpErrorWriter(w, err)
			return
		}

		if !reqFileIsExist && r.Method == http.MethodGet {
			download := func() error {
				p.sem.Acquire(reqPath)
				defer p.sem.Release(reqPath)

				url, err := endp.HashFileURL(filename, version)
				if err != nil {
					return err
				}

				if err = p.fetchHash(r.Context(), url, hashFilePath); err != nil {
					return err
				}

				url, err = endp.FileURL(filename, version)
				if err != nil {
					return err
				}

				return p.fetchFile(r.Context(), url, filePath)
			}

			if err = download(); err != nil {
				httpErrorWriter(w, err)
				return
			}

			reqFileIsExist = true
		}

		hashFileIsExist, err := fileIsExist(hashFilePath)
		if err != nil {
			httpErrorWriter(w, err)
			return
		}

		var hash string
		if hashFileIsExist {
			b, err := os.ReadFile(hashFilePath)
			if err != nil {
				httpErrorWriter(w, err)
				return
			}

			hash = string(b)
		}

		fileIsValid := true
		if len(hash) > 0 {
			w.Header().Set("ETag", hash)

			fileIsValid, err = validateSha256Hash(filePath, hash)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}

		if r.Method == http.MethodHead {
			if fileIsValid && reqFileIsExist {
				fs, _ := os.Stat(filePath)
				w.Header().Set("Content-Length", fmt.Sprintf("%d", fs.Size()))
			}
			w.Header().Set("Content-Type", "application/octet-stream")
			return
		}

		if !fileIsValid {
			if err = os.Remove(filePath); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			if hashFileIsExist {
				if err = os.Remove(hashFilePath); err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
			}

			http.Redirect(w, r, r.URL.Path, http.StatusTemporaryRedirect)
			return
		}

		sendFile(w, filePath)
	}
}

func (p *Proxy) fetchHash(ctx context.Context, url, filePath string) error {
	resp, err := p.download(ctx, url)
	if err != nil {
		return err
	}

	checksum, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if err = os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return err
	}

	if len(checksum) == 64 {
		return os.WriteFile(filePath, checksum, 0644)
	}

	filename := filepath.Base(filePath)
	lines := strings.Split(string(checksum), "\n")
	for _, line := range lines {
		if strings.Contains(line, filename) {
			hash := []byte(strings.SplitN(line, " ", 2)[0])
			return os.WriteFile(filePath, hash, 0644)
		}
	}

	return fmt.Errorf("hashsum for File %q not found", filename)
}

func (p *Proxy) fetchFile(ctx context.Context, url, filePath string) error {
	resp, err := p.download(ctx, url)
	if err != nil {
		return err
	}

	if err = os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return err
	}

	return writeToFile(filePath, resp.Body, 0644)
}
func (p *Proxy) download(ctx context.Context, url string) (*http.Response, error) {
	resp, err := get(ctx, url)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		msg := http.StatusText(resp.StatusCode)
		if b, _ := io.ReadAll(resp.Body); len(b) > 0 {
			msg = string(b)
		}

		return nil, &httpError{status: resp.StatusCode, error: errors.New(msg)}
	}

	return resp, nil
}

func writeToFile(dst string, src io.Reader, perm os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	fi, err := os.OpenFile(dst, os.O_RDWR|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		return err
	}
	defer func() {
		if e := fi.Close(); e != nil {
			err = fmt.Errorf("file %q closing failed: %v", dst, e)
		}
	}()

	return copyBuffer(fi, src)
}

func copyBuffer(dst io.Writer, src io.Reader) error {
	buf := make([]byte, 5*1024*1024)
	_, err := io.CopyBuffer(dst, src, buf)
	return err
}

func sendFile(w http.ResponseWriter, file string) {
	f, err := os.Open(file)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer func() { _ = f.Close() }()

	fs, _ := f.Stat()
	w.Header().Set("Content-Length", fmt.Sprintf("%d", fs.Size()))

	if err = copyBuffer(w, f); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// validateSha256Hash calculate sha256 hash of a File and compare with hash argument
func validateSha256Hash(file string, hash string) (bool, error) {
	if len(hash) == 0 {
		return false, nil
	}
	b, err := sha256Sum(file)
	if err != nil || len(b) == 0 {
		return false, err
	}

	return hex.EncodeToString(b) == hash, nil
}

// sha256Sum calculate sha256 hash of a File
func sha256Sum(file string) ([]byte, error) {
	if len(file) == 0 {
		return nil, fmt.Errorf("empty path to File")
	}

	f, err := os.Open(file)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer func() {
		_ = f.Close()
	}()

	h := sha256.New()
	if _, err = io.Copy(h, f); err != nil {
		return nil, err
	}

	return h.Sum(nil), nil
}

func fileIsExist(file string) (bool, error) {
	fi, err := os.Stat(file)
	if err != nil {
		if !os.IsNotExist(err) {
			return false, err
		}

		return false, nil
	}

	if fi.IsDir() {
		return false, ErrIsNotRegularFile
	}

	return true, nil
}

func httpErrorWriter(w http.ResponseWriter, err error) {
	var e *httpError
	var status int
	var msg string

	switch {
	case errors.Is(err, ErrIsNotRegularFile):
		status = http.StatusNotFound
		msg = "404 page not found"
	case errors.Is(err, ErrNotImplemented):
		status = http.StatusNotImplemented
		msg = err.Error()
	case errors.As(err, &e):
		status = e.status
		msg = e.error.Error()
	default:
		if err != nil {
			status = http.StatusInternalServerError
			msg = err.Error()
		} else {
			status = http.StatusNotFound
			msg = "404 page not found"
		}
	}

	http.Error(w, msg, status)
}

func get(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	return http.DefaultClient.Do(req)
}
