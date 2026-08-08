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

var aiOwnersCmd = &cobra.Command{
	Use:   "owners <file-path>",
	Short: "Identify code maintainers & recommended PR reviewers",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		targetPath := args[0]
		fmt.Println(mind.PerformAIOwners(targetPath))
		return nil
	},
}

var aiNextCmd = &cobra.Command{
	Use:   "next",
	Short: "AI-driven task & issue recommendation engine",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println(mind.PerformAINext())
		return nil
	},
}

var aiSecurityCmd = &cobra.Command{
	Use:     "security",
	Short:   "AI security & hardcoded secret scanner",
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

		fmt.Println(mind.PerformAISecurity(diffs))
		return nil
	},
}

var aiGateCmd = &cobra.Command{
	Use:     "gate",
	Short:   "AI pre-commit quality gate evaluation",
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

		_, report := mind.PerformAIGate(diffs)
		fmt.Println(report)
		return nil
	},
}

var aiExplainCmd = &cobra.Command{
	Use:   "explain <file-path>",
	Short: "Explain file purpose, architecture role & design decisions",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		filePath := args[0]
		fmt.Println(mind.PerformAIExplain(filePath, "", ""))
		return nil
	},
}

var aiTestCmd = &cobra.Command{
	Use:     "test",
	Short:   "Generate unit & integration testing strategy for diff",
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

		fmt.Println(mind.PerformAITest(diffs))
		return nil
	},
}

var aiConflictCmd = &cobra.Command{
	Use:   "conflict [file-path]",
	Short: "AI merge conflict resolution & intent guide",
	RunE: func(cmd *cobra.Command, args []string) error {
		filePath := "conflicted_file.go"
		if len(args) > 0 {
			filePath = args[0]
		}
		fmt.Println(mind.PerformAIConflict(filePath, ""))
		return nil
	},
}

var aiPRCmd = &cobra.Command{
	Use:     "pr [base-branch]",
	Short:   "Generate GitHub Pull Request description from diff",
	PreRunE: requireInitialized,
	RunE: func(cmd *cobra.Command, args []string) error {
		baseBranch := "main"
		if len(args) > 0 {
			baseBranch = args[0]
		}

		database := db.InitDB()
		defer database.Close()

		var prevPath string
		_ = database.QueryRow("SELECT path FROM snapshots ORDER BY created_at DESC, rowid DESC LIMIT 1").Scan(&prevPath)

		diffs, err := api.BuildDiff(prevPath, ".")
		if err != nil {
			return err
		}

		fmt.Println(mind.PerformAIPR(baseBranch, diffs))
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
	aiCmd.AddCommand(aiOwnersCmd)
	aiCmd.AddCommand(aiNextCmd)
	aiCmd.AddCommand(aiSecurityCmd)
	aiCmd.AddCommand(aiGateCmd)
	aiCmd.AddCommand(aiExplainCmd)
	aiCmd.AddCommand(aiTestCmd)
	aiCmd.AddCommand(aiConflictCmd)
	aiCmd.AddCommand(aiPRCmd)
	rootCmd.AddCommand(aiCmd)
}
