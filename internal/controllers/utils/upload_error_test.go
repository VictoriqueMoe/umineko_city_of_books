package utils

import (
	"errors"
	"fmt"
	"testing"

	"umineko_city_of_books/internal/bounds"
	"umineko_city_of_books/internal/upload"

	"github.com/stretchr/testify/assert"
)

func TestIsUploadRejection(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{name: "an oversized file, wrapped with its sizes", err: fmt.Errorf("%w: file size 60MB exceeds maximum 50MB", upload.ErrFileTooLarge), want: true},
		{name: "a file that is not an image", err: upload.ErrInvalidFileType, want: true},
		{name: "a file that is not a video", err: upload.ErrInvalidVideoType, want: true},
		{name: "a file that is not audio", err: upload.ErrInvalidAudioType, want: true},
		{name: "a file that is not an attachment type", err: upload.ErrInvalidAttachmentType, want: true},
		{name: "an image past the pixel budget", err: fmt.Errorf("%w: 90000x90000 exceeds maximum", bounds.ErrImageBounds), want: true},
		{name: "a database failure is a server fault, not a rejection", err: errors.New("pq: connection refused")},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// when
			got := IsUploadRejection(tc.err)

			// then
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestRejectionMessage(t *testing.T) {
	rejection := fmt.Errorf("save media 0: %w: file size 60MB exceeds maximum 50MB", upload.ErrFileTooLarge)
	rollback := errors.New("roll back comment 1234: delete comment: pq: connection refused")

	cases := []struct {
		name string
		err  error
		want string
	}{
		{name: "a lone rejection keeps its whole message", err: rejection, want: rejection.Error()},
		{name: "a failed rollback joined after the rejection is left out", err: errors.Join(rejection, rollback), want: rejection.Error()},
		{name: "a failed rollback joined before the rejection is left out", err: errors.Join(rollback, rejection), want: rejection.Error()},
		{name: "a wrapper around the join is looked through", err: fmt.Errorf("create comment: %w", errors.Join(rejection, rollback)), want: rejection.Error()},
		{name: "a rejection nested in a deeper join is found", err: errors.Join(rollback, errors.Join(errors.New("other"), rejection)), want: rejection.Error()},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// when
			got := rejectionMessage(tc.err)

			// then
			assert.Equal(t, tc.want, got)
			assert.NotContains(t, got, "pq:")
		})
	}
}
