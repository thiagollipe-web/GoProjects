// Package upstream talks to OpenAI-compatible providers and implements
// ordered fallback guarded by a per-provider circuit breaker.
package upstream

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/thiagollipe-web/GoProjects/llm-gateway/internal/config"
)

// ErrNoUpstream is returned when no provider serves the requested model.
var ErrNoUpstream = errors.New("no upstream available for model")

// Provider is one configured upstream plus its runtime state.
type Provider struct {
	cfg    config.Upstream
	client *http.Client
	apiKey string

	breaker *breaker

	Requests atomic.Int64
	Failures atomic.Int64
}

// Name returns the configured provider name.
func (p *Provider) Name() string { return p.cfg.Name }

// Pool holds providers in priority order.
type Pool struct {
	providers []*Provider
}

// NewPool builds providers from config; keys are read from the environment.
func NewPool(cfg *config.Config) *Pool {
	pool := &Pool{}
	for _, u := range cfg.Upstreams {
		p := &Provider{
			cfg:     u,
			client:  &http.Client{Timeout: u.Timeout.Duration},
			breaker: newBreaker(cfg.BreakerFails, cfg.BreakerCool.Duration),
		}
		if u.APIKeyEnv != "" {
			p.apiKey = os.Getenv(u.APIKeyEnv)
		}
		pool.providers = append(pool.providers, p)
	}
	return pool
}

// Providers exposes providers (for metrics).
func (pl *Pool) Providers() []*Provider { return pl.providers }

// Models lists the public aliases served by the pool, de-duplicated.
func (pl *Pool) Models() []string {
	seen := map[string]bool{}
	var out []string
	for _, p := range pl.providers {
		for alias := range p.cfg.Models {
			if !seen[alias] {
				seen[alias] = true
				out = append(out, alias)
			}
		}
	}
	return out
}

// State returns the breaker state name, for metrics.
func (p *Provider) State() string { return p.breaker.state() }

// Do sends a chat completion body to the first healthy provider that serves
// `model`, falling back on network errors, 429 and 5xx. On success the caller
// owns (and must close) the response body, so streaming works unchanged.
func (pl *Pool) Do(ctx context.Context, model string, body map[string]any) (*http.Response, string, error) {
	var lastErr error
	tried := false
	for _, p := range pl.providers {
		upModel, ok := p.cfg.Models[model]
		if !ok {
			continue
		}
		if !p.breaker.allow() {
			lastErr = fmt.Errorf("%s: circuit open", p.cfg.Name)
			continue
		}
		tried = true
		resp, err := p.send(ctx, upModel, body)
		if err != nil {
			p.Failures.Add(1)
			p.breaker.failure()
			lastErr = fmt.Errorf("%s: %w", p.cfg.Name, err)
			if ctx.Err() != nil {
				return nil, "", ctx.Err()
			}
			continue
		}
		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
			p.Failures.Add(1)
			p.breaker.failure()
			io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<16))
			resp.Body.Close()
			lastErr = fmt.Errorf("%s: status %d", p.cfg.Name, resp.StatusCode)
			continue
		}
		p.breaker.success()
		return resp, p.cfg.Name, nil
	}
	if !tried && lastErr == nil {
		return nil, "", ErrNoUpstream
	}
	return nil, "", fmt.Errorf("all upstreams failed: %w", lastErr)
}

func (p *Provider) send(ctx context.Context, upModel string, body map[string]any) (*http.Response, error) {
	p.Requests.Add(1)
	payload := make(map[string]any, len(body))
	for k, v := range body {
		payload[k] = v
	}
	payload["model"] = upModel
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	url := strings.TrimRight(p.cfg.BaseURL, "/") + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
	}
	return p.client.Do(req)
}

// breaker is a minimal circuit breaker: closed -> open after N consecutive
// failures -> half-open after the cooldown (one probe allowed).
type breaker struct {
	mu       sync.Mutex
	maxFails int
	cooldown time.Duration
	fails    int
	openedAt time.Time
	probing  bool
	now      func() time.Time
}

func newBreaker(maxFails int, cooldown time.Duration) *breaker {
	return &breaker{maxFails: maxFails, cooldown: cooldown, now: time.Now}
}

func (b *breaker) allow() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.fails < b.maxFails {
		return true
	}
	if b.now().Sub(b.openedAt) >= b.cooldown && !b.probing {
		b.probing = true
		return true
	}
	return false
}

func (b *breaker) success() {
	b.mu.Lock()
	b.fails, b.probing = 0, false
	b.mu.Unlock()
}

func (b *breaker) failure() {
	b.mu.Lock()
	b.fails++
	b.probing = false
	if b.fails >= b.maxFails {
		b.openedAt = b.now()
	}
	b.mu.Unlock()
}

func (b *breaker) state() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	switch {
	case b.fails < b.maxFails:
		return "closed"
	case b.now().Sub(b.openedAt) >= b.cooldown:
		return "half-open"
	default:
		return "open"
	}
}
