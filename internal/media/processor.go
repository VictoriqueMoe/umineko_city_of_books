package media

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"umineko_city_of_books/internal/logger"
)

const (
	defaultWorkers  = 4
	defaultQueueCap = 256
	ffmpegCRF       = "28"
	imageJobTimeout = 2 * time.Minute
	videoJobTimeout = 10 * time.Minute
)

var (
	ErrShuttingDown = errors.New("media processor shutting down")
)

type (
	JobType int

	Job struct {
		Type          JobType
		InputPath     string
		MaxWidth      int
		MaxHeight     int
		Quality       int
		SquareCrop    bool
		Callback      func(outputPath string)
		ErrorCallback func(err error)
	}

	Processor struct {
		jobs    chan Job
		quit    chan struct{}
		wg      sync.WaitGroup
		closing atomic.Bool
	}
)

const (
	JobImage JobType = iota
	JobVideo
)

func NewProcessor(workers int) *Processor {
	if workers <= 0 {
		workers = defaultWorkers
	}

	p := &Processor{
		jobs: make(chan Job, defaultQueueCap),
		quit: make(chan struct{}),
	}

	p.wg.Add(workers)
	for i := range workers {
		go p.worker(i)
	}

	logger.Log.Info().Int("workers", workers).Msg("media processor started")
	return p
}

func (p *Processor) Enqueue(job Job) {
	if p.closing.Load() {
		logger.Log.Warn().Str("path", job.InputPath).Msg("media processor is shutting down, rejecting job")
		if job.ErrorCallback != nil {
			job.ErrorCallback(ErrShuttingDown)
		}

		return
	}

	select {
	case p.jobs <- job:
	default:
		logger.Log.Warn().Str("path", job.InputPath).Msg("media processor queue full, dropping job")
		if job.ErrorCallback != nil {
			job.ErrorCallback(fmt.Errorf("media processor queue full"))
		}
	}
}

func (p *Processor) worker(id int) {
	defer p.wg.Done()

	for {
		select {
		case <-p.quit:
			return
		default:
		}

		select {
		case <-p.quit:
			return
		case job := <-p.jobs:
			p.run(id, job)
		}
	}
}

func (p *Processor) run(id int, job Job) {
	var outputPath string
	var err error

	switch job.Type {
	case JobImage:
		outputPath, err = encodeImage(job)
	case JobVideo:
		outputPath, err = encodeVideo(job.InputPath)
	}

	if err != nil {
		logger.Log.Error().Err(err).Int("worker", id).Str("input", job.InputPath).Msg("media encoding failed")
		if job.ErrorCallback != nil {
			job.ErrorCallback(err)
		}

		return
	}

	if job.Callback != nil {
		job.Callback(outputPath)
	}

	logger.Log.Debug().Int("worker", id).Str("output", outputPath).Msg("media encoding complete")
}

func (p *Processor) Shutdown(ctx context.Context) error {
	if !p.closing.CompareAndSwap(false, true) {
		return nil
	}

	close(p.quit)

	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		logger.Log.Info().Int("abandoned", len(p.jobs)).Msg("media processor drained")
		p.failPending()

		return nil
	case <-ctx.Done():
		logger.Log.Warn().Int("abandoned", len(p.jobs)).Msg("media processor drain timed out, encoding jobs were cut short")
		p.failPending()

		return ctx.Err()
	}
}

func (p *Processor) failPending() {
	for {
		select {
		case job := <-p.jobs:
			logger.Log.Warn().Str("path", job.InputPath).Msg("media job abandoned at shutdown")
			if job.ErrorCallback != nil {
				job.ErrorCallback(ErrShuttingDown)
			}
		default:
			return
		}
	}
}

func encodeImage(job Job) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), imageJobTimeout)
	defer cancel()

	return EncodeWebP(ctx, job.InputPath, WebPOptions{
		MaxWidth:   job.MaxWidth,
		MaxHeight:  job.MaxHeight,
		Quality:    job.Quality,
		SquareCrop: job.SquareCrop,
	})
}

func encodeVideo(inputPath string) (string, error) {
	if strings.HasSuffix(strings.ToLower(inputPath), ".webm") {
		return inputPath, nil
	}

	outputPath := replaceExt(inputPath, ".mp4")
	if inputPath == outputPath {
		return inputPath, nil
	}

	tmpOutput := replaceExt(outputPath, ".tmp.mp4")

	ctx, cancel := context.WithTimeout(context.Background(), videoJobTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-i", inputPath,
		"-c:v", "libx264",
		"-preset", "medium",
		"-crf", ffmpegCRF,
		"-c:a", "aac",
		"-b:a", "128k",
		"-movflags", "+faststart",
		"-y",
		tmpOutput,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		_ = os.Remove(tmpOutput)
		return "", fmt.Errorf("ffmpeg: %w: %s", err, string(out))
	}

	if err := os.Rename(tmpOutput, outputPath); err != nil {
		_ = os.Remove(tmpOutput)

		return "", fmt.Errorf("rename output: %w", err)
	}

	return outputPath, nil
}
