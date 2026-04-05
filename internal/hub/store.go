package hub

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
}

func New(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path+"?_journal_mode=WAL&_busy_timeout=5000&_cache_size=-64000")
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	db.SetMaxOpenConns(4)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(5 * time.Minute)

	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			email TEXT UNIQUE,
			role TEXT DEFAULT 'developer',
			created_at TEXT
		);

		CREATE TABLE IF NOT EXISTS api_keys (
			id TEXT PRIMARY KEY,
			user_id TEXT,
			key_hash TEXT UNIQUE,
			created_at TEXT,
			expires_at TEXT,
			FOREIGN KEY (user_id) REFERENCES users(id)
		);

		CREATE TABLE IF NOT EXISTS config_snapshots (
			id TEXT PRIMARY KEY,
			project_id TEXT,
			user_id TEXT,
			type TEXT,
			content TEXT,
			version INTEGER DEFAULT 1,
			created_at TEXT,
			message TEXT
		);

		CREATE TABLE IF NOT EXISTS sync_state (
			project_id TEXT,
			user_id TEXT,
			last_sync TEXT,
			local_version INTEGER DEFAULT 0,
			remote_version INTEGER DEFAULT 0,
			PRIMARY KEY (project_id, user_id)
		);

		CREATE TABLE IF NOT EXISTS backups (
			id TEXT PRIMARY KEY,
			project_id TEXT,
			name TEXT,
			content TEXT,
			created_at TEXT,
			created_by TEXT
		);

		CREATE TABLE IF NOT EXISTS audit_log (
			id TEXT PRIMARY KEY,
			user_id TEXT,
			action TEXT,
			detail TEXT,
			created_at TEXT
		);

		CREATE TABLE IF NOT EXISTS knowledge_entries (
			id TEXT PRIMARY KEY,
			project_id TEXT,
			workspace TEXT,
			stack TEXT,
			type TEXT,
			title TEXT,
			content TEXT,
			source TEXT,
			confidence REAL DEFAULT 0.5,
			created_at TEXT
		);

		CREATE TABLE IF NOT EXISTS heuristics_global (
			id TEXT PRIMARY KEY,
			project_id TEXT,
			workspace TEXT,
			stack TEXT,
			rule TEXT,
			rationale TEXT,
			confidence REAL DEFAULT 0.5,
			invocation_count INTEGER DEFAULT 0,
			override_count INTEGER DEFAULT 0,
			active INTEGER DEFAULT 1,
			created_at TEXT,
			updated_at TEXT
		);

		CREATE TABLE IF NOT EXISTS patterns_global (
			id TEXT PRIMARY KEY,
			project_id TEXT,
			workspace TEXT,
			stack TEXT,
			category TEXT,
			name TEXT,
			description TEXT,
			occurrences INTEGER DEFAULT 1,
			success_rate REAL DEFAULT 0.5,
			confidence REAL DEFAULT 0.5,
			promoted INTEGER DEFAULT 0,
			created_at TEXT,
			updated_at TEXT
		);

		CREATE TABLE IF NOT EXISTS project_registry (
			id TEXT PRIMARY KEY,
			name TEXT,
			workspace TEXT,
			stack TEXT,
			created_at TEXT,
			last_sync TEXT
		);

		CREATE TABLE IF NOT EXISTS session_outcomes (
			id TEXT PRIMARY KEY,
			project_id TEXT,
			workspace TEXT,
			stack TEXT,
			session_id TEXT,
			outcome TEXT,
			agents TEXT,
			skills TEXT,
			template TEXT,
			duration INTEGER,
			notes TEXT,
			created_at TEXT
		);
	`); err != nil {
		return nil, fmt.Errorf("init schema: %w", err)
	}

	return &Store{db: db}, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) CreateUser(id, email, role string) error {
	_, err := s.db.Exec(
		"INSERT OR IGNORE INTO users (id, email, role, created_at) VALUES (?, ?, ?, ?)",
		id, email, role, time.Now().UTC().Format(time.RFC3339),
	)
	return err
}

func (s *Store) GetUser(id string) (map[string]string, error) {
	row := s.db.QueryRow("SELECT id, email, role, created_at FROM users WHERE id = ?", id)
	var uid, email, role, created string
	if err := row.Scan(&uid, &email, &role, &created); err != nil {
		return nil, err
	}
	return map[string]string{
		"id":         uid,
		"email":      email,
		"role":       role,
		"created_at": created,
	}, nil
}

func (s *Store) ListUsers() ([]map[string]string, error) {
	rows, err := s.db.Query("SELECT id, email, role, created_at FROM users ORDER BY created_at")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []map[string]string
	for rows.Next() {
		var id, email, role, created string
		if err := rows.Scan(&id, &email, &role, &created); err != nil {
			return nil, err
		}
		users = append(users, map[string]string{
			"id": id, "email": email, "role": role, "created_at": created,
		})
	}
	return users, nil
}

func (s *Store) CreateAPIKey(userID, keyHash, expiresAt string) (string, error) {
	id := fmt.Sprintf("key-%d", time.Now().UnixNano())
	_, err := s.db.Exec(
		"INSERT INTO api_keys (id, user_id, key_hash, created_at, expires_at) VALUES (?, ?, ?, ?, ?)",
		id, userID, keyHash, time.Now().UTC().Format(time.RFC3339), expiresAt,
	)
	return id, err
}

func (s *Store) ValidateAPIKey(key string) (map[string]string, error) {
	hash := sha256.Sum256([]byte(key))
	hashStr := hex.EncodeToString(hash[:])

	row := s.db.QueryRow(`
		SELECT u.id, u.email, u.role, COALESCE(k.expires_at, '')
		FROM api_keys k
		JOIN users u ON k.user_id = u.id
		WHERE k.key_hash = ?
	`, hashStr)

	var uid, email, role, expires string
	if err := row.Scan(&uid, &email, &role, &expires); err != nil {
		return nil, fmt.Errorf("invalid api key")
	}

	if expires != "" {
		expTime, err := time.Parse(time.RFC3339, expires)
		if err == nil && time.Now().After(expTime) {
			return nil, fmt.Errorf("api key expired")
		}
	}

	return map[string]string{
		"user_id":    uid,
		"email":      email,
		"role":       role,
		"expires_at": expires,
	}, nil
}

func (s *Store) ListAPIKeys(userID string) ([]map[string]string, error) {
	rows, err := s.db.Query(
		"SELECT id, created_at, expires_at FROM api_keys WHERE user_id = ? ORDER BY created_at DESC",
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []map[string]string
	for rows.Next() {
		var id, created, expires string
		if err := rows.Scan(&id, &created, &expires); err != nil {
			return nil, err
		}
		keys = append(keys, map[string]string{
			"id": id, "created_at": created, "expires_at": expires,
		})
	}
	return keys, nil
}

func (s *Store) RevokeAPIKey(keyID string) error {
	_, err := s.db.Exec("DELETE FROM api_keys WHERE id = ?", keyID)
	return err
}

func (s *Store) SaveConfig(projectID, userID, configType string, content any, message string) error {
	data, _ := json.Marshal(content)
	id := fmt.Sprintf("cfg-%d", time.Now().UnixNano())

	var version int
	row := s.db.QueryRow(
		"SELECT COALESCE(MAX(version), 0) FROM config_snapshots WHERE project_id = ? AND type = ?",
		projectID, configType,
	)
	row.Scan(&version)
	version++

	_, err := s.db.Exec(
		"INSERT INTO config_snapshots (id, project_id, user_id, type, content, version, created_at, message) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		id, projectID, userID, configType, string(data), version, time.Now().UTC().Format(time.RFC3339), message,
	)
	return err
}

func (s *Store) GetLatestConfig(projectID, configType string) (string, int, error) {
	row := s.db.QueryRow(`
		SELECT content, version FROM config_snapshots
		WHERE project_id = ? AND type = ?
		ORDER BY version DESC LIMIT 1
	`, projectID, configType)

	var content string
	var version int
	if err := row.Scan(&content, &version); err != nil {
		return "", 0, err
	}
	return content, version, nil
}

func (s *Store) ListConfigVersions(projectID, configType string) ([]map[string]any, error) {
	rows, err := s.db.Query(`
		SELECT id, version, created_at, message FROM config_snapshots
		WHERE project_id = ? AND type = ?
		ORDER BY version DESC
	`, projectID, configType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var versions []map[string]any
	for rows.Next() {
		var id string
		var version int
		var created, message string
		if err := rows.Scan(&id, &version, &created, &message); err != nil {
			return nil, err
		}
		versions = append(versions, map[string]any{
			"id": id, "version": version, "created_at": created, "message": message,
		})
	}
	return versions, nil
}

func (s *Store) CreateBackup(projectID, name, createdBy string, content any) error {
	id := fmt.Sprintf("bkp-%d", time.Now().UnixNano())
	data, _ := json.Marshal(content)
	_, err := s.db.Exec(
		"INSERT INTO backups (id, project_id, name, content, created_at, created_by) VALUES (?, ?, ?, ?, ?, ?)",
		id, projectID, name, string(data), time.Now().UTC().Format(time.RFC3339), createdBy,
	)
	return err
}

func (s *Store) ListBackups(projectID string) ([]map[string]string, error) {
	rows, err := s.db.Query(
		"SELECT id, name, created_at, created_by FROM backups WHERE project_id = ? ORDER BY created_at DESC",
		projectID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var backups []map[string]string
	for rows.Next() {
		var id, name, created, createdBy string
		if err := rows.Scan(&id, &name, &created, &createdBy); err != nil {
			return nil, err
		}
		backups = append(backups, map[string]string{
			"id": id, "name": name, "created_at": created, "created_by": createdBy,
		})
	}
	return backups, nil
}

func (s *Store) GetBackup(id string) (string, error) {
	row := s.db.QueryRow("SELECT content FROM backups WHERE id = ?", id)
	var content string
	if err := row.Scan(&content); err != nil {
		return "", err
	}
	return content, nil
}

func (s *Store) LogAudit(userID, action, detail string) error {
	id := fmt.Sprintf("audit-%d", time.Now().UnixNano())
	_, err := s.db.Exec(
		"INSERT INTO audit_log (id, user_id, action, detail, created_at) VALUES (?, ?, ?, ?, ?)",
		id, userID, action, detail, time.Now().UTC().Format(time.RFC3339),
	)
	return err
}

func (s *Store) RegisterProject(id, name, workspace, stack string) error {
	_, err := s.db.Exec(
		"INSERT OR REPLACE INTO project_registry (id, name, workspace, stack, created_at) VALUES (?, ?, ?, ?, ?)",
		id, name, workspace, stack, time.Now().UTC().Format(time.RFC3339),
	)
	return err
}

func (s *Store) PushKnowledge(push KnowledgePush) error {
	now := time.Now().UTC().Format(time.RFC3339)

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, sem := range push.Semantic {
		id := fmt.Sprintf("sem-%d-%s", time.Now().UnixNano(), sem.FactKey[:minInt(8, len(sem.FactKey))])
		tx.Exec(
			"INSERT OR REPLACE INTO knowledge_entries (id, project_id, workspace, stack, type, title, content, source, confidence, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
			id, push.ProjectID, push.Workspace, push.Stack, "semantic", sem.FactKey, sem.Fact, sem.Source, sem.Confidence, now,
		)
	}

	for _, h := range push.Heuristics {
		id := fmt.Sprintf("heur-%d-%s", time.Now().UnixNano(), h.ID)
		active := 0
		if h.Active {
			active = 1
		}
		tx.Exec(
			"INSERT OR REPLACE INTO heuristics_global (id, project_id, workspace, stack, rule, rationale, confidence, invocation_count, override_count, active, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
			id, push.ProjectID, push.Workspace, push.Stack, h.Rule, h.Rationale, h.Confidence, h.InvocationCount, h.OverrideCount, active, now, now,
		)
	}

	for _, sess := range push.Sessions {
		id := fmt.Sprintf("sess-%d-%s", time.Now().UnixNano(), sess.SessionID)
		agents, _ := json.Marshal(sess.Agents)
		skills, _ := json.Marshal(sess.Skills)
		tx.Exec(
			"INSERT OR REPLACE INTO session_outcomes (id, project_id, workspace, stack, session_id, outcome, agents, skills, template, duration, notes, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
			id, push.ProjectID, push.Workspace, push.Stack, sess.SessionID, sess.Outcome, string(agents), string(skills), sess.Template, sess.Duration, sess.Notes, now,
		)
	}

	return tx.Commit()
}

func (s *Store) PullKnowledge(stack, workspace string) (*KnowledgePull, error) {
	pull := &KnowledgePull{
		Stack:     stack,
		Workspace: workspace,
		PulledAt:  time.Now().UTC().Format(time.RFC3339),
	}

	rows, err := s.db.Query(
		"SELECT type, title, content, source, confidence, created_at FROM knowledge_entries WHERE workspace = ? AND (stack = ? OR stack = '') AND confidence >= 0.60 ORDER BY confidence DESC LIMIT 50",
		workspace, stack,
	)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var k KnowledgeEntry
			if err := rows.Scan(&k.Type, &k.Title, &k.Content, &k.Source, &k.Confidence, &k.CreatedAt); err == nil {
				pull.Semantic = append(pull.Semantic, SemanticMemory{
					Fact:       k.Content,
					FactKey:    k.Title,
					Confidence: k.Confidence,
					Source:     k.Source,
					Category:   k.Type,
				})
			}
		}
	}

	hRows, err := s.db.Query(
		"SELECT rule, rationale, confidence, invocation_count, override_count, active FROM heuristics_global WHERE workspace = ? AND (stack = ? OR stack = '') AND active = 1 AND confidence >= 0.50 ORDER BY confidence DESC LIMIT 30",
		workspace, stack,
	)
	if err == nil {
		defer hRows.Close()
		for hRows.Next() {
			var h HeuristicRule
			var active int
			if err := hRows.Scan(&h.Rule, &h.Rationale, &h.Confidence, &h.InvocationCount, &h.OverrideCount, &active); err == nil {
				h.Active = active == 1
				h.ID = fmt.Sprintf("global-%d", time.Now().UnixNano())
				pull.Heuristics = append(pull.Heuristics, h)
			}
		}
	}

	kRows, err := s.db.Query(
		"SELECT id, type, title, content, source, stack, confidence, created_at FROM knowledge_entries WHERE workspace = ? AND type IN ('lesson', 'pattern', 'warning') ORDER BY confidence DESC LIMIT 30",
		workspace,
	)
	if err == nil {
		defer kRows.Close()
		for kRows.Next() {
			var k KnowledgeEntry
			if err := kRows.Scan(&k.ID, &k.Type, &k.Title, &k.Content, &k.Source, &k.Stack, &k.Confidence, &k.CreatedAt); err == nil {
				pull.Knowledge = append(pull.Knowledge, k)
			}
		}
	}

	return pull, nil
}

func (s *Store) GetGlobalHeuristics(stack, workspace string, minConf float64, limit int) ([]HeuristicRule, error) {
	if limit == 0 {
		limit = 50
	}

	rows, err := s.db.Query(
		"SELECT id, rule, rationale, confidence, invocation_count, override_count, active FROM heuristics_global WHERE workspace = ? AND (stack = ? OR stack = '') AND confidence >= ? AND active = 1 ORDER BY confidence DESC LIMIT ?",
		workspace, stack, minConf, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var heuristics []HeuristicRule
	for rows.Next() {
		var h HeuristicRule
		var active int
		if err := rows.Scan(&h.ID, &h.Rule, &h.Rationale, &h.Confidence, &h.InvocationCount, &h.OverrideCount, &active); err == nil {
			h.Active = active == 1
			heuristics = append(heuristics, h)
		}
	}
	return heuristics, nil
}

func (s *Store) PushSession(outcome SessionOutcome) error {
	id := fmt.Sprintf("sess-%d-%s", time.Now().UnixNano(), outcome.SessionID)
	agents, _ := json.Marshal(outcome.Agents)
	skills, _ := json.Marshal(outcome.Skills)
	_, err := s.db.Exec(
		"INSERT OR REPLACE INTO session_outcomes (id, project_id, workspace, stack, session_id, outcome, agents, skills, template, duration, notes, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		id, outcome.ID, outcome.Workspace, outcome.Stack, outcome.SessionID, outcome.Outcome, string(agents), string(skills), outcome.Template, outcome.Duration, outcome.Notes, outcome.CreatedAt,
	)
	return err
}

func (s *Store) GetWorkspaceKnowledge(workspace string) ([]KnowledgeEntry, error) {
	rows, err := s.db.Query(
		"SELECT id, type, title, content, source, stack, confidence, created_at FROM knowledge_entries WHERE workspace = ? ORDER BY confidence DESC LIMIT 50",
		workspace,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []KnowledgeEntry
	for rows.Next() {
		var k KnowledgeEntry
		if err := rows.Scan(&k.ID, &k.Type, &k.Title, &k.Content, &k.Source, &k.Stack, &k.Confidence, &k.CreatedAt); err == nil {
			entries = append(entries, k)
		}
	}
	return entries, nil
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
