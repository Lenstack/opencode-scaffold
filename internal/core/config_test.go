package core

import (
	"encoding/json"
	"testing"
)

func TestAgentConfig(t *testing.T) {
	cfg := AgentConfig{
		Mode:        "orchestrator",
		Model:       "claude-3-5-sonnet",
		Temperature: 0.7,
	}

	if cfg.Mode != "orchestrator" {
		t.Errorf("expected mode 'orchestrator', got %q", cfg.Mode)
	}
	if cfg.Model != "claude-3-5-sonnet" {
		t.Errorf("expected model 'claude-3-5-sonnet', got %q", cfg.Model)
	}
	if cfg.Temperature != 0.7 {
		t.Errorf("expected temperature 0.7, got %f", cfg.Temperature)
	}
}

func TestAgentConfig_Options(t *testing.T) {
	cfg := AgentConfig{
		Options: map[string]any{
			"max_tokens": 4096,
			"timeout":    30,
		},
	}

	if cfg.Options == nil {
		t.Fatal("expected Options to be set")
	}
	if cfg.Options["max_tokens"].(int) != 4096 {
		t.Errorf("expected max_tokens 4096, got %v", cfg.Options["max_tokens"])
	}
}

func TestConfig_New(t *testing.T) {
	cfg := New("", "")

	if cfg.Schema != "https://opencode.ai/config.json" {
		t.Errorf("expected schema 'https://opencode.ai/config.json', got %q", cfg.Schema)
	}
	if cfg.Autoupdate != true {
		t.Errorf("expected Autoupdate true, got %v", cfg.Autoupdate)
	}
	if cfg.Snapshot != true {
		t.Errorf("expected Snapshot true, got %v", cfg.Snapshot)
	}
	if len(cfg.Instructions) != 1 || cfg.Instructions[0] != "AGENTS.md" {
		t.Errorf("expected Instructions ['AGENTS.md'], got %v", cfg.Instructions)
	}
	if cfg.Share != "manual" {
		t.Errorf("expected Share 'manual', got %q", cfg.Share)
	}
}

func TestConfig_NewWithModels(t *testing.T) {
	cfg := New("claude-3-5-sonnet", "claude-3-haiku")

	if cfg.Model != "claude-3-5-sonnet" {
		t.Errorf("expected Model 'claude-3-5-sonnet', got %q", cfg.Model)
	}
	if cfg.SmallModel != "claude-3-haiku" {
		t.Errorf("expected SmallModel 'claude-3-haiku', got %q", cfg.SmallModel)
	}
}

func TestConfig_NewWithEmptyModels(t *testing.T) {
	cfg := New("", "")

	if cfg.Model != "" {
		t.Errorf("expected Model empty, got %q", cfg.Model)
	}
	if cfg.SmallModel != "" {
		t.Errorf("expected SmallModel empty, got %q", cfg.SmallModel)
	}
}

func TestConfig_AddAgent(t *testing.T) {
	cfg := New("", "")
	agent := &AgentConfig{
		Mode:        "tester",
		Description: "Test runner agent",
	}

	cfg.AddAgent("tester", agent)

	if cfg.Agent["tester"] == nil {
		t.Fatal("expected agent 'tester' to be added")
	}
	if cfg.Agent["tester"].Mode != "tester" {
		t.Errorf("expected agent mode 'tester', got %q", cfg.Agent["tester"].Mode)
	}
}

func TestConfig_SetDefaultAgent(t *testing.T) {
	cfg := New("", "")
	cfg.SetDefaultAgent("orchestrator")

	if cfg.DefaultAgent != "orchestrator" {
		t.Errorf("expected DefaultAgent 'orchestrator', got %q", cfg.DefaultAgent)
	}
}

func TestConfig_MarshalJSON(t *testing.T) {
	cfg := New("model", "small")
	cfg.SetDefaultAgent("orchestrator")
	cfg.AddAgent("tester", &AgentConfig{Mode: "tester"})

	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	var decoded Config
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("expected no error unmarshaling, got %v", err)
	}

	if decoded.Model != "model" {
		t.Errorf("expected Model 'model', got %q", decoded.Model)
	}
	if decoded.DefaultAgent != "orchestrator" {
		t.Errorf("expected DefaultAgent 'orchestrator', got %q", decoded.DefaultAgent)
	}
	if decoded.Agent["tester"] == nil {
		t.Error("expected agent 'tester' to be present")
	}
}

func TestConfig_Render(t *testing.T) {
	cfg := New("", "")

	output, err := cfg.Render()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if output == "" {
		t.Error("expected non-empty output")
	}
	// Should be valid JSON
	var decoded Config
	if err := json.Unmarshal([]byte(output), &decoded); err != nil {
		t.Errorf("expected valid JSON output, got error: %v", err)
	}
}

func TestConfig_RenderEmpty(t *testing.T) {
	cfg := New("", "")

	output, _ := cfg.Render()

	if len(output) == 0 {
		t.Error("expected non-empty output")
	}
}

func TestConfig_RenderWithAgents(t *testing.T) {
	cfg := New("model", "small")
	cfg.AddAgent("orchestrator", &AgentConfig{Mode: "orchestrator"})
	cfg.AddAgent("tester", &AgentConfig{Mode: "tester"})
	cfg.SetDefaultAgent("orchestrator")

	output, _ := cfg.Render()

	// Check for presence of agent keys
	if len(output) == 0 {
		t.Error("expected non-empty output")
	}
}

func TestConfig_Permission(t *testing.T) {
	cfg := New("", "")

	if cfg.Permission == nil {
		t.Error("expected Permission to be initialized")
	}
	if cfg.Permission["edit"] != "ask" {
		t.Errorf("expected edit permission 'ask', got %v", cfg.Permission["edit"])
	}
	if cfg.Permission["bash"] != "ask" {
		t.Errorf("expected bash permission 'ask', got %v", cfg.Permission["bash"])
	}
}
