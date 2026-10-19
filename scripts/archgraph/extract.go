package main

import (
	"cmp"
	"errors"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"maps"
	"path/filepath"
	"slices"

	"golang.org/x/tools/go/packages"
)

const (
	loadMode = packages.NeedName | packages.NeedFiles | packages.NeedSyntax | packages.NeedTypes | packages.NeedTypesInfo
)

type (
	extractor struct {
		root   string
		scope  scope
		refs   map[refKey][]site
		fields []fieldEdge
		impls  []implEdge
	}

	refKey struct {
		From     string `json:"from"`
		FromType string `json:"fromType,omitempty"`
		FromFunc string `json:"fromFunc,omitempty"`
		To       string `json:"to"`
		ToType   string `json:"toType,omitempty"`
		Name     string `json:"name"`
		Kind     string `json:"kind"`
	}

	ref struct {
		refKey
		Sites []site `json:"sites"`
	}

	fieldEdge struct {
		Pkg      string `json:"pkg"`
		Type     string `json:"type"`
		Field    string `json:"field"`
		ToPkg    string `json:"toPkg"`
		ToType   string `json:"toType"`
		Embedded bool   `json:"embedded,omitzero"`
		Site     site   `json:"site"`
	}

	implEdge struct {
		Pkg   string `json:"pkg"`
		Iface string `json:"iface"`
		Impl  string `json:"impl"`
	}
)

var (
	selectionKinds = map[types.SelectionKind]string{
		types.FieldVal:   "field",
		types.MethodVal:  "method",
		types.MethodExpr: "method",
	}
)

func extract(root string, inScope scope, pkgs []*packages.Package) (graph, error) {
	e := &extractor{root: root, scope: inScope, refs: map[refKey][]site{}}

	var errs []error
	for _, pkg := range pkgs {
		if err := e.visitPackage(pkg); err != nil {
			errs = append(errs, err)
		}
	}

	if err := errors.Join(errs...); err != nil {
		return graph{}, err
	}

	return graph{
		Module: inScope.module,
		Packages: slices.SortedFunc(maps.Values(inScope.packages), func(a, b pkgNode) int {
			return cmp.Compare(a.Path, b.Path)
		}),
		Refs:   e.sortedRefs(),
		Fields: e.fields,
		Impls:  e.impls,
	}, nil
}

func (e *extractor) visitPackage(pkg *packages.Package) error {
	from, inModule := moduleRel(e.scope.module, pkg.PkgPath)
	if !inModule {
		return nil
	}

	if _, inScope := e.scope.packages[from]; !inScope {
		return nil
	}

	if len(pkg.Errors) > 0 {
		errs := make([]error, 0, len(pkg.Errors))
		for _, pkgErr := range pkg.Errors {
			errs = append(errs, pkgErr)
		}

		return fmt.Errorf("type-check %s: %w", pkg.PkgPath, errors.Join(errs...))
	}

	var declared []*types.TypeName
	for _, file := range pkg.Syntax {
		if _, inScope := e.scope.files[filepath.Clean(pkg.Fset.File(file.Pos()).Name())]; !inScope {
			continue
		}

		declared = append(declared, e.visitFile(pkg, from, file)...)
	}

	e.collectImpls(from, declared)

	return nil
}

func (e *extractor) visitFile(pkg *packages.Package, from string, file *ast.File) []*types.TypeName {
	var declared []*types.TypeName
	for _, decl := range file.Decls {
		switch typed := decl.(type) {
		case *ast.FuncDecl:
			e.visitNode(pkg, refKey{From: from, FromType: receiverName(pkg.TypesInfo, typed), FromFunc: typed.Name.Name}, typed)

		case *ast.GenDecl:
			declared = append(declared, e.visitGenDecl(pkg, from, typed)...)
		}
	}

	return declared
}

func (e *extractor) visitGenDecl(pkg *packages.Package, from string, decl *ast.GenDecl) []*types.TypeName {
	var declared []*types.TypeName
	for _, spec := range decl.Specs {
		switch typed := spec.(type) {
		case *ast.TypeSpec:
			e.visitNode(pkg, refKey{From: from, FromType: typed.Name.Name}, typed)

			obj, ok := pkg.TypesInfo.Defs[typed.Name].(*types.TypeName)
			if ok && !obj.IsAlias() {
				declared = append(declared, obj)
				e.collectFields(pkg, from, obj)
			}

		case *ast.ValueSpec:
			e.visitNode(pkg, refKey{From: from}, typed)
		}
	}

	return declared
}

