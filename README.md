# Umineko City of Books

A community platform for fans of Umineko no Naku Koro ni, Higurashi, and the wider When They Cry series. The original goal was a place to declare fan theories as **blue truth**, attach quotes from the game as evidence, and have them debated on two sides: **"With love, it can be seen"** and **"Without love, it cannot be seen"**. It has since grown into a full social platform: theory debates, a Twitter-style game board, mystery boards, fan art galleries, ship declarations, fanfiction, live reading journals, chat rooms and DMs, live notifications, and themed role-based moderation.

## Table of Contents

- [Features](#features)
  - [Theory Debates](#theory-debates)
  - [Mysteries](#mysteries)
  - [Gallery and Art](#gallery-and-art)
  - [Ships](#ships)
  - [Game Board](#game-board)
  - [Fanfiction](#fanfiction)
  - [Reading Journals](#reading-journals)
  - [Chat Rooms and DMs](#chat-rooms-and-dms)
  - [Announcements](#announcements)
  - [Suggestions](#suggestions)
  - [Quote Browser](#quote-browser)
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
- Full rich-text formatting in mystery bodies and attempts (backticks, quotes, spoilers, syntax-highlighted code fences, truth colours)

### Gallery and Art

Fan art uploads with full social features.

- Upload drawings, screenshots, edits, and other image types with tags, corner, and description
- Automatic WebP conversion and thumbnail generation
- **Galleries**: bundle related art into named collections with cover image and preview strip
- Tag browsing with popular tag listings per corner
- Full comment system with threading, media uploads, embeds, likes, GIFs, and Discord-style formatting
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
- Full comment system with threading, media, GIFs, and likes
- Filter by series or by individual character
- Sort modes: new, top, crackships only, most commented, controversial

### Game Board

A Twitter-style social feed for off-topic posts and discussion.

- Posts with title, body, multiple images or video, likes, threaded comments
- **Corners**: dedicated sub-feeds for **Umineko**, **Higurashi**, **Ciconia**, **Higanbana**, and **Rose Guns Days**, each with its own post count, content rules, and sitemap
- **@Mentions** with autocomplete in posts and comments, mentioned users get notified
- **Link embeds**: YouTube links embed inline, other URLs render rich OG preview cards (title, image, description, site name). Embeds refresh daily
- **Polls** on posts with multi-option voting, per-user vote tracking, and optional expiry
- **GIF picker** on the post composer (and every comment box) backed by GIPHY, sends instantly on pick
- **Quick Reply**: one-click Reply button drops an inline comment composer under the post without leaving the feed, auto-collapses after send
- Relevance-based feed algorithm with deterministic jitter for stable pagination
- Following tab showing only posts from users you follow
- Unique post view counts
- Live like counters pushed over WebSocket
- Comment media uploads (images and video) with the shared MediaPicker component
- Editable posts and comments with an "(edited)" marker and notification to commenters

### Fanfiction

Write and publish multi-chapter fan stories.

- Fanfic entries with title, summary, language, series tag, cover image, and OC characters
- **Chapter-based structure**: add, reorder, edit, and delete chapters individually
- **Rich text editor** (TipTap) for chapter bodies with formatting, links, alignment, and colour
- Word count, character tagging, and cross-series support
- Favourite fanfics to follow new chapters
- Full comment system with threading, media, GIFs, and likes on both the fanfic and individual chapters
- Per-fanfic sitemap inclusion

### Reading Journals

Live-blog your read-throughs of Ryukishi07's works. Post reactions, theories, and predictions as you go.

- Create a journal tied to a specific work and episode
- Post updates with media, mentions, and Discord-style formatting as you read
- Threaded comment system so other players can react to each entry without spoiling
- Follow a journal to get notified when the author posts a new update
- Journals auto-archive after a period of inactivity to keep the index clean
- Per-user daily journal cap (admin configurable)

### Chat Rooms and DMs

Real-time chat in two flavours: one-to-one direct messages and named group rooms.

- **Direct Messages**: SQLite-backed DMs with unread counts, last-read cursors, and per-user enable/disable toggle in profile settings
- **Chat Rooms**: public or private group rooms, with an optional **Roleplay** flag that switches the room into a different visual and posting style
- **Emoji reactions** on messages with live count and "you reacted" state, shown across desktop and mobile
- **Pinning**: moderators and room owners can pin messages; a dedicated pinned messages panel surfaces them
- **Member management**: per-room roles, timeouts, kick, and nickname profiles scoped to the room (ghost members supported)
- **Configurable limits**: max room members and max rooms per day are site-settings driven
- **Replies and edits** on individual messages, with a floating action bar above the bubble on hover
- **GIF picker**, emoji picker, media uploads, and full Discord-style text formatting (backticks, quotes, spoilers, syntax highlighting) everywhere text is typed
- WebSocket-driven real-time delivery, pin/unpin events, reaction updates, and typing presence
- Mobile-first composer: full-width text box with Media / GIF / Send stacked below, bubbles spanning edge to edge

### Announcements

Site-wide announcements with pinning.

- Admins post announcements visible to everyone
- Pinned announcements stay at the top
- Full markdown support in the announcement body
- Full comment system reusing the shared CommentItem component, with threading, media, embeds, and likes
- Optional site-wide announcement banner settable from the admin panel

### Suggestions

A dedicated feedback channel for site improvements and bug reports.

- Posts written in the same composer as the game board, living under a dedicated "Site Improvements" corner
- Status filters: **Open**, **Done**, **Archived**
- Admins can resolve a suggestion (mark done) or archive it, with the status reflected back to the reporter
- Follows the same commenting, voting, and notification rules as the game board

### Quote Browser

A standalone interface for browsing the full Umineko quote corpus sourced from the Umineko Quote Finder API. Search, filter by character and episode, and use it as a reference when drafting evidence for theories.

### Profiles and Social Graph

- Avatar, draggable banner positioning, bio, pronouns, gender, social links (Twitter, Discord, Tumblr, Waifulist, GitHub, personal site), favourite character
- **Per-user theme, font, wide layout, and particles preferences** persisted on the profile so it follows you across devices
- Activity feed with recent theories, responses, posts, and comments
- Tabs for posts, theories, art, ships, mysteries, galleries, fanfics, and journals
- Stats box: theory count, response count, votes received, ship count, mystery count
- Follow system with follower and following lists, "Follows you" label, follower counts
- Online/offline status
- **Players Page**: browse all users grouped by role (Reality Authors, Voyager Witches, Witches) and online/offline status
- Per-user **blocks** with enforcement across feeds, comments, and DMs
- Configurable home page: each user picks their default landing page (Game Board, Theories, Ships, Fanfiction, Journals, etc.)
- Email with optional public visibility and per-user email notification toggle
- Episode progress slider used for spoiler gating on mystery pages
- **Favourite GIFs**: star any GIF in the picker or posted by someone else to save it to a personal Favourites tab

### Notifications

- Real-time WebSocket push with automatic reconnection
- Email notifications with HTML templates, deep links, and per-user opt-out
- Grouped by category on the notifications page: Game Board, Gallery, Theories, Mysteries (as GM), Mysteries (as Player), Fanfiction, Journals, Chat, Social, Moderation
- Types covered: theory response, response reply, theory upvote, response upvote, chat message, chat reaction, report, report resolved, new follower, post liked, post commented, post comment reply, mention, art liked, art commented, art comment reply, comment liked, content edited, mystery attempt, mystery reply, mystery attempt vote, mystery solved, ship commented, ship comment reply, ship comment liked, announcement commented, announcement comment reply, announcement comment liked, fanfic chapter published, fanfic commented, journal updated, journal commented
- ETag-based polling fallback when the WebSocket drops

### Moderation and Admin

- **Role system** with themed names and colour-coded usernames with glow:
  - **Reality Author** (super admin)
  - **Voyager Witch** (admin)
  - **Witch** (moderator)
- **Vanity Roles**: admin-defined custom roles with bespoke colour, label, and sort order. Assign one or more to a user independently of their moderation role. System-level vanity roles are distinguished from user-created ones
- Permission-based authorisation layer (`internal/authz`), not a raw role check
- Admin dashboard with site stats: total users, theories, responses, posts, comments, per-corner breakdown, 24h/7d/30d growth windows, most active users
- User management: assign or revoke roles, ban with reason, unban, assign vanity roles
- DB-backed site settings with hot reload: body limits, log level, registration mode, maintenance mode, turnstile, upload limits, rate limits, announcement banner, SMTP, Sentry/GlitchTip DSN, default theme
- **Invite system**: open, invite-only, or closed registration. Admins generate one-time invite codes
- **Maintenance mode** with custom title and message. Admins bypass it
- **Audit log** for admin actions
- **Reports**: users can report theories, responses, posts, comments, art, ships, users, fanfics, journals, and chat messages. Admins resolve from the admin panel with optional comment back to the reporter
- **Banned GIFs**: admins block specific GIPHY IDs from being embedded anywhere on the site; the content filter rejects matches before they render
- **Content Filter Pipeline** (`internal/contentfilter`): pluggable rule-based validation that runs on all user-generated text before it lands in the DB
- **Content rules** per section (theories, general game board, each corner, mysteries, ships, gallery, fanfiction, journals, suggestions, chat rooms, announcements), admin-editable and displayed at the top of each page
- **Per-action rate limits**: max theories, responses, posts, art, journals, and chat rooms per day, all settable from the admin panel
- **Cloudflare Turnstile** on login and registration, toggle-able from admin settings

### Platform Features

- **Ten themes**: Featherine (gold/purple, default), Beatrice (warm gold/brown), Bernkastel (blue), Lambdadelta (pink), Erika Furudo (cyan/pink), Battler, Virgilia (light mode), Rika, Mion, Satoko
- **Two font families**: default serif set or **IM Fell English** for a period-correct look, per-user preference
- **Wide layout toggle** and **ambient particles toggle** (floating butterflies plus theme-specific motifs such as candy and lollipops on Lambdadelta)
- **Discord-style text formatting** across posts, comments, DMs, chat rooms, mysteries, and art/ship descriptions:
  - Backticks for inline code, triple backticks for multi-line code blocks with syntax highlighting via highlight.js
  - `>` for block quotes that flow across wrapped lines and terminate on a blank line
  - `||spoiler||` for hover-to-reveal spoilers
  - Truth colours (`[red]...[/red]` etc.) that still glow inside quotes
- **GIPHY integration** on posts, comments, DMs, and chat rooms with Trending and per-user Favourites tabs, one-click send, and an admin banlist
- **OG embeds** for rich previews when sharing links to theories, posts, profiles, mysteries, ships, art, fanfics, journals, and chat rooms on Twitter and Discord, with locale, image dimensions, and canonical URL tags
- **Auto-generated sitemap** with a sitemap index and sub-sitemaps for static pages, theories, posts, users, ships, mysteries, galleries, fanfics, and journals
- **Media processing**: image-to-WebP and video-to-MP4 encoding via a background worker pool, local FFmpeg thumbnail generation
- **Client-side validation** of file sizes before upload, pulled from live server settings
- **Auto-expanding composers**: every text box grows as you type, capped at half the viewport before scrolling internally
- **Structured logging** with zerolog, configurable log levels, settings change listener pattern
- **GlitchTip / Sentry** error tracking via a configurable DSN, with structured attribute mapping in `internal/logger/glitchtip_shipper.go`
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
- getsentry/sentry-go for GlitchTip / Sentry error tracking
- disintegration/imaging for server-side image manipulation

**Frontend**

- React 19 with TypeScript 6
- Vite 8
- React Router v7 (not react-router-dom)
- CSS Modules
- DOMPurify + marked for safe markdown rendering
- highlight.js for syntax-highlighted code blocks
- TipTap 3 (with StarterKit, Placeholder, TextAlign, Color, TextStyle extensions) for the fanfiction rich text editor
- emoji-picker-react for chat reactions and emoji insertion
- @marsidev/react-turnstile for bot protection

**Infrastructure**

- Docker multi-stage build (Node build stage + Go build stage + Alpine runtime)
- FFmpeg and libwebp-tools in the runtime image
- Designed to sit behind Caddy or another reverse proxy in production
- Session auth with httpOnly cookies, no JWTs
- Mockery v3 (`.mockery.yml`) for generated Go interface mocks

**External**

- [Umineko Quote Finder API](https://quotes.auaurora.moe/swagger/index.html) for game quote search and evidence attachment
- GIPHY API for GIF search, trending, and favourites

## Architecture

The server is a single Go binary that embeds the compiled Vite bundle and serves both the SPA and the JSON API from one process. Every layer has a single responsibility: controllers parse HTTP, services orchestrate business logic, repositories own SQL, the hub owns live events, and the media processor owns encoding off the hot path.

### High-Level Component Map

```
                      ┌──────────────────────────────────────────────┐
                      │                  Browser                     │
                      │        React 19 SPA   +   WebSocket          │
                      └───────┬──────────────────────────┬───────────┘
                              │ HTTP                     │ WS
                              ▼                          ▼
        ┌─────────────────────────────────────────────────────────────┐
        │                       Fiber v3 app                          │
        │  etag → cache-headers → cors → logger/sentry → maintenance  │
        │                  → auth (session cookie)                    │
        └───────┬──────────────────────────────────────┬──────────────┘
                │                                      │
                ▼                                      ▼
        ┌───────────────┐                     ┌──────────────────────┐
        │  Controllers  │                     │    WebSocket hub     │
        │  (HTTP → DTO) │                     │  per-user + per-room │
        └───────┬───────┘                     └───────┬──────────────┘
                │                                     │
                ▼                                     │
        ┌───────────────┐     notify/push             │
        │   Services    │ ───────────────────────────▶│
        │ (authz, rules,│                             │
        │  orchestration)│                            │
        └───────┬───────┘                             │
                │                                     │
                ▼                                     ▼
        ┌───────────────┐                     ┌──────────────────────┐
        │ Repositories  │                     │    Media processor   │
        │ (all SQL,     │                     │   (image/video queue │
        │   WithTx)     │                     │   → ffmpeg / cwebp)  │
        └───────┬───────┘                     └──────────────────────┘
                │
                ▼
        ┌───────────────┐
        │    SQLite     │
        │  (WAL, FKs)   │
        └───────────────┘
```

### Request Lifecycle

A typical `POST /api/theories` request walks through a fixed middleware chain, lands in a controller, then flows down through service and repo:

```
 HTTP request
     │
     ▼
 ┌─────────────┐   shortcuts 304 responses before hitting handlers
 │    etag     │
 └──────┬──────┘
        ▼
 ┌─────────────┐   per-path Cache-Control (immutable assets, no-cache API)
 │ cache hdrs  │
 └──────┬──────┘
        ▼
 ┌─────────────┐   origin gated against live SettingBaseURL
 │    CORS     │
 └──────┬──────┘
        ▼
 ┌─────────────┐   request-scoped client_ip + Sentry transaction
 │  logger +   │
 │   sentry    │
 └──────┬──────┘
        ▼
 ┌─────────────┐   503 page unless viewer has bypass permission
 │ maintenance │
 └──────┬──────┘
        ▼
 ┌─────────────┐   session cookie → user, or 401
 │    auth     │
 └──────┬──────┘
        ▼
 ┌─────────────┐   authz permission check (edit_any_*, ban_users, …)
 │ controller  │
 └──────┬──────┘
        ▼
 ┌─────────────┐   content filter → business rules → DTO mapping
 │   service   │
 └──────┬──────┘
        ▼
 ┌─────────────┐   db.WithTx for multi-table writes, else direct query
 │ repository  │
 └──────┬──────┘
        ▼
 ┌─────────────┐   SQLite, WAL mode, FKs enforced
 │     DB      │
 └─────────────┘
```

### Data Layer

- **All SQL lives in `internal/repository/`**, one file per domain (theory.go, post.go, art.go, mystery.go, ship.go, fanfic.go, journal.go, chat.go, vanity_role.go, etc.).
- **Transactions** use the `db.WithTx(ctx, db, func(tx) error)` helper in `internal/db/tx.go`. Repo methods that touch multiple tables wrap themselves in `WithTx` and expose a single combined method (e.g. `CreateWithCharacters`, `UpdateWithTags`, `MarkSolved`). Services do not handle transactions directly.
- **Foreign keys** are enabled via `PRAGMA foreign_keys=ON`. Most deletes cascade through ON DELETE CASCADE. `galleries -> art.gallery_id` is `ON DELETE SET NULL`, so the gallery delete path explicitly removes child art inside a transaction.
- **WAL mode** is enabled (`PRAGMA journal_mode=WAL`) for concurrent reads.
- **Hot-reloadable settings** live in the `site_settings` table and are served through `internal/settings`. Listeners registered at startup react to changes (e.g. re-reading the log level) without a server restart.

```
  controller ──▶ service ──▶ repository ──▶  db.WithTx(ctx, func(tx) {
                                                    INSERT ...
                                                    INSERT ...
                                                    UPDATE ...
                                              })
                                ▲
                                │ one method per logical operation,
                                │ not one-method-per-table
```

### Auth and Sessions

- Server-side sessions stored in SQLite with httpOnly cookies.
- No JWTs. The session ID is the only thing in the cookie. The session row carries the user ID, expiry, and IP hash.
- Session renewal is handled by middleware, cleanup runs on a timer, and `SettingSessionDurationDays` controls lifetime.

```
   Browser                Server
  ──────────             ──────────
   login form ──────────▶ auth.Login
                          │
                          │ verify credentials
                          │ create session row
                          │
   Set-Cookie ◀───────────┘  session_id (httpOnly, Secure, SameSite=Lax)
        │
        │ every subsequent request
        ▼
    ┌───────────────┐
    │ auth middleware│── session_id → session row → user → ctx.Locals
    └───────────────┘
```

### Permission Model

- Every action is gated on a **permission**, not a raw role check. Permissions include `edit_any_theory`, `edit_any_post`, `delete_any_post`, `ban_users`, `view_reports`, `resolve_suggestion`, `manage_vanity_roles`, etc.
- Roles map to permission sets in `internal/authz/`. `super_admin` gets `PermAll`, `admin` gets most things, `moderator` gets moderation-adjacent permissions.
- Some features (the "game master" view in mysteries) check `role == super_admin` directly because the behaviour is intentionally scoped to that one role, not the permission grant.

```
  role          permissions granted
  ────────────  ─────────────────────────────────────────────────
  super_admin   PermAll
  admin         edit_any_*, delete_any_*, ban_users, view_reports,
                manage_settings, manage_invites, manage_vanity_roles, ...
  moderator     delete_any_comment, resolve_report, timeout_user, ...
  member        edit_own_*, delete_own_*, vote, comment, upload
```

### Content Filter

`internal/contentfilter` is a pluggable validation pipeline. Every text-bearing service (posts, comments, DMs, chat rooms, theories, fanfics, journals) runs its payload through the manager before writing.

```
   user text ──▶ ┌──────────────────────────────────┐
                 │  contentfilter.Manager           │
                 │                                  │
                 │  ┌─ RuleBannedGiphy ──┐          │  first failing rule
                 │  ├─ (future rule)   ──┤  ──────▶ │  stops the chain and
                 │  └─ (future rule)   ──┘          │  returns a validation
                 └──────────────────────────────────┘  error to the service
                              │
                              ▼
                         accept → service writes to repo
```

The banned-GIPHY rule reads the live banlist from `internal/giphy/banlist`. The admin banned-GIFs UI writes to that list and changes apply instantly without a restart.

### WebSocket Hub

`internal/ws` is a single in-process hub that multiplexes every live event on the site. Clients open one socket per tab, the hub keys them by user ID, and services push events through `SendToUser`, `Broadcast`, or `BroadcastToRoom`.

```
  clients (many tabs)
     │ websocket upgrade
     ▼
  ┌────────────────────────────────────────────────┐
  │                  ws.Hub                        │
  │                                                │
  │   ┌──────────────┐     ┌──────────────────┐    │
  │   │ by user ID   │     │ by room ID       │    │
  │   │  {u: [c,c]}  │     │  {r: [u,u,u]}    │    │
  │   └──────────────┘     └──────────────────┘    │
  │                                                │
  └───────▲──────────────────▲─────────────────▲───┘
          │ SendToUser       │ BroadcastToRoom │ Broadcast
          │                  │                 │
  ┌───────┴──────┐   ┌───────┴──────┐   ┌──────┴──────┐
  │ notification │   │ chat service │   │ like / view │
  │   service    │   │ (msg, react, │   │  counters   │
  │              │   │  pin, typing)│   │             │
  └──────────────┘   └──────────────┘   └─────────────┘
```

The frontend reconnects with exponential backoff and falls back to ETag-based polling for the notifications feed when the socket is unavailable.

### Notifications

A notification is both a DB row (so it shows in the notifications page) and a live event (so the bell counter updates without a reload). The notification service fans out through the hub and optionally through email.

```
   event (e.g. new response on your theory)
       │
       ▼
   notification.Service.Notify(ctx, userID, type, payload)
       │
       ├──▶ repository.Notification.Insert(...)  (persisted, paginated feed)
       ├──▶ hub.SendToUser(userID, msg)          (live bell + toast)
       └──▶ if user has email opt-in and SMTP configured:
                email.Service.Send(template, deep-link)
```

### Media Pipeline

Uploads are stored as-is first, then a background worker pool re-encodes them to WebP or MP4. The request thread never blocks on ffmpeg.

```
   controller receives multipart upload
         │
         ▼
   upload.Service.Save(file)   ──▶ original bytes land on disk (uploads/)
         │
         ▼
   media.Processor.Enqueue({type, path, callback})
         │
         ▼
   ┌─────────────────────────────────────────────┐
   │  buffered job channel (cap 256)             │
   └───────┬─────────────────────────────────────┘
           │ N worker goroutines (default 4)
           ▼
   ┌─────────────────────────────────────────────┐
   │ image worker → cwebp (q=80)                 │
   │ video worker → ffmpeg (h264 CRF 28)         │
   │                          + thumbnail frame  │
   └───────┬─────────────────────────────────────┘
           │
           ▼
   callback(outputPath) — updates DB row with the encoded asset URL
```

If the queue is full the job is dropped and logged rather than back-pressuring the request.

### OG and SEO

`internal/og` owns the SEO meta surface. On every HTML request, it matches the URL to a resolver (theory, post, profile, mystery, ship, art, gallery, fanfic, journal, chat room, corner feed, root) and injects per-page `<title>`, `<meta description>`, `og:*`, `twitter:*`, and `<link rel="canonical">` tags into the embedded Vite `index.html`. Routes that don't match fall back to the site-wide defaults.

```
   GET /theory/<id>
       │
       ▼
   og.Resolver.Resolve(ctx, path)
       │
       ├─ metaForPath() ──▶ theoryMeta(ctx, id) ──▶ repo.GetByID
       │                                              │
       │                                              ▼
       │                                         Meta{Title, Description,
       │                                              Image, URL}
       │
       └─ inject(meta)  ──▶ rewrites og:title, og:description, og:url,
                            og:image, twitter:*, <title>, canonical link
                            inside the embedded index.html
```

### Service Composition (server.go)

`server.go` wires everything explicitly. There is no DI container; the `services` struct in `initServices` is the dependency graph.

```
  config + env
     │
     ▼
  db.Open → db.Migrate (goose, embedded SQL)
     │
     ▼
  repository.New(db)  ──▶  one repo per domain
     │
     ▼
  settings.Service (DB-backed, hot reload)
     │
     ▼
  session, authz, block, notification, email,
  contentfilter, giphy, media.Processor, ws.Hub
     │
     ▼
  domain services: auth, profile, theory, post, art, ship,
                   mystery, fanfic, journal, chat, admin, report
     │
     ▼
  controllers + routes.Register(app, services)
     │
     ▼
  middleware.Setup(app, settings, session, authz)
     │
     ▼
  app.Listen(":4323")
```

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

| Variable           | Default                 | Description                                             |
|--------------------|-------------------------|---------------------------------------------------------|
| `DB_PATH`          | `truths.db`             | Path to SQLite database file                            |
| `UPLOAD_DIR`       | `uploads`               | Directory for uploaded files                            |
| `BASE_URL`         | `http://localhost:4323` | Public base URL, used for CORS and absolute links       |
| `LOG_LEVEL`        | `info`                  | Log level: trace, debug, info, warn, error, fatal       |
| `MAX_BODY_SIZE`    | `52428800`              | Fiber request body limit in bytes (default 50MB)        |
| `MAX_IMAGE_SIZE`   | `10485760`              | Max image upload size in bytes (default 10MB)           |
| `MAX_VIDEO_SIZE`   | `104857600`             | Max video upload size in bytes (default 100MB)          |
| `MAX_GENERAL_SIZE` | `52428800`              | Max other file upload size in bytes (default 50MB)      |
| `GIPHY_API_KEY`    | (empty)                 | GIPHY API key, required to enable the GIF picker        |
| `SENTRY_DSN`       | (empty)                 | GlitchTip / Sentry DSN for error shipping (optional)    |

Most runtime behaviour (registration mode, maintenance mode, turnstile keys, upload limits, rate limits, log level, email SMTP settings, default theme, Sentry DSN) is stored in the database via the `site_settings` table and editable from the admin panel at runtime with hot reload. The env file is only for things that must exist before the DB is reachable, or secrets that should not round-trip through the DB.

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
go test ./...       # run tests
./scripts/test.sh   # regenerate mocks, vet, staticcheck, test
./scripts/regen_mocks.sh  # regenerate mockery mocks only
```

Interfaces flagged in `.mockery.yml` have mock implementations auto-generated under each package (`service_mock.go`, etc.). Regenerate whenever you add or change an interface signature.

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

1. **OG tags** - `internal/og/og.go`: add path matching in `metaForPath()` and a meta method for detail pages. Canonical URL, og:title, og:description, og:image, and twitter:* tags are auto-injected from the returned `Meta`.
2. **Admin Content Rules** - `frontend/src/pages/admin/AdminContentRules.tsx`: add to `pages` array with a `rules_<page_name>` key, and register the matching `SettingRules...` in `internal/config/config.go`.
3. **Sidebar** - `frontend/src/components/layout/Sidebar/Sidebar.tsx`: add `<NavLink>` in the appropriate section.
4. **Profile settings default page** - `frontend/src/pages/profile/SettingsPage.tsx`: add `<option>` to the Home Page dropdown.
5. **Home page routes** - `frontend/src/App.tsx`: add to the `homePageRoutes` object and add a `<Route>` element.
6. **Sitemap** - `internal/controllers/sitemap_controller.go`: add the URL to `static()` or create a dynamic sitemap handler for collections.
7. **Content filter rules** - `internal/contentfilter`: if the new page accepts user text, make sure its service runs input through the content filter pipeline.
