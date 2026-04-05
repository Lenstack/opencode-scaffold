package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/Lenstack/opencode-scaffold/internal/engine/config"
	"github.com/Lenstack/opencode-scaffold/internal/hub"
)

func newHubCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "hub",
		Short: "Team collaboration hub",
		Long: `Manage team workspaces, members, and template synchronization.

Examples:
  ocs hub serve                          # Start hub server
  ocs hub connect http://hub.company.com # Connect to remote hub
  ocs hub status                         # Check hub status
  ocs hub workspace list                 # List workspaces
  ocs hub team list                      # List teams
  ocs hub member list                    # List team members
  ocs hub template sync                  # Sync workspace template
`,
	}

	cmd.AddCommand(newHubServeCmd())
	cmd.AddCommand(newHubConnectCmd())
	cmd.AddCommand(newHubStatusCmd())
	cmd.AddCommand(newHubWorkspaceCmd())
	cmd.AddCommand(newHubTeamCmd())
	cmd.AddCommand(newHubMemberCmd())
	cmd.AddCommand(newHubTemplateCmd())

	return cmd
}

func newHubServeCmd() *cobra.Command {
	var port int
	var data string

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the hub server",
		RunE: func(cmd *cobra.Command, args []string) error {
			addr := fmt.Sprintf(":%d", port)
			if data == "" {
				home, _ := os.UserHomeDir()
				data = filepath.Join(home, ".ocs", "data")
			}
			if err := os.MkdirAll(data, 0755); err != nil {
				return fmt.Errorf("create data dir: %w", err)
			}

			store, err := hub.New(filepath.Join(data, "ocs.db"))
			if err != nil {
				return err
			}
			defer store.Close()

			srv := hub.NewServer(store, addr)
			return srv.Serve()
		},
	}

	cmd.Flags().IntVar(&port, "port", 8080, "HTTP port")
	cmd.Flags().StringVar(&data, "data", "", "Data directory")
	return cmd
}

func newHubConnectCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "connect <url>",
		Short: "Connect to a remote hub",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			url := args[0]
			fmt.Printf("  Connecting to %s...\n", url)

			client := &hubClient{server: url}
			health, err := client.health()
			if err != nil {
				return fmt.Errorf("cannot connect to hub: %w", err)
			}

			color.Green("Connected to hub: %s", url)
			fmt.Printf("  Status:  %s\n", health["status"])
			fmt.Printf("  Version: %s\n", health["version"])
			return nil
		},
	}
}

func newHubStatusCmd() *cobra.Command {
	var server string

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Check hub status",
		RunE: func(cmd *cobra.Command, args []string) error {
			if server == "" {
				server = "http://localhost:8080"
			}

			client := &hubClient{server: server}
			health, err := client.health()
			if err != nil {
				return fmt.Errorf("hub unreachable: %w", err)
			}

			fmt.Println()
			bold := color.New(color.Bold)
			bold.Println("  Hub Status:")
			fmt.Println()
			fmt.Printf("  Status:    %s\n", color.GreenString(health["status"]))
			fmt.Printf("  Version:   %s\n", health["version"])
			fmt.Printf("  Time:      %s\n", health["timestamp"])
			fmt.Println()
			return nil
		},
	}

	cmd.Flags().StringVar(&server, "server", "", "Hub server URL")
	return cmd
}

func newHubWorkspaceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "workspace",
		Short: "Manage workspaces (admin)",
	}

	cmd.AddCommand(newHubWorkspaceListCmd())
	cmd.AddCommand(newHubWorkspaceCreateCmd())
	cmd.AddCommand(newHubWorkspaceShowCmd())
	cmd.AddCommand(newHubWorkspaceRemoveCmd())

	return cmd
}

func newHubWorkspaceListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List workspaces",
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := openEngine()
			if err != nil {
				return err
			}
			defer d.Close()

			var workspaces []map[string]string
			d.Iterate("hub:workspaces", func(key string, value []byte) error {
				var ws map[string]string
				if err := json.Unmarshal(value, &ws); err == nil {
					workspaces = append(workspaces, ws)
				}
				return nil
			})

			if len(workspaces) == 0 {
				fmt.Println("  No workspaces found.")
				return nil
			}

			fmt.Println()
			bold := color.New(color.Bold)
			bold.Println("  Workspaces:")
			fmt.Println()

			for _, ws := range workspaces {
				fmt.Printf("  %-25s created: %-20s by: %s\n",
					cyan.Sprint(ws["name"]), ws["created_at"], ws["created_by"])
			}
			fmt.Println()
			return nil
		},
	}
}

func newHubWorkspaceCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a workspace",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := openEngine()
			if err != nil {
				return err
			}
			defer d.Close()

			root := mustGetwd()
			ws := map[string]any{
				"name":             args[0],
				"created_at":       time.Now().UTC().Format(time.RFC3339),
				"created_by":       "user",
				"template_version": 1,
			}

			id := fmt.Sprintf("ws-%d", time.Now().UnixNano())
			if err := d.Put("hub:workspaces", id, ws); err != nil {
				return err
			}

			tracker := config.NewTracker(d, root)
			if err := tracker.TrackAllConfigs("admin", "workspace-init"); err != nil {
				return err
			}

			color.Green("Created workspace: %s", args[0])
			return nil
		},
	}

	return cmd
}

func newHubWorkspaceShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <name>",
		Short: "Show workspace details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := openEngine()
			if err != nil {
				return err
			}
			defer d.Close()

			var found bool
			d.Iterate("hub:workspaces", func(key string, value []byte) error {
				var ws map[string]string
				if err := json.Unmarshal(value, &ws); err == nil && ws["name"] == args[0] {
					fmt.Println()
					bold := color.New(color.Bold)
					bold.Printf("  Workspace: %s\n", args[0])
					fmt.Println()
					fmt.Printf("  Created:  %s\n", ws["created_at"])
					fmt.Printf("  By:       %s\n", ws["created_by"])
					fmt.Printf("  Template: v%s\n", ws["template_version"])
					fmt.Println()
					found = true
				}
				return nil
			})

			if !found {
				return fmt.Errorf("workspace %s not found", args[0])
			}
			return nil
		},
	}
}

func newHubWorkspaceRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove <name>",
		Short: "Remove a workspace",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := openEngine()
			if err != nil {
				return err
			}
			defer d.Close()

			var toDelete []string
			d.Iterate("hub:workspaces", func(key string, value []byte) error {
				var ws map[string]string
				if err := json.Unmarshal(value, &ws); err == nil && ws["name"] == args[0] {
					toDelete = append(toDelete, key)
				}
				return nil
			})

			for _, key := range toDelete {
				d.Delete("hub:workspaces", key)
			}

			color.Green("Removed workspace: %s", args[0])
			return nil
		},
	}
}

func newHubTeamCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "team",
		Short: "Manage teams (admin)",
	}

	cmd.AddCommand(newHubTeamListCmd())
	cmd.AddCommand(newHubTeamCreateCmd())
	cmd.AddCommand(newHubTeamShowCmd())
	cmd.AddCommand(newHubTeamRemoveCmd())

	return cmd
}

func newHubTeamListCmd() *cobra.Command {
	var workspace string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List teams",
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := openEngine()
			if err != nil {
				return err
			}
			defer d.Close()

			var teams []map[string]string
			d.Iterate("hub:teams", func(key string, value []byte) error {
				var team map[string]string
				if err := json.Unmarshal(value, &team); err == nil {
					if workspace == "" || team["workspace"] == workspace {
						teams = append(teams, team)
					}
				}
				return nil
			})

			if len(teams) == 0 {
				fmt.Println("  No teams found.")
				return nil
			}

			fmt.Println()
			bold := color.New(color.Bold)
			bold.Println("  Teams:")
			fmt.Println()

			for _, t := range teams {
				fmt.Printf("  %-25s workspace: %-20s created: %s\n",
					cyan.Sprint(t["name"]), t["workspace"], t["created_at"])
			}
			fmt.Println()
			return nil
		},
	}

	cmd.Flags().StringVar(&workspace, "workspace", "", "Filter by workspace")
	return cmd
}

