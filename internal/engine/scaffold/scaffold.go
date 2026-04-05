package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Lenstack/opencode-scaffold/internal/core"
	"github.com/Lenstack/opencode-scaffold/internal/detector"
	"github.com/Lenstack/opencode-scaffold/internal/hooks"
	tmpl "github.com/Lenstack/opencode-scaffold/internal/domain/template"
)

type Options struct {
	Root       string
	Stack      *detector.Stack
	Force      bool
	DryRun     bool
	Verbose    bool
	Model      string
	SmallModel string
	Agents     string
	Renderer   core.Renderer
	RunHooks   bool
	Template   string
	Empty      bool
}

type FileOp struct {
	Path    string
	Content string
	Mode    os.FileMode
	Action  string
}

type Plan struct {
	Stack *detector.Stack
	Files []FileOp
	Hooks []hooks.Hook
}

func MakePlan(opts Options) Plan {
	model := opts.Model
	if model == "" {
		model = "anthropic/claude-sonnet-4-20250514"
	}
	small := opts.SmallModel
	if small == "" {
		small = "anthropic/claude-haiku-4-20250514"
	}

	ctx := tmpl.Context{
		StackID:    opts.Stack.ID,
		StackName:  opts.Stack.Name,
		Backend:    opts.Stack.Backend,
		Framework:  opts.Stack.Framework,
		Frontend:   opts.Stack.Frontend,
		HasDB:      opts.Stack.HasDB,
		GoModule:   opts.Stack.GoModule,
		NodePkg:    opts.Stack.NodePkgName,
		Model:      model,
		SmallModel: small,
	}

	var plan Plan
	plan.Stack = opts.Stack

	if opts.Empty {
		return makeEmptyPlan(opts, ctx)
	}

	tpl, err := tmpl.GetTemplate(opts.Template)
	if err != nil {
		tpl = tmpl.Builtins()["standard"]
	}

	dirs := []string{
		".opencode/agents",
		".opencode/skills",
		".opencode/commands",
		".opencode/memory/episodic",
		".opencode/memory/semantic",
		".opencode/memory/heuristics",
		".opencode/memory/quarantine",
		".opencode/data",
		"docs/adr",
	}

	if tpl.IncludeDiscovery {
		dirs = append(dirs, ".opencode/plugins")
	}

	for _, d := range dirs {
		plan.Files = append(plan.Files, FileOp{Path: d + "/.gitkeep", Action: "create_dir"})
	}

	for _, skill := range tpl.Skills {
		dir := ".opencode/skills/" + skill
		plan.Files = append(plan.Files, FileOp{Path: dir + "/.gitkeep", Action: "create_dir"})
	}

	for _, name := range tpl.Agents {
		content, err := tmpl.RenderAgent(name, ctx)
		if err == nil {
			plan.Files = append(plan.Files, FileOp{
				Path:    ".opencode/agents/" + name + ".md",
				Content: content,
				Mode:    0644,
				Action:  fileAction(opts.Root, ".opencode/agents/"+name+".md", opts.Force),
			})
		}
	}

	for _, name := range tpl.Skills {
		content, err := tmpl.RenderSkill(name, ctx)
		if err == nil {
			plan.Files = append(plan.Files, FileOp{
				Path:    ".opencode/skills/" + name + "/SKILL.md",
				Content: content,
				Mode:    0644,
				Action:  fileAction(opts.Root, ".opencode/skills/"+name+"/SKILL.md", opts.Force),
			})
		}
	}

	for _, name := range tpl.Commands {
		content, err := tmpl.RenderCommand(name, ctx)
		if err == nil {
			plan.Files = append(plan.Files, FileOp{
				Path:    ".opencode/commands/" + name + ".md",
				Content: content,
				Mode:    0644,
				Action:  fileAction(opts.Root, ".opencode/commands/"+name+".md", opts.Force),
			})
		}
	}

	cfg := buildConfig(ctx, tpl.Agents)
	cfgContent, err := cfg.Render()
	if err == nil {
		plan.Files = append(plan.Files, FileOp{
			Path:    "opencode.json",
			Content: cfgContent,
			Mode:    0644,
			Action:  fileAction(opts.Root, "opencode.json", opts.Force),
		})
	}

	plan.Files = append(plan.Files, FileOp{
		Path: "AGENTS.md", Content: agentsMD(ctx, tpl), Mode: 0644,
		Action: fileAction(opts.Root, "AGENTS.md", opts.Force),
	})

	if tpl.IncludeDiscovery {
		plan.Files = append(plan.Files, FileOp{
			Path: ".opencode/plugins/env-protection.js", Content: envProtectionPlugin(), Mode: 0644,
			Action: fileAction(opts.Root, ".opencode/plugins/env-protection.js", opts.Force),
		})
	}

	if exists(opts.Root, ".git") {
		plan.Files = append(plan.Files, FileOp{
			Path: ".git/hooks/post-commit", Content: postCommitHook(), Mode: 0755,
			Action: fileAction(opts.Root, ".git/hooks/post-commit", opts.Force),
		})
	}

	if tpl.IncludeCI && !exists(opts.Root, ".github/workflows/opencode-ci.yml") {
		plan.Files = append(plan.Files, FileOp{
			Path: ".github/workflows/opencode-ci.yml", Content: githubActionsCI(ctx), Mode: 0644,
			Action: "create",
		})
	}

	if opts.RunHooks {
		plan.Hooks = hooks.DefaultHooks()
	}

	return plan
}

