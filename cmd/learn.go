package cmd

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/Lenstack/opencode-scaffold/internal/engine/learn"
)

func newLearnCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "learn",
		Short: "Auto-learning system",
		Long: `Track outcomes, detect patterns, and auto-improve AI agent workflows.

The learning system tracks session outcomes, detects successful patterns,
promotes them to heuristics, and continuously improves recommendations.

Examples:
  ocs learn session my-session --outcome success --agents orchestrator,tester
  ocs learn patterns                    # Show detected patterns
  ocs learn heuristics                  # Show learned heuristics
  ocs learn skills                      # Show skill effectiveness
  ocs learn templates                   # Show template effectiveness
  ocs learn agents                      # Show agent performance
  ocs learn knowledge                   # Show extracted knowledge
  ocs learn run                         # Run full learning cycle
  ocs learn run --auto                  # Run and auto-apply improvements
`,
	}

	cmd.AddCommand(newLearnSessionCmd())
	cmd.AddCommand(newLearnPatternsCmd())
	cmd.AddCommand(newLearnHeuristicsCmd())
	cmd.AddCommand(newLearnSkillsCmd())
	cmd.AddCommand(newLearnTemplatesCmd())
	cmd.AddCommand(newLearnAgentsCmd())
	cmd.AddCommand(newLearnKnowledgeCmd())
	cmd.AddCommand(newLearnRunCmd())
	cmd.AddCommand(newLearnStatsCmd())

	return cmd
}

func newLearnSessionCmd() *cobra.Command {
	var (
		outcomeStr string
		agents     []string
		skills     []string
		template   string
		stack      string
		duration   int
		notes      string
	)

	cmd := &cobra.Command{
		Use:   "session <id>",
		Short: "Record session outcome",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := openEngine()
			if err != nil {
				return err
			}
			defer d.Close()

			engine := learn.NewEngine(d)

			outcome := learn.SessionOutcome{
				SessionID: args[0],
				Outcome:   outcomeStr,
				Agents:    agents,
				Skills:    skills,
				Template:  template,
				Stack:     stack,
				Duration:  duration,
				Notes:     notes,
			}

			if err := engine.RecordSession(outcome); err != nil {
				return err
			}

			color.Green("Recorded session %s: outcome=%s agents=%v skills=%v", args[0], outcomeStr, agents, skills)
			return nil
		},
	}

	cmd.Flags().StringVar(&outcomeStr, "outcome", "success", "Session outcome: success, failure, partial")
	cmd.Flags().StringSliceVar(&agents, "agents", nil, "Agents used")
	cmd.Flags().StringSliceVar(&skills, "skills", nil, "Skills loaded")
	cmd.Flags().StringVar(&template, "template", "", "Template used")
	cmd.Flags().StringVar(&stack, "stack", "", "Project stack")
	cmd.Flags().IntVar(&duration, "duration", 0, "Session duration (seconds)")
	cmd.Flags().StringVar(&notes, "notes", "", "Session notes")

	return cmd
}

