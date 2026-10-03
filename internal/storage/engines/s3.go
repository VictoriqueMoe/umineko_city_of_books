package engines

import (
	"context"
	"errors"
	"fmt"
	"io"
	"iter"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path"
	"slices"
	"strings"
	"sync"
	"time"

	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/settings"
	"umineko_city_of_books/internal/storage/engine"

	"github.com/aws/aws-sdk-go-v2/aws"
	awshttp "github.com/aws/aws-sdk-go-v2/aws/transport/http"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

const (
	s3DeleteBatchSize = 1000
	s3ProbeTimeout    = 10 * time.Second
)

var (
	S3SettingKeys = []config.SiteSettingKey{
		config.SettingS3Endpoint.Key,
		config.SettingS3Region.Key,
		config.SettingS3Bucket.Key,
		config.SettingS3Prefix.Key,
		config.SettingS3AccessKeyID.Key,
		config.SettingS3SecretAccessKey.Key,
		config.SettingS3ForcePathStyle.Key,
	}
)

type (
	S3 struct {
		settingsSvc settings.Service

		mu      sync.RWMutex
		applied S3Config
		bucket  *s3Bucket
	}

	S3Config struct {
		Endpoint        string
		Region          string
		Bucket          string
		Prefix          string
		AccessKeyID     string
		SecretAccessKey string
		ForcePathStyle  bool
	}

	s3Bucket struct {
		client *s3.Client
		name   string
		prefix string
	}
)

func NewS3(settingsSvc settings.Service) *S3 {
	return &S3{settingsSvc: settingsSvc}
}

func S3ConfigFrom(values map[config.SiteSettingKey]string) S3Config {
	return S3Config{
		Endpoint:        strings.TrimSpace(values[config.SettingS3Endpoint.Key]),
		Region:          strings.TrimSpace(values[config.SettingS3Region.Key]),
		Bucket:          strings.TrimSpace(values[config.SettingS3Bucket.Key]),
		Prefix:          normalisePrefix(strings.TrimSpace(values[config.SettingS3Prefix.Key])),
		AccessKeyID:     strings.TrimSpace(values[config.SettingS3AccessKeyID.Key]),
		SecretAccessKey: values[config.SettingS3SecretAccessKey.Key],
		ForcePathStyle:  values[config.SettingS3ForcePathStyle.Key] == "true",
	}
}

func (c S3Config) Complete() bool {
	return c.Region != "" && c.Bucket != "" && c.AccessKeyID != "" && strings.TrimSpace(c.SecretAccessKey) != ""
}

func (c S3Config) Validate() error {
	if c.Endpoint == "" {
		return nil
	}

	endpoint, err := url.Parse(c.Endpoint)
	if err != nil || (endpoint.Scheme != "https" && endpoint.Scheme != "http") || endpoint.Host == "" {
		return fmt.Errorf("storage: s3 endpoint %q must be a full http(s) URL such as https://<account>.r2.cloudflarestorage.com", c.Endpoint)
	}

	return nil
}

func ProbeS3(ctx context.Context, cfg S3Config) error {
	if err := cfg.Validate(); err != nil {
		return err
	}

	return newS3Bucket(cfg).probe(ctx)
}

func (e *S3) ID() config.StorageBackend {
	return config.StorageBackendS3
}

func (e *S3) Enabled() bool {
	return e.current() != nil
}

func (e *S3) Reconfigure(ctx context.Context) error {
	cfg := S3ConfigFrom(e.settingsSvc.GetAll(ctx))

	e.mu.Lock()
	if cfg == e.applied && (e.bucket != nil) == cfg.Complete() {
		e.mu.Unlock()

		return nil
	}

	e.applied = cfg
	e.bucket = nil

	if !cfg.Complete() {
		e.mu.Unlock()

		return nil
	}

	if err := cfg.Validate(); err != nil {
		e.mu.Unlock()

		return err
	}

	bucket := newS3Bucket(cfg)
	e.bucket = bucket
	e.mu.Unlock()

	if err := bucket.probe(ctx); err != nil {
		return err
	}

	logger.Ctx(ctx).Info().Str("bucket", bucket.name).Str("prefix", bucket.prefix).Msg("s3 storage engine enabled")

	return nil
}

func (e *S3) OnSettingsBatchChanged(keys []config.SiteSettingKey) {
	if !slices.ContainsFunc(keys, func(key config.SiteSettingKey) bool {
		return slices.Contains(S3SettingKeys, key)
	}) {
		return
	}

	if err := e.Reconfigure(context.Background()); err != nil {
		logger.Log.Warn().Err(err).Msg("s3 storage engine reconfigure failed")
	}
}

func (e *S3) Get(ctx context.Context, location string, rng *engine.ByteRange) (io.ReadCloser, error) {
	bucket, err := e.require()
	if err != nil {
		return nil, err
	}

	bucketName, objectKey, err := parseS3Location(location)
	if err != nil {
		return nil, err
	}

	input := &s3.GetObjectInput{Bucket: aws.String(bucketName), Key: aws.String(objectKey)}
	if rng != nil {
		input.Range = aws.String(fmt.Sprintf("bytes=%d-%d", rng.Start, rng.End))
	}

	out, err := bucket.client.GetObject(ctx, input)
	if err != nil {
		return nil, mapS3Error(err, location)
	}

	return out.Body, nil
}

func (e *S3) PutFile(ctx context.Context, key string, localPath string) (string, error) {
	if err := engine.ValidateKey(key); err != nil {
		return "", err
	}

	bucket, err := e.require()
	if err != nil {
		return "", err
	}

	f, err := os.Open(localPath)
	if err != nil {
		return "", fmt.Errorf("open %s: %w", localPath, err)
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return "", fmt.Errorf("stat %s: %w", localPath, err)
	}

	objectKey := bucket.prefix + key

	_, err = bucket.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(bucket.name),
		Key:           aws.String(objectKey),
		Body:          f,
		ContentLength: aws.Int64(info.Size()),
		ContentType:   aws.String(contentTypeFor(key)),
	})
	if err != nil {
		return "", fmt.Errorf("put %s: %w", key, err)
	}

	return bucket.name + "/" + objectKey, nil
}

