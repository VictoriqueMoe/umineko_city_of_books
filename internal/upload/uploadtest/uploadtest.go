package uploadtest

import (
	"bytes"
	"image"
	"image/png"
	"io"
	"testing"

	"umineko_city_of_books/internal/bounds"
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/settings"
	"umineko_city_of_books/internal/upload"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func MemoryUpload(contentType string, filename string, data []byte, isSpoiler bool) upload.FileUpload {
	return upload.FileUpload{
		ContentType: contentType,
		Filename:    filename,
		Size:        int64(len(data)),
		IsSpoiler:   isSpoiler,
		Open: func() (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewReader(data)), nil
		},
	}
}

func PNG(t *testing.T) []byte {
	t.Helper()

	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 2, 2))))

	return buf.Bytes()
}

func PNGUpload(t *testing.T, filename string, isSpoiler bool) upload.FileUpload {
	t.Helper()

	return MemoryUpload("image/png", filename, PNG(t), isSpoiler)
}

func Files(filenames ...string) any {
	return mock.MatchedBy(func(files []upload.FileUpload) bool {
		if len(files) != len(filenames) {
			return false
		}

		for i, f := range files {
			if f.Filename != filenames[i] {
				return false
			}
		}

		return true
	})
}

func ExpectNoMedia(uploadSvc *upload.MockService) {
	noFiles := mock.MatchedBy(func(files []upload.FileUpload) bool {
		return len(files) == 0
	})
	uploadSvc.EXPECT().StoreMedia(mock.Anything, mock.Anything, noFiles).Return(nil, nil).Maybe()
	uploadSvc.EXPECT().TranscodeMedia(mock.Anything, mock.Anything, []upload.StoredMedia(nil), mock.Anything, mock.Anything, mock.Anything).Return().Maybe()
	uploadSvc.EXPECT().DiscardMedia().Return().Maybe()
}

func ExpectMediaLimits(settingsSvc *settings.MockService, maxImageSize int) {
	settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxImageSize).Return(maxImageSize).Maybe()
	settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxImagePixels).Return(bounds.FallbackMaxImagePixels).Maybe()
}
