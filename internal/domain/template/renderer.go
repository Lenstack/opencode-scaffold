package template

import (
	"bytes"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
)

//go:embed all:builtins
var builtinFS embed.FS

var skillNameRe = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

type Context struct {
	StackID    string
	StackName  string
	Backend    string
	Framework  string
	Frontend   string
	HasDB      bool
	GoModule   string
	NodePkg    string
	Model      string
	SmallModel string
}

type RenderedFile struct {
	Path    string
	Content string
	Mode    os.FileMode
}

func RenderAgent(name string, ctx Context) (string, error) {
	return renderTemplate("builtins/agents/"+name+".md.tmpl", ctx)
}

func RenderSkill(name string, ctx Context) (string, error) {
	return renderTemplate("builtins/skills/"+name+"/SKILL.md.tmpl", ctx)
}

func RenderCommand(name string, ctx Context) (string, error) {
	return renderTemplate("builtins/commands/"+name+".md.tmpl", ctx)
}

func RenderConfig(name string, ctx Context) (string, error) {
	return renderTemplate("builtins/config/"+name+".tmpl", ctx)
}

func AvailableAgents() []string {
	entries, err := builtinFS.ReadDir("builtins/agents")
	if err != nil {
		return nil
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() {
			names = append(names, e.Name()[:len(e.Name())-len(".md.tmpl")])
		}
	}
	return names
}

func AvailableSkills() []string {
	entries, err := builtinFS.ReadDir("builtins/skills")
	if err != nil {
		return nil
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	return names
}

func AvailableCommands() []string {
	entries, err := builtinFS.ReadDir("builtins/commands")
	if err != nil {
		return nil
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() {
			names = append(names, e.Name()[:len(e.Name())-len(".md.tmpl")])
		}
	}
	return names
}

func ValidateSkillName(name string) error {
	if len(name) < 1 || len(name) > 64 {
		return fmt.Errorf("skill name must be 1-64 characters (got %d)", len(name))
	}
	if !skillNameRe.MatchString(name) {
		return fmt.Errorf("skill name must match ^[a-z0-9]+(-[a-z0-9]+)*$ (got: %s)", name)
	}
	return nil
}

func renderTemplate(path string, ctx Context) (string, error) {
	content, err := builtinFS.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("template %s not found: %w", path, err)
	}

	tmpl, err := template.New(filepath.Base(path)).Funcs(FuncMap()).Parse(string(content))
	if err != nil {
		return "", fmt.Errorf("parse template %s: %w", path, err)
	}

	partials, err := builtinFS.ReadDir("builtins/_partials")
	if err == nil {
		for _, p := range partials {
			if p.IsDir() || !strings.HasSuffix(p.Name(), ".tmpl") {
				continue
			}
			pc, err := builtinFS.ReadFile("builtins/_partials/" + p.Name())
			if err != nil {
				continue
			}
			pname := strings.TrimSuffix(p.Name(), ".tmpl")
			_, err = tmpl.New(pname).Funcs(FuncMap()).Parse(string(pc))
			if err != nil {
				return "", fmt.Errorf("parse partial %s: %w", p.Name(), err)
			}
		}
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, ctx); err != nil {
		return "", fmt.Errorf("render template %s: %w", path, err)
	}

	return buf.String(), nil
}