func makeEmptyPlan(opts Options, ctx tmpl.Context) Plan {
	var plan Plan
	plan.Stack = opts.Stack

	dirs := []string{
		".opencode/agents",
		".opencode/skills",
		".opencode/commands",
		".opencode/plugins",
		".opencode/memory/episodic",
		".opencode/memory/semantic",
		".opencode/memory/heuristics",
		".opencode/memory/quarantine",
		".opencode/data",
	}

	for _, d := range dirs {
		plan.Files = append(plan.Files, FileOp{Path: d + "/.gitkeep", Action: "create_dir"})
	}

	cfg := core.New(ctx.Model, ctx.SmallModel)
	cfgContent, err := cfg.Render()
	if err == nil {
		plan.Files = append(plan.Files, FileOp{
			Path:    "opencode.json",
			Content: cfgContent,
			Mode:    0644,
			Action:  fileAction(opts.Root, "opencode.json", opts.Force),
		})
	}

	plan.Files = append(plan.Files, FileOp{
		Path: "AGENTS.md", Content: emptyAgentsMD(ctx), Mode: 0644,
		Action: fileAction(opts.Root, "AGENTS.md", opts.Force),
	})

	return plan
}

func Apply(plan Plan, opts Options) Result {
	r := Result{}
	w := &Writer{Root: opts.Root, Force: opts.Force, DryRun: opts.DryRun, Renderer: opts.Renderer, Result: &r}

	for _, f := range plan.Files {
		if f.Action == "skip" {
			if opts.Renderer != nil {
				opts.Renderer.FileSkipped(f.Path, "already exists")
			}
			r.AddSkipped(f.Path)
			continue
		}
		if f.Action == "create_dir" {
			w.Dir(f.Path[:len(f.Path)-len("/.gitkeep")])
			continue
		}
		w.File(f.Path, f.Content, f.Mode)
	}

	if opts.RunHooks && !opts.DryRun {
		hookResults := hooks.RunHooks(opts.Root, plan.Hooks)
		for _, hr := range hookResults {
			if hr.Error != nil {
				if opts.Renderer != nil {
					opts.Renderer.Error(fmt.Errorf("hook %s: %w", hr.Name, hr.Error))
				}
				r.AddError(fmt.Errorf("hook %s: %w", hr.Name, hr.Error))
			}
		}
	}

	return r
}

func Run(opts Options) Result {
	plan := MakePlan(opts)
	return Apply(plan, opts)
}

func exists(root, rel string) bool {
	_, err := os.Stat(filepath.Join(root, rel))
	return err == nil
}

func fileAction(root, rel string, force bool) string {
	if !force && exists(root, rel) {
		return "skip"
	}
	if exists(root, rel) {
		return "overwrite"
	}
	return "create"
}

func agentSet(preset string) []string {
	switch preset {
	case "minimal":
		return []string{"orchestrator", "tester", "reviewer"}
	case "standard":
		return []string{"orchestrator", "planner", "architect", "tester", "reviewer", "security"}
	default:
		return []string{"orchestrator", "planner", "architect", "tester", "reviewer", "security", "reflector"}
	}
}

