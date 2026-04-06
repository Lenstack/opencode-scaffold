package template

import (
	"testing"
)

func TestTemplate_Builtins(t *testing.T) {
	builtins := Builtins()

	if len(builtins) == 0 {
		t.Error("expected built-in templates")
	}

	if _, ok := builtins["standard"]; !ok {
		t.Error("expected 'standard' template")
	}
	if _, ok := builtins["minimal"]; !ok {
		t.Error("expected 'minimal' template")
	}
}

func TestTemplate_BuiltinsFields(t *testing.T) {
	builtins := Builtins()

	std := builtins["standard"]
	if std.ID != "standard" {
		t.Errorf("expected ID 'standard', got %q", std.ID)
	}
	if std.Name == "" {
		t.Error("expected Name to be set")
	}
	if len(std.Agents) == 0 {
		t.Error("expected Agents to be set")
	}
	if len(std.Skills) == 0 {
		t.Error("expected Skills to be set")
	}
}

func TestTemplate_UserTemplateDir(t *testing.T) {
	dir := UserTemplateDir()

	if dir == "" {
		t.Error("expected non-empty directory")
	}
	// Should contain user home directory
	if dir == ".config/ocs/templates" {
		t.Error("expected absolute path")
	}
}

func TestTemplate_LoadUserTemplates(t *testing.T) {
	// Test with non-existent directory
	templates := LoadUserTemplates()
	// Should return empty map without error
	if templates == nil {
		t.Error("expected nil map or empty map")
	}
}

func TestTemplate_AllTemplates(t *testing.T) {
	templates := AllTemplates()

	if len(templates) == 0 {
		t.Error("expected templates")
	}

	// Should include builtins
	if _, ok := templates["standard"]; !ok {
		t.Error("expected 'standard' in AllTemplates")
	}
	if _, ok := templates["minimal"]; !ok {
		t.Error("expected 'minimal' in AllTemplates")
	}
}

func TestTemplate_GetTemplate(t *testing.T) {
	tmpl, err := GetTemplate("standard")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if tmpl.ID != "standard" {
		t.Errorf("expected ID 'standard', got %q", tmpl.ID)
	}
}

func TestTemplate_GetTemplateNotFound(t *testing.T) {
	_, err := GetTemplate("nonexistent-template")
	if err == nil {
		t.Fatal("expected error for nonexistent template")
	}
}

func TestTemplate_GetTemplateMinimal(t *testing.T) {
	tmpl, err := GetTemplate("minimal")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(tmpl.Agents) == 0 {
		t.Error("expected Agents to be set")
	}
}

func TestTemplate_AddTemplate(t *testing.T) {
	// Create a temporary user template directory
	// This test verifies the function works (we won't actually create due to path issues)
	// Test that we can't add template with empty ID
	err := AddTemplate(Template{})
	if err == nil {
		t.Fatal("expected error for template without ID")
	}
}

func TestTemplate_AddTemplateNoID(t *testing.T) {
	tmpl := Template{
		Name:        "Test Template",
		Description: "Test description",
	}

	err := AddTemplate(tmpl)
	if err == nil {
		t.Fatal("expected error for template without ID")
	}
}

func TestTemplate_AddTemplateBuiltin(t *testing.T) {
	// Try to overwrite a built-in template
	tmpl := Template{
		ID:   "standard",
		Name: "Modified Standard",
	}

	err := AddTemplate(tmpl)
	if err == nil {
		t.Fatal("expected error when overwriting builtin")
	}
}

func TestTemplate_DestroyTemplate(t *testing.T) {
	// Can't destroy builtins
	err := DestroyTemplate("standard")
	if err == nil {
		t.Fatal("expected error when destroying builtin")
	}
}

func TestTemplate_DestroyTemplateBuiltin(t *testing.T) {
	err := DestroyTemplate("minimal")
	if err == nil {
		t.Fatal("expected error when destroying builtin")
	}
}

func TestTemplate_ExportTemplate(t *testing.T) {
	yaml, err := ExportTemplate("standard")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if yaml == "" {
		t.Error("expected non-empty YAML")
	}
}

