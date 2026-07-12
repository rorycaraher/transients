# The default workspace is production. Any other workspace (e.g. "dev") is
# treated as a separate, non-production environment: its R2 bucket and
# Queue get a name suffix so they never collide with production's, and it
# gets no DNS record (see dns.tf) since it's meant to run on localhost, not
# behind Caddy on a public subdomain.
locals {
  is_prod     = terraform.workspace == "default"
  name_suffix = local.is_prod ? "" : "-${terraform.workspace}"
}
