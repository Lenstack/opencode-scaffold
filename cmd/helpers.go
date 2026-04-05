package cmd

import (
	"encoding/json"
	"fmt"
	"bytes"
	"net/http"
	"os"
	"path/filepath"

	"github.com/Lenstack/opencode-scaffold/internal/hub"
)

func mustGetwd() string {
	wd, _ := os.Getwd()
	return wd
}

func openDB() (*hub.Store, error) {
	root := mustGetwd()
	return hub.New(filepath.Join(root, ".opencode", "data"))
}

func openEngine() (*hub.Engine, error) {
	root := mustGetwd()
	return hub.NewEngine(filepath.Join(root, ".opencode", "data"))
}

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

	if resp.StatusCode != 201 {
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

	if resp.StatusCode == 404 {
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