func newHubTeamCreateCmd() *cobra.Command {
	var workspace string

	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a team",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if workspace == "" {
				return fmt.Errorf("--workspace is required")
			}

			d, err := openEngine()
			if err != nil {
				return err
			}
			defer d.Close()

			team := map[string]any{
				"name":       args[0],
				"workspace":  workspace,
				"created_at": time.Now().UTC().Format(time.RFC3339),
				"created_by": "user",
			}

			id := fmt.Sprintf("team-%d", time.Now().UnixNano())
			if err := d.Put("hub:teams", id, team); err != nil {
				return err
			}

			color.Green("Created team: %s in workspace %s", args[0], workspace)
			return nil
		},
	}

	cmd.Flags().StringVar(&workspace, "workspace", "", "Workspace name")
	_ = cmd.MarkFlagRequired("workspace")
	return cmd
}

func newHubTeamShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <name>",
		Short: "Show team details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := openEngine()
			if err != nil {
				return err
			}
			defer d.Close()

			var found bool
			d.Iterate("hub:teams", func(key string, value []byte) error {
				var team map[string]string
				if err := json.Unmarshal(value, &team); err == nil && team["name"] == args[0] {
					fmt.Println()
					bold := color.New(color.Bold)
					bold.Printf("  Team: %s\n", args[0])
					fmt.Println()
					fmt.Printf("  Workspace: %s\n", team["workspace"])
					fmt.Printf("  Created:   %s\n", team["created_at"])
					fmt.Printf("  By:        %s\n", team["created_by"])
					fmt.Println()
					found = true
				}
				return nil
			})

			if !found {
				return fmt.Errorf("team %s not found", args[0])
			}
			return nil
		},
	}
}

func newHubTeamRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove <name>",
		Short: "Remove a team",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := openEngine()
			if err != nil {
				return err
			}
			defer d.Close()

			var toDelete []string
			d.Iterate("hub:teams", func(key string, value []byte) error {
				var team map[string]string
				if err := json.Unmarshal(value, &team); err == nil && team["name"] == args[0] {
					toDelete = append(toDelete, key)
				}
				return nil
			})

			for _, key := range toDelete {
				d.Delete("hub:teams", key)
			}

			color.Green("Removed team: %s", args[0])
			return nil
		},
	}
}

func newHubMemberCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "member",
		Short: "Manage team members (admin)",
	}

	cmd.AddCommand(newHubMemberListCmd())
	cmd.AddCommand(newHubMemberInviteCmd())
	cmd.AddCommand(newHubMemberRemoveCmd())
	cmd.AddCommand(newHubMemberRoleCmd())

	return cmd
}

func newHubMemberListCmd() *cobra.Command {
	var teamName string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List team members",
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := openEngine()
			if err != nil {
				return err
			}
			defer d.Close()

			var members []map[string]string
			d.Iterate("hub:members", func(key string, value []byte) error {
				var m map[string]string
				if err := json.Unmarshal(value, &m); err == nil {
					if teamName == "" || m["team"] == teamName {
						members = append(members, m)
					}
				}
				return nil
			})

			if len(members) == 0 {
				fmt.Println("  No members found.")
				return nil
			}

			fmt.Println()
			bold := color.New(color.Bold)
			bold.Println("  Members:")
			fmt.Println()

			for _, m := range members {
				fmt.Printf("  %-30s team: %-15s role: %-10s status: %s\n",
					cyan.Sprint(m["email"]), m["team"], m["role"], m["status"])
			}
			fmt.Println()
			return nil
		},
	}

	cmd.Flags().StringVar(&teamName, "team", "", "Filter by team")
	return cmd
}

