package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/Lenstack/opencode-scaffold/internal/hub"
)

func newServeCmd() *cobra.Command {
	var (
		port int
		data string
	)

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the hub server for team config sync",
		Long: `Start an HTTP server that stores and distributes OpenCode configuration.

Teams use ocs push/pull to sync agents, skills, commands, plugins, and rules.

Examples:
  ocs serve                          # Start on :8080 with default data dir
  ocs serve --port 9000              # Start on :9000
  ocs serve --data /opt/ocs-data     # Custom data directory
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			addr := fmt.Sprintf(":%d", port)

			if data == "" {
				home, _ := os.UserHomeDir()
				data = filepath.Join(home, ".ocs", "data")
			}

			if err := os.MkdirAll(data, 0755); err != nil {
				return fmt.Errorf("create data dir: %w", err)
			}

			store, err := hub.New(filepath.Join(data, "ocs.db"))
			if err != nil {
				return err
			}
			defer store.Close()

			srv := hub.NewServer(store, addr)
			return srv.Serve()
		},
	}

	cmd.Flags().IntVar(&port, "port", 8080, "HTTP port to listen on")
	cmd.Flags().StringVar(&data, "data", "", "Data directory for SQLite database")

	return cmd
}

func newPushCmd() *cobra.Command {
	var (
		server   string
		apiKey   string
		project  string
		msg      string
		cfgTypes []string
	)

	cmd := &cobra.Command{
		Use:   "push",
		Short: "Push local config to hub server",
		Long: `Push local OpenCode configuration to the hub server.

Examples:
  ocs push --server http://localhost:8080 --key ocs-xxx
  ocs push --types agents,skills,rules
  ocs push --message "Update agent permissions"
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if server == "" || apiKey == "" {
				return fmt.Errorf("--server and --key are required")
			}

			if project == "" {
				project = filepath.Base(mustGetwd())
			}

			if len(cfgTypes) == 0 {
				cfgTypes = []string{"agents", "skills", "commands", "plugins", "rules"}
			}

			root := mustGetwd()
			client := &hubClient{server: server, apiKey: apiKey}

			for _, t := range cfgTypes {
				content, err := readConfigType(root, t)
				if err != nil {
					fmt.Printf("  %s Skipping %s: %v\n", color.YellowString("WARN"), t, err)
					continue
				}

				if err := client.saveConfig(project, t, content, msg); err != nil {
					fmt.Printf("  %s Failed to push %s: %v\n", color.RedString("ERR"), t, err)
					continue
				}

				fmt.Printf("  %s Pushed %s\n", color.GreenString("OK"), t)
			}

			fmt.Println()
			return nil
		},
	}

	cmd.Flags().StringVar(&server, "server", "", "Hub server URL")
	cmd.Flags().StringVar(&apiKey, "key", "", "API key for authentication")
	cmd.Flags().StringVar(&project, "project", "", "Project ID (defaults to directory name)")
	cmd.Flags().StringVar(&msg, "message", "", "Commit message")
	cmd.Flags().StringSliceVar(&cfgTypes, "types", nil, "Config types to push (default: all)")

	return cmd
}

