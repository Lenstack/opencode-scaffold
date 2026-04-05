package hub

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Client struct {
	server string
	apiKey string
}

func NewClient(server, apiKey string) *Client {
	return &Client{server: server, apiKey: apiKey}
}

func (c *Client) Health() (map[string]string, error) {
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

func (c *Client) PushKnowledge(push KnowledgePush) error {
	return c.postJSON("/api/knowledge/push", push, nil)
}

func (c *Client) PullKnowledge(stack, workspace string) (*KnowledgePull, error) {
	url := fmt.Sprintf("%s/api/knowledge/pull?stack=%s&workspace=%s",
		c.server, stack, workspace)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	c.setAuth(req)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, c.parseError(resp)
	}

	var pull KnowledgePull
	if err := json.NewDecoder(resp.Body).Decode(&pull); err != nil {
		return nil, err
	}
	return &pull, nil
}

func (c *Client) GetGlobalHeuristics(query GlobalHeuristicQuery) ([]HeuristicRule, error) {
	url := fmt.Sprintf("%s/api/heuristics?stack=%s&workspace=%s&min_conf=%f&limit=%d",
		c.server, query.Stack, query.Workspace, query.MinConf, query.Limit)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	c.setAuth(req)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, c.parseError(resp)
	}

	var result struct {
		Heuristics []HeuristicRule `json:"heuristics"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result.Heuristics, nil
}

func (c *Client) PushSession(outcome SessionOutcome) error {
	return c.postJSON("/api/sessions/sync", outcome, nil)
}

func (c *Client) GetWorkspaceKnowledge(workspace string) ([]KnowledgeEntry, error) {
	url := fmt.Sprintf("%s/api/knowledge/global?workspace=%s", c.server, workspace)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	c.setAuth(req)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, c.parseError(resp)
	}

	var result struct {
		Knowledge []KnowledgeEntry `json:"knowledge"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result.Knowledge, nil
}

func (c *Client) postJSON(path string, body any, result any) error {
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", c.server+path, bytes.NewReader(data))
	if err != nil {
		return err
	}
	c.setAuth(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return c.parseError(resp)
	}

	if result != nil {
		return json.NewDecoder(resp.Body).Decode(result)
	}
	return nil
}

func (c *Client) setAuth(req *http.Request) {
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
}

func (c *Client) parseError(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)
	var errResp struct {
		Error string `json:"error"`
	}
	if json.Unmarshal(body, &errResp) == nil && errResp.Error != "" {
		return fmt.Errorf("hub error (%d): %s", resp.StatusCode, errResp.Error)
	}
	return fmt.Errorf("hub error (%d): %s", resp.StatusCode, string(body))
}
