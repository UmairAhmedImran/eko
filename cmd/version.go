package cmd

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
)

// Build metadata. These are overridden at release time through the Go linker,
// e.g. -ldflags "-X eko/cmd.Version=1.1.0 -X eko/cmd.Commit=8c9d1a2f".
// The defaults keep locally built binaries readable instead of printing blanks.
var (
	Version   = "dev"
	Commit    = "unknown"
	BuildDate = "unknown"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version, runtime and build information",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		_, err := fmt.Fprint(cmd.OutOrStdout(), formatVersion(Version, Commit, BuildDate))
		return err
	},
}

// formatVersion renders the version banner shared by `eko version` and the
// root-level -v/--version flags.
func formatVersion(version, commit, buildDate string) string {
	return fmt.Sprintf(
		"eko version %s (%s/%s)\nGo version: %s\nGit commit: %s\nBuild date: %s\n",
		orFallback(version, "dev"),
		runtime.GOOS,
		runtime.GOARCH,
		runtime.Version(),
		orFallback(commit, "unknown"),
		orFallback(buildDate, "unknown"),
	)
}

// orFallback guards against linker flags that inject empty strings, so the
// banner never reports a blank field.
func orFallback(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
