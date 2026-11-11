// Package server exposes the OpenAI-compatible HTTP API.
package server

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/thiagollipe-web/GoProjects/llm-gateway/internal/config"
	"github.com/thiagollipe-web/GoProjects/llm-gateway/internal/ratelimit"
	"github.com/thiagollipe-web/GoProjects/llm-gateway/internal/upstream"
)

const maxBody = 1 << 20 // 1 MiB

// Server wires config, upstream pool and limiter into an http.Handler.
type Server struct {
	pool    *upstream.Pool
	limiter *ratelimit.Limiter
	keys    []string
	log     *slog.Logger
	started time.Time
}

// New builds a Server. Client keys come from the env var named in config.
func New(cfg *config.Config, pool *upstream.Pool, log *slog.Logger) *Server {
	s := &Server{pool: pool, limiter: ratelimit.New(cfg.RatePerMinute), log: log, started: time.Now()}
	if cfg.ClientKeysEnv != "" {
		for _, k := range strings.Split(os.Getenv(cfg.ClientKeysEnv), ",") {
			if k = strings.TrimSpace(k); k != "" {
				s.keys = append(s.keys, k)
			}
		}
	}
	return s
}

// Handler returns the routed handler with middleware applied.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("GET /metrics", s.metrics)
	mux.Handle("GET /v1/models", s.protected(http.HandlerFunc(s.models)))
	mux.Handle("POST /v1/chat/completions", s.protected(http.HandlerFunc(s.chat)))
	return s.withRequestID(s.withLogging(mux))
}

// ---- middleware ----------------------------------------------------------

type ctxKey string

const reqIDKey ctxKey = "reqid"

func (s *Server) withRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-Id")
		if id == "" {
			b := make([]byte, 8)
			_, _ = rand.Read(b)
			id = hex.EncodeToString(b)
		}
		w.Header().Set("X-Request-Id", id)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), reqIDKey, id)))
	})
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(c int) { w.status = c; w.ResponseWriter.WriteHeader(c) }

// Flush keeps SSE streaming working through the wrapper.
func (w *statusWriter) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (s *Server) withLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(sw, r)
		id, _ := r.Context().Value(reqIDKey).(string)
		s.log.Info("request", "id", id, "method", r.Method, "path", r.URL.Path,
			"status", sw.status, "ms", time.Since(start).Milliseconds())
	})
}

// protected enforces client API keys (if configured) and the rate limit.
func (s *Server) protected(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		client := r.RemoteAddr
		if len(s.keys) > 0 {
			tok := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
			if !s.validKey(tok) {
				writeErr(w, http.StatusUnauthorized, "invalid_api_key", "missing or invalid API key")
				return
			}
			client = "key:" + tok
		}
		if ok, wait := s.limiter.Allow(client); !ok {
			w.Header().Set("Retry-After", strconv.Itoa(int(wait.Seconds())+1))
			writeErr(w, http.StatusTooManyRequests, "rate_limited", "rate limit exceeded")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) validKey(tok string) bool {
	ok := false
	for _, k := range s.keys {
		if subtle.ConstantTimeCompare([]byte(tok), []byte(k)) == 1 {
			ok = true
		}
	}
	return ok
}

// ---- handlers ------------------------------------------------------------

func (s *Server) models(w http.ResponseWriter, _ *http.Request) {
	data := []map[string]any{}
	for _, m := range s.pool.Models() {
		data = append(data, map[string]any{"id": m, "object": "model", "owned_by": "llm-gateway"})
	}
	writeJSON(w, http.StatusOK, map[string]any{"object": "list", "data": data})
}

func (s *Server) chat(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBody)
	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid_request", "body must be valid JSON (max 1 MiB)")
		return
	}
	model, _ := body["model"].(string)
	if model == "" {
		writeErr(w, http.StatusBadRequest, "invalid_request", "\"model\" is required")
		return
	}
	if _, ok := body["messages"].([]any); !ok {
		writeErr(w, http.StatusBadRequest, "invalid_request", "\"messages\" must be an array")
		return
	}

	resp, provider, err := s.pool.Do(r.Context(), model, body)
	if err != nil {
		code := http.StatusBadGateway
		if errors.Is(err, upstream.ErrNoUpstream) {
			code = http.StatusNotFound
		}
		writeErr(w, code, "upstream_error", err.Error())
		return
	}
	defer resp.Body.Close()

	for _, h := range []string{"Content-Type", "Cache-Control"} {
		if v := resp.Header.Get(h); v != "" {
			w.Header().Set(h, v)
		}
	}
	w.Header().Set("X-Gateway-Provider", provider)
	w.WriteHeader(resp.StatusCode)
	copyStream(w, resp.Body)
}

// copyStream copies the body and flushes after each read so SSE chunks reach
// the client immediately.
func copyStream(w http.ResponseWriter, src io.Reader) {
	fl, _ := w.(http.Flusher)
	buf := make([]byte, 4096)
	for {
		n, err := src.Read(buf)
		if n > 0 {
			if _, werr := w.Write(buf[:n]); werr != nil {
				return
			}
			if fl != nil {
				fl.Flush()
			}
		}
		if err != nil {
			return
		}
	}
}

func (s *Server) metrics(w http.ResponseWriter, _ *http.Request) {
	type pm struct {
		Name     string `json:"name"`
		State    string `json:"breaker"`
		Requests int64  `json:"requests"`
		Failures int64  `json:"failures"`
	}
	var ps []pm
	for _, p := range s.pool.Providers() {
		ps = append(ps, pm{p.Name(), p.State(), p.Requests.Load(), p.Failures.Load()})
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"uptime_seconds": int(time.Since(s.started).Seconds()),
		"providers":      ps,
	})
}

// ---- helpers -------------------------------------------------------------

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, code int, typ, msg string) {
	writeJSON(w, code, map[string]any{"error": map[string]string{"type": typ, "message": msg}})
}
