// Package gatewayclient talks to whaleshell-gateway HTTP API.
package gatewayclient

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client is a tiny HTTP client for whaleshell-gateway.
type Client struct {
	Base  string
	Token string // optional Bearer
	HTTP  *http.Client
}

// New returns a client for base URL (for example http://127.0.0.1:7443).
// The gateway requires a bearer on every /v1 route; set Token (or use
// NewWithToken) — it is attached to every request by the transport.
func New(base string) *Client {
	c := &Client{Base: strings.TrimRight(base, "/")}
	c.HTTP = &http.Client{Timeout: 10 * time.Second, Transport: &authTransport{c: c}}
	return c
}

// authTransport adds the bearer to requests that do not carry one, so no
// call site can forget authentication.
type authTransport struct {
	c    *Client
	base http.RoundTripper
}

func (t *authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	base := t.base
	if base == nil {
		base = http.DefaultTransport
	}
	if tok := t.c.Token; tok != "" && req.Header.Get("Authorization") == "" {
		req = req.Clone(req.Context())
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	return base.RoundTrip(req)
}

// NewWithToken returns a client with bearer auth.
func NewWithToken(base, token string) *Client {
	c := New(base)
	c.Token = strings.TrimSpace(token)
	return c
}

func (c *Client) auth(req *http.Request) {
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
}

// Healthz hits /healthz.
func (c *Client) Healthz(ctx context.Context) (map[string]any, error) {
	var out map[string]any
	if err := c.get(ctx, "/healthz", &out); err != nil {
		return nil, err
	}
	return out, nil
}

// Info hits /v1/info.
func (c *Client) Info(ctx context.Context) (map[string]any, error) {
	var out map[string]any
	if err := c.get(ctx, "/v1/info", &out); err != nil {
		return nil, err
	}
	return out, nil
}

// Sandbox is the registry payload.
type Sandbox struct {
	Name              string            `json:"name"`
	ID                string            `json:"id,omitempty"`
	Image             string            `json:"image,omitempty"`
	Network           string            `json:"network,omitempty"`
	Status            string            `json:"status,omitempty"`
	Labels            map[string]string `json:"labels,omitempty"`
	BasePolicyYAML    string            `json:"base_policy_yaml,omitempty"`
	AttachedProviders []string          `json:"attached_providers,omitempty"`
}

// UpsertSandbox PUT /v1/sandboxes/{name}.
func (c *Client) UpsertSandbox(ctx context.Context, sb Sandbox) error {
	b, err := json.Marshal(sb)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, c.Base+"/v1/sandboxes/"+sb.Name, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		body, _ := io.ReadAll(res.Body)
		return fmt.Errorf("gateway upsert: %s: %s", res.Status, bytes.TrimSpace(body))
	}
	return nil
}

// DeleteSandbox DELETE /v1/sandboxes/{name}.
func (c *Client) DeleteSandbox(ctx context.Context, name string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.Base+"/v1/sandboxes/"+name, nil)
	if err != nil {
		return err
	}
	res, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 && res.StatusCode != http.StatusNotFound {
		body, _ := io.ReadAll(res.Body)
		return fmt.Errorf("gateway delete: %s: %s", res.Status, bytes.TrimSpace(body))
	}
	return nil
}

// ListSandboxes GET /v1/sandboxes.
func (c *Client) ListSandboxes(ctx context.Context) ([]Sandbox, error) {
	var out struct {
		Sandboxes []Sandbox `json:"sandboxes"`
	}
	if err := c.get(ctx, "/v1/sandboxes", &out); err != nil {
		return nil, err
	}
	return out.Sandboxes, nil
}

// GetGlobalPolicy GET /v1/policy/global (YAML bytes).
func (c *Client) GetGlobalPolicy(ctx context.Context) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.Base+"/v1/policy/global", nil)
	if err != nil {
		return nil, err
	}
	res, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	if res.StatusCode >= 300 {
		return nil, fmt.Errorf("gateway global policy: %s: %s", res.Status, bytes.TrimSpace(body))
	}
	return body, nil
}

