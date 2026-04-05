package skills

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Lenstack/opencode-scaffold/internal/hub"
)

type Installer struct {
	db   *hub.Engine
	root string
}

func NewInstaller(db *hub.Engine, root string) *Installer {
	return &Installer{db: db, root: root}
}

func (i *Installer) InstallSkill(owner, repo, name string) error {
	skillDir := filepath.Join(i.root, ".opencode", "skills", name)
	skillFile := filepath.Join(skillDir, "SKILL.md")

	if _, err := os.Stat(skillFile); err == nil {
		return fmt.Errorf("skill %s already installed", name)
	}

	rawURL := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/main/skills/%s/SKILL.md", owner, repo, name)
	content, err := i.fetchSkillContent(rawURL)
	if err != nil {
		rawURL = fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/master/skills/%s/SKILL.md", owner, repo, name)
		content, err = i.fetchSkillContent(rawURL)
		if err != nil {
			return fmt.Errorf("fetch skill %s/%s/%s: %w", owner, repo, name, err)
		}
	}

	if err := os.MkdirAll(skillDir, 0755); err != nil {
		return fmt.Errorf("create skill directory: %w", err)
	}

	if err := os.WriteFile(skillFile, []byte(content), 0644); err != nil {
		return fmt.Errorf("write skill file: %w", err)
	}

	installed := map[string]any{
		"name":         name,
		"owner":        owner,
		"repo":         repo,
		"installed_at": time.Now().UTC().Format(time.RFC3339),
		"source":       "skills.sh",
		"raw_url":      rawURL,
		"status":       "installed",
	}

	if err := i.db.Put(hub.NSSkills, name+":installed", installed); err != nil {
		return fmt.Errorf("record skill installation: %w", err)
	}

	return nil
}

func (i *Installer) ListInstalledSkills() ([]map[string]string, error) {
	var skills []map[string]string

	i.db.Iterate(hub.NSSkills, func(key string, value []byte) error {
		if !strings.HasSuffix(key, ":installed") {
			return nil
		}

		var skill map[string]any
		if err := json.Unmarshal(value, &skill); err != nil {
			return nil
		}

		entry := map[string]string{}
		for k, v := range skill {
			if s, ok := v.(string); ok {
				entry[k] = s
			}
		}
		skills = append(skills, entry)
		return nil
	})

	return skills, nil
}

func (i *Installer) UninstallSkill(name string) error {
	skillFile := filepath.Join(i.root, ".opencode", "skills", name, "SKILL.md")
	if err := os.Remove(skillFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove skill file: %w", err)
	}

	if err := i.db.Delete(hub.NSSkills, name+":installed"); err != nil {
		return fmt.Errorf("remove skill record: %w", err)
	}

	return nil
}

func (i *Installer) fetchSkillContent(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}
