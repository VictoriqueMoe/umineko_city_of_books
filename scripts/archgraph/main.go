package main

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/tools/go/packages"
)

const (
	dataPlaceholder = "__ARCHGRAPH_DATA__"
	maxLintWarnings = "32768"
)

type (
	graph struct {
		Module     string      `json:"module"`
		Packages   []pkgNode   `json:"packages"`
		Refs       []ref       `json:"refs"`
		Fields     []fieldEdge `json:"fields"`
		Impls      []implEdge  `json:"impls"`
		Violations []violation `json:"violations"`
		Rules      []ruleImage `json:"rules"`
	}

	ruleImage struct {
		Kind string `json:"kind"`
		File string `json:"file"`
	}
)

var (
	//go:embed viewer.html
	viewerPage []byte

	ruleGraphKinds = []string{"flow", "di"}
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "archgraph:", err)
		os.Exit(1)
	}
}

func run() error {
	archFile := flag.String("arch", ".go-arch-lint.yml", "go-arch-lint archfile, relative to the working directory")
	out := flag.String("out", "graphs/archgraph.html", "where to write the interactive page; go-arch-lint's rule graphs land beside it")
	flag.Parse()

	if !bytes.Contains(viewerPage, []byte(dataPlaceholder)) {
		return fmt.Errorf("viewer.html has no %s placeholder", dataPlaceholder)
	}

	root, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("working directory: %w", err)
	}

	outDir, err := filepath.Abs(filepath.Dir(*out))
	if err != nil {
		return fmt.Errorf("resolve %s: %w", *out, err)
	}

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("create %s: %w", outDir, err)
	}

	lint := archLint{root: root, archFile: *archFile}

	fmt.Println("mapping packages to components")
	mapping, err := lint.run("mapping", "--json")
	if err != nil {
		return err
	}

	inScope, err := parseMapping(root, mapping)
	if err != nil {
		return err
	}

	fmt.Println("checking component rules")
	check, err := lint.run("check", "--json", "--max-warnings", maxLintWarnings)
	if err != nil {
		return err
	}

	violations, deepScan, err := parseViolations(root, check)
	if err != nil {
		return err
	}

	fmt.Println("drawing the declared rule graphs")
	rules := make([]ruleImage, 0, len(ruleGraphKinds))
	for _, kind := range ruleGraphKinds {
		file, err := lint.ruleGraph(kind, outDir)
		if err != nil {
			return err
		}

		rules = append(rules, ruleImage{Kind: kind, File: file})
	}

	fmt.Println("loading and type-checking packages")
	pkgs, err := packages.Load(&packages.Config{Mode: loadMode, Dir: root}, "./...")
	if err != nil {
		return fmt.Errorf("load packages: %w", err)
	}

	g, err := extract(root, inScope, pkgs)
	if err != nil {
		return err
	}

	g.Violations = violations
	g.Rules = rules

	data, err := json.Marshal(g)
	if err != nil {
		return fmt.Errorf("encode graph: %w", err)
	}

	if err := os.WriteFile(*out, bytes.Replace(viewerPage, []byte(dataPlaceholder), data, 1), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", *out, err)
	}

	fmt.Printf("wrote %s: %d packages, %d references, %d fields, %d implementations, %d rule violations\n", *out, len(g.Packages), len(g.Refs), len(g.Fields), len(g.Impls), len(violations))
	if deepScan > 0 {
		fmt.Printf("go-arch-lint also reported %d deep-scan warnings the page does not draw; run go-arch-lint check to read them\n", deepScan)
	}

	return nil
}
