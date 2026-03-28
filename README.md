# Umineko City of Books

A fan theory debate platform for Umineko no Naku Koro ni. Users declare theories as **blue truth**, attach quotes from the game as evidence, and others debate them on two sides: **"With love, it can be seen"** (support) and **"Without love, it cannot be seen"** (deny).

## Features

- **Theory Declarations** - Submit fan theories as blue truth with a title, body, and episode scope
- **Evidence Attachment** - Search any quote from the game (including narrator quotes) and attach as evidence
- **Debate System** - Respond with "With love, it can be seen" or "Without love, it cannot be seen", each with their own evidence
- **Threaded Replies** - Reply to responses with one level of threading, then flat with @mentions
- **Voting** - Upvote/downvote theories and responses
- **User Profiles** - Profile pages with avatar upload, bio, social links, favourite character, and stats
- **Quote Browser** - Browse game quotes filtered by episode, character, and truth type (red/blue/gold/purple)
- **Three Themes** - Featherine (gold/purple), Bernkastel (blue), Lambdadelta (pink)

## Tech Stack

- **Backend**: Go 1.26, Fiber v3, SQLite (`modernc.org/sqlite`), goose migrations
- **Frontend**: React 19, TypeScript 5.9, Vite 8, React Router
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
| `UPLOAD_DIR` | `uploads` | Directory for uploaded files (avatars) |
| `BASE_URL` | `http://localhost:4323` | Base URL for CORS |

### Development

```bash
# Backend
go run .

# Frontend (separate terminal)
cd frontend
npm install
npm run dev
```

The backend runs on `:4323`. The Vite dev server proxies `/api` and `/uploads` requests to it.

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
  middleware/        HTTP middleware (CORS, ETag, logging, auth)
  profile/          Profile service (get/update profile, avatar upload)
  repository/       All database repositories + models
  routes/           Route registration
  session/          Session management (token generation, validation)
  theory/           Theory service layer
  upload/           Centralised file upload service
  user/             User service layer
  utils/            Graceful shutdown
frontend/
  src/
    api/            API client with typed endpoints
    components/     React components (auth, theory, truth, layout, common)
    context/        Theme and auth contexts
    hooks/          Custom React hooks
    pages/          Route pages
    styles/         CSS variables and global styles
    types/          TypeScript types
```
