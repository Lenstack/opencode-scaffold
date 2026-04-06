package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/Lenstack/opencode-scaffold/internal/domain/project"
	"github.com/Lenstack/opencode-scaffold/internal/hub"
)

func newSyncCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Sync knowledge with mothership hub",
		Long: `Push and pull knowledge between this project and the mothership hub.

Knowledge includes semantic memories, heuristics, sessions, and learned patterns.
Use "ocs sync auto" for a full pull-then-push cycle.

Examples:
  ocs sync pull          # Pull global knowledge from hub
  ocs sync push          # Push local learnings to hub
  ocs sync auto          # Pull then push in one command
  ocs sync status        # Show last sync times
`,
	}

	cmd.AddCommand(newSyncPullCmd())
	cmd.AddCommand(newSyncPushCmd())
	cmd.AddCommand(newSyncAutoCmd())
	cmd.AddCommand(newSyncStatusCmd())

	return cmd
}

func newSyncPullCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "pull",
		Short: "Pull knowledge from hub",
		RunE: func(cmd *cobra.Command, args []string) error {
			root := mustGetwd()
			d, err := openEngine()
			if err != nil {
				return err
			}
			defer d.Close()

			mgr := project.NewManager(d, root)
			reg, err := mgr.Load()
			if err != nil {
				return err
			}

			if reg.HubURL == "" {
				return fmt.Errorf("no hub configured, run 'ocs register'")
			}

			stack := reg.Stack
			if stack == "" {
				stack = detectStack(root)
			}

			client := hub.NewClient(reg.HubURL, reg.APIKey)
			syncEng := hub.NewSyncEngine(d, reg.ProjectID, reg.Workspace, stack)

			pulled, err := syncEng.Pull(client)
			if err != nil {
				return err
			}

			fmt.Println()
			bold := color.New(color.Bold)
			bold.Println("  Knowledge Pulled:")
			fmt.Println()
			fmt.Printf("  Items merged: %s\n", color.GreenString(fmt.Sprint(pulled)))
			fmt.Printf("  Workspace:    %s\n", reg.Workspace)
			fmt.Printf("  Stack:        %s\n", stack)
			fmt.Println()

			if err := mgr.UpdateSyncTime(); err != nil {
				return err
			}

			return nil
		},
	}
}

func newSyncPushCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "push",
		Short: "Push knowledge to hub",
		RunE: func(cmd *cobra.Command, args []string) error {
			root := mustGetwd()
			d, err := openEngine()
			if err != nil {
				return err
			}
			defer d.Close()

			mgr := project.NewManager(d, root)
			reg, err := mgr.Load()
			if err != nil {
				return err
			}

			if reg.HubURL == "" {
				return fmt.Errorf("no hub configured, run 'ocs register'")
			}

			stack := reg.Stack
			if stack == "" {
				stack = detectStack(root)
			}

			client := hub.NewClient(reg.HubURL, reg.APIKey)
			syncEng := hub.NewSyncEngine(d, reg.ProjectID, reg.Workspace, stack)

			pushed, err := syncEng.Push(client)
			if err != nil {
				return err
			}

			fmt.Println()
			bold := color.New(color.Bold)
			bold.Println("  Knowledge Pushed:")
			fmt.Println()
			fmt.Printf("  Items sent:   %s\n", color.GreenString("%d", pushed))
			fmt.Printf("  Workspace:    %s\n", reg.Workspace)
			fmt.Printf("  Stack:        %s\n", stack)
			fmt.Println()

			if err := mgr.UpdateSyncTime(); err != nil {
				return err
			}

			return nil
		},
	}
}

func newSyncAutoCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "auto",
		Short: "Full sync cycle (pull then push)",
		Long: `Run a full sync cycle: pull global knowledge first, then push local learnings.

This is the recommended command for autonomous agents. It ensures the project
has the latest global heuristics before starting work, and shares any new
learnings when work is complete.
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			root := mustGetwd()
			d, err := openEngine()
			if err != nil {
				return err
			}
			defer d.Close()

			mgr := project.NewManager(d, root)
			reg, err := mgr.Load()
			if err != nil {
				return err
			}

			if reg.HubURL == "" {
				return fmt.Errorf("no hub configured, run 'ocs register'")
			}

			stack := reg.Stack
			if stack == "" {
				stack = detectStack(root)
			}

			client := hub.NewClient(reg.HubURL, reg.APIKey)
			syncEng := hub.NewSyncEngine(d, reg.ProjectID, reg.Workspace, stack)

			fmt.Println()
			bold := color.New(color.Bold)
			bold.Println("  Sync Cycle:")
			fmt.Println()

			pulled, err := syncEng.Pull(client)
			if err != nil {
				fmt.Printf("  Pull: %s (%s)\n", color.RedString("failed"), err)
			} else {
				fmt.Printf("  Pull: %s (%d items merged)\n", color.GreenString("ok"), pulled)
			}

			pushed, err := syncEng.Push(client)
			if err != nil {
				fmt.Printf("  Push: %s (%s)\n", color.RedString("failed"), err)
			} else {
				fmt.Printf("  Push: %s (%d items sent)\n", color.GreenString("ok"), pushed)
			}

			fmt.Println()
			fmt.Printf("  Workspace: %s\n", reg.Workspace)
			fmt.Printf("  Stack:     %s\n", stack)
			fmt.Println()

			if err := mgr.UpdateSyncTime(); err != nil {
				return err
			}

			return nil
		},
	}
}

func newSyncStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show sync status",
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := openEngine()
			if err != nil {
				return err
			}
			defer d.Close()

			syncEng := hub.NewSyncEngine(d, "", "", "")
			status, err := syncEng.Status()
			if err != nil {
				return err
			}

			fmt.Println()
			bold := color.New(color.Bold)
			bold.Println("  Sync Status:")
			fmt.Println()
			fmt.Printf("  Last Pull:    %s\n", orNever(status.LastPull))
			fmt.Printf("  Last Push:    %s\n", orNever(status.LastPush))
			fmt.Printf("  Items Pulled: %d\n", status.PulledItems)
			fmt.Printf("  Items Pushed: %d\n", status.PushedItems)
			fmt.Printf("  Status:       %s\n", status.Status)
			fmt.Println()

			return nil
		},
	}
}

func detectStack(root string) string {
	files := []string{"go.mod", "package.json", "pyproject.toml", "requirements.txt", "Cargo.toml"}
	for _, f := range files {
		if _, err := os.Stat(filepath.Join(root, f)); err == nil {
			switch f {
			case "go.mod":
				return "go"
			case "package.json":
				return "node"
			case "pyproject.toml", "requirements.txt":
				return "python"
			case "Cargo.toml":
				return "rust"
			}
		}
	}
	return ""
}
