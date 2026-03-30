# Umineko City of Books

A social networking platform for fans of Umineko no Naku Koro ni. The core feature is fan theory debates: users declare theories as **blue truth**, attach quotes from the game as evidence, and others respond on two sides: **"With love, it can be seen"** (support) and **"Without love, it cannot be seen"** (deny). Beyond theories, the platform offers user profiles with pronouns and social links, real-time notifications, role-based moderation, and is growing into a full community hub with plans for discussion boards, fanfiction, fan art, live chat, and more.

## Features

- **Theory Declarations** - Submit fan theories as blue truth with a title, body, and episode scope
- **Evidence Attachment** - Search any quote from the game (including narrator quotes) and attach as evidence
- **Debate System** - Respond with "With love, it can be seen" or "Without love, it cannot be seen", each with their own evidence
- **Credibility Score** - 0-100 score per theory based on debate quality, weighted by truth type of evidence (gold > red > purple > blue > none)
- **Threaded Replies** - Reply to responses with one level of threading, then flat with @mentions
- **Real-Time Notifications** - WebSocket-powered notifications for responses, replies, and upvotes
- **Voting** - Upvote/downvote theories and responses (popularity, separate from credibility)
- **User Profiles** - Avatar, draggable banner positioning, bio, pronouns, gender, social links, favourite character, activity feed, and online status
- **Role-Based Authorisation** - Permission-based system with admin and moderator roles. First registered user is automatically admin
- **Admin Panel** - Dashboard with site stats, user management (roles, bans), DB-backed site settings with live reload, invite system, audit log
- **Site Settings** - All configuration stored in DB, editable from admin panel, hot-reloadable (body limits, log level, registration type, maintenance mode, etc.)
- **Invite System** - Open, invite-only, or closed registration modes. Admins generate one-time invite codes
- **Maintenance Mode** - Toggle from admin panel, serves maintenance page to non-admins, admins can still access the full site
- **Quote Browser** - Browse game quotes filtered by episode, character, and truth type (red/blue/gold/purple)
- **Three Themes** - Featherine (gold/purple), Bernkastel (blue), Lambdadelta (pink)
- **Structured Logging** - zerolog with configurable log levels, settings change listener pattern

## Tech Stack

- **Backend**: Go 1.26, Fiber v3, SQLite (`modernc.org/sqlite`), goose migrations, WebSockets, zerolog
- **Frontend**: React 19, TypeScript 5.9, Vite 8, React Router, CSS Modules
- **Quotes**: [Umineko Quote Finder API](https://quotes.auaurora.moe/swagger/index.html)
- **Auth**: Server-side sessions in SQLite with httpOnly cookies
- **IDs**: UUIDv4 for users, theories, and responses (`github.com/google/uuid`)
- **Deployment**: Docker, Caddy reverse proxy

## Quick Start

### Prerequisites

- Go 1.26+
- Node.js (LTS)

### Environment

Copy `.env.example` to `.env` and configure:

```bash
cp .env.example .env
```

| Variable           | Default                 | Description                                        |
|--------------------|-------------------------|----------------------------------------------------|
| `DB_PATH`          | `truths.db`             | Path to SQLite database file                       |
| `UPLOAD_DIR`       | `uploads`               | Directory for uploaded files                       |
| `BASE_URL`         | `http://localhost:4323` | Base URL for CORS                                  |
| `LOG_LEVEL`        | `info`                  | Log level (trace, debug, info, warn, error, fatal) |
| `MAX_BODY_SIZE`    | `52428800` (50MB)       | Fiber request body limit (bytes)                   |
| `MAX_IMAGE_SIZE`   | `10485760` (10MB)       | Max size for image uploads (bytes)                 |
| `MAX_VIDEO_SIZE`   | `104857600` (100MB)     | Max size for video uploads (bytes)                 |
| `MAX_GENERAL_SIZE` | `52428800` (50MB)       | Max size for other file uploads (bytes)            |

### Development

```bash
# Backend
go run .

# Frontend (separate terminal)
cd frontend
npm install
npm run dev
```

The backend runs on `:4323`. The Vite dev server proxies `/api`, `/uploads`, and WebSocket connections to it.

The first user to register is automatically assigned the admin role.

### Production (Docker)

```bash
docker compose up -d
```