func (e *extractor) visitNode(pkg *packages.Package, caller refKey, node ast.Node) {
	for child := range ast.Preorder(node) {
		sel, ok := child.(*ast.SelectorExpr)
		if !ok {
			continue
		}

		key := caller
		key.Name = sel.Sel.Name

		if selection, ok := pkg.TypesInfo.Selections[sel]; ok {
			named := namedOf(selection.Recv())
			if named == nil {
				continue
			}

			key.ToType = named.Obj().Name()
			key.Kind = selectionKinds[selection.Kind()]
			e.record(pkg, key, named.Obj().Pkg(), sel.Sel.Pos())

			continue
		}

		obj := pkg.TypesInfo.Uses[sel.Sel]
		if obj == nil {
			continue
		}

		key.Kind = objectKind(obj)
		e.record(pkg, key, obj.Pkg(), sel.Sel.Pos())
	}
}

func (e *extractor) record(pkg *packages.Package, key refKey, target *types.Package, pos token.Pos) {
	to, ok := e.target(target, key.From)
	if !ok {
		return
	}

	key.To = to
	e.refs[key] = append(e.refs[key], e.site(pkg.Fset, pos))
}

func (e *extractor) collectFields(pkg *packages.Package, from string, owner *types.TypeName) {
	st, ok := owner.Type().Underlying().(*types.Struct)
	if !ok {
		return
	}

	for field := range st.Fields() {
		held := namedOf(field.Type())
		if held == nil {
			continue
		}

		to, ok := e.target(held.Obj().Pkg(), from)
		if !ok {
			continue
		}

		e.fields = append(e.fields, fieldEdge{
			Pkg:      from,
			Type:     owner.Name(),
			Field:    field.Name(),
			ToPkg:    to,
			ToType:   held.Obj().Name(),
			Embedded: field.Embedded(),
			Site:     e.site(pkg.Fset, field.Pos()),
		})
	}
}

func (e *extractor) collectImpls(from string, declared []*types.TypeName) {
	for _, iface := range declared {
		methods, ok := iface.Type().Underlying().(*types.Interface)
		if !ok || methods.NumMethods() == 0 || !methods.IsMethodSet() {
			continue
		}

		for _, impl := range declared {
			if types.IsInterface(impl.Type()) || !types.Implements(types.NewPointer(impl.Type()), methods) {
				continue
			}

			e.impls = append(e.impls, implEdge{Pkg: from, Iface: iface.Name(), Impl: impl.Name()})
		}
	}
}

func (e *extractor) target(pkg *types.Package, from string) (string, bool) {
	if pkg == nil {
		return "", false
	}

	to, inModule := moduleRel(e.scope.module, pkg.Path())
	if !inModule || to == from {
		return "", false
	}

	_, inScope := e.scope.packages[to]

	return to, inScope
}

func (e *extractor) site(fset *token.FileSet, pos token.Pos) site {
	position := fset.Position(pos)

	return site{File: relFile(e.root, position.Filename), Line: position.Line}
}

func (e *extractor) sortedRefs() []ref {
	refs := make([]ref, 0, len(e.refs))
	for key, sites := range e.refs {
		slices.SortFunc(sites, func(a, b site) int {
			return cmp.Or(cmp.Compare(a.File, b.File), cmp.Compare(a.Line, b.Line))
		})

		refs = append(refs, ref{refKey: key, Sites: sites})
	}

	slices.SortFunc(refs, func(a, b ref) int {
		return cmp.Or(
			cmp.Compare(a.From, b.From),
			cmp.Compare(a.FromType, b.FromType),
			cmp.Compare(a.FromFunc, b.FromFunc),
			cmp.Compare(a.To, b.To),
			cmp.Compare(a.ToType, b.ToType),
			cmp.Compare(a.Name, b.Name),
			cmp.Compare(a.Kind, b.Kind),
		)
	})

	return refs
}

func receiverName(info *types.Info, fn *ast.FuncDecl) string {
	method, ok := info.Defs[fn.Name].(*types.Func)
	if !ok || method.Signature().Recv() == nil {
		return ""
	}

	named := namedOf(method.Signature().Recv().Type())
	if named == nil {
		return ""
	}

	return named.Obj().Name()
}

func namedOf(t types.Type) *types.Named {
	for {
		switch typed := types.Unalias(t).(type) {
		case *types.Pointer:
			t = typed.Elem()

		case *types.Slice:
			t = typed.Elem()

		case *types.Array:
			t = typed.Elem()

		case *types.Map:
			t = typed.Elem()

		case *types.Chan:
			t = typed.Elem()

		case *types.Named:
			return typed.Origin()

		default:
			return nil
		}
	}
}

func objectKind(obj types.Object) string {
	switch obj.(type) {
	case *types.Func:
		return "func"

	case *types.TypeName:
		return "type"

	case *types.Const:
		return "const"

	case *types.Var:
		return "var"

	default:
		return "other"
	}
}