// PutGlobalPolicy PUT /v1/policy/global with YAML body.
func (c *Client) PutGlobalPolicy(ctx context.Context, yaml []byte) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, c.Base+"/v1/policy/global", bytes.NewReader(yaml))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/yaml")
	res, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		body, _ := io.ReadAll(res.Body)
		return fmt.Errorf("gateway put global policy: %s: %s", res.Status, bytes.TrimSpace(body))
	}
	return nil
}

// ProfileInfo is a catalog entry summary.
type ProfileInfo struct {
	ID     string `json:"id"`
	Source string `json:"source"`
}

// ListProfiles GET /v1/profiles.
func (c *Client) ListProfiles(ctx context.Context) ([]ProfileInfo, error) {
	var out struct {
		Profiles []ProfileInfo `json:"profiles"`
	}
	if err := c.get(ctx, "/v1/profiles", &out); err != nil {
		return nil, err
	}
	return out.Profiles, nil
}

// PutProfile PUT /v1/profiles/{id} with YAML body.
func (c *Client) PutProfile(ctx context.Context, id string, yaml []byte) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, c.Base+"/v1/profiles/"+id, bytes.NewReader(yaml))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/yaml")
	res, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		body, _ := io.ReadAll(res.Body)
		return fmt.Errorf("gateway put profile: %s: %s", res.Status, bytes.TrimSpace(body))
	}
	return nil
}

// ProviderRecord is a gateway provider instance (env key names only on GET).
type ProviderRecord struct {
	Name                  string            `json:"name"`
	Type                  string            `json:"type"`
	EnvVars               []string          `json:"env_vars,omitempty"`
	Credentials           map[string]string `json:"credentials,omitempty"` // write-only on PUT
	CredentialExpiresAtMS map[string]int64  `json:"credential_expires_at_ms,omitempty"`
	RuntimeCredentials    bool              `json:"runtime_credentials,omitempty"`
	Config                map[string]string `json:"config,omitempty"`
}

// PutProvider PUT /v1/providers/{name}. Credentials values are stored encrypted on the gateway.
func (c *Client) PutProvider(ctx context.Context, rec ProviderRecord) error {
	b, err := json.Marshal(rec)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, c.Base+"/v1/providers/"+rec.Name, bytes.NewReader(b))
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
		return fmt.Errorf("gateway put provider: %s: %s", res.Status, bytes.TrimSpace(body))
	}
	return nil
}

// ListProviders GET /v1/providers.
func (c *Client) ListProviders(ctx context.Context) ([]ProviderRecord, error) {
	var out struct {
		Providers []ProviderRecord `json:"providers"`
	}
	if err := c.get(ctx, "/v1/providers", &out); err != nil {
		return nil, err
	}
	return out.Providers, nil
}

// GetProvider GET /v1/providers/{name} (metadata only; no secret values).
func (c *Client) GetProvider(ctx context.Context, name string) (ProviderRecord, error) {
	var out ProviderRecord
	if err := c.get(ctx, "/v1/providers/"+name, &out); err != nil {
		return ProviderRecord{}, err
	}
	return out, nil
}

// DeleteProvider DELETE /v1/providers/{name}.
func (c *Client) DeleteProvider(ctx context.Context, name string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.Base+"/v1/providers/"+name, nil)
	if err != nil {
		return err
	}
	res, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		body, _ := io.ReadAll(res.Body)
		return fmt.Errorf("gateway delete provider: %s: %s", res.Status, bytes.TrimSpace(body))
	}
	return nil
}

// DeleteProfile DELETE /v1/profiles/{id}.
func (c *Client) DeleteProfile(ctx context.Context, id string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.Base+"/v1/profiles/"+id, nil)
	if err != nil {
		return err
	}
	res, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		body, _ := io.ReadAll(res.Body)
		return fmt.Errorf("gateway delete profile: %s: %s", res.Status, bytes.TrimSpace(body))
	}
	return nil
}

// AttachProvider PUT /v1/sandboxes/{sandbox}/providers/{provider}.
func (c *Client) AttachProvider(ctx context.Context, sandbox, provider string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, c.Base+"/v1/sandboxes/"+sandbox+"/providers/"+provider, nil)
	if err != nil {
		return err
	}
	res, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		body, _ := io.ReadAll(res.Body)
		return fmt.Errorf("gateway attach provider: %s: %s", res.Status, bytes.TrimSpace(body))
	}
	return nil
}

