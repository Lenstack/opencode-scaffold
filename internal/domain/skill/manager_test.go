package skill

import (
	"path/filepath"
	"testing"

	"github.com/Lenstack/opencode-scaffold/internal/hub"
)

func TestManager_Create(t *testing.T) {
	dir := t.TempDir()
	db, err := hub.NewEngine(filepath.Join(dir, ".opencode", "data"))
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer db.Close()

	m := NewManager(db)

	entry, err := m.Create("tdd-workflow")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if entry == nil {
		t.Fatal("expected entry to be returned")
	}
	if entry.Name != "tdd-workflow" {
		t.Errorf("expected Name 'tdd-workflow', got %q", entry.Name)
	}
	if entry.Effectiveness != 0.5 {
		t.Errorf("expected Effectiveness 0.5, got %f", entry.Effectiveness)
	}
}

func TestManager_CreateInvalidName(t *testing.T) {
	dir := t.TempDir()
	db, err := hub.NewEngine(filepath.Join(dir, ".opencode", "data"))
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer db.Close()

	m := NewManager(db)

	_, err = m.Create("")
	if err == nil {
		t.Fatal("expected error for empty name")
	}

	_, err = m.Create("invalid_name!")
	if err == nil {
		t.Fatal("expected error for invalid name")
	}
}

func TestManager_CreateDuplicate(t *testing.T) {
	dir := t.TempDir()
	db, err := hub.NewEngine(filepath.Join(dir, ".opencode", "data"))
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer db.Close()

	m := NewManager(db)

	_, err = m.Create("tdd-workflow")
	if err != nil {
		t.Fatalf("first create failed: %v", err)
	}

	_, err = m.Create("tdd-workflow")
	if err == nil {
		t.Fatal("expected error for duplicate")
	}
}

func TestManager_Get(t *testing.T) {
	dir := t.TempDir()
	db, err := hub.NewEngine(filepath.Join(dir, ".opencode", "data"))
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer db.Close()

	m := NewManager(db)

	m.Create("tdd-workflow")

	entry, err := m.Get("tdd-workflow")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if entry.Name != "tdd-workflow" {
		t.Errorf("expected Name 'tdd-workflow', got %q", entry.Name)
	}
}

func TestManager_GetNotFound(t *testing.T) {
	dir := t.TempDir()
	db, err := hub.NewEngine(filepath.Join(dir, ".opencode", "data"))
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer db.Close()

	m := NewManager(db)

	_, err = m.Get("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent skill")
	}
}

func TestManager_List(t *testing.T) {
	dir := t.TempDir()
	db, err := hub.NewEngine(filepath.Join(dir, ".opencode", "data"))
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer db.Close()

	m := NewManager(db)

	m.Create("tdd-workflow")
	m.Create("code-review")

	skills, err := m.List()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(skills) < 2 {
		t.Errorf("expected at least 2 skills, got %d", len(skills))
	}
}

func TestManager_TrackUsage(t *testing.T) {
	dir := t.TempDir()
	db, err := hub.NewEngine(filepath.Join(dir, ".opencode", "data"))
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer db.Close()

	m := NewManager(db)

	m.Create("tdd-workflow")

	err = m.TrackUsage("tdd-workflow", "session-1", "success")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	entry, _ := m.Get("tdd-workflow")
	if entry.UsageCount != 1 {
		t.Errorf("expected UsageCount 1, got %d", entry.UsageCount)
	}
	if entry.Effectiveness <= 0.5 {
		t.Errorf("expected Effectiveness > 0.5 after success, got %f", entry.Effectiveness)
	}
}

func TestManager_TrackUsageSuccess(t *testing.T) {
	dir := t.TempDir()
	db, err := hub.NewEngine(filepath.Join(dir, ".opencode", "data"))
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer db.Close()

	m := NewManager(db)

	_, err = m.Create("tdd-workflow")
	if err != nil {
		t.Fatalf("failed to create: %v", err)
	}

	err = m.TrackUsage("tdd-workflow", "session-1", "success")
	if err != nil {
		t.Fatalf("first track failed: %v", err)
	}
	err = m.TrackUsage("tdd-workflow", "session-2", "success")
	if err != nil {
		t.Fatalf("second track failed: %v", err)
	}

	entry, err := m.Get("tdd-workflow")
	if err != nil {
		t.Fatalf("failed to get: %v", err)
	}
	if entry.Effectiveness <= 0.5 {
		t.Errorf("expected Effectiveness > 0.5 after successes, got %f", entry.Effectiveness)
	}
}

