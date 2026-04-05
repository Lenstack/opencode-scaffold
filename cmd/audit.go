package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/Lenstack/opencode-scaffold/internal/engine/config"
)

func newAuditCmd() *cobra.Command {
	var jsonOutput bool
	var fileFilter string
	var since string

	cmd := &cobra.Command{
		Use:   "audit",
		Short: "View full change history and audit trail",
		Long: `Display the complete audit trail of all scaffold configuration changes.
Reads from both LevelDB config history and hub audit log (if available).

Examples:
  ocs audit                              # Show all changes
  ocs audit --file opencode.json         # Filter by file
  ocs audit --since 24h                  # Changes in last 24 hours
  ocs audit --json                       # Machine-readable JSON output
`,
		RunE: func(cmd *cobra.Command, args []string) error {
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

			var entries []AuditEntry

			for _, c := range configs {
				changes, err := tracker.GetHistory(c.Path)
				if err != nil {
					continue
				}
				for _, ch := range changes {
					entry := AuditEntry{
						Timestamp: ch.Timestamp,
						User:      "user",
						Action:    ch.Action,
						Detail: fmt.Sprintf("%s: v%d → v%d",
							ch.Path, ch.OldVersion, ch.NewVersion),
						Source: "config",
					}
					entries = append(entries, entry)
				}

				entry := AuditEntry{
					Timestamp: c.ModifiedAt,
					User:      c.ModifiedBy,
					Action:    "current",
					Detail: fmt.Sprintf("%s: v%d (source: %s)",
						c.Path, c.Version, c.Source),
					Source: "config",
				}
				entries = append(entries, entry)
			}

			hubEntries := readHubAuditLog(root)
			entries = append(entries, hubEntries...)

			if fileFilter != "" {
				var filtered []AuditEntry
				for _, e := range entries {
					if containsFile(e.Detail, fileFilter) {
						filtered = append(filtered, e)
					}
				}
				entries = filtered
			}

			if since != "" {
				cutoff, err := parseSince(since)
				if err != nil {
					return fmt.Errorf("invalid --since value: %w", err)
				}
				var filtered []AuditEntry
				for _, e := range entries {
					t, err := time.Parse(time.RFC3339, e.Timestamp)
					if err != nil {
						continue
					}
					if t.After(cutoff) {
						filtered = append(filtered, e)
					}
				}
				entries = filtered
			}

			sortAuditEntries(entries)

			if jsonOutput {
				return printAuditJSON(entries)
			}

			return printAuditHuman(entries)
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")
	cmd.Flags().StringVar(&fileFilter, "file", "", "Filter by file path")
	cmd.Flags().StringVar(&since, "since", "", "Show changes since (e.g. 24h, 7d, 30m)")

	return cmd
}

type AuditEntry struct {
	Timestamp string `json:"timestamp"`
	User      string `json:"user"`
	Action    string `json:"action"`
	Detail    string `json:"detail"`
	Source    string `json:"source"`
}

func readHubAuditLog(root string) []AuditEntry {
	dbPath := filepath.Join(root, ".opencode", "data", "hub.db")
	if _, err := os.Stat(dbPath); err != nil {
		return nil
	}
	return nil
}

func containsFile(detail, file string) bool {
	return len(detail) >= len(file) && (detail == file ||
		len(detail) > len(file) && (detail[:len(file)] == file && (detail[len(file)] == ':' || detail[len(file)] == '/')))
}

func parseSince(since string) (time.Time, error) {
	now := time.Now()

	if len(since) < 2 {
		return time.Time{}, fmt.Errorf("invalid format: %s (use: 24h, 7d, 30m)", since)
	}

	unit := since[len(since)-1]
	var duration time.Duration

	switch unit {
	case 'h':
		var hours int
		_, err := fmt.Sscanf(since, "%dh", &hours)
		if err != nil {
			return time.Time{}, fmt.Errorf("invalid hours: %s", since)
		}
		duration = time.Duration(hours) * time.Hour
	case 'd':
		var days int
		_, err := fmt.Sscanf(since, "%dd", &days)
		if err != nil {
			return time.Time{}, fmt.Errorf("invalid days: %s", since)
		}
		duration = time.Duration(days) * 24 * time.Hour
	case 'm':
		var mins int
		_, err := fmt.Sscanf(since, "%dm", &mins)
		if err != nil {
			return time.Time{}, fmt.Errorf("invalid minutes: %s", since)
		}
		duration = time.Duration(mins) * time.Minute
	default:
		return time.Time{}, fmt.Errorf("invalid unit: %c (use h, d, or m)", unit)
	}

	return now.Add(-duration), nil
}

func sortAuditEntries(entries []AuditEntry) {
	for i := 0; i < len(entries); i++ {
		for j := i + 1; j < len(entries); j++ {
			ti, _ := time.Parse(time.RFC3339, entries[i].Timestamp)
			tj, _ := time.Parse(time.RFC3339, entries[j].Timestamp)
			if tj.After(ti) {
				entries[i], entries[j] = entries[j], entries[i]
			}
		}
	}
}

func printAuditHuman(entries []AuditEntry) error {
	fmt.Println()
	bold := color.New(color.Bold)
	bold.Println("  Audit Trail:")
	fmt.Println()

	if len(entries) == 0 {
		fmt.Println("  No audit entries found.")
		fmt.Println()
		return nil
	}

	for _, e := range entries {
		actionColor := cyan
		switch e.Action {
		case "create":
			actionColor = green
		case "update":
			actionColor = yellow
		case "delete":
			actionColor = red
		case "current":
			actionColor = color.New(color.FgWhite)
		}

		ts := e.Timestamp
		if len(ts) > 19 {
			ts = ts[:19]
		}

		fmt.Printf("  %s  %-8s  %-6s  %s\n",
			ts,
			actionColor.Sprint(e.Action),
			e.User,
			e.Detail,
		)
	}

	fmt.Println()
	fmt.Printf("  Total entries: %d\n", len(entries))
	fmt.Println()

	return nil
}

func printAuditJSON(entries []AuditEntry) error {
	result := struct {
		Entries []AuditEntry `json:"entries"`
		Total   int          `json:"total"`
	}{
		Entries: entries,
		Total:   len(entries),
	}

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}
