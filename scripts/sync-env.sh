#!/usr/bin/env bash
# Pushes the Tofu-derived credentials into the VPS .env and recreates the
# container. Reads state (read-only `tofu output`); never plans or applies.
# Usage: VPS_SSH=user@host scripts/sync-env.sh
set -euo pipefail

: "${VPS_SSH:?set VPS_SSH to the ssh target, e.g. deploy@vps.example.com}"

tf=(tofu -chdir="$(dirname "$0")/../infra/opentofu")

# Non-default workspaces hold dev tokens; they must never reach production.
workspace=$("${tf[@]}" workspace show)
if [ "$workspace" != "default" ]; then
  echo "refusing: tofu workspace is '$workspace', production is 'default'" >&2
  exit 1
fi

out() {
  local v
  v=$("${tf[@]}" output -raw "$1")
  # Keeps values safe to append unquoted to an env_file and to pipe over ssh.
  if ! [[ $v =~ ^[A-Za-z0-9_.-]+$ ]]; then
    echo "refusing: output '$1' is empty or has unexpected characters" >&2
    exit 1
  fi
  printf '%s' "$v"
}

lines=$(printf '%s=%s\n' \
  R2_BUCKET "$(out r2_bucket_name)" \
  R2_ACCESS_KEY_ID "$(out r2_access_key_id)" \
  R2_SECRET_ACCESS_KEY "$(out r2_secret_access_key)" \
  CF_API_TOKEN "$(out cf_api_token)" \
  CF_QUEUE_ID "$(out queue_id)")

# Secrets travel on ssh stdin, not argv, so they stay out of process lists.
printf '%s\n' "$lines" | ssh "$VPS_SSH" '
  set -e
  cd ~/transients
  umask 077
  trap "rm -f .env.new .env.new.keys" EXIT
  cat > .env.new.keys
  awk -F= "NR==FNR{k[\$1]=1;next} !(\$1 in k)" .env.new.keys .env > .env.new
  cat .env.new.keys >> .env.new
  cp .env .env.bak
  mv .env.new .env
'

ssh "$VPS_SSH" 'cd ~/transients && docker compose up -d --force-recreate'

echo "waiting for a poll cycle..."
sleep 15
logs=$(ssh "$VPS_SSH" 'cd ~/transients && docker compose logs --since 20s')
if grep -q "ingest poll failed" <<<"$logs"; then
  echo "ingest poll still failing; check: docker compose logs --tail=20" >&2
  exit 1
fi
echo "ok: no poll errors"
