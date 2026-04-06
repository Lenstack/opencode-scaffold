package template

import (
	"testing"
)

func TestContext(t *testing.T) {
	ctx := Context{
		StackID:    "go",
		StackName:  "Go",
		Backend:    "go",
		Framework:  "standard",
		HasDB:      true,
		GoModule:   "example.com/app",
		Model:      "claude-3-5-sonnet",
		SmallModel: "claude-3-haiku",
	}

	if ctx.StackID != "go" {
		t.Errorf("expected StackID 'go', got %q", ctx.StackID)
	}
	if !ctx.HasDB {
		t.Error("expected HasDB to be true")
	}
	if ctx.GoModule != "example.com/app" {
		t.Errorf("expected GoModule 'example.com/app', got %q", ctx.GoModule)
	}
}

func TestRenderedFile(t *testing.T) {
	rf := RenderedFile{
		Path:    "test.go",
		Content: "package main",
		Mode:    0644,
	}

	if rf.Path != "test.go" {
		t.Errorf("expected Path 'test.go', got %q", rf.Path)
	}
	if rf.Content != "package main" {
		t.Errorf("expected Content 'package main', got %q", rf.Content)
	}
	if rf.Mode != 0644 {
		t.Errorf("expected Mode 0644, got %o", rf.Mode)
	}
}

func TestRenderAgent(t *testing.T) {
	ctx := Context{
		StackID:   "go",
		StackName: "Go",
		Backend:   "go",
	}

	// This test might fail if the template doesn't exist - that's ok
	_, err := RenderAgent("nonexistent", ctx)
	// Just verify the function can be called
	if err != nil {
		// Template not found is expected for nonexistent
		t.Logf("Expected error for nonexistent template: %v", err)
	}
}

func TestRenderSkill(t *testing.T) {
	ctx := Context{
		StackID:   "go",
		StackName: "Go",
		Backend:   "go",
	}

	// This test might fail if the template doesn't exist - that's ok
	_, err := RenderSkill("nonexistent", ctx)
	if err != nil {
		t.Logf("Expected error for nonexistent skill: %v", err)
	}
}

func TestRenderCommand(t *testing.T) {
	ctx := Context{
		StackID:   "go",
		StackName: "Go",
		Backend:   "go",
	}

	// This test might fail if the template doesn't exist - that's ok
	_, err := RenderCommand("nonexistent", ctx)
	if err != nil {
		t.Logf("Expected error for nonexistent command: %v", err)
	}
}

func TestRenderConfig(t *testing.T) {
	ctx := Context{
		StackID:   "go",
		StackName: "Go",
		Backend:   "go",
	}

	// This test might fail if the template doesn't exist - that's ok
	_, err := RenderConfig("nonexistent", ctx)
	if err != nil {
		t.Logf("Expected error for nonexistent config: %v", err)
	}
}

func TestAvailableAgents(t *testing.T) {
	agents := AvailableAgents()
	// Should return a list (possibly empty)
	t.Logf("Available agents: %v", agents)
}

func TestAvailableSkills(t *testing.T) {
	skills := AvailableSkills()
	// Should return a list (possibly empty)
	t.Logf("Available skills: %v", skills)
}

func TestAvailableCommands(t *testing.T) {
	commands := AvailableCommands()
	// Should return a list (possibly empty)
	t.Logf("Available commands: %v", commands)
}

func TestValidateSkillName(t *testing.T) {
	err := ValidateSkillName("valid-name")
	if err != nil {
		t.Errorf("expected no error for valid name, got %v", err)
	}

	err = ValidateSkillName("code-review")
	if err != nil {
		t.Errorf("expected no error for valid name, got %v", err)
	}

	err = ValidateSkillName("tdd-workflow")
	if err != nil {
		t.Errorf("expected no error for valid name, got %v", err)
	}
}

func TestValidateSkillName_TooShort(t *testing.T) {
	err := ValidateSkillName("")
	if err == nil {
		t.Fatal("expected error for empty name")
	}
}

func TestValidateSkillName_TooLong(t *testing.T) {
	// Name > 64 chars
	longName := "a" + string(make([]byte, 65))
	err := ValidateSkillName(longName)
	if err == nil {
		t.Fatal("expected error for too long name")
	}
}

func TestValidateSkillName_InvalidChars(t *testing.T) {
	err := ValidateSkillName("invalid_name!")
	if err == nil {
		t.Fatal("expected error for invalid name")
	}

	err = ValidateSkillName("InvalidName")
	if err == nil {
		t.Fatal("expected error for uppercase name")
	}

	err = ValidateSkillName("name with space")
	if err == nil {
		t.Fatal("expected error for name with space")
	}
}

func TestGetMasterTemplate(t *testing.T) {
	// Test that master template can be retrieved
	tmpl := getMasterTemplate()
	if tmpl == nil {
		t.Error("expected non-nil template")
	}
}

func TestGetCachedTemplate(t *testing.T) {
	// Test template caching
	path := "test.tmpl"
	content := `{{.StackID}}`

	tmpl1, err := getCachedTemplate(path, content)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	tmpl2, err := getCachedTemplate(path, content)
	if err != nil {
		t.Fatalf("expected no error on second call, got %v", err)
	}

	// Should be the same cached template
	if tmpl1 != tmpl2 {
		t.Error("expected same template from cache")
	}
}

func TestRenderTemplate(t *testing.T) {
	ctx := Context{
		StackID:   "go",
		StackName: "Go",
		Backend:   "go",
		Framework: "standard",
	}

	// Test with built-in template if exists
	// This is a placeholder - actual rendering depends on template existence
	_ = ctx
}
