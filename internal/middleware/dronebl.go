package middleware

import (
	"strings"
	"umineko_city_of_books/internal/logger"

	"umineko_city_of_books/internal/dronebl"
	"umineko_city_of_books/internal/session"

	"github.com/gofiber/fiber/v3"
)

var (
	exemptFromDroneBL = map[string]bool{
		"/livez":  true,
		"/health": true,
	}

	loginEscapePaths = map[string]bool{
		"/login":               true,
		"/api/v1/auth/login":   true,
		"/api/v1/auth/session": true,
		"/api/v1/site-info":    true,
	}
)

const (
	blockedPage = `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<meta name="robots" content="noindex">
<title>Access blocked</title>
<style>
:root { color-scheme: dark; }
body {
  margin: 0;
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #0a0710;
  color: #d8d2e0;
  font-family: Georgia, "Times New Roman", serif;
  padding: 1.5rem;
}
.card {
  max-width: 32rem;
  text-align: center;
  background: rgba(120, 80, 200, 0.12);
  border: 1px solid rgba(212, 175, 90, 0.25);
  border-radius: 4px;
  padding: 2.5rem 2rem;
}
h1 {
  margin: 0 0 1rem;
  font-size: 1.6rem;
  letter-spacing: 0.05em;
  color: #d4af5a;
  font-weight: normal;
}
p { margin: 0 0 1rem; line-height: 1.6; font-size: 0.95rem; }
.muted { color: #8d85a0; font-style: italic; font-size: 0.85rem; margin-bottom: 0; }
a.signin {
  display: inline-block;
  margin: 0.5rem 0 1.25rem;
  padding: 0.6rem 1.4rem;
  color: #d4af5a;
  text-decoration: none;
  border: 1px solid rgba(212, 175, 90, 0.4);
  border-radius: 3px;
  background: rgba(212, 175, 90, 0.1);
}
a.signin:hover { background: rgba(212, 175, 90, 0.2); }
code { font-family: ui-monospace, Consolas, monospace; font-size: 0.85rem; color: #d4af5a; }
</style>
</head>
<body>
<div class="card">
<h1>The gate is closed to you</h1>
<p>Your connection comes from an address listed on a public abuse blocklist, so this site will not answer it.</p>
<p>This is usually not about you. Addresses are recycled, and whoever held yours before may have been running a proxy or a compromised machine.</p>
<a class="signin" href="/login">Sign in</a>
<p class="muted">Members can still sign in. If you have no account, check whether your address is listed at <code>dronebl.org/lookup</code>, or reconnect to get a new one.</p>
</div>
</body>
</html>`
)

func RequireCleanIP(checker *dronebl.Checker, sessionMgr *session.Manager) fiber.Handler {
	return func(ctx fiber.Ctx) error {
		if checker == nil {
			return ctx.Next()
		}

		path := ctx.Path()
		if exemptFromDroneBL[path] {
			return ctx.Next()
		}

		if !checker.Enabled(ctx.Context()) {
			return ctx.Next()
		}

		ip, _ := ctx.Locals("client_ip").(string)
		if ip == "" {
			return ctx.Next()
		}

		if checker.Allowlisted(ctx.Context(), ip) {
			return ctx.Next()
		}

		verdict, blocked := checker.Blocked(ctx.Context(), ip)
		if !blocked {
			droneblChecks.WithLabelValues("clean").Inc()

			return ctx.Next()
		}

		if hasSession(ctx, sessionMgr) {
			droneblChecks.WithLabelValues("member").Inc()

			return ctx.Next()
		}

		if loginEscapePaths[path] || isStaticAsset(path) {
			droneblChecks.WithLabelValues("login_escape").Inc()

			return ctx.Next()
		}

		droneblChecks.WithLabelValues("blocked").Inc()
		recordBlockedClasses(verdict.Classes)
		logger.Log.Warn().
			Str("ip", ip).
			Ints("classes", verdict.Classes).
			Str("path", path).
			Msg("dronebl blocked a request")

		return respondBlocked(ctx)
	}
}

func isStaticAsset(path string) bool {
	if strings.HasPrefix(path, "/api") || strings.HasPrefix(path, "/uploads") || strings.HasPrefix(path, "/hls") {
		return false
	}

	return strings.Contains(strings.TrimPrefix(path, "/"), ".")
}

func respondBlocked(ctx fiber.Ctx) error {
	if !wantsHTML(ctx) {
		return ctx.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "your network is listed on a public abuse blocklist",
		})
	}

	ctx.Set(fiber.HeaderContentType, fiber.MIMETextHTMLCharsetUTF8)

	return ctx.Status(fiber.StatusForbidden).SendString(blockedPage)
}

func wantsHTML(ctx fiber.Ctx) bool {
	if strings.HasPrefix(ctx.Path(), "/api") {
		return false
	}

	return strings.Contains(ctx.Get(fiber.HeaderAccept), "text/html")
}
