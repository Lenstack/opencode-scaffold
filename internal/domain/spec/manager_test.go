package spec

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

	reqs := hub.SpecRequirements{
		AcceptanceCriteria: []string{"Test passes", "Code compiles"},
		EdgeCases:          []string{"Empty input", "Null value"},
	}

	entry, err := m.Create("Feature X", reqs)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if entry == nil {
		t.Fatal("expected entry to be returned")
	}
	if entry.Name != "Feature X" {
		t.Errorf("expected Name 'Feature X', got %q", entry.Name)
	}
	if entry.Status != "draft" {
		t.Errorf("expected Status 'draft', got %q", entry.Status)
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

	reqs := hub.SpecRequirements{}
	m.Create("Feature X", reqs)

	_, err = m.Create("Feature X", reqs)
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

	reqs := hub.SpecRequirements{}
	created, _ := m.Create("Feature X", reqs)

	entry, err := m.Get(created.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if entry.Name != "Feature X" {
		t.Errorf("expected Name 'Feature X', got %q", entry.Name)
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
		t.Fatal("expected error for nonexistent spec")
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

	m.Create("Feature A", hub.SpecRequirements{})
	m.Create("Feature B", hub.SpecRequirements{})

	specs, err := m.List()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(specs) < 2 {
		t.Errorf("expected at least 2 specs, got %d", len(specs))
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

	entry, _ := m.Create("Feature X", hub.SpecRequirements{})

	err = m.UpdateStatus(entry.ID, "planned")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	updated, _ := m.Get(entry.ID)
	if updated.Status != "planned" {
		t.Errorf("expected Status 'planned', got %q", updated.Status)
	}
}

func TestManager_UpdateStatusInvalid(t *testing.T) {
	dir := t.TempDir()
	db, err := hub.NewEngine(filepath.Join(dir, ".opencode", "data"))
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer db.Close()

	m := NewManager(db)

	entry, _ := m.Create("Feature X", hub.SpecRequirements{})

	err = m.UpdateStatus(entry.ID, "invalid-status")
	if err == nil {
		t.Fatal("expected error for invalid status")
	}
}

func TestManager_UpdateImplementation(t *testing.T) {
	dir := t.TempDir()
	db, err := hub.NewEngine(filepath.Join(dir, ".opencode", "data"))
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer db.Close()

	m := NewManager(db)

	entry, _ := m.Create("Feature X", hub.SpecRequirements{})

	impl := hub.SpecImplementation{
		Files: []string{"feature.go", "feature_test.go"},
		Tests: []string{"TestFeature"},
	}

	err = m.UpdateImplementation(entry.ID, impl)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestManager_Verify(t *testing.T) {
	dir := t.TempDir()
	db, err := hub.NewEngine(filepath.Join(dir, ".opencode", "data"))
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer db.Close()

	m := NewManager(db)

	entry, _ := m.Create("Feature X", hub.SpecRequirements{})

	verification := hub.SpecVerification{
		Results: []string{"Test passed", "Code compiles"},
	}

	err = m.Verify(entry.ID, verification)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	updated, _ := m.Get(entry.ID)
	if updated.Status != "done" {
		t.Errorf("expected Status 'done' after verification, got %q", updated.Status)
	}
}

func TestManager_VerifyFailed(t *testing.T) {
	dir := t.TempDir()
	db, err := hub.NewEngine(filepath.Join(dir, ".opencode", "data"))
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer db.Close()

	m := NewManager(db)

	entry, _ := m.Create("Feature X", hub.SpecRequirements{})

	verification := hub.SpecVerification{
		Results:        []string{"Test failed"},
		FailedCriteria: []string{"Test must pass"},
	}

	err = m.Verify(entry.ID, verification)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify stores the verification but doesn't change spec status on failure
	// (only on success does it call UpdateStatus to "done")
	v, err := m.GetVerification(entry.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if v.Status != "failed" {
		t.Errorf("expected Verification Status 'failed', got %q", v.Status)
	}
}

func TestManager_GetRequirements(t *testing.T) {
	dir := t.TempDir()
	db, err := hub.NewEngine(filepath.Join(dir, ".opencode", "data"))
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer db.Close()

	m := NewManager(db)

	reqs := hub.SpecRequirements{
		AcceptanceCriteria: []string{"Criteria 1"},
		EdgeCases:          []string{"Edge 1"},
	}
	entry, _ := m.Create("Feature X", reqs)

	retrieved, err := m.GetRequirements(entry.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(retrieved.AcceptanceCriteria) != 1 {
		t.Errorf("expected 1 acceptance criteria, got %d", len(retrieved.AcceptanceCriteria))
	}
}

func TestManager_GetImplementation(t *testing.T) {
	dir := t.TempDir()
	db, err := hub.NewEngine(filepath.Join(dir, ".opencode", "data"))
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer db.Close()

	m := NewManager(db)

	entry, _ := m.Create("Feature X", hub.SpecRequirements{})

	impl := hub.SpecImplementation{
		Files: []string{"file.go"},
	}
	m.UpdateImplementation(entry.ID, impl)

	retrieved, err := m.GetImplementation(entry.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(retrieved.Files) != 1 {
		t.Errorf("expected 1 file, got %d", len(retrieved.Files))
	}
}

func TestManager_GetVerification(t *testing.T) {
	dir := t.TempDir()
	db, err := hub.NewEngine(filepath.Join(dir, ".opencode", "data"))
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer db.Close()

	m := NewManager(db)

	entry, _ := m.Create("Feature X", hub.SpecRequirements{})

	verification := hub.SpecVerification{
		Results: []string{"passed"},
	}
	m.Verify(entry.ID, verification)

	retrieved, err := m.GetVerification(entry.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(retrieved.Results) != 1 {
		t.Errorf("expected 1 result, got %d", len(retrieved.Results))
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

	entry, _ := m.Create("Feature X", hub.SpecRequirements{})

	err = m.Delete(entry.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	_, err = m.Get(entry.ID)
	if err == nil {
		t.Fatal("expected error after delete")
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

	entry, _ := m.Create("Feature X", hub.SpecRequirements{})

	err = m.Archive(entry.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	updated, _ := m.Get(entry.ID)
	if updated.Status != "archived" {
		t.Errorf("expected Status 'archived', got %q", updated.Status)
	}
}

func TestSlugify(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"feature x", "feature-x"},
		{"my feature", "my-feature"},
		{"hello world", "hello-world"},
		{"already-slug", "already-slug"},
		{"test123", "test123"},
		{"Feature X", "eature"}, // uppercase F and X stripped, trailing dash trimmed
	}

	for _, tt := range tests {
		result := slugify(tt.input)
		if result != tt.expected {
			t.Errorf("slugify(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}
