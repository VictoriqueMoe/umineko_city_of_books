package cache

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNamespaces_EveryPrefixIsUnique(t *testing.T) {
	// given
	file, err := parser.ParseFile(token.NewFileSet(), "keys.go", nil, 0)
	require.NoError(t, err)

	// when
	owners := map[string][]string{}
	ast.Inspect(file, func(node ast.Node) bool {
		spec, ok := node.(*ast.ValueSpec)
		if !ok {
			return true
		}

		for i, value := range spec.Values {
			literal, ok := value.(*ast.CompositeLit)
			if !ok || len(spec.Names) <= i {
				continue
			}

			typeName, ok := literal.Type.(*ast.Ident)
			if !ok || typeName.Name != "Namespace" {
				continue
			}

			for _, element := range literal.Elts {
				field, ok := element.(*ast.KeyValueExpr)
				if !ok {
					continue
				}

				key, ok := field.Key.(*ast.Ident)
				if !ok || key.Name != "Prefix" {
					continue
				}

				prefix, ok := field.Value.(*ast.BasicLit)
				require.True(t, ok, "%s must declare its prefix as a string literal", spec.Names[i].Name)

				unquoted, err := strconv.Unquote(prefix.Value)
				require.NoError(t, err)

				owners[unquoted] = append(owners[unquoted], spec.Names[i].Name)
			}
		}

		return true
	})

	// then
	require.NotEmpty(t, owners)
	for prefix, names := range owners {
		assert.Len(t, names, 1, "prefix %q is declared by %v; one prefix is one namespace, use Key.WithTTL for a per-entry TTL", prefix, names)
	}
}

func TestKey_WithTTLKeepsTheNameAndChangesOnlyTheTTL(t *testing.T) {
	// given
	key := LinkPreview.Key("abc")

	// when
	shorter := key.WithTTL(time.Hour)

	// then
	assert.Equal(t, key.String(), shorter.String())
	assert.Equal(t, LinkPreview.TTL, key.ttl)
	assert.Equal(t, time.Hour, shorter.ttl)
}
