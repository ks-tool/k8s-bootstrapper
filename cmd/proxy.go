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

package cmd

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ks-tool/k8s-bootstrapper/internal/config"
	"github.com/ks-tool/k8s-bootstrapper/pkg/file-proxy"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// proxyCmd represents the proxy command
var proxyCmd = &cobra.Command{
	Use:   "proxy",
	Short: "Run file proxy cache",
	Run: func(cmd *cobra.Command, args []string) {
		logger := logrus.New()
		cfg, err := readConfig(cmd)
		if err != nil {
			logger.Fatal(err)
		}

		mux := http.NewServeMux()
		proxy := fileproxy.NewProxy(cfg)
		mux.HandleFunc("/coredns/", proxy.Handler(fileproxy.Github{
			Owner: "coredns",
			Repo:  "coredns",
			File: func(v string) string {
				if v[0] == 'v' {
					v = v[1:]
				}
				return fmt.Sprintf("coredns_%s_linux_amd64.tgz", v)
			},
			HashFile: func(v string) string {
				if v[0] == 'v' {
					v = v[1:]
				}
				return fmt.Sprintf("coredns_%s_linux_amd64.tgz.sha256", v)
			},
		}))

		mux.HandleFunc("/etcd/", proxy.Handler(fileproxy.Github{
			Owner: "etcd-io",
			Repo:  "etcd",
			File: func(v string) string {
				return fmt.Sprintf("etcd-%s-linux-amd64.tar.gz", v)
			},
			HashFile: func(string) string { return "SHA256SUMS" },
		}))

		mux.HandleFunc("/", proxy.Handler(fileproxy.Kube))

		w := logger.Writer()
		defer func() { _ = w.Close() }()

		srv := &http.Server{
			Addr:     fmt.Sprintf("%s:%d", cfg.ControlPlain.LocalAPIEndpoint.AdvertiseAddress, cfg.ProxyPort),
			Handler:  mux,
			ErrorLog: log.New(w, "proxy: ", 0),
		}

		go func() {
			if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				logger.Fatal(err)
			}
		}()

		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT)

		<-sig

		func() {
			srv.SetKeepAlivesEnabled(false)

			ctx, cancel := context.WithTimeout(cmd.Context(), 15*time.Second)
			defer cancel()

			if err := srv.Shutdown(ctx); err != nil {
				logger.Fatal(err)
			}
		}()
	},
}

func init() {
	rootCmd.AddCommand(proxyCmd)
}

const urlFmt = "%s/%s/%s"

type proxyUrl struct {
	pfx string
	cfg *config.Config
}

func newProxyUrl(cfg *config.Config) proxyUrl {
	return proxyUrl{
		pfx: fmt.Sprintf(
			"http://%s:%d",
			cfg.ControlPlain.LocalAPIEndpoint.AdvertiseAddress,
			cfg.ProxyPort,
		),
		cfg: cfg,
	}
}

func (p proxyUrl) etcd() string {
	return fmt.Sprintf(urlFmt, p.pfx, "etcd", p.cfg.ControlPlain.EtcdVersion)
}

func (p proxyUrl) coredns() string {
	return fmt.Sprintf(urlFmt, p.pfx, "coredns", p.cfg.ControlPlain.CorednsVersion)
}

func (p proxyUrl) kube(name string) string {
	return fmt.Sprintf(urlFmt, p.pfx, name, p.cfg.ControlPlain.KubernetesVersion)
}
