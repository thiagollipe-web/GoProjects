# llm-gateway

A small, dependency-free **LLM gateway written in Go**. It exposes one OpenAI-compatible API in front of several providers (a local Ollama, Groq, Maritaca, or any other OpenAI-compatible endpoint) and keeps your app working when one of them fails.

Built with the Go standard library only (`net/http`, `log/slog`, Go 1.22 routing). No external modules.

## What it does

- **One API, many providers** – `POST /v1/chat/completions` and `GET /v1/models`, OpenAI-compatible, so existing clients work by changing only the base URL.
- **Model aliases** – clients ask for `fast` or `smart`; each upstream maps the alias to its own model id.
- **Ordered fallback** – upstreams are tried in config order; network errors, `429` and `5xx` move on to the next one.
- **Circuit breaker per upstream** – after N consecutive failures a provider is skipped for a cooldown, then probed once (half-open) before returning to rotation.
- **Streaming (SSE) pass-through** – chunks are flushed to the client as they arrive.
- **Client API keys and rate limiting** – optional bearer keys, per-key token bucket, `Retry-After` on `429`.
- **Operations** – `/healthz`, `/metrics` (JSON: requests, failures, breaker state per provider), request IDs, structured JSON logs, graceful shutdown.
- **Secrets stay out of files** – provider keys are read from environment variables named in the config.

## Quick start

```bash
cp config.example.json config.json
export GROQ_API_KEY=...            # only for upstreams you enable
export GATEWAY_CLIENT_KEYS=my-key  # optional; comma-separated

go run ./cmd/gateway -config config.json
```

```bash
curl localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer my-key" \
  -H "Content-Type: application/json" \
  -d '{"model":"fast","messages":[{"role":"user","content":"Hello"}]}'
```

The response header `X-Gateway-Provider` tells you which upstream answered.

## Configuration

```jsonc
{
  "listen": ":8080",
  "client_keys_env": "GATEWAY_CLIENT_KEYS", // env var with accepted client keys; empty = open
  "rate_per_minute": 60,                    // per client, 0 = unlimited
  "breaker_failures": 3,
  "breaker_cooldown": "30s",
  "upstreams": [                            // order = priority
    {
      "name": "groq",
      "base_url": "https://api.groq.com/openai/v1",
      "api_key_env": "GROQ_API_KEY",
      "models": { "fast": "llama-3.1-8b-instant" },
      "timeout": "60s"
    }
  ]
}
```

See [`config.example.json`](config.example.json) for a three-provider setup (local Ollama first, then Groq, then Maritaca).

## Project layout

```
cmd/gateway            entry point, graceful shutdown
internal/config        JSON config, validation, defaults
internal/upstream      provider pool, fallback, circuit breaker
internal/ratelimit     per-key token bucket
internal/server        HTTP handlers and middleware
```

## Development

```bash
make test   # go test -race ./...
make vet
make build
```

Tests use `httptest` fake upstreams and cover fallback, alias rewriting, the circuit breaker, SSE streaming, authentication, rate limiting, input validation and the operational endpoints.

## Docker

```bash
docker build -t llm-gateway .
docker run -p 8080:8080 -v $PWD/config.json:/etc/gateway/config.json:ro \
  -e GROQ_API_KEY -e GATEWAY_CLIENT_KEYS llm-gateway
```

## Limitations

- Only `chat/completions` is proxied (no embeddings or other endpoints yet).
- Fallback happens before the response starts; a stream that fails midway is not retried.
- Rate limiting and breaker state are in memory, per process.

## License

MIT
