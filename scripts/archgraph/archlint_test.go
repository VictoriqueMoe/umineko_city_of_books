package main

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testModule = "umineko_city_of_books"
)

var (
	testRoot = filepath.Join("work", "repo")
)

func TestParseMapping_GroupsFilesIntoPackages(t *testing.T) {
	// given
	errorsFile := filepath.Join(testRoot, "internal", "post", "errors.go")
	body := encode(t, lintEnvelope[mappingPayload]{Payload: mappingPayload{
		ModuleName: testModule,
		MappingList: []mappingEntry{
			{FileName: filepath.Join(testRoot, "main.go"), ComponentName: "main"},
			{FileName: filepath.Join(testRoot, "internal", "post", "service.go"), ComponentName: "services"},
			{FileName: errorsFile, ComponentName: "services"},
		},
	}})

	// when
	got, err := parseMapping(testRoot, body)

	// then
	require.NoError(t, err)
	assert.Equal(t, testModule, got.module)
	assert.Equal(t, map[string]pkgNode{
		".":             {Path: ".", Component: "main", Files: 1},
		"internal/post": {Path: "internal/post", Component: "services", Files: 2},
	}, got.packages)
	assert.Contains(t, got.files, errorsFile)
}

func TestParseMapping_Rejects(t *testing.T) {
	tests := []struct {
		name string
		body []byte
		want string
	}{
		{
			name: "a package split across two components",
			body: encode(t, lintEnvelope[mappingPayload]{Payload: mappingPayload{
				ModuleName: testModule,
				MappingList: []mappingEntry{
					{FileName: filepath.Join(testRoot, "internal", "post", "service.go"), ComponentName: "services"},
					{FileName: filepath.Join(testRoot, "internal", "post", "errors.go"), ComponentName: "kernel"},
				},
			}}),
			want: "package internal/post spans components services and kernel",
		},
		{
			name: "output that is not JSON",
			body: []byte("failed to assemble project info"),
			want: "decode go-arch-lint mapping",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// when
			_, err := parseMapping(testRoot, tt.body)

			// then
			require.ErrorContains(t, err, tt.want)
		})
	}
}

func TestParseViolations_MapsWarningsToPackagePairs(t *testing.T) {
	// given
	body := encode(t, lintEnvelope[checkPayload]{Payload: checkPayload{
		ModuleName: testModule,
		ArchWarningsDeps: []dependencyWarning{
			{
				ComponentName:      "controllers",
				FileAbsolutePath:   filepath.Join(testRoot, "internal", "controllers", "art_controller.go"),
				ResolvedImportName: testModule + "/internal/dao",
				Reference:          warningReference{Line: 11},
			},
		},
		ArchWarningsDeepScan: []json.RawMessage{json.RawMessage(`{}`), json.RawMessage(`{}`)},
	}})

	// when
	violations, deepScan, err := parseViolations(testRoot, body)

	// then
	require.NoError(t, err)
	assert.Equal(t, []violation{{
		From:      "internal/controllers",
		To:        "internal/dao",
		Component: "controllers",
		Site:      site{File: "internal/controllers/art_controller.go", Line: 11},
	}}, violations)
	assert.Equal(t, 2, deepScan)
}

func TestParseViolations_Rejects(t *testing.T) {
	tests := []struct {
		name    string
		payload checkPayload
		want    string
	}{
		{
			name:    "a capped warning list",
			payload: checkPayload{ModuleName: testModule, OmittedCount: 3},
			want:    "go-arch-lint omitted 3 warnings",
		},
		{
			name: "an import outside the module",
			payload: checkPayload{
				ModuleName: testModule,
				ArchWarningsDeps: []dependencyWarning{{
					FileAbsolutePath:   filepath.Join(testRoot, "internal", "post", "service.go"),
					ResolvedImportName: "github.com/other/module",
				}},
			},
			want: "outside module",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// when
			_, _, err := parseViolations(testRoot, encode(t, lintEnvelope[checkPayload]{Payload: tt.payload}))

			// then
			require.ErrorContains(t, err, tt.want)
		})
	}
}

func TestModuleRel(t *testing.T) {
	tests := []struct {
		name         string
		importPath   string
		wantRel      string
		wantInModule bool
	}{
		{name: "the module root", importPath: testModule, wantRel: ".", wantInModule: true},
		{name: "a nested package", importPath: testModule + "/internal/post", wantRel: "internal/post", wantInModule: true},
		{name: "a module sharing the prefix", importPath: testModule + "_tools/gen", wantInModule: false},
		{name: "the standard library", importPath: "go/types", wantInModule: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// when
			rel, inModule := moduleRel(testModule, tt.importPath)

			// then
			assert.Equal(t, tt.wantInModule, inModule)
			if tt.wantInModule {
				assert.Equal(t, tt.wantRel, rel)
			}
		})
	}
}

func TestRelFile(t *testing.T) {
	tests := []struct {
		name string
		file string
		want string
	}{
		{name: "the root itself", file: testRoot, want: "."},
		{name: "a file under the root", file: filepath.Join(testRoot, "internal", "post", "service.go"), want: "internal/post/service.go"},
		{name: "a sibling sharing the prefix", file: filepath.Join("work", "repository", "a.go"), want: "work/repository/a.go"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// when
			got := relFile(testRoot, tt.file)

			// then
			assert.Equal(t, tt.want, got)
		})
	}
}

func encode(t *testing.T, value any) []byte {
	t.Helper()

	body, err := json.Marshal(value)
	require.NoError(t, err)

	return body
}
