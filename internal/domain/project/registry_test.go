package project

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/Lenstack/opencode-scaffold/internal/hub"
)

func TestRegistry_Structure(t *testing.T) {
	reg := &Registry{
		ProjectID:    "test-id",
		ProjectName:  "test-project",
		HubURL:       "https://hub.example.com",
		Workspace:    "/home/user/project",
		APIKey:       "secret-key",
		RegisteredAt: "2024-01-01T00:00:00Z",
		Stack:        "go",
	}

	if reg.ProjectID != "test-id" {
		t.Errorf("expected ProjectID 'test-id', got %q", reg.ProjectID)
	}
	if reg.ProjectName != "test-project" {
		t.Errorf("expected ProjectName 'test-project', got %q", reg.ProjectName)
	}
	if reg.HubURL != "https://hub.example.com" {
		t.Errorf("expected HubURL 'https://hub.example.com', got %q", reg.HubURL)
	}
	if reg.Stack != "go" {
		t.Errorf("expected Stack 'go', got %q", reg.Stack)
	}
}

func TestManager_Register(t *testing.T) {
	dir := t.TempDir()
	db, err := hub.NewEngine(filepath.Join(dir, ".opencode", "data"))
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer db.Close()

	m := NewManager(db, dir)

	reg, err := m.Register("test-project", "https://hub.example.com", "/workspace", "api-key")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if reg == nil {
		t.Fatal("expected registry to be returned")
	}
	if reg.ProjectID == "" {
		t.Error("expected ProjectID to be set")
	}
	if reg.ProjectName != "test-project" {
		t.Errorf("expected ProjectName 'test-project', got %q", reg.ProjectName)
	}
	if reg.HubURL != "https://hub.example.com" {
		t.Errorf("expected HubURL 'https://hub.example.com', got %q", reg.HubURL)
	}
	if reg.RegisteredAt == "" {
		t.Error("expected RegisteredAt to be set")
	}
}

func TestManager_RegisterDuplicate(t *testing.T) {
	dir := t.TempDir()
	db, err := hub.NewEngine(filepath.Join(dir, ".opencode", "data"))
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer db.Close()

	m := NewManager(db, dir)

	_, err = m.Register("test-project", "https://hub.example.com", "/workspace", "api-key")
	if err != nil {
		t.Fatalf("expected no error on first register, got %v", err)
	}

	_, err = m.Register("test-project", "https://hub.example.com", "/workspace", "api-key")
	if err == nil {
		t.Fatal("expected error on duplicate register")
	}
}

func TestManager_Unregister(t *testing.T) {
	dir := t.TempDir()
	db, err := hub.NewEngine(filepath.Join(dir, ".opencode", "data"))
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer db.Close()

	m := NewManager(db, dir)

	_, err = m.Register("test-project", "https://hub.example.com", "/workspace", "api-key")
	if err != nil {
		t.Fatalf("failed to register: %v", err)
	}

	err = m.Unregister()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Should not be registered anymore
	if m.IsRegistered() {
		t.Error("expected project to be unregistered")
	}
}

func TestManager_Load(t *testing.T) {
	dir := t.TempDir()
	db, err := hub.NewEngine(filepath.Join(dir, ".opencode", "data"))
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer db.Close()

	m := NewManager(db, dir)

	reg, err := m.Register("test-project", "https://hub.example.com", "/workspace", "api-key")
	if err != nil {
		t.Fatalf("failed to register: %v", err)
	}

	loaded, err := m.Load()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if loaded.ProjectID != reg.ProjectID {
		t.Errorf("expected ProjectID %q, got %q", reg.ProjectID, loaded.ProjectID)
	}
	if loaded.ProjectName != "test-project" {
		t.Errorf("expected ProjectName 'test-project', got %q", loaded.ProjectName)
	}
}

func TestManager_LoadNotRegistered(t *testing.T) {
	dir := t.TempDir()
	db, err := hub.NewEngine(filepath.Join(dir, ".opencode", "data"))
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer db.Close()

	m := NewManager(db, dir)

	_, err = m.Load()
	if err == nil {
		t.Fatal("expected error when loading unregistered project")
	}
}

func TestManager_Save(t *testing.T) {
	dir := t.TempDir()
	db, err := hub.NewEngine(filepath.Join(dir, ".opencode", "data"))
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer db.Close()

	m := NewManager(db, dir)

	reg := &Registry{
		ProjectID:   "test-id",
		ProjectName: "test-project",
		HubURL:      "https://hub.example.com",
		Workspace:   "/workspace",
	}

	err = m.Save(reg)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify file was created
	path := m.registryPath()
	if _, err := os.Stat(path); err != nil {
		t.Errorf("expected registry file to exist at %s", path)
	}
}

func TestManager_UpdateSyncTime(t *testing.T) {
	dir := t.TempDir()
	db, err := hub.NewEngine(filepath.Join(dir, ".opencode", "data"))
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer db.Close()

	m := NewManager(db, dir)

	_, err = m.Register("test-project", "https://hub.example.com", "/workspace", "api-key")
	if err != nil {
		t.Fatalf("failed to register: %v", err)
	}

	err = m.UpdateSyncTime()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// UpdateSyncTime saves to file, so read the file directly
	path := m.registryPath()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read registry file: %v", err)
	}

	var reg Registry
	if err := json.Unmarshal(data, &reg); err != nil {
		t.Fatalf("failed to parse registry: %v", err)
	}

	if reg.LastSync == "" {
		t.Error("expected LastSync to be set")
	}
}

func TestManager_IsRegistered(t *testing.T) {
	dir := t.TempDir()
	db, err := hub.NewEngine(filepath.Join(dir, ".opencode", "data"))
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer db.Close()

	m := NewManager(db, dir)

	if m.IsRegistered() {
		t.Error("expected not to be registered initially")
	}

	_, err = m.Register("test-project", "https://hub.example.com", "/workspace", "api-key")
	if err != nil {
		t.Fatalf("failed to register: %v", err)
	}

	if !m.IsRegistered() {
		t.Error("expected to be registered after registering")
	}
}

func TestManager_IsRegisteredEmpty(t *testing.T) {
	dir := t.TempDir()
	db, err := hub.NewEngine(filepath.Join(dir, ".opencode", "data"))
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer db.Close()

	m := NewManager(db, dir)

	// Empty directory should not be registered
	if m.IsRegistered() {
		t.Error("expected not to be registered")
	}
}

func TestManager_registryPath(t *testing.T) {
	dir := t.TempDir()
	db, err := hub.NewEngine(filepath.Join(dir, ".opencode", "data"))
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer db.Close()

	m := NewManager(db, dir)

	path := m.registryPath()
	expected := filepath.Join(dir, ".opencode", "data", "project.json")
	if path != expected {
		t.Errorf("expected path %q, got %q", expected, path)
	}
}
