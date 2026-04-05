package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	tmpl "github.com/Lenstack/opencode-scaffold/internal/template"
)

func newListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list <type>",
		Short: "List scaffolded components or available templates",
		Long: `List agents, skills, commands, or available template packs.

Examples:
  ocs list agents
  ocs list skills
  ocs list commands
  ocs list templates
`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, _ := os.Getwd()

			switch args[0] {
			case "agents":
				return listAgents(root)
			case "skills":
				return listSkills(root)
			case "commands":
				return listCommands(root)
			case "templates":
				return listTemplates()
			default:
				return fmt.Errorf("unknown type: %s (try: agents, skills, commands, templates)", args[0])
			}
		},
	}

	return cmd
}

func listAgents(root string) error {
	dir := filepath.Join(root, ".opencode", "agents")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("no agents found — run: ocs init")
	}

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

		fmt.Printf("  %s %-20s %s\n", cyan.Sprint("@"), name, desc)
		fmt.Printf("    mode: %s\n", mode)
		fmt.Println()
	}

	return nil
}

func listSkills(root string) error {
	dir := filepath.Join(root, ".opencode", "skills")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("no skills found — run: ocs init")
	}

	bold.Println("  Skills:")
	fmt.Println()
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		content, err := os.ReadFile(filepath.Join(dir, e.Name(), "SKILL.md"))
		if err != nil {
			continue
		}
		s := string(content)
		desc := extractFrontmatterField(s, "description")

		fmt.Printf("  %s %-25s %s\n", green.Sprint("skill"), e.Name(), desc)
		fmt.Println()
	}

	return nil
}

func listCommands(root string) error {
	dir := filepath.Join(root, ".opencode", "commands")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("no commands found — run: ocs init")
	}

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
}

func listTemplates() error {
	bold.Println("  Available Template Packs:")
	fmt.Println()

	packs := []struct {
		id, name, desc string
	}{
		{"go-encore", "Go + Encore", "Backend services with Encore.go primitives"},
		{"go-fiber", "Go + Fiber", "Fast HTTP framework for Go"},
		{"go-gin", "Go + Gin", "Web framework with martini-like API"},
		{"go-chi", "Go + Chi", "Lightweight, idiomatic router for Go"},
		{"python-fastapi", "Python + FastAPI", "Modern async Python web framework"},
		{"python-django", "Python + Django", "Full-stack Python framework"},
		{"node-nextjs", "Node + Next.js", "React framework with App Router"},
		{"rust-axum", "Rust + Axum", "Ergonomic web framework for Rust"},
		{"generic", "Generic", "Fallback for any project"},
	}

	for _, p := range packs {
		fmt.Printf("  %-20s %s\n", cyan.Sprint(p.id), p.desc)
		fmt.Printf("    %-20s %s\n", "", p.name)
		fmt.Println()
	}

	bold.Println("  Built-in Agents:")
	fmt.Println()
	for _, name := range tmpl.AvailableAgents() {
		fmt.Printf("    %s\n", name)
	}
	fmt.Println()

	bold.Println("  Built-in Skills:")
	fmt.Println()
	for _, name := range tmpl.AvailableSkills() {
		fmt.Printf("    %s\n", name)
	}
	fmt.Println()

	bold.Println("  Built-in Commands:")
	fmt.Println()
	for _, name := range tmpl.AvailableCommands() {
		fmt.Printf("    /%s\n", name)
	}
	fmt.Println()

	return nil
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
