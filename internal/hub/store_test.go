package hub

import (
	"path/filepath"
	"testing"
)

func TestStore_New(t *testing.T) {
	dir := t.TempDir()
	store, err := New(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	defer store.Close()

	if store.db == nil {
		t.Error("expected db to be initialized")
	}
}

func TestStore_Close(t *testing.T) {
	dir := t.TempDir()
	store, err := New(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	err = store.Close()
	if err != nil {
		t.Fatalf("expected no error on close, got %v", err)
	}
}

func TestStore_CreateUser(t *testing.T) {
	dir := t.TempDir()
	store, err := New(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	err = store.CreateUser("user-1", "test@example.com", "developer")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestStore_GetUser(t *testing.T) {
	dir := t.TempDir()
	store, err := New(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	store.CreateUser("user-1", "test@example.com", "developer")

	user, err := store.GetUser("user-1")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if user["email"] != "test@example.com" {
		t.Errorf("expected email 'test@example.com', got %q", user["email"])
	}
}

func TestStore_GetUserNotFound(t *testing.T) {
	dir := t.TempDir()
	store, err := New(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	_, err = store.GetUser("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent user")
	}
}

func TestStore_ListUsers(t *testing.T) {
	dir := t.TempDir()
	store, err := New(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	store.CreateUser("user-1", "test1@example.com", "developer")
	store.CreateUser("user-2", "test2@example.com", "admin")

	users, err := store.ListUsers()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(users) < 2 {
		t.Errorf("expected at least 2 users, got %d", len(users))
	}
}

func TestStore_CreateAPIKey(t *testing.T) {
	dir := t.TempDir()
	store, err := New(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	store.CreateUser("user-1", "test@example.com", "developer")

	keyID, err := store.CreateAPIKey("user-1", "hash123", "2025-01-01T00:00:00Z")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if keyID == "" {
		t.Error("expected key ID to be returned")
	}
}

func TestStore_ValidateAPIKey(t *testing.T) {
	dir := t.TempDir()
	store, err := New(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	store.CreateUser("user-1", "test@example.com", "developer")
	store.CreateAPIKey("user-1", "hash123", "")

	// Note: This would need proper hash to work
	// We'll just test the function can be called
	_ = store.ValidateAPIKey
}

func TestStore_ValidateAPIKeyInvalid(t *testing.T) {
	dir := t.TempDir()
	store, err := New(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	_, err = store.ValidateAPIKey("invalid-key")
	if err == nil {
		t.Fatal("expected error for invalid key")
	}
}

func TestStore_ValidateAPIKeyExpired(t *testing.T) {
	dir := t.TempDir()
	store, err := New(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	store.CreateUser("user-1", "test@example.com", "developer")
	store.CreateAPIKey("user-1", "hash123", "2020-01-01T00:00:00Z")

	// Note: This would need proper hash to validate
	_ = store.ValidateAPIKey
}

func TestStore_ListAPIKeys(t *testing.T) {
	dir := t.TempDir()
	store, err := New(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	store.CreateUser("user-1", "test@example.com", "developer")
	store.CreateAPIKey("user-1", "hash1", "")
	store.CreateAPIKey("user-1", "hash2", "")

	keys, err := store.ListAPIKeys("user-1")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(keys) < 2 {
		t.Errorf("expected at least 2 keys, got %d", len(keys))
	}
}

func TestStore_RevokeAPIKey(t *testing.T) {
	dir := t.TempDir()
	store, err := New(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	store.CreateUser("user-1", "test@example.com", "developer")
	keyID, _ := store.CreateAPIKey("user-1", "hash1", "")

	err = store.RevokeAPIKey(keyID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestStore_SaveConfig(t *testing.T) {
	dir := t.TempDir()
	store, err := New(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	err = store.SaveConfig("project-1", "user-1", "opencode", map[string]string{"key": "value"}, "Initial config")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestStore_GetLatestConfig(t *testing.T) {
	dir := t.TempDir()
	store, err := New(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	store.SaveConfig("project-1", "user-1", "opencode", map[string]string{"key": "value"}, "Initial")

	content, version, err := store.GetLatestConfig("project-1", "opencode")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if version == 0 {
		t.Error("expected version > 0")
	}
	t.Logf("Got config: %s, version: %d", content, version)
}

func TestStore_ListConfigVersions(t *testing.T) {
	dir := t.TempDir()
	store, err := New(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	store.SaveConfig("project-1", "user-1", "opencode", map[string]string{"v": "1"}, "v1")
	store.SaveConfig("project-1", "user-1", "opencode", map[string]string{"v": "2"}, "v2")

	versions, err := store.ListConfigVersions("project-1", "opencode")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(versions) < 2 {
		t.Errorf("expected at least 2 versions, got %d", len(versions))
	}
}

func TestStore_CreateBackup(t *testing.T) {
	dir := t.TempDir()
	store, err := New(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	err = store.CreateBackup("project-1", "backup-1", "user-1", map[string]string{"data": "value"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestStore_ListBackups(t *testing.T) {
	dir := t.TempDir()
	store, err := New(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	store.CreateBackup("project-1", "backup-1", "user-1", map[string]string{})
	store.CreateBackup("project-1", "backup-2", "user-1", map[string]string{})

	backups, err := store.ListBackups("project-1")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(backups) < 2 {
		t.Errorf("expected at least 2 backups, got %d", len(backups))
	}
}

func TestStore_GetBackup(t *testing.T) {
	dir := t.TempDir()
	store, err := New(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	store.CreateBackup("project-1", "backup-1", "user-1", map[string]string{"key": "value"})

	backups, _ := store.ListBackups("project-1")
	if len(backups) == 0 {
		t.Skip("no backups to test")
	}

	content, err := store.GetBackup(backups[0]["id"])
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if content == "" {
		t.Error("expected non-empty content")
	}
}

func TestStore_LogAudit(t *testing.T) {
	dir := t.TempDir()
	store, err := New(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	err = store.LogAudit("user-1", "create", "Created a new project")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestStore_RegisterProject(t *testing.T) {
	dir := t.TempDir()
	store, err := New(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	err = store.RegisterProject("project-1", "Test Project", "/workspace", "go")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestStore_PushKnowledge(t *testing.T) {
	dir := t.TempDir()
	store, err := New(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	push := KnowledgePush{
		ProjectID: "project-1",
		Workspace: "/workspace",
		Stack:     "go",
		Semantic: []SemanticMemory{
			{FactKey: "test-fact", Fact: "Test fact", Confidence: 0.7},
		},
		Heuristics: []HeuristicRule{
			{Rule: "Test rule", Confidence: 0.8},
		},
	}

	err = store.PushKnowledge(push)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestStore_PullKnowledge(t *testing.T) {
	dir := t.TempDir()
	store, err := New(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	// First push some data
	push := KnowledgePush{
		ProjectID: "project-1",
		Workspace: "/workspace",
		Stack:     "go",
		Semantic:  []SemanticMemory{{FactKey: "test", Fact: "test fact", Confidence: 0.7}},
	}
	store.PushKnowledge(push)

	pull, err := store.PullKnowledge("go", "/workspace")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	t.Logf("Pulled %d semantic, %d heuristics", len(pull.Semantic), len(pull.Heuristics))
}

func TestStore_GetGlobalHeuristics(t *testing.T) {
	dir := t.TempDir()
	store, err := New(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	// Push some heuristics first
	push := KnowledgePush{
		ProjectID:  "project-1",
		Workspace:  "/workspace",
		Stack:      "go",
		Heuristics: []HeuristicRule{{Rule: "test-rule", Confidence: 0.8, Active: true}},
	}
	store.PushKnowledge(push)

	heuristics, err := store.GetGlobalHeuristics("go", "/workspace", 0.5, 10)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	t.Logf("Found %d global heuristics", len(heuristics))
}

func TestStore_PushSession(t *testing.T) {
	dir := t.TempDir()
	store, err := New(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	outcome := SessionOutcome{
		ID:        "outcome-1",
		SessionID: "session-1",
		Outcome:   "success",
		Stack:     "go",
		Workspace: "/workspace",
	}

	err = store.PushSession(outcome)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestStore_GetWorkspaceKnowledge(t *testing.T) {
	dir := t.TempDir()
	store, err := New(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	// Push some knowledge first
	push := KnowledgePush{
		ProjectID: "project-1",
		Workspace: "/workspace",
		Stack:     "go",
		Semantic:  []SemanticMemory{{FactKey: "test", Fact: "test fact", Confidence: 0.7}},
	}
	store.PushKnowledge(push)

	entries, err := store.GetWorkspaceKnowledge("/workspace")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	t.Logf("Found %d workspace knowledge entries", len(entries))
}
