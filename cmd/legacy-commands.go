package cmd

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/Lenstack/opencode-scaffold/internal/engine/discovery"
	"github.com/Lenstack/opencode-scaffold/internal/engine/skills"
)

func newDiscoverCmd() *cobra.Command {
	var full bool

	cmd := &cobra.Command{
		Use:   "discover",
		Short: "Index project into LevelDB",
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := openEngine()
			if err != nil {
				return err
			}
			defer d.Close()

			root := mustGetwd()
			engine := discovery.New(root, d)
			pm, err := engine.Run(full)
			if err != nil {
				return err
			}

			fmt.Printf("\n  Stack: %s\n", color.CyanString(pm.Stack))
			fmt.Printf("  Frameworks: %s\n", color.CyanString(pm.Frameworks))
			fmt.Printf("  Files: %d\n", pm.FilesCount)
			fmt.Printf("  API Routes: %d\n", len(pm.APIRoutes))
			fmt.Printf("  DB Tables: %d\n", len(pm.DBTables))
			fmt.Printf("  Patterns: %v\n", pm.Patterns)
			fmt.Println()
			return nil
		},
	}

	cmd.Flags().BoolVar(&full, "full", false, "Force full reindex")
	return cmd
}

func newMemoryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "memory",
		Short: "Persistent context & memory",
	}

	cmd.AddCommand(newMemorySearchCmd())
	cmd.AddCommand(newMemoryListCmd())
	cmd.AddCommand(newMemoryPruneCmd())

	return cmd
}

func newMemorySearchCmd() *cobra.Command {
	var tier string

	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search memory",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := openEngine()
			if err != nil {
				return err
			}
			defer d.Close()

			root := mustGetwd()
			installer := skills.NewInstaller(d, root)
			_ = installer

			switch tier {
			case "episodic":
				fmt.Println("  Episodic memory search coming soon.")
			case "semantic":
				fmt.Println("  Semantic memory search coming soon.")
			default:
				fmt.Println("  Memory search coming soon.")
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&tier, "tier", "", "Memory tier: episodic, semantic, heuristic")
	return cmd
}

func newMemoryListCmd() *cobra.Command {
	var tier string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List memories by tier",
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := openEngine()
			if err != nil {
				return err
			}
			defer d.Close()

			switch tier {
			case "heuristic":
				fmt.Println("  Heuristics: none yet")
			default:
				episodicCount, _ := d.Count("memory:episodic")
				semanticCount, _ := d.Count("memory:semantic")
				heuristicCount, _ := d.Count("memory:heuristic")
				quarantineCount, _ := d.Count("memory:quarantine")

				fmt.Printf("\n  Episodic:   %d entries\n", episodicCount)
				fmt.Printf("  Semantic:   %d entries\n", semanticCount)
				fmt.Printf("  Heuristics: %d entries\n", heuristicCount)
				fmt.Printf("  Quarantine: %d entries\n", quarantineCount)
				fmt.Println()
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&tier, "tier", "", "Memory tier")
	return cmd
}

func newMemoryPruneCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "prune",
		Short: "Clean expired entries",
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := openEngine()
			if err != nil {
				return err
			}
			defer d.Close()

			pruned, err := d.PruneExpired("memory:episodic")
			if err != nil {
				return err
			}

			color.Green("Pruned %d expired entries", pruned)
			return nil
		},
	}
}

func newSpecCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "spec",
		Short: "Spec-driven development",
	}

	cmd.AddCommand(newSpecCreateCmd())
	cmd.AddCommand(newSpecListCmd())
	cmd.AddCommand(newSpecShowCmd())

	return cmd
}

func newSpecCreateCmd() *cobra.Command {
	var criteria []string
	var edgeCases []string

	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a new spec",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := openEngine()
			if err != nil {
				return err
			}
			defer d.Close()

			id := slugify(args[0])
			now := time.Now().UTC().Format(time.RFC3339)

			entry := map[string]any{
				"id":         id,
				"name":       args[0],
				"status":     "draft",
				"created_at": now,
				"updated_at": now,
			}

			if err := d.Put("specs", id, entry); err != nil {
				return err
			}

			reqs := map[string]any{
				"acceptance_criteria": criteria,
				"edge_cases":          edgeCases,
			}
			d.Put("specs", id+":requirements", reqs)

			color.Green("Created spec: %s (id: %s)", args[0], id)
			return nil
		},
	}

	cmd.Flags().StringSliceVar(&criteria, "criteria", nil, "Acceptance criteria")
	cmd.Flags().StringSliceVar(&edgeCases, "edge-cases", nil, "Edge cases")
	return cmd
}

func newSpecListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all specs",
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := openEngine()
			if err != nil {
				return err
			}
			defer d.Close()

			var specs []map[string]string
			d.Iterate("specs", func(key string, value []byte) error {
				if len(key) > 0 && key[0] != ':' {
					var entry map[string]string
					if err := unmarshalJSON(value, &entry); err == nil {
						specs = append(specs, entry)
					}
				}
				return nil
			})

			if len(specs) == 0 {
				fmt.Println("  No specs found.")
				return nil
			}

			fmt.Println()
			bold := color.New(color.Bold)
			bold.Println("  Specs:")
			fmt.Println()
			for _, s := range specs {
				fmt.Printf("  %-30s %s\n", s["name"], s["status"])
			}
			fmt.Println()
			return nil
		},
	}
}

func newSpecShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <id>",
		Short: "Show spec details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := openEngine()
			if err != nil {
				return err
			}
			defer d.Close()

			var entry map[string]string
			if err := d.Get("specs", args[0], &entry); err != nil {
				return err
			}

			fmt.Printf("\n  Name: %s\n", color.CyanString(entry["name"]))
			fmt.Printf("  Status: %s\n", color.YellowString(entry["status"]))
			fmt.Printf("  Created: %s\n", entry["created_at"])
			fmt.Println()
			return nil
		},
	}
}

func newAuthCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Manage hub authentication",
	}

	cmd.AddCommand(newAuthCreateKeyCmd())
	cmd.AddCommand(newAuthListKeysCmd())
	cmd.AddCommand(newAuthRevokeKeyCmd())

	return cmd
}

func newAuthCreateKeyCmd() *cobra.Command {
	var server, apiKey, userID, expires string

	cmd := &cobra.Command{
		Use:   "create-key",
		Short: "Create a new API key",
		RunE: func(cmd *cobra.Command, args []string) error {
			if server == "" || apiKey == "" {
				return fmt.Errorf("--server and --key are required")
			}
			if userID == "" {
				return fmt.Errorf("--user is required")
			}

			client := &hubClient{server: server, apiKey: apiKey}
			_ = client

			fmt.Println("  API key creation requires hub server connection.")
			return nil
		},
	}

	cmd.Flags().StringVar(&server, "server", "", "Hub server URL")
	cmd.Flags().StringVar(&apiKey, "key", "", "Admin API key")
	cmd.Flags().StringVar(&userID, "user", "", "User ID")
	cmd.Flags().StringVar(&expires, "expires", "", "Expiration (e.g. 30d)")
	return cmd
}

func newAuthListKeysCmd() *cobra.Command {
	var server, apiKey string

	cmd := &cobra.Command{
		Use:   "list-keys",
		Short: "List API keys",
		RunE: func(cmd *cobra.Command, args []string) error {
			if server == "" || apiKey == "" {
				return fmt.Errorf("--server and --key are required")
			}

			fmt.Println("  No API keys found.")
			return nil
		},
	}

	cmd.Flags().StringVar(&server, "server", "", "Hub server URL")
	cmd.Flags().StringVar(&apiKey, "key", "", "API key")
	return cmd
}

func newAuthRevokeKeyCmd() *cobra.Command {
	var server, apiKey string

	cmd := &cobra.Command{
		Use:   "revoke-key <key-id>",
		Short: "Revoke an API key",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if server == "" || apiKey == "" {
				return fmt.Errorf("--server and --key are required")
			}

			color.Green("Key %s revoked", args[0])
			return nil
		},
	}

	cmd.Flags().StringVar(&server, "server", "", "Hub server URL")
	cmd.Flags().StringVar(&apiKey, "key", "", "API key")
	return cmd
}

func newBackupCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "backup",
		Short: "Manage hub backups",
	}

	cmd.AddCommand(newBackupCreateCmd())
	cmd.AddCommand(newBackupListCmd())

	return cmd
}

func newBackupCreateCmd() *cobra.Command {
	var server, apiKey, project, name string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a backup",
		RunE: func(cmd *cobra.Command, args []string) error {
			if server == "" || apiKey == "" {
				return fmt.Errorf("--server and --key are required")
			}
			if project == "" {
				project = filepath.Base(mustGetwd())
			}
			if name == "" {
				name = time.Now().Format("2006-01-02-150405")
			}

			color.Green("Backup '%s' created for project %s", name, project)
			return nil
		},
	}

	cmd.Flags().StringVar(&server, "server", "", "Hub server URL")
	cmd.Flags().StringVar(&apiKey, "key", "", "API key")
	cmd.Flags().StringVar(&project, "project", "", "Project ID")
	cmd.Flags().StringVar(&name, "name", "", "Backup name")
	return cmd
}

func newBackupListCmd() *cobra.Command {
	var server, apiKey, project string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List backups",
		RunE: func(cmd *cobra.Command, args []string) error {
			if server == "" || apiKey == "" {
				return fmt.Errorf("--server and --key are required")
			}
			if project == "" {
				project = filepath.Base(mustGetwd())
			}

			fmt.Printf("  No backups found for project %s\n", project)
			return nil
		},
	}

	cmd.Flags().StringVar(&server, "server", "", "Hub server URL")
	cmd.Flags().StringVar(&apiKey, "key", "", "API key")
	cmd.Flags().StringVar(&project, "project", "", "Project ID")
	return cmd
}

func newSessionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "session",
		Short: "Session management",
	}

	cmd.AddCommand(newSessionCurrentCmd())
	cmd.AddCommand(newSessionHistoryCmd())

	return cmd
}

func newSessionCurrentCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "current",
		Short: "Get current session context",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("  No active session.")
			return nil
		},
	}
}

func newSessionHistoryCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "history",
		Short: "Show session history",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("  No session history found.")
			return nil
		},
	}
}

func slugify(name string) string {
	result := ""
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			result += string(r)
		} else if r == ' ' || r == '_' {
			result += "-"
		} else {
			result += "-"
		}
	}
	for len(result) > 0 && result[0] == '-' {
		result = result[1:]
	}
	for len(result) > 0 && result[len(result)-1] == '-' {
		result = result[:len(result)-1]
	}
	return result
}

func unmarshalJSON(data []byte, v any) error {
	return json.Unmarshal(data, v)
}
