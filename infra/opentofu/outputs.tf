output "r2_bucket_name" {
  value = cloudflare_r2_bucket.audio.name
}

output "queue_id" {
  value = cloudflare_queue.ingest.id
}

output "r2_access_key_id" {
  value     = cloudflare_account_token.app.id
  sensitive = true
}

output "r2_secret_access_key" {
  value     = sha256(cloudflare_account_token.app.value)
  sensitive = true
}

output "cf_api_token" {
  description = "Bearer token for the Queues pull-consumer API (CF_API_TOKEN in .env)."
  value       = cloudflare_account_token.app.value
  sensitive   = true
}

output "app_url" {
  value = local.is_prod ? "https://${var.subdomain}" : "http://localhost:8080"
}
