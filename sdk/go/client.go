package waid

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// Client is the WAID API client.
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// Option is a functional option for configuring a Client.
type Option func(*Client)

// WithAPIKey sets the X-API-Key header on all requests.
func WithAPIKey(key string) Option {
	return func(c *Client) {
		c.apiKey = key
	}
}

// WithHTTPClient replaces the default HTTP client.
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) {
		c.httpClient = hc
	}
}

// WithTimeout sets a timeout on the default HTTP client.
func WithTimeout(d time.Duration) Option {
	return func(c *Client) {
		c.httpClient.Timeout = d
	}
}

// NewClient creates a new WAID API client.
func NewClient(baseURL string, opts ...Option) *Client {
	c := &Client{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
	for _, o := range opts {
		o(c)
	}
	return c
}

// do performs an HTTP request, marshalling body to JSON and unmarshalling the response.
// Pass nil for body when there is no request body. Pass nil for out when no response body is expected.
func (c *Client) do(ctx context.Context, method, path string, body, out interface{}) error {
	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("waid: marshal request: %w", err)
		}
		reqBody = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reqBody)
	if err != nil {
		return fmt.Errorf("waid: build request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	if c.apiKey != "" {
		req.Header.Set("X-API-Key", c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("waid: http: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("waid: read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		var errResp struct {
			Error string `json:"error"`
		}
		msg := http.StatusText(resp.StatusCode)
		if json.Unmarshal(respBody, &errResp) == nil && errResp.Error != "" {
			msg = errResp.Error
		}
		return &WAIDError{StatusCode: resp.StatusCode, Message: msg, Body: respBody}
	}

	if out != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, out); err != nil {
			return fmt.Errorf("waid: unmarshal response: %w", err)
		}
	}
	return nil
}

// Resolve looks up a contact by phone number or BSUID.
func (c *Client) Resolve(ctx context.Context, phoneOrID string) (*IdentityResult, error) {
	var result IdentityResult
	if err := c.do(ctx, http.MethodGet, "/resolve/"+url.PathEscape(phoneOrID), nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// CreateContact creates a new contact.
func (c *Client) CreateContact(ctx context.Context, input CreateContactInput) (*Contact, error) {
	var contact Contact
	if err := c.do(ctx, http.MethodPost, "/contacts", input, &contact); err != nil {
		return nil, err
	}
	return &contact, nil
}

// GetContact retrieves a contact by UUID.
func (c *Client) GetContact(ctx context.Context, id string) (*Contact, error) {
	var contact Contact
	if err := c.do(ctx, http.MethodGet, "/contacts/"+url.PathEscape(id), nil, &contact); err != nil {
		return nil, err
	}
	return &contact, nil
}

// UpdateContact partially updates a contact by UUID.
func (c *Client) UpdateContact(ctx context.Context, id string, input UpdateContactInput) (*Contact, error) {
	var contact Contact
	if err := c.do(ctx, http.MethodPut, "/contacts/"+url.PathEscape(id), input, &contact); err != nil {
		return nil, err
	}
	return &contact, nil
}

// DeleteContact soft-deletes a contact by UUID.
func (c *Client) DeleteContact(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodDelete, "/contacts/"+url.PathEscape(id), nil, nil)
}

// ListContacts returns a paginated list of contacts.
func (c *Client) ListContacts(ctx context.Context, opts ListOpts) (*PaginatedContacts, error) {
	q := url.Values{}
	if opts.Page > 0 {
		q.Set("page", strconv.Itoa(opts.Page))
	}
	if opts.PerPage > 0 {
		q.Set("per_page", strconv.Itoa(opts.PerPage))
	}
	if opts.Query != "" {
		q.Set("q", opts.Query)
	}
	path := "/contacts"
	if len(q) > 0 {
		path += "?" + q.Encode()
	}
	var result PaginatedContacts
	if err := c.do(ctx, http.MethodGet, path, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ImportContacts uploads a CSV or JSON file for bulk upsert.
func (c *Client) ImportContacts(ctx context.Context, filename string, fileData io.Reader) (*ImportReport, error) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	fw, err := w.CreateFormFile("file", filename)
	if err != nil {
		return nil, fmt.Errorf("waid: create form file: %w", err)
	}
	if _, err := io.Copy(fw, fileData); err != nil {
		return nil, fmt.Errorf("waid: write form file: %w", err)
	}
	w.Close()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/import", &buf)
	if err != nil {
		return nil, fmt.Errorf("waid: build request: %w", err)
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Accept", "application/json")
	if c.apiKey != "" {
		req.Header.Set("X-API-Key", c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("waid: http: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("waid: read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		var errResp struct {
			Error string `json:"error"`
		}
		msg := http.StatusText(resp.StatusCode)
		if json.Unmarshal(respBody, &errResp) == nil && errResp.Error != "" {
			msg = errResp.Error
		}
		return nil, &WAIDError{StatusCode: resp.StatusCode, Message: msg, Body: respBody}
	}

	var report ImportReport
	if err := json.Unmarshal(respBody, &report); err != nil {
		return nil, fmt.Errorf("waid: unmarshal response: %w", err)
	}
	return &report, nil
}

// CreateWebhook registers a new webhook target.
func (c *Client) CreateWebhook(ctx context.Context, input CreateWebhookInput) (*WebhookTarget, error) {
	var target WebhookTarget
	if err := c.do(ctx, http.MethodPost, "/webhooks", input, &target); err != nil {
		return nil, err
	}
	return &target, nil
}

// ListWebhooks returns all active webhook targets.
func (c *Client) ListWebhooks(ctx context.Context) ([]WebhookTarget, error) {
	var targets []WebhookTarget
	if err := c.do(ctx, http.MethodGet, "/webhooks", nil, &targets); err != nil {
		return nil, err
	}
	return targets, nil
}

// DeleteWebhook removes a webhook target by UUID.
func (c *Client) DeleteWebhook(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodDelete, "/webhooks/"+url.PathEscape(id), nil, nil)
}

// Health checks service liveness.
func (c *Client) Health(ctx context.Context) (*HealthStatus, error) {
	var status HealthStatus
	if err := c.do(ctx, http.MethodGet, "/health", nil, &status); err != nil {
		return nil, err
	}
	return &status, nil
}