func newPullCmd() *cobra.Command {
	var (
		server   string
		apiKey   string
		project  string
		cfgTypes []string
		force    bool
	)

	cmd := &cobra.Command{
		Use:   "pull",
		Short: "Pull remote config from hub server",
		Long: `Pull OpenCode configuration from the hub server.

Examples:
  ocs pull --server http://localhost:8080 --key ocs-xxx
  ocs pull --types agents,skills
  ocs pull --force               # Overwrite local files
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if server == "" || apiKey == "" {
				return fmt.Errorf("--server and --key are required")
			}

			if project == "" {
				project = filepath.Base(mustGetwd())
			}

			if len(cfgTypes) == 0 {
				cfgTypes = []string{"agents", "skills", "commands", "plugins", "rules"}
			}

			root := mustGetwd()
			client := &hubClient{server: server, apiKey: apiKey}

			for _, t := range cfgTypes {
				content, err := client.getConfig(project, t)
				if err != nil {
					fmt.Printf("  %s %s: %v\n", color.YellowString("WARN"), t, err)
					continue
				}

				if err := writeConfigType(root, t, content, force); err != nil {
					fmt.Printf("  %s Failed to write %s: %v\n", color.RedString("ERR"), t, err)
					continue
				}

				fmt.Printf("  %s Pulled %s\n", color.GreenString("OK"), t)
			}

			fmt.Println()
			return nil
		},
	}

	cmd.Flags().StringVar(&server, "server", "", "Hub server URL")
	cmd.Flags().StringVar(&apiKey, "key", "", "API key for authentication")
	cmd.Flags().StringVar(&project, "project", "", "Project ID (defaults to directory name)")
	cmd.Flags().StringSliceVar(&cfgTypes, "types", nil, "Config types to pull (default: all)")
	cmd.Flags().BoolVar(&force, "force", false, "Overwrite local files")

	return cmd
}

func newAuthCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Manage hub authentication",
	}

	cmd.AddCommand(newAuthCreateKeyCmd())
	cmd.AddCommand(newAuthListKeysCmd())
	cmd.AddCommand(newAuthRevokeKeyCmd())

	return cmd
}

func newAuthCreateKeyCmd() *cobra.Command {
	var (
		server  string
		apiKey  string
		userID  string
		expires string
	)

	cmd := &cobra.Command{
		Use:   "create-key",
		Short: "Create a new API key",
		Long: `Create a new API key for hub authentication.

Examples:
  ocs auth create-key --server http://localhost:8080 --key ocs-admin-xxx --user user-1
  ocs auth create-key --user user-1 --expires 90d
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if server == "" || apiKey == "" {
				return fmt.Errorf("--server and --key are required")
			}
			if userID == "" {
				return fmt.Errorf("--user is required")
			}

			expiresAt := ""
			if expires != "" {
				days := parseDuration(expires)
				expiresAt = time.Now().Add(days).UTC().Format(time.RFC3339)
			}

			client := &hubClient{server: server, apiKey: apiKey}
			result, err := client.createKey(userID, expiresAt)
			if err != nil {
				return err
			}

			fmt.Println()
			bold := color.New(color.Bold)
			bold.Println("  API Key Created:")
			fmt.Println()
			fmt.Printf("  Key ID:    %s\n", result["key_id"])
			fmt.Printf("  Key:       %s\n", color.GreenString(result["key"]))
			fmt.Printf("  Note:      %s\n", color.YellowString(result["note"]))
			fmt.Println()
			return nil
		},
	}

	cmd.Flags().StringVar(&server, "server", "", "Hub server URL")
	cmd.Flags().StringVar(&apiKey, "key", "", "Admin API key")
	cmd.Flags().StringVar(&userID, "user", "", "User ID to create key for")
	cmd.Flags().StringVar(&expires, "expires", "", "Key expiration (e.g. 30d, 90d, 1y)")

	return cmd
}

func newAuthListKeysCmd() *cobra.Command {
	var (
		server string
		apiKey string
		userID string
	)

	cmd := &cobra.Command{
		Use:   "list-keys",
		Short: "List API keys",
		RunE: func(cmd *cobra.Command, args []string) error {
			if server == "" || apiKey == "" {
				return fmt.Errorf("--server and --key are required")
			}

			client := &hubClient{server: server, apiKey: apiKey}
			keys, err := client.listKeys(userID)
			if err != nil {
				return err
			}

			fmt.Println()
			bold := color.New(color.Bold)
			bold.Println("  API Keys:")
			fmt.Println()
			for _, k := range keys {
				expires := k["expires_at"]
				if expires == "" {
					expires = "never"
				}
				fmt.Printf("  %-30s created: %-20s expires: %s\n", k["id"], k["created_at"], expires)
			}
			fmt.Println()
			return nil
		},
	}

	cmd.Flags().StringVar(&server, "server", "", "Hub server URL")
	cmd.Flags().StringVar(&apiKey, "key", "", "API key for authentication")
	cmd.Flags().StringVar(&userID, "user", "", "User ID (defaults to authenticated user)")

	return cmd
}

