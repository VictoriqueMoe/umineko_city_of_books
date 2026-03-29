# Umineko City of Books

A fan theory debate platform for Umineko no Naku Koro ni. Users declare theories as **blue truth**, attach quotes from the game as evidence, and others debate them on two sides: **"With love, it can be seen"** (support) and **"Without love, it cannot be seen"** (deny).

## Features

- **Theory Declarations** - Submit fan theories as blue truth with a title, body, and episode scope
- **Evidence Attachment** - Search any quote from the game (including narrator quotes) and attach as evidence
- **Debate System** - Respond with "With love, it can be seen" or "Without love, it cannot be seen", each with their own evidence
- **Threaded Replies** - Reply to responses with one level of threading, then flat with @mentions
- **Real-Time Notifications** - WebSocket-powered notifications for responses, replies, and upvotes
- **Voting** - Upvote/downvote theories and responses
- **User Profiles** - Profile pages with avatar, banner, bio, gender, social links, favourite character, activity feed, and online status
- **Quote Browser** - Browse game quotes filtered by episode, character, and truth type (red/blue/gold/purple)
- **Three Themes** - Featherine (gold/purple), Bernkastel (blue), Lambdadelta (pink)

## Tech Stack

- **Backend**: Go 1.26, Fiber v3, SQLite (`modernc.org/sqlite`), goose migrations, WebSockets
- **Frontend**: React 19, TypeScript 5.9, Vite 8, React Router, CSS Modules
- **Quotes**: [Umineko Quote Finder API](https://quotes.auaurora.moe/swagger/index.html)
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

| Variable | Default | Description |
|----------|---------|-------------|
| `DB_PATH` | `truths.db` | Path to SQLite database file |
| `UPLOAD_DIR` | `uploads` | Directory for uploaded files |
| `BASE_URL` | `http://localhost:4323` | Base URL for CORS |
| `LOG_LEVEL` | `info` | Log level (trace, debug, info, warn, error, fatal) |
| `MAX_BODY_SIZE` | `52428800` (50MB) | Fiber request body limit (bytes) |
| `MAX_IMAGE_SIZE` | `10485760` (10MB) | Max size for image uploads (bytes) |
| `MAX_VIDEO_SIZE` | `104857600` (100MB) | Max size for video uploads (bytes) |
| `MAX_GENERAL_SIZE` | `52428800` (50MB) | Max size for other file uploads (bytes) |

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

### Production (Docker)

```bash
docker compose up -d
```

## Project Structure

```
internal/
  auth/             Auth service (register, login, logout)
  config/           Environment configuration (godotenv)
  controllers/      HTTP handlers (Fiber v3, FSetupRoute pattern)
  db/               SQLite connection, goose migrations
  dto/              Data transfer objects
  middleware/       HTTP middleware (CORS, ETag, logging, auth)
  notification/     Notification service (real-time + persisted)
  profile/          Profile service (get/update profile, avatar upload)
  repository/       All database repositories + models
  routes/           Route registration
  session/          Session management (token generation, validation)
  theory/           Theory service layer
  upload/           Centralised file upload service
  user/             User service layer
  utils/            Graceful shutdown
  ws/               WebSocket hub (generic, reusable for live chat)
frontend/
  src/
    api/            API client with typed endpoints
    components/     React components (each with own folder + CSS module)
      Button/       Themed button with variants (primary/secondary/danger/ghost)
      Input/        Themed text input
      TextArea/     Themed textarea
      Select/       Themed select dropdown
      Modal/        Reusable modal overlay
      Pagination/   Page navigation
      ProfileLink/  Avatar + name link to profile
      ToggleSwitch/  Toggle switch input
      layout/       Header, Sidebar, ThemeSelector, NotificationBell, Butterflies
      auth/         LoginButton, UserMenu
      theory/       TheoryCard, TheoryForm, ResponseCard, ResponseEditor, EvidenceList, VoteButton
      truth/        TruthCard, TruthChip, TruthPicker
    context/        Theme, auth, and notification contexts
    hooks/          Custom React hooks
    pages/          Route pages grouped by feature (theories/, auth/, profile/, quotes/)
    styles/         CSS variables and minimal global styles
    types/          TypeScript types
```
