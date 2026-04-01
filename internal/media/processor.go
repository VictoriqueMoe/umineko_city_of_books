package media

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"umineko_city_of_books/internal/logger"
)

const (
	defaultWorkers  = 4
	defaultQueueCap = 256
	cwebpQuality    = "80"
	ffmpegCRF       = "28"
)

type (
	JobType int

	Job struct {
		Type      JobType
		InputPath string
		Callback  func(outputPath string)
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
	}
}

func (p *Processor) worker(id int) {
	for job := range p.jobs {
		var outputPath string
		var err error

		switch job.Type {
		case JobImage:
			outputPath, err = encodeImage(job.InputPath)
		case JobVideo:
			outputPath, err = encodeVideo(job.InputPath)
		}

		if err != nil {
			logger.Log.Error().Err(err).Int("worker", id).Str("input", job.InputPath).Msg("media encoding failed")
			continue
		}

		if job.Callback != nil {
			job.Callback(outputPath)
		}

		logger.Log.Debug().Int("worker", id).Str("output", outputPath).Msg("media encoding complete")
	}
}

func encodeImage(inputPath string) (string, error) {
	if strings.HasSuffix(strings.ToLower(inputPath), ".webp") {
		return inputPath, nil
	}

	outputPath := replaceExt(inputPath, ".webp")

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "cwebp", "-q", cwebpQuality, inputPath, "-o", outputPath)
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("cwebp: %w: %s", err, string(out))
	}

	_ = os.Remove(inputPath)
	return outputPath, nil
}

func encodeVideo(inputPath string) (string, error) {
	if strings.HasSuffix(strings.ToLower(inputPath), ".webm") {
		return inputPath, nil
	}

	outputPath := replaceExt(inputPath, ".mp4")
	if inputPath == outputPath {
		return inputPath, nil
	}

	tmpOutput := outputPath + ".tmp"

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
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

func replaceExt(path, newExt string) string {
	ext := filepath.Ext(path)
	return path[:len(path)-len(ext)] + newExt
}
