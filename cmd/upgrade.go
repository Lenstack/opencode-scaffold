package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

func newUpgradeCmd() *cobra.Command {
	var dryRun bool

	return &cobra.Command{
		Use:   "upgrade",
		Short: "Upgrade scaffold to latest conventions",
		Long:  "Re-runs scaffold with --force on system files only (agents, skills, config). Preserves memory and heuristics.",
		RunE: func(cmd *cobra.Command, args []string) error {
			root, _ := os.Getwd()
			fmt.Println()
			bold.Println("  OpenCode Scaffold Upgrade")
			fmt.Println()

			systemFiles := []string{
				".opencode/agents/orchestrator.md",
				".opencode/agents/planner.md",
				".opencode/agents/architect.md",
				".opencode/agents/tester.md",
				".opencode/agents/reviewer.md",
				".opencode/agents/security.md",
				".opencode/agents/reflector.md",
				".opencode/skills/tdd-workflow/SKILL.md",
				".opencode/skills/code-review/SKILL.md",
				".opencode/skills/security-audit/SKILL.md",
				".opencode/skills/git-workflow/SKILL.md",
				".opencode/skills/api-design/SKILL.md",
				".opencode/skills/observability/SKILL.md",
				".opencode/skills/refactor/SKILL.md",
				".opencode/skills/performance/SKILL.md",
				".opencode/commands/plan.md",
				".opencode/commands/review.md",
				".opencode/commands/ship.md",
				".opencode/commands/reflect.md",
			}

			protected := []string{
				"AGENTS.md",
				"opencode.json",
				".opencode/data/",
				".opencode/plugins/",
			}

			fmt.Println("  The following system files will be updated:")
			for _, f := range systemFiles {
				exists := ""
				if _, err := os.Stat(filepath.Join(root, f)); err == nil {
					exists = " (exists)"
				}
				fmt.Printf("    - %s%s\n", f, exists)
			}
			fmt.Println()
			fmt.Println("  The following files will be PRESERVED:")
			for _, f := range protected {
				fmt.Printf("    - %s\n", f)
			}
			fmt.Println()

			if dryRun {
				cyan.Println("  Dry run — no files changed.")
				return nil
			}

			fmt.Print("  Proceed? [y/N] ")
			var answer string
			fmt.Scanln(&answer)
			if !strings.EqualFold(strings.TrimSpace(answer), "y") {
				fmt.Println("  Upgrade cancelled.")
				return nil
			}

			for _, f := range systemFiles {
				full := filepath.Join(root, f)
				if _, err := os.Stat(full); err == nil {
					if err := os.Remove(full); err != nil {
						fmt.Printf("  %s Failed to remove %s: %v\n", red.Sprint("ERR"), f, err)
					}
				}
			}

			green.Println("\n  System files reset. Run: ocs init to regenerate.")
			return nil
		},
	}
}
