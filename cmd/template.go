package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/Lenstack/opencode-scaffold/internal/detector"
	tmpl "github.com/Lenstack/opencode-scaffold/internal/domain/template"
	"github.com/Lenstack/opencode-scaffold/internal/engine/config"
	"github.com/Lenstack/opencode-scaffold/internal/engine/skills"
)

func newTemplateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "template",
		Short: "Manage workflow templates",
		Long: `List, create, destroy, import, and export workflow templates.

Templates define complete AI agent workflows including agents, skills,
commands, and pipeline configuration.

Examples:
  ocs template list                  # List all templates
  ocs template show standard         # Show template details
  ocs template detect                # Auto-detect best template for project
  ocs template add my-workflow       # Create custom template from current
  ocs template destroy my-workflow   # Delete custom template
  ocs template export standard       # Export template to YAML
  ocs template import workflow.yaml  # Import template from YAML
`,
	}

	cmd.AddCommand(newTemplateListCmd())
	cmd.AddCommand(newTemplateShowCmd())
	cmd.AddCommand(newTemplateDetectCmd())
	cmd.AddCommand(newTemplateAddCmd())
	cmd.AddCommand(newTemplateDestroyCmd())
	cmd.AddCommand(newTemplateExportCmd())
	cmd.AddCommand(newTemplateImportCmd())

	return cmd
}

func newTemplateListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all available templates",
		RunE: func(cmd *cobra.Command, args []string) error {
			all := tmpl.AllTemplates()
			builtins := tmpl.Builtins()
			user := tmpl.LoadUserTemplates()

			fmt.Println()
			bold := color.New(color.Bold)
			bold.Println("  Built-in Templates:")
			fmt.Println()

			for _, t := range []string{"standard", "minimal", "solo-dev", "team-production", "api-backend", "frontend-app", "fullstack", "empty"} {
				tpl := builtins[t]
				fmt.Printf("    %-20s %s\n", cyan.Sprint(tpl.ID), tpl.Description)
				fmt.Printf("      Agents: %-3d  Skills: %-3d  Commands: %d\n", len(tpl.Agents), len(tpl.Skills), len(tpl.Commands))
			}

			if len(user) > 0 {
				fmt.Println()
				bold.Println("  Custom Templates:")
				fmt.Println()
				for id, t := range user {
					fmt.Printf("    %-20s %s\n", green.Sprint(id), t.Description)
					fmt.Printf("      Agents: %-3d  Skills: %-3d  Commands: %d\n", len(t.Agents), len(t.Skills), len(t.Commands))
				}
			}

			fmt.Println()
			fmt.Printf("  Total: %d templates (%d built-in, %d custom)\n", len(all), len(builtins), len(user))
			fmt.Println()
			return nil
		},
	}
}

func newTemplateShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <name>",
		Short: "Show template details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			tpl, err := tmpl.GetTemplate(args[0])
			if err != nil {
				return err
			}

			fmt.Println()
			bold := color.New(color.Bold)
			bold.Printf("  Template: %s\n", tpl.Name)
			fmt.Println()
			fmt.Printf("    ID:          %s\n", tpl.ID)
			fmt.Printf("    Description: %s\n", tpl.Description)
			fmt.Printf("    CI:          %v\n", tpl.IncludeCI)
			fmt.Printf("    Discovery:   %v\n", tpl.IncludeDiscovery)
			fmt.Println()

			if len(tpl.Agents) > 0 {
				fmt.Printf("    Agents (%d):\n", len(tpl.Agents))
				for _, a := range tpl.Agents {
					fmt.Printf("      - %s\n", a)
				}
				fmt.Println()
			}

			if len(tpl.Skills) > 0 {
				fmt.Printf("    Skills (%d):\n", len(tpl.Skills))
				for _, s := range tpl.Skills {
					fmt.Printf("      - %s\n", s)
				}
				fmt.Println()
			}

			if len(tpl.Commands) > 0 {
				fmt.Printf("    Commands (%d):\n", len(tpl.Commands))
				for _, c := range tpl.Commands {
					fmt.Printf("      - /%s\n", c)
				}
				fmt.Println()
			}

			return nil
		},
	}
}

func newTemplateDetectCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "detect",
		Short: "Auto-detect best template for current project",
		RunE: func(cmd *cobra.Command, args []string) error {
			root, _ := os.Getwd()
			stack := detector.Detect(root)
			fileCount := countFiles(root)
			hasCI := hasCI(root)

			detected := tmpl.DetectTemplate(stack, fileCount, hasCI)
			tpl, _ := tmpl.GetTemplate(detected)

			fmt.Println()
			bold := color.New(color.Bold)
			bold.Println("  Template Detection Results:")
			fmt.Println()

			fmt.Printf("    Stack:     %s\n", stack.Name)
			fmt.Printf("    Files:     %d\n", fileCount)
			fmt.Printf("    Has CI:    %v\n", hasCI)
			fmt.Println()

			green.Printf("    Detected:  %s (%s)\n", tpl.Name, tpl.ID)
			fmt.Printf("    Description: %s\n", tpl.Description)
			fmt.Println()

			fmt.Printf("    Agents:    %d\n", len(tpl.Agents))
			fmt.Printf("    Skills:    %d\n", len(tpl.Skills))
			fmt.Printf("    Commands:  %d\n", len(tpl.Commands))
			fmt.Println()

			fmt.Printf("  To use this template:\n")
			fmt.Printf("    ocs init --template %s\n", detected)
			fmt.Println()

			return nil
		},
	}
}

func newTemplateAddCmd() *cobra.Command {
	var description string
	var agents, skills, commands []string
	var includeCI, includeDiscovery bool

	cmd := &cobra.Command{
		Use:   "add <name>",
		Short: "Create a custom template",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			tpl := tmpl.Template{
				ID:               args[0],
				Name:             args[0],
				Description:      description,
				Agents:           agents,
				Skills:           skills,
				Commands:         commands,
				IncludeCI:        includeCI,
				IncludeDiscovery: includeDiscovery,
			}

			if err := tmpl.AddTemplate(tpl); err != nil {
				return err
			}

			color.Green("Created template: %s", args[0])
			fmt.Printf("  Edit it at: %s/%s.yaml\n", tmpl.UserTemplateDir(), args[0])
			return nil
		},
	}

	cmd.Flags().StringVar(&description, "description", "", "Template description")
	cmd.Flags().StringSliceVar(&agents, "agents", nil, "Agent names to include")
	cmd.Flags().StringSliceVar(&skills, "skills", nil, "Skill names to include")
	cmd.Flags().StringSliceVar(&commands, "commands", nil, "Command names to include")
	cmd.Flags().BoolVar(&includeCI, "ci", false, "Include CI workflow")
	cmd.Flags().BoolVar(&includeDiscovery, "discovery", false, "Include discovery plugin")

	return cmd
}

func newTemplateDestroyCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "destroy <name>",
		Short: "Delete a custom template",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := tmpl.DestroyTemplate(args[0]); err != nil {
				return err
			}

			color.Green("Destroyed template: %s", args[0])
			return nil
		},
	}
}

func newTemplateExportCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "export <name>",
		Short: "Export a template to YAML",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			yaml, err := tmpl.ExportTemplate(args[0])
			if err != nil {
				return err
			}

			fmt.Println(yaml)
			return nil
		},
	}
}

func newTemplateImportCmd() *cobra.Command {
	var file string

	cmd := &cobra.Command{
		Use:   "import",
		Short: "Import a template from YAML file or stdin",
		RunE: func(cmd *cobra.Command, args []string) error {
			var content string

			if file != "" {
				data, err := os.ReadFile(file)
				if err != nil {
					return fmt.Errorf("read file: %w", err)
				}
				content = string(data)
			} else {
				data := make([]byte, 0)
				buf := make([]byte, 1024)
				for {
					n, _ := os.Stdin.Read(buf)
					if n > 0 {
						data = append(data, buf[:n]...)
					}
					if n == 0 {
						break
					}
				}
				content = string(data)
			}

			if err := tmpl.ImportTemplate(content); err != nil {
				return err
			}

			color.Green("Template imported successfully")
			return nil
		},
	}

	cmd.Flags().StringVarP(&file, "file", "f", "", "YAML file to import")

	return cmd
}

func newSkillCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "skill",
		Short: "Manage skills (local and skills.sh)",
		Long: `Manage agent skills including discovery from skills.sh registry.

Examples:
  ocs skill search tdd                    # Search skills.sh for TDD skills
  ocs skill install obra/superpowers      # Install skill from skills.sh
  ocs skill list                          # List locally installed skills
  ocs skill list --remote                 # List skills.sh registry
  ocs skill uninstall <name>              # Remove a skill
`,
	}

	cmd.AddCommand(newSkillCreateCmd())
	cmd.AddCommand(newSkillSearchCmd())
	cmd.AddCommand(newSkillInstallCmd())
	cmd.AddCommand(newSkillListCmd())
	cmd.AddCommand(newSkillUninstallCmd())

	return cmd
}