func newLearnPatternsCmd() *cobra.Command {
	var category string
	var minOccurrences int

	cmd := &cobra.Command{
		Use:   "patterns",
		Short: "Show detected patterns",
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := openEngine()
			if err != nil {
				return err
			}
			defer d.Close()

			engine := learn.NewEngine(d)
			patterns, err := engine.GetPatterns(category, minOccurrences)
			if err != nil {
				return err
			}

			if len(patterns) == 0 {
				fmt.Println("  No patterns detected yet. Record some sessions first.")
				return nil
			}

			fmt.Println()
			bold := color.New(color.Bold)
			cyan := color.New(color.FgCyan)
			bold.Println("  Detected Patterns:")
			fmt.Println()

			for _, p := range patterns {
				status := ""
				if p.Promoted {
					status = color.GreenString(" [PROMOTED]")
				}
				fmt.Printf("  %-40s %s\n", cyan.Sprint(p.ID), status)
				fmt.Printf("    Category: %-12s Occurrences: %-3d Success: %.0f%%\n",
					p.Category, p.Occurrences, p.SuccessRate*100)
				fmt.Printf("    Last seen: %s\n", p.LastSeen)
				fmt.Println()
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&category, "category", "", "Filter by category: workflow, skill, template")
	cmd.Flags().IntVar(&minOccurrences, "min", 1, "Minimum occurrences")

	return cmd
}

func newLearnHeuristicsCmd() *cobra.Command {
	var minConfidence float64

	cmd := &cobra.Command{
		Use:   "heuristics",
		Short: "Show learned heuristics",
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := openEngine()
			if err != nil {
				return err
			}
			defer d.Close()

			engine := learn.NewEngine(d)
			heuristics, err := engine.GetHeuristics(minConfidence)
			if err != nil {
				return err
			}

			if len(heuristics) == 0 {
				fmt.Println("  No heuristics learned yet. Record more sessions.")
				return nil
			}

			fmt.Println()
			bold := color.New(color.Bold)
			bold.Println("  Learned Heuristics:")
			fmt.Println()

			for _, h := range heuristics {
				fmt.Printf("  %-30s confidence: %.2f  success: %.0f%%\n",
					cyan.Sprint(h.Rule), h.Confidence, h.SuccessRate*100)
				fmt.Printf("    Source: %s  Invocations: %d  Overrides: %d\n",
					h.Source, h.InvocationCount, h.OverrideCount)
				fmt.Printf("    %s\n", h.Rationale)
				fmt.Println()
			}

			return nil
		},
	}

	cmd.Flags().Float64Var(&minConfidence, "confidence", 0.0, "Minimum confidence (0.0-1.0)")

	return cmd
}

func newLearnSkillsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "skills",
		Short: "Show skill effectiveness",
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := openEngine()
			if err != nil {
				return err
			}
			defer d.Close()

			engine := learn.NewEngine(d)
			stats, err := engine.GetSkillStats()
			if err != nil {
				return err
			}

			if len(stats) == 0 {
				fmt.Println("  No skill stats yet. Record some sessions first.")
				return nil
			}

			fmt.Println()
			bold := color.New(color.Bold)
			bold.Println("  Skill Effectiveness:")
			fmt.Println()

			for _, s := range stats {
				bar := effectivenessBar(s.Effectiveness)
				fmt.Printf("  %-25s %s %.0f%%\n", cyan.Sprint(s.Name), bar, s.Effectiveness*100)
				fmt.Printf("    Usage: %d  Success: %d  Failure: %d  Last: %s\n",
					s.UsageCount, s.SuccessCount, s.FailureCount, s.LastUsed)
				fmt.Println()
			}

			return nil
		},
	}

	return cmd
}

func newLearnTemplatesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "templates",
		Short: "Show template effectiveness",
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := openEngine()
			if err != nil {
				return err
			}
			defer d.Close()

			engine := learn.NewEngine(d)
			stats, err := engine.GetTemplateStats()
			if err != nil {
				return err
			}

			if len(stats) == 0 {
				fmt.Println("  No template stats yet. Record some sessions first.")
				return nil
			}

			fmt.Println()
			bold := color.New(color.Bold)
			bold.Println("  Template Effectiveness:")
			fmt.Println()

			for _, s := range stats {
				bar := effectivenessBar(s.Effectiveness)
				fmt.Printf("  %-20s %s %.0f%%\n", cyan.Sprint(s.Name), bar, s.Effectiveness*100)
				fmt.Printf("    Usage: %d  Success: %d  Failure: %d  Stacks: %v\n",
					s.UsageCount, s.SuccessCount, s.FailureCount, s.Stacks)
				fmt.Println()
			}

			return nil
		},
	}

	return cmd
}

func newLearnAgentsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agents",
		Short: "Show agent performance",
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := openEngine()
			if err != nil {
				return err
			}
			defer d.Close()

			engine := learn.NewEngine(d)
			stats, err := engine.GetAgentStats()
			if err != nil {
				return err
			}

			if len(stats) == 0 {
				fmt.Println("  No agent stats yet. Record some sessions first.")
				return nil
			}

			fmt.Println()
			bold := color.New(color.Bold)
			bold.Println("  Agent Performance:")
			fmt.Println()

			for _, s := range stats {
				bar := effectivenessBar(s.Effectiveness)
				fmt.Printf("  %-20s %s %.0f%%\n", cyan.Sprint(s.Name), bar, s.Effectiveness*100)
				fmt.Printf("    Usage: %d  Success: %d  Failure: %d  Avg Duration: %ds\n",
					s.UsageCount, s.SuccessCount, s.FailureCount, s.AvgDuration)
				fmt.Println()
			}

			return nil
		},
	}

	return cmd
}

