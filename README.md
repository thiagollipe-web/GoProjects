# GoProjects

Three small Go projects, each a separate module in its own folder, using only the standard library.

| Project | What it is |
|---|---|
| [`distillkit`](distillkit) | CLI that prepares teacher-generated Q&A data for fine-tuning a small model (Gemma 2 2B): cleaning, de-duplication, deterministic train/validation split, Alpaca/ShareGPT/Gemma export. Includes the Ollama `Modelfile` used to serve the model. |
| [`llm-gateway`](llm-gateway) | OpenAI-compatible HTTP gateway in front of several LLM providers (local Ollama, Groq, Maritaca): ordered fallback, circuit breaker, rate limiting, SSE streaming, metrics. |
| [`go-reasoner`](go-reasoner) | Offline symbolic conversational agent: Naive Bayes intents, forward/backward chaining with proof traces, a persistent knowledge graph and a validated Go code generator. |

Each folder has its own README, tests and `go.mod`. CI runs `gofmt`, `go vet` and `go test -race` for all three.

```bash
cd llm-gateway && go test -race ./...
```

## License

MIT
