package upload

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateAttachmentFile(t *testing.T) {
	pdf := []byte("%PDF-1.4\n%case notes\n")

	tests := []struct {
		name    string
		file    FileUpload
		maxSize int64
		wantErr error
	}{
		{name: "a pdf within the limit passes", file: memoryUpload("application/pdf", "notes.pdf", pdf, false), maxSize: 1024},
		{name: "a file over the limit is refused before it is read", file: memoryUpload("application/pdf", "notes.pdf", pdf, false), maxSize: 1, wantErr: ErrFileTooLarge},
		{name: "html is refused whatever it claims to be", file: memoryUpload("application/pdf", "page.pdf", []byte("<!DOCTYPE html><html><body>hi</body></html>"), false), maxSize: 1024, wantErr: ErrInvalidAttachmentType},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// when
			err := ValidateAttachmentFile(tc.file, tc.maxSize)

			// then
			if tc.wantErr == nil {
				require.NoError(t, err)
				return
			}

			require.ErrorIs(t, err, tc.wantErr)
		})
	}
}
