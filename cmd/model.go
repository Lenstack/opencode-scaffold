package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

func newModelCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "model",
		Short: "Manage LLM models for agents",
		Long: `Set, change, or list models used by agents in opencode.json and agent files.

Examples:
  ocs model list                              # Show current model config
  ocs model set anthropic/claude-sonnet-4     # Set default model everywhere
  ocs model set opencode/gpt-5.1-codex        # Set default model everywhere
  ocs model set planner anthropic/claude-haiku-4  # Set model for specific agent
  ocs model small opencode/qwen3.6-plus-free  # Set small model
`,
	}

	cmd.AddCommand(newModelListCmd())
	cmd.AddCommand(newModelSetCmd())
	cmd.AddCommand(newModelSmallCmd())

	return cmd
}

func newModelListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "Show current model configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			root := mustGetwd()
			cfgPath := filepath.Join(root, "opencode.json")

			data, err := os.ReadFile(cfgPath)
			if err != nil {
				return fmt.Errorf("read opencode.json: %w", err)
			}

			var cfg map[string]any
			if err := json.Unmarshal(data, &cfg); err != nil {
				return fmt.Errorf("parse opencode.json: %w", err)
			}

			fmt.Println()
			bold := color.New(color.Bold)
			bold.Println("  Model Configuration:")
			fmt.Println()

			if model, ok := cfg["model"].(string); ok {
				fmt.Printf("    Default:     %s\n", color.GreenString(model))
			} else {
				fmt.Printf("    Default:     %s\n", color.YellowString("not set"))
			}

			if small, ok := cfg["small_model"].(string); ok {
				fmt.Printf("    Small:       %s\n", color.GreenString(small))
			} else {
				fmt.Printf("    Small:       %s\n", color.YellowString("not set"))
			}

			// Show agent markdown files
			agentsDir := filepath.Join(root, ".opencode", "agents")
			if entries, err := os.ReadDir(agentsDir); err == nil {
				fmt.Println()
				fmt.Println("    Agent files (.opencode/agents/):")
				for _, e := range entries {
					if !strings.HasSuffix(e.Name(), ".md") {
						continue
					}
					fpath := filepath.Join(agentsDir, e.Name())
					content, _ := os.ReadFile(fpath)
					model := extractModelFromFrontmatter(string(content))
					name := strings.TrimSuffix(e.Name(), ".md")
					fmt.Printf("      %-20s %s\n", name, model)
				}
			}

			if agents, ok := cfg["agent"].(map[string]any); ok {
				fmt.Println()
				fmt.Println("    opencode.json overrides:")
				for name, val := range agents {
					if agent, ok := val.(map[string]any); ok {
						if m, ok := agent["model"].(string); ok {
							fmt.Printf("      %-20s %s\n", name, m)
						}
					}
				}
			}

			fmt.Println()
			return nil
		},
	}
}

