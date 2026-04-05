package skills

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type SkillEntry struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Owner       string `json:"owner"`
	Repo        string `json:"repo"`
	Description string `json:"description"`
	Installs    int    `json:"installs"`
	URL         string `json:"url"`
	RawURL      string `json:"raw_url"`
}

type Registry struct {
	Skills   []SkillEntry `json:"skills"`
	CachedAt string       `json:"cached_at"`
	Total    int          `json:"total"`
}

func FetchRegistry() (*Registry, error) {
	resp, err := http.Get("https://skills.sh/api/skills.json")
	if err != nil {
		return nil, fmt.Errorf("fetch skills registry: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("skills.sh returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var registry Registry
	if err := json.Unmarshal(body, &registry); err != nil {
		return nil, fmt.Errorf("parse skills registry: %w", err)
	}

	registry.CachedAt = time.Now().UTC().Format(time.RFC3339)
	return &registry, nil
}

func SearchSkills(query string) ([]SkillEntry, error) {
	registry, err := FetchRegistry()
	if err != nil {
		return nil, err
	}

	query = strings.ToLower(query)
	var results []SkillEntry
	for _, s := range registry.Skills {
		if strings.Contains(strings.ToLower(s.Name), query) ||
			strings.Contains(strings.ToLower(s.Description), query) ||
			strings.Contains(strings.ToLower(s.Owner), query) {
			results = append(results, s)
		}
	}

	return results, nil
}

func GetSkill(owner, repo, name string) (*SkillEntry, error) {
	registry, err := FetchRegistry()
	if err != nil {
		return nil, err
	}

	for _, s := range registry.Skills {
		if s.Owner == owner && s.Repo == repo && s.Name == name {
			return &s, nil
		}
	}

	return nil, fmt.Errorf("skill %s/%s/%s not found", owner, repo, name)
}

func ParseSkillRef(ref string) (owner, repo, name string, err error) {
	parts := strings.Split(ref, "/")
	switch len(parts) {
	case 2:
		owner = parts[0]
		repo = parts[1]
		name = parts[1]
	case 3:
		owner = parts[0]
		repo = parts[1]
		name = parts[2]
	default:
		return "", "", "", fmt.Errorf("invalid skill reference: %s (expected owner/repo or owner/repo/name)", ref)
	}
	return owner, repo, name, nil
}
