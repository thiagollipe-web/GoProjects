package server

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/thiagollipe-web/GoProjects/llm-gateway/internal/config"
	"github.com/thiagollipe-web/GoProjects/llm-gateway/internal/upstream"
)

func newGateway(t *testing.T, cfg *config.Config) *httptest.Server {
	t.Helper()
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}
	s := New(cfg, upstream.NewPool(cfg), slog.New(slog.NewTextHandler(io.Discard, nil)))
	gw := httptest.NewServer(s.Handler())
	t.Cleanup(gw.Close)
	return gw
}

func post(t *testing.T, url, auth, body string) *http.Response {
	t.Helper()
	req, _ := http.NewRequest(http.MethodPost, url+"/v1/chat/completions", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if auth != "" {
		req.Header.Set("Authorization", "Bearer "+auth)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

const chatBody = `{"model":"fast","messages":[{"role":"user","content":"hi"}]}`

func TestFallbackOnServerError(t *testing.T) {
	var primaryHits, backupHits atomic.Int32
	primary := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		primaryHits.Add(1)
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer primary.Close()
	backup := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		backupHits.Add(1)
		var in map[string]any
		_ = json.NewDecoder(r.Body).Decode(&in)
		if in["model"] != "backup-model" { // alias must be rewritten
			t.Errorf("model not rewritten: %v", in["model"])
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"ok"}}]}`))
	}))
	defer backup.Close()

	gw := newGateway(t, &config.Config{Upstreams: []config.Upstream{
		{Name: "primary", BaseURL: primary.URL, Models: map[string]string{"fast": "p-model"}},
		{Name: "backup", BaseURL: backup.URL, Models: map[string]string{"fast": "backup-model"}},
	}})

	resp := post(t, gw.URL, "", chatBody)
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status %d", resp.StatusCode)
	}
	if got := resp.Header.Get("X-Gateway-Provider"); got != "backup" {
		t.Fatalf("provider = %q, want backup", got)
	}
	if primaryHits.Load() != 1 || backupHits.Load() != 1 {
		t.Fatalf("hits primary=%d backup=%d", primaryHits.Load(), backupHits.Load())
	}
}

func TestCircuitBreakerSkipsFailingProvider(t *testing.T) {
	var primaryHits atomic.Int32
	primary := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		primaryHits.Add(1)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer primary.Close()
	backup := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{}`))
	}))
	defer backup.Close()

	gw := newGateway(t, &config.Config{BreakerFails: 2, Upstreams: []config.Upstream{
		{Name: "primary", BaseURL: primary.URL, Models: map[string]string{"fast": "m"}},
		{Name: "backup", BaseURL: backup.URL, Models: map[string]string{"fast": "m"}},
	}})

	for i := 0; i < 5; i++ {
		post(t, gw.URL, "", chatBody).Body.Close()
	}
	if n := primaryHits.Load(); n != 2 {
		t.Fatalf("primary should be skipped once the breaker opens; hits=%d, want 2", n)
	}
}

func TestStreamingPassThrough(t *testing.T) {
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fl := w.(http.Flusher)
		for _, c := range []string{"one", "two"} {
			_, _ = io.WriteString(w, "data: "+c+"\n\n")
			fl.Flush()
		}
		_, _ = io.WriteString(w, "data: [DONE]\n\n")
	}))
	defer up.Close()
	gw := newGateway(t, &config.Config{Upstreams: []config.Upstream{
		{Name: "u", BaseURL: up.URL, Models: map[string]string{"fast": "m"}},
	}})

	resp := post(t, gw.URL, "", `{"model":"fast","stream":true,"messages":[]}`)
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if !bytes.Contains(b, []byte("data: one")) || !bytes.HasSuffix(b, []byte("[DONE]\n\n")) {
		t.Fatalf("unexpected stream: %q", b)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "text/event-stream" {
		t.Fatalf("content-type = %q", ct)
	}
}

func TestAuthAndRateLimit(t *testing.T) {
	t.Setenv("TEST_CLIENT_KEYS", "k1, k2")
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{}`))
	}))
	defer up.Close()
	gw := newGateway(t, &config.Config{ClientKeysEnv: "TEST_CLIENT_KEYS", RatePerMinute: 2,
		Upstreams: []config.Upstream{{Name: "u", BaseURL: up.URL, Models: map[string]string{"fast": "m"}}}})

	if r := post(t, gw.URL, "", chatBody); r.StatusCode != 401 {
		t.Fatalf("no key: status %d, want 401", r.StatusCode)
	}
	if r := post(t, gw.URL, "wrong", chatBody); r.StatusCode != 401 {
		t.Fatalf("wrong key: status %d, want 401", r.StatusCode)
	}
	for i := 0; i < 2; i++ {
		if r := post(t, gw.URL, "k1", chatBody); r.StatusCode != 200 {
			t.Fatalf("request %d: status %d", i, r.StatusCode)
		}
	}
	r := post(t, gw.URL, "k1", chatBody)
	if r.StatusCode != 429 || r.Header.Get("Retry-After") == "" {
		t.Fatalf("expected 429 with Retry-After, got %d", r.StatusCode)
	}
	if r := post(t, gw.URL, "k2", chatBody); r.StatusCode != 200 {
		t.Fatalf("second client must have its own bucket, got %d", r.StatusCode)
	}
}

func TestValidationAndUnknownModel(t *testing.T) {
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {}))
	defer up.Close()
	gw := newGateway(t, &config.Config{Upstreams: []config.Upstream{
		{Name: "u", BaseURL: up.URL, Models: map[string]string{"fast": "m"}},
	}})

	cases := []struct {
		name, body string
		want       int
	}{
		{"bad json", `{`, 400},
		{"no model", `{"messages":[]}`, 400},
		{"no messages", `{"model":"fast"}`, 400},
		{"unknown model", `{"model":"nope","messages":[]}`, 404},
	}
	for _, c := range cases {
		r := post(t, gw.URL, "", c.body)
		r.Body.Close()
		if r.StatusCode != c.want {
			t.Errorf("%s: status %d, want %d", c.name, r.StatusCode, c.want)
		}
	}
}

func TestModelsHealthAndMetrics(t *testing.T) {
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {}))
	defer up.Close()
	gw := newGateway(t, &config.Config{Upstreams: []config.Upstream{
		{Name: "u", BaseURL: up.URL, Models: map[string]string{"fast": "m", "smart": "m2"}},
	}})

	for _, path := range []string{"/healthz", "/v1/models", "/metrics"} {
		resp, err := http.Get(gw.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		b, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode != 200 || !json.Valid(b) {
			t.Errorf("%s: status %d body %q", path, resp.StatusCode, b)
		}
		if resp.Header.Get("X-Request-Id") == "" {
			t.Errorf("%s: missing X-Request-Id", path)
		}
	}
}
