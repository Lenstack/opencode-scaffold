package session

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

	entry, err := m.Create("session-1", "Test Session", "orchestrator", "claude-3-5-sonnet")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if entry == nil {
		t.Fatal("expected entry to be returned")
	}
	if entry.ID != "session-1" {
		t.Errorf("expected ID 'session-1', got %q", entry.ID)
	}
	if entry.Title != "Test Session" {
		t.Errorf("expected Title 'Test Session', got %q", entry.Title)
	}
	if entry.Agent != "orchestrator" {
		t.Errorf("expected Agent 'orchestrator', got %q", entry.Agent)
	}
	if entry.Status != "active" {
		t.Errorf("expected Status 'active', got %q", entry.Status)
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

	_, err = m.Create("session-1", "Test Session", "orchestrator", "claude")
	if err != nil {
		t.Fatalf("failed to create: %v", err)
	}

	entry, err := m.Get("session-1")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if entry.ID != "session-1" {
		t.Errorf("expected ID 'session-1', got %q", entry.ID)
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
		t.Fatal("expected error for nonexistent session")
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

	m.Create("session-1", "Session 1", "orchestrator", "claude")
	m.Create("session-2", "Session 2", "tester", "claude")

	sessions, err := m.List()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(sessions) < 2 {
		t.Errorf("expected at least 2 sessions, got %d", len(sessions))
	}
}

func TestManager_UpdateStatus(t *testing.T) {
	dir := t.TempDir()
	db, err := hub.NewEngine(filepath.Join(dir, ".opencode", "data"))
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer db.Close()

	m := NewManager(db)

	m.Create("session-1", "Test", "orchestrator", "claude")

	err = m.UpdateStatus("session-1", "completed")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	entry, _ := m.Get("session-1")
	if entry.Status != "completed" {
		t.Errorf("expected Status 'completed', got %q", entry.Status)
	}
}

func TestManager_AddContext(t *testing.T) {
	dir := t.TempDir()
	db, err := hub.NewEngine(filepath.Join(dir, ".opencode", "data"))
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer db.Close()

	m := NewManager(db)

	m.Create("session-1", "Test", "orchestrator", "claude")

	err = m.AddContext("session-1", []string{"file.go"}, []string{"output.go"}, []string{"decision 1"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	ctx, _ := m.GetContext("session-1")
	if len(ctx.FilesRead) != 1 || ctx.FilesRead[0] != "file.go" {
		t.Errorf("expected FilesRead ['file.go'], got %v", ctx.FilesRead)
	}
	if len(ctx.FilesWritten) != 1 || ctx.FilesWritten[0] != "output.go" {
		t.Errorf("expected FilesWritten ['output.go'], got %v", ctx.FilesWritten)
	}
}

func TestManager_GetContext(t *testing.T) {
	dir := t.TempDir()
	db, err := hub.NewEngine(filepath.Join(dir, ".opencode", "data"))
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer db.Close()

	m := NewManager(db)

	m.Create("session-1", "Test", "orchestrator", "claude")

	ctx, err := m.GetContext("session-1")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Context should exist (even if empty)
	if ctx == nil {
		t.Fatal("expected context to be returned")
	}
}

func TestManager_SetSummary(t *testing.T) {
	dir := t.TempDir()
	db, err := hub.NewEngine(filepath.Join(dir, ".opencode", "data"))
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer db.Close()

	m := NewManager(db)

	m.Create("session-1", "Test", "orchestrator", "claude")

	summary := hub.SessionSummary{
		Summary:    "Completed successfully",
		Duration:   120,
		TokensUsed: 5000,
		Outcome:    "success",
	}

	err = m.SetSummary("session-1", summary)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestManager_GetSummary(t *testing.T) {
	dir := t.TempDir()
	db, err := hub.NewEngine(filepath.Join(dir, ".opencode", "data"))
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer db.Close()

	m := NewManager(db)

	m.Create("session-1", "Test", "orchestrator", "claude")

	summary := hub.SessionSummary{
		Summary:  "Completed",
		Duration: 120,
		Outcome:  "success",
	}
	m.SetSummary("session-1", summary)

	retrieved, err := m.GetSummary("session-1")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if retrieved.Summary != "Completed" {
		t.Errorf("expected Summary 'Completed', got %q", retrieved.Summary)
	}
}

func TestManager_Delete(t *testing.T) {
	dir := t.TempDir()
	db, err := hub.NewEngine(filepath.Join(dir, ".opencode", "data"))
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer db.Close()

	m := NewManager(db)

	m.Create("session-1", "Test", "orchestrator", "claude")

	err = m.Delete("session-1")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	_, err = m.Get("session-1")
	if err == nil {
		t.Fatal("expected error after delete")
	}
}

func TestManager_GetCurrent(t *testing.T) {
	dir := t.TempDir()
	db, err := hub.NewEngine(filepath.Join(dir, ".opencode", "data"))
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer db.Close()

	m := NewManager(db)

	m.Create("session-1", "Test", "orchestrator", "claude")
	m.UpdateStatus("session-1", "completed")
	m.Create("session-2", "Test2", "orchestrator", "claude")

	current, err := m.GetCurrent()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if current.ID != "session-2" {
		t.Errorf("expected current session 'session-2', got %q", current.ID)
	}
}

func TestManager_GetCurrentEmpty(t *testing.T) {
	dir := t.TempDir()
	db, err := hub.NewEngine(filepath.Join(dir, ".opencode", "data"))
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer db.Close()

	m := NewManager(db)

	_, err = m.GetCurrent()
	if err == nil {
		t.Fatal("expected error when no sessions")
	}
}

func TestSessionEntry(t *testing.T) {
	entry := hub.SessionEntry{
		ID:        "test-id",
		Title:     "Test Session",
		Agent:     "orchestrator",
		Model:     "claude-3-5-sonnet",
		StartedAt: "2024-01-01T00:00:00Z",
		Status:    "active",
	}

	if entry.ID != "test-id" {
		t.Errorf("expected ID 'test-id', got %q", entry.ID)
	}
	if entry.Status != "active" {
		t.Errorf("expected Status 'active', got %q", entry.Status)
	}
}

func TestSessionContext(t *testing.T) {
	ctx := hub.SessionContext{
		FilesRead:    []string{"file1.go", "file2.go"},
		FilesWritten: []string{"output.go"},
		Decisions:    []string{"Decision 1"},
	}

	if len(ctx.FilesRead) != 2 {
		t.Errorf("expected 2 files read, got %d", len(ctx.FilesRead))
	}
	if len(ctx.FilesWritten) != 1 {
		t.Errorf("expected 1 file written, got %d", len(ctx.FilesWritten))
	}
}

func TestSessionSummary(t *testing.T) {
	summary := hub.SessionSummary{
		Summary:    "Completed successfully",
		Duration:   300,
		TokensUsed: 10000,
		Outcome:    "success",
	}

	if summary.Outcome != "success" {
		t.Errorf("expected Outcome 'success', got %q", summary.Outcome)
	}
	if summary.TokensUsed != 10000 {
		t.Errorf("expected TokensUsed 10000, got %d", summary.TokensUsed)
	}
}
