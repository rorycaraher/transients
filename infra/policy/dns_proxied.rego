package main

import rego.v1

# Unproxied records would expose the VPS's IP directly instead of
# fronting it through Cloudflare.
deny contains msg if {
	some rc in input.resource_changes
	rc.type == "cloudflare_dns_record"
	after := rc.change.after
	after != null
	object.get(after, "proxied", false) != true
	not exempt("dns_proxied", rc.address)
	msg := sprintf("%s: must be proxied", [rc.address])
}
