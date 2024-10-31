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
	"os"

	"github.com/ks-tool/k8s-bootstrapper/internal/config"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/util/yaml"
)

var rootCmd = &cobra.Command{
	Use:   "k8s-bootstrapper",
	Short: "Fast bootstrap a Kubernetes cluster",
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().String("config", "", "")
}

func readConfig(cmd *cobra.Command) (*config.Config, error) {
	cfg := new(config.Config)

	configPath, _ := cmd.Flags().GetString("config")
	if len(configPath) > 0 {
		if err := func() error {
			fi, err := os.Open(configPath)
			if err != nil {
				return err
			}
			defer func() { _ = fi.Close() }()

			return yaml.NewYAMLToJSONDecoder(fi).Decode(cfg)
		}(); err != nil {
			return nil, err
		}
	}

	if err := config.SetDefaults(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
