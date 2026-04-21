package media

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
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
		jobs chan Job
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
	}

	for i := range workers {
		go p.worker(i)
	}

	logger.Log.Info().Int("workers", workers).Msg("media processor started")
	return p
}

func (p *Processor) Enqueue(job Job) {
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
	for job := range p.jobs {
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
			continue
		}

		if job.Callback != nil {
			job.Callback(outputPath)
		}

		logger.Log.Debug().Int("worker", id).Str("output", outputPath).Msg("media encoding complete")
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

	_ = os.Remove(inputPath)
	if err := os.Rename(tmpOutput, outputPath); err != nil {
		return "", fmt.Errorf("rename output: %w", err)
	}

	return outputPath, nil
}
