package hooks

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

type Hook struct {
	Name      string
	Cmd       string
	Args      []string
	Condition func(root string) bool
	Timeout   time.Duration
}

func DefaultHooks() []Hook {
	return []Hook{
		{
			Name:      "go mod tidy",
			Cmd:       "go",
			Args:      []string{"mod", "tidy"},
			Condition: func(root string) bool { return fileExists(root, "go.mod") },
			Timeout:   30 * time.Second,
		},
		{
			Name:      "git init",
			Cmd:       "git",
			Args:      []string{"init"},
			Condition: func(root string) bool { return !dirExists(root, ".git") },
			Timeout:   10 * time.Second,
		},
		{
			Name:      "npm install",
			Cmd:       "npm",
			Args:      []string{"install"},
			Condition: func(root string) bool { return fileExists(root, "package.json") },
			Timeout:   120 * time.Second,
		},
		{
			Name:      "pip install",
			Cmd:       "pip",
			Args:      []string{"install", "-e", ".[dev]"},
			Condition: func(root string) bool { return fileExists(root, "pyproject.toml") },
			Timeout:   60 * time.Second,
		},
		{
			Name:      "cargo check",
			Cmd:       "cargo",
			Args:      []string{"check"},
			Condition: func(root string) bool { return fileExists(root, "Cargo.toml") },
			Timeout:   60 * time.Second,
		},
	}
}

type HookResult struct {
	Name    string
	Success bool
	Output  string
	Error   error
}

func RunHooks(root string, hooks []Hook) []HookResult {
	var results []HookResult
	for _, h := range hooks {
		if h.Condition != nil && !h.Condition(root) {
			continue
		}

		ctx, cancel := context.WithTimeout(context.Background(), h.Timeout)
		defer cancel()

		cmd := exec.CommandContext(ctx, h.Cmd, h.Args...)
		cmd.Dir = root
		cmd.Env = append(os.Environ(), "CI=1")

		output, err := cmd.CombinedOutput()
		results = append(results, HookResult{
			Name:    h.Name,
			Success: err == nil,
			Output:  string(output),
			Error:   err,
		})
	}
	return results
}

func fileExists(root, name string) bool {
	_, err := os.Stat(filepath.Join(root, name))
	return err == nil
}

func dirExists(root, name string) bool {
	info, err := os.Stat(filepath.Join(root, name))
	return err == nil && info.IsDir()
}

func FormatResults(results []HookResult) string {
	var s string
	for _, r := range results {
		status := "OK"
		if r.Error != nil {
			status = fmt.Sprintf("FAIL: %v", r.Error)
		}
		s += fmt.Sprintf("  [%s] %s\n", status, r.Name)
	}
	return s
}
