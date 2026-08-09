# Docker container deployment instead of bare binary + systemd

**Status**: accepted

The deploy model up to now (bare static binary + systemd, see `CLAUDE.md`)
was itself a deliberate choice: `modernc.org/sqlite` is pure Go specifically
so the binary cross-compiles with `CGO_ENABLED=0` and needs no toolchain on
the VPS. We're moving to Docker anyway, purely for portability — keeping
the option to move off this specific VPS later, and to let other people
self-host the app — not because the old model had an environment-drift
problem to fix.

Containerizing does not remove the app's dependency on Cloudflare R2 and
Queue as external managed services; a self-hoster still needs their own
Cloudflare account and credentials, same as this deployment does.

## Decisions folded into this

- SQLite persists via a **host bind mount** (`/var/lib/transients`), not a
  named volume, so it stays a plain file on disk — backups, inspection, and
  migrating off Docker later are unaffected.
- **Caddy stays on the host**, unchanged, reverse-proxying to a published
  container port. It already fronts unrelated sites on this VPS; folding it
  into the container setup would drag those sites into scope for no reason.
- **No registry.** Dockerfile-only, build-it-yourself. A registry solves
  "other people run this regularly and want easy updates," which isn't a
  problem yet on a single-admin project — cheap to add later.
- Final image is **`scratch`**-based: the binary is already `CGO_ENABLED=0`
  pure Go, so nothing needs a shell or libc. Runs as **non-root** with
  **`read_only: true`**, carrying forward the old systemd unit's
  `NoNewPrivileges`/`ProtectSystem=strict` hardening into the container
  boundary instead of dropping it.
- `deploy/transients.service` is deleted. Docker's `restart: unless-stopped`
  fully replaces it as supervisor — running both would just be two
  disagreeing sources of truth for "is it up."
- CI still builds the image in GitHub Actions (no on-VPS build step) and
  ships it via `docker save` + `scp` + `docker load`, mirroring the old
  build-in-CI/ship-artifact shape rather than introducing a registry.
