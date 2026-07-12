variable "cloudflare_account_id" {
  description = "Cloudflare account ID (found in the dashboard sidebar)."
  type        = string
}

variable "cloudflare_zone_id" {
  description = "Zone ID for the domain the app's subdomain will live under."
  type        = string
}

variable "subdomain" {
  description = "Full hostname the app will be served on, e.g. share.yourdomain.com. Unused outside the default (production) workspace."
  type        = string
  default     = ""
}

variable "vps_ipv4" {
  description = "Public IPv4 address of the existing Hetzner VPS running Caddy. Unused outside the default (production) workspace."
  type        = string
  default     = ""
}

variable "bucket_name" {
  description = "Name of the R2 bucket that stores audio files."
  type        = string
  default     = "transients-audio"
}
