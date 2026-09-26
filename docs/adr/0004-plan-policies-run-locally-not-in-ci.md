# Plan-based policies run locally, not in CI

**Status**: accepted

The infra guardrails are split by what they need. TFLint, Checkov, and
`tofu validate` work on source, so they run in CI (`checks.yml`) and block
PRs. The Conftest policies in `infra/policy/` run against `tofu show -json`
output of a saved plan, so they run only on the admin's machine
(`mise run plan-check`), between `tofu plan -out=tfplan` and
`tofu apply tfplan`. CI only runs their Rego unit tests
(`conftest verify`), so a policy that can never fire is still caught there.

The alternative is running Conftest against the HCL source in CI, which is
enforced automatically but sees unevaluated expressions — `count`, workspace-
dependent names, and the resolved CORS origin are exactly what the policies
care about, and they are only concrete in a plan. A plan in CI would need the
Cloudflare token (a data source in `tokens.tf` is read at plan time) plus a
backend override, and would show an everything-is-a-create plan against empty
state, which says nothing about real infrastructure. It would also move the
plan out of the hands of the human who reads every plan before apply.

The cost is that the plan gate is opt-in: nothing stops `tofu apply` without
`plan-check`. Reversing this means either giving CI real state and credentials
or writing a second, HCL-shaped set of policies.
