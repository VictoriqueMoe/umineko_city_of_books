package engines

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"iter"
	"os"
	"path"
	"path/filepath"
	"strings"

	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/settings"
	"umineko_city_of_books/internal/storage/engine"

	"github.com/google/uuid"
)

var (
	errStopListing = errors.New("storage: listing stopped")
)

type (
	Local struct {
		settingsSvc settings.Service
	}

	rangeReader struct {
		io.Reader
		io.Closer
	}
)

func NewLocal(settingsSvc settings.Service) *Local {
	return &Local{settingsSvc: settingsSvc}
}

func (l *Local) ID() config.StorageBackend {
	return config.StorageBackendLocal
}

func (l *Local) Enabled() bool {
	return true
}

func (l *Local) Get(ctx context.Context, location string, rng *engine.ByteRange) (io.ReadCloser, error) {
	key := location
	fullPath, err := l.resolve(ctx, key)
	if err != nil {
		return nil, err
	}

	f, err := os.Open(fullPath)
	if err != nil {
		return nil, mapLocalError(err, key)
	}

	info, err := f.Stat()
	if err != nil {
		_ = f.Close()

		return nil, fmt.Errorf("stat %s: %w", key, err)
	}

	if info.IsDir() {
		_ = f.Close()

		return nil, fmt.Errorf("%w: %s", engine.ErrNotFound, key)
	}

	if rng == nil {
		return f, nil
	}

	if _, err := f.Seek(rng.Start, io.SeekStart); err != nil {
		_ = f.Close()

		return nil, fmt.Errorf("seek %s: %w", key, err)
	}

	return rangeReader{Reader: io.LimitReader(f, rng.Length()), Closer: f}, nil
}

func (l *Local) PutFile(ctx context.Context, key string, localPath string) (string, error) {
	dest, err := l.resolve(ctx, key)
	if err != nil {
		return "", err
	}

	dir := filepath.Dir(dest)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("create directory: %w", err)
	}

	tmp := filepath.Join(dir, "."+filepath.Base(dest)+".tmp-"+uuid.NewString())
	if err := os.Link(localPath, tmp); err != nil {
		if err := copyFile(localPath, tmp); err != nil {
			_ = os.Remove(tmp)

			return "", err
		}
	}

	if err := os.Rename(tmp, dest); err != nil {
		_ = os.Remove(tmp)

		return "", fmt.Errorf("move into place: %w", err)
	}

	return key, nil
}

func (l *Local) Head(ctx context.Context, location string) (engine.ObjectInfo, error) {
	key := location
	fullPath, err := l.resolve(ctx, key)
	if err != nil {
		return engine.ObjectInfo{}, err
	}

	info, err := os.Stat(fullPath)
	if err != nil {
		return engine.ObjectInfo{}, mapLocalError(err, key)
	}

	if info.IsDir() {
		return engine.ObjectInfo{}, fmt.Errorf("%w: %s", engine.ErrNotFound, key)
	}

	return engine.ObjectInfo{Key: key, Location: key, Size: info.Size(), ModTime: info.ModTime(), Backend: l.ID()}, nil
}

func (l *Local) Delete(ctx context.Context, locations ...string) error {
	var errs []error
	for _, key := range locations {
		fullPath, err := l.resolve(ctx, key)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		if err := os.Remove(fullPath); err != nil && !errors.Is(err, fs.ErrNotExist) {
			errs = append(errs, fmt.Errorf("delete %s: %w", key, err))
		}
	}

	return errors.Join(errs...)
}

func (l *Local) List(ctx context.Context, prefix string) iter.Seq2[engine.ObjectInfo, error] {
	return func(yield func(engine.ObjectInfo, error) bool) {
		root, err := l.root(ctx)
		if err != nil {
			yield(engine.ObjectInfo{}, err)

			return
		}

		start := root
		if dir := path.Dir(prefix); dir != "." {
			if err := engine.ValidateKey(dir); err != nil {
				yield(engine.ObjectInfo{}, err)

				return
			}

			start = filepath.Join(root, filepath.FromSlash(dir))
		}

		walkErr := filepath.WalkDir(start, func(fullPath string, entry fs.DirEntry, err error) error {
			if err != nil {
				if fullPath == start && errors.Is(err, fs.ErrNotExist) {
					return filepath.SkipAll
				}

				if !yield(engine.ObjectInfo{}, fmt.Errorf("list %s: %w", fullPath, err)) {
					return errStopListing
				}

				if entry != nil && entry.IsDir() {
					return filepath.SkipDir
				}

				return nil
			}

			if ctxErr := ctx.Err(); ctxErr != nil {
				return ctxErr
			}

			if fullPath != start && strings.HasPrefix(entry.Name(), ".") {
				if entry.IsDir() {
					return filepath.SkipDir
				}

				return nil
			}

			if entry.IsDir() {
				return nil
			}

			rel, err := filepath.Rel(root, fullPath)
			if err != nil {
				return err
			}

			key := filepath.ToSlash(rel)
			if !strings.HasPrefix(key, prefix) {
				return nil
			}

			info, err := entry.Info()
			if errors.Is(err, fs.ErrNotExist) {
				return nil
			}

			if err != nil {
				if !yield(engine.ObjectInfo{}, fmt.Errorf("stat %s: %w", fullPath, err)) {
					return errStopListing
				}

				return nil
			}

			if !yield(engine.ObjectInfo{Key: key, Location: key, Size: info.Size(), ModTime: info.ModTime(), Backend: l.ID()}, nil) {
				return errStopListing
			}

			return nil
		})

		if walkErr != nil && !errors.Is(walkErr, errStopListing) {
			yield(engine.ObjectInfo{}, fmt.Errorf("list %q: %w", prefix, walkErr))
		}
	}
}

func (l *Local) root(ctx context.Context) (string, error) {
	root, err := filepath.Abs(l.settingsSvc.Get(ctx, config.SettingUploadDir))
	if err != nil {
		return "", fmt.Errorf("resolve upload directory: %w", err)
	}

	return root, nil
}

func (l *Local) resolve(ctx context.Context, key string) (string, error) {
	if err := engine.ValidateKey(key); err != nil {
		return "", err
	}

	root, err := l.root(ctx)
	if err != nil {
		return "", err
	}

	return filepath.Join(root, filepath.FromSlash(key)), nil
}

func mapLocalError(err error, key string) error {
	if errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("%w: %s", engine.ErrNotFound, key)
	}

	return fmt.Errorf("%s: %w", key, err)
}

func copyFile(src string, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open source: %w", err)
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}

	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()

		return fmt.Errorf("write file: %w", err)
	}

	if err := out.Close(); err != nil {
		return fmt.Errorf("close file: %w", err)
	}

	return nil
}
