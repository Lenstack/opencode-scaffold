package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/Lenstack/opencode-scaffold/internal/engine/config"
)

func newRollbackCmd() *cobra.Command {
	var targetVersion int
	var dryRun bool
	var fileFilter string

	cmd := &cobra.Command{
		Use:   "rollback",
		Short: "Revert scaffold config to a previous version",
		Long: `Restore scaffold configuration files to a previous tracked version.
All config changes are tracked in LevelDB with version history.

Examples:
  ocs rollback --to 1                  # Rollback all files to version 1
  ocs rollback --to 2 --dry-run        # Preview rollback without applying
  ocs rollback --to 3 --file opencode.json  # Rollback only opencode.json
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if targetVersion < 1 {
				return fmt.Errorf("version must be >= 1")
			}

			root := mustGetwd()
			d, err := openEngine()
			if err != nil {
				return fmt.Errorf("open database: %w", err)
			}
			defer d.Close()

			tracker := config.NewTracker(d, root)
			configs, err := tracker.ListConfigs()
			if err != nil {
				return fmt.Errorf("list configs: %w", err)
			}

			if len(configs) == 0 {
				return fmt.Errorf("no config tracked yet — run: ocs config track")
			}

			// Filter configs
			var targets []config.ConfigEntry
			for _, c := range configs {
				if c.Version > targetVersion {
					continue
				}
				if fileFilter != "" && c.Path != fileFilter {
					continue
				}
				// Get the latest version <= targetVersion for each file
				targets = append(targets, c)
			}

			// Deduplicate: keep only the highest version <= targetVersion per path
			latest := make(map[string]config.ConfigEntry)
			for _, c := range targets {
				existing, ok := latest[c.Path]
				if !ok || c.Version > existing.Version {
					latest[c.Path] = c
				}
			}

			if len(latest) == 0 {
				if fileFilter != "" {
					return fmt.Errorf("no tracked config for file: %s", fileFilter)
				}
				return fmt.Errorf("no config found at version %d", targetVersion)
			}

			if dryRun {
				return printRollbackDryRun(latest, targetVersion)
			}

			return applyRollback(latest, root, tracker)
		},
	}

	cmd.Flags().IntVar(&targetVersion, "to", 0, "Target version to rollback to (required)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview rollback without applying")
	cmd.Flags().StringVar(&fileFilter, "file", "", "Rollback only this specific file")

	cmd.MarkFlagRequired("to")

	return cmd
}

func printRollbackDryRun(latest map[string]config.ConfigEntry, targetVersion int) error {
	fmt.Println()
	bold := color.New(color.Bold)
	bold.Printf("  Rollback Dry Run — Target: v%d\n", targetVersion)
	fmt.Println()

	for path, entry := range latest {
		fmt.Printf("  Would restore: %s (v%d → v%d)\n",
			cyan.Sprint(path), entry.Version, targetVersion)
		fmt.Printf("    Modified: %s by %s\n", entry.ModifiedAt, entry.ModifiedBy)
		fmt.Println()
	}

	fmt.Printf("  Total files to restore: %d\n", len(latest))
	fmt.Println()
	return nil
}

func applyRollback(latest map[string]config.ConfigEntry, root string, tracker *config.Tracker) error {
	fmt.Println()
	bold := color.New(color.Bold)
	bold.Println("  Rollback:")
	fmt.Println()

	applied := 0
	errors := 0

	for path, entry := range latest {
		full := filepath.Join(root, path)

		// Create directory if needed
		if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
			fmt.Printf("  %s Failed to create dir for %s: %v\n", color.RedString("ERR"), path, err)
			errors++
			continue
		}

		// Write the content
		if err := os.WriteFile(full, []byte(entry.Content), 0644); err != nil {
			fmt.Printf("  %s Failed to write %s: %v\n", color.RedString("ERR"), path, err)
			errors++
			continue
		}

		// Track the rollback as a new version
		if err := tracker.TrackConfig(path, entry.Content, "rollback", "cli"); err != nil {
			fmt.Printf("  %s Failed to track rollback for %s: %v\n", color.YellowString("WARN"), path, err)
		}

		fmt.Printf("  %s %s (restored to v%d)\n", color.GreenString("OK"), path, entry.Version)
		applied++
	}

	fmt.Println()
	fmt.Printf("  Applied: %d  Errors: %d  At: %s\n", applied, errors, time.Now().UTC().Format(time.RFC3339))
	fmt.Println()

	if errors > 0 {
		return fmt.Errorf("rollback completed with %d error(s)", errors)
	}

	return nil
}

// RollbackResult is the JSON output structure.
type RollbackResult struct {
	TargetVersion int      `json:"target_version"`
	Applied       int      `json:"applied"`
	Errors        int      `json:"errors"`
	Files         []string `json:"files"`
	Timestamp     string   `json:"timestamp"`
}
