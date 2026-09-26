package main

import rego.v1

# The only cross-origin request R2 sees is the browser's presigned PUT
# upload, so anything broader than that is a misconfiguration.
deny contains msg if {
	some rc in input.resource_changes
	rc.type == "cloudflare_r2_bucket_cors"
	not exempt("cors", rc.address)
	some rule in rc.change.after.rules
	some origin in rule.allowed.origins
	origin == "*"
	msg := sprintf("%s: rule %q allows any origin", [rc.address, rule.id])
}

deny contains msg if {
	some rc in input.resource_changes
	rc.type == "cloudflare_r2_bucket_cors"
	not exempt("cors", rc.address)
	some rule in rc.change.after.rules
	some method in rule.allowed.methods
	method != "PUT"
	msg := sprintf("%s: rule %q allows method %s; only PUT is needed for uploads", [rc.address, rule.id, method])
}

deny contains msg if {
	some rc in input.resource_changes
	rc.type == "cloudflare_r2_bucket_cors"
	not exempt("cors", rc.address)
	some rule in rc.change.after.rules
	some origin in rule.allowed.origins
	not secure_origin(origin)
	msg := sprintf("%s: rule %q allows non-https origin %s", [rc.address, rule.id, origin])
}

secure_origin(origin) if startswith(origin, "https://")

secure_origin("http://localhost")

secure_origin(origin) if startswith(origin, "http://localhost:")
