package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/Lenstack/opencode-scaffold/internal/core"
	"github.com/Lenstack/opencode-scaffold/internal/detector"
	tmpl "github.com/Lenstack/opencode-scaffold/internal/domain/template"
	"github.com/Lenstack/opencode-scaffold/internal/engine/scaffold"
)

func newInitCmd() *cobra.Command {
	var (
		force      bool
		dryRun     bool
		verbose    bool
		model      string
		smallModel string
		stackFlag  string
		agents     string
		outputFmt  string
		runHooks   bool
		template   string
		empty      bool
	)

	cmd := &cobra.Command{
		Use:   "init [directory]",
		Short: "Scaffold OpenCode workflow in a project",
		Long: `Scaffold an OpenCode production workflow in your project.

Automatically detects your stack and creates the appropriate agents,
skills, memory, and config following official OpenCode conventions.

Examples:
  ocs init                          # auto-detect template and scaffold
  ocs init --empty                  # minimal scaffold only (no agents/skills)
  ocs init --template api-backend   # use specific template
  ocs init --template fullstack     # full-stack development workflow
  ocs init --model anthropic/claude-sonnet-4-20250514
  ocs init --force                  # overwrite existing files
  ocs init --dry-run                # preview without writing
  ocs init --output json            # machine-readable output for agents

Available templates:
  standard        Default production workflow with full pipeline
  minimal         Quick iterations for solo developers
  solo-dev        Lightweight workflow for single developer
  team-production Strict quality gates for team workflows
  api-backend     Backend API development with security focus
  frontend-app    Frontend/UI development workflow
  fullstack       Full-stack development with parallel pipelines
  empty           Minimal scaffold structure only
`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRoot(args)
			if err != nil {
				return err
			}

			stack := detector.Detect(root)
			if stackFlag != "" {
				applyStackOverride(stack, stackFlag)
			}

			if template == "" && !empty {
				template = tmpl.DetectTemplate(stack, countFiles(root), hasCI(root))
			}

			if empty {
				template = "empty"
			}

			tpl, _ := tmpl.GetTemplate(template)

			renderer := core.NewRenderer(outputFmt)

			if outputFmt == "human" {
				printBannerInit(stack, model, tpl)
			}

			if !force && !dryRun && hasExistingConfig(root) {
				fmt.Printf("\n%s .opencode/ already exists in %s\n", yellow.Sprint("WARN"), root)
				fmt.Print("  Overwrite existing files? [y/N] ")
				var answer string
				fmt.Scanln(&answer)
				if !strings.EqualFold(strings.TrimSpace(answer), "y") {
					fmt.Println("  Use --force to skip this prompt.")
					return nil
				}
				force = true
			}

			if dryRun && outputFmt == "human" {
				fmt.Printf("\n%s Dry run — no files will be written\n\n", cyan.Sprint("DRY RUN"))
			}

			opts := scaffold.Options{
				Root:       root,
				Stack:      stack,
				Force:      force,
				DryRun:     dryRun,
				Model:      model,
				SmallModel: smallModel,
				Verbose:    verbose,
				Agents:     agents,
				Renderer:   renderer,
				RunHooks:   runHooks,
				Template:   template,
				Empty:      empty,
			}

			result := scaffold.Run(opts)

			if outputFmt == "human" {
				printResults(result, verbose, dryRun)
				if !dryRun {
					printNextStepsInit(root, stack, tpl)
				}
			} else {
				renderer.Summary(len(result.Created), len(result.Skipped), len(result.Errors))
			}

			if len(result.Errors) > 0 {
				return fmt.Errorf("scaffold completed with %d error(s)", len(result.Errors))
			}

			return nil
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "Overwrite existing files")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be created without writing")
	cmd.Flags().StringVar(&model, "model", "anthropic/claude-sonnet-4-20250514", "Primary LLM model")
	cmd.Flags().StringVar(&smallModel, "small-model", "anthropic/claude-haiku-4-20250514", "Fast model for lightweight subagents")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Show all created files")
	cmd.Flags().StringVar(&stackFlag, "stack", "", "Force stack detection (e.g. go-encore, python-fastapi, rust-axum)")
	cmd.Flags().StringVar(&agents, "agents", "standard", "Agent preset: minimal, standard, full")
	cmd.Flags().StringVar(&outputFmt, "output", "human", "Output format: human, json, ndjson")
	cmd.Flags().BoolVar(&runHooks, "hooks", false, "Run post-generation hooks (go mod tidy, git init, etc.)")
	cmd.Flags().StringVar(&template, "template", "", "Template to use (auto-detected if not set)")
	cmd.Flags().BoolVar(&empty, "empty", false, "Create minimal scaffold only (no agents, skills, commands)")

	return cmd
}

