# go-reasoner

A **symbolic conversational agent written in pure Go** (standard library only, zero dependencies). It runs fully offline, builds into a single static binary and was designed to run on a phone (Termux/ARM) as well as on a desktop.

It does not use a neural network or an LLM. Everything it "knows" comes from explicit components you can inspect: an intent classifier, a rule engine with a proof trace, a persistent knowledge graph and a code generator whose output is validated before being shown.

## Components

| Package | What it does |
|---|---|
| `internal/nlp` | Accent-insensitive tokenizer and light stemmer (PT/EN), Naive Bayes intent classifier with an exact-match fast path, entity/preference extraction, short-term dialog memory |
| `internal/reasoner` | Forward and backward chaining over weighted rules, with a proof trace (`/why`) |
| `internal/memory` | Persistent knowledge graph (JSON) with fuzzy search, typed nodes/edges and weight adjustment from feedback |
| `internal/nlg` | Response templates and a small Markov generator seeded from the graph |
| `internal/coder` | Template-based Go code generator; every snippet is checked with the Go AST (syntax, unused imports) and `go vet` when the toolchain is available |
| `internal/connector` | Optional lookups against Wikipedia, DuckDuckGo and the GitHub search API (plain `net/http`, with context and timeouts) |
| `cmd/reasoner` | Interactive CLI |

## How it behaves

- **Learns from feedback.** Positive/negative feedback ("perfeito", "errado") raises or lowers the weight of the last graph node and of the rules used in the last answer.
- **Explains itself.** `/why` prints the chain of rules that led to the last decision.
- **Remembers preferences.** A taught preference such as "sempre use context" becomes a weighted rule and is written into generated code as a comment.
- **Generates code.** "Escreva um handler http" derives pattern and libraries by rule chaining (`topic=http → pattern=http_handler → library=net/http, context`) and emits validated Go.

## Run

```bash
go run ./cmd/reasoner            # or: make run
```

Commands: `/help`, `/why`, `/graph`, `/rules`, `/quit`. Memory is stored in `data/memory.json` (change with `-memory`).

```bash
make test    # go test -race ./...
make build   # static binary in bin/reasoner
```

On Android with Termux: `scripts/termux-install.sh`.

## Tests

Unit tests cover the classifier (including short inputs such as "oi"), forward/backward chaining and the proof trace, graph persistence and weight clamping, and the code generator plus its validator. CI runs `gofmt`, `go vet` and `go test -race`.

## Limitations

- Intent recognition is a small Naive Bayes model trained on a seed list; unknown phrasing falls back to topic lookup or Markov text, which can be vague.
- The rule base is small and hand-written (HTTP handlers, worker pools, CLIs).
- Network connectors depend on public APIs and are best-effort.

## License

MIT
