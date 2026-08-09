# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

A self-hosted, single-admin audio file host and player: upload an audio
file, get a shareable link, recipients stream it in-browser with a minimal
progress-line player. Go backend, SQLite for metadata, server-rendered
`html/template` + minimal vanilla JS — no build pipeline, one binary. There
is no user table; `auth` implements exactly one operator via a bcrypt
password check against `ADMIN_PASSWORD_HASH` and an HMAC-signed session
cookie (no server-side session store).

## Commands

```sh
go build ./... && go vet ./... && go test ./...   # what CI runs
go test ./internal/store/...                       # single package
go test ./internal/auth/ -run TestCheckPassword     # single test
```

Running the server locally needs real Cloudflare R2 + Queue credentials —
see "Local mode" in README.md for the `dev` Tofu workspace + `.env.local`
setup. There's no offline/mock mode; `config.Load()` hard-fails if any
required env var is missing.

## Architecture

**Ingestion has two independent producers, one consumer.** Audio arrives
either via a browser's presigned PUT (`POST /admin/upload/request` →
`r2.PresignPut`) or via `rclone` dropping a file straight into the bucket.
Both converge on `internal/ingest.Poller`, which polls Cloudflare Queue
(R2 object-create event notifications, HTTP pull-consumer API in
`queue_client.go`) every `PollInterval` and calls `ingestObject`:
- if a `store.Track` row already exists for that object key (the browser-PUT
  path, created as `pending` by `CreatePending` before the PUT even starts),
  it's updated in place;
- otherwise (the rclone path) a new row is created on the fly via
  `CreateFromDiscovery`, slug = random ID, title = filename.

Either way, `ingestObject` does an R2 `Head()` to confirm the object exists
and grab content-type/size, then `store.MarkReady`.

**Share links never expose real R2 URLs at rest.** `handleShare` mints a
fresh presigned GET URL (`r2.PresignGet`) on every page load, so link expiry
is enforced by R2 itself (the presign TTL), not just by hiding a page.
`Track.Expired()` is an additional application-level check against
`expires_at` shown before even generating that URL.

**Templates**: `internal/web/templates.go` embeds `templates/*.html` and
`static/` via `go:embed`. Every page template is parsed together with
`layout.html` under a `pageNames` list — adding a new page means adding it
to both `pageNames` and the mux in `server.go`. Because static assets and
templates are embedded, **any change under `internal/web/templates/` or
`internal/web/static/` changes the compiled binary**, not just "frontend
files" — this matters for the deploy workflow's path filter (see below).

**Routing**: `net/http.ServeMux` (Go 1.22+ pattern syntax) in
`server.go:Mux()`. The exact-path root pattern is `GET /{$}`, not `GET /` —
plain `/` is a subtree pattern and conflicts with `/static/`.

**DB**: schema changes go through goose migrations in
`internal/db/migrations/*.sql`, embedded via `go:embed` and applied
automatically by `db.Open()` on every process start — there is no separate
`goose` CLI step and none is needed on the VPS, consistent with the
single-binary deploy model. `provider.Close()` must never be called on the
goose `Provider` built in `db.migrate()`: it closes the underlying
`*sql.DB`, which here is the app's long-lived connection, not something
scoped to migrations alone. `00001_baseline.sql` is the pre-goose
`CREATE TABLE IF NOT EXISTS` schema (safe no-op against the already-existing
prod table); later migrations add/drop columns from there.

## Deployment

Docker Compose on the VPS — see `docs/adr/0002-docker-container-deployment.md`
for why (portability; the prior bare-binary/systemd model wasn't broken).
The final image is `FROM scratch`: `modernc.org/sqlite` is a pure-Go driver,
so `CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build` needs no libc, and the
image needs no shell. The container runs as non-root (`65532:65532`) with a
read-only rootfs; the SQLite file is the one exception, bind-mounted from
`/var/lib/transients` on the host, which must be pre-`chown`ed to
`65532:65532` or the container can't write its DB file. `cmd/healthcheck`
is a tiny static Go binary that GETs `/healthz` — it exists only because
`scratch` has no `curl`/`wget` for the Dockerfile's `HEALTHCHECK` to exec.
No registry: `.github/workflows/deploy.yml` builds the image in CI,
`docker save`s it to a tarball, ships it over `scp`, and `docker load`s +
`docker compose up -d`s it on the VPS — `docker-compose.yml` and `.env`
live in `~/transients` there (the deploy user's home dir, not `/opt`, so
setup needs no `sudo` beyond the one-time chown on the bind-mounted DB
directory). Caddy is untouched by any of this: it
still runs directly on the host and reverse-proxies to `PORT`, since it
also fronts other sites on the same VPS.

## Conventions

- No comments explaining *what* code does; only for non-obvious *why*
  (see the `Track.Expired()` / presign-TTL note above for the kind of thing
  worth a comment).
- Package doc comments describe the package's role in the pipeline, not
  just its contents (see the top of `ingest`, `r2`, `auth`, `web`).
- `docs/adr/` holds hard-to-reverse, non-obvious frontend/infra decisions
  (currently just the no-CDN-assets ADR) — check there before assuming a
  design choice (e.g. dark-only theme, system fonts only) was an oversight.
