package learn

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/Lenstack/opencode-scaffold/internal/hub"
)

const (
	NSSessions      = "learn:sessions"
	NSPatterns      = "learn:patterns"
	NSHeuristics    = "learn:heuristics"
	NSSkillStats    = "learn:skill-stats"
	NSTemplateStats = "learn:template-stats"
	NSAgentStats    = "learn:agent-stats"
	NSKnowledge     = "learn:knowledge"
)

type SessionOutcome struct {
	ID        string   `json:"id"`
	SessionID string   `json:"session_id"`
	Outcome   string   `json:"outcome"` // success, failure, partial
	Agents    []string `json:"agents"`
	Skills    []string `json:"skills"`
	Template  string   `json:"template"`
	Stack     string   `json:"stack"`
	Duration  int      `json:"duration"`
	Notes     string   `json:"notes"`
	CreatedAt string   `json:"created_at"`
}

type Pattern struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Category    string  `json:"category"` // workflow, skill, agent, config
	Occurrences int     `json:"occurrences"`
	SuccessRate float64 `json:"success_rate"`
	LastSeen    string  `json:"last_seen"`
	Promoted    bool    `json:"promoted"`
	Confidence  float64 `json:"confidence"`
}

type Heuristic struct {
	ID              string  `json:"id"`
	Rule            string  `json:"rule"`
	Rationale       string  `json:"rationale"`
	Source          string  `json:"source"` // auto-promoted, manual
	Confidence      float64 `json:"confidence"`
	InvocationCount int     `json:"invocation_count"`
	OverrideCount   int     `json:"override_count"`
	SuccessRate     float64 `json:"success_rate"`
	CreatedAt       string  `json:"created_at"`
	UpdatedAt       string  `json:"updated_at"`
}

type SkillStats struct {
	Name          string  `json:"name"`
	UsageCount    int     `json:"usage_count"`
	SuccessCount  int     `json:"success_count"`
	FailureCount  int     `json:"failure_count"`
	Effectiveness float64 `json:"effectiveness"`
	LastUsed      string  `json:"last_used"`
}

type TemplateStats struct {
	Name          string   `json:"name"`
	UsageCount    int      `json:"usage_count"`
	SuccessCount  int      `json:"success_count"`
	FailureCount  int      `json:"failure_count"`
	Effectiveness float64  `json:"effectiveness"`
	Stacks        []string `json:"stacks"`
}

type AgentStats struct {
	Name          string  `json:"name"`
	UsageCount    int     `json:"usage_count"`
	SuccessCount  int     `json:"success_count"`
	FailureCount  int     `json:"failure_count"`
	Effectiveness float64 `json:"effectiveness"`
	AvgDuration   int     `json:"avg_duration"`
}

type KnowledgeEntry struct {
	ID         string  `json:"id"`
	Type       string  `json:"type"` // lesson, pattern, tip, warning
	Title      string  `json:"title"`
	Content    string  `json:"content"`
	Source     string  `json:"source"`
	Confidence float64 `json:"confidence"`
	CreatedAt  string  `json:"created_at"`
}

type Engine struct {
	db *hub.Engine
}

func NewEngine(db *hub.Engine) *Engine {
	return &Engine{db: db}
}

// Session Outcome Tracking

func (e *Engine) RecordSession(outcome SessionOutcome) error {
	if outcome.ID == "" {
		outcome.ID = fmt.Sprintf("session-%d", time.Now().UnixNano())
	}
	outcome.CreatedAt = time.Now().UTC().Format(time.RFC3339)

	if err := e.db.Put(NSSessions, outcome.ID, outcome); err != nil {
		return fmt.Errorf("record session: %w", err)
	}

	// Update skill stats
	for _, skill := range outcome.Skills {
		e.updateSkillStats(skill, outcome.Outcome)
	}

	// Update agent stats
	for _, agent := range outcome.Agents {
		e.updateAgentStats(agent, outcome.Outcome, outcome.Duration)
	}

	// Update template stats
	if outcome.Template != "" {
		e.updateTemplateStats(outcome.Template, outcome.Outcome, outcome.Stack)
	}

	// Detect patterns
	e.detectPatterns(outcome)

	return nil
}