func newHubMemberInviteCmd() *cobra.Command {
	var teamName string

	cmd := &cobra.Command{
		Use:   "invite <email>",
		Short: "Invite a member to a team",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if teamName == "" {
				return fmt.Errorf("--team is required")
			}

			d, err := openEngine()
			if err != nil {
				return err
			}
			defer d.Close()

			member := map[string]any{
				"email":      args[0],
				"team":       teamName,
				"role":       "viewer",
				"invited_at": time.Now().UTC().Format(time.RFC3339),
				"status":     "invited",
			}

			id := fmt.Sprintf("member-%d", time.Now().UnixNano())
			if err := d.Put("hub:members", id, member); err != nil {
				return err
			}

			color.Green("Invited %s to team %s", args[0], teamName)
			return nil
		},
	}

	cmd.Flags().StringVar(&teamName, "team", "", "Team name")
	_ = cmd.MarkFlagRequired("team")
	return cmd
}

func newHubMemberRemoveCmd() *cobra.Command {
	var teamName string

	cmd := &cobra.Command{
		Use:   "remove <email>",
		Short: "Remove a member from a team",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if teamName == "" {
				return fmt.Errorf("--team is required")
			}

			d, err := openEngine()
			if err != nil {
				return err
			}
			defer d.Close()

			var toDelete []string
			d.Iterate("hub:members", func(key string, value []byte) error {
				var m map[string]string
				if err := json.Unmarshal(value, &m); err == nil && m["email"] == args[0] && m["team"] == teamName {
					toDelete = append(toDelete, key)
				}
				return nil
			})

			for _, key := range toDelete {
				d.Delete("hub:members", key)
			}

			color.Green("Removed %s from team %s", args[0], teamName)
			return nil
		},
	}

	cmd.Flags().StringVar(&teamName, "team", "", "Team name")
	_ = cmd.MarkFlagRequired("team")
	return cmd
}

func newHubMemberRoleCmd() *cobra.Command {
	var teamName string

	cmd := &cobra.Command{
		Use:   "role <email> <role>",
		Short: "Set member role (admin/editor/viewer)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if teamName == "" {
				return fmt.Errorf("--team is required")
			}

			role := args[1]
			if role != "admin" && role != "editor" && role != "viewer" {
				return fmt.Errorf("invalid role: %s (must be admin, editor, or viewer)", role)
			}

			d, err := openEngine()
			if err != nil {
				return err
			}
			defer d.Close()

			d.Iterate("hub:members", func(key string, value []byte) error {
				var m map[string]any
				if err := json.Unmarshal(value, &m); err == nil {
					if m["email"] == args[0] && m["team"] == teamName {
						m["role"] = role
						d.Put("hub:members", key, m)
					}
				}
				return nil
			})

			color.Green("Set %s role to %s in team %s", args[0], role, teamName)
			return nil
		},
	}

	cmd.Flags().StringVar(&teamName, "team", "", "Team name")
	_ = cmd.MarkFlagRequired("team")
	return cmd
}

func newHubTemplateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "template",
		Short: "Manage workspace templates",
		Long: `Sync, update, and manage workspace templates.

Examples:
  ocs hub template show                  # Show workspace template
  ocs hub template sync                  # Sync local from workspace
  ocs hub template diff                  # Show diff local vs workspace
  ocs hub template update                # Update workspace template (admin)
  ocs hub template history               # Show template version history
  ocs hub template rollback v1           # Rollback to version (admin)
`,
	}

	cmd.AddCommand(newHubTemplateShowCmd())
	cmd.AddCommand(newHubTemplateSyncCmd())
	cmd.AddCommand(newHubTemplateDiffCmd())
	cmd.AddCommand(newHubTemplateForceSyncCmd())
	cmd.AddCommand(newHubTemplateUpdateCmd())
	cmd.AddCommand(newHubTemplateHistoryCmd())
	cmd.AddCommand(newHubTemplateRollbackCmd())

	return cmd
}

