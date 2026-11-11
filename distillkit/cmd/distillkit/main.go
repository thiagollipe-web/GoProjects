// Command distillkit prepares teacher-generated Q&A data for fine-tuning a small model.
package main

import (
	"flag"
	"fmt"
	"os"
	"sort"

	"github.com/thiagollipe-web/GoProjects/distillkit/internal/dataset"
)

const usage = `distillkit - prepare a teacher knowledge base for fine-tuning

Usage:
  distillkit stats  -in base.json
  distillkit export -in base.json -format alpaca|sharegpt|gemma [-val 0.05] [-train train.jsonl] [-valout val.jsonl] [-min 30]
`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
	cmd, args := os.Args[1], os.Args[2:]
	fs := flag.NewFlagSet(cmd, flag.ExitOnError)
	in := fs.String("in", "", "knowledge base JSON")
	min := fs.Int("min", 30, "minimum answer length in characters")
	format := fs.String("format", "alpaca", "alpaca, sharegpt or gemma")
	val := fs.Float64("val", 0.05, "validation fraction (0 disables)")
	trainOut := fs.String("train", "train.jsonl", "training output file")
	valOut := fs.String("valout", "val.jsonl", "validation output file")
	_ = fs.Parse(args)

	if *in == "" {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
	base, err := dataset.LoadBase(*in)
	if err != nil {
		fail(err)
	}
	ex, rep := dataset.Build(base, dataset.Options{MinAnswerChars: *min})

	switch cmd {
	case "stats":
		printReport(rep)
	case "export":
		tr, va, err := dataset.Export(ex, *format, *trainOut, *valOut, *val)
		if err != nil {
			fail(err)
		}
		printReport(rep)
		fmt.Printf("\nformat=%s  train=%d (%s)  validation=%d (%s)\n", *format, tr, *trainOut, va, *valOut)
	default:
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
}

func printReport(r dataset.Report) {
	fmt.Printf("total=%d kept=%d too_short=%d duplicates=%d avg_answer_chars=%d\n",
		r.Total, r.Kept, r.TooShort, r.Duplicates, r.AvgChars)
	areas := make([]string, 0, len(r.PerArea))
	for a := range r.PerArea {
		areas = append(areas, a)
	}
	sort.Slice(areas, func(i, j int) bool { return r.PerArea[areas[i]] > r.PerArea[areas[j]] })
	for _, a := range areas {
		fmt.Printf("  %4d  %s\n", r.PerArea[a], a)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}
