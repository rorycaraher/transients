plugin "terraform" {
  enabled = true
  preset  = "recommended"
}

# Community ruleset, not official — pinned to an exact release.
plugin "cloudflare" {
  enabled = true
  version = "0.1.0"
  source  = "github.com/alexraskin/tflint-ruleset-cloudflare"
}
