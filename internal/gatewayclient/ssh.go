package gatewayclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// ErrSandboxNotReady is returned when the sandbox supervisor relay is not
// connected (gateway 412, OpenShell FailedPrecondition "sandbox is not ready").
var ErrSandboxNotReady = errors.New("sandbox is not ready (supervisor relay not connected)")

// SSHSession is a CreateSshSession response. Token is shown once.
type SSHSession struct {
	SessionID     string `json:"session_id"`
	SandboxID     string `json:"sandbox_id"`
	Token         string `json:"token"`
	GatewayScheme string `json:"gateway_scheme"`
	GatewayHost   string `json:"gateway_host"`
	GatewayPort   int    `json:"gateway_port"`
	ExpiresAtMS   int64  `json:"expires_at_ms"`
}

// SSHSessionInfo is a listed session (no token).
type SSHSessionInfo struct {
	ID          string `json:"id"`
	Sandbox     string `json:"sandbox"`
	Subject     string `json:"subject,omitempty"`
	CreatedAtMS int64  `json:"created_at_ms"`
	ExpiresAtMS int64  `json:"expires_at_ms,omitempty"`
	Revoked     bool   `json:"revoked,omitempty"`
}

func (c *Client) do(ctx context.Context, method, path string, body any, want int, dest any) error {
	var rd io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		rd = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.Base+path, rd)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	c.auth(req)
	res, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(res.Body, 4<<20))
	if res.StatusCode == http.StatusPreconditionFailed {
		return fmt.Errorf("%w: %s", ErrSandboxNotReady, bytes.TrimSpace(b))
	}
	if res.StatusCode != want {
		return fmt.Errorf("gateway %s %s: %s: %s", method, path, res.Status, bytes.TrimSpace(b))
	}
	if dest != nil {
		return json.Unmarshal(b, dest)
	}
	return nil
}

// CreateSSHSession mints a short-lived SSH session token for a sandbox
// (OpenShell CreateSshSession). Requires a connected supervisor relay.
func (c *Client) CreateSSHSession(ctx context.Context, sandbox string) (SSHSession, error) {
	var s SSHSession
	err := c.do(ctx, http.MethodPost, "/v1/sandboxes/"+url.PathEscape(sandbox)+"/ssh-session", nil, http.StatusOK, &s)
	return s, err
}

// RevokeSSHSession invalidates a session by id (OpenShell RevokeSshSession).
func (c *Client) RevokeSSHSession(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodDelete, "/v1/ssh-sessions/"+url.PathEscape(id), nil, http.StatusNoContent, nil)
}

// ListSSHSessions lists sessions ("" = all sandboxes).
func (c *Client) ListSSHSessions(ctx context.Context, sandbox string) ([]SSHSessionInfo, error) {
	path := "/v1/ssh-sessions"
	if sandbox != "" {
		path += "?sandbox=" + url.QueryEscape(sandbox)
	}
	var out struct {
		Sessions []SSHSessionInfo `json:"sessions"`
	}
	err := c.do(ctx, http.MethodGet, path, nil, http.StatusOK, &out)
	return out.Sessions, err
}

// IssueSandboxToken rotates the sandbox supervisor token (proxy sidecar
// credential scoped to one sandbox) and returns it once.
func (c *Client) IssueSandboxToken(ctx context.Context, sandbox string) (string, error) {
	var out struct {
		Token string `json:"sandbox_token"`
	}
	err := c.do(ctx, http.MethodPost, "/v1/sandboxes/"+url.PathEscape(sandbox)+"/supervisor-token", nil, http.StatusOK, &out)
	return out.Token, err
}