func newLearnKnowledgeCmd() *cobra.Command {
	var entryType string

	cmd := &cobra.Command{
		Use:   "knowledge",
		Short: "Show extracted knowledge",
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := openEngine()
			if err != nil {
				return err
			}
			defer d.Close()

			engine := learn.NewEngine(d)
			entries, err := engine.GetKnowledge(entryType)
			if err != nil {
				return err
			}

			if len(entries) == 0 {
				fmt.Println("  No knowledge extracted yet. Run: ocs learn run")
				return nil
			}

			fmt.Println()
			bold := color.New(color.Bold)
			bold.Println("  Extracted Knowledge:")
			fmt.Println()

			for _, e := range entries {
				icon := "📝"
				if e.Type == "warning" {
					icon = "⚠️"
				} else if e.Type == "tip" {
					icon = "💡"
				}
				fmt.Printf("  %s %-30s confidence: %.2f\n", icon, cyan.Sprint(e.Title), e.Confidence)
				fmt.Printf("    %s\n", e.Content)
				fmt.Printf("    Source: %s  Created: %s\n", e.Source, e.CreatedAt)
				fmt.Println()
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&entryType, "type", "", "Filter by type: lesson, pattern, tip, warning")

	return cmd
}

func newLearnRunCmd() *cobra.Command {
	var auto bool

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run full learning cycle",
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := openEngine()
			if err != nil {
				return err
			}
			defer d.Close()

			engine := learn.NewEngine(d)
			results, err := engine.RunLearningCycle()
			if err != nil {
				return err
			}

			fmt.Println()
			bold := color.New(color.Bold)
			bold.Println("  Learning Cycle Complete:")
			fmt.Println()
			fmt.Printf("  Patterns promoted:    %d\n", results["patterns_promoted"])
			fmt.Printf("  Heuristics demoted:   %d\n", results["heuristics_demoted"])
			fmt.Printf("  Knowledge extracted:  %d\n", results["knowledge_extracted"])
			fmt.Println()

			if auto {
				color.Green("Auto-apply improvements: enabled")
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&auto, "auto", false, "Auto-apply improvements")

	return cmd
}

func newLearnStatsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stats",
		Short: "Show overall learning statistics",
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := openEngine()
			if err != nil {
				return err
			}
			defer d.Close()

			engine := learn.NewEngine(d)
			stats, err := engine.GetSessionStats()
			if err != nil {
				return err
			}

			fmt.Println()
			bold := color.New(color.Bold)
			bold.Println("  Learning Statistics:")
			fmt.Println()
			fmt.Printf("  Total sessions:  %d\n", stats["total"])
			fmt.Printf("  Success:         %d\n", stats["success"])
			fmt.Printf("  Failure:         %d\n", stats["failure"])
			fmt.Printf("  Partial:         %d\n", stats["partial"])

			if stats["total"] > 0 {
				successRate := float64(stats["success"]) / float64(stats["total"]) * 100
				fmt.Printf("  Success rate:    %.0f%%\n", successRate)
			}

			patternCount, _ := d.Count(learn.NSPatterns)
			heuristicCount, _ := d.Count(learn.NSHeuristics)
			knowledgeCount, _ := d.Count(learn.NSKnowledge)

			fmt.Printf("  Patterns:        %d\n", patternCount)
			fmt.Printf("  Heuristics:      %d\n", heuristicCount)
			fmt.Printf("  Knowledge:       %d\n", knowledgeCount)
			fmt.Println()

			return nil
		},
	}

	return cmd
}

func effectivenessBar(rate float64) string {
	bars := int(rate * 10)
	filled := ""
	for i := 0; i < 10; i++ {
		if i < bars {
			filled += "█"
		} else {
			filled += "░"
		}
	}

	if rate >= 0.8 {
		return color.GreenString("[%s]", filled)
	} else if rate >= 0.5 {
		return color.YellowString("[%s]", filled)
	}
	return color.RedString("[%s]", filled)
}
