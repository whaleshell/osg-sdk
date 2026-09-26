// Package whaleshell is the Go SDK for the whaleshell gateway control plane.
//
// Every /v1 gateway route requires a bearer token (OIDC access token or the
// local-dev token from `<gateway data dir>/auth_token`). New picks up
// WHALESHELL_GATEWAY_TOKEN; NewWithToken sets it explicitly.
//
// Exec runs over the gateway supervisor SSH relay (OpenShell ExecSandbox).
// Interactive sessions and IDE access use SSH sessions: CreateSSHSession plus
// `whaleshell ssh-proxy` / `whaleshell sandbox connect`.
package whaleshell

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	gc "github.com/whaleshell/whaleshell-sdk/internal/gatewayclient"
)

// EnvToken is the default bearer for New.
const EnvToken = "WHALESHELL_GATEWAY_TOKEN"

// ErrConnectUnsupported means interactive sessions stay on the CLI.
var ErrConnectUnsupported = errors.New("whaleshell-sdk: interactive connect is not supported; use: whaleshell sandbox connect <name>")

// ErrSandboxNotReady is returned when the sandbox supervisor relay is not connected.
var ErrSandboxNotReady = gc.ErrSandboxNotReady

// Client wraps the gateway HTTP API.
type Client struct {
	*gc.Client
}

// New builds a client for a gateway base URL (e.g. http://127.0.0.1:7443),
// authenticated with $WHALESHELL_GATEWAY_TOKEN when set.
func New(baseURL string) *Client {
	return NewWithToken(baseURL, os.Getenv(EnvToken))
}

// NewWithToken returns a client with bearer auth.
func NewWithToken(base, token string) *Client {
	c := gc.NewWithToken(base, strings.TrimSpace(token))
	c.HTTP.Timeout = 70 * time.Second // relay exec
	return &Client{Client: c}
}

// Stable type aliases (gateway HTTP payloads).
type (
	Sandbox        = gc.Sandbox
	ExecResult     = gc.ExecResult
	LogLine        = gc.LogLine
	Proposal       = gc.Proposal
	ProviderRecord = gc.ProviderRecord
	InferenceRoute = gc.InferenceRoute
	ServiceRecord  = gc.ServiceRecord
	SSHSession     = gc.SSHSession
	SSHSessionInfo = gc.SSHSessionInfo
)

// Create registers (upserts) a sandbox in the gateway registry.
func (c *Client) Create(ctx context.Context, sb Sandbox) error {
	if sb.Name == "" {
		return fmt.Errorf("sandbox name required")
	}
	return c.UpsertSandbox(ctx, sb)
}

// List returns registered sandboxes.
func (c *Client) List(ctx context.Context) ([]Sandbox, error) {
	return c.ListSandboxes(ctx)
}

// Get returns one sandbox by name.
func (c *Client) Get(ctx context.Context, name string) (Sandbox, error) {
	return c.GetSandbox(ctx, name)
}

// Delete removes a sandbox from the registry (revokes its SSH sessions and
// supervisor token).
func (c *Client) Delete(ctx context.Context, name string) error {
	return c.DeleteSandbox(ctx, name)
}

// Exec runs argv in the sandbox over the supervisor SSH relay.
func (c *Client) Exec(ctx context.Context, name string, argv ...string) (ExecResult, error) {
	if name == "" || len(argv) == 0 {
		return ExecResult{}, fmt.Errorf("usage: Exec(name, argv...)")
	}
	return c.Client.Exec(ctx, name, argv)
}

// Connect is intentionally unsupported in the SDK.
func (c *Client) Connect(ctx context.Context, name string) error {
	_ = ctx
	_ = name
	return ErrConnectUnsupported
}
