# Architecture

## 1. Scope

This document describes how the code in this repository is arranged and why it is arranged that way. On the Go side that is the controller, service, repository and DAO layering: controllers parse HTTP and gate on permissions, services hold the business rules and orchestrate, repositories own the contract and the cache seam, and DAOs own every line of SQL. On the React side it is the four frontend layers: render, orchestration (with data hooks as a named sub-layer), pure domain, and adapters. Each of those boundaries exists to enforce a single rule, and this document states that rule next to the layer it belongs to, because a boundary nobody can name is a boundary that erodes.

It is not a feature list: see [FEATURES.md](FEATURES.md) for what the site does. It is not an operations guide: see [DEPLOYMENT.md](DEPLOYMENT.md) for running, deploying and configuring it. It is not a contributor checklist: see [ADDING_A_PAGE.md](ADDING_A_PAGE.md) for the steps a new page needs. Those documents describe behaviour and procedure; this one describes the shape they live in.

The working drafts and diagrams behind these decisions are on a [Miro board](https://miro.com/app/board/uXjVGrJQlmo=/?share_link_id=82208113990). That board is where the designs were sketched before they were built, so it shows the reasoning and the discarded alternatives rather than only the result. It is a live working surface and can be ahead of, or behind, what this document describes; where the two disagree, the code and this document win.

## 2. System shape

The server is a single Go binary that embeds the compiled Vite bundle and serves both the SPA and the JSON API from one process. Every layer has a single responsibility: controllers parse HTTP, services orchestrate business logic, repositories own SQL, the hub owns live events, and the media processor owns encoding off the hot path.

### 2.1 High-level component map

```
        ┌──────────────────────────────┐  ┌──────────────────────────────┐
        │      Browser (React 19)      │  │   Capacitor app (same SPA)   │
        │  session cookie + WebSocket  │  │   bearer token + WebSocket   │
        └───────────────┬──────────────┘  └───────────────┬──────────────┘
                        │ HTTP / WS                       │ HTTP / WS
                        └────────────────┬────────────────┘
                                         ▼
        ┌─────────────────────────────────────────────────────────────────┐
        │                          Fiber v3 app                           │
        │  recover → tracing → host allow-list → security headers → etag  │
        │  → cache headers → cors → access log → maintenance → metrics    │
        │  → last-seen IP        (auth and authz attach per route)        │
        └────────┬────────────────────────────────────────┬───────────────┘
                 │                                        │
                 ▼                                        ▼
        ┌────────────────┐                       ┌──────────────────────┐
        │  Controllers   │                       │    WebSocket hub     │
        │  (HTTP → DTO,  │                       │  per user / room /   │
        │   authz gate)  │                       │  topic, in-process   │
        └────────┬───────┘                       └───────┬──────────────┘
                 │                                       │
                 ▼         notify / push                  │
        ┌────────────────┐ ─────────────────────────────▶ │
        │    Services    │                                │
        │ (rules, filter,│                                ▼
        │ orchestration) │                       ┌──────────────────────┐
        └────────┬───────┘                       │    Media processor   │
                 │                               │  (image/video queue  │
                 ▼                               │   → ffmpeg / cwebp)  │
        ┌────────────────┐   ┌────────────────┐  └──────────────────────┘
        │  Repositories  │──▶│ internal/cache │
        │ (interfaces +  │◀──│ Valkey, opt-in │
        │  cache seam)   │   └────────────────┘
        └────────┬───────┘
                 ▼
        ┌────────────────┐
        │      DAOs      │
        │   (all SQL,    │
        │   db.WithTx)   │
        └────────┬───────┘
                 ▼
        ┌────────────────┐
        │   PostgreSQL   │
        │ (UUID, JSONB,  │
        │  CITEXT, FKs)  │
        └────────────────┘
```

Voice, watch parties, and live streaming run on a separate media plane: audio and video travel from the browser to the LiveKit SFU (or to the Hyperbeam VM) directly, and the Go process only mints signed tokens and tracks presence, so none of that traffic passes through the layers above.

## 3. Backend

### 3.1 The layers at a glance

Four layers, one direction. A request enters at the top, and each layer may hand work down to exactly the one beneath it.

```
   HTTP request
        │
        ▼
 ┌────────────────────────────────────────────────────────────────────────┐
 │ controller    internal/controllers                                     │
 │               parses query and body, reads userID from ctx.Locals,     │
 │               calls one service method, returns JSON. Owns no rule.    │
 └───────────────────────────────────┬────────────────────────────────────┘
                                     ▼
 ┌────────────────────────────────────────────────────────────────────────┐
 │ service       internal/<domain>                                        │
 │               the rules: content filter, per-day caps, ownership and   │
 │               permission decisions, notification fan-out, DTO mapping. │
 └───────────────────────────────────┬────────────────────────────────────┘
                                     ▼
 ┌────────────────────────────────────────────────────────────────────────┐
 │ repository    internal/repository                                      │
 │               the contract the service depends on, the composites that │
 │               span several DAOs in one transaction, the cache seam.    │
 └───────────────────────────────────┬────────────────────────────────────┘
                                     ▼
 ┌────────────────────────────────────────────────────────────────────────┐
 │ dao           internal/dao                                             │
 │               every SQL statement in the codebase, one file per        │
 │               domain, one logical statement per method.                │
 └───────────────────────────────────┬────────────────────────────────────┘
                                     ▼
 ┌────────────────────────────────────────────────────────────────────────┐
 │ PostgreSQL    native UUID, JSONB, CITEXT, TIMESTAMPTZ, FKs enforced    │
 └────────────────────────────────────────────────────────────────────────┘
```

**A controller never reaches a repository.** Every field on the controller surface (`internal/controllers/service.go:51-98`) is a service, the session manager, a hub, the media processor, an OG helper or the embedded filesystem. Four non-test controller files do import `internal/repository`, and between them they reference five sentinel errors and one enum type (`repository.ErrArtNotOwned`, `repository.ErrRefutationRejected`, the three `ErrBasePrompt*` values, and `repository.AuditAction`); none holds a repository and none calls one.

**A service never writes SQL.** Outside `internal/dao`, the strings `SELECT `, `INSERT INTO`, `UPDATE ... SET` and `DELETE FROM` appear in exactly three non-test files, and all three are deliberate: `internal/repository/search.go` (the declarative search registry described in section 3.5), `internal/db/seed.go` (boot-time content seeding) and `internal/db/dbtest/dbtest.go` (test infrastructure). No service package contains a query.

**A DAO never learns a `fiber.Ctx` exists.** Neither `internal/dao` nor `internal/repository` imports `gofiber` anywhere, which is the one-line mechanical check that the bottom two layers are transport-agnostic and that a DAO could be reused unchanged behind a CLI or a queue consumer.

**The wiring runs the other way.** `internal/store/new.go` is 61 lines and is the only place the object graph is assembled: `store.New(db, cache)` builds every DAO, wraps each one in its repository, and passes the cache manager to the ten repositories that take it. `initServices` then receives the assembled `*repository.Repositories` and hands those interfaces down to the service constructors. Construction runs bottom-up, calls run top-down, and neither direction goes through a DI container.

### 3.2 Controllers

A controller owns the HTTP shape and nothing else. `listTheories` (`internal/controllers/theory_controller.go:73-99`) is the whole job in twenty-seven lines: it reads seven query parameters with their defaults, resolves the viewer with `utils.UserID(ctx)`, parses one optional UUID and returns 400 if it is malformed, folds the rest into a `params.ListParams`, calls `TheoryService.ListTheories`, and returns `ctx.JSON(result)`. There is no rule in it about what a theory is. Which sort orders are legal, what happens when `series` is empty and how limit and offset are bounded all live in `params.NewListParams` (`internal/theory/params/list_params.go:21-46`), one layer down, because those are domain facts rather than HTTP facts.

**Routes are never registered inline.** This is the backend rule a contributor gets wrong first, because nothing about it fails a build. Each controller declares a `getAllXRoutes() []FSetupRoute` returning one `setupX(r fiber.Router)` per endpoint, where `FSetupRoute` is just `func(fiber.Router)` (`internal/controllers/f_setup_route.go:6`). `getAllTheoryRoutes` (`theory_controller.go:18-31`) lists its ten setup methods; each one is two lines, for example `setupListTheoriesRoute` at `:33-35`. `internal/controllers/service.go` appends every such slice to `GetAPIRoutes()` (thirty groups, mounted under `/api/v1`) or `GetPageRoutes()` (seven groups, mounted at the app root), and `internal/routes/public_routes.go` walks both and calls each entry. The tree currently holds 37 `getAll*` functions and 394 `setup*` functions, and `public_routes.go` has needed no change for any of them.

**Auth and authorisation attach per route, inside the setup method.** `r.Get("/theories", s.optionalAuth(), s.listTheories)` and `r.Post("/theories", s.requireAuth(), s.createTheory)` sit one line apart, so the access rules for an endpoint are readable exactly where the endpoint is declared rather than in a middleware table elsewhere. The helpers are thin wrappers over the middleware package (`internal/controllers/route_auth.go`): `requireAuth`, `optionalAuth` and `requireEstablished`. Permission gating uses `s.requirePerm(authz.PermViewUsers)` (`internal/controllers/admin_controller.go:85-87`), which wraps `middleware.RequirePermission`, and two mystery routes call `middleware.RequirePermission` directly.

### 3.3 Services

The service holds the rules. Everything the controller declined to decide (whether the text passes the content filter, whether the author has hit the per-day cap, whether the viewer is blocked, who gets notified, what the response DTO looks like) is decided here.

**Dependencies arrive as interfaces through a constructor.** `theory.NewService` (`internal/theory/service.go:54-80`) takes eleven collaborators and stores them in an unexported `service` struct (`:39-51`). Eight are interfaces: `repository.TheoryRepository`, `repository.UserRepository`, `repository.FollowRepository`, `repository.AuditLogRepository`, `authz.Service`, `block.Service`, `notification.Service` and `settings.Service`. Three are concrete because they hold no state worth faking: `*credibility.Service`, `*quotefinder.Client` and `*contentfilter.Manager`. `NewService` returns the `Service` interface, never the struct, so a caller can only reach the ten methods the interface declares.

**That constructor is what makes the package unit-testable with no database.** `newTestService` (`internal/theory/service_test.go:43-54`) builds all eight interface dependencies as mockery mocks (`repository.NewMockTheoryRepository(t)`, `authz.NewMockService(t)`, and so on), constructs the three concrete collaborators for real, and hands the lot to `NewService`. The result is 1302 lines of behaviour tests that never open a connection, never start a container and run in milliseconds. Mocks are generated from `.mockery.yml` rather than hand-rolled, so a mock that has drifted from its interface is a compile error rather than a test that quietly passes.

**Services do not open transactions and do not write SQL.** Neither `db.WithTx` nor `BeginTx` appears in any non-test file outside `internal/repository`, `internal/dao` and `internal/db`. A service that needs two writes to be atomic asks the repository for a composite method, described in section 3.6, rather than reaching for a transaction handle it has no way to obtain.

### 3.4 Repositories

`internal/repository` owns the contract, not the queries. A domain file declares the interface the service depends on, the row models it returns (`internal/repository/model/`), the sentinel errors the service matches on, and a thin passthrough struct that wraps the DAO and adds whatever cannot be a single statement.

**Where the repository orchestrates, the interface splits in two.** `internal/repository/art.go` is the worked example. `ArtDAO` (`:20-77`) lists the fifty database methods: `CreateArt`, `InsertTags`, `GetComments`, `ListArtInGallery` and the rest, each one a statement. `ArtRepository` (`:79-88`) embeds `ArtDAO` and adds only the six composites that need more than one statement to be atomic: `CreateWithTags`, `UpdateWithTags`, `DeleteWithImage`, `UpdateCommentWithDetails`, `DeleteCommentWithAudit` and `DeleteGallery`. The service depends on `ArtRepository` and sees both halves as one flat surface.

**The split is what makes the boundary enforceable by the compiler rather than by review.** `dao.NewArt(db)` is declared as returning `repository.ArtDAO` (`internal/dao/new.go:60`), not `repository.ArtRepository`, so a DAO physically cannot be handed to a service, and an orchestration method written on the DAO struct by mistake satisfies nothing and is dead on arrival. Twenty-two of the forty-four constructors in `internal/dao/new.go` return an `XDAO` this way; the other twenty-two belong to domains with no cross-DAO composite, which declare a single `XRepository` implemented twice, once by the DAO and once by the passthrough (`internal/repository/permission.go:14-20` is the smallest of these).

**`internal/store/new.go` is the single wiring point.** Sixty-one lines, one line per domain, always the same shape: `repository.NewArtRepo(db, dao.NewArt(db), repos.Post, repos.AuditLog)` (`:29`). A repository takes `*sql.DB` only when it owns a transaction, and takes `*cache.Manager` only when it caches. Nothing else in the tree calls `dao.NewX`, test files included.

### 3.5 DAOs

Every SQL statement in the codebase lives under `internal/dao`: 55 non-test files, one per domain (`theory.go`, `post.go`, `art.go`, `mystery.go`, `ship.go`, `fanfic.go`, `journal.go`, `chat.go`, `permission.go` and the rest), plus a handful of shared files described below. The structs are unexported and are only reachable through the constructors in `internal/dao/new.go`.

A method is one logical statement, and the table it writes is its own. `artDAO.CreateArt` (`internal/dao/art.go:60-83`) is the shape: a single `QueryRowContext` running one `INSERT ... RETURNING` wrapped in a CTE, scanned into a `model.ArtRow`. The read half of that statement joins `users` and `user_roles` for the display columns, which is fine; what a DAO method may not do is write a second table. That restriction is what makes section 3.6's transaction rule enforceable.

**Repeated shapes are generic and embedded by promotion.** Comments, likes, media attachments, view counters and votes are each written once and parameterised by table and foreign-key name at construction: `newCommentDAO[K comparable](db, table, fk, likesTable, mediaTable)` (`internal/dao/comments.go:28`), `newLikeDAO(db, table, fk)` (`likes.go:19`), `newMediaDAO(db, table, fk)` (`media.go:21`), `newViewDAO(db, viewsTable, fk, entityTable)` (`views.go:18`) and `newVoteDAO(db, table, fk, action)` (`votes.go:20`). Each domain DAO embeds the pointer, so the methods promote onto the outer struct and satisfy the domain interface without a line of forwarding. Nine comment systems share the one implementation: announcements, art, fanfics, journals, mysteries, OCs, posts, secrets and ships. Eight of them embed `*commentDAO[uuid.UUID]`; secrets embed `*commentDAO[string]`, because a secret is keyed by slug rather than by UUID, which is the reason the type parameter exists at all.

**The single deliberate exception is `internal/repository/search.go`.** It holds the `SearchSource` registry: a struct of SQL fragments per searchable entity (`:28-47`) and one entry per entity in `searchSources` (`:83`), currently 22 of them. Each entry names its `From` clause, its author and parent joins, its ID, title and body expressions, its `search_vector` column and its trigram columns. The search DAO assembles those fragments into the union query. The registry sits in the repository package rather than the DAO package because adding a searchable entity is a declaration, not a query, and putting it next to the interface keeps the whole surface of "what is searchable" on one screen.

**DAO tests run against a real database.** They boot a `postgres:18` container per test binary via testcontainers-go, then create a per-test database from a pre-migrated template. The public test API is `daotest.NewRepos(t)`, `daotest.CreateUser(t, repos, opts...)` and `daotest.CreateSession(t, repos, userID)` (`internal/dao/daotest/daotest.go:96,136,172`), and the image name is pinned in one place (`internal/db/dbtest/dbtest.go:21`). These tests need Docker on the host, which is why they are the one layer that cannot be run everywhere.

### 3.6 Transactions, and who may open one

**Every DAO and repository method takes a trailing `tx ...*sql.Tx`.** The variadic is the whole mechanism: a caller with a transaction in hand passes it, a caller without one passes nothing, and no method needs two versions of itself. `txOrDB(db, tx)` (`internal/dao/tx.go:16-22`) returns `tx[0]` when one was supplied and the pool otherwise, typed as a three-method `dbtx` interface that `*sql.DB` and `*sql.Tx` both satisfy. It has 668 call sites across 51 files, which is very close to "every statement in the codebase", and it is the reason a statement does not have to know whether it is inside a transaction.

**A repository that needs several writes to be atomic opens the transaction.** `db.WithTx(ctx, r.db, tx, func(tx *sql.Tx) error)` (`internal/db/tx.go:26-32`) is used 72 times across 23 files under `internal/repository`. `artRepository.CreateWithTags` (`internal/repository/art.go:175-193`) is the canonical shape: open once, call `r.dao.CreateArt(ctx, spec.NewArt, tx)` then `r.dao.InsertTags(ctx, created.ID, spec.Tags, tx)`, and let the deferred rollback undo both if the second fails.

**`WithTx` joins an inbound transaction rather than nesting it.** If `tx` is non-empty it simply calls `fn(tx[0])` and returns, leaving commit and rollback to whoever opened it; only when `tx` is empty does it fall through to `withTx`, which begins, defers a rollback and commits. This matters because `database/sql` has no savepoint API, so a genuine nested transaction is not expressible. Joining means a composite repository method is safe to call both standalone and as one step of a larger unit of work, which is what lets `chatWatchPartyRepository` and `chatbotRepository` compose other repositories without either of them knowing who started the transaction.

**The rule: a DAO may only open a transaction whose every statement hits its own table.** Anything crossing tables moves up to the repository, where the composite method can be named after the unit of work. Four DAO methods qualify today, and each one is a batch against exactly one table: `permissionDAO.SetRolePermissions` (`internal/dao/permission.go:32`) and `SetVanityRolePermissions` (`:66`) clear then reinsert `role_permissions` and `vanity_role_permissions`, `settingsDAO.SetMultiple` (`settings.go:62`) upserts a map of keys into `site_settings` in a loop, and `ocDAO.Update` (`oc.go:81`) wraps its single `UPDATE ocs` so that a zero-rows-affected ownership failure rolls back rather than returning an error after a committed write. No service and no controller opens a transaction anywhere in the tree.

```
  service                         repository                        dao
  ───────                         ──────────                        ───

  CreateArt(spec) ──────────────▶ CreateWithTags(ctx, spec)
                                       │
                                       │ db.WithTx(ctx, r.db, tx, func(tx) error {
                                       │
                                       ├──▶ dao.CreateArt(ctx, spec, tx)  ──▶ INSERT INTO art
                                       │
                                       └──▶ dao.InsertTags(ctx, id, tags, tx) ──▶ INSERT INTO art_tags
                                         })
                                       │
                                       │ commit, or roll back both
                                       ▼
                                  (*model.ArtRow, error)

  one transaction, two tables, one method name the service can read.
  each dao method still runs on txOrDB(db, tx): the tx it was handed,
  or the pool when it was handed none.
```

### 3.7 The cache seam

`internal/cache` exposes one type, `*cache.Manager`, holding an ordered list of `engine.Engine` implementations (`internal/cache/manager.go:12-14`). `cache.New()` builds it with Valkey first and a byte-capped in-memory LRU second (`internal/cache/factory.go:8-17`), and `current()` (`manager.go:123-135`) returns the first engine reporting `Enabled()`. The in-memory engine is always enabled, so losing Valkey degrades the cache rather than removing it. Above that sit four generic methods, which are generic methods rather than package functions because Go 1.27 allows it: `m.Get[T]`, `m.Set[T]`, `m.SetMany[T]` and `m.Load[T]` (`internal/cache/typed.go`). A nil `*Manager` is a valid receiver on all of them: `Get` returns `ErrMiss`, the setters return nil. Every namespace and its TTL is declared in one file (`internal/cache/keys.go`), never at the call site.

**Interception sits in the repository passthrough.** `permissionRepository` (`internal/repository/permission.go`) is the whole pattern in 78 lines: each getter wraps its DAO call in a closure and hands it to `r.cache.Load(ctx, cache.RolePermissions, load)` (`:32-38`), and each setter calls the DAO first, then `r.cache.Del(ctx, cache.RolePermissions.Key())` and logs if the invalidation fails (`:40-50`). Ten repositories take the manager: chatbot, chatbot base prompt, game room, mystery, permission, role, settings, user, user secret and vanity role.

**The passthrough is the right place for it because it is the only place that is already both.** The layers above cannot cache correctly: a service would have to know which of its repository calls are reads and where every other writer of the same table lives, and a controller cannot see a table at all. The layer below cannot cache correctly either: a DAO method is one statement, so the DAO has no vantage point from which a read and the write that invalidates it are the same subject, and putting a cache there would mean the cache lives inside the thing it is meant to avoid calling. The passthrough is the single narrow point every read and every write of a table already funnels through (section 3.4), which gives three properties at once. A cached read and its invalidation are adjacent in one file, so the question "what invalidates this" is answered by scrolling. There is still exactly one writer per table, so no cache entry can be orphaned by a write that bypassed the seam. And the cache is invisible to both neighbours: the service depends on `PermissionRepository` and cannot tell whether a manager was passed, which is why the service tests in section 3.3 need no cache at all.

**Caching something that is not a table read is allowed above the seam.** Six service-side packages hold a manager directly, across seven files (`internal/og/og.go`, `internal/og/image.go`, `internal/homefeed/service.go`, `internal/giphy/service.go`, `internal/linkpreview/service.go`, `internal/dronebl/service.go`, `internal/dronebl/feed/service.go`), and every one of them caches something no repository owns: a rendered OG image, an assembled meta block, a derived daily aggregate, or a third-party HTTP response. The rule is not "only repositories may touch the cache", it is "a table read is cached at the passthrough or not at all".

### 3.8 Request lifecycle

A typical `POST /api/v1/theories` request walks through a fixed global middleware chain, picks up its auth and permission middleware at the route itself, lands in a controller, then flows down through service, repository, and DAO:

```
 POST /api/v1/theories
     │
     ▼
 ┌─────────────┐   panic guard, stack trace to the log
 │   recover   │
 └──────┬──────┘
        ▼
 ┌─────────────┐   OpenTelemetry server span, trace_id local + X-Trace-ID header
 │   tracing   │
 └──────┬──────┘
        ▼
 ┌─────────────┐   Host must match the base_url hostname
 │ host allow  │   (/health, /livez, /metrics, LiveKit webhook exempt)
 └──────┬──────┘
        ▼
 ┌─────────────┐   helmet: HSTS, nosniff, frame-deny, enforced + report-only CSP
 │  sec hdrs   │
 └──────┬──────┘
        ▼
 ┌─────────────┐   shortcuts 304 responses before hitting handlers
 │    etag     │
 └──────┬──────┘
        ▼
 ┌─────────────┐   per-path Cache-Control (immutable assets, no-cache API,
 │ cache hdrs  │   split playlist/segment rules for /hls)
 └──────┬──────┘
        ▼
 ┌─────────────┐   origin gated against live SettingBaseURL or a Capacitor app origin
 │    CORS     │
 └──────┬──────┘
        ▼
 ┌─────────────┐   request-scoped client_ip, access log with trace_id
 │   logger    │
 └──────┬──────┘
        ▼
 ┌─────────────┐   JSON 503 on /api unless the caller has manage_settings
 │ maintenance │   (site-info, login, session, and ws stay reachable)
 └──────┬──────┘
        ▼
 ┌─────────────┐   duration and in-flight histograms exported on /metrics
 │   metrics   │
 └──────┬──────┘
        ▼
 ┌─────────────┐   per route: RequireAuth / OptionalAuth / RequirePermission
 │    auth     │   bearer or cookie → session row → ban, lock, and verify gates
 └──────┬──────┘
        ▼
 ┌─────────────┐   binds the DTO, reads userID from ctx.Locals
 │ controller  │
 └──────┬──────┘
        ▼
 ┌─────────────┐   content filter → per-day caps → business rules → DTO mapping
 │   service   │
 └──────┬──────┘
        ▼
 ┌─────────────┐   interface plus cache seam (internal/repository)
 │ repository  │
 └──────┬──────┘
        ▼
 ┌─────────────┐   all SQL, db.WithTx for multi-table writes (internal/dao)
 │     DAO     │
 └──────┬──────┘
        ▼
 ┌─────────────┐   PostgreSQL, FKs enforced, native UUID/JSONB/CITEXT
 │     DB      │
 └─────────────┘
```

Rate limiting is not part of the global chain. `RateLimitCredentials` (10 per minute per IP) and `RateLimitMail` (5 per hour per IP) are attached only to the auth routes, and Turnstile only to register, login, and forgot-password.

### 3.9 Cross-cutting services

Not every service belongs to a domain. `session`, `authz`, `settings`, `contentfilter`, `notification`, `media.Processor`, `upload`, `email`, `push`, `block` and the two `ws.Hub` instances are built before any domain service in `init_services.go` and then handed to the domains that need them, which is why they sit in the upper half of the diagram in section 3.10. `theory.NewService` is representative: four of its eleven collaborators are repositories and the other seven are shared services and clients (`authz`, `block`, `notification`, `settings`, `credibility`, the quote-API client and the content filter).

They obey the same rules as a domain service. They take interfaces through a constructor (`session.NewManager(repo repository.SessionRepository, settingsSvc settings.Service)`, `notification.NewService(repo repository.NotificationRepository, ...)`), they reach the database through a repository and never through a DAO, and where one owns process state that a domain must not duplicate (the hub's client registry, the media processor's job channel) it keeps that state private and exposes methods over it. Each has its own chapter below: sessions in 6.3, permissions in 6.5, the content filter in 6.6, the hub in 6.7, notifications in 6.8, the media pipeline in 6.9.

### 3.10 Composition and shutdown

Wiring is explicit and split across four files at the repo root. There is no DI container: `initServices` builds every service in dependency order and returns the `services` struct, which is the dependency graph.

- `init_db.go` (`initDatabase`): telemetry, `db.Open`, `db.Migrate`, `db.SeedContent`, the repositories, and the settings service. It receives the cache manager as a parameter rather than building it, because `initServer` needs the same manager for both `store.New` and the shutdown path. Once settings are loaded it re-applies the log level and applies the OTLP endpoint and the Pyroscope endpoint.
- `init_services.go` (`initServices`): every service, in dependency order.
- `server.go` (`initServer`, `initApp`): builds the Fiber app, installs middleware, assembles the services into a `controllers.Service`, registers routes, and returns the app plus a cleanup func.
- `init_jobs.go` (`registerListeners`): settings listeners and the background job tickers, returning a stop func.

```
  config + env
     │
     ▼
  cache.New()                                              (server.go)
     │   valkey engine, then always-on in-memory LRU
     ▼
  telemetry.Init → db.Open → db.Migrate → db.SeedContent    (init_db.go)
     │
     ▼
  store.New(db, cache)  ──────────────────▶  one repo per domain,
     │                                       SQL in internal/dao
     ▼
  settings.NewService (DB-backed, hot reload)
     │  └─▶ re-apply log level, OTLP, Pyroscope
     ▼                                                  (init_services.go)
  session, media.Processor, upload, authz, giphy + banlist,
  contentfilter, user, ws.Hub (main and overlay), email, push,
  block, overlay, notification, report, hyperbeam, livekit, stream
     │
     ▼
  domain services: chat, post, openai + chatbot, follow, art, ship, oc,
                   mystery, fanfic, journal, secret, gameroom, announcement,
                   homefeed, sidebar, vanityrole, usersecret, search, auth,
                   health, sitemap, siteinfo, og.Resolver, og.ImageService
     │
     ▼                                                        (server.go)
  fiber.New → middleware.Setup(app, settings, session, authz)
            → metrics + pprof routes
            → controllers.Service{...} → routes.PublicRoutes(ctrl, app)
     │
     ▼                                                     (init_jobs.go)
  settings listeners + background jobs (orphaned uploads, notification
  prune, expired sessions, crawler ranges, journal and room archiving,
  idle games, voice presence, live-stream reconcile)
     │
     ▼
  utils.StartServerWithGracefulShutdown(app, ":4323")         (main.go)
```

A few edges are genuinely circular and are closed with setters after construction rather than by a container: `sessionMgr.SetDisconnector(hub)`, `streamSvc.SetChatBinder(chatSvc)`, `chatSvc.SetMessageObserver(chatbotSvc)`, and `postSvc.SetCommentObserver(chatbotSvc)`.

Shutdown runs the other way. The cleanup func returned by `initServer` (`server.go:148-167`) drains the chatbot, drains the media processor, drains the game-room tickers, stops the background jobs, and closes the cache, all sharing one 15-second budget (`drainTimeout`, `server.go:73`).

### 3.11 Rationale: what each boundary keeps findable

Each of these four boundaries is justified by one question it keeps cheap to answer. A boundary that does not make some question cheaper is decoration, and erodes within a year because nobody can say what it was protecting.

**Controller and service**, so that "what does this endpoint accept" and "what is the rule" are never the same question. The first is answered by reading twenty lines in `internal/controllers` that never mention a table; the second by reading the service method, which never mentions a status code. Collapsing them means every rule change forces a reader through parameter parsing, and every parameter change risks a rule, which is how a route file becomes the place nobody wants to open.

**Service and repository**, so a service can be tested without a database. This is the boundary with the most direct payoff, because the interface is the seam mockery generates against: `theory.NewService` takes `repository.TheoryRepository`, `.mockery.yml:159-161` sets `all: true` for the whole `internal/repository` package so every interface in it gets a generated mock, and `internal/theory/service_test.go` gets a strict mock that fails on an unexpected call rather than silently absorbing it. That is the difference between 1302 lines of rule tests that run in milliseconds and a suite that needs Docker to check whether a per-day cap works.

**Repository and DAO**, so there is exactly one writer per table and exactly one place a query can live. One writer is what makes the cache seam correct at all (section 3.7): if a second path could write `role_permissions`, no invalidation would be trustworthy. One place per query is what makes a schema change tractable, because renaming a column is a search of one package rather than a search of the tree.

**DAO and database**, so "which statements touch this table" is answerable by opening one file. `internal/dao/art.go` is the complete answer for `art`, `art_tags` and `galleries`, and the generic shared files (`comments.go`, `likes.go`, `media.go`, `views.go`, `votes.go`) are the complete answer for the shapes nine domains have in common, with `dao.NewArt` (`internal/dao/new.go:60-67`) naming exactly which tables the generic halves were bound to. Without that boundary the honest answer to a question about a table is "grep the whole repository and hope", which is exactly the answer this layering exists to make unnecessary.

## 4. Frontend

Four layers and one named sub-layer, described here as the tree stands after the last phase of `design/FRONTEND_REARCHITECTURE.md`. Where the plan and the tree disagree, this section follows the tree and says so.

### 4.1 The layers at a glance

```
 ┌────────────────────────────────────────────────────────────────────────┐
 │ 1 render      src/components/** (153), src/pages/** (115), App.tsx     │
 │               receives data and callbacks, renders them, owns only     │
 │               view state that dies with the component.                 │
 └───────────────────────────────────┬────────────────────────────────────┘
                                     ▼
 ┌────────────────────────────────────────────────────────────────────────┐
 │ 2 orchestration                                                        │
 │               src/hooks/** (131), src/context/** (12),                 │
 │               src/games/*/hooks/**                                     │
 │               the frontend's service layer: queries, mutations,        │
 │               invalidation, socket subscriptions, derived state,       │
 │               permission-dependent behaviour, navigation.              │
 │  ┌──────────────────────────────────────────────────────────────────┐  │
 │  │ 2a data hooks   src/hooks/queries/** (31), src/hooks/mutations/**│  │
 │  │                 (25). The only modules outside src/api that may  │  │
 │  │                 import api/endpoints or the api/queryClient      │  │
 │  │                 singleton. Each one binds a queryKeys builder to │  │
 │  │                 an endpoint function and does nothing else.      │  │
 │  └──────────────────────────────────────────────────────────────────┘  │
 └───────────────────────────────────┬────────────────────────────────────┘
                                     ▼
 ┌────────────────────────────────────────────────────────────────────────┐
 │ 3 pure        src/domain/** (43), src/utils/** (7), src/types/** (2),  │
 │               src/games/** minus its hooks                             │
 │               plain TypeScript: validation, transformations, state     │
 │               machines, reducers, parsers, selectors, permission       │
 │               rules. No React import, no api import.                   │
 └───────────────────────────────────┬────────────────────────────────────┘
                                     ▼
 ┌──────────────────────────────────┐ ┌───────────────────────────────────┐
 │ 4 api      src/api/** (65)       │ │ 4 platform  src/platform/** (11)  │
 │   adapts to the server: client,  │ │   adapts to the device:           │
 │   endpoints/ (29 modules),       │ │   capabilities, sound, install    │
 │   queryKeys, queryClient,        │ │   prompt, OTA mechanics, native   │
 │   authToken, telemetry, ota,     │ │   and web push, desktop           │
 │   realtime/ (transport, typed    │ │   notifications, site origin,     │
 │   event contract, sync/), cache/,│ │   popout windows, last location.  │
 │   beacons/, livekit/, hyperbeam/ │ │   Never imports api.              │
 └──────────────────────────────────┘ └───────────────────────────────────┘
```

`src/types/api.ts` holds the wire DTOs and is importable from any layer, because a type is not a dependency. `src/main.tsx` is the module-graph composition root and is exempt from the import rule by design, the same way `App.tsx` is the React composition root. `src/test-utils/**` belongs to no layer and is covered by the catch-all described below.

The pure layer is the one that pays for itself: `src/domain/**` is eight subdirectories (`art`, `chat`, `fanfic`, `games`, `live`, `mystery`, `user`, `watchParty`) plus flat modules such as `permissions.ts`, `mentions.ts` and `notifications.ts`, and every one of them is testable with no renderer and no mocked transport. `src/utils/**` survives shrunk to seven app-agnostic helpers (`download`, `errorMessage`, `fileValidation`, `gif`, `nonFatal`, `time`, `youtube`); anything with domain vocabulary in it moved to `domain/`.

### 4.2 Allowed import directions

| From \ May import           | render | orchestration | data hooks | pure | api (non-transport) | api/endpoints, api/queryClient | platform                     | types |
|-----------------------------|--------|---------------|------------|------|---------------------|--------------------------------|------------------------------|-------|
| render                      | yes    | yes           | yes        | yes  | **no**              | **no**                         | yes, unenforced              | yes   |
| orchestration               | no     | yes           | yes        | yes  | yes                 | **no**                         | yes                          | yes   |
| data hooks                  | no     | no            | yes        | yes  | yes                 | yes                            | no                           | yes   |
| pure (domain, utils, games) | no     | no            | no         | yes  | **no**              | **no**                         | no                           | yes   |
| api                         | no     | **no**        | no         | yes  | yes                 | yes                            | `platform/capabilities` only | yes   |
| platform                    | no     | **no**        | no         | yes  | **no**              | **no**                         | yes                          | yes   |
| test-utils                  | yes    | yes           | yes        | yes  | yes                 | **no**                         | yes                          | yes   |
| main.tsx                    | yes    | yes           | yes        | yes  | yes                 | yes                            | yes                          | yes   |

"api (non-transport)" means `api/queryKeys`, `api/realtime`, `api/cache`, `api/livekit`, `api/beacons`, `api/telemetry`. Two cells differ from the plan's version of this table and are written here as the tree is:

- **render to platform is not restricted.** The plan reserved "predicates only" for render, meaning `platform/capabilities` and nothing else. No rule was ever written for it and the tree does not obey it: render files import `platform/siteOrigin` seven times, `platform/lastLocation` twice, and `platform/installPrompt` and `platform/appUpdate` once each, against three imports of `platform/capabilities`. The restriction that does exist and is enforced is the one on `api`, where the whole adapter imports `platform` exactly twice and both are the permitted predicate module: `api/client.ts:2` takes `clientPlatform` and `isNativeApp`, and `api/realtime/socket.ts:2` takes `isNativeApp`.
- **test-utils is a row of its own.** It is not a layer, but the catch-all block gives it the same transport ban the orchestration layer has, so a fixture cannot reach `api/endpoints` or `api/queryClient` either.

The hard rules behind the table are unchanged: no layer skipping, render never touches cache internals or fetches, pure never imports React, query keys are built in `src/api/queryKeys.ts` alone, and the adapters never import upward.

### 4.3 What enforces each row

One built-in oxlint rule id and five rules from a local oxlint plugin do all of it, configured per glob in `frontend/oxlint.layers.ts` and spread into `frontend/oxlint.config.ts` as `layerRules`. `no-restricted-imports` is oxlint's own; the five `layers/*` rules live in `frontend/oxlint-plugin-layers.mjs`, which is a JS plugin loaded through `jsPlugins` and holds the AST selectors oxlint has no built-in rule for. Lint runs at `--max-warnings=0`, so a new violating file fails on its first commit.

| Boundary                                           | Rule id                                   | Where it is configured                                                                                     |
|----------------------------------------------------|-------------------------------------------|------------------------------------------------------------------------------------------------------------|
| render may not import `src/api` at all             | `no-restricted-imports`                   | `RENDER` block, group `**/api/*`, `**/api/**`, `allowTypeImports: false`                                   |
| render may not import react-query                  | `no-restricted-imports`                   | `RENDER` block, group `@tanstack/react-query`, `allowTypeImports: false`                                   |
| orchestration may not import the transport modules | `no-restricted-imports`                   | `ORCHESTRATION` block, `excludeFiles: DATA_HOOKS`, groups `api/client`, `api/queryClient`, `api/endpoints` |
| data hooks may import the transport                | `no-restricted-imports`                   | `DATA_HOOKS` block sets the rule to `off`                                                                  |
| pure may not import `src/api`                      | `no-restricted-imports`                   | `PURE` block, group `**/api/*`, `**/api/**`                                                                |
| pure may not import React or react-query           | `no-restricted-imports`                   | `PURE` block, group `react`, `react-dom`, `@tanstack/react-query`                                          |
| adapters may not import upward                     | `no-restricted-imports`                   | `API` and `PLATFORM` blocks, group `**/components/**`, `**/pages/**`, `**/hooks/**`, `**/context/**`       |
| `src/api` may import `platform/capabilities` only  | `no-restricted-imports`                   | `API` block, group `**/platform/*`, `**/platform/**` with `!**/platform/capabilities`                      |
| `src/platform` may not import `src/api`            | `no-restricted-imports`                   | `PLATFORM` block, group `**/api/*`, `**/api/**`                                                            |
| everything unclaimed may not reach the transport   | `no-restricted-imports`                   | `EVERY_SOURCE_FILE` block, `excludeFiles` the five layers plus `src/main.tsx`                              |
| query keys are built in one file                   | `layers/no-raw-query-key`                 | armed on every source file, switched `off` for `CACHE_KEY_ASSERTING_TESTS` and for `src/api/queryKeys.ts`  |
| dynamic and inline type imports of `src/api`       | `layers/no-dynamic-api-import`            | armed on every source file                                                                                 |
| only the transport's own tests stub global fetch   | `layers/no-stub-global-fetch`             | armed on every source file, switched `off` for `MAY_STUB_GLOBAL_FETCH`                                     |
| a render test mocks hooks, not the transport       | `layers/no-transport-mock-in-render-test` | armed on `RENDER_TESTS`, switched `off` for `RENDER_TESTS_STILL_MOCKING_TRANSPORT`                         |
| a pure test needs no DOM                           | `layers/no-render-import-in-pure-test`    | armed on `PURE_TESTS`                                                                                      |

The layer globs match `*.test.ts` and `*.test.tsx` as well as source, so a test file is bound by the layer it sits in. `src/main.tsx` is exempt from the import rule only; the three `layers/*` rules armed on every source file still apply to it.

Where ESLint expressed the last five rows as one `no-restricted-syntax` rule holding a list of selectors, and relaxed a file by re-listing a shorter list, oxlint expresses them as five separate rules and relaxes a file by switching one of them `off`. The selector strings and the message text are unchanged: oxlint runs real esquery inside a JS plugin, so they moved across verbatim.

Two rows are conventions with no rule behind them, and are listed here so nobody mistakes them for enforced: render to platform (above), and "the pure layer holds no domain vocabulary in `utils/`", which is a naming judgement a linter cannot make.

### 4.4 The documented exceptions

There are four. A fifth may not be added without adding it here.

1. **`src/api/realtime/sync/**`** holds twelve hooks that subscribe to a typed socket event and patch the react-query cache. Eight of them (`useNotificationFeedSync`, `useUserIdentitySync`, `usePermissionsSync`, `useChatUnreadSync`, `useLiveGamesCountSync`, `useChatbotsSync`, `useStreamDirectorySync`, `useSiteInfoSync`) are composed by `RealtimeSync.tsx`, which the notification provider renders; three of the other four are pulled in by the feature hook that needs them (`useGameRoomSync` by `hooks/queries/gameRoom.ts`, `useOwnOCSync` by `hooks/queries/oc.ts`, `useStreamDetailSync` by `hooks/useLiveStream.ts`), and the fourth, `useSecretAnnouncementSync`, is called straight from a render component (`components/secrets/SecretClosedToast/SecretClosedToast.tsx`) through the `src/hooks/useSecretAnnouncementSync.ts` re-export, because its toast has no feature hook to compose it into. They import React and react-query, so by the letter of 4.1 they are layer 2. They live under `api/realtime` because they are the server-push half of the same transport the data hooks read. Nothing enforces this either way: the react ban exists on the pure layer only, and `src/api` was never given one.
2. **The five quote-API functions** in `src/api/endpoints/quote.ts` call global `fetch` directly (`:30`, `:59`, `:73`, `:86`, `:102`). That is a different origin with a different error contract and no session cookie, so routing them through `api/client` would be wrong. `src/api/endpoints/quote.test.ts` is in `MAY_STUB_GLOBAL_FETCH` for that reason.
3. **`api/ota.ts:getOtaManifest`** calls global `fetch` (`src/api/ota.ts:7`) for `/app-bundles/latest.json`, a static file on our own origin rather than an `/api/v1` route: no session, `cache: "no-store"`, and no envelope for `api/client` to unwrap. `src/api/ota.test.ts` is in `MAY_STUB_GLOBAL_FETCH`.
4. **`api/beacons/watchPartyLeave.ts`** calls global `fetch` with `keepalive: true` on page unload. It borrows `apiUrl` and `authHeaders` from `api/client` but issues the request itself, because `api/client` has no way to express a keepalive beacon. This is the exception the plan did not record: before the refactor the same `fetch` sat in a render-layer hook (`components/chat/WatchParty/useWatchParty.ts` on `b952a8a`), so moving it into `src/api/beacons` narrowed the bypass rather than creating one. `src/api/beacons/watchPartyLeave.test.ts` is in `MAY_STUB_GLOBAL_FETCH`.

### 4.5 The two standing holdout lists

The Phase 4 migration exemptions are gone: `migrationExemptions` no longer exists, and `oxlint.config.ts` spreads `layerRules` alone. Two lists remain inside `layerRules`, and neither is a migration leftover.

**`CACHE_KEY_ASSERTING_TESTS`, 21 files, permanent.** These are test files that write a query-key array literal, and they should. Two reasons, both structural. A data-hook test that asserts which key a mutation invalidated has to state the literal (`expect(invalidate).toHaveBeenCalledWith({ queryKey: ["admin"] })` in `src/hooks/mutations/admin.test.ts`): calling the builder there would compare the builder to itself and pass whatever the builder did. And `src/api/cache/patchUser.test.ts` and `src/api/queryClient.test.ts` seed synthetic cache entries such as `["profile", "u1"]` and `["theories"]` to exercise generic cache machinery, where the arrays are test data rather than app keys. The block switches `layers/no-raw-query-key` off and leaves every other rule armed, so those files are still held to the dynamic-import and global-fetch rules.

**`RENDER_TESTS_STILL_MOCKING_TRANSPORT`, 7 files, open.** These render tests `vi.mock` an `api/endpoints/*` module so that the component's real data hook runs against a faked transport. They are genuine violations of the rule in 4.3, kept as a ratchet holdout rather than a blessing: every render test not on this list is held to the rule, and a new one cannot join. Draining them means moving each page's data assertions down into the owning hook's test, which is the work that was done for `ChatPage`, `RoomPage`, `LiveDirectory` and `StreamChatPopout` and is still owed for `WatchPartyModal`, `GameChat`, `GoLivePanel`, `LiveWatchPage`, `StreamChatPanel`, `StreamOverlaySection` and `RoomsListPage`.

## 5. Where the halves meet

One process serves both halves: `//go:embed static/*` (`server.go:77-78`) compiles the built Vite bundle into the binary, so a browser loads the SPA and calls the API on the same origin. Three surfaces cross the line, and nothing else does.

- **JSON under `/api/v1`.** Every API route hangs off one group (`internal/routes/public_routes.go:11`) and answers either with a type from `internal/dto` or with a `fiber.Map` literal. Those shapes are mirrored by hand in `frontend/src/types/api.ts`; nothing generates either side from the other, so a field added in Go is only real to the frontend once it is added there too.
- **The WebSocket at `/api/v1/ws`.** Same origin and same session cookie, with a `?token=` query parameter for the packaged app, which has no cookie jar (`frontend/src/api/realtime/socket.ts:46-50`). The hub is section 6.7.
- **The HTML document itself.** The SPA catch-all rewrites the meta tags of the embedded `index.html` per path before serving it (section 6.11), which is the one place the server has an opinion about the page the client is about to render.

The mobile app adds no fourth surface. It is the same SPA (section 7.2), crossing the same three with a bearer token instead of a cookie and with FCM instead of a socket while it is backgrounded.

## 6. Subsystem detail

### 6.1 Data Layer

- **All SQL lives in `internal/dao/`**, one file per domain (theory.go, post.go, art.go, mystery.go, ship.go, fanfic.go, journal.go, chat.go, permission.go, etc.). The DAO structs are unexported and built through `dao.NewTheory(db)` and friends in `internal/dao/new.go`.
- **`internal/repository/` owns the contract, not the queries.** Each domain file declares the interface services depend on, the row models (`internal/repository/model/`), and a thin passthrough struct that wraps the DAO. `internal/store/new.go` is the single wiring point: `repository.NewXRepo(dao.NewX(db), cache)`, or `repository.NewXRepo(db, dao.NewX(db), cache)` for the repositories that own a transaction.
- **The passthrough is the cache seam.** Because every read and write already funnels through it, it is the only place a table read is allowed to be cached: read-through on gets, explicit `Del` on writes. `internal/repository/permission.go` is the canonical example, with `r.cache.Load(ctx, ns, load)` on the gets and `r.cache.Del(ctx, ns.Key())` on `SetRolePermissions` and `SetVanityRolePermissions`. Section 3.7 sets out why this layer and not one either side of it.
- **`internal/cache` is a hot-reloadable manager over an ordered list of engines.** The `valkey_url` site setting swaps the Valkey client at runtime; a byte-capped in-memory LRU sits behind it and is always enabled, so clearing the URL degrades the cache to process-local rather than switching it off. The typed surface is four generic methods on `*Manager` (`m.Get[T]`, `m.Set[T]`, `m.SetMany[T]` and `m.Load[T]`), which JSON round-trip any value and pass `string` and `[]byte` through untouched, and namespaces with their TTLs are declared in `internal/cache/keys.go`. A nil `*Manager` is a valid receiver on all four, so a caller never has to know whether caching is switched on. Hits, misses, command latency, and Valkey server stats are exported to Prometheus on `/metrics`. Ten repositories currently take the manager: user, role, settings, mystery, vanity role, permission, user secret, game room, chatbot, and chatbot base prompt.
- **Shared DAOs for repeated shapes**: comments, likes, media, and view counters are generic over the parent key (`newCommentDAO[K]`, `newLikeDAO`, `newMediaDAO`, `newViewDAO`) and embedded into each domain DAO by promotion, so nine comment systems share one implementation parameterised by table and foreign-key name.
- **Transactions are owned by the repository, never by the DAO.** Every DAO and repository method takes an optional trailing `tx ...*sql.Tx`; `txOrDB(db, tx)` in `internal/dao/tx.go` runs the statement on that transaction when one is supplied and on the pool otherwise. A repository that needs several writes to be atomic opens the transaction with `db.WithTx(ctx, db, tx, fn)` from `internal/db/tx.go` and threads it through each DAO call, which is how one unit of work can span several DAOs (e.g. `CreateWithCharacters`, `UpdateWithTags`, `MarkSolved`, `CreateBotWithAccount` writing `users`, `user_vanity_roles` and `chatbots` together). `WithTx` joins an inbound transaction rather than nesting, since `database/sql` has no savepoints. **A DAO may only open a transaction whose every statement hits its own table**, which is why only four remain in `internal/dao` (`settings.SetMultiple`, both `permission.Set*Permissions`, and `oc.Update`). Services still do not handle transactions directly.
- **DAO and repository interfaces are split where the repository orchestrates.** Most domains declare a single `XRepository` implemented twice, by the DAO and by the passthrough. Domains that own a cross-DAO transaction instead declare `XDAO` (the database methods) and `XRepository interface { XDAO; ...composites... }`, so the type system prevents a DAO from ever implementing an orchestration method. `dao.NewX` returns `repository.XDAO` for those domains.
- **Native Postgres types** throughout the schema: `UUID` for primary and foreign keys, `BIGINT GENERATED BY DEFAULT AS IDENTITY` for auto-increment columns, `BOOLEAN` for flags (no more `INTEGER 0/1`), `TIMESTAMPTZ` for time columns, `JSONB` for `state_json` / `action_json`, and `CITEXT` (case-insensitive text) for unique-by-name lookups like fanfic series, languages, and OC characters.
- **Foreign keys** are enforced by Postgres. Most deletes cascade through `ON DELETE CASCADE`; `galleries -> art.gallery_id` is `ON DELETE SET NULL`, so `artRepository.DeleteGallery` explicitly removes child art and the gallery row inside one transaction.
- **Hot-reloadable settings** live in the `site_settings` table and are served through `internal/settings`. Listeners registered at startup react to changes (e.g. re-reading the log level, reconnecting the cache) without a server restart.
- **The one exception to "no SQL outside the DAO" among query code** is `internal/repository/search.go`, which holds the `SearchSource` registry: a declarative SQL fragment per searchable entity that the search DAO assembles into the union query. Schema and seed SQL live one layer lower again, in `internal/db` (`migrations/` and `seed.go`), which is a different category: those statements run at boot, not per request.
- **DAO tests** boot a real `postgres:18` container per test binary via testcontainers-go, then create a per-test database from a pre-migrated template. Public test API: `daotest.NewRepos(t)`, `daotest.CreateUser(t, repos, opts...)`, `daotest.CreateSession(t, repos, userID)`. Tests need Docker on the host.

```
  controller ──▶ service ──▶ repository ──▶ dao ──▶ txOrDB(db, tx).ExecContext(...)
                                 │           ▲        one table per method
                                 │           │
                                 ▼           │      the tx the repository passed in,
                          internal/cache     │      or the pool when it passed none
                          (read-through,     │
                           Del on write)     │
                                             │
      db.WithTx(ctx, db, tx, func(tx) error {
          r.dao.CreateArt(ctx, ..., tx)          ── art
          r.dao.InsertTags(ctx, id, tags, tx)    ── art_tags
      })                                          one method per logical operation,
                                                  atomic across both tables
```

### 6.2 Cache (Valkey)

`internal/cache` is a read-through cache in front of the hot lookups every request makes. Its Valkey engine stays off unless a `valkey_url` is saved, and that Valkey is deliberately a different instance from the one LiveKit's ingress and egress coordinate over.

- **An ordered list of engines, not a single client**: `cache.New()` builds the manager with a Valkey engine first and a byte-capped in-memory LRU second (`internal/cache/factory.go`), and every operation runs against the first engine reporting `Enabled()`. The in-memory engine is always enabled, so an unconfigured or unreachable Valkey degrades the cache to process-local rather than removing it. Only a nil `*Manager` disables caching outright.
- **Hot-reloadable connection**: the Valkey engine subscribes to the `valkey_url` site setting. Saving a new URL closes the old client and opens and pings a new one; clearing the URL closes the client and drops that engine out of the rotation, leaving the in-memory engine to serve. No restart, and the admin panel probes the URL as a setting validator so an unreachable address is rejected at save time rather than at first read.
- **Typed helpers**: `m.Get[T]`, `m.Set[T]`, `m.SetMany[T]` and `m.Load[T]` are generic methods on `*Manager` (`internal/cache/typed.go`) and marshal through JSON, with `string` and `[]byte` passed through untouched. A nil manager or a missing key both return `ErrMiss`, so every call site is a plain cache-miss branch and no caller has to know whether caching is switched on. `Load` is the one most call sites use: it takes a namespace and a loader closure and collapses get, miss, load and set into a single line.
- **Namespaces own their TTL**: each cached thing is a `cache.Namespace` with a key prefix and a TTL, declared in one file (`internal/cache/keys.go`) rather than scattered across call sites. Site settings, user roles, vanity assignments, secret progress, and the mystery and game leaderboards are held indefinitely and invalidated by whoever writes them; the authz role and vanity permission tables hold for a minute; OG metadata for five minutes and rendered OG images for a day.
- **Interception of a table read lives in the repository passthroughs**, not in services and not in the DAOs that own the SQL, so a cached read and its invalidation sit next to each other and there is still exactly one writer per table. Six service-side packages hold a manager directly (`og`, `homefeed`, `giphy`, `linkpreview`, `dronebl` and `dronebl/feed`), but none of them caches a table read: what they hold are rendered OG images and meta blocks, a derived daily aggregate, and third-party HTTP responses that no repository owns.
- **Instrumented**: hit and miss counters, per-command duration and error metrics, an OpenTelemetry client span per command, and a collector that scrapes the server's own `INFO` for key count, memory, evictions, and connection stats (see [Observability](#612-observability)).

### 6.3 Auth and Sessions

- Server-side sessions stored in Postgres with httpOnly cookies. No JWTs.
- The cookie (`ut_session`) carries a random 32-byte hex token and nothing else. The `sessions` row stores the **SHA-256 of that token** as its primary key, alongside the user ID and expiry, so a database leak does not hand over usable sessions.
- Cookie flags: `HTTPOnly`, `SameSite=Lax`, `Path=/`, and `Secure` whenever the live `base_url` starts with `https://`. `SettingSessionDurationDays` (default 30) sets both the cookie `MaxAge` and the row's `expires_at`.
- **Sessions are not renewed.** Lifetime is fixed at creation, so changing the duration setting only affects sessions created afterwards. Expired rows are swept by a background job every 24 hours.
- **The mobile app uses the same session, carried differently.** When a request arrives with an `X-Client-Platform` header, login and register echo the token back in an `X-Session-Token` response header (exposed through CORS); the app stores it in Capacitor Preferences and sends it as `Authorization: Bearer <token>` on every later call. `SessionToken()` prefers the bearer header over the cookie, so both clients hit the same validation path. The WebSocket upgrade reads the cookie and falls back to a `?token=` query parameter, since a webview cannot set headers on an upgrade.
- Auth middleware is attached **per route**, not globally: `RequireAuth`, `OptionalAuth`, or `RequirePermission`. Beyond resolving the user it drops the session and returns 403 for banned accounts, blocks write methods from locked accounts, and blocks write methods from unverified emails, each with a small exempt list (marking notifications read, reading a chat room, and the email-verification endpoints themselves).
- Password reset and moderator action delete every session row for the user and immediately disconnect their live WebSockets through the hub, so a revoked account cannot ride out its cookie.

```
   Browser / app          Server
  ───────────────        ──────────
   login form ──────────▶ auth.Login
                          │
                          │ verify credentials, reject if banned
                          │ 32 random bytes → hex token
                          │ INSERT sessions (sha256(token), user_id, expires_at)
                          │
   Set-Cookie ◀───────────┤  ut_session (httpOnly, SameSite=Lax, Secure on https)
   X-Session-Token ◀──────┘  only when the request carried X-Client-Platform
        │
        │ every subsequent request
        ▼
   ┌────────────────┐
   │ auth middleware│── Authorization: Bearer … or the ut_session cookie
   │  (per route)   │── sha256 → sessions row → expiry → ban / lock / verify gates
   └────────────────┘── userID into ctx.Locals
```

### 6.4 Account Security

Registration takes an email address, and the account is only half usable until that address is confirmed.

- **Email verification**: a 24-hour token is mailed on registration and again whenever the address changes. Until it is used, write requests (anything that is not a GET, minus a small exempt list so verifying and resending still work) are refused with an `email_unverified` code. Each user row carries a `verify_grace_until` timestamp, which defaults to "now" for new accounts but let the existing population be migrated with a window rather than locked out on the day the requirement landed.
- **Password reset**: a one-hour token mailed to the address on file. Changing an email also notifies the previous address, so a hijacked account cannot quietly move itself.
- **Ban versus lock**: a ban is terminal, the session is deleted on the next request and the account is refused outright. A **lock** is the softer, reversible option: the account can still read the site but every write is refused. Both record who applied them and why, and both surface on the admin user detail page.
- **Rate limits on credentials**: ten credential attempts per client IP per minute (login, register, reset) and five mail-sending attempts per hour (forgot password, set email, resend verification), keyed off the same resolved client IP the logger uses, so `CF-Connecting-IP` behind Cloudflare rather than the proxy address.
- **Reserved usernames** are refused at registration, and Cloudflare Turnstile can be required on login and registration from the admin panel.

### 6.5 Permission Model

- Every action is gated on a **permission**, not a raw role check. Permissions are declared once in `internal/authz/permissions.go` as a catalogue of `PermissionDef{Permission, Label, Scope}`, and that catalogue is what the admin UI renders.
- **Permissions are editable at runtime.** `/admin/permissions` (gated on `manage_roles`) writes them to the database rather than to a compiled table: the moderator role and every custom vanity role can have permissions ticked on and off, and a change takes effect on save.
- **Admin and super admin are immutable.** They always hold every permission and are deliberately absent from the permissions page, so nothing edited there can lock an administrator out of the site.
- Every permission carries a **scope** that decides who may ever be granted it:
  - `staff`: assignable to the moderator role, never to a vanity role. Most of the catalogue, e.g. `ban_user`, `edit_any_post`, `delete_any_comment`, `view_audit_log`
  - `general`: assignable to the moderator role *and* to vanity roles, which is how an opt-in perk reaches ordinary members. Currently just `use_chatbot`
  - `restricted`: `manage_settings` and `manage_roles`, assignable to neither, because handing either one out is an escalation path to effective admin
- A user's effective permissions are the union of their system role's grants and the grants of every vanity role assigned to them. When the Valkey cache is on, both lookup tables and each user's vanity-role list are cached with a one-minute TTL, so a permission change propagates within about a minute rather than instantly.
- Some features (the "game master" view in mysteries) check `role == super_admin` directly because the behaviour is intentionally scoped to that one role, not to a permission grant. Ownership checks ("is this your own post") are likewise separate from the permission catalogue.

```
  role            permissions
  ──────────────  ─────────────────────────────────────────────────
  super_admin     everything, not editable
  admin           everything, not editable
  moderator       editable, seeded with view_admin_panel, view_users,
                  ban_user, edit_any_*, delete_any_*, use_chatbot, ...
  vanity roles    editable, general-scope permissions only
  member          no catalogue permissions; owns its own content
```

### 6.6 Content Filter

`internal/contentfilter` is a pluggable validation pipeline. Every text-bearing service runs its payload through the manager before writing: registration (username and display name), profile, theories, posts and comments, art, ships, OCs, mysteries, fanfics, journals, secrets, game-room spectator chat, chat rooms and DMs.

```
   user text ──▶ ┌──────────────────────────────────┐
                 │  contentfilter.Manager           │
                 │                                  │
                 │  ┌─ RuleSlurs       ──┐          │  first failing rule
                 │  └─ RuleBannedGiphy ──┘  ──────▶ │  stops the chain and
                 │                                  │  returns *RejectedError
                 └──────────────────────────────────┘  to the caller
                              │
                              ▼
                         accept → service writes to repo
```

Rules are registered in order in `initServices` (slurs, then banned GIPHY) and each one sees every non-empty text field of the payload in a single call. Text is NFKC-normalised with Unicode format characters stripped before matching, so zero-width joiners and lookalike compositions cannot smuggle a match past a rule.

The banned-GIPHY rule reads the live banlist from `internal/giphy/banlist` and rejects both individual GIF IDs and whole GIPHY channels, resolving the uploader of an unrecognised ID through the GIPHY API when it has to. The admin banned-GIFs UI writes to that list and changes apply instantly without a restart.

The per-room chat word filter lives in the same package but deliberately outside the manager. `ChatBannedWordsRule` is held by the chat service and called as `CheckForRoom(ctx, roomID, texts...)`, because its rules are scoped to one room and carry an action (delete or kick) rather than a plain reject. Compiled patterns are cached per rule row and invalidated when a moderator edits the rule.

### 6.7 WebSocket Hub

`internal/ws` is a single in-process hub that multiplexes every live event on the site. Clients open one socket per tab, the hub keys them by user ID, and services push events through `SendToUser`, `Broadcast`, `BroadcastPublic`, `BroadcastToRoom`, or `BroadcastToTopic`. A signed-out visitor still gets a socket: it registers in the anonymous set, may only ping, and receives `BroadcastPublic` events (live-stream and chatbot changes) so public pages update without an account.

```
  authed clients (many tabs)            anonymous clients (no session)
     │ websocket upgrade                   │ websocket upgrade
     ▼                                     ▼
  ┌───────────────────────────────────────────────────────────────┐
  │                            ws.Hub                             │
  │                                                               │
  │  ┌──────────────┐  ┌──────────────┐  ┌─────────────────────┐  │
  │  │ by user ID   │  │ by room ID   │  │ viewers per room    │  │
  │  │  {u: [c,c]}  │  │  {r: {u,u}}  │  │  {r: {u: tabs,      │  │
  │  └──────────────┘  └──────────────┘  │       active/idle}} │  │
  │  ┌──────────────┐  ┌──────────────┐  └─────────────────────┘  │
  │  │ anon clients │  │ always online│                           │
  │  │  {c, c}      │  │  {bot, bot}  │                           │
  │  └──────────────┘  └──────────────┘                           │
  └──────▲────────────────────▲──────────────────▲────────────────┘
         │ SendToUser         │ BroadcastToRoom  │ Broadcast
         │                    │ BroadcastToTopic │ BroadcastPublic
  ┌──────┴───────┐   ┌────────┴─────┐   ┌────────┴────┐
  │ notification │   │ chat service │   │ like / view │
  │   service    │   │ (msg, react, │   │  counters   │
  │              │   │  pin, typing)│   │             │
  └──────────────┘   └──────────────┘   └─────────────┘
```

- **Rooms** are UUID-keyed sets of user IDs. Chat rooms are joined at connect time from the caller's membership list. **Topics** reuse the same map with a synthetic UUID derived from a string (`TopicUUID`), which is how `secret:<id>` progress and per-game spectator chat fan out only to the people currently looking at that page.
- **Viewer presence** is separate from room membership: `AddViewer` / `RemoveViewer` reference-count open tabs per room and carry an `active` / `idle` state, driven by the client's `join_room`, `leave_room`, and `viewer_state` frames. Losing the last socket clears the viewer rows and broadcasts the departure.
- **Always-online set**: chatbot users are pushed into `SetAlwaysOnline` whenever the chatbot config reloads, so `IsOnline` reports them as present in member lists without a socket of their own, and the notification path does not try to send them a mobile push.
- **Back pressure**: every client has a 64-message buffer and a non-blocking enqueue. A consumer that cannot keep up is killed and reaped rather than stalling the broadcaster.
- **Connection hygiene**: 8 KB inbound frame cap, 100 frames per second with a burst of 200, a 30-second server ping against a 90-second read deadline, and a session revalidation plus ban check at most every five minutes on pong. `session.Manager` holds the hub as its `Disconnector`, so revoking a user's sessions closes their live sockets.
- Connections and inbound frames are exported to Prometheus as `ws_connections{hub,authed}`, `ws_connections_total`, `ws_inbound_messages_total{type}`, and `ws_inbound_dropped_total{authed}`. There are two hub instances: `main` for the site and `overlay` for the OBS browser-source alert feed.

The frontend keeps one socket for the whole app on `/api/v1/ws`, pings every 20 seconds, closes the socket itself if nothing arrives for 90 seconds while the tab is visible, and reconnects with full-jitter exponential backoff capped at 30 seconds. On reconnect it refetches the affected queries; there is no polling fallback.

### 6.8 Notifications

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

### 6.9 Media Pipeline

Every upload lands on disk first, then goes through `media.Processor`, a fixed pool of worker goroutines fed by a buffered channel. Images and video take different routes through it: an image upload waits for its own encode so the caller can persist the final `.webp` URL, while video is recorded at its raw path and transcoded behind the request.

```
   controller receives multipart upload
         │
         ├── image ──▶ upload.Service.SaveImage
         │               original bytes land on disk (uploads/<subdir>/)
         │               pixel guard (max_image_pixels) rejects decode bombs
         │               enqueue JobImage, then block on the callback
         │
         └── video ──▶ media.Uploader.SaveAndRecord
                         original bytes land on disk, media row written,
                         enqueue JobVideo and return to the client
                                 │
   ┌─────────────────────────────┴───────────────┐
   │  buffered job channel (cap 256)             │
   └───────┬─────────────────────────────────────┘
           │ N worker goroutines (4 at startup)
           ▼
   ┌─────────────────────────────────────────────┐
   │ image worker → cwebp                        │
   │   q80 by default, q60 square 96px avatars,  │
   │   q72 1600px banners, EXIF auto-orient,     │
   │   GIF → animated WebP via ffmpeg            │
   │ video worker → ffmpeg libx264 CRF 28,       │
   │   AAC 128k, faststart; .webm untouched      │
   └───────┬─────────────────────────────────────┘
           │
           ▼
   image: caller gets the .webp path, source file removed
   video: callback repoints the media row at the .mp4, removes the source,
          then ffmpeg grabs a random frame as a 200px-tall WebP thumbnail
```

The image path is synchronous on purpose. `SaveImage` enqueues the job and selects on the result, the error, and the request context, so a failed or dropped encode surfaces as a failed upload instead of a media row pointing at a file nobody will serve. A `.webp` upload is re-encoded in place, and an animated one is left alone. The video path is fire and forget: if the transcode fails, the row keeps pointing at the raw upload and the failure is logged.

If the queue is full the job is dropped and its error callback fires immediately rather than back-pressuring the request. Shutdown behaves the same way: the processor stops accepting work, waits up to 15 seconds for in-flight encodes, then fails whatever is still queued.

### 6.10 Background Jobs

Recurring work runs as plain goroutines started at boot by `registerListeners` in `init_jobs.go`. Each job is a `scheduleJob(stop, wg, name, successMsg, interval, fn)`: it runs once immediately, then on a ticker, logs a count only when it actually did something, and stops on the shared `stop` channel at shutdown, with the wait group ensuring a redeploy never kills work halfway through.

| Job                         | Interval   |
|-----------------------------|------------|
| Reconcile voice presence    | 30 seconds |
| Reconcile live streams      | 1 minute   |
| Cancel idle games           | 5 minutes  |
| Archive stale journals      | 1 hour     |
| Archive stale chat rooms    | 1 hour     |
| Clean orphaned upload files | 24 hours   |
| Prune old notifications     | 24 hours   |
| Clean expired sessions      | 24 hours   |
| Refresh crawler ranges      | 24 hours   |

The same function registers the settings listeners that make hot reload work: log level, OTLP endpoint, Pyroscope URL, request body limit, native push credentials, the chatbot opt-in role migrator, the crawler feed list, SMTP, and the chatbot and OpenAI settings blocks. The Valkey URL listener is not named individually: `registerListeners` walks `svc.cache.Engines()` and subscribes any engine that implements `settings.Listener`, so a future engine with its own setting is wired by existing. It also ensures the system chat rooms exist at startup.

Shutdown drains in order: the chatbot worker pool, then the media processor, then the game-room tickers, then the job tickers, then the cache client, all inside a single 15-second budget.

### 6.11 OG and SEO

`internal/og` owns the SEO meta surface. The SPA catch-all serves every extension-less path through `og.Resolver.Resolve(ctx, path, partyID)`, which matches the URL to a resolver and rewrites the meta tags of the embedded Vite `index.html` in memory. The base document already carries `og:type`, `og:locale`, `twitter:card`, and the default image dimensions; the resolver overwrites `<title>`, `description`, `og:title` / `og:description` / `og:url` / `og:site_name` / `og:image`, the matching `twitter:*` tags, and `<link rel="canonical">`. Paths that match nothing keep the site-wide defaults, and `__BASE_URL__` in the document is substituted once at startup.

```
   GET /theory/<id>
       │
       ▼
   og.Resolver.Resolve(ctx, path, party)
       │
       ├─ resolveMeta() ─▶ app cache (og:meta:, 5 min)
       │                     │ miss
       │                     ▼
       │                 metaForPath() ─▶ theoryMeta(ctx, id) ─▶ repo.GetByID
       │                                                          │
       │                                                          ▼
       │                                                     Meta{Title, Description,
       │                                                          Image, URL}
       │
       └─ inject(meta)  ──▶ rewrites og:*, twitter:*, <title>, description
                            and the canonical link in the embedded index.html
```

- Detail resolvers exist for theories, posts, profiles, art, galleries, mysteries, ships, OCs, fanfics, announcements, journals and journal entries, chat rooms, watch parties (`?party=<id>` on a room URL), secrets, live streams, and individual games. Section index pages, game-board corners, gallery corners, and the games hub get static per-page copy.
- Resolved `Meta` is cached in the app cache for five minutes, so a link passed around Discord does not re-query the repo for every unfurl. With no Valkey configured this falls to the in-memory engine, which is still a real cache, just per-process and lost on restart.
- Uploaded images are WebP, which several scrapers still refuse, so `og:image` is rewritten to `/og-image/<path>.jpg`. That route decodes the WebP (downscaling through `dwebp` when it is wider than 1200px, pulling frame one out with `webpmux` when it is animated), encodes JPEG at quality 85, and caches the bytes for 24 hours keyed on the file's modification time and size. If conversion fails it serves the original WebP. When a per-entity image is injected, the static `og:image:width` / `og:image:height` tags are stripped because they no longer describe it.
- The fallback image is the built-in `/Featherine.jpg`, overridable from admin settings (`og_default_image`, which must be an uploaded `.jpg`).

Adding a new page means adding a branch to `metaForPath()`; see [Adding a New Page](ADDING_A_PAGE.md).

### 6.12 Observability

Five independent signals. Metrics, traces, profiling and health need no redeploy: traces and profiling are pointed at their collector from **Admin → Settings**. Logs are the exception: the app only chooses its stdout format, and collection is owned entirely by the observability stack.

- **Metrics**: a Prometheus registry is served on `/metrics`, and every request is timed by route, method, and status (`http_request_duration_seconds`, `http_requests_in_flight`), with static assets and uploads exempt so the histogram is not swamped. Alongside it sit database pool gauges (`db_pool_*`), WebSocket connection and inbound-frame counters (`ws_*`), cache hit and miss counters plus Valkey command latency and server stats (`cache_*`, with `cache_hits_total` and `cache_misses_total` counted in the manager so they cover whichever engine served the lookup), and chatbot invocation, drop, and token counters (`chatbot_*`). `/metrics`, `/health`, `/livez`, and the LiveKit webhook are exempt from host authorisation so an internal scraper can reach them by IP; `docker-compose.prod.yml` carries the matching `prometheus-*` labels for scrape discovery and runs a `postgres-exporter` sidecar.
- **Traces**: an OpenTelemetry span for every HTTP request, with W3C trace context extracted from the incoming headers and the trace ID echoed back as `X-Trace-ID`, plus a span per SQL statement through `otelsql` and per Valkey command through the cache hook. Set an **OTLP endpoint** and a batch exporter is registered on the live tracer provider; clear it and the processor is flushed, shut down, and unregistered, all without a restart.
- **Profiling**: setting a **Pyroscope URL** starts continuous profiling (CPU, alloc objects and space, in-use objects and space, goroutines, mutex, and block) tagged with the hostname, and clearing it stops the profiler. The standard `/debug/pprof/*` handlers are also mounted, gated behind the `manage_settings` permission rather than left open.
- **Logs**: the app ships nothing itself. With `LOG_FORMAT=json` it writes structured JSON to stdout and Grafana Alloy tails the container from the Docker daemon API into Loki, which is how every other container on the box is collected too. `logger.Ctx(ctx)` stamps `trace_id` and `span_id` on a log event, Alloy lifts both into Loki structured metadata, and the derived field on the Loki datasource jumps to the matching Tempo trace. The **log level** setting still hot-reloads and is the runtime volume lever.
- **Health**: `/livez` is a bare liveness probe, which is what the container healthcheck hits. `/health` runs real dependency checks and returns 503 when one fails: Postgres is a hard dependency, LiveKit is checked only when voice is configured and is allowed to fail without failing the whole probe.

```
   scraper ──▶ GET /metrics ──▶ http_* , db_pool_* , ws_* , cache_* , chatbot_*
   probe   ──▶ GET /livez   ──▶ 200 always (process is up)
   probe   ──▶ GET /health  ──▶ 200 / 503 (postgres hard, livekit soft)

   otlp_endpoint  ──▶ batch span processor ──▶ tempo
   pyroscope_url  ──▶ continuous profiler  ──▶ pyroscope
   LOG_FORMAT=json ─▶ stdout ──▶ alloy (docker api) ──▶ loki
```

## 7. Technology choices

The README lists the stack. This section holds the reasons, because a version number is a fact that ages on its own, while a reason is a decision somebody has to revisit deliberately. Everything below was checked against `go.mod` and `frontend/package.json`.

### 7.1 Backend

- **Go 1.27** (`go.mod:3`). The generic methods in `internal/cache/typed.go` (`m.Get[T]`, `m.Load[T]`) are a 1.27 feature and do not compile on an earlier toolchain, so the version floor is load-bearing rather than aspirational.
- **Fiber v3** (`gofiber/fiber/v3 v3.5.0`) as the HTTP router. Routes are registered through the `FSetupRoute` indirection described in section 3.2 rather than against the app directly, so the router is reachable from one file if it ever has to be swapped.
- **PostgreSQL via `jackc/pgx/v5`** (`v5.10.0`), used through the `pgx/v5/stdlib` adapter rather than the native pgx interface. That is the one dependency choice worth stating outright: going through `database/sql` keeps `*sql.DB` and `*sql.Tx` as the currency of the whole data layer, which is what lets `txOrDB` and `db.WithTx` be seven lines each, and it keeps `XSAM/otelsql` (`v0.43.0`) able to wrap the driver for a span per statement. The native interface would be marginally faster and would cost both.
- **Goose** (`pressly/goose/v3 v3.27.3`) for migrations, run in-process at boot from `internal/db/migrations`. Migration files are always created through the goose CLI, never by hand.
- **testcontainers-go** (`v0.44.0`, plus the `modules/postgres` helper) for the DAO tests of section 3.5. Running the bottom layer against a real `postgres:18` is what licenses the layers above to be tested entirely against mocks: the SQL is checked once, for real, rather than asserted about in a mock expectation that can only ever restate what the test already assumed.
- **`gofiber/contrib/v3/websocket`** (`v1.2.3`) for the hub in `internal/ws`. `fasthttp/websocket` (`v1.5.12`) is a direct dependency but a client-side one, used only by tests that dial a socket (`internal/overlay/handler_test.go`).
- **zerolog** (`v1.35.1`) for structured logging, with `logger.Ctx(ctx)` stamping `trace_id` and `span_id` so a log line joins a trace (section 6.12).
- **`wneessen/go-mail`** (`v0.8.1`) for SMTP delivery, **`disintegration/imaging`** (`v1.6.2`) and `golang.org/x/image` for server-side image work, and **`rwcarlsen/goexif`** for orientation.
- **`microcosm-cc/bluemonday`** (`v1.0.27`) sanitises the two places user HTML reaches the page: fanfic chapter bodies (`internal/fanfic/sanitize.go`) and display names (`internal/user/display_name.go`).
- **`openai/openai-go/v3`** (`v3.52.0`, Responses API) for the chatbot character accounts.
- **`valkey-io/valkey-go`** (`v1.0.77`) with `valkeyhook` for the tracing and metrics hook, behind the engine interface of section 3.7 so the app is correct with no Valkey at all.
- **`prometheus/client_golang`** (`v1.24.1`) for the `/metrics` registry and the custom collectors, the **OpenTelemetry Go SDK** (`v1.46.0`) with the OTLP/HTTP trace exporter, **`grafana/pyroscope-go`** (`v1.4.2`) for continuous profiling, and **`hellofresh/health-go/v5`** (`v5.5.5`) for the `/health` dependency checks. All four are configured from admin settings at runtime, which is why they are dependencies rather than deployment concerns.
- **`firebase.google.com/go/v4`** (`v4.21.0`) for native FCM push to the packaged mobile app, and **`livekit/server-sdk-go/v2`** (`v2.18.1`, with `livekit/protocol` and `twitchtv/twirp` for the control plane) for the SFU, ingress and egress.
- **`corentings/chess/v2`** (`v2.6.0`) is the server-side move validator for the chess game room (`internal/game/chess/handler.go`), so game legality is decided by the server rather than trusted from the client.
- **Mockery v3** (`v3.7.4`) and **staticcheck** are declared in the `go.mod` `tool` block, so both run from the pinned module version with no separately installed binary. Mocks are always generated from `.mockery.yml`; a hand-rolled fake is not accepted.

### 7.2 Frontend

- **React 19** (`react ^19.2.8`) with **TypeScript 6** (`typescript` aliased to `@typescript/typescript6@^6.0.2`, with `@typescript/native` alongside it), built by **Vite 8** (`^8.2.1`). Type checking is not a separate CI step: `npm run build` is `tsc -b && vite build`, so a type error fails the build job.
- **React Router v8** (`react-router ^8.3.0`). The package is `react-router`, not `react-router-dom`, which was folded in from v7 onwards; importing from `react-router-dom` is the mistake this line exists to prevent.
- **TanStack Query v5** (`@tanstack/react-query ^5.101.4`) is the server-state layer, and invalidating a query on a WebSocket event is what replaces polling (section 6.7). Keys are meant to come from `src/api/queryKeys.ts`; a handful of pages still build theirs inline, which Phase 1 of `design/FRONTEND_REARCHITECTURE.md` closes before a lint rule can enforce it.
- **CSS Modules**, no CSS framework and no runtime styling library. Colours must come from theme tokens rather than literals, because fourteen themes redefine them (`frontend/src/context/ThemeContext.tsx:16-30`).
- **DOMPurify** (`^3.4.13`) and **marked** (`^18.0.9`) for markdown, with **highlight.js** (`^11.12.0`) for code blocks. Sanitising happens on the client for rendering and on the server for storage; neither is trusted alone.
- **TipTap 3** (`^3.29.2`, with StarterKit, Placeholder, TextAlign, Color, TextStyle and Link) for the fanfiction rich-text editor, the one place the site needs formatted input rather than markdown.
- **livekit-client** and **`@livekit/components-react`** for voice, screen share and streaming, **hls.js** (`^1.6.16`) for stream playback, and **`@hyperbeam/web`** for virtual-browser watch parties.
- **`chess.js`** and **react-chessboard** for the chess board. `chess.js` computes legal-move highlights and rejects an illegal drag before it is sent (`frontend/src/components/games/chess/ChessBoardView.tsx:210,243`); the server's own validator above remains the authority, and the client copy exists only so the board feels immediate.
- **emoji-picker-react** for chat reactions, **`@marsidev/react-turnstile`** for bot protection, and **firebase** for web push.
- **Capacitor 8** (`@capacitor/*`) with **`@capgo/capacitor-updater`** packages the same SPA as the mobile app, using bearer-token auth and native push. There is no second frontend.
- **Vitest 4** with Testing Library and jsdom, **oxlint** run as `oxlint --max-warnings=0 .`, and **oxfmt** run as `oxfmt --check ./src`. CI runs oxfmt, then oxlint, then the tests, then the build (`.github/workflows/ci.yml:39-49`), and a warning fails the job exactly as an error does. Both replaced ESLint 10 and Prettier 3; the rule set was carried across one for one, and the only two rules with no oxlint equivalent are `no-octal` and `no-dupe-args`, which tsc and the parser already catch.

### 7.3 Infrastructure

- **Docker multi-stage build**: `node:lts-alpine` for the frontend, `golang:1.27-alpine` for the binary, `alpine:latest` for the runtime, with `ffmpeg` and `libwebp-tools` installed in the final image because the media pipeline of section 6.9 shells out to both.
- **Two Valkey instances** in the compose files, deliberately separate: `valkey` coordinates LiveKit ingress and egress, and `valkey-cache` is the optional app cache, capped with `--maxmemory 4gb --maxmemory-policy allkeys-lru` and configured with no persistence, because everything in it is reconstructible.
- **Designed to sit behind Caddy or another reverse proxy** in production. The app resolves the client IP from `CF-Connecting-IP` where present, which is what the credential rate limiter keys on.
- **`docker-compose.prod.yml` carries `prometheus-*` labels** for label-based scrape discovery and runs a `postgres-exporter` sidecar, so metrics collection needs no application change.

### 7.4 External services

None of these is required to boot the server, and the feature each one powers is hidden or degrades when it is unconfigured or unreachable. The quote API is the only one that needs no credential: it is a public endpoint, called with a ten-second timeout and a hardcoded default base URL (`internal/quotefinder/client.go:49-59`), so a failure there costs a quote picker rather than a page.

- The [Umineko Quote Finder API](https://quotes.auaurora.moe/swagger/index.html) for game quote search and evidence attachment.
- The GIPHY API for GIF search, trending and favourites.
- [Hyperbeam](https://hyperbeam.com/) for the shared-browser VM behind virtual-browser watch parties.
- [LiveKit](https://livekit.io/), self-hosted, as the SFU carrying voice in chat rooms, DMs and watch parties, plus screen-share watch parties and live streaming.
- OpenAI for the chatbot character accounts, keyed from the admin panel.
- Firebase Cloud Messaging for native push to the packaged mobile app.
- Entirely external and entirely optional: a Prometheus scraper, an OTLP trace collector, a Pyroscope server, and a Loki instance with a Grafana Alloy collector. None are bundled in the compose files, and the app ships nothing to any of them by default.

## 8. Testing strategy per layer

The open question this section was held for, whether it should describe today's layout or the post-refactor target, closed when the refactor landed: they are now the same tree. What follows is the frontend half, written against it. The backend half is section 3, where each layer's test style is stated with the layer.

A test belongs to the layer its subject sits in, and it may reach exactly one layer down. That is not a convention: the layer globs in `frontend/oxlint.layers.ts` match `*.test.ts` and `*.test.tsx` as well as source, so a test file inherits its directory's import rules, and three `layers/*` rules police the test-only habits. The two holdout lists in 4.5 are the only exceptions, and both are named files rather than globs.

| Layer         | What its test does                                                                    | What it may not do                                                            |
|---------------|---------------------------------------------------------------------------------------|-------------------------------------------------------------------------------|
| pure          | calls the function and compares values: no renderer, no providers, no mocks           | import `@testing-library/react`, React or `src/api`                           |
| api endpoints | drives `endpoints/testHarness.ts` and asserts each call's method, path and body       | reach the network, outside the four `MAY_STUB_GLOBAL_FETCH` files             |
| data hooks    | `renderHook` over one mocked endpoint module, asserting the key and the invalidation  | restate transport shape, which the endpoint test already owns                 |
| orchestration | `renderHook` over the real hook with its data hooks mocked                            | render the component that consumes it                                         |
| render        | `renderWithProviders` and Testing Library queries, with the data hook mocked          | `vi.mock` `api/endpoints`, `api/queryKeys`, `api/client` or `api/queryClient` |
| platform      | stubs the device API, re-importing under `vi.resetModules()` where module scope reads | import `src/api`                                                              |

`endpoints/testHarness.ts` exports eight typed transport mocks, one per `api/client` primitive, plus `runRequestCases` and the `beforeEach` reset. `vi.mock` is hoisted per file and does not travel through an import, so the `vi.mock` calls stay literally in each of the 27 endpoint test files and only the type, the handles and the runner come from the harness. Twenty-six of them mock `../client` and `../../platform/capabilities`, `auth.test.ts` adds `../authToken` as a third, and `quote.test.ts` mocks nothing because the quote family never touches the app transport. Seven of the ten `src/platform` tests use the `vi.resetModules()` re-import, because their subject reads the environment at module scope.

Shared machinery lives in `src/test-utils`: `render.tsx` for `renderWithProviders`, `providerWrapper` and `createTestQueryClient`, `query.ts` for cache assertions such as `expectInvalidated`, `ws.ts` for the `FakeWebSocket` and `emitRealtimeEvent`, and `fixtures/` for the entity factories. A fixture whose type is part of a hook's published contract lives next to that hook as `*.fixture.ts` instead, so `tsc -b` fails when the contract changes; `useRoomController.fixture.ts` and `useDmController.fixture.ts` are the two.

There are no coverage thresholds and no `autoUpdate` in `vitest.config.ts`, deliberately. A threshold that ratchets itself teaches people to delete tests that lower it. What guards the suite instead is `npm run test:names`, which prints every test name sorted: a refactor that claims to move tests rather than drop them proves it with a name-set diff, and a phase that deliberately deletes a test says which names went and why.

## 9. File map

The backend half first, then the frontend. Section 4 gives the frontend directories their layer and their rules; this is the same tree read as a map.

```
  main.go               process entry and graceful shutdown
  server.go             fiber app, middleware, routes, the embedded SPA, the cleanup func
  init_db.go            telemetry, database, migrations, seed, repositories, settings
  init_services.go      every service, in dependency order
  init_jobs.go          settings listeners and the background job tickers
  pprof.go              the pprof handlers, gated on manage_settings

  internal/controllers  the HTTP surface, one file per domain, each with its FSetupRoute list
  internal/routes       mounts GetAPIRoutes under /api/v1 and GetPageRoutes at the app root
  internal/middleware   the global chain of section 3.8
  internal/repository   interfaces, row models, sentinel errors, composites, the search registry
  internal/dao          every SQL statement, plus daotest for the container-backed tests
  internal/store        the single wiring point, store.New(db, cache)
  internal/db           connection, migrations, seed, WithTx, dbtest
  internal/dto          the wire types, mirrored by hand in frontend/src/types/api.ts
  internal/cache        the manager, its engines, and the namespace and TTL table
  internal/ws           the hub, its clients, and the broadcast helpers
  internal/game         one package per game, holding that game's rules and state
  internal/og           the meta resolver and the OG image service
  internal/media        encoding, thumbnails and the processor queue, with internal/upload
  internal/<domain>     one package per domain service: theory, mystery, art, ship, oc, post,
                        fanfic, journal, chat, chatbot, secret, gameroom, stream, profile,
                        follow, block, report, search, announcement and the rest
```

The frontend, rooted at `frontend/`. Every directory below `src/` carries the layer section 4 gives it, and `frontend/oxlint.layers.ts` is where that layer is spelled as a glob.

```
  index.html            the SPA shell vite builds; the built output goes to ../static
  vite.config.ts        build, dev proxy and the ../static outDir
  vitest.config.ts      jsdom, the setup file, and the coverage include and exclude lists
  oxlint.layers.ts      the layer globs, the import and syntax rules, and the named file lists
  oxlint.config.ts      the base config, which spreads layerRules last
  oxlint-plugin-layers.mjs  the five esquery rules the layer blocks switch on and off
  .oxfmtrc.json         the formatter settings, migrated from .prettierrc
  capacitor.config.ts   the Android wrapper's app id, web dir, plugins and dev server override
  scripts/              build-time helpers: the OTA bundle, test:names, the local Capacitor run
  android/              the generated Capacitor project, kept in the repository

  src/main.tsx          composition root: the providers, createRoot, and the error handlers
                        import that has to evaluate first
  src/App.tsx           React composition root: BrowserRouter, the route table, the app chrome
  src/pages             render layer, one directory per surface; 100 of its 113 components
                        have a test beside them
  src/components        render layer, shared and feature components with their CSS modules
  src/hooks             orchestration layer: the feature hooks and controllers
  src/hooks/queries     data hooks, read side: one file per domain, binding a key to an endpoint
  src/hooks/mutations   data hooks, write side, including the invalidations each one owns
  src/hooks/chat        the chat controllers' own sub-hooks
  src/context           orchestration layer: six providers, each split into a value module
  src/domain            pure layer: reducers, selectors, permission rules, parsers, per domain
  src/utils             pure layer: seven app-agnostic helpers, no domain vocabulary
  src/games             per-game rules and view models, with each game's hooks under */hooks
  src/types             api.ts mirrors internal/dto by hand; app.ts holds view-only types
  src/api               server adapter: client, queryKeys, queryClient, authToken, telemetry
  src/api/endpoints     29 per-domain modules, one function per route, plus testHarness.ts
  src/api/realtime      the socket, the typed event contract, the bus, and sync/ cache patches
  src/api/cache         the cache writers a patch reaches for, such as patchUser
  src/api/beacons       the unload-time keepalive posts api/client cannot express
  src/api/livekit       the LiveKit room and track plumbing; hyperbeam/ is the watch-party frame
  src/platform          device adapter: capabilities, sound, push, OTA, popouts, last location
  src/styles            the global stylesheet and the theme token definitions
  src/test-utils        render and query harnesses, the websocket fake, and fixtures/
```