func resolveRoot(args []string) (string, error) {
	if len(args) == 0 {
		wd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("cannot get working directory: %w", err)
		}
		return wd, nil
	}
	abs, err := filepath.Abs(args[0])
	if err != nil {
		return "", fmt.Errorf("invalid path: %w", err)
	}
	if _, err := os.Stat(abs); os.IsNotExist(err) {
		return "", fmt.Errorf("directory does not exist: %s", abs)
	}
	return abs, nil
}

func hasExistingConfig(root string) bool {
	_, err := os.Stat(filepath.Join(root, ".opencode"))
	return err == nil
}

func applyStackOverride(stack *detector.Stack, flag string) {
	parts := strings.SplitN(flag, "-", 2)
	switch parts[0] {
	case "go":
		stack.Backend = "go"
		if len(parts) > 1 {
			stack.Framework = parts[1]
		}
	case "python":
		stack.Backend = "python"
		if len(parts) > 1 {
			stack.Framework = parts[1]
		}
	case "rust":
		stack.Backend = "rust"
		if len(parts) > 1 {
			stack.Framework = parts[1]
		}
	case "node", "nextjs":
		stack.Backend = "node"
		stack.Frontend = "nextjs"
		stack.Framework = "nextjs"
	}
	stack.ID = flag
	stack.Name = flag
}

func printBanner(stack *detector.Stack, model string) {
	fmt.Println()
	green.Println("  OpenCode Scaffold  -  Production AI Workflow")
	green.Println("  " + strings.Repeat("-", 50))
	fmt.Println()

	cyan.Println("  Detected Stack:")
	fmt.Printf("    Backend:   %s\n", color.CyanString(stack.Backend))
	if stack.Framework != "" {
		fmt.Printf("    Framework: %s\n", color.CyanString(stack.Framework))
	}
	if stack.Frontend != "" {
		fmt.Printf("    Frontend:  %s\n", color.CyanString(stack.Frontend))
	}
	if stack.HasDB {
		fmt.Printf("    Database:  %s\n", color.CyanString("yes"))
	}
	fmt.Printf("    Model:     %s\n", color.CyanString(model))
	fmt.Println()
}

func printResults(r scaffold.Result, verbose bool, dryRun bool) {
	action := "Created"
	if dryRun {
		action = "Would create"
	}

	fmt.Println()
	bold.Println("  Results:")
	fmt.Printf("    %s %d files %s\n", green.Sprint("OK"), len(r.Created), strings.ToLower(action))
	if len(r.Skipped) > 0 {
		fmt.Printf("    %s %d files skipped (already exist)\n", yellow.Sprint("WARN"), len(r.Skipped))
	}

	if verbose {
		for _, f := range r.Created {
			fmt.Printf("    %s %s\n", green.Sprint("+"), f)
		}
		for _, f := range r.Skipped {
			fmt.Printf("    %s %s\n", yellow.Sprint("~"), f)
		}
	}

	for _, e := range r.Errors {
		fmt.Printf("    %s %v\n", red.Sprint("ERR"), e)
	}

	fmt.Println()
}

func printNextSteps(root string, stack *detector.Stack) {
	bold.Println("  Next Steps:")
	fmt.Println()

	steps := []string{
		"1. Review generated files in .opencode/",
		"2. Configure your API key: opencode /connect",
		"3. Run opencode and use /init to analyze the project",
		"4. Start with: @orchestrator <your feature request>",
	}

	for _, s := range steps {
		fmt.Printf("    %s\n", s)
	}

	fmt.Println()
	cyan.Println("  Key Files Created:")
	files := []string{
		"opencode.json              — project config (model, agents, permissions)",
		"AGENTS.md                  — pipeline rules (read by all agents)",
		".opencode/agents/          — specialized agent definitions",
		".opencode/skills/          — reusable agent skills (loaded on-demand)",
		".opencode/commands/        — custom slash commands (/plan, /review, /ship)",
		".opencode/plugins/         — OpenCode plugins (env protection)",
		".opencode/data/            — LevelDB (discovery, memory, sessions, specs, skills)",
	}
	for _, f := range files {
		fmt.Printf("    - %s\n", f)
	}

	fmt.Println()
	yellow.Println("  Pro Tips:")
	fmt.Println("    - Use @planner first for new features (creates acceptance criteria)")
	fmt.Println("    - Use /plan command to see changes before building")
	fmt.Println("    - Use /reflect after each session to update memory")
	fmt.Println("    - Run ocs doctor to validate scaffold health")
	fmt.Println()

	if root != "." {
		fmt.Printf("  %s %s\n\n", color.GreenString("Project root:"), root)
	}
}

