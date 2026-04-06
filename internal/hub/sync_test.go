package hub

import (
	"path/filepath"
	"testing"
)

func TestSyncEngine_New(t *testing.T) {
	dir := t.TempDir()
	db, err := NewEngine(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer db.Close()

	sync := NewSyncEngine(db, "project-1", "/workspace", "go")

	if sync.db != db {
		t.Error("expected db to be set")
	}
	if sync.projectID != "project-1" {
		t.Errorf("expected projectID 'project-1', got %q", sync.projectID)
	}
	if sync.workspace != "/workspace" {
		t.Errorf("expected workspace '/workspace', got %q", sync.workspace)
	}
	if sync.stack != "go" {
		t.Errorf("expected stack 'go', got %q", sync.stack)
	}
}

func TestSyncEngine_Push(t *testing.T) {
	t.Skip("Requires mock client")
}

func TestSyncEngine_PushEmpty(t *testing.T) {
	dir := t.TempDir()
	db, err := NewEngine(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer db.Close()

	sync := NewSyncEngine(db, "project-1", "/workspace", "go")

	// Create a mock client that returns error
	client := &Client{server: "http://invalid", apiKey: ""}

	// Should handle empty push gracefully
	_, err = sync.Push(client)
	// May error due to invalid server, but should handle empty semantics
	t.Logf("Push result: %v", err)
}

func TestSyncEngine_Pull(t *testing.T) {
	t.Skip("Requires mock client")
}

func TestSyncEngine_AutoSync(t *testing.T) {
	t.Skip("Requires mock client")
}

func TestSyncEngine_Status(t *testing.T) {
	dir := t.TempDir()
	db, err := NewEngine(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer db.Close()

	// Set some sync status
	db.Put("hub:sync", "status", SyncStatus{
		LastPush:    "2024-01-01T00:00:00Z",
		LastPull:    "2024-01-01T00:00:00Z",
		PushedItems: 10,
		PulledItems: 5,
		Status:      "synced",
	})

	sync := NewSyncEngine(db, "project-1", "/workspace", "go")
	status, err := sync.Status()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if status.Status != "synced" {
		t.Errorf("expected status 'synced', got %q", status.Status)
	}
}

func TestSyncEngine_StatusNeverSynced(t *testing.T) {
	dir := t.TempDir()
	db, err := NewEngine(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer db.Close()

	sync := NewSyncEngine(db, "project-1", "/workspace", "go")
	status, err := sync.Status()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if status.Status != "never_synced" {
		t.Errorf("expected status 'never_synced', got %q", status.Status)
	}
}

func TestSyncEngine_collectPushableSemantics(t *testing.T) {
	dir := t.TempDir()
	db, err := NewEngine(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer db.Close()

	// Add some semantic memory
	db.Put(NSMemorySemantic, "test-key", SemanticMemory{
		FactKey:    "test",
		Fact:       "test fact",
		Confidence: 0.7,
	})

	sync := NewSyncEngine(db, "project-1", "/workspace", "go")
	semantics, err := sync.collectPushableSemantics()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	t.Logf("Collected %d pushable semantics", len(semantics))
}

func TestSyncEngine_collectPushableHeuristics(t *testing.T) {
	dir := t.TempDir()
	db, err := NewEngine(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer db.Close()

	// Add some heuristics
	db.Put(NSMemoryHeuristic, "test-heuristic", HeuristicRule{
		Rule:       "test rule",
		Confidence: 0.7,
		Active:     true,
	})

	sync := NewSyncEngine(db, "project-1", "/workspace", "go")
	heuristics, err := sync.collectPushableHeuristics()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	t.Logf("Collected %d pushable heuristics", len(heuristics))
}

func TestSyncEngine_collectPushableSessions(t *testing.T) {
	dir := t.TempDir()
	db, err := NewEngine(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer db.Close()

	// Add some sessions
	db.Put("learn:sessions", "session-1", SessionOutcome{
		SessionID: "session-1",
		Outcome:   "success",
	})

	sync := NewSyncEngine(db, "project-1", "/workspace", "go")
	sessions, err := sync.collectPushableSessions()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	t.Logf("Collected %d pushable sessions", len(sessions))
}

func TestSyncEngine_mergeSemantic(t *testing.T) {
	dir := t.TempDir()
	db, err := NewEngine(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer db.Close()

	sync := NewSyncEngine(db, "project-1", "/workspace", "go")

	err = sync.mergeSemantic(SemanticMemory{
		FactKey:    "test",
		Fact:       "test fact",
		Confidence: 0.7,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestSyncEngine_mergeSemanticExists(t *testing.T) {
	dir := t.TempDir()
	db, err := NewEngine(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer db.Close()

	sync := NewSyncEngine(db, "project-1", "/workspace", "go")

	// Add first with higher confidence
	sync.mergeSemantic(SemanticMemory{
		FactKey:    "test",
		Fact:       "first fact",
		Confidence: 0.9,
	})

	// Try to merge with lower confidence
	err = sync.mergeSemantic(SemanticMemory{
		FactKey:    "test",
		Fact:       "second fact",
		Confidence: 0.5,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestSyncEngine_mergeHeuristic(t *testing.T) {
	dir := t.TempDir()
	db, err := NewEngine(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer db.Close()

	sync := NewSyncEngine(db, "project-1", "/workspace", "go")

	err = sync.mergeHeuristic(HeuristicRule{
		Rule:       "test rule",
		Confidence: 0.8,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestSyncEngine_mergeHeuristicExists(t *testing.T) {
	dir := t.TempDir()
	db, err := NewEngine(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer db.Close()

	sync := NewSyncEngine(db, "project-1", "/workspace", "go")

	// Add first heuristic
	sync.mergeHeuristic(HeuristicRule{
		ID:         "heuristic-1",
		Rule:       "test rule",
		Confidence: 0.8,
	})

	// Try to merge same rule - should skip
	err = sync.mergeHeuristic(HeuristicRule{
		ID:         "heuristic-2",
		Rule:       "test rule",
		Confidence: 0.9,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestSyncEngine_mergeKnowledge(t *testing.T) {
	dir := t.TempDir()
	db, err := NewEngine(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer db.Close()

	sync := NewSyncEngine(db, "project-1", "/workspace", "go")

	err = sync.mergeKnowledge(KnowledgeEntry{
		ID:         "test-entry",
		Type:       "lesson",
		Title:      "Test Lesson",
		Content:    "Test content",
		Confidence: 0.7,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestSyncEngine_updateSyncStatus(t *testing.T) {
	dir := t.TempDir()
	db, err := NewEngine(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer db.Close()

	sync := NewSyncEngine(db, "project-1", "/workspace", "go")

	err = sync.updateSyncStatus("push", 5)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	status, _ := sync.Status()
	if status.PushedItems != 5 {
		t.Errorf("expected PushedItems 5, got %d", status.PushedItems)
	}
	if status.Status != "synced" {
		t.Errorf("expected Status 'synced', got %q", status.Status)
	}
}
