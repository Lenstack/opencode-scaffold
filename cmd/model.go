package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

func newModelCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "model",
		Short: "Manage LLM models for agents",
		Long: `Set, change, or list models used by agents in opencode.json.

Examples:
  ocs model list                              # Show current model config
  ocs model set anthropic/claude-sonnet-4     # Set default model
  ocs model set opencode/gpt-5.1-codex        # Set default model
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

			if agents, ok := cfg["agent"].(map[string]any); ok {
				fmt.Println()
				fmt.Println("    Agent overrides:")
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

			data, err := os.ReadFile(cfgPath)
			if err != nil {
				return fmt.Errorf("read opencode.json: %w", err)
			}

			var cfg map[string]any
			if err := json.Unmarshal(data, &cfg); err != nil {
				return fmt.Errorf("parse opencode.json: %w", err)
			}

			if agentFlag != "" {
				// Set model for specific agent
				agents, ok := cfg["agent"].(map[string]any)
				if !ok {
					agents = make(map[string]any)
					cfg["agent"] = agents
				}

				agentCfg, ok := agents[agentFlag].(map[string]any)
				if !ok {
					agentCfg = make(map[string]any)
					agents[agentFlag] = agentCfg
				}

				agentCfg["model"] = model
				fmt.Printf("  Set model for agent %q: %s\n", color.GreenString(agentFlag), model)
			} else {
				// Set default model
				cfg["model"] = model
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