func printBannerInit(stack *detector.Stack, model string, tpl tmpl.Template) {
	fmt.Println()
	green.Println("  OpenCode Scaffold  -  Production AI Workflow")
	green.Println("  " + strings.Repeat("-", 50))
	fmt.Println()

	cyan.Println("  Detected Stack:")
	fmt.Printf("    Backend:   %s\n", color.CyanString(stack.Backend))
	if stack.Framework != "" {
		fmt.Printf("    Framework: %s\n", color.CyanString(stack.Framework))
	}
	if stack.Frontend != "" {
		fmt.Printf("    Frontend:  %s\n", color.CyanString(stack.Frontend))
	}
	if stack.HasDB {
		fmt.Printf("    Database:  %s\n", color.CyanString("yes"))
	}
	fmt.Printf("    Model:     %s\n", color.CyanString(model))
	fmt.Println()

	cyan.Println("  Template:")
	fmt.Printf("    Name:        %s\n", tpl.Name)
	fmt.Printf("    Description: %s\n", tpl.Description)
	fmt.Printf("    Agents:      %d\n", len(tpl.Agents))
	fmt.Printf("    Skills:      %d\n", len(tpl.Skills))
	fmt.Printf("    Commands:    %d\n", len(tpl.Commands))
	fmt.Println()
}

func printNextStepsInit(root string, stack *detector.Stack, tpl tmpl.Template) {
	bold.Println("  Next Steps:")
	fmt.Println()

	steps := []string{
		"1. Review generated files in .opencode/",
		"2. Configure your API key: opencode /connect",
		"3. Run opencode and use /init to analyze the project",
		"4. Start with: @orchestrator <your feature request>",
	}

	for _, s := range steps {
		fmt.Printf("    %s\n", s)
	}

	fmt.Println()
	cyan.Println("  Key Files Created:")
	files := []string{
		"opencode.json              — project config (model, agents, permissions)",
		"AGENTS.md                  — pipeline rules (read by all agents)",
	}
	if len(tpl.Agents) > 0 {
		files = append(files, ".opencode/agents/          — specialized agent definitions")
	}
	if len(tpl.Skills) > 0 {
		files = append(files, ".opencode/skills/          — reusable agent skills (loaded on-demand)")
	}
	if len(tpl.Commands) > 0 {
		files = append(files, ".opencode/commands/        — custom slash commands")
	}
	if tpl.IncludeDiscovery {
		files = append(files, ".opencode/plugins/         — OpenCode plugins (env protection)")
	}
	files = append(files, ".opencode/data/            — LevelDB (discovery, memory, sessions, specs, skills)")
	for _, f := range files {
		fmt.Printf("    - %s\n", f)
	}

	fmt.Println()
	yellow.Println("  Pro Tips:")
	fmt.Println("    - Use /ocs-init to re-analyze and regenerate workflow")
	fmt.Println("    - Use ocs template list to see all available templates")
	fmt.Println("    - Use ocs template detect to see what template fits your project")
	fmt.Println("    - Run ocs doctor to validate scaffold health")
	fmt.Println()

	if root != "." {
		fmt.Printf("  %s %s\n\n", color.GreenString("Project root:"), root)
	}
}

func countFiles(root string) int {
	count := 0
	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		ext := filepath.Ext(path)
		if ext == ".go" || ext == ".ts" || ext == ".tsx" || ext == ".py" || ext == ".rs" || ext == ".sql" || ext == ".js" || ext == ".jsx" {
			if !strings.Contains(path, "node_modules") && !strings.Contains(path, ".git") && !strings.Contains(path, ".opencode") {
				count++
			}
		}
		return nil
	})
	return count
}

func hasCI(root string) bool {
	_, err := os.Stat(filepath.Join(root, ".github", "workflows"))
	if err == nil {
		return true
	}
	_, err = os.Stat(filepath.Join(root, ".gitlab-ci.yml"))
	return err == nil
}
