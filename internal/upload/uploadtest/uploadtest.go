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

func MemoryUpload(contentType string, filename string, data []byte) upload.FileUpload {
	return upload.FileUpload{
		ContentType: contentType,
		Filename:    filename,
		Size:        int64(len(data)),
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

func PNGUpload(t *testing.T, filename string) upload.FileUpload {
	t.Helper()

	return MemoryUpload("image/png", filename, PNG(t))
}

func ExpectMediaLimits(settingsSvc *settings.MockService, maxImageSize int) {
	settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxImageSize).Return(maxImageSize).Maybe()
	settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxImagePixels).Return(bounds.FallbackMaxImagePixels).Maybe()
	settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxQueuedUploadsPerUser).Return(1).Maybe()
	settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxWaitingUploadsPerUser).Return(1).Maybe()
}
