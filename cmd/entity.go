package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/Lenstack/opencode-scaffold/internal/engine/config"
)

func newAgentCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agent",
		Short: "Manage agents",
		Long: `Create, edit, and manage OpenCode agents.

Examples:
  ocs agent create reviewer --mode subagent --description "Code review agent"
  ocs agent list
  ocs agent show reviewer
  ocs agent edit reviewer
  ocs agent remove reviewer
  ocs agent rename reviewer code-reviewer
  ocs agent clone reviewer reviewer-v2
`,
	}

	cmd.AddCommand(newAgentCreateCmd())
	cmd.AddCommand(newAgentListCmd())
	cmd.AddCommand(newAgentShowCmd())
	cmd.AddCommand(newAgentEditCmd())
	cmd.AddCommand(newAgentRemoveCmd())
	cmd.AddCommand(newAgentRenameCmd())
	cmd.AddCommand(newAgentCloneCmd())

	return cmd
}

func newAgentCreateCmd() *cobra.Command {
	var mode, description, model string
	var steps int
	var temperature float64

	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a new agent",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			root := mustGetwd()
			path := filepath.Join(root, ".opencode", "agents", name+".md")

			if _, err := os.Stat(path); err == nil {
				return fmt.Errorf("agent %s already exists", name)
			}

			if description == "" {
				description = name + " agent"
			}

			content := fmt.Sprintf(`---
description: %s
mode: %s
model: %s
temperature: %.2f
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
`, description, mode, model, temperature, steps, titleCase(name))

			if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
				return err
			}
			if err := os.WriteFile(path, []byte(content), 0644); err != nil {
				return err
			}

			// Track in config DB
			d, err := openEngine()
			if err == nil {
				defer d.Close()
				tracker := config.NewTracker(d, root)
				tracker.TrackConfig(filepath.Join(".opencode", "agents", name+".md"), content, "user", "cli")
			}

			color.Green("Created agent: %s", name)
			fmt.Printf("   Path: %s\n", path)
			fmt.Printf("   Mode: %s\n", mode)
			return nil
		},
	}

	cmd.Flags().StringVar(&mode, "mode", "subagent", "Agent mode: primary, subagent, all")
	cmd.Flags().StringVar(&description, "description", "", "Agent description")
	cmd.Flags().StringVar(&model, "model", "anthropic/claude-sonnet-4-20250514", "Model to use")
	cmd.Flags().IntVar(&steps, "steps", 15, "Max steps before forced response")
	cmd.Flags().Float64Var(&temperature, "temperature", 0.1, "Temperature (0.0-1.0)")

	return cmd
}

func newAgentListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all agents",
		RunE: func(cmd *cobra.Command, args []string) error {
			root := mustGetwd()
			dir := filepath.Join(root, ".opencode", "agents")
			entries, err := os.ReadDir(dir)
			if err != nil {
				return fmt.Errorf("no agents found — run: ocs init")
			}

			fmt.Println()
			bold := color.New(color.Bold)
			bold.Println("  Agents:")
			fmt.Println()

			for _, e := range entries {
				if !strings.HasSuffix(e.Name(), ".md") {
					continue
				}
				name := strings.TrimSuffix(e.Name(), ".md")
				content, _ := os.ReadFile(filepath.Join(dir, e.Name()))
				s := string(content)

				desc := extractFrontmatterField(s, "description")
				mode := extractFrontmatterField(s, "mode")

				fmt.Printf("  @%-20s %s\n", name, desc)
				fmt.Printf("    mode: %s\n", mode)
				fmt.Println()
			}

			return nil
		},
	}
}

func newAgentShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <name>",
		Short: "Show agent details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root := mustGetwd()
			path := filepath.Join(root, ".opencode", "agents", args[0]+".md")
			content, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("agent %s not found", args[0])
			}

			s := string(content)
			desc := extractFrontmatterField(s, "description")
			mode := extractFrontmatterField(s, "mode")
			model := extractFrontmatterField(s, "model")

			fmt.Println()
			bold := color.New(color.Bold)
			bold.Printf("  Agent: %s\n", args[0])
			fmt.Println()
			fmt.Printf("  Description: %s\n", desc)
			fmt.Printf("  Mode:        %s\n", mode)
			fmt.Printf("  Model:       %s\n", model)
			fmt.Printf("  Path:        %s\n", path)
			fmt.Println()
			fmt.Println(s)
			return nil
		},
	}
}

func newAgentEditCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "edit <name>",
		Short: "Edit an agent file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root := mustGetwd()
			path := filepath.Join(root, ".opencode", "agents", args[0]+".md")

			if _, err := os.Stat(path); os.IsNotExist(err) {
				return fmt.Errorf("agent %s not found", args[0])
			}

			editor := os.Getenv("EDITOR")
			if editor == "" {
				editor = "vi"
			}

			c := exec.Command(editor, path)
			c.Stdin = os.Stdin
			c.Stdout = os.Stdout
			c.Stderr = os.Stderr
			return c.Run()
		},
	}
}

func newAgentRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove <name>",
		Short: "Remove an agent",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root := mustGetwd()
			path := filepath.Join(root, ".opencode", "agents", args[0]+".md")

			if _, err := os.Stat(path); os.IsNotExist(err) {
				return fmt.Errorf("agent %s not found", args[0])
			}

			if err := os.Remove(path); err != nil {
				return err
			}

			color.Green("Removed agent: %s", args[0])
			return nil
		},
	}
}

func newAgentRenameCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rename <old> <new>",
		Short: "Rename an agent",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			root := mustGetwd()
			oldPath := filepath.Join(root, ".opencode", "agents", args[0]+".md")
			newPath := filepath.Join(root, ".opencode", "agents", args[1]+".md")

			if _, err := os.Stat(oldPath); os.IsNotExist(err) {
				return fmt.Errorf("agent %s not found", args[0])
			}

			if _, err := os.Stat(newPath); err == nil {
				return fmt.Errorf("agent %s already exists", args[1])
			}

			if err := os.Rename(oldPath, newPath); err != nil {
				return err
			}

			color.Green("Renamed agent: %s → %s", args[0], args[1])
			return nil
		},
	}
}

func newAgentCloneCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "clone <source> <new>",
		Short: "Clone an agent",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			root := mustGetwd()
			srcPath := filepath.Join(root, ".opencode", "agents", args[0]+".md")
			dstPath := filepath.Join(root, ".opencode", "agents", args[1]+".md")

			if _, err := os.Stat(srcPath); os.IsNotExist(err) {
				return fmt.Errorf("agent %s not found", args[0])
			}

			if _, err := os.Stat(dstPath); err == nil {
				return fmt.Errorf("agent %s already exists", args[1])
			}

			content, err := os.ReadFile(srcPath)
			if err != nil {
				return err
			}

			// Update description in cloned agent
			s := string(content)
			s = strings.Replace(s, fmt.Sprintf("description: %s", extractFrontmatterField(s, "description")), fmt.Sprintf("description: %s (clone of %s)", extractFrontmatterField(s, "description"), args[0]), 1)

			if err := os.WriteFile(dstPath, []byte(s), 0644); err != nil {
				return err
			}

			color.Green("Cloned agent: %s → %s", args[0], args[1])
			return nil
		},
	}
}

func extractFrontmatterField(content, field string) string {
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
		if inFrontmatter {
			if strings.HasPrefix(line, field+":") {
				return strings.TrimSpace(strings.TrimPrefix(line, field+":"))
			}
		}
	}
	return ""
}

func newCommandCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "command",
		Short: "Manage custom slash commands",
		Long: `Create, edit, and manage OpenCode custom slash commands.

Examples:
  ocs command create deploy --description "Deploy to production"
  ocs command list
  ocs command show deploy
  ocs command edit deploy
  ocs command remove deploy
  ocs command rename deploy deploy-prod
`,
	}

	cmd.AddCommand(newCommandCreateCmd())
	cmd.AddCommand(newCommandListCmd())
	cmd.AddCommand(newCommandShowCmd())
	cmd.AddCommand(newCommandEditCmd())
	cmd.AddCommand(newCommandRemoveCmd())
	cmd.AddCommand(newCommandRenameCmd())

	return cmd
}

func newCommandCreateCmd() *cobra.Command {
	var description, agent string

	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a new command",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			root := mustGetwd()
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

			// Track in config DB
			d, err := openEngine()
			if err == nil {
				defer d.Close()
				tracker := config.NewTracker(d, root)
				tracker.TrackConfig(filepath.Join(".opencode", "commands", name+".md"), content, "user", "cli")
			}

			color.Green("Created command: /%s", name)
			fmt.Printf("   Path: %s\n", path)
			return nil
		},
	}

	cmd.Flags().StringVar(&description, "description", "", "Command description")
	cmd.Flags().StringVar(&agent, "agent", "", "Agent to execute this command")

	return cmd
}

func newCommandListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all commands",
		RunE: func(cmd *cobra.Command, args []string) error {
			root := mustGetwd()
			dir := filepath.Join(root, ".opencode", "commands")
			entries, err := os.ReadDir(dir)
			if err != nil {
				return fmt.Errorf("no commands found — run: ocs init")
			}

			fmt.Println()
			bold := color.New(color.Bold)
			bold.Println("  Commands:")
			fmt.Println()

			for _, e := range entries {
				if !strings.HasSuffix(e.Name(), ".md") {
					continue
				}
				name := strings.TrimSuffix(e.Name(), ".md")
				content, _ := os.ReadFile(filepath.Join(dir, e.Name()))
				s := string(content)
				desc := extractFrontmatterField(s, "description")

				fmt.Printf("  /%-20s %s\n", name, desc)
			}
			fmt.Println()
			return nil
		},
	}
}

func newCommandShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <name>",
		Short: "Show command details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root := mustGetwd()
			path := filepath.Join(root, ".opencode", "commands", args[0]+".md")
			content, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("command %s not found", args[0])
			}

			s := string(content)
			desc := extractFrontmatterField(s, "description")
			agent := extractFrontmatterField(s, "agent")

			fmt.Println()
			bold := color.New(color.Bold)
			bold.Printf("  Command: /%s\n", args[0])
			fmt.Println()
			fmt.Printf("  Description: %s\n", desc)
			fmt.Printf("  Agent:       %s\n", agent)
			fmt.Printf("  Path:        %s\n", path)
			fmt.Println()
			fmt.Println(s)
			return nil
		},
	}
}

func newCommandEditCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "edit <name>",
		Short: "Edit a command file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root := mustGetwd()
			path := filepath.Join(root, ".opencode", "commands", args[0]+".md")

			if _, err := os.Stat(path); os.IsNotExist(err) {
				return fmt.Errorf("command %s not found", args[0])
			}

			editor := os.Getenv("EDITOR")
			if editor == "" {
				editor = "vi"
			}

			c := exec.Command(editor, path)
			c.Stdin = os.Stdin
			c.Stdout = os.Stdout
			c.Stderr = os.Stderr
			return c.Run()
		},
	}
}

func newCommandRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove <name>",
		Short: "Remove a command",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root := mustGetwd()
			path := filepath.Join(root, ".opencode", "commands", args[0]+".md")

			if _, err := os.Stat(path); os.IsNotExist(err) {
				return fmt.Errorf("command %s not found", args[0])
			}

			if err := os.Remove(path); err != nil {
				return err
			}

			color.Green("Removed command: /%s", args[0])
			return nil
		},
	}
}

func newCommandRenameCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rename <old> <new>",
		Short: "Rename a command",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			root := mustGetwd()
			oldPath := filepath.Join(root, ".opencode", "commands", args[0]+".md")
			newPath := filepath.Join(root, ".opencode", "commands", args[1]+".md")

			if _, err := os.Stat(oldPath); os.IsNotExist(err) {
				return fmt.Errorf("command %s not found", args[0])
			}

			if _, err := os.Stat(newPath); err == nil {
				return fmt.Errorf("command %s already exists", args[1])
			}

			if err := os.Rename(oldPath, newPath); err != nil {
				return err
			}

			color.Green("Renamed command: /%s → /%s", args[0], args[1])
			return nil
		},
	}
}

func newPluginCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "plugin",
		Short: "Manage plugins",
		Long: `Install, enable, and manage OpenCode plugins.

Examples:
  ocs plugin list
  ocs plugin install env-protection
  ocs plugin show env-protection
  ocs plugin disable env-protection
  ocs plugin enable env-protection
  ocs plugin remove env-protection
`,
	}

	cmd.AddCommand(newPluginListCmd())
	cmd.AddCommand(newPluginInstallCmd())
	cmd.AddCommand(newPluginShowCmd())
	cmd.AddCommand(newPluginEnableCmd())
	cmd.AddCommand(newPluginDisableCmd())
	cmd.AddCommand(newPluginRemoveCmd())
	cmd.AddCommand(newPluginUpgradeCmd())

	return cmd
}

func newPluginListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List installed plugins",
		RunE: func(cmd *cobra.Command, args []string) error {
			root := mustGetwd()
			dir := filepath.Join(root, ".opencode", "plugins")
			entries, err := os.ReadDir(dir)
			if err != nil {
				fmt.Println("  No plugins directory found.")
				return nil
			}

			fmt.Println()
			bold := color.New(color.Bold)
			bold.Println("  Plugins:")
			fmt.Println()

			for _, e := range entries {
				if e.IsDir() || !strings.HasSuffix(e.Name(), ".js") {
					continue
				}
				name := strings.TrimSuffix(e.Name(), ".js")
				info, _ := e.Info()
				fmt.Printf("  %-25s size: %-8s modified: %s\n", name, formatSize(info.Size()), info.ModTime().Format("2006-01-02"))
			}
			fmt.Println()
			return nil
		},
	}
}

func newPluginInstallCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "install <name>",
		Short: "Install a plugin",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root := mustGetwd()
			path := filepath.Join(root, ".opencode", "plugins", args[0]+".js")

			if _, err := os.Stat(path); err == nil {
				return fmt.Errorf("plugin %s already installed", args[0])
			}

			// Create default plugin template
			content := fmt.Sprintf(`// %s plugin
// Add your plugin logic here
export const %s = async () => {
  return {
    // Add event handlers here
  }
}
`, args[0], titleCase(strings.ReplaceAll(args[0], "-", "")))

			if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
				return err
			}
			if err := os.WriteFile(path, []byte(content), 0644); err != nil {
				return err
			}

			color.Green("Installed plugin: %s", args[0])
			return nil
		},
	}
}

func newPluginShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <name>",
		Short: "Show plugin details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root := mustGetwd()
			path := filepath.Join(root, ".opencode", "plugins", args[0]+".js")
			content, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("plugin %s not found", args[0])
			}

			info, _ := os.Stat(path)
			fmt.Println()
			bold := color.New(color.Bold)
			bold.Printf("  Plugin: %s\n", args[0])
			fmt.Println()
			fmt.Printf("  Path:        %s\n", path)
			fmt.Printf("  Size:        %s\n", formatSize(info.Size()))
			fmt.Printf("  Modified:    %s\n", info.ModTime().Format("2006-01-02 15:04:05"))
			fmt.Println()
			fmt.Println(string(content))
			return nil
		},
	}
}

func newPluginEnableCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "enable <name>",
		Short: "Enable a plugin",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root := mustGetwd()
			path := filepath.Join(root, ".opencode", "plugins", args[0]+".js")
			disabledPath := path + ".disabled"

			if _, err := os.Stat(path); os.IsNotExist(err) {
				if _, err := os.Stat(disabledPath); err == nil {
					return os.Rename(disabledPath, path)
				}
				return fmt.Errorf("plugin %s not found", args[0])
			}

			color.Green("Plugin %s is already enabled", args[0])
			return nil
		},
	}
}

func newPluginDisableCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "disable <name>",
		Short: "Disable a plugin",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root := mustGetwd()
			path := filepath.Join(root, ".opencode", "plugins", args[0]+".js")
			disabledPath := path + ".disabled"

			if _, err := os.Stat(path); os.IsNotExist(err) {
				return fmt.Errorf("plugin %s not found", args[0])
			}

			if err := os.Rename(path, disabledPath); err != nil {
				return err
			}

			color.Green("Disabled plugin: %s", args[0])
			return nil
		},
	}
}

func newPluginRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove <name>",
		Short: "Remove a plugin",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root := mustGetwd()
			path := filepath.Join(root, ".opencode", "plugins", args[0]+".js")
			disabledPath := path + ".disabled"

			if _, err := os.Stat(path); err == nil {
				return os.Remove(path)
			}
			if _, err := os.Stat(disabledPath); err == nil {
				return os.Remove(disabledPath)
			}

			return fmt.Errorf("plugin %s not found", args[0])
		},
	}
}

func newPluginUpgradeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "upgrade",
		Short: "Upgrade all plugins",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("  Plugin upgrades are managed via the hub server.")
			fmt.Println("  Run: ocs hub template sync")
			return nil
		},
	}
}

func formatSize(bytes int64) string {
	if bytes < 1024 {
		return fmt.Sprintf("%d B", bytes)
	}
	if bytes < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(bytes)/1024)
	}
	return fmt.Sprintf("%.1f MB", float64(bytes)/(1024*1024))
}
