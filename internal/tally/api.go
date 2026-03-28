package tally

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// Client is an HTTP client for the Tally REST API.
type Client struct {
	BaseURL    string
	Token      string
	HTTPClient *http.Client
}

// NewClient creates a new Tally API client.
func NewClient(baseURL, token string) *Client {
	return &Client{
		BaseURL:    baseURL,
		Token:      token,
		HTTPClient: http.DefaultClient,
	}
}

// CreateForm creates a new form via POST /forms.
func (c *Client) CreateForm(req *CreateFormRequest) (*TallyForm, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	resp, err := c.do("POST", "/forms", body)
	if err != nil {
		return nil, err
	}

	var form TallyForm
	if err := json.Unmarshal(resp, &form); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}
	return &form, nil
}

// UpdateForm updates an existing form via PATCH /forms/{id}.
func (c *Client) UpdateForm(formID string, req *UpdateFormRequest) (*TallyForm, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	resp, err := c.do("PATCH", "/forms/"+formID, body)
	if err != nil {
		return nil, err
	}

	var form TallyForm
	if err := json.Unmarshal(resp, &form); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}
	return &form, nil
}

// GetForm retrieves a form via GET /forms/{id}.
func (c *Client) GetForm(formID string) (*TallyForm, error) {
	resp, err := c.do("GET", "/forms/"+formID, nil)
	if err != nil {
		return nil, err
	}

	var form TallyForm
	if err := json.Unmarshal(resp, &form); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}
	return &form, nil
}

// GetSubmissions retrieves all submissions for a form.
func (c *Client) GetSubmissions(formID string) (*SubmissionsResponse, error) {
	resp, err := c.do("GET", "/forms/"+formID+"/submissions?limit=500&status=FINISHED", nil)
	if err != nil {
		return nil, err
	}

	var subs SubmissionsResponse
	if err := json.Unmarshal(resp, &subs); err != nil {
		return nil, fmt.Errorf("unmarshal submissions: %w", err)
	}
	return &subs, nil
}

func (c *Client) do(method, path string, body []byte) ([]byte, error) {
	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequest(method, c.BaseURL+path, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.Token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}
