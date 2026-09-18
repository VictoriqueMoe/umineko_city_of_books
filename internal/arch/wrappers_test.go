package arch

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const repoRoot = "../.."

var (
	skippedDirs = map[string]struct{}{
		".git":         {},
		".idea":        {},
		"build":        {},
		"dist":         {},
		"frontend":     {},
		"node_modules": {},
		"vendor":       {},
	}

	exemptNames = map[string]struct{}{
		"init": {},
		"main": {},
	}
)

func TestNoForwardingFunctions(t *testing.T) {
	// given
	root, err := filepath.Abs(repoRoot)
	require.NoError(t, err)

	fset := token.NewFileSet()

	// when
	var offenders []string
	walkErr := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		if entry.IsDir() {
			if _, skip := skippedDirs[entry.Name()]; skip {
				return fs.SkipDir
			}

			return nil
		}

		if !strings.HasSuffix(entry.Name(), ".go") {
			return nil
		}

		file, parseErr := parser.ParseFile(fset, path, nil, parser.ParseComments|parser.SkipObjectResolution)
		if parseErr != nil {
			return parseErr
		}

		if ast.IsGenerated(file) {
			return nil
		}

		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			rel = path
		}

		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || !forwards(fn) {
				continue
			}

			offenders = append(offenders, fmt.Sprintf("%s:%d: %s", filepath.ToSlash(rel), fset.Position(fn.Pos()).Line, fn.Name.Name))
		}

		return nil
	})
	require.NoError(t, walkErr)

	// then
	assert.Empty(t, offenders, "these package-level functions are empty or forward their own parameters straight into a single call; delete them and call the target directly:\n%s", strings.Join(offenders, "\n"))
}

func forwards(fn *ast.FuncDecl) bool {
	if fn.Recv != nil || fn.Body == nil {
		return false
	}

	if _, exempt := exemptNames[fn.Name.Name]; exempt {
		return false
	}

	if len(fn.Body.List) == 0 {
		return true
	}

	if len(fn.Body.List) != 1 {
		return false
	}

	call := onlyCall(fn.Body.List[0])
	if call == nil {
		return false
	}

	return argsAreParams(fn.Type.Params, call.Args)
}

func onlyCall(stmt ast.Stmt) *ast.CallExpr {
	switch typed := stmt.(type) {
	case *ast.ExprStmt:
		call, _ := typed.X.(*ast.CallExpr)
		return call

	case *ast.ReturnStmt:
		if len(typed.Results) != 1 {
			return nil
		}

		call, _ := typed.Results[0].(*ast.CallExpr)
		return call
	}

	return nil
}

func argsAreParams(params *ast.FieldList, args []ast.Expr) bool {
	var names []string
	if params != nil {
		for _, field := range params.List {
			if len(field.Names) == 0 {
				return false
			}

			for _, name := range field.Names {
				if name.Name == "_" {
					return false
				}

				names = append(names, name.Name)
			}
		}
	}

	if len(names) != len(args) {
		return false
	}

	for i, arg := range args {
		ident, ok := arg.(*ast.Ident)
		if !ok || ident.Name != names[i] {
			return false
		}
	}

	return true
}