// DetachProvider DELETE /v1/sandboxes/{sandbox}/providers/{provider}.
func (c *Client) DetachProvider(ctx context.Context, sandbox, provider string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.Base+"/v1/sandboxes/"+sandbox+"/providers/"+provider, nil)
	if err != nil {
		return err
	}
	res, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		body, _ := io.ReadAll(res.Body)
		return fmt.Errorf("gateway detach provider: %s: %s", res.Status, bytes.TrimSpace(body))
	}
	return nil
}

// EffectivePolicy GET /v1/sandboxes/{name}/effective-policy (YAML).
func (c *Client) EffectivePolicy(ctx context.Context, sandbox string) ([]byte, error) {
	return c.GetSandboxPolicy(ctx, sandbox, "full")
}

// GetSandboxPolicy GET /v1/sandboxes/{name}/policy?view=base|full (YAML).
// view "base" is the editable sandbox layer; "full" is Compose(base, providers).
func (c *Client) GetSandboxPolicy(ctx context.Context, sandbox, view string) ([]byte, error) {
	view = strings.ToLower(strings.TrimSpace(view))
	switch view {
	case "", "full", "effective":
		view = "full"
	case "base":
		// ok
	default:
		return nil, fmt.Errorf("policy view must be base or full")
	}
	u := c.Base + "/v1/sandboxes/" + sandbox + "/policy?view=" + view
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	res, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	if res.StatusCode >= 300 {
		// Older gateways only expose effective-policy.
		if view == "full" {
			req2, err := http.NewRequestWithContext(ctx, http.MethodGet, c.Base+"/v1/sandboxes/"+sandbox+"/effective-policy", nil)
			if err != nil {
				return nil, err
			}
			res2, err := c.HTTP.Do(req2)
			if err != nil {
				return nil, err
			}
			defer res2.Body.Close()
			body2, _ := io.ReadAll(res2.Body)
			if res2.StatusCode >= 300 {
				return nil, fmt.Errorf("gateway policy get: %s: %s", res.Status, bytes.TrimSpace(body))
			}
			return body2, nil
		}
		return nil, fmt.Errorf("gateway policy get: %s: %s", res.Status, bytes.TrimSpace(body))
	}
	return body, nil
}

// PutSandboxPolicy PUT /v1/sandboxes/{name}/policy — stores base YAML, returns effective YAML.
// OpenShell-style: providers stay attached; gateway re-composes before accepting.
func (c *Client) PutSandboxPolicy(ctx context.Context, sandbox string, baseYAML []byte) (effective []byte, stripped int, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, c.Base+"/v1/sandboxes/"+sandbox+"/policy", bytes.NewReader(baseYAML))
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Content-Type", "application/yaml")
	res, err := c.HTTP.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	if res.StatusCode >= 300 {
		return nil, 0, fmt.Errorf("gateway policy set: %s: %s", res.Status, bytes.TrimSpace(body))
	}
	if v := res.Header.Get("X-Whaleshell-Stripped-Provider-Rules"); v != "" {
		_, _ = fmt.Sscanf(v, "%d", &stripped)
	}
	return body, stripped, nil
}

// PolicyRevisionMeta is metadata for policy list (no YAML body).
type PolicyRevisionMeta struct {
	Rev       int       `json:"rev"`
	UpdatedAt time.Time `json:"updated_at"`
	Bytes     int       `json:"bytes"`
	Status    string    `json:"status"`
}

// ListPolicyRevisions GET /v1/sandboxes/{name}/policy-revisions.
func (c *Client) ListPolicyRevisions(ctx context.Context, sandbox string) ([]PolicyRevisionMeta, error) {
	var out struct {
		Revisions []PolicyRevisionMeta `json:"revisions"`
	}
	if err := c.get(ctx, "/v1/sandboxes/"+sandbox+"/policy-revisions", &out); err != nil {
		return nil, err
	}
	return out.Revisions, nil
}

// GetPolicyRevision GET /v1/sandboxes/{name}/policy-revisions?rev=N (YAML body).
func (c *Client) GetPolicyRevision(ctx context.Context, sandbox string, rev int) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		fmt.Sprintf("%s/v1/sandboxes/%s/policy-revisions?rev=%d", c.Base, sandbox, rev), nil)
	if err != nil {
		return "", err
	}
	res, err := c.HTTP.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	if res.StatusCode >= 300 {
		return "", fmt.Errorf("gateway policy rev: %s: %s", res.Status, bytes.TrimSpace(body))
	}
	return string(body), nil
}