func buildConfig(ctx tmpl.Context, agents []string) *core.Config {
	cfg := core.New(ctx.Model, ctx.SmallModel)

	for _, name := range agents {
		agent := &core.AgentConfig{
			Description: name + " agent",
			Mode:        "subagent",
			Model:       ctx.Model,
		}
		if name == "orchestrator" {
			agent.Mode = "primary"
			agent.Steps = 40
			agent.Temperature = 0.1
			agent.Permission = map[string]any{
				"bash": map[string]string{
					"*":           "ask",
					"git status*": "allow",
					"git diff*":   "allow",
					"git log*":    "allow",
				},
			}
		}
		if name == "planner" {
			agent.Steps = 10
			agent.Temperature = 0.15
			agent.Permission = map[string]any{
				"edit": "deny",
				"bash": map[string]string{"*": "deny"},
			}
		}
		if name == "architect" {
			agent.Steps = 10
			agent.Temperature = 0.1
			agent.Permission = map[string]any{
				"bash": map[string]string{"*": "deny"},
			}
		}
		if name == "tester" {
			agent.Steps = 15
			agent.Temperature = 0.05
			agent.Permission = map[string]any{
				"bash": map[string]string{
					"*":             "ask",
					"go test*":      "allow",
					"npm test*":     "allow",
					"npm run test*": "allow",
					"pytest*":       "allow",
					"cargo test*":   "allow",
				},
			}
		}
		if name == "reviewer" {
			agent.Steps = 8
			agent.Temperature = 0.1
			agent.Permission = map[string]any{
				"edit": "deny",
				"bash": map[string]string{
					"*":         "deny",
					"git diff*": "allow",
					"grep*":     "allow",
				},
			}
		}
		if name == "security" {
			agent.Steps = 10
			agent.Temperature = 0.05
			agent.Permission = map[string]any{
				"edit": "deny",
				"bash": map[string]string{
					"*":         "ask",
					"grep*":     "allow",
					"git diff*": "allow",
				},
			}
		}
		if name == "reflector" {
			agent.Model = ctx.SmallModel
			agent.Steps = 12
			agent.Temperature = 0.3
			agent.Permission = map[string]any{
				"bash": map[string]string{"*": "deny"},
			}
		}
		cfg.AddAgent(name, agent)
	}

	cfg.SetDefaultAgent("orchestrator")
	return cfg
}

func agentsMD(ctx tmpl.Context, tpl tmpl.Template) string {
	now := time.Now().Format("2006-01-02")
	frontend := ctx.Frontend
	if frontend == "" {
		frontend = "N/A"
	}
	framework := ctx.Framework
	if framework == "" {
		framework = "N/A"
	}

	pipeline := tpl.Pipeline
	if pipeline == "" {
		pipeline = defaultPipeline(ctx)
	}

	return fmt.Sprintf(`# Project Production Rules
# Stack: %s | Framework: %s | Template: %s | Generated: %s
#
# This file is loaded by OpenCode via "instructions" in opencode.json.
# All agents read these rules. Edit freely — commit to Git.

## Stack Context
- Backend: %s
- Frontend: %s
- Framework: %s
- Database: %v

## Agent Pipeline (execute in order, no exceptions)

`+"```"+`
%s
`+"```"+`

## Non-Negotiable Rules (ALL agents must respect)

1. **TDD mandatory**: tester writes failing tests BEFORE any implementation
2. **No hardcoded secrets**: use env vars / config / secrets managers only
3. **No debug code in production**: no fmt.Println, console.log, debugger in non-test files
4. **All DB migrations additive**: never DROP COLUMN, RENAME COLUMN, or DROP TABLE
5. **All TODOs need ticket numbers**: TODO(#123) not bare TODO
6. **ADR for every design decision**: write to docs/adr/NNNN-slug.md
7. **Reflector runs after EVERY task**: memory must be updated
8. **Changed files only in reviews**: never re-read the full codebase in gates
9. **Skills loaded on-demand**: agents use the skill tool, not _index.md
10. **Self-heal max 2 retries**: then escalate to user with full error

## Definition of Done (10 items — orchestrator validates all via bash)

1. No console.log/fmt.Println in production files (grep verified)
2. No hardcoded secrets (grep verified)
3. All DB migrations are additive (grep verified)
4. All new public functions have tests (coverage verified)
5. All planner acceptance criteria addressed
6. ADR created or updated
7. CHANGELOG.md updated
8. No TODO without ticket number
9. Backend build passes
10. Frontend build passes (if applicable)

## Memory Protocol

Memory is stored in LevelDB at .opencode/data/ — managed by the ocs binary.

- **Tier 1 (Episodic)**: TTL 7 days — query via "ocs memory search --tier episodic"
- **Tier 2 (Semantic)**: TTL 90 days, confidence-scored — query via "ocs memory search --tier semantic"
- **Tier 3 (Heuristics)**: Permanent, promoted by @dreamer — query via "ocs memory list --tier heuristic"
- **Quarantine**: Facts with confidence < 0.60 after 14 days — auto-pruned

Reflector must:
- Run "ocs memory prune" to clean expired entries
- UPSERT semantic facts (new=0.50, confirmed=+0.25, contradicted=-0.20)
- Move facts with confidence < 0.60 to quarantine
- Check if dream_needed (candidates with session_count >= 3)

## Discovery

Project indexing is handled by the ocs binary:
- Run "ocs discover" for full reindex
- Run "ocs discover --incremental" for changed files only (uses checksum)
- Results stored in LevelDB at .opencode/data/
- No external dependencies (no Python3, no shell scripts)
`, ctx.StackName, framework, tpl.Name, now, ctx.Backend, frontend, framework, ctx.HasDB, pipeline)
}

