// Copyright 2019 The Kubernetes Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"fmt"
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/klog"

	"sigs.k8s.io/krew/cmd/krew/cmd/internal"
	"sigs.k8s.io/krew/internal/index/indexscanner"
	"sigs.k8s.io/krew/internal/installation"
)

func init() {
	var noUpdateIndex *bool

	// upgradeCmd represents the upgrade command
	var upgradeCmd = &cobra.Command{
		Use:   "upgrade",
		Short: "Upgrade installed plugins to newer versions",
		Long: `Upgrade installed plugins to a newer version.
This will reinstall all plugins that have a newer version in the local index.
Use "kubectl krew update" to renew the index.
To only upgrade single plugins provide them as arguments:
kubectl krew upgrade foo bar"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			var ignoreUpgraded bool
			var skipErrors bool

			var pluginNames []string
			if len(args) == 0 {
				// Upgrade all plugins.
				installed, err := installation.ListInstalledPlugins(paths.InstallReceiptsPath())
				if err != nil {
					return errors.Wrap(err, "failed to find all installed versions")
				}
				for name := range installed {
					pluginNames = append(pluginNames, name)
				}
				ignoreUpgraded = true
				skipErrors = true
			} else {
				// Upgrade certain plugins
				pluginNames = args
			}

			var nErrors int
			for _, name := range pluginNames {
				plugin, err := indexscanner.LoadPluginByName(paths.IndexPluginsPath(), name)
				if err != nil {
					if !os.IsNotExist(err) {
						return errors.Wrapf(err, "failed to load the plugin manifest for plugin %s", name)
					} else if !skipErrors {
						return errors.Errorf("plugin %q does not exist in the plugin index", name)
					}
				}

				if err == nil {
					fmt.Fprintf(os.Stderr, "Upgrading plugin: %s\n", name)
					err = installation.Upgrade(paths, plugin)
					if ignoreUpgraded && err == installation.ErrIsAlreadyUpgraded {
						fmt.Fprintf(os.Stderr, "Skipping plugin %s, it is already on the newest version\n", name)
						continue
					}
				}
				if err != nil {
					nErrors++
					if skipErrors {
						fmt.Fprintf(os.Stderr, "WARNING: failed to upgrade plugin %q, skipping (error: %v)\n", name, err)
						continue
					}
					return errors.Wrapf(err, "failed to upgrade plugin %q", name)
				}
				fmt.Fprintf(os.Stderr, "Upgraded plugin: %s\n", name)
				internal.PrintSecurityNotice(plugin.Name)
			}
			if nErrors > 0 {
				fmt.Fprintf(os.Stderr, "WARNING: Some plugins failed to upgrade, check logs above.\n")
			}
			return nil
		},
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if *noUpdateIndex {
				klog.V(4).Infof("--no-update-index specified, skipping updating local copy of plugin index")
				return nil
			}
			return ensureIndexUpdated(cmd, args)
		},
	}

	noUpdateIndex = upgradeCmd.Flags().Bool("no-update-index", false, "(Experimental) do not update local copy of plugin index before upgrading")
	rootCmd.AddCommand(upgradeCmd)
}