func TestTemplate_ImportTemplate(t *testing.T) {
	yaml := `
id: test-template
name: Test Template
description: A test template
agents:
  - orchestrator
  - tester
skills:
  - tdd-workflow
commands:
  - ocs-plan
`

	err := ImportTemplate(yaml)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify we can get it
	tmpl, err := GetTemplate("test-template")
	if err != nil {
		t.Fatalf("expected template to exist: %v", err)
	}
	if tmpl.Name != "Test Template" {
		t.Errorf("expected Name 'Test Template', got %q", tmpl.Name)
	}

	// Cleanup
	DestroyTemplate("test-template")
}

func TestTemplate_ImportTemplateNoID(t *testing.T) {
	yaml := `
name: Test Template
description: A test template
`

	err := ImportTemplate(yaml)
	if err == nil {
		t.Fatal("expected error for template without ID")
	}
}

func TestTemplate_InvalidateTemplates(t *testing.T) {
	// Call invalidate
	InvalidateTemplates()

	// Get templates again - should work
	templates := AllTemplates()
	if len(templates) == 0 {
		t.Error("expected templates after invalidation")
	}
}

func TestHasExt(t *testing.T) {
	tests := []struct {
		name     string
		exts     []string
		expected bool
	}{
		{"test.yaml", []string{".yaml", ".yml"}, true},
		{"test.yml", []string{".yaml", ".yml"}, true},
		{"test.txt", []string{".yaml", ".yml"}, false},
		{"test", []string{".yaml"}, false},
	}

	for _, tt := range tests {
		result := hasExt(tt.name, tt.exts...)
		if result != tt.expected {
			t.Errorf("hasExt(%q, %v) = %v, want %v", tt.name, tt.exts, result, tt.expected)
		}
	}
}

func TestTmplName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"test.yaml", "test"},
		{"test.yml", "test"},
		{"template.yaml", "template"},
	}

	for _, tt := range tests {
		result := tmplName(tt.input)
		if result != tt.expected {
			t.Errorf("tmplName(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestTemplate_SoloDev(t *testing.T) {
	tmpl := Builtins()["solo-dev"]
	if tmpl.ID != "solo-dev" {
		t.Errorf("expected ID 'solo-dev', got %q", tmpl.ID)
	}
	if len(tmpl.Agents) != 2 {
		t.Errorf("expected 2 agents, got %d", len(tmpl.Agents))
	}
}

func TestTemplate_TeamProduction(t *testing.T) {
	tmpl := Builtins()["team-production"]
	if tmpl.ID != "team-production" {
		t.Errorf("expected ID 'team-production', got %q", tmpl.ID)
	}
	if !tmpl.IncludeCI {
		t.Error("expected IncludeCI to be true")
	}
}

func TestTemplate_APIBackend(t *testing.T) {
	tmpl := Builtins()["api-backend"]
	if tmpl.ID != "api-backend" {
		t.Errorf("expected ID 'api-backend', got %q", tmpl.ID)
	}
	if len(tmpl.Skills) == 0 {
		t.Error("expected skills to be set")
	}
}

func TestTemplate_FrontendApp(t *testing.T) {
	tmpl := Builtins()["frontend-app"]
	if tmpl.ID != "frontend-app" {
		t.Errorf("expected ID 'frontend-app', got %q", tmpl.ID)
	}
}

func TestTemplate_Fullstack(t *testing.T) {
	tmpl := Builtins()["fullstack"]
	if tmpl.ID != "fullstack" {
		t.Errorf("expected ID 'fullstack', got %q", tmpl.ID)
	}
	if !tmpl.IncludeDiscovery {
		t.Error("expected IncludeDiscovery to be true")
	}
}

func TestTemplate_Empty(t *testing.T) {
	tmpl := Builtins()["empty"]
	if tmpl.ID != "empty" {
		t.Errorf("expected ID 'empty', got %q", tmpl.ID)
	}
	if len(tmpl.Agents) != 0 {
		t.Errorf("expected 0 agents, got %d", len(tmpl.Agents))
	}
	if tmpl.IncludeCI {
		t.Error("expected IncludeCI to be false")
	}
}
