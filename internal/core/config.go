package core

import "encoding/json"

type AgentConfig struct {
	Mode        string         `json:"mode"`
	Model       string         `json:"model,omitempty"`
	Description string         `json:"description"`
	Steps       int            `json:"steps,omitempty"`
	Temperature float64        `json:"temperature,omitempty"`
	Permission  map[string]any `json:"permission,omitempty"`
	Hidden      bool           `json:"hidden,omitempty"`
	Color       string         `json:"color,omitempty"`
	TopP        float64        `json:"top_p,omitempty"`
}

type Config struct {
	Schema       string                  `json:"$schema"`
	Model        string                  `json:"model,omitempty"`
	SmallModel   string                  `json:"small_model,omitempty"`
	Autoupdate   bool                    `json:"autoupdate"`
	Snapshot     bool                    `json:"snapshot"`
	Instructions []string                `json:"instructions"`
	Permission   map[string]any          `json:"permission,omitempty"`
	Compaction   map[string]any          `json:"compaction,omitempty"`
	Agent        map[string]*AgentConfig `json:"agent,omitempty"`
	DefaultAgent string                  `json:"default_agent,omitempty"`
}

func New(model, smallModel string) *Config {
	cfg := &Config{
		Schema:       "https://opencode.ai/config.json",
		Autoupdate:   true,
		Snapshot:     true,
		Instructions: []string{"AGENTS.md"},
		Permission: map[string]any{
			"edit":  "ask",
			"bash":  "ask",
			"skill": map[string]string{"*": "allow"},
		},
		Compaction: map[string]any{"enabled": true},
		Agent:      make(map[string]*AgentConfig),
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
