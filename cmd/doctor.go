package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

func newDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check OpenCode scaffold health",
		Long:  "Validates the scaffold structure, required files, and skill frontmatter.",
		RunE: func(cmd *cobra.Command, args []string) error {
			root, _ := os.Getwd()
			fmt.Println()
			bold.Println("  OpenCode Scaffold Doctor")
			fmt.Println()

			checks := []struct {
				name string
				path string
			}{
				{"opencode.json", "opencode.json"},
				{"AGENTS.md", "AGENTS.md"},
				{"Agents directory", ".opencode/agents"},
				{"Skills directory", ".opencode/skills"},
				{"Commands directory", ".opencode/commands"},
				{"LevelDB data dir", ".opencode/data"},
			}

			allOK := true
			for _, c := range checks {
				full := filepath.Join(root, c.path)
				if _, err := os.Stat(full); err == nil {
					fmt.Printf("  %s %s\n", green.Sprint("OK"), c.name)
				} else {
					fmt.Printf("  %s %s (%s missing)\n", red.Sprint("FAIL"), c.name, c.path)
					allOK = false
				}
			}

			fmt.Println()
			bold.Println("  Skill Validation:")
			skillsDir := filepath.Join(root, ".opencode", "skills")
			entries, err := os.ReadDir(skillsDir)
			if err == nil {
				for _, e := range entries {
					if !e.IsDir() {
						continue
					}
					skillFile := filepath.Join(skillsDir, e.Name(), "SKILL.md")
					content, err := os.ReadFile(skillFile)
					if err != nil {
						fmt.Printf("  %s %s (SKILL.md missing)\n", red.Sprint("FAIL"), e.Name())
						allOK = false
						continue
					}
					s := string(content)
					hasName := strings.Contains(s, "name: "+e.Name())
					hasDesc := strings.Contains(s, "description:")
					if hasName && hasDesc {
						fmt.Printf("  %s %s\n", green.Sprint("OK"), e.Name())
					} else {
						issues := []string{}
						if !hasName {
							issues = append(issues, fmt.Sprintf("name must be '%s'", e.Name()))
						}
						if !hasDesc {
							issues = append(issues, "missing description")
						}
						fmt.Printf("  %s %s (%s)\n", yellow.Sprint("WARN"), e.Name(), strings.Join(issues, ", "))
					}
				}
			}

			fmt.Println()
			bold.Println("  Agent Validation:")
			agentsDir := filepath.Join(root, ".opencode", "agents")
			agentEntries, err := os.ReadDir(agentsDir)
			if err == nil {
				for _, e := range agentEntries {
					if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
						continue
					}
					agentFile := filepath.Join(agentsDir, e.Name())
					content, err := os.ReadFile(agentFile)
					if err != nil {
						fmt.Printf("  %s %s (unreadable)\n", red.Sprint("FAIL"), e.Name())
						allOK = false
						continue
					}
					s := string(content)
					hasDesc := strings.Contains(s, "description:")
					hasMode := strings.Contains(s, "mode:")
					if hasDesc && hasMode {
						fmt.Printf("  %s %s\n", green.Sprint("OK"), e.Name())
					} else {
						issues := []string{}
						if !hasDesc {
							issues = append(issues, "missing description")
						}
						if !hasMode {
							issues = append(issues, "missing mode")
						}
						fmt.Printf("  %s %s (%s)\n", yellow.Sprint("WARN"), e.Name(), strings.Join(issues, ", "))
					}
				}
			}

			fmt.Println()
			if allOK {
				color.Green("  All checks passed!")
			} else {
				yellow.Println("  Some checks failed. Run: ocs init --force to fix.")
			}
			fmt.Println()

			if !allOK {
				return fmt.Errorf("scaffold health check failed")
			}
			return nil
		},
	}
}