func (e *Engine) GetSessions(limit int) ([]SessionOutcome, error) {
	var sessions []SessionOutcome

	e.db.Iterate(NSSessions, func(key string, value []byte) error {
		var s SessionOutcome
		if err := unmarshalJSON(value, &s); err == nil {
			sessions = append(sessions, s)
		}
		return nil
	})

	// Sort by created_at descending (simple bubble sort for small datasets)
	for i := 0; i < len(sessions); i++ {
		for j := i + 1; j < len(sessions); j++ {
			if sessions[i].CreatedAt < sessions[j].CreatedAt {
				sessions[i], sessions[j] = sessions[j], sessions[i]
			}
		}
	}

	if limit > 0 && len(sessions) > limit {
		sessions = sessions[:limit]
	}

	return sessions, nil
}

func (e *Engine) GetSessionStats() (map[string]int, error) {
	sessions, err := e.GetSessions(0)
	if err != nil {
		return nil, err
	}

	stats := map[string]int{"total": len(sessions), "success": 0, "failure": 0, "partial": 0}
	for _, s := range sessions {
		switch s.Outcome {
		case "success":
			stats["success"]++
		case "failure":
			stats["failure"]++
		case "partial":
			stats["partial"]++
		}
	}

	return stats, nil
}

// Pattern Recognition

func (e *Engine) detectPatterns(outcome SessionOutcome) {
	// Pattern: Successful session with specific agent combo
	key := fmt.Sprintf("agents:%v", outcome.Agents)
	e.incrementPattern(key, "workflow", outcome.Outcome == "success")

	// Pattern: Successful session with specific skill combo
	if len(outcome.Skills) > 0 {
		key := fmt.Sprintf("skills:%v", outcome.Skills)
		e.incrementPattern(key, "skill", outcome.Outcome == "success")
	}

	// Pattern: Successful session with specific template + stack
	if outcome.Template != "" && outcome.Stack != "" {
		key := fmt.Sprintf("template:%s+stack:%s", outcome.Template, outcome.Stack)
		e.incrementPattern(key, "template", outcome.Outcome == "success")
	}
}

func (e *Engine) incrementPattern(key, category string, success bool) {
	var pattern Pattern
	if err := e.db.Get(NSPatterns, key, &pattern); err != nil {
		pattern = Pattern{
			ID:       key,
			Name:     key,
			Category: category,
		}
	}

	pattern.Occurrences++
	if success {
		pattern.SuccessRate = float64(pattern.SuccessRate*float64(pattern.Occurrences-1)+1.0) / float64(pattern.Occurrences)
	} else {
		pattern.SuccessRate = float64(pattern.SuccessRate*float64(pattern.Occurrences-1)) / float64(pattern.Occurrences)
	}
	pattern.LastSeen = time.Now().UTC().Format(time.RFC3339)
	pattern.Confidence = pattern.SuccessRate

	e.db.Put(NSPatterns, key, pattern)
}

func (e *Engine) GetPatterns(category string, minOccurrences int) ([]Pattern, error) {
	var patterns []Pattern

	e.db.Iterate(NSPatterns, func(key string, value []byte) error {
		var p Pattern
		if err := unmarshalJSON(value, &p); err == nil {
			if category == "" || p.Category == category {
				if p.Occurrences >= minOccurrences {
					patterns = append(patterns, p)
				}
			}
		}
		return nil
	})

	// Sort by occurrences descending
	for i := 0; i < len(patterns); i++ {
		for j := i + 1; j < len(patterns); j++ {
			if patterns[i].Occurrences < patterns[j].Occurrences {
				patterns[i], patterns[j] = patterns[j], patterns[i]
			}
		}
	}

	return patterns, nil
}

func (e *Engine) PromotePattern(patternID string) error {
	var pattern Pattern
	if err := e.db.Get(NSPatterns, patternID, &pattern); err != nil {
		return fmt.Errorf("pattern not found: %s", patternID)
	}

	pattern.Promoted = true
	pattern.Confidence = pattern.SuccessRate
	if err := e.db.Put(NSPatterns, patternID, pattern); err != nil {
		return err
	}

	// Create heuristic from pattern
	heuristic := Heuristic{
		ID:          fmt.Sprintf("heuristic-%d", time.Now().UnixNano()),
		Rule:        pattern.ID,
		Rationale:   fmt.Sprintf("Auto-promoted from pattern (occurrences: %d, success rate: %.2f)", pattern.Occurrences, pattern.SuccessRate),
		Source:      "auto-promoted",
		Confidence:  pattern.SuccessRate,
		SuccessRate: pattern.SuccessRate,
		CreatedAt:   time.Now().UTC().Format(time.RFC3339),
		UpdatedAt:   time.Now().UTC().Format(time.RFC3339),
	}

	return e.db.Put(NSHeuristics, heuristic.ID, heuristic)
}

