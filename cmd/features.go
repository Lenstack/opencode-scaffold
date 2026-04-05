package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/Lenstack/opencode-scaffold/internal/hub"
	"github.com/Lenstack/opencode-scaffold/internal/engine/discovery"
	"github.com/Lenstack/opencode-scaffold/internal/domain/memory"
	"github.com/Lenstack/opencode-scaffold/internal/domain/session"
	"github.com/Lenstack/opencode-scaffold/internal/domain/skill"
	"github.com/Lenstack/opencode-scaffold/internal/domain/spec"
)

func dataDir() string {
	wd, _ := os.Getwd()
	return filepath.Join(wd, ".opencode", "data")
}

func openDB() (*hub.Engine, error) {
	return hub.NewEngine(dataDir())
}

func newDiscoverCmd() *cobra.Command {
	var full bool

	cmd := &cobra.Command{
		Use:   "discover",
		Short: "Index project into LevelDB",
		Long:  "Scan the project and store file metadata, API routes, DB tables, and patterns in LevelDB.",
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := openDB()
			if err != nil {
				return err
			}
			defer d.Close()

			root, _ := os.Getwd()
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

func newSpecCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "spec",
		Short: "Spec-driven development",
	}

	cmd.AddCommand(newSpecCreateCmd())
	cmd.AddCommand(newSpecListCmd())
	cmd.AddCommand(newSpecShowCmd())
	cmd.AddCommand(newSpecValidateCmd())
	cmd.AddCommand(newSpecStatusCmd())
	cmd.AddCommand(newSpecArchiveCmd())

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
			d, err := openDB()
			if err != nil {
				return err
			}
			defer d.Close()

			mgr := spec.NewManager(d)
			entry, err := mgr.Create(args[0], hub.SpecRequirements{
				AcceptanceCriteria: criteria,
				EdgeCases:          edgeCases,
			})
			if err != nil {
				return err
			}

			color.Green("Created spec: %s (id: %s)", entry.Name, entry.ID)
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
			d, err := openDB()
			if err != nil {
				return err
			}
			defer d.Close()

			mgr := spec.NewManager(d)
			specs, err := mgr.List()
			if err != nil {
				return err
			}

			if len(specs) == 0 {
				fmt.Println("  No specs found.")
				return nil
			}

			fmt.Println()
			bold := color.New(color.Bold)
			bold.Println("  Specs:")
			fmt.Println()
			for _, s := range specs {
				statusColor := color.FgYellow
				if s.Status == "done" {
					statusColor = color.FgGreen
				} else if s.Status == "draft" {
					statusColor = color.FgCyan
				}
				fmt.Printf("  %-30s %s\n", s.Name, color.New(statusColor).Sprint(s.Status))
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
			d, err := openDB()
			if err != nil {
				return err
			}
			defer d.Close()

			mgr := spec.NewManager(d)
			entry, err := mgr.Get(args[0])
			if err != nil {
				return err
			}

			fmt.Printf("\n  Name: %s\n", color.CyanString(entry.Name))
			fmt.Printf("  Status: %s\n", color.YellowString(entry.Status))
			fmt.Printf("  Created: %s\n", entry.CreatedAt)

			reqs, _ := mgr.GetRequirements(args[0])
			if reqs != nil {
				fmt.Printf("\n  Acceptance Criteria:\n")
				for i, c := range reqs.AcceptanceCriteria {
					fmt.Printf("    %d. %s\n", i+1, c)
				}
				fmt.Printf("\n  Edge Cases:\n")
				for i, e := range reqs.EdgeCases {
					fmt.Printf("    %d. %s\n", i+1, e)
				}
			}
			fmt.Println()
			return nil
		},
	}
}

func newSpecValidateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "validate <id>",
		Short: "Validate implementation against spec",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := openDB()
			if err != nil {
				return err
			}
			defer d.Close()

			mgr := spec.NewManager(d)
			reqs, err := mgr.GetRequirements(args[0])
			if err != nil {
				return err
			}

			impl, err := mgr.GetImplementation(args[0])
			if err != nil {
				return fmt.Errorf("no implementation found — run implementation first")
			}

			var failed []string
			for _, c := range reqs.AcceptanceCriteria {
				found := false
				for _, f := range impl.Files {
					content, _ := os.ReadFile(f)
					if strings.Contains(string(content), c) {
						found = true
						break
					}
				}
				if !found {
					failed = append(failed, c)
				}
			}

			if len(failed) == 0 {
				color.Green("\n  All acceptance criteria met! Spec verified.\n")
				return mgr.Verify(args[0], hub.SpecVerification{
					Status:         "verified",
					Results:        reqs.AcceptanceCriteria,
					FailedCriteria: nil,
				})
			}

			color.Red("\n  Failed criteria:\n")
			for _, f := range failed {
				fmt.Printf("    ❌ %s\n", f)
			}
			fmt.Println()
			return mgr.Verify(args[0], hub.SpecVerification{
				Status:         "failed",
				FailedCriteria: failed,
			})
		},
	}
}

func newSpecStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status <id> --status <status>",
		Short: "Update spec status",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			status, _ := cmd.Flags().GetString("status")
			if status == "" {
				return fmt.Errorf("--status is required")
			}

			d, err := openDB()
			if err != nil {
				return err
			}
			defer d.Close()

			mgr := spec.NewManager(d)
			return mgr.UpdateStatus(args[0], status)
		},
	}
}

func newSpecArchiveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "archive <id>",
		Short: "Archive a completed spec",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := openDB()
			if err != nil {
				return err
			}
			defer d.Close()

			mgr := spec.NewManager(d)
			return mgr.Archive(args[0])
		},
	}
}

func newMemoryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "memory",
		Short: "Persistent context & memory",
	}

	cmd.AddCommand(newMemoryGetCmd())
	cmd.AddCommand(newMemorySearchCmd())
	cmd.AddCommand(newMemoryListCmd())
	cmd.AddCommand(newMemoryPruneCmd())

	return cmd
}

func newMemoryGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <key>",
		Short: "Get memory value",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := openDB()
			if err != nil {
				return err
			}
			defer d.Close()

			tier, _ := cmd.Flags().GetString("tier")
			ns := map[string]string{
				"episodic":   hub.NSMemoryEpisodic,
				"semantic":   hub.NSMemorySemantic,
				"heuristic":  hub.NSMemoryHeuristic,
				"quarantine": hub.NSMemoryQuarantine,
			}[tier]
			if ns == "" {
				ns = hub.NSMemorySemantic
			}

			var data map[string]any
			if err := d.Get(ns, args[0], &data); err != nil {
				return err
			}

			b, _ := json.MarshalIndent(data, "", "  ")
			fmt.Println(string(b))
			return nil
		},
	}
}

func newMemorySearchCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "search <query>",
		Short: "Search memory",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := openDB()
			if err != nil {
				return err
			}
			defer d.Close()

			mgr := memory.NewManager(d)
			tier, _ := cmd.Flags().GetString("tier")

			switch tier {
			case "episodic":
				results, _ := mgr.SearchEpisodic(args[0])
				b, _ := json.MarshalIndent(results, "", "  ")
				fmt.Println(string(b))
			case "semantic":
				results, _ := mgr.SearchSemantic(args[0], 0.5)
				b, _ := json.MarshalIndent(results, "", "  ")
				fmt.Println(string(b))
			default:
				episodic, _ := mgr.SearchEpisodic(args[0])
				semantic, _ := mgr.SearchSemantic(args[0], 0.5)
				all := map[string]any{
					"episodic": episodic,
					"semantic": semantic,
				}
				b, _ := json.MarshalIndent(all, "", "  ")
				fmt.Println(string(b))
			}

			return nil
		},
	}
}

func newMemoryListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List memories by tier",
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := openDB()
			if err != nil {
				return err
			}
			defer d.Close()

			tier, _ := cmd.Flags().GetString("tier")
			mgr := memory.NewManager(d)

			switch tier {
			case "heuristic":
				rules, _ := mgr.ListHeuristics()
				for _, r := range rules {
					fmt.Printf("  %s: %s (confidence: %.2f, active: %v)\n",
						r.ID, r.Rule, r.Confidence, r.Active)
				}
			default:
				episodicCount, _ := d.Count(hub.NSMemoryEpisodic)
				semanticCount, _ := d.Count(hub.NSMemorySemantic)
				heuristicCount, _ := d.Count(hub.NSMemoryHeuristic)
				quarantineCount, _ := d.Count(hub.NSMemoryQuarantine)

				fmt.Printf("\n  Episodic:   %d entries\n", episodicCount)
				fmt.Printf("  Semantic:   %d entries\n", semanticCount)
				fmt.Printf("  Heuristics: %d entries\n", heuristicCount)
				fmt.Printf("  Quarantine: %d entries\n", quarantineCount)
				fmt.Println()
			}

			return nil
		},
	}
}

func newMemoryPruneCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "prune",
		Short: "Clean expired entries",
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := openDB()
			if err != nil {
				return err
			}
			defer d.Close()

			mgr := memory.NewManager(d)
			pruned, err := mgr.PruneExpired()
			if err != nil {
				return err
			}

			color.Green("Pruned %d expired entries", pruned)
			return nil
		},
	}
}

func newSessionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "session",
		Short: "Session management",
	}

	cmd.AddCommand(newSessionCurrentCmd())
	cmd.AddCommand(newSessionListCmd())

	return cmd
}

func newSessionCurrentCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "current",
		Short: "Get current session context",
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := openDB()
			if err != nil {
				return err
			}
			defer d.Close()

			mgr := session.NewManager(d)
			s, err := mgr.GetCurrent()
			if err != nil {
				return err
			}

			b, _ := json.MarshalIndent(s, "", "  ")
			fmt.Println(string(b))
			return nil
		},
	}
}

func newSessionListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "history",
		Short: "Show session history",
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := openDB()
			if err != nil {
				return err
			}
			defer d.Close()

			mgr := session.NewManager(d)
			sessions, err := mgr.List()
			if err != nil {
				return err
			}

			fmt.Println()
			for _, s := range sessions {
				fmt.Printf("  %-20s %-10s %-20s\n", s.Title, s.Status, s.StartedAt)
			}
			fmt.Println()
			return nil
		},
	}
}

func newSkillCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "skill",
		Short: "Skill management & optimization",
	}

	cmd.AddCommand(newSkillListCmd())
	cmd.AddCommand(newSkillCreateCmd())
	cmd.AddCommand(newSkillOptimizeCmd())
	cmd.AddCommand(newSkillSuggestCmd())

	return cmd
}

func newSkillListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all skills",
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := openDB()
			if err != nil {
				return err
			}
			defer d.Close()

			mgr := skill.NewManager(d)
			skills, err := mgr.List()
			if err != nil {
				return err
			}

			fmt.Println()
			bold := color.New(color.Bold)
			bold.Println("  Skills:")
			fmt.Println()
			for _, s := range skills {
				fmt.Printf("  %-25s usage: %-4d effectiveness: %.2f\n",
					s.Name, s.UsageCount, s.Effectiveness)
			}
			fmt.Println()
			return nil
		},
	}
}

func newSkillCreateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "create <name>",
		Short: "Create new skill",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := openDB()
			if err != nil {
				return err
			}
			defer d.Close()

			mgr := skill.NewManager(d)
			entry, err := mgr.Create(args[0])
			if err != nil {
				return err
			}

			color.Green("Created skill: %s", entry.Name)
			return nil
		},
	}
}

func newSkillOptimizeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "optimize [name]",
		Short: "Optimize skills based on project knowledge",
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := openDB()
			if err != nil {
				return err
			}
			defer d.Close()

			mgr := skill.NewManager(d)
			skills, err := mgr.List()
			if err != nil {
				return err
			}

			if len(skills) == 0 {
				fmt.Println("  No skills to optimize.")
				return nil
			}

			fmt.Println()
			color.Green("Optimizing %d skills...", len(skills))
			fmt.Println()

			for _, s := range skills {
				if len(args) > 0 && s.Name != args[0] {
					continue
				}

				knowledge, _ := mgr.GetKnowledge(s.Name)
				if knowledge == nil {
					knowledge = &hub.SkillKnowledge{}
				}

				fmt.Printf("  %s: effectiveness %.2f → %.2f\n",
					s.Name, s.Effectiveness, s.Effectiveness+0.05)

				mgr.LogOptimization(s.Name, "auto-optimization", "usage-based", 0.05)
			}

			fmt.Println()
			return nil
		},
	}
}

func newSkillSuggestCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "suggest",
		Short: "Suggest new skills based on project patterns",
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := openDB()
			if err != nil {
				return err
			}
			defer d.Close()

			var pm hub.ProjectMap
			d.Get(hub.NSDiscovery, "project_map", &pm)

			mgr := skill.NewManager(d)
			suggestions := mgr.Suggest(pm.Stack)

			fmt.Println()
			bold := color.New(color.Bold)
			bold.Println("  Suggested skills:")
			fmt.Println()
			for _, s := range suggestions {
				fmt.Printf("    - %s\n", s)
			}
			fmt.Println()
			return nil
		},
	}
}