func TestManager_TrackUsageFailure(t *testing.T) {
	dir := t.TempDir()
	db, err := hub.NewEngine(filepath.Join(dir, ".opencode", "data"))
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer db.Close()

	m := NewManager(db)

	m.Create("tdd-workflow")

	m.TrackUsage("tdd-workflow", "session-1", "failure")

	entry, _ := m.Get("tdd-workflow")
	if entry.Effectiveness >= 0.5 {
		t.Errorf("expected Effectiveness < 0.5 after failure, got %f", entry.Effectiveness)
	}
}

func TestManager_Optimize(t *testing.T) {
	dir := t.TempDir()
	db, err := hub.NewEngine(filepath.Join(dir, ".opencode", "data"))
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer db.Close()

	m := NewManager(db)

	m.Create("tdd-workflow")

	knowledge := hub.SkillKnowledge{
		Patterns:       []string{"pattern1"},
		AntiPatterns:   []string{"anti1"},
		ProjectContext: "test project",
	}

	err = m.Optimize("tdd-workflow", knowledge)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestManager_GetKnowledge(t *testing.T) {
	dir := t.TempDir()
	db, err := hub.NewEngine(filepath.Join(dir, ".opencode", "data"))
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer db.Close()

	m := NewManager(db)

	m.Create("tdd-workflow")

	knowledge, err := m.GetKnowledge("tdd-workflow")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if knowledge == nil {
		t.Fatal("expected knowledge to be returned")
	}
}

func TestManager_LogOptimization(t *testing.T) {
	dir := t.TempDir()
	db, err := hub.NewEngine(filepath.Join(dir, ".opencode", "data"))
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer db.Close()

	m := NewManager(db)

	err = m.LogOptimization("tdd-workflow", "improved pattern", "better results", 0.1)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestManager_Archive(t *testing.T) {
	dir := t.TempDir()
	db, err := hub.NewEngine(filepath.Join(dir, ".opencode", "data"))
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer db.Close()

	m := NewManager(db)

	m.Create("tdd-workflow")

	err = m.Archive("tdd-workflow")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	_, err = m.Get("tdd-workflow")
	if err == nil {
		t.Fatal("expected error after archive")
	}
}

func TestManager_Suggest(t *testing.T) {
	dir := t.TempDir()
	db, err := hub.NewEngine(filepath.Join(dir, ".opencode", "data"))
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer db.Close()

	m := NewManager(db)

	suggestions := m.Suggest("go")
	if len(suggestions) == 0 {
		t.Error("expected suggestions for go stack")
	}

	suggestions = m.Suggest("python")
	if len(suggestions) == 0 {
		t.Error("expected suggestions for python stack")
	}

	suggestions = m.Suggest("unknown")
	if len(suggestions) == 0 {
		t.Error("expected default suggestions for unknown stack")
	}
}

func TestValidateName(t *testing.T) {
	if err := ValidateName("valid-name"); err != nil {
		t.Errorf("expected no error for valid name, got %v", err)
	}
	if err := ValidateName("code-review"); err != nil {
		t.Errorf("expected no error for valid name, got %v", err)
	}
}

func TestValidateName_TooShort(t *testing.T) {
	err := ValidateName("")
	if err == nil {
		t.Fatal("expected error for empty name")
	}
}

func TestValidateName_TooLong(t *testing.T) {
	err := ValidateName("a" + string(make([]byte, 64)))
	if err == nil {
		t.Fatal("expected error for too long name")
	}
}

func TestValidateName_InvalidChars(t *testing.T) {
	err := ValidateName("invalid_name!")
	if err == nil {
		t.Fatal("expected error for invalid chars")
	}
}

func TestSkillEntry(t *testing.T) {
	entry := hub.SkillEntry{
		Name:          "tdd-workflow",
		CreatedAt:     "2024-01-01T00:00:00Z",
		UsageCount:    10,
		Effectiveness: 0.8,
	}

	if entry.Name != "tdd-workflow" {
		t.Errorf("expected Name 'tdd-workflow', got %q", entry.Name)
	}
	if entry.Effectiveness != 0.8 {
		t.Errorf("expected Effectiveness 0.8, got %f", entry.Effectiveness)
	}
}
