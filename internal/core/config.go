package core

import "encoding/json"

type AgentConfig struct {
	Mode        string         `json:"mode,omitempty"`
	Model       string         `json:"model,omitempty"`
	Variant     string         `json:"variant,omitempty"`
	Description string         `json:"description,omitempty"`
	Steps       int            `json:"steps,omitempty"`
	Temperature float64        `json:"temperature,omitempty"`
	TopP        float64        `json:"top_p,omitempty"`
	Prompt      string         `json:"prompt,omitempty"`
	Disable     bool           `json:"disable,omitempty"`
	Hidden      bool           `json:"hidden,omitempty"`
	Color       string         `json:"color,omitempty"`
	Permission  map[string]any `json:"permission,omitempty"`
	Options     map[string]any `json:"options,omitempty"`
}

type Config struct {
	Schema       string                  `json:"$schema,omitempty"`
	LogLevel     string                  `json:"logLevel,omitempty"`
	Model        string                  `json:"model,omitempty"`
	SmallModel   string                  `json:"small_model,omitempty"`
	Autoupdate   any                     `json:"autoupdate,omitempty"`
	Snapshot     bool                    `json:"snapshot,omitempty"`
	Share        string                  `json:"share,omitempty"`
	Disabled     []string                `json:"disabled_providers,omitempty"`
	Enabled      []string                `json:"enabled_providers,omitempty"`
	Instructions []string                `json:"instructions,omitempty"`
	Permission   map[string]any          `json:"permission,omitempty"`
	Watcher      map[string]any          `json:"watcher,omitempty"`
	Skills       map[string]any          `json:"skills,omitempty"`
	Command      map[string]any          `json:"command,omitempty"`
	Server       map[string]any          `json:"server,omitempty"`
	Plugin       []any                   `json:"plugin,omitempty"`
	Agent        map[string]*AgentConfig `json:"agent,omitempty"`
	DefaultAgent string                  `json:"default_agent,omitempty"`
	Username     string                  `json:"username,omitempty"`
}

func New(model, smallModel string) *Config {
	cfg := &Config{
		Schema:       "https://opencode.ai/config.json",
		Autoupdate:   true,
		Snapshot:     true,
		Instructions: []string{"AGENTS.md"},
		Share:        "manual",
		Watcher:      map[string]any{"ignore": []string{}},
		Permission: map[string]any{
			"edit":  "ask",
			"bash":  "ask",
			"skill": map[string]string{"*": "allow"},
		},
		Agent: make(map[string]*AgentConfig),
	}
	if model != "" {
		cfg.Model = model
	}
	if smallModel != "" {
		cfg.SmallModel = smallModel
	}
	return cfg
}

func (c *Config) AddAgent(name string, agent *AgentConfig) {
	c.Agent[name] = agent
}

func (c *Config) SetDefaultAgent(name string) {
	c.DefaultAgent = name
}

func (c *Config) MarshalJSON() ([]byte, error) {
	type Alias Config
	return json.Marshal((*Alias)(c))
}

func (c *Config) Render() (string, error) {
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b) + "\n", nil
}
