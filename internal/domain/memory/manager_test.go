package memory

import (
	"path/filepath"
	"testing"

	"github.com/Lenstack/opencode-scaffold/internal/hub"
)

func TestManager_AddEpisodic(t *testing.T) {
	dir := t.TempDir()
	db, err := hub.NewEngine(filepath.Join(dir, ".opencode", "data"))
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer db.Close()

	m := NewManager(db)

	mem := hub.EpisodicMemory{
		SessionID: "session-1",
		Feature:   "test feature",
		AgentsRan: []string{"tester", "reviewer"},
		Outcome:   "success",
		KeyLesson: "Always write tests first",
	}

	err = m.AddEpisodic(mem)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestManager_AddSemantic(t *testing.T) {
	dir := t.TempDir()
	db, err := hub.NewEngine(filepath.Join(dir, ".opencode", "data"))
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer db.Close()

	m := NewManager(db)

	mem := hub.SemanticMemory{
		Category: "workflow",
		FactKey:  "tdd-workflow",
		Fact:     "Always write failing tests before implementation",
		Source:   "session-1",
	}

	err = m.AddSemantic(mem)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestManager_AddSemanticExisting(t *testing.T) {
	dir := t.TempDir()
	db, err := hub.NewEngine(filepath.Join(dir, ".opencode", "data"))
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer db.Close()

	m := NewManager(db)

	mem := hub.SemanticMemory{
		Category: "workflow",
		FactKey:  "tdd-workflow",
		Fact:     "Always write failing tests before implementation",
		Source:   "session-1",
	}

	// Add twice
	err = m.AddSemantic(mem)
	if err != nil {
		t.Fatalf("expected no error on first add, got %v", err)
	}

	err = m.AddSemantic(mem)
	if err != nil {
		t.Fatalf("expected no error on second add, got %v", err)
	}
}

func TestManager_AddHeuristic(t *testing.T) {
	dir := t.TempDir()
	db, err := hub.NewEngine(filepath.Join(dir, ".opencode", "data"))
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer db.Close()

	m := NewManager(db)

	rule := hub.HeuristicRule{
		Rule:           "Always use TDD",
		Rationale:      "Tests ensure correct behavior",
		SourceSessions: []string{"session-1"},
		Active:         true,
		Confidence:     0.8,
	}

	err = m.AddHeuristic(rule)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestManager_AddQuarantine(t *testing.T) {
	dir := t.TempDir()
	db, err := hub.NewEngine(filepath.Join(dir, ".opencode", "data"))
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer db.Close()

	m := NewManager(db)

	err = m.AddQuarantine("low confidence fact", 0.4, "below threshold")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestManager_SearchEpisodic(t *testing.T) {
	dir := t.TempDir()
	db, err := hub.NewEngine(filepath.Join(dir, ".opencode", "data"))
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer db.Close()

	m := NewManager(db)

	mem := hub.EpisodicMemory{
		SessionID: "session-1",
		Feature:   "TDD workflow",
		KeyLesson: "Write tests first",
	}

	err = m.AddEpisodic(mem)
	if err != nil {
		t.Fatalf("failed to add: %v", err)
	}

	results, err := m.SearchEpisodic("TDD")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(results) == 0 {
		t.Error("expected to find results")
	}
}

func TestManager_SearchSemantic(t *testing.T) {
	dir := t.TempDir()
	db, err := hub.NewEngine(filepath.Join(dir, ".opencode", "data"))
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer db.Close()

	m := NewManager(db)

	mem := hub.SemanticMemory{
		Category:   "workflow",
		FactKey:    "tdd-workflow",
		Fact:       "Always write failing tests",
		Confidence: 0.7,
	}

	err = m.AddSemantic(mem)
	if err != nil {
		t.Fatalf("failed to add: %v", err)
	}

	results, err := m.SearchSemantic("tdd", 0.5)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(results) == 0 {
		t.Error("expected to find results")
	}
}

func TestManager_SearchSemanticMinConfidence(t *testing.T) {
	dir := t.TempDir()
	db, err := hub.NewEngine(filepath.Join(dir, ".opencode", "data"))
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer db.Close()

	m := NewManager(db)

	mem := hub.SemanticMemory{
		Category:   "workflow",
		FactKey:    "low-confidence",
		Fact:       "Some fact",
		Confidence: 0.3,
	}

	err = m.AddSemantic(mem)
	if err != nil {
		t.Fatalf("failed to add: %v", err)
	}

	results, err := m.SearchSemantic("low", 0.6)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Should not find because confidence is below threshold
	if len(results) > 0 {
		t.Error("expected no results due to low confidence")
	}
}

func TestManager_ListHeuristics(t *testing.T) {
	dir := t.TempDir()
	db, err := hub.NewEngine(filepath.Join(dir, ".opencode", "data"))
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer db.Close()

	m := NewManager(db)

	rule := hub.HeuristicRule{
		Rule:       "Test rule",
		Active:     true,
		Confidence: 0.7,
	}

	err = m.AddHeuristic(rule)
	if err != nil {
		t.Fatalf("failed to add: %v", err)
	}

	rules, err := m.ListHeuristics()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(rules) == 0 {
		t.Error("expected to find rules")
	}
}

func TestManager_PruneExpired(t *testing.T) {
	dir := t.TempDir()
	db, err := hub.NewEngine(filepath.Join(dir, ".opencode", "data"))
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer db.Close()

	m := NewManager(db)

	// Add some data
	mem := hub.EpisodicMemory{
		SessionID: "session-1",
		Feature:   "test",
	}
	m.AddEpisodic(mem)

	count, err := m.PruneExpired()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Should succeed even if nothing is pruned
	if count < 0 {
		t.Errorf("expected non-negative count, got %d", count)
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

	_, err = m.Get("test")
	if err == nil {
		t.Fatal("expected error from Get")
	}
}

func TestContainsAny(t *testing.T) {
	tests := []struct {
		text  string
		query string
		want  bool
	}{
		{"hello world", "hello", true},
		{"hello world", "world", true},
		{"hello world", "foo", false},
		{"test feature", "feature", true},
		{"some text", "other", false},
		{"short", "somethinglonger", false},
	}

	for _, tt := range tests {
		result := containsAny(tt.text, tt.query)
		if result != tt.want {
			t.Errorf("containsAny(%q, %q) = %v, want %v", tt.text, tt.query, result, tt.want)
		}
	}
}

func TestSplitWords(t *testing.T) {
	result := splitWords("hello world test")
	if len(result) < 2 {
		t.Errorf("expected at least 2 words, got %v", result)
	}
}

func TestHashKey(t *testing.T) {
	key1 := hashKey("test")
	key2 := hashKey("test")
	key3 := hashKey("different")

	if key1 != key2 {
		t.Error("expected same input to produce same hash")
	}
	if key1 == key3 {
		t.Error("expected different inputs to produce different hashes")
	}
}
