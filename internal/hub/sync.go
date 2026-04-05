package hub

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"time"
)

type SyncEngine struct {
	db        *Engine
	projectID string
	workspace string
	stack     string
}

func NewSyncEngine(db *Engine, projectID, workspace, stack string) *SyncEngine {
	return &SyncEngine{
		db:        db,
		projectID: projectID,
		workspace: workspace,
		stack:     stack,
	}
}

func (s *SyncEngine) Push(client *Client) (int, error) {
	semantics, err := s.collectPushableSemantics()
	if err != nil {
		return 0, fmt.Errorf("collect semantics: %w", err)
	}

	heuristics, err := s.collectPushableHeuristics()
	if err != nil {
		return 0, fmt.Errorf("collect heuristics: %w", err)
	}

	sessions, err := s.collectPushableSessions()
	if err != nil {
		return 0, fmt.Errorf("collect sessions: %w", err)
	}

	if len(semantics) == 0 && len(heuristics) == 0 && len(sessions) == 0 {
		return 0, nil
	}

	push := KnowledgePush{
		ProjectID:  s.projectID,
		Workspace:  s.workspace,
		Stack:      s.stack,
		Semantic:   semantics,
		Heuristics: heuristics,
		Sessions:   sessions,
		PushedAt:   time.Now().UTC().Format(time.RFC3339),
	}

	if err := client.PushKnowledge(push); err != nil {
		return 0, fmt.Errorf("push to hub: %w", err)
	}

	pushed := len(semantics) + len(heuristics) + len(sessions)
	s.updateSyncStatus("push", pushed)

	return pushed, nil
}

func (s *SyncEngine) Pull(client *Client) (int, error) {
	pulled, err := client.PullKnowledge(s.stack, s.workspace)
	if err != nil {
		return 0, fmt.Errorf("pull from hub: %w", err)
	}

	merged := 0

	for _, sem := range pulled.Semantic {
		if err := s.mergeSemantic(sem); err == nil {
			merged++
		}
	}

	for _, h := range pulled.Heuristics {
		if err := s.mergeHeuristic(h); err == nil {
			merged++
		}
	}

	for _, k := range pulled.Knowledge {
		if err := s.mergeKnowledge(k); err == nil {
			merged++
		}
	}

	s.updateSyncStatus("pull", merged)
	return merged, nil
}

func (s *SyncEngine) AutoSync(client *Client) (map[string]int, error) {
	results := map[string]int{"pushed": 0, "pulled": 0}

	pulled, err := s.Pull(client)
	if err != nil {
		return results, fmt.Errorf("pull failed: %w", err)
	}
	results["pulled"] = pulled

	pushed, err := s.Push(client)
	if err != nil {
		return results, fmt.Errorf("push failed: %w", err)
	}
	results["pushed"] = pushed

	return results, nil
}

func (s *SyncEngine) Status() (*SyncStatus, error) {
	var status SyncStatus
	if err := s.db.Get("hub:sync", "status", &status); err != nil {
		return &SyncStatus{Status: "never_synced"}, nil
	}
	return &status, nil
}

func (s *SyncEngine) collectPushableSemantics() ([]SemanticMemory, error) {
	var semantics []SemanticMemory
	s.db.Iterate(NSMemorySemantic, func(key string, value []byte) error {
		var mem SemanticMemory
		if err := json.Unmarshal(value, &mem); err == nil {
			if mem.Confidence >= 0.70 {
				semantics = append(semantics, mem)
			}
		}
		return nil
	})
	return semantics, nil
}

func (s *SyncEngine) collectPushableHeuristics() ([]HeuristicRule, error) {
	var heuristics []HeuristicRule
	s.db.Iterate(NSMemoryHeuristic, func(key string, value []byte) error {
		var rule HeuristicRule
		if err := json.Unmarshal(value, &rule); err == nil {
			if rule.Active && rule.Confidence >= 0.60 {
				heuristics = append(heuristics, rule)
			}
		}
		return nil
	})
	return heuristics, nil
}

func (s *SyncEngine) collectPushableSessions() ([]SessionOutcome, error) {
	var outcomes []SessionOutcome
	s.db.Iterate("learn:sessions", func(key string, value []byte) error {
		var sess SessionOutcome
		if err := json.Unmarshal(value, &sess); err == nil {
			outcomes = append(outcomes, sess)
		}
		return nil
	})
	return outcomes, nil
}

func (s *SyncEngine) mergeSemantic(mem SemanticMemory) error {
	var existing SemanticMemory
	key := fmt.Sprintf("%016x", hashKey(mem.FactKey))
	if err := s.db.Get(NSMemorySemantic, key, &existing); err == nil {
		if existing.Confidence >= mem.Confidence {
			return nil
		}
	}

	mem.TS = time.Now().UTC().Format(time.RFC3339)
	mem.Source = "hub:" + mem.Source
	return s.db.Put(NSMemorySemantic, key, mem)
}

func (s *SyncEngine) mergeHeuristic(rule HeuristicRule) error {
	exists := false
	s.db.Iterate(NSMemoryHeuristic, func(key string, value []byte) error {
		var r HeuristicRule
		if err := json.Unmarshal(value, &r); err == nil && r.Rule == rule.Rule {
			exists = true
		}
		return nil
	})

	if exists {
		return nil
	}

	rule.SourceSessions = append(rule.SourceSessions, "hub-import")
	return s.db.Put(NSMemoryHeuristic, rule.ID, rule)
}

func (s *SyncEngine) mergeKnowledge(entry KnowledgeEntry) error {
	id := fmt.Sprintf("knowledge-%d", time.Now().UnixNano())
	return s.db.Put("learn:knowledge", id, entry)
}

func (s *SyncEngine) updateSyncStatus(action string, count int) error {
	var status SyncStatus
	s.db.Get("hub:sync", "status", &status)

	now := time.Now().UTC().Format(time.RFC3339)
	if action == "push" {
		status.LastPush = now
		status.PushedItems += count
	} else {
		status.LastPull = now
		status.PulledItems += count
	}
	status.Status = "synced"

	return s.db.Put("hub:sync", "status", status)
}

func hashKey(s string) uint64 {
	h := fnv.New64a()
	h.Write([]byte(s))
	return h.Sum64()
}
