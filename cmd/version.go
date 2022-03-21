package cmd

import (
	"fmt"
	"runtime"

	"github.com/iter8-tools/iter8/base"
	"github.com/spf13/cobra"
)

var versionDesc = `
Print the version of Iter8 CLI.

	$ iter8 version

The output may look as follows:

	$ version.BuildInfo{Version:"v0.8.32", GitCommit:"fe51cd1e31e6a202cba7aliv9552a6d418ded79a", GitTreeState:"clean", GoVersion:"go1.17.6"}

In the sample output shown above:

- Version is the semantic version of the Iter8 CLI.
- GitCommit is the SHA hash for the commit that this version was built from.
- GitTreeState is "clean" if there are no local code changes when this binary was built, and "dirty" if the binary was built from locally modified code.
- GoVersion is the version of Go that was used to compile Iter8 CLI.
`

var (
	// version is intended to be set using LDFLAGS at build time
	// In the absence of complete semantic versioning info, the best we can do is major minor
	version = base.MajorMinor
	// metadata is extra build time data
	metadata = ""
	// gitCommit is the git sha1
	gitCommit = ""
	// gitTreeState is the state of the git tree
	gitTreeState = ""
)

// BuildInfo describes the compile time information.
type BuildInfo struct {
	// Version is the current semver.
	Version string `json:"version,omitempty"`
	// GitCommit is the git sha1.
	GitCommit string `json:"git_commit,omitempty"`
	// GitTreeState is the state of the git tree.
	GitTreeState string `json:"git_tree_state,omitempty"`
	// GoVersion is the version of the Go compiler used to compile Iter8.
	GoVersion string `json:"go_version,omitempty"`
}

// newVersionCmd creates the version command
func newVersionCmd() *cobra.Command {
	var short bool
	// versionCmd represents the version command
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print Iter8 CLI version",
		Long:  versionDesc,
		Run: func(_ *cobra.Command, _ []string) {
			v := getBuildInfo()
			if short {
				if len(v.GitCommit) >= 7 {
					fmt.Printf("%s+g%s", v.Version, v.GitCommit[:7])
					fmt.Println()
					return
				}
				fmt.Println(getVersion())
				return
			}
			fmt.Printf("%#v", v)
			fmt.Println()
		},
	}
	addVersionFlags(cmd, &short)
	return cmd
}

// getVersion returns the semver string of the version
func getVersion() string {
	if metadata == "" {
		return version
	}
	return version + "+" + metadata
}

// get returns build info
func getBuildInfo() BuildInfo {
	v := BuildInfo{
		Version:      getVersion(),
		GitCommit:    gitCommit,
		GitTreeState: gitTreeState,
		GoVersion:    runtime.Version(),
	}
	return v
}

// addVersionFlags adds flags to the version command
func addVersionFlags(cmd *cobra.Command, shortPtr *bool) {
	cmd.Flags().BoolVar(shortPtr, "short", false, "print abbreviated version info")
	cmd.Flags().Lookup("short").NoOptDefVal = "true"
}

func init() {
	vCmd := newVersionCmd()
	rootCmd.AddCommand(vCmd)
}