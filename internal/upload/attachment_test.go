package upload

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAllowedAttachmentTypes(t *testing.T) {
	// given
	tests := []struct {
		name        string
		sniffedType string
		wantExt     string
		wantAllowed bool
	}{
		{name: "pdf keeps its extension", sniffedType: "application/pdf", wantExt: ".pdf", wantAllowed: true},
		{name: "plain text becomes txt", sniffedType: "text/plain", wantExt: ".txt", wantAllowed: true},
		{name: "docx arrives sniffed as zip", sniffedType: "application/zip", wantExt: ".docx", wantAllowed: true},
		{name: "html is rejected", sniffedType: "text/html"},
		{name: "svg is rejected", sniffedType: "image/svg+xml"},
		{name: "unrecognised binary is rejected", sniffedType: "application/octet-stream"},
		{name: "empty type is rejected", sniffedType: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// when
			ext, ok := AllowedAttachmentTypes[NormaliseSniffedType(tt.sniffedType)]

			// then
			assert.Equal(t, tt.wantAllowed, ok)
			assert.Equal(t, tt.wantExt, ext)
		})
	}
}
