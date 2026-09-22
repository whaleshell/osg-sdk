package gatewayclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// InferenceRoute is GET/PUT /v1/inference.
type InferenceRoute struct {
	Provider   string `json:"provider"`
	Model      string `json:"model,omitempty"`
	TimeoutSec int    `json:"timeout_sec,omitempty"`
	Version    int    `json:"version,omitempty"`
}

// GetInference GET /v1/inference.
func (c *Client) GetInference(ctx context.Context) (InferenceRoute, error) {
	var out InferenceRoute
	if err := c.get(ctx, "/v1/inference", &out); err != nil {
		return InferenceRoute{}, err
	}
	return out, nil
}

// PutInference PUT /v1/inference.
func (c *Client) PutInference(ctx context.Context, route InferenceRoute) (InferenceRoute, error) {
	b, err := json.Marshal(route)
	if err != nil {
		return InferenceRoute{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, c.Base+"/v1/inference", bytes.NewReader(b))
	if err != nil {
		return InferenceRoute{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	c.auth(req)
	res, err := c.HTTP.Do(req)
	if err != nil {
		return InferenceRoute{}, err
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	if res.StatusCode >= 300 {
		return InferenceRoute{}, fmt.Errorf("gateway put inference: %s: %s", res.Status, bytes.TrimSpace(body))
	}
	var out InferenceRoute
	if err := json.Unmarshal(body, &out); err != nil {
		return InferenceRoute{}, err
	}
	return out, nil
}

// DeleteInference DELETE /v1/inference.
func (c *Client) DeleteInference(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.Base+"/v1/inference", nil)
	if err != nil {
		return err
	}
	c.auth(req)
	res, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		body, _ := io.ReadAll(res.Body)
		return fmt.Errorf("gateway delete inference: %s: %s", res.Status, bytes.TrimSpace(body))
	}
	return nil
}

// Whoami GET /v1/whoami.
func (c *Client) Whoami(ctx context.Context) (map[string]any, error) {
	var out map[string]any
	if err := c.get(ctx, "/v1/whoami", &out); err != nil {
		return nil, err
	}
	return out, nil
}

// PutSetting PUT /v1/settings/{key}.
func (c *Client) PutSetting(ctx context.Context, key, value string) error {
	b, _ := json.Marshal(map[string]string{"value": value})
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, c.Base+"/v1/settings/"+key, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	c.auth(req)
	res, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		body, _ := io.ReadAll(res.Body)
		return fmt.Errorf("gateway put setting: %s: %s", res.Status, bytes.TrimSpace(body))
	}
	return nil
}

// GetSetting GET /v1/settings/{key}.
func (c *Client) GetSetting(ctx context.Context, key string) (string, error) {
	var out struct {
		Value string `json:"value"`
	}
	if err := c.get(ctx, "/v1/settings/"+key, &out); err != nil {
		return "", err
	}
	return out.Value, nil
}

// ListSettings GET /v1/settings.
func (c *Client) ListSettings(ctx context.Context) (map[string]string, error) {
	var out struct {
		Settings map[string]string `json:"settings"`
	}
	if err := c.get(ctx, "/v1/settings", &out); err != nil {
		return nil, err
	}
	return out.Settings, nil
}

// ConfigureProviderRefresh PUT /v1/providers/{name}/refresh/{key}.
func (c *Client) ConfigureProviderRefresh(ctx context.Context, name, key, strategy string, material map[string]string, expiresAtMS int64) error {
	body := map[string]any{
		"credential_key": key,
		"strategy":       strategy,
		"material":       material,
		"expires_at_ms":  expiresAtMS,
	}
	b, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, c.Base+"/v1/providers/"+name+"/refresh/"+key, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	c.auth(req)
	res, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		raw, _ := io.ReadAll(res.Body)
		return fmt.Errorf("gateway refresh configure: %s: %s", res.Status, bytes.TrimSpace(raw))
	}
	return nil
}

// RotateProviderRefresh POST /v1/providers/{name}/refresh/{key}/rotate.
func (c *Client) RotateProviderRefresh(ctx context.Context, name, key string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.Base+"/v1/providers/"+name+"/refresh/"+key+"/rotate", nil)
	if err != nil {
		return err
	}
	c.auth(req)
	res, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		raw, _ := io.ReadAll(res.Body)
		return fmt.Errorf("gateway refresh rotate: %s: %s", res.Status, bytes.TrimSpace(raw))
	}
	return nil
}

// DeleteProviderRefresh DELETE /v1/providers/{name}/refresh/{key}.
func (c *Client) DeleteProviderRefresh(ctx context.Context, name, key string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.Base+"/v1/providers/"+name+"/refresh/"+key, nil)
	if err != nil {
		return err
	}
	c.auth(req)
	res, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		raw, _ := io.ReadAll(res.Body)
		return fmt.Errorf("gateway refresh delete: %s: %s", res.Status, bytes.TrimSpace(raw))
	}
	return nil
}

// AuthLogin GET /v1/auth/login — local-dev token mint.
func (c *Client) AuthLogin(ctx context.Context) (string, error) {
	var out struct {
		Token string `json:"token"`
	}
	if err := c.get(ctx, "/v1/auth/login", &out); err != nil {
		return "", err
	}
	return out.Token, nil
}
