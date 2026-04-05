package cmd

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/Lenstack/opencode-scaffold/internal/detector"
	tmpl "github.com/Lenstack/opencode-scaffold/internal/template"
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