func (e *S3) Head(ctx context.Context, location string) (engine.ObjectInfo, error) {
	bucket, err := e.require()
	if err != nil {
		return engine.ObjectInfo{}, err
	}

	bucketName, objectKey, err := parseS3Location(location)
	if err != nil {
		return engine.ObjectInfo{}, err
	}

	out, err := bucket.client.HeadObject(ctx, &s3.HeadObjectInput{Bucket: aws.String(bucketName), Key: aws.String(objectKey)})
	if err != nil {
		return engine.ObjectInfo{}, mapS3Error(err, location)
	}

	return engine.ObjectInfo{
		Location: location,
		Size:     aws.ToInt64(out.ContentLength),
		ModTime:  aws.ToTime(out.LastModified),
		Backend:  e.ID(),
	}, nil
}

func (e *S3) Delete(ctx context.Context, locations ...string) error {
	bucket, err := e.require()
	if err != nil {
		return err
	}

	var errs []error
	byBucket := make(map[string][]types.ObjectIdentifier)
	for _, location := range locations {
		bucketName, objectKey, err := parseS3Location(location)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		byBucket[bucketName] = append(byBucket[bucketName], types.ObjectIdentifier{Key: aws.String(objectKey)})
	}

	for bucketName, objects := range byBucket {
		for batch := range slices.Chunk(objects, s3DeleteBatchSize) {
			out, err := bucket.client.DeleteObjects(ctx, &s3.DeleteObjectsInput{
				Bucket: aws.String(bucketName),
				Delete: &types.Delete{Objects: batch, Quiet: aws.Bool(true)},
			})
			if err != nil {
				errs = append(errs, fmt.Errorf("delete %d objects from %s: %w", len(batch), bucketName, err))
				continue
			}

			for _, failure := range out.Errors {
				if aws.ToString(failure.Code) == "NoSuchKey" {
					continue
				}

				errs = append(errs, fmt.Errorf("delete %s/%s: %s %s", bucketName, aws.ToString(failure.Key), aws.ToString(failure.Code), aws.ToString(failure.Message)))
			}
		}
	}

	return errors.Join(errs...)
}