// SandboxProviderAttachment is metadata for sandbox provider list.
type SandboxProviderAttachment struct {
	Name    string   `json:"name"`
	Type    string   `json:"type,omitempty"`
	EnvVars []string `json:"env_vars,omitempty"`
}

// ListSandboxProviders GET /v1/sandboxes/{name}/providers.
func (c *Client) ListSandboxProviders(ctx context.Context, sandbox string) ([]SandboxProviderAttachment, error) {
	var out struct {
		Providers []SandboxProviderAttachment `json:"providers"`
	}
	if err := c.get(ctx, "/v1/sandboxes/"+sandbox+"/providers", &out); err != nil {
		return nil, err
	}
	return out.Providers, nil
}

func (c *Client) get(ctx context.Context, path string, dest any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.Base+path, nil)
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
		return fmt.Errorf("gateway %s: %s: %s", path, res.Status, bytes.TrimSpace(body))
	}
	return json.NewDecoder(res.Body).Decode(dest)
}

// ResolveSecrets GET /v1/sandboxes/{name}/secrets — sidecar credential map.
func (c *Client) ResolveSecrets(ctx context.Context, sandbox string) (map[string]string, error) {
	var out struct {
		Secrets map[string]string `json:"secrets"`
	}
	if err := c.get(ctx, "/v1/sandboxes/"+sandbox+"/secrets", &out); err != nil {
		return nil, err
	}
	if out.Secrets == nil {
		out.Secrets = map[string]string{}
	}
	return out.Secrets, nil
}

// PostLogs POST /v1/sandboxes/{name}/logs — ingest observation lines.
func (c *Client) PostLogs(ctx context.Context, sandbox string, lines []LogLine) error {
	b, err := json.Marshal(map[string]any{"lines": lines})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.Base+"/v1/sandboxes/"+sandbox+"/logs", bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		body, _ := io.ReadAll(res.Body)
		return fmt.Errorf("gateway post logs: %s: %s", res.Status, bytes.TrimSpace(body))
	}
	return nil
}

