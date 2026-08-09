# transients

A self-hosted, single-admin audio file host and player: upload an audio
file, get a shareable link, recipients stream it in-browser, or optionally
enable download per-file.

## How it works

- Go backend, SQLite for metadata, server-rendered templates + minimal JS — no build
  pipeline, one binary.
- Audio lives in a private Cloudflare R2 bucket. Files arrive two ways: the
  browser uploads directly to R2 via a presigned PUT, or you `rclone` files
  straight into the bucket.
- Both paths converge on one ingest pipeline: R2 fires object-create events
  into a Cloudflare Queue, the app polls it every ~10s, and marks the track
  ready once the object is confirmed to exist in R2.
- Share pages mint a fresh short-lived presigned GET URL on each load, so
  link expiry is enforced by R2 itself, not just by hiding the page.
- Schema changes go through goose migrations (`internal/db/migrations`),
  embedded in the binary and applied automatically on startup — no
  separate migration step.
- OpenTofu (`infra/opentofu`) manages the Cloudflare side: R2 bucket,
  Queue, DNS record, a scoped API token. The VPS and Caddy are not
  Tofu-managed.

## Local development

```sh
go build ./... && go vet ./... && go test ./...
```

There's no offline/mock mode — `go run ./cmd/server` always needs real R2 +
Cloudflare credentials, ideally a separate `dev` Tofu workspace
(`tofu workspace new dev`) rather than production's, so a local poller
never races production's queue. Copy `.env.local.example` to `.env.local`
and fill in that workspace's `tofu output` values.

`BASE_URL=http://localhost:8080` in local dev is deliberate: browsers treat
literal `localhost` as a secure context, so the `Secure` session cookie
still works with zero code changes. `DB_PATH=./local.db` keeps the dev database out of git and
away from production's.

No hot-reload — re-run `go run ./cmd/server` after code changes.

## Environment variables

| Var | Notes |
|---|---|
| `PORT` | defaults to `8080` (prod template here uses `8090`) |
| `BASE_URL` | e.g. `https://share.yourdomain.com` |
| `DB_PATH` | defaults to `/var/lib/transients/app.db` |
| `ADMIN_PASSWORD_HASH` | bcrypt hash — generate with `go run ./cmd/hashpw` |
| `SESSION_SECRET` | `openssl rand -hex 32` |
| `R2_ACCOUNT_ID`, `R2_BUCKET`, `R2_ACCESS_KEY_ID`, `R2_SECRET_ACCESS_KEY` | from `tofu output` in `infra/opentofu` |
| `CF_API_TOKEN`, `CF_QUEUE_ID` | from `tofu output` in `infra/opentofu` |

See `.env.example` (prod, loaded via Docker Compose's `env_file:`) and
`.env.local.example` (local dev, loaded via `source` — note the comment
there about single-quoting the bcrypt hash, since bash expands `$` and
systemd doesn't).

## Deploying

The app runs as a Docker container — see
`docs/adr/0002-docker-container-deployment.md` for why (portability; the
prior bare-binary/systemd setup wasn't broken, this isn't a fix). There's
no image registry yet, so the image is built in CI and shipped as a
tarball rather than pulled.

One-time setup on the VPS:

```sh
sudo mkdir -p /var/lib/transients /opt/transients
sudo chown 65532:65532 /var/lib/transients   # matches the container's non-root UID
```

Copy `docker-compose.yml` and your filled-in `.env` (see `.env.example`)
to `/opt/transients/`. Caddy is unaffected by any of this — it still runs
directly on the host and reverse-proxies to `PORT`, since it also fronts
other sites on the box — see `Caddyfile.snippet`.

`.github/workflows/deploy.yml` automates redeploys on push to `main` (only
when a change actually touches the compiled binary, the Dockerfile, or
`docker-compose.yml` — Go files, templates, and static assets count too,
since those are embedded). It builds the image, `docker save`s it to a
tarball, `scp`s the tarball and `docker-compose.yml` to the VPS, then
`docker load`s and `docker compose up -d`s it there, polling the
container's own `HEALTHCHECK` status before declaring success.

To do the same by hand:

```sh
docker build -t transients:local .
docker save transients:local | gzip > transients.tar.gz
scp transients.tar.gz docker-compose.yml you@vps:/opt/transients/
ssh you@vps 'cd /opt/transients && gunzip -c transients.tar.gz | docker load && docker compose up -d'
```

### Verify

- `curl https://your-domain/healthz` returns `200`.
- `/login` with the password you hashed.
- Upload a file from the dashboard; it should flip pending → ready within
  ~10-15s.
- `rclone copy somefile.mp3 r2:<bucket>/` and confirm it appears on the
  next poll, titled from the filename, no expiry.
- Open a share link in an incognito window (no login) and confirm
  playback.
