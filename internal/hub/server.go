package hub

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Server struct {
	store *Store
	addr  string
	mux   *http.ServeMux
}

func NewServer(store *Store, addr string) *Server {
	s := &Server{store: store, addr: addr, mux: http.NewServeMux()}
	s.registerRoutes()
	return s
}

func (s *Server) Serve() error {
	fmt.Printf("🚀 ocs hub listening on %s\n", s.addr)
	return http.ListenAndServe(s.addr, s.mux)
}

func (s *Server) registerRoutes() {
	s.mux.HandleFunc("POST /api/auth/keys", s.handleCreateKey)
	s.mux.HandleFunc("GET /api/auth/keys", s.handleListKeys)
	s.mux.HandleFunc("DELETE /api/auth/keys/{id}", s.handleRevokeKey)

	s.mux.HandleFunc("POST /api/config/{type}", s.handleSaveConfig)
	s.mux.HandleFunc("GET /api/config/{type}", s.handleGetConfig)
	s.mux.HandleFunc("GET /api/config/{type}/versions", s.handleListVersions)

	s.mux.HandleFunc("POST /api/backup", s.handleCreateBackup)
	s.mux.HandleFunc("GET /api/backup", s.handleListBackups)
	s.mux.HandleFunc("GET /api/backup/{id}", s.handleGetBackup)

	s.mux.HandleFunc("GET /api/users", s.handleListUsers)
	s.mux.HandleFunc("GET /api/health", s.handleHealth)
}

func (s *Server) requireAuth(next func(http.ResponseWriter, *http.Request, map[string]string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth == "" {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing api key"})
			return
		}

		key := strings.TrimPrefix(auth, "Bearer ")
		user, err := s.store.ValidateAPIKey(key)
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": err.Error()})
			return
		}

		next(w, r, user)
	}
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status":    "ok",
		"version":   "2.0.0",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

func (s *Server) handleListUsers(w http.ResponseWriter, r *http.Request) {
	s.requireAuth(func(w http.ResponseWriter, r *http.Request, user map[string]string) {
		if user["role"] != "admin" {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "admin only"})
			return
		}

		users, err := s.store.ListUsers()
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"users": users})
	})(w, r)
}

func (s *Server) handleCreateKey(w http.ResponseWriter, r *http.Request) {
	s.requireAuth(func(w http.ResponseWriter, r *http.Request, user map[string]string) {
		if user["role"] != "admin" {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "admin only"})
			return
		}

		var req struct {
			UserID    string `json:"user_id"`
			ExpiresAt string `json:"expires_at"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
			return
		}

		rawKey := fmt.Sprintf("ocs-%d-%s", time.Now().UnixNano(), user["user_id"])
		hash := sha256.Sum256([]byte(rawKey))
		hashStr := hex.EncodeToString(hash[:])

		keyID, err := s.store.CreateAPIKey(req.UserID, hashStr, req.ExpiresAt)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}

		s.store.LogAudit(user["user_id"], "create_key", fmt.Sprintf("key_id=%s user_id=%s", keyID, req.UserID))

		writeJSON(w, http.StatusCreated, map[string]string{
			"key_id": keyID,
			"key":    rawKey,
			"note":   "Save this key — it won't be shown again",
		})
	})(w, r)
}

func (s *Server) handleListKeys(w http.ResponseWriter, r *http.Request) {
	s.requireAuth(func(w http.ResponseWriter, r *http.Request, user map[string]string) {
		userID := r.URL.Query().Get("user_id")
		if userID == "" {
			userID = user["user_id"]
		}

		keys, err := s.store.ListAPIKeys(userID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"keys": keys})
	})(w, r)
}

func (s *Server) handleRevokeKey(w http.ResponseWriter, r *http.Request) {
	s.requireAuth(func(w http.ResponseWriter, r *http.Request, user map[string]string) {
		keyID := r.PathValue("id")
		if err := s.store.RevokeAPIKey(keyID); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}

		s.store.LogAudit(user["user_id"], "revoke_key", fmt.Sprintf("key_id=%s", keyID))
		writeJSON(w, http.StatusOK, map[string]string{"status": "revoked"})
	})(w, r)
}

func (s *Server) handleSaveConfig(w http.ResponseWriter, r *http.Request) {
	s.requireAuth(func(w http.ResponseWriter, r *http.Request, user map[string]string) {
		configType := r.PathValue("type")
		projectID := r.URL.Query().Get("project_id")
		message := r.URL.Query().Get("message")

		if projectID == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "project_id required"})
			return
		}

		var content any
		if err := json.NewDecoder(r.Body).Decode(&content); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
			return
		}

		if err := s.store.SaveConfig(projectID, user["user_id"], configType, content, message); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}

		s.store.LogAudit(user["user_id"], "save_config", fmt.Sprintf("project=%s type=%s", projectID, configType))

		contentStr, version, _ := s.store.GetLatestConfig(projectID, configType)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		fmt.Fprintf(w, `{"status":"saved","version":%d,"content":%s}`, version, contentStr)
	})(w, r)
}

func (s *Server) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	s.requireAuth(func(w http.ResponseWriter, r *http.Request, user map[string]string) {
		configType := r.PathValue("type")
		projectID := r.URL.Query().Get("project_id")

		if projectID == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "project_id required"})
			return
		}

		contentStr, version, err := s.store.GetLatestConfig(projectID, configType)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "config not found"})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"type":"%s","version":%d,"content":%s}`, configType, version, contentStr)
	})(w, r)
}

func (s *Server) handleListVersions(w http.ResponseWriter, r *http.Request) {
	s.requireAuth(func(w http.ResponseWriter, r *http.Request, user map[string]string) {
		configType := r.PathValue("type")
		projectID := r.URL.Query().Get("project_id")

		if projectID == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "project_id required"})
			return
		}

		versions, err := s.store.ListConfigVersions(projectID, configType)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"versions": versions})
	})(w, r)
}

func (s *Server) handleCreateBackup(w http.ResponseWriter, r *http.Request) {
	s.requireAuth(func(w http.ResponseWriter, r *http.Request, user map[string]string) {
		var req struct {
			ProjectID string `json:"project_id"`
			Name      string `json:"name"`
			Content   any    `json:"content"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
			return
		}

		if err := s.store.CreateBackup(req.ProjectID, req.Name, user["user_id"], req.Content); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}

		s.store.LogAudit(user["user_id"], "create_backup", fmt.Sprintf("project=%s name=%s", req.ProjectID, req.Name))
		writeJSON(w, http.StatusCreated, map[string]string{"status": "backup created"})
	})(w, r)
}

func (s *Server) handleListBackups(w http.ResponseWriter, r *http.Request) {
	s.requireAuth(func(w http.ResponseWriter, r *http.Request, user map[string]string) {
		projectID := r.URL.Query().Get("project_id")
		if projectID == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "project_id required"})
			return
		}

		backups, err := s.store.ListBackups(projectID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"backups": backups})
	})(w, r)
}

func (s *Server) handleGetBackup(w http.ResponseWriter, r *http.Request) {
	s.requireAuth(func(w http.ResponseWriter, r *http.Request, user map[string]string) {
		id := r.PathValue("id")
		content, err := s.store.GetBackup(id)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "backup not found"})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, content)
	})(w, r)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