func newAuthRevokeKeyCmd() *cobra.Command {
	var (
		server string
		apiKey string
	)

	cmd := &cobra.Command{
		Use:   "revoke-key <key-id>",
		Short: "Revoke an API key",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if server == "" || apiKey == "" {
				return fmt.Errorf("--server and --key are required")
			}

			client := &hubClient{server: server, apiKey: apiKey}
			if err := client.revokeKey(args[0]); err != nil {
				return err
			}

			color.Green("Key %s revoked", args[0])
			return nil
		},
	}

	cmd.Flags().StringVar(&server, "server", "", "Hub server URL")
	cmd.Flags().StringVar(&apiKey, "key", "", "API key for authentication")

	return cmd
}

func newBackupCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "backup",
		Short: "Manage hub backups",
	}

	cmd.AddCommand(newBackupCreateCmd())
	cmd.AddCommand(newBackupListCmd())
	cmd.AddCommand(newBackupExportCmd())
	cmd.AddCommand(newBackupImportCmd())

	return cmd
}

func newBackupCreateCmd() *cobra.Command {
	var (
		server  string
		apiKey  string
		project string
		name    string
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a backup of all config",
		RunE: func(cmd *cobra.Command, args []string) error {
			if server == "" || apiKey == "" {
				return fmt.Errorf("--server and --key are required")
			}
			if project == "" {
				project = filepath.Base(mustGetwd())
			}
			if name == "" {
				name = time.Now().Format("2006-01-02-150405")
			}

			root := mustGetwd()
			content := map[string]any{}
			for _, t := range []string{"agents", "skills", "commands", "plugins", "rules"} {
				data, _ := readConfigType(root, t)
				if data != nil {
					content[t] = data
				}
			}

			client := &hubClient{server: server, apiKey: apiKey}
			if err := client.createBackup(project, name, content); err != nil {
				return err
			}

			color.Green("Backup '%s' created for project %s", name, project)
			return nil
		},
	}

	cmd.Flags().StringVar(&server, "server", "", "Hub server URL")
	cmd.Flags().StringVar(&apiKey, "key", "", "API key for authentication")
	cmd.Flags().StringVar(&project, "project", "", "Project ID")
	cmd.Flags().StringVar(&name, "name", "", "Backup name (defaults to timestamp)")

	return cmd
}

func newBackupListCmd() *cobra.Command {
	var (
		server  string
		apiKey  string
		project string
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List backups",
		RunE: func(cmd *cobra.Command, args []string) error {
			if server == "" || apiKey == "" {
				return fmt.Errorf("--server and --key are required")
			}
			if project == "" {
				project = filepath.Base(mustGetwd())
			}

			client := &hubClient{server: server, apiKey: apiKey}
			backups, err := client.listBackups(project)
			if err != nil {
				return err
			}

			fmt.Println()
			bold := color.New(color.Bold)
			bold.Printf("  Backups for %s:\n", project)
			fmt.Println()
			for _, b := range backups {
				fmt.Printf("  %-30s created: %-20s by: %s\n", b["name"], b["created_at"], b["created_by"])
			}
			fmt.Println()
			return nil
		},
	}

	cmd.Flags().StringVar(&server, "server", "", "Hub server URL")
	cmd.Flags().StringVar(&apiKey, "key", "", "API key for authentication")
	cmd.Flags().StringVar(&project, "project", "", "Project ID")

	return cmd
}

func newBackupExportCmd() *cobra.Command {
	var (
		server string
		apiKey string
		backup string
	)

	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export a backup to stdout",
		RunE: func(cmd *cobra.Command, args []string) error {
			if server == "" || apiKey == "" || backup == "" {
				return fmt.Errorf("--server, --key, and --backup are required")
			}

			client := &hubClient{server: server, apiKey: apiKey}
			content, err := client.getBackup(backup)
			if err != nil {
				return err
			}

			fmt.Println(content)
			return nil
		},
	}

	cmd.Flags().StringVar(&server, "server", "", "Hub server URL")
	cmd.Flags().StringVar(&apiKey, "key", "", "API key for authentication")
	cmd.Flags().StringVar(&backup, "backup", "", "Backup ID to export")

	return cmd
}

