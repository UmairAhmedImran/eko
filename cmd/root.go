package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"eko/internal/telemetry"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "eko",
	Short: "eko – AI Snapshot Versioning CLI",

	SilenceUsage:  true,
	SilenceErrors: true,
}

func Execute() {
	shutdown, err := telemetry.Init(context.Background())
	if err != nil {
		fmt.Fprintln(
			os.Stderr,
			"Warning: telemetry initialization failed:",
			err,
		)
	} else {
		defer func() {
			ctx, cancel := context.WithTimeout(
				context.Background(),
				5*time.Second,
			)
			defer cancel()

			if err := shutdown(ctx); err != nil {
				fmt.Fprintln(
					os.Stderr,
					"Warning: telemetry shutdown failed:",
					err,
				)
			}
		}()
	}

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
