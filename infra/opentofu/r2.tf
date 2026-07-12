resource "cloudflare_r2_bucket" "audio" {
  account_id = var.cloudflare_account_id
  name       = "${var.bucket_name}${local.name_suffix}"
  location   = "enam" # adjust to wherever the VPS/listeners actually are
}
