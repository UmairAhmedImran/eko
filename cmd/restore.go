package cmd

import (
	"bufio"
	"eko/internal/db"
	"eko/internal/snapshot"
	"eko/internal/util"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var (
	restoreYes      bool
	restoreProgress bool
)

// errRestoreNeedsTTY is returned when confirmation is required but there is no
// terminal to ask on. Failing is the only safe option here: assuming "yes" would
// delete the working directory of a script that never opted in, and blocking on a
// read would hang a CI job until it times out.
var errRestoreNeedsTTY = errors.New(
	"restore needs confirmation but input is not a terminal; re-run with --yes to confirm")

var restoreCmd = &cobra.Command{
	Use:     "restore [id]",
	Short:   "Restore snapshot",
	Args:    cobra.ExactArgs(1),
	PreRunE: requireInitialized,
	RunE: func(cmd *cobra.Command, args []string) error {
		target := args[0]
		database := db.InitDB()
		defer database.Close()

		id, path, err := db.ResolveSnapshot(database, target)
		if err != nil {
			return err
		}

		if !restoreYes {
			confirmed, err := confirmRestore(cmd, id, path)
			if err != nil {
				return err
			}
			if !confirmed {
				fmt.Fprintln(cmd.OutOrStdout(), "Restore cancelled. Nothing was deleted.")
				return nil
			}
		}

		// Set up progress bar if enabled and stderr is a TTY
		var onProgress func()
		showProgress := restoreProgress && util.IsTTY(os.Stderr)
		if showProgress {
			// For restore, we use the pending changes count as an estimate
			changes, err := snapshot.PendingRestoreChanges(path)
			if err == nil && len(changes) > 0 {
				prog := util.NewProgress(len(changes), os.Stderr, "Restoring snapshot...")
				prog.Start()
				defer prog.Stop()
				onProgress = prog.Increment
			}
		}

		err = snapshot.RestoreSnapshot(path, onProgress)
		if err != nil {
			return err
		}
		fmt.Printf("Restored: %s (tag: %s)\n", id, target)

		return nil
	},
}

// confirmRestore lists what restore is about to overwrite or delete and waits
// for an explicit "y". Anything else — including a bare Enter — cancels.
func confirmRestore(cmd *cobra.Command, id, path string) (bool, error) {
	in := cmd.InOrStdin()
	if f, ok := in.(*os.File); ok {
		info, statErr := f.Stat()
		if statErr != nil {
			return false, fmt.Errorf("cannot inspect input stream: %w", statErr)
		}
		if info.Mode()&os.ModeCharDevice == 0 {
			return false, errRestoreNeedsTTY
		}
	}

	changes, err := snapshot.PendingRestoreChanges(path)
	if err != nil {
		return false, fmt.Errorf("cannot determine what restore would change: %w", err)
	}

	out := cmd.OutOrStdout()
	if len(changes) == 0 {
		fmt.Fprintf(out, "Restoring snapshot %s. Nothing in the working directory will be overwritten or deleted.\n", id)
	} else {
		fmt.Fprintf(out,
			"Restoring snapshot %s will overwrite or delete %s in the working directory:\n",
			id, pluralPaths(len(changes)))
		for _, name := range changes {
			fmt.Fprintf(out, "  %s\n", name)
		}
		fmt.Fprintln(out, "Any changes made since your last save will be lost.")
	}
	fmt.Fprint(out, "Continue? [y/N]: ")

	line, err := bufio.NewReader(in).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return false, fmt.Errorf("cannot read confirmation: %w", err)
	}

	answer := strings.ToLower(strings.TrimSpace(line))
	return answer == "y" || answer == "yes", nil
}

func pluralPaths(n int) string {
	if n == 1 {
		return "1 path"
	}
	return fmt.Sprintf("%d paths", n)
}

func init() {
	restoreCmd.Flags().BoolVarP(&restoreYes, "yes", "y", false,
		"skip the confirmation prompt (required when stdin is not a terminal)")
	restoreCmd.Flags().BoolVar(&restoreProgress, "progress", true,
		"show progress bar during restore (default true when TTY)")
	rootCmd.AddCommand(restoreCmd)
}