func emptyAgentsMD(ctx tmpl.Context) string {
	now := time.Now().Format("2006-01-02")
	return fmt.Sprintf(`# Project Production Rules
# Stack: %s | Generated: %s
#
# This file is loaded by OpenCode via "instructions" in opencode.json.
# Edit this file to add project-specific rules for your AI agents.
#
# Add agents with: ocs add agent <name>
# Add skills with: ocs add skill <name>
# Add commands with: ocs add command <name>
#
# Run "ocs init" to generate a full production workflow.
`, ctx.StackName, now)
}

func defaultPipeline(ctx tmpl.Context) string {
	return `[User Request]
      |
      v
  orchestrator (primary agent)
      |
      +- Phase 0: @explore   -> run "ocs discover" to index project
      |                       load heuristics from "ocs memory list --tier heuristic"
      |
      +- Phase 1: @planner   -> acceptance criteria, edge cases, task breakdown
      |
      +- Phase 2: @architect -> ADR, data model, API contract
      |
      +- Phase 3: @tester    -> Phase 1: write FAILING tests (TDD contract)
      |
      +- Phase 4: implement  -> against failing tests
      |
      +- Phase 5: @tester    -> Phase 2: execute tests (must be GREEN)
      |           RED -> self-heal -> retry Phase 4 (max 2 attempts)
      |
      +- Phase 6: @reviewer + @security (parallel)
      |           Each scans CHANGED FILES ONLY
      |           FAIL -> unified fix -> retry once
      |
      +- Phase 7: cleaner (bash: gofmt/eslint/debug-removal)
      |
      +- Phase 8: Definition of Done (10 checks via bash)
      |
      +- Phase 9: @reflector -> update memory via "ocs memory" commands`
}

func postCommitHook() string {
	return `#!/bin/bash
# OpenCode: incremental discovery after commit
ocs discover 2>/dev/null || true
`
}

func envProtectionPlugin() string {
	return `// .opencode/plugins/env-protection.js
// Prevents OpenCode from reading .env files
export const EnvProtection = async () => {
  return {
    "tool.execute.before": async (input, output) => {
      if (input.tool === "read" && output.args.filePath.includes(".env")) {
        throw new Error("Do not read .env files")
      }
    },
  }
}
`
}

func githubActionsCI(ctx tmpl.Context) string {
	var jobs string

	switch ctx.Backend {
	case "go":
		jobs = `  backend:
    name: "Go Backend"
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: '1.22', cache: true }
      - run: go vet ./...
      - run: go test ./... -race -cover
      - run: go build ./...
`
	case "python":
		jobs = `  backend:
    name: "Python Backend"
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-python@v5
        with: { python-version: '3.12' }
      - run: pip install -e ".[dev]"
      - run: ruff check .
      - run: mypy .
      - run: pytest -v --cov
`
	case "rust":
		jobs = `  backend:
    name: "Rust Backend"
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: dtolnay/rust-toolchain@stable
      - run: cargo clippy -- -D warnings
      - run: cargo test
      - run: cargo build --release
`
	}

	if ctx.Frontend != "" {
		jobs += `  frontend:
    name: "Frontend"
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with: { node-version: '20', cache: 'npm' }
      - run: npm ci
      - run: npm run type-check
      - run: npm run test -- --coverage
      - run: npm run build
`
	}

	return fmt.Sprintf(`name: OpenCode CI
on:
  pull_request:
    branches: [main]
  push:
    branches: [main]

jobs:
%s
  discovery-check:
    name: "Discovery Freshness"
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Install ocs
        run: go install github.com/Lenstack/opencode-scaffold@latest
      - run: ocs discover --full
`, jobs)
}
