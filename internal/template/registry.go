package template

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Template struct {
	ID               string   `yaml:"id"`
	Name             string   `yaml:"name"`
	Description      string   `yaml:"description"`
	Agents           []string `yaml:"agents"`
	Skills           []string `yaml:"skills"`
	Commands         []string `yaml:"commands"`
	Pipeline         string   `yaml:"pipeline"`
	IncludeCI        bool     `yaml:"include_ci"`
	IncludeDiscovery bool     `yaml:"include_discovery"`
}

func Builtins() map[string]Template {
	return map[string]Template{
		"standard": {
			ID:               "standard",
			Name:             "Standard Production",
			Description:      "Default production workflow with full pipeline",
			Agents:           []string{"orchestrator", "planner", "architect", "tester", "reviewer", "security", "reflector"},
			Skills:           []string{"tdd-workflow", "code-review", "security-audit", "git-workflow", "api-design", "observability", "refactor", "performance"},
			Commands:         []string{"ocs-plan", "ocs-review", "ocs-ship", "ocs-reflect", "ocs-discover", "ocs-init"},
			IncludeCI:        true,
			IncludeDiscovery: true,
		},
		"empty": {
			ID:               "empty",
			Name:             "Empty",
			Description:      "Minimal scaffold structure only",
			Agents:           []string{},
			Skills:           []string{},
			Commands:         []string{},
			IncludeCI:        false,
			IncludeDiscovery: false,
		},
	}
}

func UserTemplateDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "ocs", "templates")
}

func LoadUserTemplates() map[string]Template {
	templates := map[string]Template{}
	dir := UserTemplateDir()

	entries, err := os.ReadDir(dir)
	if err != nil {
		return templates
	}

	for _, e := range entries {
		if e.IsDir() || !hasExt(e.Name(), ".yaml", ".yml") {
			continue
		}

		content, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}

		var tmpl Template
		if err := yaml.Unmarshal(content, &tmpl); err != nil {
			continue
		}

		if tmpl.ID == "" {
			tmpl.ID = tmplName(e.Name())
		}
		templates[tmpl.ID] = tmpl
	}

	return templates
}

func AllTemplates() map[string]Template {
	all := Builtins()
	for k, v := range LoadUserTemplates() {
		all[k] = v
	}
	return all
}

func GetTemplate(id string) (Template, error) {
	all := AllTemplates()
	tmpl, ok := all[id]
	if !ok {
		return Template{}, fmt.Errorf("template %q not found", id)
	}
	return tmpl, nil
}

func AddTemplate(tmpl Template) error {
	if tmpl.ID == "" {
		return fmt.Errorf("template ID is required")
	}

	// Don't allow overwriting built-in templates
	if _, ok := Builtins()[tmpl.ID]; ok {
		return fmt.Errorf("cannot overwrite built-in template %q", tmpl.ID)
	}

	dir := UserTemplateDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create template dir: %w", err)
	}

	data, err := yaml.Marshal(&tmpl)
	if err != nil {
		return fmt.Errorf("marshal template: %w", err)
	}

	path := filepath.Join(dir, tmpl.ID+".yaml")
	return os.WriteFile(path, data, 0644)
}

func DestroyTemplate(id string) error {
	if _, ok := Builtins()[id]; ok {
		return fmt.Errorf("cannot delete built-in template %q", id)
	}

	path := filepath.Join(UserTemplateDir(), id+".yaml")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		path = filepath.Join(UserTemplateDir(), id+".yml")
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("template %q not found", id)
	}

	return os.Remove(path)
}

func ExportTemplate(id string) (string, error) {
	tmpl, err := GetTemplate(id)
	if err != nil {
		return "", err
	}

	data, err := yaml.Marshal(&tmpl)
	if err != nil {
		return "", fmt.Errorf("marshal template: %w", err)
	}

	return string(data), nil
}

func ImportTemplate(yamlContent string) error {
	var tmpl Template
	if err := yaml.Unmarshal([]byte(yamlContent), &tmpl); err != nil {
		return fmt.Errorf("parse template YAML: %w", err)
	}

	if tmpl.ID == "" {
		return fmt.Errorf("template ID is required in YAML")
	}

	return AddTemplate(tmpl)
}

func hasExt(name string, exts ...string) bool {
	for _, ext := range exts {
		if len(name) >= len(ext) && name[len(name)-len(ext):] == ext {
			return true
		}
	}
	return false
}

func tmplName(filename string) string {
	for _, ext := range []string{".yaml", ".yml"} {
		if len(filename) > len(ext) && filename[len(filename)-len(ext):] == ext {
			return filename[:len(filename)-len(ext)]
		}
	}
	return filename
}
