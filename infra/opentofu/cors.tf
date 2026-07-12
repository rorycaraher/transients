locals {
  # The only cross-origin request R2 ever sees is the browser's direct PUT
  # for uploads (internal/web/static/upload.js) — playback happens through a
  # plain <audio> element, which doesn't need CORS.
  upload_origin = local.is_prod ? "https://${var.subdomain}" : "http://localhost:8080"
}

resource "cloudflare_r2_bucket_cors" "audio" {
  account_id  = var.cloudflare_account_id
  bucket_name = cloudflare_r2_bucket.audio.name
  rules = [{
    id = "browser-upload"
    allowed = {
      methods = ["PUT"]
      origins = [local.upload_origin]
      headers = ["content-type"]
    }
    max_age_seconds = 3600
  }]
}
