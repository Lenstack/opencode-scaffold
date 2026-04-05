package template

import (
	"bytes"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
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

var (
	masterTemplate   *template.Template
	masterTemplateMu sync.Once
)

func getMasterTemplate() *template.Template {
	masterTemplateMu.Do(func() {
		masterTemplate = template.New("").Funcs(FuncMap())
		partials, err := builtinFS.ReadDir("builtins/_partials")
		if err != nil {
			return
		}
		for _, p := range partials {
			if p.IsDir() || !strings.HasSuffix(p.Name(), ".tmpl") {
				continue
			}
			pc, err := builtinFS.ReadFile("builtins/_partials/" + p.Name())
			if err != nil {
				continue
			}
			pname := strings.TrimSuffix(p.Name(), ".tmpl")
			masterTemplate.New(pname).Funcs(FuncMap()).Parse(string(pc))
		}
	})
	return masterTemplate
}

var (
	templateCache   = make(map[string]*template.Template)
	templateCacheMu sync.RWMutex
)

func getCachedTemplate(path string, content string) (*template.Template, error) {
	templateCacheMu.RLock()
	tmpl, ok := templateCache[path]
	templateCacheMu.RUnlock()
	if ok {
		return tmpl, nil
	}

	templateCacheMu.Lock()
	defer templateCacheMu.Unlock()

	if tmpl, ok := templateCache[path]; ok {
		return tmpl, nil
	}

	master := getMasterTemplate()
	cloned, err := master.Clone()
	if err != nil {
		return nil, fmt.Errorf("clone master template: %w", err)
	}

	tmpl, err = cloned.New(filepath.Base(path)).Funcs(FuncMap()).Parse(content)
	if err != nil {
		return nil, fmt.Errorf("parse template %s: %w", path, err)
	}

	templateCache[path] = tmpl
	return tmpl, nil
}

func renderTemplate(path string, ctx Context) (string, error) {
	content, err := builtinFS.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("template %s not found: %w", path, err)
	}

	tmpl, err := getCachedTemplate(path, string(content))
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, ctx); err != nil {
		return "", fmt.Errorf("render template %s: %w", path, err)
	}

	return buf.String(), nil
}
