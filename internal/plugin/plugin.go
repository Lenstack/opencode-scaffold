package plugin

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	gotemplate "text/template"

	"github.com/Lenstack/opencode-scaffold/internal/domain/template"
)

var skillNameRe = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

type Entry struct {
	ID          string
	Name        string
	Description string
	Category    string
	Files       []template.RenderedFile
	DependsOn   []string
}

type Plugin interface {
	Name() string
	Entries() []Entry
	TemplateFuncs() gotemplate.FuncMap
}

func DiscoverPlugins(configDir string) ([]Plugin, error) {
	var plugins []Plugin

	pluginDirs := []string{
		filepath.Join(configDir, "plugins"),
	}

	home, _ := os.UserHomeDir()
	if home != "" {
		pluginDirs = append(pluginDirs, filepath.Join(home, ".config", "ocs", "plugins"))
	}

	for _, dir := range pluginDirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			name := strings.TrimSuffix(e.Name(), filepath.Ext(e.Name()))
			plugins = append(plugins, &filePlugin{
				name: name,
				path: filepath.Join(dir, e.Name()),
			})
		}
	}

	return plugins, nil
}

type filePlugin struct {
	name string
	path string
}

func (p *filePlugin) Name() string {
	return p.name
}

func (p *filePlugin) Entries() []Entry {
	return nil
}

func (p *filePlugin) TemplateFuncs() gotemplate.FuncMap {
	return nil
}

func ValidateSkillName(name string) bool {
	if len(name) < 1 || len(name) > 64 {
		return false
	}
	return skillNameRe.MatchString(name)
}
