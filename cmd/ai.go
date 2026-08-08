package cmd

import (
	"context"
	"fmt"

	"eko/internal/ai/mind"
	"eko/internal/api"
	"eko/internal/db"

	"github.com/spf13/cobra"
)

var aiCmd = &cobra.Command{
	Use:   "ai",
	Short: "AI-Powered Git & Workspace Intelligence Engine",
	Long: `The Eko AI Git Intelligence Engine provides smart code reviews, intent-based status analysis,
automated commit message generation, security scanning, and risk evaluation.`,
}

var aiStatusCmd = &cobra.Command{
	Use:     "status",
	Short:   "Intelligent workspace status with intent & risk analysis",
	PreRunE: requireInitialized,
	RunE: func(cmd *cobra.Command, args []string) error {
		database := db.InitDB()
		defer database.Close()

		var prevPath string
		_ = database.QueryRow("SELECT path FROM snapshots ORDER BY created_at DESC, rowid DESC LIMIT 1").Scan(&prevPath)

		diffs, err := api.BuildDiff(prevPath, ".")
		if err != nil {
			return err
		}

		res, err := mind.PerformAIStatus(context.Background(), diffs)
		if err != nil {
			return err
		}

		fmt.Println("🤖 AI Workspace Status")
		fmt.Println("──────────────────────────────────────────────────")
		fmt.Printf("🎯 Intent: %s\n\n", res.Intent)
		fmt.Println("Files:")
		for f, role := range res.FileRoles {
			fmt.Printf("  ✓ %-24s (%s)\n", f, role)
		}
		if len(res.Concerns) > 0 {
			fmt.Println("\n⚠️ Potential Concerns:")
			for _, c := range res.Concerns {
				fmt.Printf("  • %s\n", c)
			}
		}
		fmt.Printf("\n💡 Suggested Next Step:\n  → %s\n", res.NextStep)
		return nil
	},
}

var aiReviewCmd = &cobra.Command{
	Use:     "review",
	Short:   "Perform automated AI code review & risk evaluation on diff",
	PreRunE: requireInitialized,
	RunE: func(cmd *cobra.Command, args []string) error {
		database := db.InitDB()
		defer database.Close()

		var prevPath string
		_ = database.QueryRow("SELECT path FROM snapshots ORDER BY created_at DESC, rowid DESC LIMIT 1").Scan(&prevPath)

		diffs, err := api.BuildDiff(prevPath, ".")
		if err != nil {
			return err
		}

		res, err := mind.PerformAIReview(context.Background(), diffs, "auto")
		if err != nil {
			return err
		}

		fmt.Println("🤖 AI Code Review & Risk Analysis")
		fmt.Println("──────────────────────────────────────────────────")
		fmt.Printf("Summary: %s\n", res.Summary)
		fmt.Printf("Commit Risk Score: %d/100\n\n", res.RiskScore)

		for _, issue := range res.Issues {
			icon := "ℹ️"
			switch issue.Severity {
			case "CRITICAL":
				icon = "🚨"
			case "HIGH":
				icon = "⚠️"
			case "MEDIUM":
				icon = "🟡"
			case "GOOD":
				icon = "✅"
			}
			fmt.Printf("%s [%s] %s\n   %s\n", icon, issue.Severity, issue.Location, issue.Message)
		}
		return nil
	},
}

var aiSemDiffCmd = &cobra.Command{
	Use:     "semdiff",
	Short:   "Behavioral semantic diff analysis",
	PreRunE: requireInitialized,
	RunE: func(cmd *cobra.Command, args []string) error {
		database := db.InitDB()
		defer database.Close()

		var prevPath string
		_ = database.QueryRow("SELECT path FROM snapshots ORDER BY created_at DESC, rowid DESC LIMIT 1").Scan(&prevPath)

		diffs, err := api.BuildDiff(prevPath, ".")
		if err != nil {
			return err
		}

		fmt.Println(mind.PerformAISemDiff(diffs))
		return nil
	},
}

var aiRiskCmd = &cobra.Command{
	Use:     "risk",
	Short:   "Multi-dimensional commit risk analysis",
	PreRunE: requireInitialized,
	RunE: func(cmd *cobra.Command, args []string) error {
		database := db.InitDB()
		defer database.Close()

		var prevPath string
		_ = database.QueryRow("SELECT path FROM snapshots ORDER BY created_at DESC, rowid DESC LIMIT 1").Scan(&prevPath)

		diffs, err := api.BuildDiff(prevPath, ".")
		if err != nil {
			return err
		}

		_, report := mind.PerformAIRiskAnalysis(diffs)
		fmt.Println(report)
		return nil
	},
}

var aiImpactCmd = &cobra.Command{
	Use:     "impact",
	Short:   "Change impact & subsystem dependency graph",
	PreRunE: requireInitialized,
	RunE: func(cmd *cobra.Command, args []string) error {
		database := db.InitDB()
		defer database.Close()

		var prevPath string
		_ = database.QueryRow("SELECT path FROM snapshots ORDER BY created_at DESC, rowid DESC LIMIT 1").Scan(&prevPath)

		diffs, err := api.BuildDiff(prevPath, ".")
		if err != nil {
			return err
		}

		fmt.Println(mind.PerformAIImpactAnalysis(diffs))
		return nil
	},
}

var aiBisectCmd = &cobra.Command{
	Use:   "bisect [failing-test]",
	Short: "Automated AI regression bug isolation",
	RunE: func(cmd *cobra.Command, args []string) error {
		testTarget := "go test ./..."
		if len(args) > 0 {
			testTarget = args[0]
		}
		fmt.Println(mind.PerformAIBisect(testTarget))
		return nil
	},
}

var aiAskCmd = &cobra.Command{
	Use:   "ask <query>",
	Short: "Query repository architecture memory",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		query := args[0]
		fmt.Println(mind.PerformAIAsk(query))
		return nil
	},
}

func init() {
	aiCmd.AddCommand(aiStatusCmd)
	aiCmd.AddCommand(aiReviewCmd)
	aiCmd.AddCommand(aiSemDiffCmd)
	aiCmd.AddCommand(aiRiskCmd)
	aiCmd.AddCommand(aiImpactCmd)
	aiCmd.AddCommand(aiBisectCmd)
	aiCmd.AddCommand(aiAskCmd)
	rootCmd.AddCommand(aiCmd)
}