func newHubTemplateShowCmd() *cobra.Command {
	var workspace string

	cmd := &cobra.Command{
		Use:   "show",
		Short: "Show workspace template",
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := openEngine()
			if err != nil {
				return err
			}
			defer d.Close()

			if workspace == "" {
				workspace = "default"
			}

			var tpl map[string]any
			if err := d.Get("hub:templates", workspace, &tpl); err != nil {
				return fmt.Errorf("no template found for workspace %s", workspace)
			}

			fmt.Println()
			bold := color.New(color.Bold)
			bold.Printf("  Workspace Template: %s\n", workspace)
			fmt.Println()
			fmt.Printf("  Version:   %v\n", tpl["version"])
			fmt.Printf("  Updated:   %v\n", tpl["updated_at"])
			fmt.Printf("  By:        %v\n", tpl["updated_by"])
			fmt.Println()

			if agents, ok := tpl["agents"].(map[string]any); ok {
				fmt.Printf("  Agents (%d):\n", len(agents))
				for name := range agents {
					fmt.Printf("    - %s\n", name)
				}
				fmt.Println()
			}

			if skills, ok := tpl["skills"].(map[string]any); ok {
				fmt.Printf("  Skills (%d):\n", len(skills))
				for name := range skills {
					fmt.Printf("    - %s\n", name)
				}
				fmt.Println()
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&workspace, "workspace", "", "Workspace name")
	return cmd
}

func newHubTemplateSyncCmd() *cobra.Command {
	var workspace string

	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Sync local config from workspace template",
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := openEngine()
			if err != nil {
				return err
			}
			defer d.Close()

			if workspace == "" {
				workspace = "default"
			}

			root := mustGetwd()

			var tpl map[string]any
			if err := d.Get("hub:templates", workspace, &tpl); err != nil {
				return fmt.Errorf("no workspace template found")
			}

			if agents, ok := tpl["agents"].(map[string]any); ok {
				for name, content := range agents {
					path := filepath.Join(root, ".opencode", "agents", name)
					os.MkdirAll(filepath.Dir(path), 0755)
					os.WriteFile(path, []byte(content.(string)), 0644)
					fmt.Printf("  %s %s\n", color.GreenString("+"), path)
				}
			}

			if rules, ok := tpl["rules"].(string); ok {
				path := filepath.Join(root, "AGENTS.md")
				os.WriteFile(path, []byte(rules), 0644)
				fmt.Printf("  %s %s\n", color.GreenString("+"), path)
			}

			sync := map[string]any{
				"last_sync":      time.Now().UTC().Format(time.RFC3339),
				"remote_version": tpl["version"],
				"status":         "synced",
			}
			d.Put("hub:sync", workspace+":user", sync)

			fmt.Println()
			color.Green("Synced from workspace template (v%v)", tpl["version"])
			return nil
		},
	}

	cmd.Flags().StringVar(&workspace, "workspace", "", "Workspace name")
	return cmd
}

func newHubTemplateDiffCmd() *cobra.Command {
	var workspace string

	cmd := &cobra.Command{
		Use:   "diff",
		Short: "Show diff between local and workspace template",
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := openEngine()
			if err != nil {
				return err
			}
			defer d.Close()

			if workspace == "" {
				workspace = "default"
			}

			var tpl map[string]any
			if err := d.Get("hub:templates", workspace, &tpl); err != nil {
				return fmt.Errorf("no workspace template found")
			}

			root := mustGetwd()
			tracker := config.NewTracker(d, root)
			configs, _ := tracker.ListConfigs()

			hasDiff := false
			for _, c := range configs {
				if tplContent, ok := tpl[c.Path]; ok {
					if c.Content != tplContent {
						fmt.Printf("  %s %s\n", color.YellowString("~"), c.Path)
						hasDiff = true
					}
				} else {
					fmt.Printf("  %s %s (local only)\n", color.RedString("-"), c.Path)
					hasDiff = true
				}
			}

			if !hasDiff {
				color.Green("  Local config matches workspace template")
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&workspace, "workspace", "", "Workspace name")
	return cmd
}

func newHubTemplateForceSyncCmd() *cobra.Command {
	var workspace string

	cmd := &cobra.Command{
		Use:   "force-sync",
		Short: "Force sync (overwrite local with workspace template)",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("  This will overwrite local config with workspace template.")
			fmt.Print("  Continue? [y/N] ")
			var answer string
			fmt.Scanln(&answer)
			if answer != "y" && answer != "Y" {
				fmt.Println("  Cancelled.")
				return nil
			}

			// Reuse sync command
			syncCmd := newHubTemplateSyncCmd()
			syncCmd.Flags().Set("workspace", workspace)
			return syncCmd.RunE(cmd, args)
		},
	}

	cmd.Flags().StringVar(&workspace, "workspace", "", "Workspace name")
	return cmd
}