func (e *S3) List(ctx context.Context, prefix string) iter.Seq2[engine.ObjectInfo, error] {
	return func(yield func(engine.ObjectInfo, error) bool) {
		bucket, err := e.require()
		if err != nil {
			yield(engine.ObjectInfo{}, err)

			return
		}

		pages := s3.NewListObjectsV2Paginator(bucket.client, &s3.ListObjectsV2Input{
			Bucket: aws.String(bucket.name),
			Prefix: aws.String(bucket.prefix + prefix),
		})

		for pages.HasMorePages() {
			page, err := pages.NextPage(ctx)
			if err != nil {
				yield(engine.ObjectInfo{}, fmt.Errorf("list %q: %w", prefix, err))

				return
			}

			for _, object := range page.Contents {
				objectKey := aws.ToString(object.Key)
				info := engine.ObjectInfo{
					Key:      strings.TrimPrefix(objectKey, bucket.prefix),
					Location: bucket.name + "/" + objectKey,
					Size:     aws.ToInt64(object.Size),
					ModTime:  aws.ToTime(object.LastModified),
					Backend:  e.ID(),
				}

				if !yield(info, nil) {
					return
				}
			}
		}
	}
}

func (e *S3) current() *s3Bucket {
	e.mu.RLock()
	defer e.mu.RUnlock()

	return e.bucket
}

func (e *S3) require() (*s3Bucket, error) {
	bucket := e.current()
	if bucket == nil {
		return nil, engine.ErrDisabled
	}

	return bucket, nil
}

func (b *s3Bucket) probe(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, s3ProbeTimeout)
	defer cancel()

	if _, err := b.client.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: aws.String(b.name)}); err != nil {
		return fmt.Errorf("storage: s3 bucket %q is unreachable: %w", b.name, err)
	}

	return nil
}

func newS3Bucket(cfg S3Config) *s3Bucket {
	options := s3.Options{
		Region:                     cfg.Region,
		Credentials:                credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		UsePathStyle:               cfg.ForcePathStyle,
		RequestChecksumCalculation: aws.RequestChecksumCalculationWhenRequired,
		ResponseChecksumValidation: aws.ResponseChecksumValidationWhenRequired,
	}
	if cfg.Endpoint != "" {
		options.BaseEndpoint = aws.String(cfg.Endpoint)
	}

	return &s3Bucket{
		client: s3.New(options),
		name:   cfg.Bucket,
		prefix: cfg.Prefix,
	}
}

func parseS3Location(location string) (string, string, error) {
	bucketName, objectKey, ok := strings.Cut(location, "/")
	if !ok || bucketName == "" {
		return "", "", fmt.Errorf("%w: s3 location %q", engine.ErrInvalidKey, location)
	}

	if err := engine.ValidateKey(objectKey); err != nil {
		return "", "", err
	}

	return bucketName, objectKey, nil
}

func normalisePrefix(raw string) string {
	trimmed := strings.Trim(raw, "/")
	if trimmed == "" {
		return ""
	}

	return trimmed + "/"
}

func contentTypeFor(key string) string {
	contentType := mime.TypeByExtension(path.Ext(key))
	if contentType == "" {
		return "application/octet-stream"
	}

	return contentType
}

func mapS3Error(err error, location string) error {
	if _, ok := errors.AsType[*types.NoSuchKey](err); ok {
		return fmt.Errorf("%w: %s", engine.ErrNotFound, location)
	}

	if _, ok := errors.AsType[*types.NotFound](err); ok {
		return fmt.Errorf("%w: %s", engine.ErrNotFound, location)
	}

	if respErr, ok := errors.AsType[*awshttp.ResponseError](err); ok && respErr.HTTPStatusCode() == http.StatusNotFound {
		return fmt.Errorf("%w: %s", engine.ErrNotFound, location)
	}

	return fmt.Errorf("%s: %w", location, err)
}
