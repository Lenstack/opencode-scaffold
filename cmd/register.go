package cmd

import (
	"fmt"
	"net/http"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/Lenstack/opencode-scaffold/internal/domain/project"
)

func newRegisterCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "register",
		Short: "Register project with a mothership hub",
		Long: `Register this project with a hub server to enable knowledge sharing.

After registration, use "ocs sync pull" to fetch global knowledge and
"ocs sync push" to share your project's learnings with the workspace.

Examples:
  ocs register --hub http://hub.company.com --workspace myteam
  ocs register --hub http://localhost:8080 --workspace default
  ocs register --hub http://hub.company.com --workspace myteam --key ocs-xxx
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			hubURL, _ := cmd.Flags().GetString("hub")
			workspace, _ := cmd.Flags().GetString("workspace")
			apiKey, _ := cmd.Flags().GetString("key")

			if hubURL == "" {
				return fmt.Errorf("--hub is required")
			}
			if workspace == "" {
				workspace = "default"
			}

			root := mustGetwd()
			d, err := openEngine()
			if err != nil {
				return err
			}
			defer d.Close()

			if apiKey == "" {
				fmt.Println("  No API key provided (--key). Hub operations will be unauthenticated.")
				fmt.Println("  Create a key on the hub server and re-register for authenticated access.")
			}

			name := filepath.Base(root)
			mgr := project.NewManager(d, root)

			reg, err := mgr.Register(name, hubURL, workspace, apiKey)
			if err != nil {
				return err
			}

			if apiKey != "" {
				_, err := http.Get(hubURL + "/api/health")
				if err != nil {
					fmt.Printf("  Warning: hub unreachable at %s, registration saved locally\n", hubURL)
				} else {
					color.Green("  Hub reachable and healthy")
				}
			}

			fmt.Println()
			bold := color.New(color.Bold)
			bold.Println("  Project Registered:")
			fmt.Println()
			fmt.Printf("  ID:           %s\n", color.CyanString(reg.ProjectID))
			fmt.Printf("  Name:         %s\n", reg.ProjectName)
			fmt.Printf("  Hub:          %s\n", reg.HubURL)
			fmt.Printf("  Workspace:    %s\n", reg.Workspace)
			fmt.Printf("  Registered:   %s\n", reg.RegisteredAt)
			fmt.Println()
			fmt.Println("  Next steps:")
			fmt.Printf("    %s\n", color.CyanString("ocs sync pull"))
			fmt.Printf("    %s\n", color.CyanString("ocs sync push"))
			fmt.Println()

			return nil
		},
	}

	cmd.Flags().String("hub", "", "Hub server URL (required)")
	cmd.Flags().String("workspace", "default", "Workspace name")
	cmd.Flags().String("key", "", "API key for authentication")
	_ = cmd.MarkFlagRequired("hub")

	return cmd
}

func newUnregisterCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "unregister",
		Short: "Remove project registration from hub",
		Long: `Disconnect this project from the hub server.

Local knowledge (memory, sessions, learnings) is preserved.
Only the hub linkage and API key are removed.
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			root := mustGetwd()
			d, err := openEngine()
			if err != nil {
				return err
			}
			defer d.Close()

			mgr := project.NewManager(d, root)
			if !mgr.IsRegistered() {
				return fmt.Errorf("project is not registered")
			}

			if err := mgr.Unregister(); err != nil {
				return err
			}

			color.Green("  Project unregistered from hub")
			fmt.Println("  Local knowledge preserved. Run 'ocs register' to reconnect.")
			return nil
		},
	}
}

func newProjectStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show project registration and sync status",
		RunE: func(cmd *cobra.Command, args []string) error {
			root := mustGetwd()
			d, err := openEngine()
			if err != nil {
				return err
			}
			defer d.Close()

			mgr := project.NewManager(d, root)

			fmt.Println()
			bold := color.New(color.Bold)
			bold.Println("  Project Status:")
			fmt.Println()

			reg, err := mgr.Load()
			if err != nil {
				fmt.Printf("  Registration: %s\n", color.RedString("not registered"))
				fmt.Println()
				fmt.Println("  Run 'ocs register --hub <url>' to connect to a hub.")
				fmt.Println()
				return nil
			}

			fmt.Printf("  Registration: %s\n", color.GreenString("registered"))
			fmt.Printf("  Project ID:   %s\n", color.CyanString(reg.ProjectID))
			fmt.Printf("  Name:         %s\n", reg.ProjectName)
			fmt.Printf("  Hub:          %s\n", reg.HubURL)
			fmt.Printf("  Workspace:    %s\n", reg.Workspace)
			fmt.Printf("  API Key:      %s\n", maskKey(reg.APIKey))
			fmt.Printf("  Last Sync:    %s\n", orNever(reg.LastSync))
			fmt.Println()

			if reg.HubURL != "" {
				client := &hubClient{server: reg.HubURL, apiKey: reg.APIKey}
				health, err := client.health()
				if err != nil {
					fmt.Printf("  Hub Status:   %s\n", color.RedString("unreachable"))
				} else {
					fmt.Printf("  Hub Status:   %s (v%s)\n", color.GreenString("connected"), health["version"])
				}
			}

			fmt.Println()
			return nil
		},
	}
}

func maskKey(key string) string {
	if key == "" {
		return "(none)"
	}
	if len(key) <= 8 {
		return "****"
	}
	return key[:6] + "..." + key[len(key)-4:]
}

func orNever(ts string) string {
	if ts == "" {
		return color.YellowString("never")
	}
	return ts
}