// Heuristic Management

func (e *Engine) GetHeuristics(minConfidence float64) ([]Heuristic, error) {
	var heuristics []Heuristic

	e.db.Iterate(NSHeuristics, func(key string, value []byte) error {
		var h Heuristic
		if err := unmarshalJSON(value, &h); err == nil {
			if h.Confidence >= minConfidence {
				heuristics = append(heuristics, h)
			}
		}
		return nil
	})

	// Sort by confidence descending
	for i := 0; i < len(heuristics); i++ {
		for j := i + 1; j < len(heuristics); j++ {
			if heuristics[i].Confidence < heuristics[j].Confidence {
				heuristics[i], heuristics[j] = heuristics[j], heuristics[i]
			}
		}
	}

	return heuristics, nil
}

func (e *Engine) DemoteHeuristic(id string) error {
	var h Heuristic
	if err := e.db.Get(NSHeuristics, id, &h); err != nil {
		return fmt.Errorf("heuristic not found: %s", id)
	}

	h.Confidence = max(0.0, h.Confidence-0.20)
	h.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

	if h.Confidence < 0.50 {
		// Move to knowledge as warning
		knowledge := KnowledgeEntry{
			ID:         fmt.Sprintf("knowledge-%d", time.Now().UnixNano()),
			Type:       "warning",
			Title:      fmt.Sprintf("Deprecated: %s", h.Rule),
			Content:    h.Rationale,
			Source:     "demoted-heuristic",
			Confidence: h.Confidence,
			CreatedAt:  time.Now().UTC().Format(time.RFC3339),
		}
		e.db.Put(NSKnowledge, knowledge.ID, knowledge)
		return e.db.Delete(NSHeuristics, id)
	}

	return e.db.Put(NSHeuristics, id, h)
}

// Skill Effectiveness

func (e *Engine) updateSkillStats(skillName, outcome string) {
	var stats SkillStats
	if err := e.db.Get(NSSkillStats, skillName, &stats); err != nil {
		stats = SkillStats{Name: skillName}
	}

	stats.UsageCount++
	stats.LastUsed = time.Now().UTC().Format(time.RFC3339)

	if outcome == "success" {
		stats.SuccessCount++
	} else {
		stats.FailureCount++
	}

	if stats.UsageCount > 0 {
		stats.Effectiveness = float64(stats.SuccessCount) / float64(stats.UsageCount)
	}

	e.db.Put(NSSkillStats, skillName, stats)
}

func (e *Engine) GetSkillStats() ([]SkillStats, error) {
	var stats []SkillStats

	e.db.Iterate(NSSkillStats, func(key string, value []byte) error {
		var s SkillStats
		if err := unmarshalJSON(value, &s); err == nil {
			stats = append(stats, s)
		}
		return nil
	})

	// Sort by effectiveness descending
	for i := 0; i < len(stats); i++ {
		for j := i + 1; j < len(stats); j++ {
			if stats[i].Effectiveness < stats[j].Effectiveness {
				stats[i], stats[j] = stats[j], stats[i]
			}
		}
	}

	return stats, nil
}

// Template Effectiveness

func (e *Engine) updateTemplateStats(templateName, outcome, stack string) {
	var stats TemplateStats
	if err := e.db.Get(NSTemplateStats, templateName, &stats); err != nil {
		stats = TemplateStats{Name: templateName}
	}

	stats.UsageCount++
	if outcome == "success" {
		stats.SuccessCount++
	} else {
		stats.FailureCount++
	}

	if stats.UsageCount > 0 {
		stats.Effectiveness = float64(stats.SuccessCount) / float64(stats.UsageCount)
	}

	// Track stacks
	found := false
	for _, s := range stats.Stacks {
		if s == stack {
			found = true
			break
		}
	}
	if !found && stack != "" {
		stats.Stacks = append(stats.Stacks, stack)
	}

	e.db.Put(NSTemplateStats, templateName, stats)
}

func (e *Engine) GetTemplateStats() ([]TemplateStats, error) {
	var stats []TemplateStats

	e.db.Iterate(NSTemplateStats, func(key string, value []byte) error {
		var s TemplateStats
		if err := unmarshalJSON(value, &s); err == nil {
			stats = append(stats, s)
		}
		return nil
	})

	// Sort by effectiveness descending
	for i := 0; i < len(stats); i++ {
		for j := i + 1; j < len(stats); j++ {
			if stats[i].Effectiveness < stats[j].Effectiveness {
				stats[i], stats[j] = stats[j], stats[i]
			}
		}
	}

	return stats, nil
}