func newSkillCreateCmd() *cobra.Command {
	var description string

	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a new local skill",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			root := mustGetwd()
			dir := filepath.Join(root, ".opencode", "skills", name)
			skillFile := filepath.Join(dir, "SKILL.md")

			if _, err := os.Stat(skillFile); err == nil {
				return fmt.Errorf("skill %s already exists", name)
			}

			if description == "" {
				description = name + " skill"
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
			if err := os.WriteFile(skillFile, []byte(content), 0644); err != nil {
				return err
			}

			// Track in config DB
			d, err := openEngine()
			if err == nil {
				defer d.Close()
				tracker := config.NewTracker(d, root)
				tracker.TrackConfig(filepath.Join(".opencode", "skills", name, "SKILL.md"), content, "user", "cli")
			}

			color.Green("Created skill: %s", name)
			fmt.Printf("   Path: %s\n", skillFile)
			return nil
		},
	}

	cmd.Flags().StringVar(&description, "description", "", "Skill description")
	return cmd
}

func newSkillSearchCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "search <query>",
		Short: "Search skills.sh registry",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			results, err := skills.SearchSkills(args[0])
			if err != nil {
				return err
			}

			if len(results) == 0 {
				fmt.Println("  No skills found.")
				return nil
			}

			fmt.Println()
			bold := color.New(color.Bold)
			bold.Printf("  Found %d skills for %q:\n", len(results), args[0])
			fmt.Println()

			limit := 20
			if len(results) < limit {
				limit = len(results)
			}

			for _, s := range results[:limit] {
				fmt.Printf("  %-30s %s\n", cyan.Sprint(s.Name), s.Description)
				fmt.Printf("    %s/%s  •  %d installs\n", s.Owner, s.Repo, s.Installs)
				fmt.Println()
			}

			if len(results) > 20 {
				fmt.Printf("  ... and %d more.\n", len(results)-20)
			}

			fmt.Println()
			return nil
		},
	}
}

func newSkillInstallCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "install <owner/repo[/name]>",
		Short: "Install a skill from skills.sh",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			owner, repo, name, err := skills.ParseSkillRef(args[0])
			if err != nil {
				return err
			}

			root := mustGetwd()
			d, err := openEngine()
			if err != nil {
				return err
			}
			defer d.Close()

			installer := skills.NewInstaller(d, root)

			if name == repo {
				registry, err := skills.FetchRegistry()
				if err != nil {
					return err
				}

				installed := 0
				for _, s := range registry.Skills {
					if s.Owner == owner && s.Repo == repo {
						if err := installer.InstallSkill(s.Owner, s.Repo, s.Name); err != nil {
							fmt.Printf("  %s %s: %v\n", color.YellowString("WARN"), s.Name, err)
							continue
						}
						fmt.Printf("  %s %s\n", color.GreenString("OK"), s.Name)
						installed++
					}
				}

				if installed == 0 {
					return fmt.Errorf("no skills found in %s/%s", owner, repo)
				}

				fmt.Printf("\n  Installed %d skill(s) from %s/%s\n", installed, owner, repo)
			} else {
				if err := installer.InstallSkill(owner, repo, name); err != nil {
					return err
				}
				color.Green("Installed skill: %s/%s/%s", owner, repo, name)
			}

			fmt.Println()
			return nil
		},
	}
}

func newSkillListCmd() *cobra.Command {
	var remote bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List skills",
		RunE: func(cmd *cobra.Command, args []string) error {
			if remote {
				registry, err := skills.FetchRegistry()
				if err != nil {
					return err
				}

				fmt.Println()
				bold := color.New(color.Bold)
				bold.Printf("  skills.sh Registry (%d skills):\n", registry.Total)
				fmt.Println()

				limit := 30
				if len(registry.Skills) < limit {
					limit = len(registry.Skills)
				}

				for i, s := range registry.Skills[:limit] {
					fmt.Printf("  %3d. %-30s %s\n", i+1, cyan.Sprint(s.Name), s.Description)
					fmt.Printf("       %s/%s  •  %d installs\n", s.Owner, s.Repo, s.Installs)
				}
				fmt.Println()
				return nil
			}

			root := mustGetwd()
			d, err := openEngine()
			if err != nil {
				return err
			}
			defer d.Close()

			installer := skills.NewInstaller(d, root)
			skillList, err := installer.ListInstalledSkills()
			if err != nil {
				return err
			}

			if len(skillList) == 0 {
				fmt.Println("  No skills installed.")
				return nil
			}

			fmt.Println()
			bold := color.New(color.Bold)
			bold.Println("  Installed Skills:")
			fmt.Println()

			for _, s := range skillList {
				name := s["name"]
				source := s["source"]
				installed := s["installed_at"]
				fmt.Printf("  %-25s source: %-12s installed: %s\n", cyan.Sprint(name), source, installed)
			}
			fmt.Println()

			return nil
		},
	}

	cmd.Flags().BoolVar(&remote, "remote", false, "List skills.sh registry instead of local skills")
	return cmd
}

func newSkillUninstallCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "uninstall <name>",
		Short: "Uninstall a skill",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root := mustGetwd()
			d, err := openEngine()
			if err != nil {
				return err
			}
			defer d.Close()

			installer := skills.NewInstaller(d, root)
			if err := installer.UninstallSkill(args[0]); err != nil {
				return err
			}

			color.Green("Uninstalled skill: %s", args[0])
			return nil
		},
	}
}

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage configuration tracking",
		Long: `Track, version, and restore OpenCode configuration.

All configuration changes are tracked in LevelDB with version history.

Examples:
  ocs config list                     # List all tracked config
  ocs config show opencode.json       # Show config content
  ocs config history opencode.json    # Show version history
  ocs config track                    # Track all current config
  ocs config export                   # Export all config to files
  ocs config import                   # Import config from files to DB
`,
	}

	cmd.AddCommand(newConfigListCmd())
	cmd.AddCommand(newConfigShowCmd())
	cmd.AddCommand(newConfigHistoryCmd())
	cmd.AddCommand(newConfigTrackCmd())
	cmd.AddCommand(newConfigExportCmd())
	cmd.AddCommand(newConfigImportCmd())

	return cmd
}

func newConfigListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all tracked config",
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := openEngine()
			if err != nil {
				return err
			}
			defer d.Close()

			tracker := config.NewTracker(d, mustGetwd())
			configs, err := tracker.ListConfigs()
			if err != nil {
				return err
			}

			if len(configs) == 0 {
				fmt.Println("  No config tracked yet. Run: ocs config track")
				return nil
			}

			fmt.Println()
			bold := color.New(color.Bold)
			bold.Println("  Tracked Configuration:")
			fmt.Println()

			for _, c := range configs {
				fmt.Printf("  %-40s v%-3d  %s  by: %s\n",
					cyan.Sprint(c.Path), c.Version, c.ModifiedAt, c.ModifiedBy)
			}
			fmt.Println()
			return nil
		},
	}
}

func newConfigShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <path>",
		Short: "Show config content from DB",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := openEngine()
			if err != nil {
				return err
			}
			defer d.Close()

			tracker := config.NewTracker(d, mustGetwd())
			entry, err := tracker.GetConfig(args[0])
			if err != nil {
				return err
			}

			fmt.Printf("\n# %s (v%d)\n", args[0], entry.Version)
			fmt.Printf("# Modified: %s by %s\n", entry.ModifiedAt, entry.ModifiedBy)
			fmt.Printf("# Source: %s\n\n", entry.Source)
			fmt.Println(entry.Content)
			return nil
		},
	}
}

func newConfigHistoryCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "history <path>",
		Short: "Show config version history",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := openEngine()
			if err != nil {
				return err
			}
			defer d.Close()

			tracker := config.NewTracker(d, mustGetwd())
			changes, err := tracker.GetHistory(args[0])
			if err != nil {
				return err
			}

			if len(changes) == 0 {
				fmt.Printf("  No history for %s\n", args[0])
				return nil
			}

			fmt.Println()
			bold := color.New(color.Bold)
			bold.Printf("  History for %s:\n", args[0])
			fmt.Println()

			for _, c := range changes {
				fmt.Printf("  %-25s v%d → v%d  (%s)\n",
					c.Timestamp, c.OldVersion, c.NewVersion, c.Action)
			}
			fmt.Println()
			return nil
		},
	}
}

func newConfigTrackCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "track",
		Short: "Track all current configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := openEngine()
			if err != nil {
				return err
			}
			defer d.Close()

			tracker := config.NewTracker(d, mustGetwd())
			if err := tracker.TrackAllConfigs("user", "manual"); err != nil {
				return err
			}

			color.Green("Configuration tracked")
			return nil
		},
	}
}

func newConfigExportCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "export",
		Short: "Export tracked config to files",
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := openEngine()
			if err != nil {
				return err
			}
			defer d.Close()

			tracker := config.NewTracker(d, mustGetwd())
			configs, err := tracker.ListConfigs()
			if err != nil {
				return err
			}

			for _, c := range configs {
				path := filepath.Join(mustGetwd(), c.Path)
				if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
					fmt.Printf("  %s Failed to create dir for %s: %v\n", color.RedString("ERR"), c.Path, err)
					continue
				}
				if err := os.WriteFile(path, []byte(c.Content), 0644); err != nil {
					fmt.Printf("  %s Failed to write %s: %v\n", color.RedString("ERR"), c.Path, err)
					continue
				}
				fmt.Printf("  %s %s (v%d)\n", color.GreenString("OK"), c.Path, c.Version)
			}

			fmt.Println()
			return nil
		},
	}
}

func newConfigImportCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "import",
		Short: "Import config from files to DB",
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := openEngine()
			if err != nil {
				return err
			}
			defer d.Close()

			tracker := config.NewTracker(d, mustGetwd())
			if err := tracker.TrackAllConfigs("user", "import"); err != nil {
				return err
			}

			color.Green("Configuration imported")
			return nil
		},
	}
}
