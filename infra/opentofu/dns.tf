resource "cloudflare_dns_record" "app" {
  count   = local.is_prod ? 1 : 0
  zone_id = var.cloudflare_zone_id
  name    = var.subdomain
  type    = "A"
  content = var.vps_ipv4
  ttl     = 1 # "automatic" when proxied
  proxied = true
  comment = "transients app, reverse-proxied by Caddy on the VPS"
}
