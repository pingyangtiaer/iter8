/*
Copyright © 2021 Iter8 authors

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
	ia "github.com/iter8-tools/iter8/action"

	"github.com/iter8-tools/iter8/base/log"
	"github.com/spf13/cobra"
)

const hubDesc = `
Download an experiment chart to a local directory.

	$ iter8 hub -c load-test-http

This command is intended for development and testing of experiment charts. For production usage, the iter8 launch command is recommended.
`

// newHubCmd creates the hub command
func newHubCmd() *cobra.Command {
	actor := ia.NewHubOpts()

	cmd := &cobra.Command{
		Use:   "hub",
		Short: "Download experiment chart",
		Long:  hubDesc,
		Run: func(_ *cobra.Command, _ []string) {
			if err := actor.LocalRun(); err != nil {
				log.Logger.Error(err)
			}
		},
	}
	addChartFlags(cmd, &actor.ChartPathOptions, &actor.ChartNameAndDestOptions)
	return cmd
}

func init() {
	rootCmd.AddCommand(newHubCmd())
}