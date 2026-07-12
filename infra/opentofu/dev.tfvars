# Applied only in the "dev" workspace: `tofu workspace select dev && tofu
# apply -var-file=dev.tfvars`. cloudflare_account_id/cloudflare_zone_id
# still come from terraform.tfvars (same Cloudflare account for both
# environments) - this file is deliberately near-empty since bucket/queue
# names and the skipped DNS record are already handled automatically based
# on the workspace name (see locals.tf, dns.tf). Add overrides here if a
# dev-specific value is ever needed.
