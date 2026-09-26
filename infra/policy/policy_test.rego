package main

import rego.v1

plan(changes) := {"resource_changes": changes}

change(type, address, actions, after) := {
	"type": type,
	"address": address,
	"mode": "managed",
	"change": {"actions": actions, "after": after},
}

cors_after(methods, origins) := {"rules": [{"id": "browser-upload", "allowed": {"methods": methods, "origins": origins}}]}

# bucket_protection

test_bucket_replace_denied if {
	p := plan([change("cloudflare_r2_bucket", "cloudflare_r2_bucket.audio", ["delete", "create"], {})])
	count(deny) == 1 with input as p
}

test_bucket_destroy_denied if {
	p := plan([change("cloudflare_r2_bucket", "cloudflare_r2_bucket.audio", ["delete"], null)])
	count(deny) == 1 with input as p
}

test_bucket_update_allowed if {
	p := plan([change("cloudflare_r2_bucket", "cloudflare_r2_bucket.audio", ["update"], {})])
	count(deny) == 0 with input as p
}

test_bucket_replace_exempted if {
	p := plan([change("cloudflare_r2_bucket", "cloudflare_r2_bucket.audio", ["delete", "create"], {})])
	ex := [{"policy": "bucket_protection", "address": "cloudflare_r2_bucket.audio", "reason": "migrating regions"}]
	count(deny) == 0 with input as p with exemptions as ex
}

test_exemption_without_reason_denied if {
	ex := [{"policy": "bucket_protection", "address": "cloudflare_r2_bucket.audio", "reason": " "}]
	count(deny) == 1 with input as plan([]) with exemptions as ex
}

# cors

test_cors_upload_only_allowed if {
	p := plan([change("cloudflare_r2_bucket_cors", "cloudflare_r2_bucket_cors.audio", ["create"], cors_after(["PUT"], ["https://share.example.com"]))])
	count(deny) == 0 with input as p
}

test_cors_localhost_http_allowed if {
	p := plan([change("cloudflare_r2_bucket_cors", "cloudflare_r2_bucket_cors.audio", ["create"], cors_after(["PUT"], ["http://localhost:8080"]))])
	count(deny) == 0 with input as p
}

test_cors_wildcard_origin_denied if {
	p := plan([change("cloudflare_r2_bucket_cors", "cloudflare_r2_bucket_cors.audio", ["create"], cors_after(["PUT"], ["*"]))])

	# `*` is also a non-https origin, so it trips both rules.
	count(deny) == 2 with input as p
}

test_cors_extra_method_denied if {
	p := plan([change("cloudflare_r2_bucket_cors", "cloudflare_r2_bucket_cors.audio", ["create"], cors_after(["PUT", "GET"], ["https://share.example.com"]))])
	count(deny) == 1 with input as p
}

test_cors_http_remote_origin_denied if {
	p := plan([change("cloudflare_r2_bucket_cors", "cloudflare_r2_bucket_cors.audio", ["create"], cors_after(["PUT"], ["http://share.example.com"]))])
	count(deny) == 1 with input as p
}

test_cors_lookalike_localhost_denied if {
	p := plan([change("cloudflare_r2_bucket_cors", "cloudflare_r2_bucket_cors.audio", ["create"], cors_after(["PUT"], ["http://localhost.evil.com"]))])
	count(deny) == 1 with input as p
}

test_cors_deleted_ignored if {
	p := plan([change("cloudflare_r2_bucket_cors", "cloudflare_r2_bucket_cors.audio", ["delete"], null)])
	count(deny) == 0 with input as p
}

# dns_proxied

test_dns_proxied_allowed if {
	p := plan([change("cloudflare_dns_record", "cloudflare_dns_record.app[0]", ["create"], {"proxied": true})])
	count(deny) == 0 with input as p
}

test_dns_unproxied_denied if {
	p := plan([change("cloudflare_dns_record", "cloudflare_dns_record.app[0]", ["create"], {"proxied": false})])
	count(deny) == 1 with input as p
}

test_dns_proxied_unset_denied if {
	p := plan([change("cloudflare_dns_record", "cloudflare_dns_record.app[0]", ["create"], {})])
	count(deny) == 1 with input as p
}

test_dns_deleted_ignored if {
	p := plan([change("cloudflare_dns_record", "cloudflare_dns_record.app[0]", ["delete"], null)])
	count(deny) == 0 with input as p
}

# resource_types

test_allowlisted_type_allowed if {
	p := plan([change("cloudflare_queue", "cloudflare_queue.ingest", ["create"], {})])
	count(deny) == 0 with input as p
}

test_unlisted_type_denied if {
	p := plan([change("cloudflare_r2_custom_domain", "cloudflare_r2_custom_domain.public", ["create"], {})])
	count(deny) == 1 with input as p
}

test_unlisted_type_removal_allowed if {
	p := plan([change("cloudflare_r2_custom_domain", "cloudflare_r2_custom_domain.public", ["delete"], null)])
	count(deny) == 0 with input as p
}

test_data_sources_ignored if {
	p := plan([{
		"type": "cloudflare_account_api_token_permission_groups_list",
		"address": "data.cloudflare_account_api_token_permission_groups_list.all",
		"mode": "data",
		"change": {"actions": ["read"], "after": {}},
	}])
	count(deny) == 0 with input as p
}