func newHubTemplateUpdateCmd() *cobra.Command {
	var workspace string

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update workspace template from local (admin)",
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := openEngine()
			if err != nil {
				return err
			}
			defer d.Close()

			if workspace == "" {
				workspace = "default"
			}

			root := mustGetwd()
			tracker := config.NewTracker(d, root)

			var tpl map[string]any
			var version int
			if err := d.Get("hub:templates", workspace, &tpl); err == nil {
				if v, ok := tpl["version"].(float64); ok {
					version = int(v)
				}
			}
			version++

			configs, _ := tracker.ListConfigs()
			template := map[string]any{
				"version":    version,
				"updated_at": time.Now().UTC().Format(time.RFC3339),
				"updated_by": "admin",
			}

			agents := map[string]any{}
			skills := map[string]any{}
			for _, c := range configs {
				if strings.HasPrefix(c.Path, ".opencode/agents/") {
					name := strings.TrimPrefix(c.Path, ".opencode/agents/")
					agents[name] = c.Content
				} else if strings.HasPrefix(c.Path, ".opencode/skills/") {
					name := strings.TrimPrefix(c.Path, ".opencode/skills/")
					name = strings.TrimSuffix(name, "/SKILL.md")
					skills[name] = c.Content
				} else if c.Path == "AGENTS.md" {
					template["rules"] = c.Content
				}
			}

			template["agents"] = agents
			template["skills"] = skills

			if err := d.Put("hub:templates", workspace, template); err != nil {
				return err
			}

			color.Green("Updated workspace template to v%d", version)
			return nil
		},
	}

	cmd.Flags().StringVar(&workspace, "workspace", "", "Workspace name")
	return cmd
}

func newHubTemplateHistoryCmd() *cobra.Command {
	var workspace string

	cmd := &cobra.Command{
		Use:   "history",
		Short: "Show template version history",
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := openEngine()
			if err != nil {
				return err
			}
			defer d.Close()

			if workspace == "" {
				workspace = "default"
			}

			var tpl map[string]any
			if err := d.Get("hub:templates", workspace, &tpl); err != nil {
				return fmt.Errorf("no template found")
			}

			fmt.Println()
			bold := color.New(color.Bold)
			bold.Println("  Template History:")
			fmt.Println()
			fmt.Printf("  v%-3d  %s  by: %v\n", tpl["version"], tpl["updated_at"], tpl["updated_by"])
			fmt.Println()

			return nil
		},
	}

	cmd.Flags().StringVar(&workspace, "workspace", "", "Workspace name")
	return cmd
}

func newHubTemplateRollbackCmd() *cobra.Command {
	var workspace string

	cmd := &cobra.Command{
		Use:   "rollback <version>",
		Short: "Rollback template to previous version (admin)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := openEngine()
			if err != nil {
				return err
			}
			defer d.Close()

			if workspace == "" {
				workspace = "default"
			}

			var history []map[string]any
			d.Iterate("hub:template_history", func(key string, value []byte) error {
				var entry map[string]any
				if err := json.Unmarshal(value, &entry); err == nil {
					if v, ok := entry["version"].(float64); ok && int(v) == parseInt(args[0]) {
						history = append(history, entry)
					}
				}
				return nil
			})

			if len(history) == 0 {
				return fmt.Errorf("version %s not found in history", args[0])
			}

			entry := history[0]
			if err := d.Put("hub:templates", workspace, entry); err != nil {
				return err
			}

			color.Green("Rolled back template to version %s", args[0])
			return nil
		},
	}

	cmd.Flags().StringVar(&workspace, "workspace", "", "Workspace name")
	return cmd
}

func parseInt(s string) int {
	var n int
	fmt.Sscanf(s, "%d", &n)
	return n
}
