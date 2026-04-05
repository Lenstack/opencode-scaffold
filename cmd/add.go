package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	tmpl "github.com/Lenstack/opencode-scaffold/internal/domain/template"
)

func newAddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add <component>",
		Short: "Add a component to an existing scaffold",
		Long: `Add specific components to an already-scaffolded project.

Available components:
  agent <name>     Create a new subagent
  skill <name>     Create a new skill
  command <name>   Create a new custom command

Examples:
  ocs add agent documentation
  ocs add skill database-migrations
  ocs add command deploy
`,
	}

	cmd.AddCommand(newAddAgentCmd())
	cmd.AddCommand(newAddSkillCmd())
	cmd.AddCommand(newAddCommandCmd())

	return cmd
}

func newAddAgentCmd() *cobra.Command {
	var (
		mode        string
		description string
		model       string
		steps       int
	)

	cmd := &cobra.Command{
		Use:   "agent <name>",
		Short: "Create a new agent",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := strings.ToLower(args[0])
			root, _ := os.Getwd()
			path := filepath.Join(root, ".opencode", "agents", name+".md")

			if _, err := os.Stat(path); err == nil {
				return fmt.Errorf("agent %s already exists at %s", name, path)
			}

			if description == "" {
				description = name + " agent"
			}

			content := fmt.Sprintf(`---
description: %s
mode: %s
model: %s
temperature: 0.1
steps: %d
permission:
  edit: ask
  bash:
    "*": ask
---

# %s Agent

<Describe what this agent does and when it's invoked.>

## When to Use

<Describe the conditions under which this agent should be invoked.>

## Process

1. <Step 1>
2. <Step 2>
3. <Step 3>

## Output

<Describe the expected output format.>
`, description, mode, model, steps, strings.Title(name))

			if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
				return err
			}
			if err := os.WriteFile(path, []byte(content), 0644); err != nil {
				return err
			}

			color.Green("Created .opencode/agents/%s.md", name)
			fmt.Printf("   Invoke with: @%s\n", name)
			return nil
		},
	}

	cmd.Flags().StringVar(&mode, "mode", "subagent", "Agent mode: primary, subagent, or all")
	cmd.Flags().StringVar(&description, "description", "", "Agent description (shown in skill selection)")
	cmd.Flags().StringVar(&model, "model", "anthropic/claude-sonnet-4-20250514", "Model to use")
	cmd.Flags().IntVar(&steps, "steps", 15, "Max steps before forced response")

	return cmd
}

func newAddSkillCmd() *cobra.Command {
	var description string

	cmd := &cobra.Command{
		Use:   "skill <name>",
		Short: "Create a new agent skill",
		Long: `Create a new skill in .opencode/skills/<name>/SKILL.md

Skill names must be lowercase alphanumeric with hyphens.
The directory name MUST match the 'name' field in frontmatter.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := strings.ToLower(args[0])

			if err := tmpl.ValidateSkillName(name); err != nil {
				return err
			}

			if description == "" {
				description = "<describe when to use this skill>"
			}
			if len(description) > 1024 {
				return fmt.Errorf("description must be <= 1024 characters (got %d)", len(description))
			}

			root, _ := os.Getwd()
			dir := filepath.Join(root, ".opencode", "skills", name)
			path := filepath.Join(dir, "SKILL.md")

			if _, err := os.Stat(path); err == nil {
				return fmt.Errorf("skill %s already exists", name)
			}

			content := fmt.Sprintf(`---
name: %s
description: %s
license: MIT
compatibility: opencode
---

# %s Skill

## When to Use This Skill

<Describe when an agent should load this skill.>

## Guidelines

<Main content — what agents should do when using this skill.>

## Patterns

`+"```"+`
<Code examples or templates>
`+"```"+`

## Anti-patterns

Never do:
- <Anti-pattern 1>
- <Anti-pattern 2>
`, name, description, strings.Title(strings.ReplaceAll(name, "-", " ")))

			if err := os.MkdirAll(dir, 0755); err != nil {
				return err
			}
			if err := os.WriteFile(path, []byte(content), 0644); err != nil {
				return err
			}

			color.Green("Created .opencode/skills/%s/SKILL.md", name)
			fmt.Printf("   Agents load it with: @skill %s\n", name)
			return nil
		},
	}

	cmd.Flags().StringVar(&description, "description", "", "Skill description (1-1024 chars, required)")
	_ = cmd.MarkFlagRequired("description")

	return cmd
}

func newAddCommandCmd() *cobra.Command {
	var (
		agent       string
		description string
	)

	cmd := &cobra.Command{
		Use:   "command <name>",
		Short: "Create a new custom slash command",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := strings.ToLower(args[0])
			root, _ := os.Getwd()
			path := filepath.Join(root, ".opencode", "commands", name+".md")

			if _, err := os.Stat(path); err == nil {
				return fmt.Errorf("command %s already exists", name)
			}

			if description == "" {
				description = name + " command"
			}

			var agentLine string
			if agent != "" {
				agentLine = fmt.Sprintf("agent: %s\n", agent)
			}

			content := fmt.Sprintf(`---
description: %s
%s---

<Add instructions here. This becomes the prompt when /%s is run.>

You can:
- Reference files with @path/to/file
- Invoke subagents with @agent-name
- Load skills with @skill skill-name
- Run bash commands inline
`, description, agentLine, name)

			if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
				return err
			}
			if err := os.WriteFile(path, []byte(content), 0644); err != nil {
				return err
			}

			color.Green("Created .opencode/commands/%s.md", name)
			fmt.Printf("   Run it in OpenCode with: /%s\n", name)
			return nil
		},
	}

	cmd.Flags().StringVar(&agent, "agent", "", "Agent to execute this command")
	cmd.Flags().StringVar(&description, "description", "", "Command description")

	return cmd
}
