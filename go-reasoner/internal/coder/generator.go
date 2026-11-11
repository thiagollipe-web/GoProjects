package coder

import (
	"bytes"
	"fmt"
	"go/format"
	"strings"
	"text/template"
)

type Request struct {
	Topic        string
	Library      string
	Pattern      string
	Prefs        []string
	Replacements map[string]string
}

var templates = map[string]string{
	"http_handler": `package main

{{.PrefComments}}import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"
)

type Response struct {
	Topic     string ` + "`json:\"topic\"`" + `
	Status    string ` + "`json:\"status\"`" + `
	Timestamp string ` + "`json:\"timestamp\"`" + `
}

func Handler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	_ = ctx

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	resp := Response{
		Topic:     {{printf "%q" .Topic}},
		Status:    "ok",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", Handler)
	srv := &http.Server{
		Addr:         ":8080",
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	log.Fatal(srv.ListenAndServe())
}
`,
	"worker_pool": `package main

{{.PrefComments}}import (
	"context"
	"fmt"
	"sync"
)

type Job struct {
	ID      int
	Payload string
}

type Result struct {
	JobID  int
	Output string
}

func RunWorkerPool(ctx context.Context, numWorkers int, jobs <-chan Job) <-chan Result {
	results := make(chan Result)
	var wg sync.WaitGroup

	worker := func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case job, ok := <-jobs:
				if !ok {
					return
				}
				results <- Result{
					JobID:  job.ID,
					Output: fmt.Sprintf("processed [%s]: %s", {{printf "%q" .Topic}}, job.Payload),
				}
			}
		}
	}

	wg.Add(numWorkers)
	for i := 0; i < numWorkers; i++ {
		go worker()
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	return results
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	jobs := make(chan Job, 4)
	for i := 1; i <= 4; i++ {
		jobs <- Job{ID: i, Payload: fmt.Sprintf("task-%d", i)}
	}
	close(jobs)

	for res := range RunWorkerPool(ctx, 2, jobs) {
		fmt.Println(res.Output)
	}
}
`,
	"cli_tool": `package main

{{.PrefComments}}import (
	"flag"
	"fmt"
	"os"
)

func main() {
	input := flag.String("input", {{printf "%q" .Topic}}, "parâmetro de entrada")
	verbose := flag.Bool("v", false, "modo detalhado")
	flag.Parse()

	if *input == "" {
		fmt.Fprintln(os.Stderr, "erro: -input é obrigatório")
		os.Exit(1)
	}
	if *verbose {
		fmt.Fprintf(os.Stderr, "[debug] executando tópico: %s\n", *input)
	}
	fmt.Printf("Resultado para %s: concluído com sucesso\n", *input)
}
`,
	"generic_func": `package main

{{.PrefComments}}import "fmt"

func Execute(input string) (string, error) {
	if input == "" {
		return "", fmt.Errorf("entrada vazia para %s", {{printf "%q" .Topic}})
	}
	return fmt.Sprintf("[%s] processado: %s", {{printf "%q" .Topic}}, input), nil
}

func main() {
	out, err := Execute("exemplo")
	if err != nil {
		panic(err)
	}
	fmt.Println(out)
}
`,
}

type tmplData struct {
	Topic        string
	Library      string
	PrefComments string
}

func Generate(req Request) (string, error) {
	pat := req.Pattern
	if pat == "" {
		pat = inferPatternFromTopic(req.Topic)
	}
	rawTmpl, ok := templates[pat]
	if !ok {
		rawTmpl = templates["generic_func"]
	}

	topic := strings.TrimSpace(req.Topic)
	if topic == "" {
		topic = "operação padrão"
	}

	var prefBuilder strings.Builder
	for _, p := range req.Prefs {
		fmt.Fprintf(&prefBuilder, "// Preferência ativa: %s\n", p)
	}
	if prefBuilder.Len() > 0 {
		prefBuilder.WriteString("\n")
	}

	t, err := template.New(pat).Parse(rawTmpl)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	err = t.Execute(&buf, tmplData{
		Topic:        topic,
		Library:      req.Library,
		PrefComments: prefBuilder.String(),
	})
	if err != nil {
		return "", err
	}

	code := buf.String()
	for oldPkg, newPkg := range req.Replacements {
		if oldPkg != "" && newPkg != "" {
			code = strings.ReplaceAll(code, oldPkg, newPkg)
		}
	}

	formatted, err := format.Source([]byte(code))
	if err != nil {
		return code, fmt.Errorf("go/format: %w", err)
	}
	return string(formatted), nil
}

func inferPatternFromTopic(topic string) string {
	low := strings.ToLower(topic)
	switch {
	case strings.Contains(low, "http") || strings.Contains(low, "handler") || strings.Contains(low, "server") || strings.Contains(low, "api"):
		return "http_handler"
	case strings.Contains(low, "worker") || strings.Contains(low, "pool") || strings.Contains(low, "concurren") || strings.Contains(low, "goroutine"):
		return "worker_pool"
	case strings.Contains(low, "cli") || strings.Contains(low, "flag") || strings.Contains(low, "command"):
		return "cli_tool"
	default:
		return "generic_func"
	}
}