// Agent Performance

func (e *Engine) updateAgentStats(agentName, outcome string, duration int) {
	var stats AgentStats
	if err := e.db.Get(NSAgentStats, agentName, &stats); err != nil {
		stats = AgentStats{Name: agentName}
	}

	stats.UsageCount++
	if outcome == "success" {
		stats.SuccessCount++
	} else {
		stats.FailureCount++
	}

	if stats.UsageCount > 0 {
		stats.Effectiveness = float64(stats.SuccessCount) / float64(stats.UsageCount)
	}

	// Running average for duration
	stats.AvgDuration = (stats.AvgDuration*(stats.UsageCount-1) + duration) / stats.UsageCount

	e.db.Put(NSAgentStats, agentName, stats)
}

func (e *Engine) GetAgentStats() ([]AgentStats, error) {
	var stats []AgentStats

	e.db.Iterate(NSAgentStats, func(key string, value []byte) error {
		var s AgentStats
		if err := unmarshalJSON(value, &s); err == nil {
			stats = append(stats, s)
		}
		return nil
	})

	// Sort by effectiveness descending
	for i := 0; i < len(stats); i++ {
		for j := i + 1; j < len(stats); j++ {
			if stats[i].Effectiveness < stats[j].Effectiveness {
				stats[i], stats[j] = stats[j], stats[i]
			}
		}
	}

	return stats, nil
}

// Knowledge Extraction

func (e *Engine) ExtractKnowledge(sinceDays int) ([]KnowledgeEntry, error) {
	var entries []KnowledgeEntry

	cutoff := time.Now().AddDate(0, 0, -sinceDays).UTC().Format(time.RFC3339)

	sessions, _ := e.GetSessions(0)
	for _, s := range sessions {
		if s.CreatedAt < cutoff {
			continue
		}

		if s.Outcome == "failure" && s.Notes != "" {
			entries = append(entries, KnowledgeEntry{
				ID:         fmt.Sprintf("knowledge-%d", time.Now().UnixNano()),
				Type:       "warning",
				Title:      fmt.Sprintf("Session failure: %s", s.SessionID),
				Content:    s.Notes,
				Source:     "session-failure",
				Confidence: 0.5,
				CreatedAt:  s.CreatedAt,
			})
		}

		if s.Outcome == "success" && s.Notes != "" {
			entries = append(entries, KnowledgeEntry{
				ID:         fmt.Sprintf("knowledge-%d", time.Now().UnixNano()),
				Type:       "lesson",
				Title:      fmt.Sprintf("Session success: %s", s.SessionID),
				Content:    s.Notes,
				Source:     "session-success",
				Confidence: 0.7,
				CreatedAt:  s.CreatedAt,
			})
		}
	}

	return entries, nil
}

func (e *Engine) GetKnowledge(entryType string) ([]KnowledgeEntry, error) {
	var entries []KnowledgeEntry

	e.db.Iterate(NSKnowledge, func(key string, value []byte) error {
		var k KnowledgeEntry
		if err := unmarshalJSON(value, &k); err == nil {
			if entryType == "" || k.Type == entryType {
				entries = append(entries, k)
			}
		}
		return nil
	})

	return entries, nil
}

// Auto-Learning Cycle

func (e *Engine) RunLearningCycle() (map[string]int, error) {
	results := map[string]int{
		"patterns_promoted":   0,
		"heuristics_demoted":  0,
		"knowledge_extracted": 0,
	}

	// 1. Auto-promote patterns
	patterns, _ := e.GetPatterns("", 3)
	for _, p := range patterns {
		if !p.Promoted && p.SuccessRate >= 0.70 {
			if err := e.PromotePattern(p.ID); err == nil {
				results["patterns_promoted"]++
			}
		}
	}

	// 2. Demote low-confidence heuristics
	heuristics, _ := e.GetHeuristics(0)
	for _, h := range heuristics {
		if h.Confidence < 0.50 {
			if err := e.DemoteHeuristic(h.ID); err == nil {
				results["heuristics_demoted"]++
			}
		}
	}

	// 3. Extract knowledge from recent sessions
	entries, _ := e.ExtractKnowledge(7)
	for _, entry := range entries {
		e.db.Put(NSKnowledge, entry.ID, entry)
		results["knowledge_extracted"]++
	}

	return results, nil
}

func unmarshalJSON(data []byte, v any) error {
	return json.Unmarshal(data, v)
}

func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
