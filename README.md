<p align="center"><img src="docs/banner.svg" alt="GoProjects" width="100%"></p>

<p align="center">
  <a href="https://github.com/thiagollipe-web/GoProjects/actions/workflows/ci.yml"><img alt="CI" src="https://github.com/thiagollipe-web/GoProjects/actions/workflows/ci.yml/badge.svg"></a>
  <img alt="Go 1.22+" src="https://img.shields.io/badge/Go-1.22%2B-00ADD8?logo=go&logoColor=white">
  <img alt="Dependencies" src="https://img.shields.io/badge/dependencies-stdlib%20only-2ea44f">
  <img alt="License MIT" src="https://img.shields.io/badge/license-MIT-blue">
</p>

Three self-directed Go projects, each its own module with its own tests, built with the **standard library only**. They cover the path from preparing data for a small language model, to serving models behind one API, to a fully offline reasoning agent.

## Projects

| Project | What it is | Highlights |
|---|---|---|
| [**llm-gateway**](llm-gateway) | OpenAI-compatible HTTP gateway in front of several LLM providers (local Ollama, Groq, Maritaca) | Ordered fallback · circuit breaker · per-client rate limit · SSE streaming · `/metrics` |
| [**distillkit**](distillkit) | CLI that turns teacher-model Q&A into clean fine-tuning datasets for a small model (Gemma 2 2B) | Cleaning · de-duplication · deterministic train/val split · Alpaca, ShareGPT and Gemma formats |
| [**go-reasoner**](go-reasoner) | Offline symbolic conversational agent, no neural network | Intent classifier · forward/backward chaining with proof trace · knowledge graph · validated code generation |

## How they fit together

```mermaid
flowchart LR
    T[Teacher model<br/>large LLM] -->|Q&A answers| K[(Knowledge base<br/>JSON)]
    K --> D[distillkit<br/>clean, dedupe, split, export]
    D -->|train.jsonl / val.jsonl| F[Fine-tune small model<br/>Gemma 2 2B]
    F -->|GGUF + Modelfile| O[Ollama]
    O --> G[llm-gateway<br/>one OpenAI-style API]
    P[Cloud providers<br/>Groq, Maritaca] --> G
    G --> A[Applications]
    R[go-reasoner<br/>offline symbolic agent]:::solo
    classDef solo stroke-dasharray: 4 3
```

`go-reasoner` is independent: it runs fully offline and does not call any model.

## Repository layout

```
GoProjects/
├── llm-gateway/      cmd/ internal/{config,upstream,ratelimit,server}
├── distillkit/       cmd/ internal/dataset  deploy/Modelfile
├── go-reasoner/      cmd/ internal/{nlp,nlg,memory,reasoner,coder,connector}
├── docs/             banner and assets
├── go.work           lets all three modules be developed together
├── Makefile          test / vet / fmt / build for every project
└── .github/workflows CI (gofmt, go vet, go test -race) per project
```

## Quick start

```bash
git clone https://github.com/thiagollipe-web/GoProjects && cd GoProjects

make test          # run every test suite with the race detector
make build         # binaries in ./bin
make help          # list all targets
```

Each project can also be used on its own:

```bash
cd llm-gateway && go run ./cmd/gateway -config config.example.json
cd distillkit  && go run ./cmd/distillkit stats -in base.json
cd go-reasoner && go run ./cmd/reasoner
```

## Engineering notes

- **No third-party modules.** Everything builds with a stock Go toolchain, including on Android (Termux).
- **Tested.** Unit tests use `httptest` fake upstreams, temporary directories and deterministic inputs; CI runs `gofmt`, `go vet` and `go test -race`.
- **Secrets stay out of files.** Provider keys are read from environment variables named in the config.
- **Honest scope.** Each README ends with a *Limitations* or *Scope* section.

## License

[MIT](LICENSE) © Thiago Fillipe Soares