func newBackupImportCmd() *cobra.Command {
	var (
		server  string
		apiKey  string
		project string
		name    string
	)

	cmd := &cobra.Command{
		Use:   "import",
		Short: "Import a backup from stdin",
		RunE: func(cmd *cobra.Command, args []string) error {
			if server == "" || apiKey == "" {
				return fmt.Errorf("--server and --key are required")
			}
			_ = project
			if project == "" {
				project = filepath.Base(mustGetwd())
			}
			if name == "" {
				name = time.Now().Format("2006-01-02-150405")
			}

			data, err := io.ReadAll(os.Stdin)
			if err != nil {
				return err
			}

			var content any
			if err := json.Unmarshal(data, &content); err != nil {
				return fmt.Errorf("invalid JSON: %w", err)
			}

			client := &hubClient{server: server, apiKey: apiKey}
			return client.createBackup(project, name, content)
		},
	}

	cmd.Flags().StringVar(&server, "server", "", "Hub server URL")
	cmd.Flags().StringVar(&apiKey, "key", "", "API key for authentication")
	cmd.Flags().StringVar(&project, "project", "", "Project ID")
	cmd.Flags().StringVar(&name, "name", "", "Backup name")

	return cmd
}

func newStatusCmd() *cobra.Command {
	var (
		server string
		apiKey string
	)

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Check hub server health",
		RunE: func(cmd *cobra.Command, args []string) error {
			if server == "" {
				return fmt.Errorf("--server is required")
			}

			client := &hubClient{server: server, apiKey: apiKey}
			health, err := client.health()
			if err != nil {
				return fmt.Errorf("server unreachable: %w", err)
			}

			fmt.Println()
			bold := color.New(color.Bold)
			bold.Println("  Hub Status:")
			fmt.Println()
			fmt.Printf("  Status:    %s\n", color.GreenString(health["status"]))
			fmt.Printf("  Version:   %s\n", health["version"])
			fmt.Printf("  Time:      %s\n", health["timestamp"])
			fmt.Println()
			return nil
		},
	}

	cmd.Flags().StringVar(&server, "server", "", "Hub server URL")
	cmd.Flags().StringVar(&apiKey, "key", "", "API key (optional for health check)")

	return cmd
}

// hubClient is a simple HTTP client for the hub server
type hubClient struct {
	server string
	apiKey string
}

func (c *hubClient) health() (map[string]string, error) {
	resp, err := http.Get(c.server + "/api/health")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *hubClient) createKey(userID, expiresAt string) (map[string]string, error) {
	body := map[string]string{"user_id": userID, "expires_at": expiresAt}
	data, _ := json.Marshal(body)

	req, _ := http.NewRequest("POST", c.server+"/api/auth/keys", bytes.NewReader(data))
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *hubClient) listKeys(userID string) ([]map[string]string, error) {
	url := c.server + "/api/auth/keys"
	if userID != "" {
		url += "?user_id=" + userID
	}

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Keys []map[string]string `json:"keys"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result.Keys, nil
}

func (c *hubClient) revokeKey(keyID string) error {
	req, _ := http.NewRequest("DELETE", c.server+"/api/auth/keys/"+keyID, nil)
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp map[string]string
		json.NewDecoder(resp.Body).Decode(&errResp)
		return fmt.Errorf("revoke failed: %s", errResp["error"])
	}
	return nil
}

func (c *hubClient) saveConfig(projectID, configType string, content any, message string) error {
	data, _ := json.Marshal(content)
	url := fmt.Sprintf("%s/api/config/%s?project_id=%s&message=%s",
		c.server, configType, projectID, message)

	req, _ := http.NewRequest("POST", url, bytes.NewReader(data))
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		var errResp map[string]string
		json.NewDecoder(resp.Body).Decode(&errResp)
		return fmt.Errorf("save failed: %s", errResp["error"])
	}
	return nil
}

func (c *hubClient) getConfig(projectID, configType string) (any, error) {
	url := fmt.Sprintf("%s/api/config/%s?project_id=%s",
		c.server, configType, projectID)

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("config not found")
	}

	var result struct {
		Content any `json:"content"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result.Content, nil
}

