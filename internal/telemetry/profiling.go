package telemetry

import (
	"strings"
	"sync"

	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/logger"

	"github.com/grafana/pyroscope-go"
)

var (
	profilingMu       sync.Mutex
	profiler          *pyroscope.Profiler
	currentPyroscope  string
	profilingService  string
	profilingInstance string
)

func InitProfiling(serviceName, instance, url string) error {
	profilingMu.Lock()
	defer profilingMu.Unlock()
	profilingService = serviceName
	profilingInstance = instance
	return applyProfilingLocked(url)
}

func ApplyProfiling(url string) error {
	profilingMu.Lock()
	defer profilingMu.Unlock()
	if profilingService == "" {
		return nil
	}
	return applyProfilingLocked(url)
}

func applyProfilingLocked(url string) error {
	url = strings.TrimSpace(url)
	if url == currentPyroscope {
		return nil
	}

	if profiler != nil {
		_ = profiler.Stop()
		profiler = nil
	}

	currentPyroscope = url

	if url == "" {
		logger.Log.Info().Msg("pyroscope profiling disabled")
		return nil
	}

	p, err := pyroscope.Start(pyroscope.Config{
		ApplicationName: profilingService,
		ServerAddress:   url,
		Logger:          nil,
		Tags: map[string]string{
			"instance": profilingInstance,
		},
		ProfileTypes: []pyroscope.ProfileType{
			pyroscope.ProfileCPU,
			pyroscope.ProfileAllocObjects,
			pyroscope.ProfileAllocSpace,
			pyroscope.ProfileInuseObjects,
			pyroscope.ProfileInuseSpace,
			pyroscope.ProfileGoroutines,
			pyroscope.ProfileMutexCount,
			pyroscope.ProfileMutexDuration,
			pyroscope.ProfileBlockCount,
			pyroscope.ProfileBlockDuration,
		},
	})
	if err != nil {
		return err
	}
	profiler = p
	logger.Log.Info().Str("url", url).Msg("pyroscope profiling enabled")
	return nil
}

func ShutdownProfiling() {
	profilingMu.Lock()
	defer profilingMu.Unlock()
	if profiler == nil {
		return
	}
	_ = profiler.Stop()
	profiler = nil
}

type ProfilingSettingsListener struct{}

func NewProfilingSettingsListener() *ProfilingSettingsListener {
	return &ProfilingSettingsListener{}
}

func (l *ProfilingSettingsListener) OnSettingChanged(key config.SiteSettingKey, value string) {
	if key != config.SettingPyroscopeURL.Key {
		return
	}
	if err := ApplyProfiling(value); err != nil {
		logger.Log.Warn().Err(err).Msg("failed to apply pyroscope url change")
	}
}
