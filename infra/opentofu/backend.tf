# Terraform/OpenTofu backend blocks can't reference variables, so the R2
# state bucket details are hardcoded here. Create the state bucket by hand
# once (dashboard, or `rclone mkdir`/wrangler) before running `tofu init` -
# it must exist before Tofu can use it to store its own state, and it's
# deliberately separate from the audio bucket managed below.
#
# Replace <ACCOUNT_ID> below, then run:
#   tofu init -backend-config="access_key=<state-bucket-access-key>" \
#              -backend-config="secret_key=<state-bucket-secret-key>"
# (keep those two out of this file; pass them at init time or via
# TF_VAR-style env vars AWS_ACCESS_KEY_ID / AWS_SECRET_ACCESS_KEY instead).

terraform {
  backend "s3" {
    bucket                      = "transients-tofu-state"
    key                         = "transients/terraform.tfstate"
    region                      = "auto"
    endpoints                   = { s3 = "https://62aa79ca5a4eb69594dcd5b96f00b4bd.r2.cloudflarestorage.com" }
    skip_credentials_validation = true
    skip_region_validation      = true
    skip_requesting_account_id  = true
    skip_s3_checksum            = true
    use_path_style              = true
  }
}
