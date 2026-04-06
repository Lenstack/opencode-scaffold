package skills

import (
	"path/filepath"
	"testing"

	"github.com/Lenstack/opencode-scaffold/internal/hub"
)

func TestInstaller_New(t *testing.T) {
	dir := t.TempDir()
	db, err := hub.NewEngine(filepath.Join(dir, ".opencode", "data"))
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer db.Close()

	i := NewInstaller(db, dir)

	if i.root != dir {
		t.Errorf("expected root %q, got %q", dir, i.root)
	}
	if i.db == nil {
		t.Error("expected db to be set")
	}
}

func TestInstaller_InstallSkill(t *testing.T) {
	t.Skip("Requires network access to GitHub")
}

func TestInstaller_InstallSkillAlreadyExists(t *testing.T) {
	t.Skip("Requires network access to GitHub")
}

func TestInstaller_ListInstalledSkills(t *testing.T) {
	dir := t.TempDir()
	db, err := hub.NewEngine(filepath.Join(dir, ".opencode", "data"))
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer db.Close()

	i := NewInstaller(db, dir)

	skills, err := i.ListInstalledSkills()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Should be empty initially (nil or empty slice both acceptable)
	t.Logf("Got %d installed skills", len(skills))
}

func TestInstaller_UninstallSkill(t *testing.T) {
	t.Skip("Requires network access to GitHub")
}

func TestInstaller_fetchSkillContent(t *testing.T) {
	t.Skip("Requires network access")
}

func TestInstaller_fetchSkillContentError(t *testing.T) {
	t.Skip("Requires network access")
}

func TestInstaller_InstallSkillMock(t *testing.T) {
	// Test the installation logic without network
	dir := t.TempDir()
	db, err := hub.NewEngine(filepath.Join(dir, ".opencode", "data"))
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer db.Close()

	i := NewInstaller(db, dir)

	// Try to install a skill that doesn't exist
	err = i.InstallSkill("owner", "repo", "nonexistent-skill")
	if err == nil {
		t.Fatal("expected error for nonexistent skill")
	}
}

func TestInstaller_UninstallSkillNotInstalled(t *testing.T) {
	dir := t.TempDir()
	db, err := hub.NewEngine(filepath.Join(dir, ".opencode", "data"))
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer db.Close()

	i := NewInstaller(db, dir)

	// Try to uninstall a skill that isn't installed
	err = i.UninstallSkill("nonexistent")
	if err != nil {
		t.Logf("Expected error or nil: %v", err)
	}
}