func (c *hubClient) createBackup(projectID, name string, content any) error {
	body := map[string]any{"project_id": projectID, "name": name, "content": content}
	data, _ := json.Marshal(body)

	req, _ := http.NewRequest("POST", c.server+"/api/backup", bytes.NewReader(data))
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		var errResp map[string]string
		json.NewDecoder(resp.Body).Decode(&errResp)
		return fmt.Errorf("backup failed: %s", errResp["error"])
	}
	return nil
}

func (c *hubClient) listBackups(projectID string) ([]map[string]string, error) {
	url := fmt.Sprintf("%s/api/backup?project_id=%s", c.server, projectID)

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Backups []map[string]string `json:"backups"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result.Backups, nil
}

func (c *hubClient) getBackup(id string) (string, error) {
	req, _ := http.NewRequest("GET", c.server+"/api/backup/"+id, nil)
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// Helper functions

func mustGetwd() string {
	wd, _ := os.Getwd()
	return wd
}

func parseDuration(s string) time.Duration {
	if strings.HasSuffix(s, "d") {
		days := parseInt(strings.TrimSuffix(s, "d"))
		return time.Duration(days) * 24 * time.Hour
	}
	if strings.HasSuffix(s, "y") {
		years := parseInt(strings.TrimSuffix(s, "y"))
		return time.Duration(years) * 365 * 24 * time.Hour
	}
	return 0
}

func parseInt(s string) int {
	var n int
	fmt.Sscanf(s, "%d", &n)
	return n
}

func readConfigType(root, configType string) (any, error) {
	switch configType {
	case "agents":
		return readDirJSON(root, ".opencode/agents")
	case "skills":
		return readDirJSON(root, ".opencode/skills")
	case "commands":
		return readDirJSON(root, ".opencode/commands")
	case "plugins":
		return readDirJSON(root, ".opencode/plugins")
	case "rules":
		return readFile(root, "AGENTS.md")
	default:
		return nil, fmt.Errorf("unknown config type: %s", configType)
	}
}

func writeConfigType(root, configType string, content any, force bool) error {
	switch configType {
	case "agents":
		return writeDirFromJSON(root, ".opencode/agents", content, force)
	case "skills":
		return writeDirFromJSON(root, ".opencode/skills", content, force)
	case "commands":
		return writeDirFromJSON(root, ".opencode/commands", content, force)
	case "plugins":
		return writeDirFromJSON(root, ".opencode/plugins", content, force)
	case "rules":
		return writeFile(root, "AGENTS.md", content, force)
	default:
		return fmt.Errorf("unknown config type: %s", configType)
	}
}

func readDirJSON(root, relDir string) (map[string]string, error) {
	dir := filepath.Join(root, relDir)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	result := map[string]string{}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		content, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err == nil {
			result[e.Name()] = string(content)
		}
	}
	return result, nil
}

func writeDirFromJSON(root, relDir string, content any, force bool) error {
	m, ok := content.(map[string]any)
	if !ok {
		return fmt.Errorf("expected map for directory content")
	}

	dir := filepath.Join(root, relDir)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	for name, val := range m {
		str, ok := val.(string)
		if !ok {
			continue
		}
		path := filepath.Join(dir, name)
		if !force {
			if _, err := os.Stat(path); err == nil {
				continue
			}
		}
		if err := os.WriteFile(path, []byte(str), 0644); err != nil {
			return err
		}
	}
	return nil
}

func readFile(root, name string) (string, error) {
	content, err := os.ReadFile(filepath.Join(root, name))
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func writeFile(root, name string, content any, force bool) error {
	path := filepath.Join(root, name)
	if !force {
		if _, err := os.Stat(path); err == nil {
			return nil
		}
	}

	str, ok := content.(string)
	if !ok {
		return fmt.Errorf("expected string for file content")
	}
	return os.WriteFile(path, []byte(str), 0644)
}
