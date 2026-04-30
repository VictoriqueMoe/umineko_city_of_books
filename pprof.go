package main

import (
	"net/http"
	"net/http/pprof"

	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/middleware"
	"umineko_city_of_books/internal/session"

	"github.com/gofiber/fiber/v3"
)

type (
	pprofResponseWriter struct {
		ctx    fiber.Ctx
		header http.Header
		wrote  bool
	}
)

func registerPprofRoutes(app *fiber.App, sessionMgr *session.Manager, authzSvc authz.Service) {
	gate := middleware.RequirePermission(sessionMgr, authzSvc, authz.PermManageSettings)

	pprofAdapter := func(h http.HandlerFunc) fiber.Handler {
		return func(ctx fiber.Ctx) error {
			req, err := http.NewRequest(ctx.Method(), ctx.OriginalURL(), nil)
			if err != nil {
				return err
			}
			rw := &pprofResponseWriter{ctx: ctx, header: http.Header{}}
			h.ServeHTTP(rw, req)
			return nil
		}
	}

	routes := []struct {
		path    string
		handler http.HandlerFunc
	}{
		{"/debug/pprof/", pprof.Index},
		{"/debug/pprof/cmdline", pprof.Cmdline},
		{"/debug/pprof/profile", pprof.Profile},
		{"/debug/pprof/symbol", pprof.Symbol},
		{"/debug/pprof/trace", pprof.Trace},
		{"/debug/pprof/allocs", pprof.Handler("allocs").ServeHTTP},
		{"/debug/pprof/block", pprof.Handler("block").ServeHTTP},
		{"/debug/pprof/goroutine", pprof.Handler("goroutine").ServeHTTP},
		{"/debug/pprof/heap", pprof.Handler("heap").ServeHTTP},
		{"/debug/pprof/mutex", pprof.Handler("mutex").ServeHTTP},
		{"/debug/pprof/threadcreate", pprof.Handler("threadcreate").ServeHTTP},
	}

	for _, r := range routes {
		app.Get(r.path, gate, pprofAdapter(r.handler))
	}
}

func (w *pprofResponseWriter) Header() http.Header {
	return w.header
}

func (w *pprofResponseWriter) WriteHeader(status int) {
	if w.wrote {
		return
	}
	w.wrote = true
	for k, v := range w.header {
		if len(v) > 0 {
			w.ctx.Set(k, v[0])
		}
	}
	w.ctx.Status(status)
}

func (w *pprofResponseWriter) Write(p []byte) (int, error) {
	if !w.wrote {
		w.WriteHeader(http.StatusOK)
	}
	_, err := w.ctx.Write(p)
	return len(p), err
}
