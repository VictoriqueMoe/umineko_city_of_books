# Umineko City of Books

<p align="center">
  <img src="https://waifuvault.moe/f/65d13bad-a630-4a61-baa2-13502b5ce115/clipboard-image-1776640341031.png" alt="Umineko City of Books" width="900">
</p>

<p align="center">
  <sub>
    Artwork by <a href="https://meta.auaurora.moe/user/alexwithaa21">alexwithaa21</a>
    (<a href="https://x.com/alexwithaa21">@alexwithaa21</a> on X)
  </sub>
</p>

A community platform for fans of Umineko no Naku Koro ni, Higurashi, Ciconia, and the wider When They Cry series. The original goal was a place to declare fan theories as **blue truth**, attach quotes from the game as evidence, and have them debated on two sides: **"With love, it can be seen"** and **"Without love, it cannot be seen"**. It has since grown into a full social platform: theory debates across all three series, a Twitter-style game board, mystery boards, fan art galleries, ship declarations, fanfiction, live reading journals, chat rooms and DMs, secret unlock hunts, live notifications, and themed role-based moderation.

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
  - [Secrets and Unlock Hunts](#secrets-and-unlock-hunts)
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

- Theory declarations with title, body, and episode / arc / chapter scope depending on series
- Evidence attachment by searching any quote (including narrator lines) across Umineko, Higurashi, and Ciconia via the Umineko Quote Finder API, with per-series chapter/episode/arc filters and a main/additional character split for Higurashi and Ciconia
- Two-sided debate with **"With love, it can be seen"** (support) and **"Without love, it cannot be seen"** (deny), each with its own evidence
- **Credibility score** per theory (0 to 100), weighted by the truth type of evidence attached to responses (gold > red > purple > blue > none)
- Threaded replies on responses with flat rendering and @username attribution
- Upvotes and downvotes on both theories and responses, separate from the credibility score
- Per-series feeds at `/theories/umineko`, `/theories/higurashi`, and `/theories/ciconia`, each with its own sidebar entry

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

- Fanfic entries with title, summary, language, series tag (Umineko / Higurashi / Ciconia / OC), cover image, and character tagging across all three series
- **Chapter-based structure**: add, reorder, edit, and delete chapters individually
- **Rich text editor** (TipTap) for chapter bodies: bold, italic, strike, headings, blockquotes, bullet/ordered lists, horizontal rule, text alignment (left/centre/right), colour swatches, and links
- **Server-side HTML sanitisation** (bluemonday UGCPolicy) on every write, plus client-side DOMPurify before render, so the full Tiptap toolbar survives but `<script>`, event handlers, `javascript:` URLs, iframes, and SVG payloads are stripped
- Word count, character tagging across Umineko / Higurashi / Ciconia / OC
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

- **Direct Messages** with unread counts, last-read cursors, and per-user enable/disable toggle in profile settings
- **Chat Rooms**: public or private group rooms, with an optional **Roleplay** flag that switches the room into a different visual and posting style
- **Emoji reactions** on messages with live count and "you reacted" state, shown across desktop and mobile
- **Pinning**: moderators and room owners can pin messages; a dedicated pinned messages panel surfaces them
- **Member management**: per-room roles, timeouts, kick, and nickname profiles scoped to the room (ghost members supported)
- **Per-room bans** that stick. Banned users cannot rejoin, send, read, list members, or see the room in their list. Available to the room host, site moderators, and admins. Banned targets receive a live WS kick event plus an optional reason.
- **Banned-words filter** with two scopes and three match modes:
  - **Global** rules (admin `/admin/banned-words`) apply to every chat room
  - **Local** rules (per-room moderation dialog, open to host + mods + admins) apply only to that room and see global rules read-only
  - Match mode `Substring` / `Whole word` / `Regex` plus a `Case sensitive` toggle; regex syntax validated on save
  - Action `Delete message` rejects the send with an inline error; action `Kick` also evicts the sender from the room (they can rejoin; a ban is a separate, intentional moderator action)
  - Room hosts, site moderators, admins, and superadmins are immune. Automated hits log to the audit trail with a NULL actor ("System")
  - Rules are editable: pattern, mode, case, and action can all be changed after creation
- **Configurable limits**: max room members and max rooms per day are site-settings driven
- **Replies and edits** on individual messages, with a floating action bar above the bubble on hover
- **GIF picker**, emoji picker, media uploads, and full Discord-style text formatting (backticks, quotes, spoilers, syntax highlighting) everywhere text is typed
- WebSocket-driven real-time delivery, pin/unpin events, reaction updates, and typing presence
- Mobile-first composer: full-width text box with Media / GIF / Send stacked below, bubbles spanning edge to edge

### Secrets and Unlock Hunts

Hidden puzzles scattered across the UI, declared in code (`internal/secrets/`), surfaced on a public hub page at `/secrets`.

- Each hunt is a **parent secret** (e.g. `witchHunter`) plus a set of piece sub-secrets. Pieces are collected by finding tiny sparkles (`PieceTrigger`) tucked in ordinary UI spots (a tagline, a button, a rule, a subtitle, a chip, a sentence), deliberately varied so pattern-spotting doesn't shortcut the hunt
- **Listed metadata** (title, description, riddle, icon, reward vanity role) is kept in the registry; pieces stay hidden implementation detail
- **Server-side guard** refuses submission of the final phrase unless every piece is already unlocked for the caller, so even a leaked answer can't bypass the hunt
- **Secrets hub page** (`/secrets`) lists every declared hunt with your viewer progress, the first solver, comment count, and a **solvers leaderboard** ranking every user with at least one solved hunt
- **Detail page** (`/secrets/:id`) shows the riddle, a live **progress leaderboard** that reorders in real time via WebSocket as people collect pieces, a pinned first-solver row, and a full-featured discussion thread that stays open forever
- **WS presence per secret**: viewers join a `secret:<id>` room on mount and leave on unmount. Progress and solve events only fan out to current viewers, not the whole site
- **Global events on solve**: when a hunt with a reward role is solved, a `vanity_roles_changed` broadcast refreshes site-info on every connected client so the new role pill appears without a reload
- **Trophy case** on every profile: solved hunts show as live-updating trophies in an Achievements section, owner-clickable to re-open the hunt panel; the in-progress rose icon lives next to the owner's display name until they solve
- The v5 hunt is **The Witch's Epitaph**. Maria has hidden twelve letters across the site; finding all twelve unlocks the Maria theme and the sparkling Witch Hunter role

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

A standalone interface for browsing the full quote corpus across all three series, sourced from the Umineko Quote Finder API. Switch between Umineko / Higurashi / Ciconia tabs, filter by chapter/episode/arc, and filter by character with a main/additional split where the quote service exposes one. Ciconia and Higurashi quotes ship with Japanese text inline, and the language picker now honours it across all series.

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
- **Audit log** for admin actions. Automated moderation events (word-filter hits) log with a NULL actor and render as "System" in the admin audit page, distinguishing them from human-initiated actions
- **Reports**: users can report theories, responses, posts, comments, art, ships, users, fanfics, journals, and chat messages. Admins resolve from the admin panel with optional comment back to the reporter
- **Banned GIFs**: admins block specific GIPHY IDs from being embedded anywhere on the site; the content filter rejects matches before they render
- **Banned Words** (`/admin/banned-words`): global word-filter rules for chat rooms with regex / whole-word / substring match modes, editable in place
- **Content Filter Pipeline** (`internal/contentfilter`): pluggable rule-based validation that runs on all user-generated text before it lands in the DB
- **Content rules** per section (theories, general game board, each corner, mysteries, ships, gallery, fanfiction, journals, suggestions, chat rooms, announcements), admin-editable and displayed at the top of each page
- **Per-action rate limits**: max theories, responses, posts, art, journals, and chat rooms per day, all settable from the admin panel
- **Cloudflare Turnstile** on login and registration, toggle-able from admin settings

### Platform Features

- **Fourteen themes** grouped by series in the theme picker:
  - **Umineko**: Featherine (gold/purple, default), Beatrice (warm gold/brown), Bernkastel (blue), Lambdadelta (pink), Erika Furudo (cyan/pink), Battler, Virgilia (light mode)
  - **Higurashi**: Rika, Mion, Satoko
  - **Ciconia**: Miyao (deep navy with gold and sky-blue), Lingji (crimson and gold), Stanis&#322;aw (silver on near-black)
  - **Unlockable**: Maria Ushiromiya (rosy pink), granted by solving the Witch's Epitaph hunt
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
- PostgreSQL via `jackc/pgx/v5` (through the `pgx/v5/stdlib` adapter so `database/sql` and `otelsql` instrumentation still apply)
- Goose for migrations
- testcontainers-go for repo-layer tests against a real Postgres
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
        │  PostgreSQL   │
        │  (UUID, JSONB,│
        │   CITEXT, FKs)│
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
 ┌─────────────┐   PostgreSQL, FKs enforced, native UUID/JSONB/CITEXT
 │     DB      │
 └─────────────┘
```

### Data Layer

- **All SQL lives in `internal/repository/`**, one file per domain (theory.go, post.go, art.go, mystery.go, ship.go, fanfic.go, journal.go, chat.go, vanity_role.go, etc.).
- **Transactions** use the `db.WithTx(ctx, db, func(tx) error)` helper in `internal/db/tx.go`. Repo methods that touch multiple tables wrap themselves in `WithTx` and expose a single combined method (e.g. `CreateWithCharacters`, `UpdateWithTags`, `MarkSolved`). Services do not handle transactions directly.
- **Native Postgres types** throughout the schema: `UUID` for primary and foreign keys, `BIGINT GENERATED BY DEFAULT AS IDENTITY` for auto-increment columns, `BOOLEAN` for flags (no more `INTEGER 0/1`), `TIMESTAMPTZ` for time columns, `JSONB` for `state_json` / `action_json`, and `CITEXT` (case-insensitive text) for unique-by-name lookups like fanfic series, languages, and OC characters.
- **Foreign keys** are enforced by Postgres. Most deletes cascade through `ON DELETE CASCADE`; `galleries -> art.gallery_id` is `ON DELETE SET NULL`, so the gallery delete path explicitly removes child art inside a transaction.
- **Hot-reloadable settings** live in the `site_settings` table and are served through `internal/settings`. Listeners registered at startup react to changes (e.g. re-reading the log level) without a server restart.
- **Repo tests** boot a real `postgres:latest` container per test binary via testcontainers-go, then create a per-test database from a pre-migrated template. Public test API: `repotest.NewRepos(t)`, `repotest.CreateUser(t, repos, opts...)`, `repotest.CreateSession(t, repos, userID)`. Tests need Docker on the host.

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

- Server-side sessions stored in Postgres with httpOnly cookies.
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
   callback(outputPath): updates DB row with the encoded asset URL
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
- Docker (for the Postgres container, plus repo-layer tests via testcontainers-go)
- FFmpeg (for video transcoding and thumbnails)
- libwebp-tools (`cwebp`) for WebP conversion
- `psql` CLI (optional, handy for poking at the DB)

### Environment

Two env files live next to each other:

- **`postgres.env`** — Postgres bootstrap credentials. Read by the postgres container at init and by the app at connect time. Copy from `postgres.env.example`:
  ```bash
  cp postgres.env.example postgres.env
  ```
  | Variable            | Description                                                |
  |---------------------|------------------------------------------------------------|
  | `POSTGRES_USER`     | DB role for the app (`umineko` by convention)              |
  | `POSTGRES_PASSWORD` | DB password                                                |
  | `POSTGRES_DB`       | Database name (`umineko_city_of_books` by convention)      |

- **`.env`** — everything else the app reads. Copy from `.env.example`:
  ```bash
  cp .env.example .env
  ```
  | Variable            | Default                 | Description                                                                       |
  |---------------------|-------------------------|-----------------------------------------------------------------------------------|
  | `BASE_URL`          | `http://localhost:4323` | Public base URL, used for CORS and absolute links                                 |
  | `UPLOAD_DIR`        | `uploads`               | Directory for uploaded files (relative to working dir)                            |
  | `LOG_LEVEL`         | `info`                  | Initial log level (overridable from admin panel at runtime)                       |
  | `POSTGRES_HOST`     | `localhost`             | Postgres host. `postgres` (the compose service name) under docker-compose         |
  | `POSTGRES_PORT`     | `5432`                  | Postgres port. The internal container port — not the host-mapped 5007             |
  | `POSTGRES_SSL_MODE` | `disable`               | Postgres SSL mode (`disable`, `require`, `verify-ca`, `verify-full`)              |
  | `DATABASE_URL`      | (empty)                 | Full connection string. If set, overrides the discrete `POSTGRES_*` vars          |
  | `GIPHY_API_KEY`     | (empty)                 | GIPHY API key, required to enable the GIF picker                                  |
  | `SENTRY_DSN`        | (empty)                 | GlitchTip / Sentry DSN for error shipping (optional)                              |

Most runtime behaviour (registration mode, maintenance mode, turnstile keys, upload limits, rate limits, log level, email SMTP settings, default theme, Sentry DSN) is stored in the database via the `site_settings` table and editable from the admin panel at runtime with hot reload. The env file is only for things that must exist before the DB is reachable, or secrets that should not round-trip through the DB.

### Running Locally

The app needs Postgres reachable before it'll boot. Two paths:

**Option A: Run only Postgres in Docker, the app on the host**

```bash
# start just the postgres service from compose
docker compose up -d postgres

# backend (from repo root) — connects to Postgres on host port 5007
POSTGRES_HOST=localhost POSTGRES_PORT=5007 go run .

# frontend (separate terminal)
cd frontend
npm install
npm run dev
```

**Option B: Run the full stack in Docker**

```bash
docker compose up -d --build
```

This builds and runs the app container alongside Postgres on the internal docker network; the app reads `POSTGRES_HOST=postgres` automatically from compose. Visit `http://localhost:2312`.

The backend serves on `:4323` (mapped to `:2312` from the host under docker-compose). The Vite dev server proxies `/api`, `/uploads`, `/sitemap`, and WebSocket upgrades to the Go server.

**The first user to register is automatically assigned the super admin role**, so start there to unlock the admin panel.

## Database and Migrations

All migrations live in `internal/db/migrations/` and are embedded into the binary via `go:embed`. They run automatically on startup via goose against the configured Postgres database.

The schema is currently a single consolidated initial migration squashed during the SQLite-to-Postgres cutover. Any further schema changes are added as fresh migrations on top.

**Always create migrations with the goose CLI**, never by hand, so the timestamp format stays consistent:

```bash
goose -dir internal/db/migrations create <name> sql
```

Then edit the generated file to fill in the `-- +goose Up` and `-- +goose Down` sections. On next `go run .` the migration applies automatically.

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

Two persistent stores need to survive container rebuilds:

- **Postgres data** — backed by the named docker volume `postgres_data` mounted at `/var/lib/postgresql` inside the postgres container. Survives `docker compose up -d` and image upgrades.
- **Uploaded media** — the `umineko-city-of-books` service bind-mounts `./data:/app/data` so `data/uploads/` lives on the host. Set `UPLOAD_DIR=data/uploads` in your `.env` so the app reads from this mount.

For backups: a daily `pg_dump | gzip` cron is the recommended path for the database, plus a periodic tarball of `./data/uploads/` for media. Restore via `gunzip -c <dump>.sql.gz | docker compose exec -T postgres psql -U umineko -d umineko_city_of_books`.

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
