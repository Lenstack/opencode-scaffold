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
	"gopkg.in/yaml.v3"

	tmpl "github.com/Lenstack/opencode-scaffold/internal/domain/template"
)

func newLintCmd() *cobra.Command {
	var jsonOutput bool
	var fix bool

	cmd := &cobra.Command{
		Use:   "lint",
		Short: "Validate scaffold integrity and quality",
		Long: `Check scaffold files for integrity issues including malformed
frontmatter, broken references, invalid syntax, and quality warnings.

Examples:
  ocs lint                          # Run all lint checks
  ocs lint --json                   # Machine-readable JSON output
  ocs lint --fix                    # Auto-fix common issues
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			root := mustGetwd()
			var issues []LintIssue

			issues = append(issues, lintAgents(root)...)
			issues = append(issues, lintSkills(root)...)
			issues = append(issues, lintCommands(root)...)
			issues = append(issues, lintOpenCodeJSON(root)...)
			issues = append(issues, lintAgentsMD(root)...)
			issues = append(issues, lintTemplateRefs(root)...)
			issues = append(issues, lintDebugCode(root)...)
			issues = append(issues, lintBareTODOs(root)...)

			if jsonOutput {
				return printLintJSON(issues)
			}

			return printLintHuman(issues, fix)
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")
	cmd.Flags().BoolVar(&fix, "fix", false, "Auto-fix common issues")

	return cmd
}

// LintIssue represents a single lint finding.
type LintIssue struct {
	Severity string `json:"severity"`
	File     string `json:"file"`
	Message  string `json:"message"`
	Fixable  bool   `json:"fixable"`
}

func lintAgents(root string) []LintIssue {
	var issues []LintIssue
	dir := filepath.Join(root, ".opencode", "agents")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return append(issues, LintIssue{
			Severity: "error",
			File:     ".opencode/agents/",
			Message:  "agents directory not found",
		})
	}

	validModes := map[string]bool{"primary": true, "subagent": true}

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		fpath := filepath.Join(dir, e.Name())
		content, err := os.ReadFile(fpath)
		if err != nil {
			issues = append(issues, LintIssue{
				Severity: "error",
				File:     ".opencode/agents/" + e.Name(),
				Message:  "cannot read file: " + err.Error(),
			})
			continue
		}

		s := string(content)
		fm, hasFM := parseFrontmatter(s)

		if !hasFM {
			issues = append(issues, LintIssue{
				Severity: "error",
				File:     ".opencode/agents/" + e.Name(),
				Message:  "missing YAML frontmatter",
				Fixable:  false,
			})
			continue
		}

		// OpenCode uses filename as agent name — 'name' field is optional
		if fm["description"] == "" {
			issues = append(issues, LintIssue{
				Severity: "warning",
				File:     ".opencode/agents/" + e.Name(),
				Message:  "missing 'description' in frontmatter",
			})
		}

		mode, _ := fm["mode"].(string)
		if mode == "" {
			issues = append(issues, LintIssue{
				Severity: "error",
				File:     ".opencode/agents/" + e.Name(),
				Message:  "missing 'mode' in frontmatter",
			})
		} else if !validModes[mode] {
			issues = append(issues, LintIssue{
				Severity: "warning",
				File:     ".opencode/agents/" + e.Name(),
				Message:  fmt.Sprintf("invalid mode %q (expected 'primary' or 'subagent')", mode),
			})
		}

		if len(strings.TrimSpace(s)) == 0 {
			issues = append(issues, LintIssue{
				Severity: "error",
				File:     ".opencode/agents/" + e.Name(),
				Message:  "file is empty",
			})
		}
	}

	return issues
}

func lintSkills(root string) []LintIssue {
	var issues []LintIssue
	dir := filepath.Join(root, ".opencode", "skills")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return append(issues, LintIssue{
			Severity: "error",
			File:     ".opencode/skills/",
			Message:  "skills directory not found",
		})
	}

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		skillFile := filepath.Join(dir, e.Name(), "SKILL.md")
		content, err := os.ReadFile(skillFile)
		if err != nil {
			issues = append(issues, LintIssue{
				Severity: "error",
				File:     ".opencode/skills/" + e.Name() + "/SKILL.md",
				Message:  "SKILL.md not found or unreadable",
			})
			continue
		}

		s := string(content)
		fm, hasFM := parseFrontmatter(s)

		if !hasFM {
			issues = append(issues, LintIssue{
				Severity: "error",
				File:     ".opencode/skills/" + e.Name() + "/SKILL.md",
				Message:  "missing YAML frontmatter",
			})
			continue
		}

		name, _ := fm["name"].(string)
		if name == "" {
			issues = append(issues, LintIssue{
				Severity: "error",
				File:     ".opencode/skills/" + e.Name() + "/SKILL.md",
				Message:  "missing 'name' in frontmatter",
			})
		} else if name != e.Name() {
			issues = append(issues, LintIssue{
				Severity: "warning",
				File:     ".opencode/skills/" + e.Name() + "/SKILL.md",
				Message:  fmt.Sprintf("frontmatter name %q doesn't match directory name %q", name, e.Name()),
			})
		}

		if fm["description"] == "" {
			issues = append(issues, LintIssue{
				Severity: "warning",
				File:     ".opencode/skills/" + e.Name() + "/SKILL.md",
				Message:  "missing 'description' in frontmatter",
			})
		}

		if len(strings.TrimSpace(s)) == 0 {
			issues = append(issues, LintIssue{
				Severity: "error",
				File:     ".opencode/skills/" + e.Name() + "/SKILL.md",
				Message:  "file is empty",
			})
		}
	}

	return issues
}

func lintCommands(root string) []LintIssue {
	var issues []LintIssue
	dir := filepath.Join(root, ".opencode", "commands")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return append(issues, LintIssue{
			Severity: "error",
			File:     ".opencode/commands/",
			Message:  "commands directory not found",
		})
	}

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		fpath := filepath.Join(dir, e.Name())
		content, err := os.ReadFile(fpath)
		if err != nil {
			issues = append(issues, LintIssue{
				Severity: "error",
				File:     ".opencode/commands/" + e.Name(),
				Message:  "cannot read file: " + err.Error(),
			})
			continue
		}

		_, hasFM := parseFrontmatter(string(content))

		if !hasFM {
			issues = append(issues, LintIssue{
				Severity: "error",
				File:     ".opencode/commands/" + e.Name(),
				Message:  "missing YAML frontmatter",
			})
			continue
		}

		if len(strings.TrimSpace(string(content))) == 0 {
			issues = append(issues, LintIssue{
				Severity: "error",
				File:     ".opencode/commands/" + e.Name(),
				Message:  "file is empty",
			})
		}
	}

	return issues
}

func lintOpenCodeJSON(root string) []LintIssue {
	var issues []LintIssue
	fpath := filepath.Join(root, "opencode.json")
	content, err := os.ReadFile(fpath)
	if err != nil {
		return append(issues, LintIssue{
			Severity: "error",
			File:     "opencode.json",
			Message:  "file not found or unreadable",
		})
	}

	var parsed map[string]any
	if err := json.Unmarshal(content, &parsed); err != nil {
		return append(issues, LintIssue{
			Severity: "error",
			File:     "opencode.json",
			Message:  "invalid JSON: " + err.Error(),
		})
	}

	if _, ok := parsed["agent"]; !ok {
		issues = append(issues, LintIssue{
			Severity: "warning",
			File:     "opencode.json",
			Message:  "missing 'agent' configuration",
		})
	}

	if _, ok := parsed["model"]; !ok {
		issues = append(issues, LintIssue{
			Severity: "warning",
			File:     "opencode.json",
			Message:  "missing 'model' configuration",
		})
	}

	return issues
}

func lintAgentsMD(root string) []LintIssue {
	var issues []LintIssue
	fpath := filepath.Join(root, "AGENTS.md")
	content, err := os.ReadFile(fpath)
	if err != nil {
		return append(issues, LintIssue{
			Severity: "error",
			File:     "AGENTS.md",
			Message:  "file not found or unreadable",
		})
	}

	s := string(content)
	if len(strings.TrimSpace(s)) == 0 {
		issues = append(issues, LintIssue{
			Severity: "error",
			File:     "AGENTS.md",
			Message:  "file is empty",
		})
	}

	requiredSections := []string{"Stack Context", "Agent Pipeline", "Non-Negotiable Rules", "Definition of Done"}
	for _, section := range requiredSections {
		if !strings.Contains(s, section) {
			issues = append(issues, LintIssue{
				Severity: "warning",
				File:     "AGENTS.md",
				Message:  fmt.Sprintf("missing section: %q", section),
			})
		}
	}

	return issues
}

func lintTemplateRefs(root string) []LintIssue {
	var issues []LintIssue

	opencodePath := filepath.Join(root, "opencode.json")
	content, err := os.ReadFile(opencodePath)
	if err != nil {
		return issues
	}

	var parsed map[string]any
	if err := json.Unmarshal(content, &parsed); err != nil {
		return issues
	}

	agents, ok := parsed["agent"].(map[string]any)
	if !ok {
		return issues
	}

	agentsDir := filepath.Join(root, ".opencode", "agents")
	for name := range agents {
		agentFile := filepath.Join(agentsDir, name+".md")
		if _, err := os.Stat(agentFile); os.IsNotExist(err) {
			issues = append(issues, LintIssue{
				Severity: "error",
				File:     "opencode.json",
				Message:  fmt.Sprintf("agent %q referenced but file missing: .opencode/agents/%s.md", name, name),
			})
		}
	}

	for _, tpl := range tmpl.AllTemplates() {
		for _, skill := range tpl.Skills {
			skillPath := filepath.Join(root, ".opencode", "skills", skill, "SKILL.md")
			if _, err := os.Stat(skillPath); os.IsNotExist(err) {
				issues = append(issues, LintIssue{
					Severity: "warning",
					File:     "template:" + tpl.ID,
					Message:  fmt.Sprintf("skill %q referenced in template but not installed", skill),
				})
			}
		}
	}

	return issues
}

var debugPatterns = []*regexp.Regexp{
	regexp.MustCompile(`fmt\.Println\(`),
	regexp.MustCompile(`console\.log\(`),
	regexp.MustCompile(`debugger`),
}

func lintDebugCode(root string) []LintIssue {
	var issues []LintIssue

	dirs := []string{
		filepath.Join(root, ".opencode"),
	}

	for _, dir := range dirs {
		filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if info.IsDir() {
				if info.Name() == "node_modules" {
					return filepath.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(path, ".md") && !strings.HasSuffix(path, ".js") {
				return nil
			}

			content, err := os.ReadFile(path)
			if err != nil {
				return nil
			}

			rel, _ := filepath.Rel(root, path)
			for _, re := range debugPatterns {
				if re.Match(content) {
					issues = append(issues, LintIssue{
						Severity: "warning",
						File:     rel,
						Message:  fmt.Sprintf("debug code detected: %s", re.String()),
						Fixable:  true,
					})
				}
			}
			return nil
		})
	}

	return issues
}

var bareTODORe = regexp.MustCompile(`(?i)TODO\b`)

func lintBareTODOs(root string) []LintIssue {
	var issues []LintIssue

	dirs := []string{
		filepath.Join(root, ".opencode"),
	}

	for _, dir := range dirs {
		filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if info.IsDir() {
				if info.Name() == "node_modules" {
					return filepath.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(path, ".md") {
				return nil
			}

			content, err := os.ReadFile(path)
			if err != nil {
				return nil
			}

			rel, _ := filepath.Rel(root, path)
			if bareTODORe.Match(content) {
				issues = append(issues, LintIssue{
					Severity: "warning",
					File:     rel,
					Message:  "bare TODO without ticket number (use TODO(#123))",
					Fixable:  false,
				})
			}
			return nil
		})
	}

	return issues
}

func parseFrontmatter(content string) (map[string]any, bool) {
	content = strings.TrimSpace(content)
	if !strings.HasPrefix(content, "---") {
		return nil, false
	}

	endIdx := strings.Index(content[3:], "---")
	if endIdx == -1 {
		return nil, false
	}

	yamlContent := content[3 : 3+endIdx]
	var fm map[string]any
	if err := yaml.Unmarshal([]byte(yamlContent), &fm); err != nil {
		return nil, false
	}

	return fm, true
}

func printLintHuman(issues []LintIssue, fix bool) error {
	fmt.Println()
	bold := color.New(color.Bold)
	bold.Println("  Scaffold Lint:")
	fmt.Println()

	if len(issues) == 0 {
		color.Green("  All checks passed! No issues found.")
		fmt.Println()
		return nil
	}

	errors := 0
	warnings := 0
	for _, issue := range issues {
		switch issue.Severity {
		case "error":
			errors++
			fmt.Printf("  %s %s: %s\n", red.Sprint("ERR"), issue.File, issue.Message)
		case "warning":
			warnings++
			fmt.Printf("  %s %s: %s\n", yellow.Sprint("WARN"), issue.File, issue.Message)
		default:
			fmt.Printf("  %s %s: %s\n", cyan.Sprint("INFO"), issue.File, issue.Message)
		}
	}

	fmt.Println()
	fmt.Printf("  %d error(s), %d warning(s)\n", errors, warnings)

	if fix {
		fmt.Println()
		color.Yellow("  Auto-fix is not yet implemented for all issues.")
	}

	fmt.Println()
	return nil
}

func printLintJSON(issues []LintIssue) error {
	result := struct {
		Issues []LintIssue `json:"issues"`
		Errors int         `json:"errors"`
		Warns  int         `json:"warnings"`
	}{
		Issues: issues,
	}
	for _, i := range issues {
		if i.Severity == "error" {
			result.Errors++
		} else if i.Severity == "warning" {
			result.Warns++
		}
	}

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}
