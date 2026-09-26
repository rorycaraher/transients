# Plan-check policies (Conftest, run against `tofu show -json` output by
# `mise run plan-check`). Every policy lives in package main so Conftest
# picks up its `deny` rules by default.
package main

import rego.v1

# Exemptions are {policy, address, reason} triples, one per suppressed
# resource. `reason` is mandatory; an empty one is itself a violation.
exemptions := []

exempt(policy, address) if {
	some e in exemptions
	e.policy == policy
	e.address == address
}

deny contains msg if {
	some e in exemptions
	trim_space(e.reason) == ""
	msg := sprintf("exemption for %s on %s has no reason", [e.policy, e.address])
}
