terraform {
  required_providers {
    cloudflare = {
      source  = "cloudflare/cloudflare"
      version = "~> 5"
    }
  }
}

# Bootstrap credential: needs enough privilege to create the R2 bucket,
# queue, DNS record, and the scoped app token below. Set via
# CLOUDFLARE_API_TOKEN env var — do not hardcode it here.
provider "cloudflare" {}
