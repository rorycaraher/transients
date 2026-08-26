# File-replace deletes the old R2 object before the replacement is confirmed

**Status**: accepted

Replacing a track's audio file could delete the old R2 object either
immediately (when the admin submits the replace) or only after the new
object is confirmed `ready` by the ingest poller. We chose **immediate
deletion**, accepting a window — if the browser upload then fails or is
abandoned — where the track is left broken (old file gone, new one never
arrived) until the admin notices and replaces again.

The deferred alternative is safer but requires persisting the old object
key somewhere (the track row's `object_key` column has to be overwritten to
the new key *before* the upload starts, so `ingestObject` routes the
replacement to the existing row instead of minting a new track — see
`internal/ingest/poller.go`), plus poller-side logic to delete the old key
once `MarkReady` fires for the replacement. That's real complexity — a
second cleanup path through the async ingest pipeline — for a failure
window that's small and self-evident: the admin who just clicked replace
will notice immediately if the share link breaks, and nothing else in this
single-admin, no-versioning app treats "undo a destructive action" as a
concern (`handleDelete` is likewise immediate and unrecoverable).

Reversing this later — deferring the delete — would mean adding a
`pending_delete_key`-style column and teaching the poller to clean it up,
so it's worth recording that the simpler behavior was deliberate, not an
oversight.