func newModelSetCmd() *cobra.Command {
	var agentFlag string

	cmd := &cobra.Command{
		Use:   "set <model> [agent]",
		Short: "Set the default model or model for a specific agent",
		Long: `Set the model for the default agent or a specific named agent.
Updates both opencode.json and agent markdown files.

Examples:
  ocs model set anthropic/claude-sonnet-4-20250514
  ocs model set opencode/gpt-5.1-codex
  ocs model set anthropic/claude-haiku-4-20250514 --agent planner
  ocs model set openai/gpt-4o --agent orchestrator
`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			model := args[0]
			root := mustGetwd()
			cfgPath := filepath.Join(root, "opencode.json")

			// Support both --agent flag and positional [agent] argument
			targetAgent := agentFlag
			if targetAgent == "" && len(args) > 1 {
				targetAgent = args[1]
			}

			data, err := os.ReadFile(cfgPath)
			if err != nil {
				return fmt.Errorf("read opencode.json: %w", err)
			}

			var cfg map[string]any
			if err := json.Unmarshal(data, &cfg); err != nil {
				return fmt.Errorf("parse opencode.json: %w", err)
			}

			if targetAgent != "" {
				// Set model for specific agent in opencode.json
				agents, ok := cfg["agent"].(map[string]any)
				if !ok {
					agents = make(map[string]any)
					cfg["agent"] = agents
				}

				agentCfg, ok := agents[targetAgent].(map[string]any)
				if !ok {
					agentCfg = make(map[string]any)
					agents[targetAgent] = agentCfg
				}

				agentCfg["model"] = model

				// Update agent markdown file
				agentFile := filepath.Join(root, ".opencode", "agents", targetAgent+".md")
				if _, err := os.Stat(agentFile); err == nil {
					if err := updateModelInFrontmatter(agentFile, model); err != nil {
						fmt.Printf("  %s Failed to update %s: %v\n", color.YellowString("WARN"), agentFile, err)
					} else {
						fmt.Printf("  Updated %s\n", color.GreenString(targetAgent+".md"))
					}
				}

				fmt.Printf("  Set model for agent %q: %s\n", color.GreenString(targetAgent), model)
			} else {
				// Set default model in opencode.json
				cfg["model"] = model

				// Update ALL agent markdown files
				agentsDir := filepath.Join(root, ".opencode", "agents")
				if entries, err := os.ReadDir(agentsDir); err == nil {
					for _, e := range entries {
						if !strings.HasSuffix(e.Name(), ".md") {
							continue
						}
						fpath := filepath.Join(agentsDir, e.Name())
						if err := updateModelInFrontmatter(fpath, model); err != nil {
							fmt.Printf("  %s Failed to update %s: %v\n", color.YellowString("WARN"), e.Name(), err)
						} else {
							fmt.Printf("  Updated %s\n", color.GreenString(e.Name()))
						}
					}
				}

				fmt.Printf("  Set default model: %s\n", color.GreenString(model))
			}

			out, err := json.MarshalIndent(cfg, "", "  ")
			if err != nil {
				return fmt.Errorf("marshal config: %w", err)
			}

			if err := os.WriteFile(cfgPath, append(out, '\n'), 0644); err != nil {
				return fmt.Errorf("write opencode.json: %w", err)
			}

			fmt.Println()
			return nil
		},
	}

	cmd.Flags().StringVarP(&agentFlag, "agent", "a", "", "Set model for a specific agent")

	return cmd
}

func newModelSmallCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "small <model>",
		Short: "Set the small model (used for fast/cheap operations)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			model := args[0]
			root := mustGetwd()
			cfgPath := filepath.Join(root, "opencode.json")

			data, err := os.ReadFile(cfgPath)
			if err != nil {
				return fmt.Errorf("read opencode.json: %w", err)
			}

			var cfg map[string]any
			if err := json.Unmarshal(data, &cfg); err != nil {
				return fmt.Errorf("parse opencode.json: %w", err)
			}

			cfg["small_model"] = model
			fmt.Printf("  Set small model: %s\n", color.GreenString(model))

			out, err := json.MarshalIndent(cfg, "", "  ")
			if err != nil {
				return fmt.Errorf("marshal config: %w", err)
			}

			if err := os.WriteFile(cfgPath, append(out, '\n'), 0644); err != nil {
				return fmt.Errorf("write opencode.json: %w", err)
			}

			fmt.Println()
			return nil
		},
	}
}

// modelRe matches "model: <value>" in YAML frontmatter
var modelRe = regexp.MustCompile(`(?m)^model:\s*.*$`)

func updateModelInFrontmatter(filePath, model string) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	s := string(content)

	// Check if frontmatter has a model line
	if modelRe.MatchString(s) {
		s = modelRe.ReplaceAllString(s, "model: "+model)
	} else {
		// Insert model after description line in frontmatter
		s = strings.Replace(s, "\ndescription:", "\nmodel: "+model+"\ndescription:", 1)
	}

	return os.WriteFile(filePath, []byte(s), 0644)
}

func extractModelFromFrontmatter(content string) string {
	lines := strings.Split(content, "\n")
	inFrontmatter := false
	for _, line := range lines {
		if strings.TrimSpace(line) == "---" {
			if inFrontmatter {
				break
			}
			inFrontmatter = true
			continue
		}
		if inFrontmatter && strings.HasPrefix(line, "model:") {
			return strings.TrimSpace(strings.TrimPrefix(line, "model:"))
		}
	}
	return "(inherited)"
}
