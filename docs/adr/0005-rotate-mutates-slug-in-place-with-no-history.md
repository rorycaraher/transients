# Rotating a track's link mutates its slug in place, with no history kept

**Status**: accepted

Rotating a track's share link — generating a new URL that breaks anyone
holding the old one, without touching the file or resetting `expires_at` —
could either overwrite the `slug` column in place on the existing row, or
add a surrogate id plus a slug-history/alias table so the old link could
still be recognized (redirected, or shown a distinct "this link was
rotated" page) instead of going fully dark. We chose **in-place mutation**:
`RotateSlug` does a plain `UPDATE tracks SET slug = ?` (see
`internal/store/store.go`), and the old slug is never recorded anywhere.
Once rotated, `/t/{oldSlug}` and `/admin/tracks/{oldSlug}/...` 404 through
the exact same `GetBySlug` → `ErrNotFound` path as a slug that never
existed (`lookupReadyTrack` in `internal/web/share.go`).

This works cleanly here because `slug` is the table's actual primary key,
not a display alias over some other id — every route, `ShareURL`/
`EmbedURL`/`OEmbedURL` construction, and the R2 object key are already
built from whatever the current `slug` value is, and none of them are
persisted anywhere that would need a second update. The alternative — a
history table so old links could redirect or show a dedicated "rotated"
message — would also mean deliberately telling a visitor with an old link
that it *used* to be valid, a minor information leak a flat 404 avoids for
free. Nothing else in this single-admin, no-versioning app keeps a trail of
superseded state (`handleDelete` and file-replace, ADR 0003, are equally
final and silent about what came before), so keeping no record of the old
slug matches the rest of the admin surface rather than being a special
case.

Reversing this later — supporting old-link redirects or a "rotated" page —
would mean adding that history table and a lookup fallback in
`lookupReadyTrack`, neither of which exists today.
