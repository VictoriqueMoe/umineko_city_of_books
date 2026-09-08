# Umineko City of Books

<p align="center">
  <img src="https://waifuvault.moe/f/2076c25a-2637-45af-bd6b-bc3a9d4d9a45/featherine%20augustus%20aurora.png" alt="Umineko City of Books" width="900">
</p>

<p align="center">
  <sub>
    Artwork by <a href="https://m.twitch.tv/meru">Meru</a>
  </sub>
</p>

A community platform for fans of Umineko no Naku Koro ni, Higurashi, Ciconia, and the wider When They Cry series. The original goal was a place to declare fan theories as **blue truth**, attach quotes from the game as evidence, and have them debated on two sides: **"With love, it can be seen"** and **"Without love, it cannot be seen"**. It has since grown into a full social platform: theory debates across all three series, a Twitter-style game board, mystery boards, fan art galleries, ship and OC declarations, fanfiction, live reading journals, chat rooms with shared-browser watch parties, DMs, secret unlock hunts, multiplayer games (chess, checkers, othello, minesweeper, snakes & ladders, and real-time pong) with vanity titles for the top players, site-wide search, live notifications, and themed role-based moderation.

## Table of Contents

- [Features](#features)
- [Tech Stack](#tech-stack)
- [Architecture](#architecture)
- [Getting Started](#getting-started)
- [Database and Migrations](#database-and-migrations)
- [Development Workflow](#development-workflow)
- [Deployment](#deployment)
- [Adding a New Page](#adding-a-new-page)
- [Documentation](#documentation)
- [Provenance](#provenance)
- [License](#license)

## Features

- **[Theory Debates](docs/FEATURES.md#theory-debates)**: the original heart of the site: submit a fan theory as a blue truth, attach quote evidence, and let others refute or support it.
- **[Mysteries](docs/FEATURES.md#mysteries)**: a gamified puzzle mode where a Game Master poses a mystery with graduated clues and other players submit attempts.
- **[Gallery and Art](docs/FEATURES.md#gallery-and-art)**: fan art uploads with full social features, bundled into galleries and blurred behind a spoiler cover where a piece needs one.
- **[Ships](docs/FEATURES.md#ships)**: declare character pairings, mixed-series or featuring your own OCs, and rally votes for them.
- **[Original Characters](docs/FEATURES.md#original-characters)**: a dedicated home for player-created OCs, separate from the canon-character ship list.
- **[Game Board](docs/FEATURES.md#game-board)**: a Twitter-style social feed for off-topic posts and discussion, with its own corner for each series.
- **[Fanfiction](docs/FEATURES.md#fanfiction)**: write and publish multi-chapter fan stories, with FFN-style metadata and a remembered reading position.
- **[Reading Journals](docs/FEATURES.md#reading-journals)**: live-blog your read-throughs of Ryukishi07's works, posting reactions, theories, and predictions as you go.
- **[Chat Rooms and DMs](docs/FEATURES.md#chat-rooms-and-dms)**: real-time chat in two flavours: one-to-one direct messages and named group rooms.
- **[Chatbots](docs/FEATURES.md#chatbots)**: character accounts that answer in their own voice, backed by an OpenAI model, summoned by a mention, a reply, or a DM.
- **[Watch Parties](docs/FEATURES.md#watch-parties)**: shared-viewing sessions launched from inside a group chat room: a remote browser everyone takes turns driving, or your own screen broadcast to the room.
- **[Voice Chat](docs/FEATURES.md#voice-chat)**: real-time voice in group rooms, DMs, and watch parties, backed by a self-hosted LiveKit SFU rather than a peer-to-peer mesh.
- **[Live Streaming](docs/FEATURES.md#live-streaming)**: a public broadcast directory at `/live`: any member can go live from OBS or Streamlabs over WHIP, and anyone, logged in or not, can watch.
- **[Games](docs/FEATURES.md#games)**: multiplayer mini-games hosted entirely inside the site, from correspondence chess to a real-time pong duel, each with live games, past games, and spectators.
- **[Secrets and Unlock Hunts](docs/FEATURES.md#secrets-and-unlock-hunts)**: hidden puzzles scattered across the UI, collected piece by piece and surfaced on a public hub at `/secrets`, where the first solve closes the hunt for everyone.
- **[Announcements](docs/FEATURES.md#announcements)**: site-wide announcements with pinning, full markdown, and the same threaded comment system as everywhere else.
- **[Suggestions](docs/FEATURES.md#suggestions)**: a dedicated feedback channel for site improvements and bug reports, tracked as Open, Done, or Archived.
- **[Search](docs/FEATURES.md#search)**: a single search bar covers the whole site, backed by Postgres `tsvector` columns on every searchable entity.
- **[Quote Browser](docs/FEATURES.md#quote-browser)**: a standalone interface for browsing the full quote corpus across all three series, sourced from the Umineko Quote Finder API.
- **[Profiles and Social Graph](docs/FEATURES.md#profiles-and-social-graph)**: avatar, banner, bio, pronouns and favourite character, plus per-user theme and font, an activity feed, follows, blocks, and reading progress recorded per series.
- **[Notifications](docs/FEATURES.md#notifications)**: every notification is both a database row for the notifications page and a live event for the bell, fanning out to the socket, the stream overlay, mobile push, and email.
- **[Stream Overlay](docs/FEATURES.md#stream-overlay)**: site events can drive on-stream alert popups through SAMMI, from a personal connector file you download and import.
- **[Moderation and Admin](docs/FEATURES.md#moderation-and-admin)**: themed roles (Reality Author, Voyager Witch, Witch) over a permission-based authorisation layer, with vanity roles, reports, an audit log, and hot-reloading site settings.
- **[Platform Features](docs/FEATURES.md#platform-features)**: fourteen themes grouped by series, two font families, Discord-style formatting wherever text is typed, OG embeds, an auto-generated sitemap, background media processing, and a Capacitor mobile app.

Every one of those is written out in full, with its rules, limits and edge cases, in [docs/FEATURES.md](docs/FEATURES.md).

## Tech Stack

**Backend.** Go 1.27, Fiber v3, PostgreSQL via `jackc/pgx/v5` behind the `pgx/v5/stdlib` adapter, goose for migrations, testcontainers-go for DAO tests against a real Postgres, `gofiber/contrib/v3/websocket` for the hub, zerolog, `wneessen/go-mail`, `disintegration/imaging`, bluemonday, `openai/openai-go` v3, `valkey-io/valkey-go`, `prometheus/client_golang`, the OpenTelemetry Go SDK with XSAM/otelsql, `grafana/pyroscope-go`, `hellofresh/health-go`, `firebase.google.com/go`, `livekit/server-sdk-go`, `corentings/chess`, and mockery plus staticcheck pinned in the `go.mod` tool block.

**Frontend.** React 19, TypeScript 6, Vite 8, React Router v8 (the `react-router` package, never `react-router-dom`), TanStack Query v5, CSS Modules, DOMPurify with marked and highlight.js, TipTap 3, livekit-client with `@livekit/components-react`, hls.js, `@hyperbeam/web`, chess.js with react-chessboard, emoji-picker-react, `@marsidev/react-turnstile`, firebase, Capacitor 8 with `@capgo/capacitor-updater`, and Vitest 4 with Testing Library, oxlint and oxfmt.

**Infrastructure.** A Docker multi-stage build (Node build stage, Go build stage, Alpine runtime carrying FFmpeg and libwebp-tools), two Valkey instances (one coordinating LiveKit ingress and egress, one LRU-capped app cache), Caddy or another reverse proxy in front, session auth on httpOnly cookies with no JWTs, mockery v3 from `.mockery.yml` for every Go interface mock, and a `docker-compose.prod.yml` carrying `prometheus-*` scrape labels beside a `postgres-exporter` sidecar.

**External.** The [Umineko Quote Finder API](https://quotes.auaurora.moe/swagger/index.html) for quote search and evidence, the GIPHY API, [Hyperbeam](https://hyperbeam.com/) for virtual-browser watch parties, self-hosted [LiveKit](https://livekit.io/) for voice, screen share and streaming, OpenAI for the chatbot character accounts, and Firebase Cloud Messaging for native push. Optionally and entirely externally: a Prometheus scraper, an OTLP trace collector, a Pyroscope server, and Loki with a Grafana Alloy collector. None of those four is bundled in the compose files.

Why each of those was chosen, checked against `go.mod` and `frontend/package.json`, is in [ARCHITECTURE.md](docs/ARCHITECTURE.md#7-technology-choices).

## Architecture

The server is a single Go binary that embeds the compiled Vite bundle and serves both the SPA and the JSON API from one process. Every layer has a single responsibility: controllers parse HTTP, services orchestrate business logic, repositories own transactions and caching, DAOs own SQL, the hub owns live events, and the media processor owns encoding off the hot path. Almost all of that SQL is generated by sqlc from `internal/dao/queries` and checked against the migrations at build time.

```
   ┌────────────────────┐        ┌────────────────────┐
   │  Browser (React)   │        │  Capacitor app     │
   └──────────┬─────────┘        └─────────┬──────────┘
              └─────────────┬──────────────┘
                            ▼  HTTP / WS
   ┌──────────────────────────────────────────────────┐   ┌──────────────────┐
   │                   Fiber v3 app                   │──▶│  WebSocket hub   │
   └─────┬────────────────────────────────────────────┘   └──────────────────┘
         ▼
   Controllers ─▶ Services ─▶ Repositories ─▶ DAOs ─▶ PostgreSQL
                     │
                     ▼
            ┌──────────────────┐
            │ Media processor  │
            └──────────────────┘
```

Voice, watch parties, and live streaming run on a separate media plane: audio and video travel from the browser to the LiveKit SFU (or to the Hyperbeam VM) directly, and the Go process only mints signed tokens and tracks presence, so none of that traffic passes through the layers above.

[ARCHITECTURE.md](docs/ARCHITECTURE.md) has the rest: the full component map, the request lifecycle, the rule each layer boundary enforces, and a chapter per subsystem (data layer, cache, auth, permissions, content filter, WebSocket hub, notifications, media pipeline, background jobs, OG and SEO, observability).

## Getting Started

### Prerequisites

- Go 1.27 or newer
- Node.js LTS
- Docker (for the Postgres and Valkey containers, plus repo-layer tests via testcontainers-go)
- FFmpeg, both `ffmpeg` and `ffprobe`, for video transcoding and thumbnails
- libwebp-tools for WebP work: `cwebp` for conversion, `dwebp` and `webpmux` for the OG JPEG path
- The goose CLI, if you need to author a migration (see [Database and Migrations](#database-and-migrations))
- `psql` CLI (optional, handy for poking at the DB)

### Environment

Two env files live next to each other:

- **`postgres.env`**, the Postgres bootstrap credentials. Read by the postgres container at init and by the app at connect time. Copy from `postgres.env.example`:
  ```bash
  cp postgres.env.example postgres.env
  ```
  | Variable            | Description                                           |
  |---------------------|-------------------------------------------------------|
  | `POSTGRES_USER`     | DB role for the app (`umineko` by convention)         |
  | `POSTGRES_PASSWORD` | DB password                                           |
  | `POSTGRES_DB`       | Database name (`umineko_city_of_books` by convention) |

- **`.env`**, everything else the app reads. Copy from `.env.example`:
  ```bash
  cp .env.example .env
  ```

Only a short list of variables is read from the environment for its own sake. Everything else in `.env` is a first-boot seed for a row in `site_settings`.

**Read from the environment**

| Variable               | Default     | Description                                                                                                                                                                     |
|------------------------|-------------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `POSTGRES_HOST`        | `localhost` | Postgres host. `postgres` (the compose service name) under docker-compose                                                                                                       |
| `POSTGRES_PORT`        | `5432`      | Postgres port. The internal container port, not the host-mapped 5007                                                                                                            |
| `POSTGRES_SSL_MODE`    | `disable`   | Postgres SSL mode (`disable`, `require`, `verify-ca`, `verify-full`)                                                                                                            |
| `DATABASE_URL`         | (empty)     | Full connection string. If set, overrides the discrete `POSTGRES_*` vars                                                                                                        |
| `GIPHY_API_KEY`        | (empty)     | GIPHY API key. There is no admin setting for it: without it the GIF picker is disabled and direct-URL GIF bans cannot resolve uploaders                                          |
| `FCM_CREDENTIALS_FILE` | (empty)     | Path to the Firebase service-account JSON for native push. Compose mounts `./fcm-service-account.json` read-only at `/app/fcm-service-account.json`. Also needs the `push_enabled` site setting turned on |

**First-boot seeds for site settings**

At startup the app uppercases every site-setting key and, when an env var of that name exists, uses its value as that setting's default. Missing settings are then written into the database with those defaults the first time the app boots. From then on the stored row wins and the env var is ignored, so these variables only bite on a fresh database and editing one later changes nothing.

| Variable            | Seeds                | Default                 | Description                                                                                                                                                                                                |
|---------------------|----------------------|-------------------------|--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `BASE_URL`          | `base_url`           | `http://localhost:4323` | Public base URL, used for CORS and absolute links. No admin field today, so the seeded value sticks                                                                                                          |
| `UPLOAD_DIR`        | `upload_dir`         | `uploads`               | Directory for uploaded files (relative to working dir). No admin field today, so the seeded value sticks                                                                                                     |
| `LOG_LEVEL`         | `log_level`          | `info`                  | Initial log level, overridable from the admin panel at runtime                                                                                                                                              |
| `VALKEY_URL`        | `valkey_url`         | (empty)                 | App cache connection URL, separate from the LiveKit Valkey. Normally left empty and enabled from **Admin → Settings → Cache (Valkey)** (`redis://valkey-cache:6379/0` in Docker, `redis://localhost:6381/0` on the host) |
| `HYPERBEAM_API_KEY` | `hyperbeam_api_key`  | (empty)                 | Hyperbeam API key for virtual-browser watch parties. Now set in admin, see below                                                                                                                            |
| `HYPERBEAM_REGION`  | `hyperbeam_region`   | `EU`                    | Default Hyperbeam VM region (`NA`, `EU`, or `AS`), overridable per session from the start-party dialog. Now set in admin, see below                                                                          |

Any site-setting key works this way, not just the rows above, but these are the ones worth setting before the first boot.

> **Hyperbeam moved.** `HYPERBEAM_API_KEY` and `HYPERBEAM_REGION` are now the `hyperbeam_api_key` and `hyperbeam_region` site settings, edited under **Admin → Settings → Watch Parties, Voice & Streaming**, and the env vars survive only as first-boot seeds.

`LOG_FORMAT` is the one observability knob that is a plain env var rather than a site setting. Set `LOG_FORMAT=json` and the logger writes structured JSON to stdout for a collector to parse; leave it unset and you get the human-readable `ConsoleWriter` output. `docker-compose.prod.yml` sets it, the dev compose does not. It is deliberately not hot-reloadable, because changing the stdout format underneath a running collector would break its parse stages mid-stream.

`.env.example` ships a deliberate subset: `GIPHY_API_KEY`, `POSTGRES_PORT`, and `VALKEY_URL` are all read by the app but are not in the template, so add them by hand when you need them.

Everything else (registration mode, maintenance mode, turnstile keys, upload limits, rate limits, log level, email provider and SMTP settings, LiveKit and streaming credentials, chatbot configuration, default theme) is stored in the database via the `site_settings` table and editable from the admin panel at runtime with hot reload. The env file is only for things that must exist before the DB is reachable, and for the handful of secrets that never round-trip through the DB.

### Running Locally

The Go binary embeds the built frontend (`//go:embed static/*` in `server.go`) and reads `static/index.html` at startup, and `static/` is gitignored. On a fresh clone nothing Go-side will even compile until that directory exists, so build the frontend once first:

```bash
cd frontend
npm ci
npm run build     # writes ../static/
```

If you only care about compiling and testing the backend, a placeholder is enough: `mkdir -p static && touch static/.gitkeep`, which is exactly what CI does.

The app also needs Postgres reachable before it'll boot. Two paths:

**Option A: Run only Postgres in Docker, the app on the host**

```bash
# start just the postgres service from compose
docker compose up -d postgres

# backend (from repo root), connects to Postgres on host port 5007
POSTGRES_HOST=localhost POSTGRES_PORT=5007 go run .

# frontend (separate terminal)
cd frontend
npm run dev
```

Work against `http://localhost:5173`, the Vite dev server. It proxies `/api`, `/api/v1/ws`, `/uploads`, and `/sitemap` through to the Go server on `:4323`. Hitting `:4323` directly serves the last `npm run build` output from `static/`, not your live edits.

**Option B: Run the full stack in Docker**

A bare `docker compose up -d --build` starts every service in the file, including `livekit`, `livekit-ingress`, and `livekit-egress`, which bind-mount the gitignored `livekit.yaml`, `ingress.yaml`, and `egress.yaml`. Unless you have already done the [Voice Chat](docs/DEPLOYMENT.md#voice-chat-livekit) and [Live Streaming](docs/DEPLOYMENT.md#live-streaming-livekit-ingress) setup, Docker creates empty directories in their place and those three containers crash-loop. Name the app service instead, and compose brings up the two it depends on (`postgres` and `valkey-cache`) with it:

```bash
docker compose up -d --build umineko-city-of-books
```

Visit `http://localhost:2312`. The container picks up `POSTGRES_HOST=postgres` from `.env`, which compose loads via `env_file`, which is why `.env.example` ships `postgres` rather than `localhost` as the host.

The backend serves on `:4323` (mapped to `:2312` from the host under docker-compose).

**The first user to register is automatically assigned the super admin role**, so start there to unlock the admin panel.

## Database and Migrations

All migrations live in `internal/db/migrations/` and are embedded into the binary via `go:embed`. They run automatically on startup via goose against the configured Postgres database (`db.Migrate` in `internal/db/db.go`).

The schema started as a single consolidated initial migration squashed during the SQLite-to-Postgres cutover. Everything since is a fresh migration stacked on top of it, around fifty of them now.

**Always create migrations with the goose CLI**, never by hand, so the timestamp format stays consistent:

```bash
goose -dir internal/db/migrations create <name> sql
```

goose is a library dependency here, not a `tool` directive in `go.mod` (mockery, staticcheck and sqlc are), so install the CLI separately if you do not already have it:

```bash
go install github.com/pressly/goose/v3/cmd/goose@latest
```

Then edit the generated file to fill in the `-- +goose Up` and `-- +goose Down` sections. On next `go run .` the migration applies automatically.

**Regenerate sqlc after every migration.** `sqlc.yaml` reads `internal/db/migrations` as its schema, so a migration changes what sqlc validates queries against and what types it emits:

```bash
./scripts/regen_sqlc.sh
```

Adding a column or table produces a diff in `internal/dao/sqlcgen/models.go` even when no query uses it, because sqlc emits a struct per table. Renaming or dropping a column a query does use makes generation *fail*, naming the query file, which is the point. CI fails the build if the committed generated code does not match its inputs.

**The Down half is not optional.** `internal/dao/migration_roundtrip_test.go` migrates a throwaway database all the way up, rolls it back to `20260726205627` with `db.MigrateDownTo`, then migrates up again. A missing or broken `-- +goose Down` in any migration newer than that fails the test suite, so write the rollback at the same time as the forward change rather than leaving it empty.

To inspect the database directly (host-side, via the mapped port):

```bash
psql -h localhost -p 5007 -U umineko -d umineko_city_of_books
\dt
\d theories
```

Or from inside the running postgres container:

```bash
docker compose exec postgres psql -U umineko -d umineko_city_of_books
```

## Development Workflow

### Backend

```bash
go build ./...            # compile
go vet ./...              # static analysis
go tool staticcheck ./... # linter, pinned by the tool directive in go.mod
go test ./...             # run tests
./scripts/test.sh         # regenerate mocks, then vet, staticcheck, test
./scripts/regen_mocks.sh  # regenerate mockery mocks only
./scripts/regen_sqlc.sh   # expand query templates, then regenerate sqlc
```

All of those need `static/` to exist first, because the root package embeds it. See [Running Locally](#running-locally). The repository and DAO tests boot a real Postgres through testcontainers-go, so Docker has to be running for `go test ./...` to pass.

### Generated code

Three categories of Go file in this repository are generated, committed, and must never be hand-edited. Each carries a `Code generated ... DO NOT EDIT.` header.

| What | From | Regenerate with |
| ----------------------------------- | ------------------------------------------ | -------------------------- |
| `*_mock.go`                          | `.mockery.yml` plus the interface it mocks  | `./scripts/regen_mocks.sh` |
| `internal/dao/sqlcgen/*.go`          | `internal/dao/queries/*.sql` + the migrations | `./scripts/regen_sqlc.sh`  |
| `internal/dao/queries/gen/*.sql` and `internal/dao/*_bind_gen.go` | `internal/dao/queries/templates/*.tmpl` | `./scripts/regen_sqlc.sh` |

Interfaces listed in `.mockery.yml` get a mock generated next to the interface and named after it, so `Service` becomes `service_mock.go`. `internal/repository` and `internal/dao` are configured with `all: true`, so every interface in those packages is mocked without being listed individually. Regenerate whenever you add or change an interface signature.

The third row exists because sqlc needs a literal table name, which the generic DAOs (`commentDAO[K]`, `mediaDAO`, `likeDAO`, `viewDAO`, `voteDAO`, `ownedDAO`) do not have: they are parameterised by table so that nine comment systems share one implementation. A small generator in `internal/dao/queries/expand` expands one template per shape into a query set per entity before sqlc runs, so the source of truth stays a single template. Adding a tenth commentable entity is one line in that generator's registry.

`sqlc` runs with `CGO_ENABLED=0`, which selects the pure-Go wasm Postgres parser instead of the cgo one. That removes any dependency on a working local C toolchain, so the step behaves identically on Windows, on Linux and in CI.

Two DAOs are deliberately **not** generated and live in `internal/dao/dynamicsql`, plus `internal/dao/chat_dynamic.go`. They build their SQL text at run time, which sqlc cannot validate: search chooses between one and twenty-two `UNION ALL` branches from the caller's entity list, upload discovers its tables from `information_schema`, and the two chat room listings assemble a `WHERE` from five independent optional filters. The bar for living there is that the query text cannot exist until run time, not that it is awkward to write.

CI (`.github/workflows/ci.yml`) creates a `static/.gitkeep` placeholder, then runs `go vet ./...`, `go tool staticcheck ./...`, a generated-code drift check, `go test ./... -count=1`, and `go build ./...` in that order. The drift check reruns both generators and fails if anything changes, so stale generated code cannot be merged. Putting `[skip tests]` in the commit message skips the test step only.

### Frontend

```bash
cd frontend
npm run dev         # dev server with HMR on :5173
npm run build       # tsc + vite build into ../static/
npm run typecheck   # tsc -b only, no bundle
npm test            # vitest run
npm run test:watch  # vitest in watch mode
npm run test:coverage
npm run lint        # oxlint, --max-warnings=0
npm run lint:fix    # oxlint with autofix
npm run format      # oxfmt check
npm run format:fix
```

Tests are vitest and React Testing Library under jsdom, colocated with the code (`Foo.tsx` next to `Foo.test.tsx`), with the shared render helpers, fixtures and jsdom setup in `frontend/src/test-utils/`. `tsconfig.json` includes the test files, so `npm run typecheck` (and therefore `npm run build`) typechecks them too.

CI runs `npm run format`, `npm run lint`, `npm test`, then `npm run build`. Run the same four before committing frontend changes; all of them need to pass cleanly.

### Mobile app (Capacitor)

The same React frontend is packaged as a native iOS/Android app via Capacitor, with the project living in `frontend/`. The build commands, local iteration against a live dev server, OTA bundle signing, the app's baked-in API base URL, and native push are all in [docs/MOBILE.md](docs/MOBILE.md).

## Deployment

### Self-hosted Docker

```bash
docker compose up -d --build
```

This builds the multi-stage image locally (frontend -> static assets -> Go binary -> Alpine runtime with FFmpeg and libwebp-tools) and runs it on port `2312` by default, forwarding to the container's `:4323`.

`docker-compose.yml` defines seven services, not just the app and its database: `postgres`, `umineko-city-of-books`, `valkey-cache` (the app cache), `valkey` (the LiveKit coordination bus), `livekit`, `livekit-ingress`, and `livekit-egress`. The three LiveKit services bind-mount `livekit.yaml`, `ingress.yaml` and `egress.yaml`, all of which are gitignored, so on a fresh clone they restart-loop until you copy the templates (see [docs/DEPLOYMENT.md](docs/DEPLOYMENT.md)). The same applies to the `./fcm-service-account.json` mount: create the file, or drop the mount, unless you are using native push, otherwise Docker creates a directory in its place.

If you do not want voice, streaming or HLS, bring up the core only and let compose pull in its own dependencies:

```bash
docker compose up -d --build umineko-city-of-books
```

That starts `postgres` and `valkey-cache` (both declared under `depends_on`) and nothing else.

### Persistent Data

Two stores hold real site data and need to survive container rebuilds:

- **Postgres data**, the named docker volume `postgres_data` mounted at `/var/lib/postgresql` inside the postgres container. Survives `docker compose up -d` and image upgrades.
- **Uploaded media**, the `umineko-city-of-books` service bind-mounts `./data:/app/data` so `data/uploads/` lives on the host. Set `UPLOAD_DIR=data/uploads` in your `.env` so the app reads from this mount. Note that `UPLOAD_DIR` only seeds the initial default of the `upload_dir` site setting; once it has been saved from the admin panel, the stored value wins.

The container runs as a non-root user (uid `10001`, `cap_drop: ALL`, `no-new-privileges`), so the host `./data` directory has to be writable by that uid. Live HLS segments land under the same mount at `data/hls/`, and the app does the cleanup itself, removing each per-stream directory when the broadcast ends and sweeping orphans on the reconcile pass, so it needs write access there and not only read.

`./fcm-service-account.json` is bind-mounted read-only into the container by both compose files and is gitignored. Create it (or remove the mount) before the first `up`, otherwise Docker creates a directory in its place.

`docker-compose.prod.yml` adds a third named volume, `valkey_data`, for the host-networked LiveKit coordination valkey it runs with `--appendonly yes`. That holds ephemeral SFU coordination state rather than site data, so it does not need backing up.

For backups: a daily `pg_dump | gzip` cron is the recommended path for the database, plus a periodic tarball of `./data/uploads/` for media. Restore via `gunzip -c <dump>.sql.gz | docker compose exec -T postgres psql -U umineko -d umineko_city_of_books`.

### Prebuilt image, reverse proxy, voice and streaming

Running the published `ghcr.io` image, the Caddy reverse-proxy configuration, and the three LiveKit deployments ([voice SFU](docs/DEPLOYMENT.md#voice-chat-livekit), [streaming ingress](docs/DEPLOYMENT.md#live-streaming-livekit-ingress), [HLS egress](docs/DEPLOYMENT.md#smooth-playback-livekit-egress--hls)) are all in [docs/DEPLOYMENT.md](docs/DEPLOYMENT.md). They are one document rather than four because anyone enabling voice is also configuring the proxy and opening the firewall.

## Adding a New Page

A new page or section has to be wired into ten places: OG tags, the admin content rules, the sidebar, the profile home-page dropdown, the lazy page exports, the frontend routes, the backend routes, the sitemap, the content filter, and search. Missing any one of them fails quietly rather than loudly, so the full checklist lives on one screen in [docs/ADDING_A_PAGE.md](docs/ADDING_A_PAGE.md).

## Documentation

- [ARCHITECTURE.md](docs/ARCHITECTURE.md): how the code is arranged and why. The backend layering, the request lifecycle, a chapter per subsystem, and the reasoning behind the stack above.
- [PROVENANCE.md](PROVENANCE.md): authorship, the project timeline, and third-party code.
- [docs/FEATURES.md](docs/FEATURES.md): the full catalogue behind the twenty-four entries above, with the rules and limits of each.
- [docs/DEPLOYMENT.md](docs/DEPLOYMENT.md): the prebuilt image, reverse proxy, and LiveKit voice, ingress and HLS egress setup.
- [docs/MOBILE.md](docs/MOBILE.md): the Capacitor app. Builds, local iteration, OTA bundles, API base URL, native push.
- [docs/ADDING_A_PAGE.md](docs/ADDING_A_PAGE.md): the ten places a new page has to be registered.
- `design/updates.md`: the public changelog, alongside the other working trackers. `design/` holds documents with a lifecycle, meant to be worked through and eventually closed, and is excluded by `.gitignore`, which is why this entry is a path rather than a link. `docs/` is the other half and holds durable reference that stays true until the code changes.

## Provenance

The source in this repository is written by hand. Authorship, the project timeline, commit-message tooling, and third-party code are set out in [PROVENANCE.md](PROVENANCE.md).

## License

Released under the [MIT License](LICENSE). Umineko no Naku Koro ni and the wider When They Cry series are © 07th Expansion; this project is an unofficial fan platform with no affiliation.
