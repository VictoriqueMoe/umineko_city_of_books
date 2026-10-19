package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

type (
	archLint struct {
		root     string
		archFile string
	}

	lintEnvelope[T any] struct {
		Payload T `json:"Payload"`
	}

	mappingPayload struct {
		ModuleName  string         `json:"ModuleName"`
		MappingList []mappingEntry `json:"MappingList"`
	}

	mappingEntry struct {
		FileName      string `json:"FileName"`
		ComponentName string `json:"ComponentName"`
	}

	checkPayload struct {
		ModuleName           string              `json:"ModuleName"`
		ArchWarningsDeps     []dependencyWarning `json:"ArchWarningsDeps"`
		ArchWarningsDeepScan []json.RawMessage   `json:"ArchWarningsDeepScan"`
		OmittedCount         int                 `json:"OmittedCount"`
	}

	dependencyWarning struct {
		ComponentName      string           `json:"ComponentName"`
		FileAbsolutePath   string           `json:"FileAbsolutePath"`
		ResolvedImportName string           `json:"ResolvedImportName"`
		Reference          warningReference `json:"Reference"`
	}

	warningReference struct {
		Line int `json:"Line"`
	}

	scope struct {
		module   string
		packages map[string]pkgNode
		files    map[string]struct{}
	}

	pkgNode struct {
		Path      string `json:"path"`
		Component string `json:"component"`
		Files     int    `json:"files"`
	}

	violation struct {
		From      string `json:"from"`
		To        string `json:"to"`
		Component string `json:"component"`
		Site      site   `json:"site"`
	}

	site struct {
		File string `json:"file"`
		Line int    `json:"line"`
	}
)

func (l archLint) run(args ...string) ([]byte, error) {
	cmd := exec.Command("go-arch-lint", append(args, "--arch-file", l.archFile, "--project-path", l.root)...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if exitErr, ok := errors.AsType[*exec.ExitError](err); ok && exitErr.ExitCode() == 1 && json.Valid(stdout.Bytes()) {
		return stdout.Bytes(), nil
	}

	if err != nil {
		return nil, fmt.Errorf("go-arch-lint %s: %w: %s", args[0], err, strings.TrimSpace(stdout.String()+stderr.String()))
	}

	return stdout.Bytes(), nil
}

func (l archLint) ruleGraph(kind string, outDir string) (string, error) {
	name := "rules-" + kind + ".svg"

	if _, err := l.run("graph", "--type", kind, "--out", filepath.Join(outDir, name)); err != nil {
		return "", err
	}

	return name, nil
}

func parseMapping(root string, body []byte) (scope, error) {
	var envelope lintEnvelope[mappingPayload]
	if err := json.Unmarshal(body, &envelope); err != nil {
		return scope{}, fmt.Errorf("decode go-arch-lint mapping: %w", err)
	}

	result := scope{
		module:   envelope.Payload.ModuleName,
		packages: map[string]pkgNode{},
		files:    map[string]struct{}{},
	}

	for _, entry := range envelope.Payload.MappingList {
		dir := relFile(root, filepath.Dir(entry.FileName))

		node, seen := result.packages[dir]
		if seen && node.Component != entry.ComponentName {
			return scope{}, fmt.Errorf("package %s spans components %s and %s", dir, node.Component, entry.ComponentName)
		}

		result.packages[dir] = pkgNode{Path: dir, Component: entry.ComponentName, Files: node.Files + 1}
		result.files[filepath.Clean(entry.FileName)] = struct{}{}
	}

	return result, nil
}

func parseViolations(root string, body []byte) ([]violation, int, error) {
	var envelope lintEnvelope[checkPayload]
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, 0, fmt.Errorf("decode go-arch-lint check: %w", err)
	}

	payload := envelope.Payload
	if payload.OmittedCount > 0 {
		return nil, 0, fmt.Errorf("go-arch-lint omitted %d warnings; raise --max-warnings", payload.OmittedCount)
	}

	violations := make([]violation, 0, len(payload.ArchWarningsDeps))
	for _, warning := range payload.ArchWarningsDeps {
		to, inModule := moduleRel(payload.ModuleName, warning.ResolvedImportName)
		if !inModule {
			return nil, 0, fmt.Errorf("go-arch-lint flagged %s, which is outside module %s", warning.ResolvedImportName, payload.ModuleName)
		}

		violations = append(violations, violation{
			From:      relFile(root, filepath.Dir(warning.FileAbsolutePath)),
			To:        to,
			Component: warning.ComponentName,
			Site:      site{File: relFile(root, warning.FileAbsolutePath), Line: warning.Reference.Line},
		})
	}

	return violations, len(payload.ArchWarningsDeepScan), nil
}

func moduleRel(module string, importPath string) (string, bool) {
	if importPath == module {
		return ".", true
	}

	return strings.CutPrefix(importPath, module+"/")
}

func relFile(root string, file string) string {
	slashed := filepath.ToSlash(filepath.Clean(file))
	base := filepath.ToSlash(filepath.Clean(root))
	if slashed == base {
		return "."
	}

	rel, inRoot := strings.CutPrefix(slashed, base+"/")
	if !inRoot {
		return slashed
	}

	return rel
}
