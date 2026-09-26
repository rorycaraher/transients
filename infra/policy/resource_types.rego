package main

import rego.v1

# Guardrail against the stack quietly growing new kinds of infrastructure
# (a public bucket domain, an account member, zone settings). Adding a
# type here is the deliberate act.
allowed_resource_types := {
	"cloudflare_account_token",
	"cloudflare_dns_record",
	"cloudflare_queue",
	"cloudflare_queue_consumer",
	"cloudflare_r2_bucket",
	"cloudflare_r2_bucket_cors",
	"cloudflare_r2_bucket_event_notification",
}

deny contains msg if {
	some rc in input.resource_changes
	rc.mode == "managed"
	rc.change.actions != ["delete"]
	not rc.type in allowed_resource_types
	not exempt("resource_types", rc.address)
	msg := sprintf("%s: resource type %s is not in the allowlist (infra/policy/resource_types.rego)", [rc.address, rc.type])
}
