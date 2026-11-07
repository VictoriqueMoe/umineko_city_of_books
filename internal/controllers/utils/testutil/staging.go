package testutil

import (
	"testing"

	"umineko_city_of_books/internal/config"
	settingssvc "umineko_city_of_books/internal/settings"
	"umineko_city_of_books/internal/upload"

	"github.com/stretchr/testify/mock"
)

func NewStaging(t *testing.T) upload.Service {
	t.Helper()

	settingsSvc := settingssvc.NewMockService(t)
	settingsSvc.EXPECT().Get(mock.Anything, config.SettingUploadDir).Return(t.TempDir()).Maybe()

	return upload.NewService(settingsSvc, nil, nil)
}

func FileUploadOf(contentType string, filename string, isSpoiler bool) any {
	return mock.MatchedBy(func(f upload.FileUpload) bool {
		return f.ContentType == contentType && f.Filename == filename && f.IsSpoiler == isSpoiler && f.Size > 0
	})
}

func ExpectStaging(t *testing.T, uploadSvc *upload.MockService) {
	t.Helper()

	staging := NewStaging(t)
	uploadSvc.EXPECT().Stage(mock.Anything, mock.Anything).RunAndReturn(staging.Stage).Maybe()
	uploadSvc.EXPECT().Discard(mock.Anything).Run(staging.Discard).Maybe()
}
