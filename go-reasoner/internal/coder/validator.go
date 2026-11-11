package coder

import (
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func Validate(src string) error {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "main.go", src, parser.AllErrors)
	if err != nil {
		return fmt.Errorf("erro sintático AST: %w", err)
	}

	if err := checkUnusedImportsAST(file); err != nil {
		return err
	}

	goBin, err := exec.LookPath("go")
	if err != nil {
		return nil
	}

	dir, err := os.MkdirTemp("", "reasoner-sandbox-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)

	gomod := "module sandbox\n\ngo 1.22\n"
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(gomod), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(src), 0o644); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, goBin, "vet", "./...")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("go vet falhou: %s (%w)", strings.TrimSpace(string(out)), err)
	}
	return nil
}

func checkUnusedImportsAST(file *ast.File) error {
	imported := map[string]string{}
	for _, imp := range file.Imports {
		pathVal, err := strconv.Unquote(imp.Path.Value)
		if err != nil {
			continue
		}
		var local string
		if imp.Name != nil {
			local = imp.Name.Name
		} else {
			parts := strings.Split(pathVal, "/")
			local = parts[len(parts)-1]
		}
		if local == "_" || local == "." {
			continue
		}
		imported[local] = pathVal
	}

	used := map[string]bool{}
	ast.Inspect(file, func(n ast.Node) bool {
		sel, ok := n.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		if ident, ok := sel.X.(*ast.Ident); ok {
			used[ident.Name] = true
		}
		return true
	})

	for local, fullPath := range imported {
		if !used[local] {
			return fmt.Errorf("import %q (%s) declarado mas não utilizado", fullPath, local)
		}
	}
	return nil
}
