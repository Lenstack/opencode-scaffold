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
	// Core
	rootCmd.AddCommand(newInitCmd())
	rootCmd.AddCommand(newDiscoverCmd())
	rootCmd.AddCommand(newDoctorCmd())
	rootCmd.AddCommand(newUpgradeCmd())
	rootCmd.AddCommand(newInfoCmd())
	rootCmd.AddCommand(newVersionCmd())
	rootCmd.AddCommand(newCompletionCmd())

	// Entity management
	rootCmd.AddCommand(newAgentCmd())
	rootCmd.AddCommand(newSkillCmd())
	rootCmd.AddCommand(newCommandCmd())
	rootCmd.AddCommand(newPluginCmd())
	rootCmd.AddCommand(newMemoryCmd())
	rootCmd.AddCommand(newSpecCmd())

	// Templates
	rootCmd.AddCommand(newTemplateCmd())

	// Config
	rootCmd.AddCommand(newConfigCmd())

	// Hub
	rootCmd.AddCommand(newHubCmd())
	rootCmd.AddCommand(newAuthCmd())
	rootCmd.AddCommand(newBackupCmd())

	// Legacy aliases
	rootCmd.AddCommand(newAddCmd())
	rootCmd.AddCommand(newListCmd())
	rootCmd.AddCommand(newSessionCmd())
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("ocs %s\n", version)
		},
	}
}

func newInfoCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "info",
		Short: "Show project info",
		RunE: func(cmd *cobra.Command, args []string) error {
			root := mustGetwd()

			fmt.Println()
			bold := color.New(color.Bold)
			bold.Println("  Project Info:")
			fmt.Println()

			// Stack info
			fmt.Printf("  Root:      %s\n", root)

			// Check for opencode.json
			if _, err := os.Stat(root + "/opencode.json"); err == nil {
				fmt.Printf("  Config:    %s\n", color.GreenString("opencode.json found"))
			} else {
				fmt.Printf("  Config:    %s\n", color.RedString("opencode.json missing"))
			}

			// Check for AGENTS.md
			if _, err := os.Stat(root + "/AGENTS.md"); err == nil {
				fmt.Printf("  Rules:     %s\n", color.GreenString("AGENTS.md found"))
			} else {
				fmt.Printf("  Rules:     %s\n", color.RedString("AGENTS.md missing"))
			}

			// Check for .opencode
			if _, err := os.Stat(root + "/.opencode"); err == nil {
				fmt.Printf("  Scaffold:  %s\n", color.GreenString(".opencode/ found"))
			} else {
				fmt.Printf("  Scaffold:  %s\n", color.RedString(".opencode/ missing"))
			}

			// Check for LevelDB
			if _, err := os.Stat(root + "/.opencode/data"); err == nil {
				fmt.Printf("  Database:  %s\n", color.GreenString("LevelDB found"))
			} else {
				fmt.Printf("  Database:  %s\n", color.RedString("LevelDB missing"))
			}

			// Check for git
			if _, err := os.Stat(root + "/.git"); err == nil {
				fmt.Printf("  Git:       %s\n", color.GreenString("git repo found"))
			} else {
				fmt.Printf("  Git:       %s\n", color.RedString("not a git repo"))
			}

			fmt.Println()
			return nil
		},
	}
}