// Proposal is a policy.local chunk stored on the gateway.
type Proposal struct {
	ID               string    `json:"id"`
	Sandbox          string    `json:"sandbox"`
	Status           string    `json:"status"`
	IntentSummary    string    `json:"intent_summary,omitempty"`
	RuleName         string    `json:"rule_name,omitempty"`
	RuleYAML         string    `json:"rule_yaml,omitempty"`
	Hosts            []string  `json:"hosts,omitempty"`
	RejectionReason  string    `json:"rejection_reason,omitempty"`
	ValidationResult string    `json:"validation_result,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
	DecidedAt        time.Time `json:"decided_at,omitempty"`
}

// ListProposals GET /v1/sandboxes/{name}/proposals.
func (c *Client) ListProposals(ctx context.Context, sandbox, status string) ([]Proposal, error) {
	path := "/v1/sandboxes/" + sandbox + "/proposals"
	if status != "" {
		path += "?status=" + url.QueryEscape(status)
	}
	var out struct {
		Proposals []Proposal `json:"proposals"`
	}
	if err := c.get(ctx, path, &out); err != nil {
		return nil, err
	}
	return out.Proposals, nil
}

// GetProposal GET /v1/sandboxes/{name}/proposals/{id}.
func (c *Client) GetProposal(ctx context.Context, sandbox, id string) (Proposal, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.Base+"/v1/sandboxes/"+sandbox+"/proposals/"+id, nil)
	if err != nil {
		return Proposal{}, err
	}
	c.auth(req)
	res, err := c.HTTP.Do(req)
	if err != nil {
		return Proposal{}, err
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	if res.StatusCode >= 300 {
		return Proposal{}, fmt.Errorf("get proposal: %s: %s", res.Status, bytes.TrimSpace(body))
	}
	var p Proposal
	if err := json.Unmarshal(body, &p); err != nil {
		return Proposal{}, err
	}
	return p, nil
}

// ApproveProposal POST /v1/sandboxes/{name}/proposals/{id}/approve — merges rule into base.
func (c *Client) ApproveProposal(ctx context.Context, sandbox, id string) (Proposal, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.Base+"/v1/sandboxes/"+sandbox+"/proposals/"+id+"/approve", nil)
	if err != nil {
		return Proposal{}, err
	}
	c.auth(req)
	res, err := c.HTTP.Do(req)
	if err != nil {
		return Proposal{}, err
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	if res.StatusCode >= 300 {
		return Proposal{}, fmt.Errorf("approve proposal: %s: %s", res.Status, bytes.TrimSpace(body))
	}
	var p Proposal
	if err := json.Unmarshal(body, &p); err != nil {
		return Proposal{}, err
	}
	return p, nil
}

// RejectProposal POST /v1/sandboxes/{name}/proposals/{id}/reject.
func (c *Client) RejectProposal(ctx context.Context, sandbox, id, reason string) (Proposal, error) {
	b, _ := json.Marshal(map[string]string{"reason": reason})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.Base+"/v1/sandboxes/"+sandbox+"/proposals/"+id+"/reject", bytes.NewReader(b))
	if err != nil {
		return Proposal{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	c.auth(req)
	res, err := c.HTTP.Do(req)
	if err != nil {
		return Proposal{}, err
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	if res.StatusCode >= 300 {
		return Proposal{}, fmt.Errorf("reject proposal: %s: %s", res.Status, bytes.TrimSpace(body))
	}
	var p Proposal
	if err := json.Unmarshal(body, &p); err != nil {
		return Proposal{}, err
	}
	return p, nil
}

// LogLine is one observation event for ingest/SSE.
type LogLine struct {
	TS     time.Time `json:"ts"`
	Source string    `json:"source"`
	Level  string    `json:"level"`
	Text   string    `json:"text"`
}

// FollowLogs streams SSE log lines to w. names may be multiple; all=true uses gateway ?all=1.
func (c *Client) FollowLogs(ctx context.Context, names []string, all bool, since, source, level string, w io.Writer) error {
	u := c.Base + "/v1/logs?"
	q := []string{}
	if all {
		q = append(q, "all=1")
	}
	for _, n := range names {
		q = append(q, "name="+n)
	}
	q = append(q, "follow=1")
	if since != "" {
		q = append(q, "since="+since)
	}
	if source != "" {
		q = append(q, "source="+source)
	}
	if level != "" {
		q = append(q, "level="+level)
	}
	u += strings.Join(q, "&")

	// Single-name optimized path
	if !all && len(names) == 1 {
		u = c.Base + "/v1/sandboxes/" + names[0] + "/logs?follow=1"
		if since != "" {
			u += "&since=" + since
		}
		if source != "" {
			u += "&source=" + source
		}
		if level != "" {
			u += "&level=" + level
		}
	}

	httpClient := c.HTTP
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	// SSE needs no overall timeout
	sseClient := *httpClient
	sseClient.Timeout = 0

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "text/event-stream")
	res, err := sseClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		body, _ := io.ReadAll(res.Body)
		return fmt.Errorf("gateway follow logs: %s: %s", res.Status, bytes.TrimSpace(body))
	}
	sc := bufio.NewScanner(res.Body)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := sc.Text()
		if strings.HasPrefix(line, "data: ") {
			if _, err := fmt.Fprintln(w, strings.TrimPrefix(line, "data: ")); err != nil {
				return err
			}
		}
	}
	return sc.Err()
}

// GetLogsSnapshot returns recent log lines (non-follow JSON).
func (c *Client) GetLogsSnapshot(ctx context.Context, name, since, source, level string) ([]LogLine, error) {
	path := "/v1/sandboxes/" + name + "/logs?"
	parts := []string{}
	if since != "" {
		parts = append(parts, "since="+since)
	}
	if source != "" {
		parts = append(parts, "source="+source)
	}
	if level != "" {
		parts = append(parts, "level="+level)
	}
	path += strings.Join(parts, "&")
	var out struct {
		Lines []LogLine `json:"lines"`
	}
	if err := c.get(ctx, path, &out); err != nil {
		return nil, err
	}
	return out.Lines, nil
}
