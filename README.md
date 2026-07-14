# transients

A self-hosted, single-admin audio file host and player: upload an audio
file, get a shareable link, recipients stream it in-browser with a minimal
progress-line player.

## How it works

- Go backend, SQLite for metadata, server-rendered templates + minimal JS
  (a plain `<audio>` element driving a custom seek bar) — no build
  pipeline, one binary.
- Audio lives in a private Cloudflare R2 bucket. Files arrive two ways:
  - the browser uploads directly to R2 via a presigned PUT, or
  - you `rclone` files straight into the bucket from the CLI.
- Both paths converge on one ingest pipeline: R2 fires object-create events
  into a Cloudflare Queue, the app polls that queue every ~10s, and marks
  the track ready as soon as the object is confirmed to exist in R2.
- Share pages mint a fresh short-lived presigned GET URL on each load, so
  link expiry is enforced by R2 itself, not just by hiding the page.
- OpenTofu manages the Cloudflare side (R2 bucket, Queue, DNS record, a
  scoped API token). The VPS and Caddy are not Tofu-managed.

## Local development

```sh
go build ./... && go vet ./... && go test ./...
```

Running the server locally still needs real R2 + Cloudflare credentials
(see below) — there's no offline/mock mode.

## Deploying

### 1. Create the Tofu state bucket

OpenTofu's own state has to live somewhere before Tofu can manage anything
else, so this one bucket is created by hand (dashboard, or `rclone mkdir`):

```sh
rclone mkdir r2:transients-tofu-state
```

Edit `infra/opentofu/backend.tf` and replace `<ACCOUNT_ID>` with your real
Cloudflare account ID.

### 2. Provision Cloudflare infra with OpenTofu

This step needs two *different* kinds of bootstrap credentials, used only
for `tofu apply` runs (the running app itself uses a separate, narrowly
scoped token that Tofu creates for you — see step 3):

- **A Cloudflare API token** (bearer token) for the `cloudflare` provider to
  actually create resources. Create via **Dashboard → My Profile → API
  Tokens → Create Token → Custom Token**, with: Account → Workers R2
  Storage → Edit; Account → Queues → Edit; Account → Account API Tokens →
  Edit (needed because `tokens.tf` creates another token on your behalf);
  Zone → DNS → Edit, scoped to just your zone. Avoid the legacy Global API
  Key — it's broader than this needs.
- **An R2 API token** (Access Key ID + Secret Access Key) so the S3-compatible
  backend can read/write the state file. Create via **Dashboard → R2 →
  Manage R2 API tokens → Create API token**, scoped to "Object Read &
  Write" on just the state bucket.

