package learn

import (
	"path/filepath"
	"testing"

	"github.com/Lenstack/opencode-scaffold/internal/hub"
)

func TestEngine_New(t *testing.T) {
	dir := t.TempDir()
	db, err := hub.NewEngine(filepath.Join(dir, ".opencode", "data"))
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer db.Close()

	e := NewEngine(db)

	if e.db == nil {
		t.Error("expected db to be set")
	}
}

func TestEngine_RecordSession(t *testing.T) {
	dir := t.TempDir()
	db, err := hub.NewEngine(filepath.Join(dir, ".opencode", "data"))
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer db.Close()

	e := NewEngine(db)

	outcome := SessionOutcome{
		SessionID: "session-1",
		Outcome:   "success",
		Agents:    []string{"orchestrator", "tester"},
		Skills:    []string{"tdd-workflow"},
		Template:  "standard",
		Stack:     "go",
		Duration:  120,
	}

	err = e.RecordSession(outcome)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestEngine_GetSessions(t *testing.T) {
	dir := t.TempDir()
	db, err := hub.NewEngine(filepath.Join(dir, ".opencode", "data"))
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer db.Close()

	e := NewEngine(db)

	// Record some sessions
	e.RecordSession(SessionOutcome{SessionID: "session-1", Outcome: "success"})
	e.RecordSession(SessionOutcome{SessionID: "session-2", Outcome: "failure"})

	sessions, err := e.GetSessions(0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(sessions) < 2 {
		t.Errorf("expected at least 2 sessions, got %d", len(sessions))
	}
}

func TestEngine_GetSessionsLimit(t *testing.T) {
	dir := t.TempDir()
	db, err := hub.NewEngine(filepath.Join(dir, ".opencode", "data"))
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer db.Close()

	e := NewEngine(db)

	// Record multiple sessions
	for i := 0; i < 5; i++ {
		e.RecordSession(SessionOutcome{SessionID: "session-" + string(rune('0'+i)), Outcome: "success"})
	}

	sessions, err := e.GetSessions(2)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(sessions) > 2 {
		t.Errorf("expected at most 2 sessions, got %d", len(sessions))
	}
}

func TestEngine_GetSessionStats(t *testing.T) {
	dir := t.TempDir()
	db, err := hub.NewEngine(filepath.Join(dir, ".opencode", "data"))
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer db.Close()

	e := NewEngine(db)

	e.RecordSession(SessionOutcome{Outcome: "success"})
	e.RecordSession(SessionOutcome{Outcome: "success"})
	e.RecordSession(SessionOutcome{Outcome: "failure"})

	stats, err := e.GetSessionStats()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if stats["total"] != 3 {
		t.Errorf("expected total 3, got %d", stats["total"])
	}
	if stats["success"] != 2 {
		t.Errorf("expected success 2, got %d", stats["success"])
	}
	if stats["failure"] != 1 {
		t.Errorf("expected failure 1, got %d", stats["failure"])
	}
}

func TestEngine_detectPatterns(t *testing.T) {
	dir := t.TempDir()
	db, err := hub.NewEngine(filepath.Join(dir, ".opencode", "data"))
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer db.Close()

	e := NewEngine(db)

	outcome := SessionOutcome{
		Agents:   []string{"orchestrator", "tester"},
		Outcome:  "success",
		Template: "standard",
		Stack:    "go",
	}

	e.detectPatterns(outcome)
	// This test just verifies the function runs without error
}

func TestEngine_incrementPattern(t *testing.T) {
	dir := t.TempDir()
	db, err := hub.NewEngine(filepath.Join(dir, ".opencode", "data"))
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer db.Close()

	e := NewEngine(db)

	e.incrementPattern("test-pattern", "workflow", true)
	e.incrementPattern("test-pattern", "workflow", true)
	e.incrementPattern("test-pattern", "workflow", false)

	patterns, _ := e.GetPatterns("workflow", 1)
	if len(patterns) == 0 {
		t.Error("expected pattern to be created")
	}
}

func TestEngine_GetPatterns(t *testing.T) {
	dir := t.TempDir()
	db, err := hub.NewEngine(filepath.Join(dir, ".opencode", "data"))
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer db.Close()

	e := NewEngine(db)

	e.incrementPattern("pattern1", "workflow", true)
	e.incrementPattern("pattern1", "workflow", true)
	e.incrementPattern("pattern2", "skill", true)

	patterns, err := e.GetPatterns("workflow", 1)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(patterns) == 0 {
		t.Error("expected patterns")
	}
}

func TestEngine_GetPatternsMinOccurrences(t *testing.T) {
	dir := t.TempDir()
	db, err := hub.NewEngine(filepath.Join(dir, ".opencode", "data"))
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer db.Close()

	e := NewEngine(db)

	e.incrementPattern("rare-pattern", "workflow", true)

	patterns, err := e.GetPatterns("workflow", 5)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Should not find because min occurrences is 5
	if len(patterns) > 0 {
		t.Error("expected no patterns with high min occurrences")
	}
}

func TestEngine_PromotePattern(t *testing.T) {
	dir := t.TempDir()
	db, err := hub.NewEngine(filepath.Join(dir, ".opencode", "data"))
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer db.Close()

	e := NewEngine(db)

	e.incrementPattern("test-pattern", "workflow", true)
	e.incrementPattern("test-pattern", "workflow", true)
	e.incrementPattern("test-pattern", "workflow", true)

	err = e.PromotePattern("test-pattern")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	patterns, _ := e.GetPatterns("workflow", 1)
	for _, p := range patterns {
		if p.ID == "test-pattern" && !p.Promoted {
			t.Error("expected pattern to be promoted")
		}
	}
}

func TestEngine_GetHeuristics(t *testing.T) {
	dir := t.TempDir()
	db, err := hub.NewEngine(filepath.Join(dir, ".opencode", "data"))
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer db.Close()

	e := NewEngine(db)

	// Get heuristics when none exist
	heuristics, err := e.GetHeuristics(0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	t.Logf("Found %d heuristics", len(heuristics))
}

func TestEngine_GetHeuristicsMinConfidence(t *testing.T) {
	dir := t.TempDir()
	db, err := hub.NewEngine(filepath.Join(dir, ".opencode", "data"))
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer db.Close()

	e := NewEngine(db)

	heuristics, err := e.GetHeuristics(0.9)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Should be empty or filtered by confidence
	t.Logf("Found %d high-confidence heuristics", len(heuristics))
}

func TestEngine_DemoteHeuristic(t *testing.T) {
	dir := t.TempDir()
	db, err := hub.NewEngine(filepath.Join(dir, ".opencode", "data"))
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer db.Close()

	e := NewEngine(db)

	// First need to create a heuristic via PromotePattern
	e.incrementPattern("demote-pattern", "workflow", true)
	e.incrementPattern("demote-pattern", "workflow", true)
	e.incrementPattern("demote-pattern", "workflow", true)
	e.PromotePattern("demote-pattern")

	// Then demote it
	err = e.DemoteHeuristic("demote-pattern-1") // ID may vary
	if err != nil {
		t.Logf("Expected error or success: %v", err)
	}
}

func TestEngine_updateSkillStats(t *testing.T) {
	dir := t.TempDir()
	db, err := hub.NewEngine(filepath.Join(dir, ".opencode", "data"))
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer db.Close()

	e := NewEngine(db)

	e.updateSkillStats("tdd-workflow", "success")
	e.updateSkillStats("tdd-workflow", "success")
	e.updateSkillStats("tdd-workflow", "failure")

	stats, _ := e.GetSkillStats()
	if len(stats) == 0 {
		t.Error("expected skill stats")
	}
}

func TestEngine_GetSkillStats(t *testing.T) {
	dir := t.TempDir()
	db, err := hub.NewEngine(filepath.Join(dir, ".opencode", "data"))
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer db.Close()

	e := NewEngine(db)

	stats, err := e.GetSkillStats()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Should return empty slice when no stats exist
	t.Logf("Got %d skill stats", len(stats))
}

func TestEngine_updateTemplateStats(t *testing.T) {
	dir := t.TempDir()
	db, err := hub.NewEngine(filepath.Join(dir, ".opencode", "data"))
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer db.Close()

	e := NewEngine(db)

	e.updateTemplateStats("standard", "success", "go")
	e.updateTemplateStats("standard", "success", "go")
	e.updateTemplateStats("standard", "failure", "python")

	stats, _ := e.GetTemplateStats()
	if len(stats) == 0 {
		t.Error("expected template stats")
	}
}

func TestEngine_GetTemplateStats(t *testing.T) {
	dir := t.TempDir()
	db, err := hub.NewEngine(filepath.Join(dir, ".opencode", "data"))
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer db.Close()

	e := NewEngine(db)

	stats, err := e.GetTemplateStats()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	t.Logf("Found %d template stats", len(stats))
}

func TestEngine_updateAgentStats(t *testing.T) {
	dir := t.TempDir()
	db, err := hub.NewEngine(filepath.Join(dir, ".opencode", "data"))
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer db.Close()

	e := NewEngine(db)

	e.updateAgentStats("orchestrator", "success", 100)
	e.updateAgentStats("orchestrator", "success", 100)
	e.updateAgentStats("orchestrator", "failure", 100)

	stats, _ := e.GetAgentStats()
	if len(stats) == 0 {
		t.Error("expected agent stats")
	}
}

func TestEngine_GetAgentStats(t *testing.T) {
	dir := t.TempDir()
	db, err := hub.NewEngine(filepath.Join(dir, ".opencode", "data"))
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer db.Close()

	e := NewEngine(db)

	stats, err := e.GetAgentStats()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	t.Logf("Found %d agent stats", len(stats))
}

func TestEngine_ExtractKnowledge(t *testing.T) {
	dir := t.TempDir()
	db, err := hub.NewEngine(filepath.Join(dir, ".opencode", "data"))
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer db.Close()

	e := NewEngine(db)

	e.RecordSession(SessionOutcome{
		Outcome: "failure",
		Notes:   "Test failure note",
	})

	entries, err := e.ExtractKnowledge(7)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	t.Logf("Extracted %d knowledge entries", len(entries))
}

func TestEngine_GetKnowledge(t *testing.T) {
	dir := t.TempDir()
	db, err := hub.NewEngine(filepath.Join(dir, ".opencode", "data"))
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer db.Close()

	e := NewEngine(db)

	entries, err := e.GetKnowledge("")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	t.Logf("Found %d knowledge entries", len(entries))
}

func TestEngine_RunLearningCycle(t *testing.T) {
	dir := t.TempDir()
	db, err := hub.NewEngine(filepath.Join(dir, ".opencode", "data"))
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer db.Close()

	e := NewEngine(db)

	// Add some data
	e.RecordSession(SessionOutcome{Outcome: "success", Notes: "Good session"})

	results, err := e.RunLearningCycle()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	t.Logf("Learning cycle results: %v", results)
}

func TestEngine_SessionOutcome(t *testing.T) {
	outcome := SessionOutcome{
		ID:        "test-id",
		SessionID: "session-1",
		Outcome:   "success",
		Agents:    []string{"orchestrator"},
		Skills:    []string{"tdd-workflow"},
		Template:  "standard",
		Stack:     "go",
		Duration:  120,
		Notes:     "Test notes",
	}

	if outcome.Outcome != "success" {
		t.Errorf("expected Outcome 'success', got %q", outcome.Outcome)
	}
	if len(outcome.Agents) != 1 {
		t.Errorf("expected 1 agent, got %d", len(outcome.Agents))
	}
}

func TestEngine_Pattern(t *testing.T) {
	pattern := Pattern{
		ID:          "test-pattern",
		Name:        "Test Pattern",
		Description: "A test pattern",
		Category:    "workflow",
		Occurrences: 5,
		SuccessRate: 0.8,
		Promoted:    false,
		Confidence:  0.8,
	}

	if pattern.Occurrences != 5 {
		t.Errorf("expected Occurrences 5, got %d", pattern.Occurrences)
	}
	if pattern.SuccessRate != 0.8 {
		t.Errorf("expected SuccessRate 0.8, got %f", pattern.SuccessRate)
	}
}

func TestEngine_Heuristic(t *testing.T) {
	heuristic := Heuristic{
		ID:              "test-heuristic",
		Rule:            "Always use TDD",
		Rationale:       "Tests ensure correctness",
		Source:          "manual",
		Confidence:      0.9,
		InvocationCount: 10,
		OverrideCount:   1,
		SuccessRate:     0.9,
	}

	if heuristic.Confidence != 0.9 {
		t.Errorf("expected Confidence 0.9, got %f", heuristic.Confidence)
	}
}

func TestEngine_SkillStats(t *testing.T) {
	stats := SkillStats{
		Name:          "tdd-workflow",
		UsageCount:    10,
		SuccessCount:  8,
		FailureCount:  2,
		Effectiveness: 0.8,
	}

	if stats.Effectiveness != 0.8 {
		t.Errorf("expected Effectiveness 0.8, got %f", stats.Effectiveness)
	}
}

func TestEngine_TemplateStats(t *testing.T) {
	stats := TemplateStats{
		Name:          "standard",
		UsageCount:    5,
		SuccessCount:  4,
		FailureCount:  1,
		Effectiveness: 0.8,
		Stacks:        []string{"go", "python"},
	}

	if len(stats.Stacks) != 2 {
		t.Errorf("expected 2 stacks, got %d", len(stats.Stacks))
	}
}

func TestEngine_AgentStats(t *testing.T) {
	stats := AgentStats{
		Name:          "orchestrator",
		UsageCount:    10,
		SuccessCount:  8,
		FailureCount:  2,
		Effectiveness: 0.8,
		AvgDuration:   120,
	}

	if stats.AvgDuration != 120 {
		t.Errorf("expected AvgDuration 120, got %d", stats.AvgDuration)
	}
}

func TestEngine_KnowledgeEntry(t *testing.T) {
	entry := KnowledgeEntry{
		ID:         "test-entry",
		Type:       "lesson",
		Title:      "Test Lesson",
		Content:    "Test content",
		Source:     "session-1",
		Confidence: 0.7,
	}

	if entry.Type != "lesson" {
		t.Errorf("expected Type 'lesson', got %q", entry.Type)
	}
}
