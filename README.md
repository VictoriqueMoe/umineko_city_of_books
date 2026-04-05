# Umineko City of Books

A community platform for fans of Umineko no Naku Koro ni, Higurashi, and the wider When They Cry series. The original goal was a place to declare fan theories as **blue truth**, attach quotes from the game as evidence, and have them debated on two sides: **"With love, it can be seen"** and **"Without love, it cannot be seen"**. It has since grown into a full social platform: theory debates, a Twitter-style game board, mystery boards, fan art galleries, ship declarations, DMs, live notifications, and themed role-based moderation.

## Table of Contents

- [Features](#features)
  - [Theory Debates](#theory-debates)
  - [Mysteries](#mysteries)
  - [Gallery and Art](#gallery-and-art)
  - [Ships](#ships)
  - [Game Board](#game-board)
  - [Chat and DMs](#chat-and-dms)
  - [Announcements](#announcements)
  - [Profiles and Social Graph](#profiles-and-social-graph)
  - [Notifications](#notifications)
  - [Moderation and Admin](#moderation-and-admin)
  - [Platform Features](#platform-features)
- [Tech Stack](#tech-stack)
- [Architecture](#architecture)
- [Getting Started](#getting-started)
- [Database and Migrations](#database-and-migrations)
- [Development Workflow](#development-workflow)
- [Deployment](#deployment)
- [Adding a New Page](#adding-a-new-page)

## Features

### Theory Debates

The original heart of the site. Submit a fan theory as a **blue truth**, attach quote evidence, and let others refute or support it.

- Theory declarations with title, body, and episode scope
- Evidence attachment by searching any quote in the game (including narrator lines) via the Umineko Quote Finder API
- Two-sided debate with **"With love, it can be seen"** (support) and **"Without love, it cannot be seen"** (deny), each with its own evidence
- **Credibility score** per theory (0 to 100), weighted by the truth type of evidence attached to responses (gold > red > purple > blue > none)
- Threaded replies on responses with flat rendering and @username attribution
- Upvotes and downvotes on both theories and responses, separate from the credibility score
- Per-series filtering so Higurashi theories do not bleed into Umineko

### Mysteries

A gamified puzzle mode where a user (the Game Master) poses a mystery with graduated clues, and other players submit attempts.

- Create mysteries with difficulty, body, and an ordered list of clues tagged by truth type (red/blue/gold/purple)
- Attempts are threaded with reply chains
- Game master marks the winning attempt, which pins to the top of the page
- Piece counter showing how many players have attempted
- **Leaderboard** of top solvers
- Role-based visibility: as a **super admin** you see attempts grouped by player (GM-style view) with collapsible groups, player pills, and red-dot unread indicators backed by a localStorage read cursor. Admins, mods, and regular players see the normal flat thread view
- Real-time updates over WebSocket when new attempts or replies land
- Separate notification categories: **Mysteries (as Game Master)** and **Mysteries (as Player)**

### Gallery and Art

Fan art uploads with full social features.

- Upload drawings, screenshots, edits, and other image types with tags, corner, and description
- Automatic WebP conversion and thumbnail generation
- **Galleries**: bundle related art into named collections with cover image and preview strip
- Tag browsing with popular tag listings per corner
- Full comment system with threading, media uploads, embeds, and likes
- Lightbox viewer for full-size images
- View counts unique per viewer (hashed user ID or IP)
- Per-corner filtering (Umineko, Higurashi, Ciconia)

### Ships

Declare character pairings and rally votes for them.

- Pick characters from Umineko or Higurashi via a character picker, or add original characters
- Mixed-series ships are supported
- Optional ship image with automatic WebP conversion and lightbox viewer
- Upvote and downvote each ship, sorted by popularity
- Ships that drop below a threshold automatically get the **Crackship** badge
- Inline edit form on the ship detail page for authors and admins
- Full comment system with threading, media, and likes
- Filter by series or by individual character
- Sort modes: new, top, crackships only, most commented, controversial

### Game Board

A Twitter-style social feed for off-topic posts and discussion.

- Posts with title, body, multiple images or video, likes, threaded comments
- **Corners**: dedicated sub-feeds for Umineko, Higurashi, and Ciconia, each with its own post count and content rules
- **@Mentions** with autocomplete in posts and comments, mentioned users get notified
- **Link embeds**: YouTube links embed inline, other URLs render rich OG preview cards (title, image, description, site name). Embeds refresh daily
- Relevance-based feed algorithm with deterministic jitter for stable pagination
- Following tab showing only posts from users you follow
- Unique post view counts
- Live like counters pushed over WebSocket
- Comment media uploads (images and video) with the shared MediaPicker component
- Editable posts and comments with an "(edited)" marker and notification to commenters

### Chat and DMs

- Direct messages and group rooms backed by SQLite
- Per-user DM enable/disable toggle in profile settings
- Unread counts and last-read cursors
- Notifications for new chat messages

### Announcements

Site-wide announcements with pinning.

- Admins post announcements visible to everyone
- Pinned announcements stay at the top
- Full comment system reusing the shared CommentItem component, with threading, media, embeds, and likes

### Profiles and Social Graph

- Avatar, draggable banner positioning, bio, pronouns, gender, social links (Twitter, Discord, Tumblr, Waifulist, GitHub, personal site), favourite character
- Activity feed with recent theories, responses, posts, and comments
- Tabs for posts, theories, art, ships, mysteries, and galleries
- Stats box: theory count, response count, votes received, ship count, mystery count
- Follow system with follower and following lists, "Follows you" label, follower counts
- Online/offline status
- **Players Page**: browse all users grouped by role (Reality Authors, Voyager Witches, Witches) and online/offline status
- Per-user **blocks** with enforcement across feeds, comments, and DMs
- Configurable home page: each user picks their default landing page (Game Board, Theories, Ships, etc.)
- Email with optional public visibility and per-user email notification toggle
- Episode progress slider used for spoiler gating on mystery pages

### Notifications

- Real-time WebSocket push with automatic reconnection
- Email notifications with HTML templates, deep links, and per-user opt-out
- Grouped by category on the notifications page: Game Board, Gallery, Theories, Mysteries (as GM), Mysteries (as Player), Social, Moderation
- Types covered: theory response, response reply, theory upvote, response upvote, chat message, report, report resolved, new follower, post liked, post commented, post comment reply, mention, art liked, art commented, art comment reply, comment liked, content edited, mystery attempt, mystery reply, mystery attempt vote, mystery solved, ship commented, ship comment reply, ship comment liked, announcement commented, announcement comment reply, announcement comment liked
- ETag-based polling fallback when the WebSocket drops

### Moderation and Admin

- **Role system** with themed names and colour-coded usernames with glow:
  - **Reality Author** (super admin)
  - **Voyager Witch** (admin)
  - **Witch** (moderator)
- Permission-based authorisation layer (`internal/authz`), not a raw role check
- Admin dashboard with site stats: total users, theories, responses, posts, comments, per-corner breakdown, 24h/7d/30d growth windows, most active users
- User management: assign or revoke roles, ban with reason, unban
- DB-backed site settings with hot reload: body limits, log level, registration mode, maintenance mode, turnstile, upload limits, rate limits
- **Invite system**: open, invite-only, or closed registration. Admins generate one-time invite codes
- **Maintenance mode** with custom title and message. Admins bypass it
- **Audit log** for admin actions
- **Reports**: users can report theories, responses, posts, comments, art, ships, and users. Admins resolve from the admin panel with optional comment back to the reporter
- **Content rules** per section (theories, general game board, each corner, mysteries, ships, gallery, announcements), admin-editable and displayed at the top of each page
- **Cloudflare Turnstile** on login and registration, toggle-able from admin settings

### Platform Features

- **Five themes**: Featherine (gold/purple, default), Beatrice (warm gold/brown), Bernkastel (blue), Lambdadelta (pink), Erika Furudo (cyan/pink)
- **OG embeds** for rich previews when sharing links to theories, posts, profiles, mysteries, ships, and art on Twitter and Discord
- **Auto-generated sitemap** with a sitemap index and sub-sitemaps for static pages, theories, posts, users, ships, mysteries, and galleries
- **Media processing**: image-to-WebP and video-to-MP4 encoding via a background worker pool, local FFmpeg thumbnail generation
- **Client-side validation** of file sizes before upload, pulled from live server settings
- **Structured logging** with zerolog, configurable log levels, settings change listener pattern
- Fully **mobile responsive** across all pages
- **Cache headers**: `/static/assets/*` is immutable, HTML is `no-store`, API responses are `no-cache`

## Tech Stack

**Backend**

- Go 1.26
- Fiber v3 (HTTP router)
- SQLite via `modernc.org/sqlite` (pure Go, no CGO)
- Goose for migrations
- fasthttp/websocket for the WebSocket hub
- zerolog for structured logging
- wneessen/go-mail for email delivery

**Frontend**

- React 19 with TypeScript 5.9
- Vite 8
- React Router v7 (not react-router-dom)
- CSS Modules
- DOMPurify + marked for safe markdown rendering
- @marsidev/react-turnstile for bot protection

**Infrastructure**

- Docker multi-stage build (Node build stage + Go build stage + Alpine runtime)
- FFmpeg and libwebp-tools in the runtime image
- Designed to sit behind Caddy or another reverse proxy in production
- Session auth with httpOnly cookies, no JWTs

**External**

- [Umineko Quote Finder API](https://quotes.auaurora.moe/swagger/index.html) for game quote search and evidence attachment

## Architecture

### Project Layout

```
.
├── main.go                  Entry point, bootstraps logger and server
├── server.go                Fiber setup, middleware, route registration
├── internal/
│   ├── admin/               Admin dashboard business logic
│   ├── art/                 Gallery, art, and comments service
│   ├── auth/                Login, registration, sessions, turnstile
│   ├── authz/               Permission model (role -> perms mapping)
│   ├── block/               User block enforcement
│   ├── chat/                DMs and group rooms
│   ├── config/              Env config and setting definitions
│   ├── controllers/         HTTP handlers (Fiber route functions)
│   ├── credibility/         Theory credibility score calculator
│   ├── db/                  DB open, migrations, WithTx helper
│   │   └── migrations/      Goose migrations
│   ├── dto/                 Request/response types shared with frontend
│   ├── email/               SMTP client and templates
│   ├── follow/              Follow graph
│   ├── logger/              zerolog init and helpers
│   ├── media/               Image and video processing worker pool
│   ├── middleware/          Auth, rate limit, etag, cache headers
│   ├── notification/        Notify fan-out, email templates
│   ├── og/                  Open Graph meta generation per route
│   ├── post/                Game Board posts and comments
│   ├── profile/             User profile service
│   ├── quotefinder/         Upstream quote API client
│   ├── report/              User-submitted reports
│   ├── repository/          All SQL code, one file per domain
│   ├── role/                Role enum and helpers
│   ├── routes/              Route group registration
│   ├── session/             Session storage
│   ├── settings/            DB-backed settings with hot reload
│   ├── ship/                Ships and crackships
│   ├── social/              @mention parsing and fan-out
│   ├── theory/              Theory and response service
│   ├── upload/              Upload file handling
│   ├── user/                Users, profiles, bans
│   ├── utils/               Tree builders, graceful shutdown, helpers
│   └── ws/                  WebSocket hub and event broadcast
├── frontend/
│   ├── src/
│   │   ├── api/             API client (`apiFetch`, `apiPost`, etc.)
│   │   ├── components/      Shared components (Button, Modal, CommentItem, MediaPicker, ProfileLink, etc.)
│   │   ├── contexts/        AuthContext, NotificationsContext, ThemeContext
│   │   ├── hooks/           useAuth, useWebSocket, useNotifications
│   │   ├── pages/           One folder per page/section
│   │   ├── themes/          Five CSS theme files
│   │   ├── types/           TypeScript types mirroring the Go DTOs
│   │   └── utils/           notifications, permissions, linkify, mentions
│   └── package.json
├── static/                  Vite build output, embedded into the Go binary
├── uploads/                 Runtime upload directory
├── Dockerfile               Multi-stage production build
├── docker-compose.yml       Local/self-host compose file
└── docker-compose.prod.yml  Compose file pulling the prebuilt GHCR image
```

### Data Layer

- **All SQL lives in `internal/repository/`**, one file per domain (theory.go, post.go, art.go, mystery.go, ship.go, etc.).
- **Transactions** use the `db.WithTx(ctx, db, func(tx) error)` helper in `internal/db/tx.go`. Repo methods that touch multiple tables wrap themselves in `WithTx` and expose a single combined method (e.g. `CreateWithCharacters`, `UpdateWithTags`, `MarkSolved`). Services do not handle transactions directly.
- **Foreign keys** are enabled via `PRAGMA foreign_keys=ON`. Most deletes cascade through ON DELETE CASCADE. `galleries -> art.gallery_id` is `ON DELETE SET NULL`, so the gallery delete path explicitly removes child art inside a transaction.
- **WAL mode** is enabled (`PRAGMA journal_mode=WAL`) for concurrent reads.

### Auth and Sessions

- Server-side sessions stored in SQLite with httpOnly cookies.
- No JWTs. The session ID is the only thing in the cookie.
- Session renewal handled by middleware, cleanup runs on a timer.

### Permission Model

- Every action is gated on a **permission**, not a raw role check. Permissions include `edit_any_theory`, `edit_any_post`, `delete_any_post`, `ban_users`, `view_reports`, etc.
- Roles map to permission sets in `internal/authz/`. `super_admin` gets `PermAll`, `admin` gets most things, `moderator` gets moderation-adjacent permissions.
- Some features (the "game master" view in mysteries) check `role == super_admin` directly because the behaviour is intentionally scoped to that one role, not the permission grant.

### WebSocket Hub

- Single hub (`internal/ws`) managing all connected clients, keyed by user ID.
- Broadcasts to individual users (notifications) or broadcast groups (like counters, mystery updates).
- Clients auto-reconnect with exponential backoff on the frontend side.

## Getting Started

### Prerequisites

- Go 1.26 or newer
- Node.js LTS
- FFmpeg (for video transcoding and thumbnails)
- libwebp-tools (`cwebp`) for WebP conversion
- SQLite 3 CLI (optional, handy for poking at the DB)

### Environment

Copy `.env.example` to `.env` and adjust:

```bash
cp .env.example .env
```

| Variable           | Default                 | Description                                        |
|--------------------|-------------------------|----------------------------------------------------|
| `DB_PATH`          | `truths.db`             | Path to SQLite database file                       |
| `UPLOAD_DIR`       | `uploads`               | Directory for uploaded files                       |
| `BASE_URL`         | `http://localhost:4323` | Public base URL, used for CORS and absolute links  |
| `LOG_LEVEL`        | `info`                  | Log level: trace, debug, info, warn, error, fatal  |
| `MAX_BODY_SIZE`    | `52428800`              | Fiber request body limit in bytes (default 50MB)   |
| `MAX_IMAGE_SIZE`   | `10485760`              | Max image upload size in bytes (default 10MB)      |
| `MAX_VIDEO_SIZE`   | `104857600`             | Max video upload size in bytes (default 100MB)     |
| `MAX_GENERAL_SIZE` | `52428800`              | Max other file upload size in bytes (default 50MB) |

Most runtime behaviour (registration mode, maintenance mode, turnstile keys, upload limits, rate limits, log level, email SMTP settings) is stored in the database via the `site_settings` table and editable from the admin panel at runtime with hot reload. The env file is only for things that must exist before the DB is reachable.

### Running Locally

```bash
# Backend (from repo root)
go run .

# Frontend (separate terminal)
cd frontend
npm install
npm run dev
```

The backend serves on `:4323`. The Vite dev server proxies `/api`, `/uploads`, `/sitemap`, and WebSocket upgrades to the Go server.

**The first user to register is automatically assigned the super admin role**, so start there to unlock the admin panel.

## Database and Migrations

All migrations live in `internal/db/migrations/` and are embedded into the binary via `go:embed`. They run automatically on startup via goose.

**Always create migrations with the goose CLI**, never by hand, so the timestamp format stays consistent:

```bash
goose -dir internal/db/migrations create <name> sql
```

Then edit the generated file to fill in the `-- +goose Up` and `-- +goose Down` sections. On next `go run .` the migration applies automatically.

To inspect the database directly:

```bash
sqlite3 truths.db
.tables
.schema theories
```

## Development Workflow

### Backend

```bash
go build ./...      # compile
go vet ./...        # static analysis
go test ./...       # run tests (sparse coverage at the moment)
```

### Frontend

```bash
cd frontend
npm run dev         # dev server with HMR
npm run build       # tsc + vite build into ../static/
npm run lint        # eslint, --max-warnings=0
npm run lint:fix    # eslint with autofix
npm run prettier    # prettier check
npm run prettier:fix
```

Run lint and build before committing frontend changes. Both need to pass cleanly.

### Project Conventions

- **React Router v7**, not `react-router-dom` (it is deprecated).
- **UK English** in copy and comments (colour, favourite, organisation). US spelling stays in code identifiers where it is conventional (e.g. CSS `color`).
- **Minimal comments**: only when logic is not self-evident.
- **No single-line `if` statements**; always brace.
- **Grouped Go declarations**: `type (...)`, `const (...)`, `var (...)` using parentheses.
- **Repository layer owns all SQL**. Services orchestrate, controllers are thin.
- **Transactions go in the repo layer** via `db.WithTx`, not in the service layer.

## Deployment

### Self-hosted Docker

```bash
docker compose up -d --build
```

This builds the multi-stage image locally (frontend -> static assets -> Go binary -> Alpine runtime with FFmpeg and libwebp-tools) and runs it on port `2312` by default, forwarding to the container's `:4323`.

### Prebuilt image

```bash
docker compose -f docker-compose.prod.yml up -d
```

This pulls `ghcr.io/victoriquemoe/umineko_city_of_books:latest` instead of building locally.

### Persistent Data

The compose file mounts `./data:/app/data` inside the container. Put `DB_PATH=/app/data/truths.db` and `UPLOAD_DIR=/app/data/uploads` in your `.env` so both the database and uploads survive container rebuilds.

### Reverse Proxy

Run behind Caddy, Nginx, or similar for TLS. The server sets the right cache headers itself (`/static/assets/*` immutable, HTML `no-store`, API `no-cache`), so the proxy only needs to forward requests and upgrade WebSocket connections on `/ws`.

## Adding a New Page

When creating a new page or section, update **all** of the following:

1. **OG tags** - `internal/og/og.go`: add path matching in `metaForPath()` and a meta method for detail pages
2. **Admin Content Rules** - `frontend/src/pages/admin/AdminContentRules.tsx`: add to `pages` array with a `rules_<page_name>` key
3. **Sidebar** - `frontend/src/components/layout/Sidebar/Sidebar.tsx`: add `<NavLink>` in the appropriate section
4. **Profile settings default page** - `frontend/src/pages/profile/SettingsPage.tsx`: add `<option>` to the Home Page dropdown
5. **Home page routes** - `frontend/src/App.tsx`: add to the `homePageRoutes` object and add a `<Route>` element
6. **Sitemap** - `internal/controllers/sitemap_controller.go`: add the URL to `static()` or create a dynamic sitemap handler for collections
