package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"streamagent/internal/model"
)

type Client struct {
	baseURL string
	http    *http.Client
	debug   bool
	logf    func(string, ...any)
}

// New 创建对业务 API 的访问客户端。
func New(baseURL string, debug bool, logf func(string, ...any)) *Client {
	if logf == nil {
		logf = func(string, ...any) {}
	}
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		http:    &http.Client{},
		debug:   debug,
		logf:    logf,
	}
}

// Unlock 请求 /api/unlock，并返回节点和平台映射。
func (c *Client) Unlock(ctx context.Context, token string) (*model.UnlockResponse, error) {
	if c.debug {
		c.logf("unlock request: %s/api/unlock", c.baseURL)
	}
	endpoint, err := url.Parse(c.baseURL + "/api/unlock")
	if err != nil {
		return nil, fmt.Errorf("parse unlock endpoint: %w", err)
	}
	q := endpoint.Query()
	q.Set("token", token)
	endpoint.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create unlock request: %w", err)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do unlock request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		if len(body) > 0 {
			return nil, fmt.Errorf("unlock api returned status %s: %s", resp.Status, strings.TrimSpace(string(body)))
		}
		return nil, fmt.Errorf("unlock api returned status %s", resp.Status)
	}

	var payload model.UnlockResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode unlock response: %w", err)
	}
	if payload.Code != http.StatusOK {
		if payload.Msg == "" {
			payload.Msg = resp.Status
		}
		return nil, fmt.Errorf("unlock api failed: %s", payload.Msg)
	}
	if c.debug {
		c.logf("unlock response: code=%d nodes=%d platforms=%d", payload.Code, len(payload.Data.Node), len(payload.Data.Platform))
	}
	return &payload, nil
}

// Upload 请求 /api/upload，把本机检测到的解锁平台上报给服务端。
func (c *Client) Upload(ctx context.Context, token string, payload model.UploadPayload) error {
	if c.debug {
		c.logf("upload request: %s/api/upload id=%d platforms=%d", c.baseURL, payload.ID, len(payload.Platform))
	}
	endpoint, err := url.Parse(c.baseURL + "/api/upload")
	if err != nil {
		return fmt.Errorf("parse upload endpoint: %w", err)
	}
	q := endpoint.Query()
	q.Set("token", token)
	endpoint.RawQuery = q.Encode()

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal upload payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), strings.NewReader(string(body)))
	if err != nil {
		return fmt.Errorf("create upload request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("do upload request: %w", err)
	}
	defer resp.Body.Close()

	var respPayload model.UploadResponse
	if err := json.NewDecoder(resp.Body).Decode(&respPayload); err != nil {
		return fmt.Errorf("decode upload response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("upload api returned status %s", resp.Status)
	}
	if respPayload.Code != http.StatusOK {
		if respPayload.Msg == "" {
			respPayload.Msg = resp.Status
		}
		return fmt.Errorf("upload api failed: %s", respPayload.Msg)
	}
	if c.debug {
		c.logf("upload response: code=%d not_platforms=%v", respPayload.Code, respPayload.Data.NotPlatforms)
	}
	return nil
}
