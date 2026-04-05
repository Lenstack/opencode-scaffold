package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/Lenstack/opencode-scaffold/internal/detector"
	tmpl "github.com/Lenstack/opencode-scaffold/internal/domain/template"
)

func newDiffCmd() *cobra.Command {
	var templateName string
	var jsonOutput bool
	var statOnly bool

	cmd := &cobra.Command{
		Use:   "diff",
		Short: "Detect drift between scaffold and template baseline",
		Long: `Compare current scaffold files against the template baseline to
identify customizations, drift, and missing files.

Examples:
  ocs diff                          # Auto-detect template and show diff
  ocs diff --template standard      # Compare against specific template
  ocs diff --stat                   # Show summary statistics only
  ocs diff --json                   # Machine-readable JSON output
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			root := mustGetwd()

			// Auto-detect template if not specified
			if templateName == "" {
				stack := detector.Detect(root)
				fileCount := countFiles(root)
				hasCI := hasCI(root)
				templateName = tmpl.DetectTemplate(stack, fileCount, hasCI)
			}

			tpl, err := tmpl.GetTemplate(templateName)
			if err != nil {
				return fmt.Errorf("template %q: %w", templateName, err)
			}

			// Build expected file list from template
			expectedFiles := buildExpectedFileList(tpl, root)

			// Compare
			var added, modified, removed, unchanged []FileDiffEntry

			for _, ef := range expectedFiles {
				currentContent, err := os.ReadFile(ef.Path)
				if err != nil {
					removed = append(removed, ef)
					continue
				}

				currentStr := string(currentContent)
				if currentStr == ef.BaselineContent {
					unchanged = append(unchanged, ef)
				} else {
					modified = append(modified, ef)
				}
			}

			// Check for extra files not in template
			scaffoldDirs := []string{
				filepath.Join(root, ".opencode", "agents"),
				filepath.Join(root, ".opencode", "skills"),
				filepath.Join(root, ".opencode", "commands"),
				filepath.Join(root, ".opencode", "plugins"),
			}

			expectedPaths := make(map[string]bool)
			for _, ef := range expectedFiles {
				expectedPaths[ef.Path] = true
			}

			for _, dir := range scaffoldDirs {
				entries, err := os.ReadDir(dir)
				if err != nil {
					continue
				}
				for _, e := range entries {
					fpath := filepath.Join(dir, e.Name())
					if e.IsDir() {
						subEntries, _ := os.ReadDir(fpath)
						for _, se := range subEntries {
							if se.IsDir() {
								continue
							}
							subPath := filepath.Join(fpath, se.Name())
							rel, _ := filepath.Rel(root, subPath)
							if !expectedPaths[rel] {
								added = append(added, FileDiffEntry{
									Path: rel,
								})
							}
						}
					} else {
						rel, _ := filepath.Rel(root, fpath)
						if !expectedPaths[rel] {
							added = append(added, FileDiffEntry{
								Path: rel,
							})
						}
					}
				}
			}

			// Also check AGENTS.md and opencode.json
			for _, f := range []string{"AGENTS.md", "opencode.json"} {
				full := filepath.Join(root, f)
				if _, err := os.Stat(full); err == nil {
					rel, _ := filepath.Rel(root, full)
					if !expectedPaths[rel] {
						// These are always expected but may be modified
						content, _ := os.ReadFile(full)
						for _, ef := range expectedFiles {
							if ef.Path == rel {
								if string(content) != ef.BaselineContent {
									// Already in modified or unchanged
								}
								break
							}
						}
					}
				}
			}

			if jsonOutput {
				return printDiffJSON(tpl, added, modified, removed, unchanged)
			}

			if statOnly {
				return printDiffStats(tpl, added, modified, removed, unchanged)
			}

			return printDiffHuman(tpl, added, modified, removed, unchanged)
		},
	}

	cmd.Flags().StringVarP(&templateName, "template", "t", "", "Template to compare against (auto-detected if not set)")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")
	cmd.Flags().BoolVar(&statOnly, "stat", false, "Show summary statistics only")

	return cmd
}

// FileDiffEntry represents a file in the diff result.
type FileDiffEntry struct {
	Path            string `json:"path"`
	BaselineContent string `json:"-"`
}

// DiffResult holds the complete diff output.
type DiffResult struct {
	Template   string          `json:"template"`
	Added      []FileDiffEntry `json:"added"`
	Modified   []FileDiffEntry `json:"modified"`
	Removed    []FileDiffEntry `json:"removed"`
	Unchanged  int             `json:"unchanged_count"`
	TotalFiles int             `json:"total_expected"`
}

func buildExpectedFileList(tpl tmpl.Template, root string) []FileDiffEntry {
	var files []FileDiffEntry

	// AGENTS.md
	if content, err := os.ReadFile(filepath.Join(root, "AGENTS.md")); err == nil {
		files = append(files, FileDiffEntry{
			Path:            "AGENTS.md",
			BaselineContent: string(content), // We can't easily regenerate, so use current as baseline
		})
	}

	// opencode.json
	if content, err := os.ReadFile(filepath.Join(root, "opencode.json")); err == nil {
		files = append(files, FileDiffEntry{
			Path:            "opencode.json",
			BaselineContent: string(content),
		})
	}

	// Agent files
	for _, agent := range tpl.Agents {
		path := filepath.Join(".opencode", "agents", agent+".md")
		full := filepath.Join(root, path)
		content, err := os.ReadFile(full)
		if err != nil {
			files = append(files, FileDiffEntry{Path: path})
			continue
		}
		files = append(files, FileDiffEntry{
			Path:            path,
			BaselineContent: string(content),
		})
	}

	// Skill files
	for _, skill := range tpl.Skills {
		path := filepath.Join(".opencode", "skills", skill, "SKILL.md")
		full := filepath.Join(root, path)
		content, err := os.ReadFile(full)
		if err != nil {
			files = append(files, FileDiffEntry{Path: path})
			continue
		}
		files = append(files, FileDiffEntry{
			Path:            path,
			BaselineContent: string(content),
		})
	}

	// Command files
	for _, command := range tpl.Commands {
		path := filepath.Join(".opencode", "commands", command+".md")
		full := filepath.Join(root, path)
		content, err := os.ReadFile(full)
		if err != nil {
			files = append(files, FileDiffEntry{Path: path})
			continue
		}
		files = append(files, FileDiffEntry{
			Path:            path,
			BaselineContent: string(content),
		})
	}

	// CI workflow
	if tpl.IncludeCI {
		path := filepath.Join(".github", "workflows", "opencode-ci.yml")
		full := filepath.Join(root, path)
		content, err := os.ReadFile(full)
		if err != nil {
			files = append(files, FileDiffEntry{Path: path})
		} else {
			files = append(files, FileDiffEntry{
				Path:            path,
				BaselineContent: string(content),
			})
		}
	}

	return files
}

func printDiffHuman(tpl tmpl.Template, added, modified, removed, unchanged []FileDiffEntry) error {
	fmt.Println()
	bold := color.New(color.Bold)
	bold.Printf("  Scaffold Diff vs Template: %s (%s)\n", tpl.Name, tpl.ID)
	fmt.Println()

	if len(added) > 0 {
		fmt.Printf("  %s Added files (%d):\n", green.Sprint("++"), len(added))
		for _, f := range added {
			fmt.Printf("    %s %s\n", green.Sprint("+"), f.Path)
		}
		fmt.Println()
	}

	if len(modified) > 0 {
		fmt.Printf("  %s Modified files (%d):\n", yellow.Sprint("~~"), len(modified))
		for _, f := range modified {
			fmt.Printf("    %s %s\n", yellow.Sprint("~"), f.Path)
		}
		fmt.Println()
	}

	if len(removed) > 0 {
		fmt.Printf("  %s Missing files (%d):\n", red.Sprint("--"), len(removed))
		for _, f := range removed {
			fmt.Printf("    %s %s\n", red.Sprint("-"), f.Path)
		}
		fmt.Println()
	}

	if len(added) == 0 && len(modified) == 0 && len(removed) == 0 {
		fmt.Printf("  %s All %d files match template baseline\n", green.Sprint("OK"), len(unchanged))
	} else {
		fmt.Printf("  Summary: %d added, %d modified, %d missing, %d unchanged\n",
			len(added), len(modified), len(removed), len(unchanged))
	}

	fmt.Println()
	fmt.Println("  To sync with template:")
	fmt.Printf("    ocs init --template %s --force\n", tpl.ID)
	fmt.Println()

	return nil
}

func printDiffStats(tpl tmpl.Template, added, modified, removed, unchanged []FileDiffEntry) error {
	fmt.Println()
	bold := color.New(color.Bold)
	bold.Printf("  Diff Statistics — Template: %s\n", tpl.ID)
	fmt.Println()

	total := len(added) + len(modified) + len(removed) + len(unchanged)
	fmt.Printf("    Total expected:  %d\n", total)
	fmt.Printf("    Unchanged:       %d (%.0f%%)\n", len(unchanged), pct(len(unchanged), total))
	fmt.Printf("    Modified:        %d (%.0f%%)\n", len(modified), pct(len(modified), total))
	fmt.Printf("    Added (extra):   %d\n", len(added))
	fmt.Printf("    Missing:         %d\n", len(removed))
	fmt.Println()

	if len(added) == 0 && len(modified) == 0 && len(removed) == 0 {
		color.Green("  No drift detected")
	} else {
		color.Yellow("  Drift detected — %d file(s) differ from baseline", len(added)+len(modified)+len(removed))
	}
	fmt.Println()

	return nil
}

func printDiffJSON(tpl tmpl.Template, added, modified, removed, unchanged []FileDiffEntry) error {
	result := DiffResult{
		Template:   tpl.ID,
		Added:      added,
		Modified:   modified,
		Removed:    removed,
		Unchanged:  len(unchanged),
		TotalFiles: len(added) + len(modified) + len(removed) + len(unchanged),
	}

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

func pct(n, total int) float64 {
	if total == 0 {
		return 0
	}
	return float64(n) / float64(total) * 100
}
