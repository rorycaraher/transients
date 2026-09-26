package main

import rego.v1

# Renaming `bucket_name` forces replacement, and replacing a bucket deletes
# every uploaded track.
deny contains msg if {
	some rc in input.resource_changes
	rc.type == "cloudflare_r2_bucket"
	"delete" in rc.change.actions
	not exempt("bucket_protection", rc.address)
	msg := sprintf("%s: would be destroyed or replaced, which deletes every object in the bucket", [rc.address])
}