Rather than exporting these into your shell directly (where they'd apply to
every command, not just this project), use
[direnv](https://direnv.net/) to scope them to this directory only:

```sh
cd infra/opentofu
cp .envrc.example .envrc   # then edit .envrc with the real values
direnv allow
```

direnv loads `.envrc` only while your shell is inside `infra/opentofu/`, and
unloads it the moment you `cd` back out — these credentials never touch
your global shell profile or leak into other projects. `.envrc` is
gitignored, so it never gets committed.

Copy `terraform.tfvars.example` to `terraform.tfvars` and fill in the real
values — OpenTofu loads `terraform.tfvars` automatically, so no `-var` flags
are needed on `plan`/`apply`. It's gitignored, so it never gets committed.

```sh
cp terraform.tfvars.example terraform.tfvars   # then edit it

tofu init
tofu apply
```

If `apply` fails on a missing permission-group name in `tokens.tf`, run
`tofu console` and inspect
`data.cloudflare_account_api_token_permission_groups_list.all.result` to see
what your account's R2/Queues permission groups are actually called, then
adjust the names in `tokens.tf`.

When it succeeds, grab the values you'll need:

```sh
tofu output r2_bucket_name
tofu output -raw r2_access_key_id
tofu output -raw r2_secret_access_key
tofu output -raw cf_api_token
tofu output queue_id
```

### 3. Build the env file

Copy `.env.example` to `.env` and fill it in:

- `BASE_URL` — `https://share.yourdomain.com` (the subdomain from step 2)
- `ADMIN_PASSWORD_HASH` — generate with `go run ./cmd/hashpw` (prompts for a
  password, prints the bcrypt hash — nothing is echoed or logged)
- `SESSION_SECRET` — `openssl rand -hex 32`
- `R2_ACCOUNT_ID`, `R2_BUCKET`, `R2_ACCESS_KEY_ID`, `R2_SECRET_ACCESS_KEY`,
  `CF_API_TOKEN`, `CF_QUEUE_ID` — from the `tofu output` values above

You'll copy this to `/etc/transients/env` on the VPS in step 5.

### 4. Wire up Caddy

Add the block from `Caddyfile.snippet` to the Caddyfile already running on
the VPS (don't replace the whole file — this is additive, alongside
whatever else Caddy is already serving), then:

```sh
sudo systemctl reload caddy
```

### 5. Build and install the binary

The app is a single static binary — `modernc.org/sqlite` is a pure-Go SQLite
driver, so cross-compiling from your dev machine needs no C toolchain and no
Go installed on the VPS at all:

```sh
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o transients ./cmd/server
```

Check `uname -m` on the VPS first — use `GOARCH=arm64` instead if it's an ARM
box (e.g. Hetzner's CAX line).

One-time setup on the VPS:

```sh
sudo useradd --system --home /var/lib/transients --create-home --shell /usr/sbin/nologin transients
sudo mkdir -p /etc/transients
```

Copy the binary, the env file from step 3, and the systemd unit over (run
from your dev machine):

```sh
scp transients you@vps:/tmp/transients
scp .env you@vps:/tmp/env
scp deploy/transients.service you@vps:/tmp/transients.service
```

Then on the VPS:

```sh
sudo mv /tmp/transients /usr/local/bin/transients
sudo chmod +x /usr/local/bin/transients

sudo mv /tmp/env /etc/transients/env
sudo chown root:transients /etc/transients/env
sudo chmod 640 /etc/transients/env

sudo mv /tmp/transients.service /etc/systemd/system/transients.service
sudo chown transients:transients /var/lib/transients

sudo systemctl daemon-reload
sudo systemctl enable --now transients
```

The app listens on `PORT` from your `.env` (`8090` in this repo's
template — not exposed publicly on its own). Caddy is what terminates TLS
and reverse-proxies to it, so `Caddyfile.snippet`'s `reverse_proxy` target
must be kept in sync with whatever you set `PORT` to.

To ship new code later: rebuild the binary, `scp` it to `/tmp/transients` on
the VPS, then `sudo mv /tmp/transients /usr/local/bin/transients && sudo
systemctl restart transients`.

### Verify

- Visit `https://share.yourdomain.com/login`, log in with the password you
  hashed in step 3.
- Upload a file from the admin dashboard; it should flip from "pending" to
  "ready" within ~10-15s.
- `rclone copy somefile.mp3 r2:<bucket>/` and confirm it shows up in the
  dashboard on the next poll, with the filename as its title and no expiry.
- Open a share link in an incognito window (no login) and confirm playback.

## Local mode

Run the app directly on `localhost` with `go run`, still talking to real R2
and a real Cloudflare Queue — just a separate `dev` set of resources, never
production's. (Sharing production's bucket/queue would mean two pollers
racing over the same Queue messages, and test uploads polluting your real
dashboard.)

**One-time setup — a `dev` Tofu workspace:**

```sh
cd infra/opentofu
tofu workspace new dev
tofu apply -var-file=dev.tfvars
```

This provisions a second R2 bucket/Queue/token named with a `-dev` suffix
(e.g. `transients-audio-dev`) and skips the DNS record entirely — dev has no
subdomain, since it's not behind Caddy. Grab its outputs the same way as
prod:

```sh
tofu output r2_bucket_name
tofu output -raw r2_access_key_id
tofu output -raw r2_secret_access_key
tofu output -raw cf_api_token
tofu output queue_id
```

Switch back to production any time with `tofu workspace select default`.

**Configure and run:**

```sh
cp .env.local.example .env.local   # then fill in the dev workspace's outputs above
set -a; source .env.local; set +a
go run ./cmd/server
```

`DB_PATH=./local.db` in `.env.local.example` keeps the dev SQLite file
local to your checkout (already covered by `.gitignore`'s `*.db` pattern),
so it can never collide with a production database.
`BASE_URL=http://localhost:8080` is deliberate: modern browsers treat
`localhost` itself as a secure context, so the app's `Secure` session
cookie still gets set and sent correctly with zero code changes — this
only holds for literally `http://localhost`, not a LAN IP or another
hostname.

Re-run `go run ./cmd/server` after code changes; there's no hot-reload
tooling here, by design.
