package coder

import (
	"strings"
	"testing"
)

func TestGenerateTemplatesPassValidation(t *testing.T) {
	for _, pat := range []string{"http_handler", "worker_pool", "cli_tool", "generic_func"} {
		src, err := Generate(Request{Topic: "teste", Pattern: pat, Prefs: []string{"sempre use context"}})
		if err != nil {
			t.Fatalf("%s: %v", pat, err)
		}
		if !strings.Contains(src, "Preferência ativa: sempre use context") {
			t.Errorf("%s: preference comment missing", pat)
		}
		if err := Validate(src); err != nil {
			t.Errorf("%s: generated code failed validation: %v", pat, err)
		}
	}
}

func TestValidateRejectsBadCode(t *testing.T) {
	if err := Validate("package main\nfunc main( {"); err == nil {
		t.Error("syntax error should be rejected")
	}
	if err := Validate("package main\nimport \"os\"\nfunc main() {}\n"); err == nil {
		t.Error("unused import should be rejected")
	}
}
