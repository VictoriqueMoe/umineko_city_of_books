package main

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/pprof"
	"time"

	"umineko_city_of_books/internal/logger"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	internalReadHeaderTimeout = 5 * time.Second
)

func startInternalServer(addr string) error {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen on internal address %s: %w", addr, err)
	}

	mux := http.NewServeMux()
	mux.Handle("GET /metrics", promhttp.Handler())
	mux.HandleFunc("GET /debug/pprof/", pprof.Index)
	mux.HandleFunc("GET /debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("GET /debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("GET /debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("GET /debug/pprof/trace", pprof.Trace)

	server := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: internalReadHeaderTimeout,
	}

	go func() {
		err := server.Serve(listener)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Log.Error().Err(err).Str("addr", addr).Msg("internal server stopped")
		}
	}()

	return nil
}
