// Package dataset turns a "teacher" knowledge base (question/answer articles
// collected from a large model) into clean fine-tuning datasets.
package dataset

import (
	"bufio"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"unicode"
)

// Article is one teacher answer.
type Article struct {
	Question string `json:"pergunta"`
	Area     string `json:"area"`
	Answer   string `json:"resposta"`
}

// Base is the on-disk knowledge base: {"artigos": {key: Article, ...}}.
type Base struct {
	Articles map[string]Article `json:"artigos"`
}

// LoadBase reads the JSON knowledge base.
func LoadBase(path string) (*Base, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var b Base
	if err := json.NewDecoder(f).Decode(&b); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return &b, nil
}

// markers that the teacher appends after the useful answer.
var trailingMarkers = []string{"NOVAS PERGUNTAS:", "NOVAS FRONTEIRAS DE ESTUDO:", "PRÓXIMAS DÚVIDAS:"}

// CleanAnswer cuts follow-up question sections and trims whitespace.
func CleanAnswer(s string) string {
	for _, m := range trailingMarkers {
		if i := strings.Index(s, m); i >= 0 {
			s = s[:i]
		}
	}
	return strings.TrimSpace(s)
}

// NormalizeQuestion makes questions comparable for de-duplication.
func NormalizeQuestion(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
		case unicode.IsSpace(r):
			b.WriteRune(' ')
		}
	}
	return strings.Join(strings.Fields(b.String()), " ")
}

// Example is a cleaned training pair.
type Example struct {
	Instruction string
	Area        string
	Output      string
}

// Options controls Build.
type Options struct {
	MinAnswerChars int // answers shorter than this are dropped (default 30)
}

// Report counts what happened to the input.
type Report struct {
	Total, Kept, TooShort, Duplicates int
	PerArea                           map[string]int
	AvgChars                          int
}

// Build cleans, filters and de-duplicates. Output order is deterministic.
func Build(b *Base, o Options) ([]Example, Report) {
	if o.MinAnswerChars <= 0 {
		o.MinAnswerChars = 30
	}
	keys := make([]string, 0, len(b.Articles))
	for k := range b.Articles {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	rep := Report{Total: len(keys), PerArea: map[string]int{}}
	seen := map[string]bool{}
	var out []Example
	var chars int
	for _, k := range keys {
		a := b.Articles[k]
		q := strings.TrimSpace(a.Question)
		if q == "" {
			q = k
		}
		ans := CleanAnswer(a.Answer)
		if len([]rune(ans)) < o.MinAnswerChars {
			rep.TooShort++
			continue
		}
		nq := NormalizeQuestion(q)
		if seen[nq] {
			rep.Duplicates++
			continue
		}
		seen[nq] = true
		area := a.Area
		if area == "" {
			area = "Ciência Universal"
		}
		out = append(out, Example{Instruction: q, Area: area, Output: ans})
		rep.PerArea[area]++
		chars += len([]rune(ans))
	}
	rep.Kept = len(out)
	if rep.Kept > 0 {
		rep.AvgChars = chars / rep.Kept
	}
	return out, rep
}

// IsValidation deterministically assigns an example to the validation split
// from a hash of its question, so reruns never move examples between splits.
func IsValidation(question string, fraction float64) bool {
	if fraction <= 0 {
		return false
	}
	h := sha256.Sum256([]byte(NormalizeQuestion(question)))
	v := float64(binary.BigEndian.Uint32(h[:4])) / float64(1<<32)
	return v < fraction
}

// Write serialises one example as a JSON line in the chosen format.
func Write(w io.Writer, e Example, format string) error {
	var rec any
	switch format {
	case "alpaca":
		rec = map[string]string{"instruction": e.Instruction, "input": "Disciplina: " + e.Area, "output": e.Output}
	case "sharegpt":
		rec = map[string]any{"conversations": []map[string]string{
			{"from": "human", "value": e.Instruction + "\n\nDisciplina: " + e.Area},
			{"from": "gpt", "value": e.Output},
		}}
	case "gemma":
		// Gemma 2 chat template; the model turn closes with <end_of_turn>.
		rec = map[string]string{"text": "<start_of_turn>user\n" + e.Instruction +
			"\n\nDisciplina: " + e.Area + "<end_of_turn>\n<start_of_turn>model\n" + e.Output + "<end_of_turn>\n"}
	default:
		return fmt.Errorf("unknown format %q (use alpaca, sharegpt or gemma)", format)
	}
	enc := json.NewEncoder(w) // Encode appends the newline
	enc.SetEscapeHTML(false)  // keep <start_of_turn> readable
	return enc.Encode(rec)
}

// Export writes train/validation files and returns how many went to each.
func Export(ex []Example, format, trainPath, valPath string, valFraction float64) (train, val int, err error) {
	tf, err := os.Create(trainPath)
	if err != nil {
		return 0, 0, err
	}
	defer tf.Close()
	tw := bufio.NewWriter(tf)
	var vw *bufio.Writer
	if valFraction > 0 {
		vf, err := os.Create(valPath)
		if err != nil {
			return 0, 0, err
		}
		defer vf.Close()
		vw = bufio.NewWriter(vf)
		defer vw.Flush()
	}
	for _, e := range ex {
		if vw != nil && IsValidation(e.Instruction, valFraction) {
			if err := Write(vw, e, format); err != nil {
				return 0, 0, err
			}
			val++
			continue
		}
		if err := Write(tw, e, format); err != nil {
			return 0, 0, err
		}
		train++
	}
	return train, val, tw.Flush()
}
