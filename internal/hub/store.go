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
