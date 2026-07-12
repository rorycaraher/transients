data "cloudflare_account_api_token_permission_groups_list" "all" {
  account_id = var.cloudflare_account_id
}

locals {
  r2_bucket_resource = "com.cloudflare.edge.r2.bucket.${var.cloudflare_account_id}_default_${cloudflare_r2_bucket.audio.name}"
  account_resource   = "com.cloudflare.api.account.${var.cloudflare_account_id}"

  # Cloudflare's permission-group catalog reuses the same display name
  # across unrelated resource-type scopes (account vs zone vs user), so a
  # flat name->id map collides. Disambiguate each lookup by name AND the
  # resource-type scope we actually intend to grant it at. `one(...)`
  # errors clearly if this ever matches zero or more than one entry, rather
  # than silently picking the wrong one.
  #
  # If `tofu apply` fails with a "list of length 0" error here, run
  # `tofu console` and inspect:
  #   [for pg in data.cloudflare_account_api_token_permission_groups_list.all.result : { name = pg.name, scopes = pg.scopes } if strcontains(pg.name, "R2") || strcontains(pg.name, "Queue")]
  # to see what your account's R2/Queues permission groups are actually
  # called and scoped to, then adjust the name/scope pairs below.
  permission_id = {
    r2_read = one([
      for pg in data.cloudflare_account_api_token_permission_groups_list.all.result :
      pg.id if pg.name == "Workers R2 Storage Bucket Item Read" && contains(pg.scopes, "com.cloudflare.edge.r2.bucket")
    ])
    r2_write = one([
      for pg in data.cloudflare_account_api_token_permission_groups_list.all.result :
      pg.id if pg.name == "Workers R2 Storage Bucket Item Write" && contains(pg.scopes, "com.cloudflare.edge.r2.bucket")
    ])
    queues_read = one([
      for pg in data.cloudflare_account_api_token_permission_groups_list.all.result :
      pg.id if pg.name == "Queues Read" && contains(pg.scopes, "com.cloudflare.api.account")
    ])
    queues_write = one([
      for pg in data.cloudflare_account_api_token_permission_groups_list.all.result :
      pg.id if pg.name == "Queues Write" && contains(pg.scopes, "com.cloudflare.api.account")
    ])
  }
}

# One token, scoped to exactly what the app needs: read/write on the audio
# bucket, and read/write on Queues (write is required to ack messages, not
# just to consume them). Its `id` doubles as the R2 Access Key ID and
# sha256(value) as the R2 Secret Access Key (see
# https://developers.cloudflare.com/r2/api/tokens/#get-s3-api-credentials-from-an-api-token),
# and `value` itself is the bearer token for the Queues pull-consumer API -
# so this single resource yields all three credentials the app needs.
#
# Uses cloudflare_account_token (POST /accounts/{account_id}/tokens, an
# "account-owned" token) rather than cloudflare_api_token (POST
# /user/tokens, a user-owned token): the latter requires "Account API
# Tokens Edit" granted at *User* scope on the bootstrap token, which is
# easy to get wrong in the dashboard's custom-token UI and ties the token's
# lifecycle to a specific human user. The account-owned variant just needs
# the account-scoped permission the bootstrap token already has.
resource "cloudflare_account_token" "app" {
  account_id = var.cloudflare_account_id
  name       = "transients-app${local.name_suffix}"
  policies = [{
    effect = "allow"
    permission_groups = [
      { id = local.permission_id.r2_read },
      { id = local.permission_id.r2_write },
      { id = local.permission_id.queues_read },
      { id = local.permission_id.queues_write },
    ]
    resources = jsonencode({
      (local.r2_bucket_resource) = "*"
      (local.account_resource)   = "*"
    })
  }]
}
