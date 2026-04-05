package cmd

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	version = "2.0.0"
	bold    = color.New(color.Bold)
	green   = color.New(color.FgGreen, color.Bold)
	cyan    = color.New(color.FgCyan)
	yellow  = color.New(color.FgYellow)
	red     = color.New(color.FgRed)
)

var rootCmd = &cobra.Command{
	Use:   "ocs",
	Short: "OpenCode Scaffold — production-grade project scaffolder for AI agent workflows",
	Long: `
  OpenCode Scaffold (ocs)  v` + version + `
  Scaffolds projects with AI-agent workflows
  following official OpenCode best practices

  Creates .opencode/ configuration, agents, skills,
  memory structures, and AGENTS.md tuned to your stack.

  Stack support: Go+Encore, Go+Fiber, Go+Gin, Go+Chi,
                 Node+Next.js, Python+FastAPI, Python+Django,
                 Rust+Axum, Generic.
`,
	SilenceUsage: true,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(newInitCmd())
	rootCmd.AddCommand(newAddCmd())
	rootCmd.AddCommand(newDoctorCmd())
	rootCmd.AddCommand(newListCmd())
	rootCmd.AddCommand(newUpgradeCmd())
	rootCmd.AddCommand(newCompletionCmd())
	rootCmd.AddCommand(newDiscoverCmd())
	rootCmd.AddCommand(newSpecCmd())
	rootCmd.AddCommand(newMemoryCmd())
	rootCmd.AddCommand(newSessionCmd())
	rootCmd.AddCommand(newSkillCmd())
	rootCmd.AddCommand(newServeCmd())
	rootCmd.AddCommand(newPushCmd())
	rootCmd.AddCommand(newPullCmd())
	rootCmd.AddCommand(newAuthCmd())
	rootCmd.AddCommand(newBackupCmd())
	rootCmd.AddCommand(newStatusCmd())
	rootCmd.AddCommand(newTemplateCmd())
	rootCmd.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Show version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("ocs %s\n", version)
		},
	})
}
