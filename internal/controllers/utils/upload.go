package utils

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"os"

	"umineko_city_of_books/internal/middleware"
	"umineko_city_of_books/internal/upload"

	"github.com/gofiber/fiber/v3"
)

type (
	Upload struct {
		staging         upload.Service
		values          map[string][]string
		files           map[string][]ReceivedFile
		restrictedHours int
		restricted      bool
	}

	partReader struct {
		part io.Reader
		err  error
	}

	ReceivedFile struct {
		Filename    string
		ContentType string
		Path        string
		Size        int64
	}

	countedReader struct {
		reader io.Reader
		limit  int64
		read   int64
	}

	OpenedFile struct {
		*os.File
		ReceivedFile
		Form *Upload
	}
)

func (c *countedReader) Read(p []byte) (int, error) {
	if c.read > c.limit {
		return 0, ErrUploadTooLarge
	}

	if allowed := c.limit - c.read + 1; int64(len(p)) > allowed {
		p = p[:allowed]
	}

	n, err := c.reader.Read(p)
	c.read += int64(n)
	if c.read > c.limit {
		return n - 1, ErrUploadTooLarge
	}

	return n, err
}

func ReceiveUpload(ctx fiber.Ctx, staging upload.Service, maxBytes int64) (*Upload, error) {
	mediaType, params, err := mime.ParseMediaType(ctx.Get(fiber.HeaderContentType))
	if err != nil || mediaType != fiber.MIMEMultipartForm || params["boundary"] == "" {
		return nil, ErrNotMultipart
	}

	var body = ctx.Request().BodyStream()
	if body == nil {
		body = bytes.NewReader(ctx.Body())
	}

	hours, restricted := ctx.Locals(middleware.RestrictedUploadHoursLocal).(int)

	received := &Upload{staging: staging, values: map[string][]string{}, files: map[string][]ReceivedFile{}, restrictedHours: hours, restricted: restricted}
	if err := received.read(multipart.NewReader(&countedReader{reader: body, limit: maxBytes}, params["boundary"])); err != nil {
		received.Close()

		return nil, err
	}

	return received, nil
}

func (p *partReader) Read(b []byte) (int, error) {
	n, err := p.part.Read(b)
	if err != nil && !errors.Is(err, io.EOF) {
		p.err = err
	}

	return n, err
}

func (u *Upload) read(parts *multipart.Reader) error {
	for {
		part, err := parts.NextPart()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return uploadError(err)
		}

		if part.FileName() == "" {
			err = u.readValue(part)
		} else {
			err = u.readFile(part)
		}
		_ = part.Close()

		if err != nil {
			return err
		}
	}
}

func (u *Upload) readValue(part *multipart.Part) error {
	var value bytes.Buffer
	if _, err := io.Copy(&value, part); err != nil {
		return uploadError(err)
	}

	u.values[part.FormName()] = append(u.values[part.FormName()], value.String())

	return nil
}

func (u *Upload) readFile(part *multipart.Part) error {
	if u.restricted {
		return fmt.Errorf("%w for their first %d hours", ErrNewAccountUpload, u.restrictedHours)
	}

	if !validFilename(part.FileName()) {
		return fmt.Errorf("%w: unusable file name %q", ErrNotMultipart, part.FileName())
	}

	src := &partReader{part: part}

	staged, err := u.staging.Stage(part.FileName(), src)
	if src.err != nil {
		return uploadError(src.err)
	}
	if err != nil {
		return fmt.Errorf("stage upload: %w", err)
	}

	info, err := os.Stat(staged)
	if err != nil {
		u.staging.Discard(staged)

		return fmt.Errorf("stat upload: %w", err)
	}

	u.files[part.FormName()] = append(u.files[part.FormName()], ReceivedFile{
		Filename:    part.FileName(),
		ContentType: part.Header.Get(fiber.HeaderContentType),
		Path:        staged,
		Size:        info.Size(),
	})

	return nil
}

func validFilename(name string) bool {
	return name != "." && name != ".." && name != "/" && name != `\`
}

func uploadError(err error) error {
	if errors.Is(err, ErrUploadTooLarge) {
		return ErrUploadTooLarge
	}

	return fmt.Errorf("%w: %w", ErrNotMultipart, err)
}

func (u *Upload) Close() {
	for _, files := range u.files {
		for _, file := range files {
			u.staging.Discard(file.Path)
		}
	}
}

func (u *Upload) Value(key string) string {
	values := u.values[key]
	if len(values) == 0 {
		return ""
	}

	return values[0]
}

func (u *Upload) File(field string) (ReceivedFile, bool) {
	files := u.files[field]
	if len(files) == 0 {
		return ReceivedFile{}, false
	}

	return files[0], true
}

func (u *Upload) FileUploads(field string, spoilers map[int]struct{}) []upload.FileUpload {
	files := u.files[field]
	if len(files) == 0 {
		return nil
	}

	uploads := make([]upload.FileUpload, 0, len(files))
	for i, file := range files {
		_, isSpoiler := spoilers[i]

		uploads = append(uploads, file.FileUpload(isSpoiler))
	}

	return uploads
}

func (f ReceivedFile) FileUpload(isSpoiler bool) upload.FileUpload {
	return upload.FileUpload{
		ContentType: f.ContentType,
		Filename:    f.Filename,
		Size:        f.Size,
		IsSpoiler:   isSpoiler,
		Open: func() (io.ReadCloser, error) {
			return os.Open(f.Path)
		},
	}
}

func ReceiveFile(ctx fiber.Ctx, staging upload.Service, field string) (*OpenedFile, error) {
	form, err := ReceiveUpload(ctx, staging, int64(ctx.App().Server().MaxRequestBodySize))
	if err != nil {
		return nil, err
	}

	received, ok := form.File(field)
	if !ok {
		form.Close()

		return nil, fmt.Errorf("%w: %s", ErrNoFile, field)
	}

	file, err := os.Open(received.Path)
	if err != nil {
		form.Close()

		return nil, fmt.Errorf("open upload: %w", err)
	}

	return &OpenedFile{File: file, ReceivedFile: received, Form: form}, nil
}

func (f *OpenedFile) Close() error {
	err := f.File.Close()
	f.Form.Close()

	return err
}

func UploadFailed(ctx fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, ErrUploadTooLarge):
		return ctx.Status(fiber.StatusRequestEntityTooLarge).JSON(fiber.Map{"error": err.Error()})
	case errors.Is(err, ErrNewAccountUpload):
		return ctx.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": err.Error()})
	case errors.Is(err, ErrNoFile):
		return BadRequest(ctx, "no file provided")
	case errors.Is(err, ErrNotMultipart):
		return BadRequest(ctx, "invalid form data")
	default:
		return InternalError(ctx, "failed to receive the upload", err)
	}
}
