resource "cloudflare_queue" "ingest" {
  account_id = var.cloudflare_account_id
  queue_name = "transients-ingest${local.name_suffix}"
}

# The app polls this queue via the HTTP pull-consumer API (internal/ingest) —
# without this resource, pull requests are rejected with "messages cannot be
# pulled unless http_pull mode is enabled".
resource "cloudflare_queue_consumer" "ingest" {
  account_id = var.cloudflare_account_id
  queue_id   = cloudflare_queue.ingest.id
  type       = "http_pull"
}

resource "cloudflare_r2_bucket_event_notification" "on_upload" {
  account_id  = var.cloudflare_account_id
  bucket_name = cloudflare_r2_bucket.audio.name
  queue_id    = cloudflare_queue.ingest.id
  rules = [{
    actions     = ["PutObject", "CopyObject", "CompleteMultipartUpload"]
    description = "Notify the ingest queue whenever an object is created (presigned upload or rclone drop)"
  }]
}
