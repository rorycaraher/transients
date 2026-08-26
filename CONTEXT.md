# Transients

A self-hosted, single-admin audio file host and player: upload an audio file, get a shareable link, recipients stream it in-browser.

## Language

**Share Page**:
The full page at a track's share URL — title, player, download link. The only surface the admin's own UI fully owns, and therefore the only place player behavior (like keyboard shortcuts) can safely assume it's the only thing on the page.
_Avoid_: player page

**Embed**:
The bare, chrome-less player page (`share_embed.html`) meant to be iframed into a third-party page (a tweet, a blog post) via oEmbed/Twitter Card. It shares `player.js` with the Share Page but does not own the surrounding page, so it must not claim keyboard input the host page may need for its own shortcuts.
_Avoid_: embed page, player embed

**Note**:
A free-text, admin-authored annotation on a Track, shown publicly on the Share Page only (never the Embed, which stays chrome-less). Multi-line, unvalidated length, hidden entirely when empty.
_Avoid_: description, comment, liner notes

**Replace**:
Swapping the audio file behind an existing Track while its slug, title, expiry, downloadable flag, and play count all stay unchanged. The previous R2 object is deleted immediately, before the replacement upload is confirmed (see [ADR 0003](./docs/adr/0003-replace-deletes-old-object-before-confirming-new.md)) — the share link 404s (same as any not-yet-ready track) until ingest confirms the new file.
_Avoid_: re-upload, update file

### Playhead movement

Three distinct ways to change the player's position, kept distinct in code and conversation so they aren't conflated:

**Seek**:
Moving to an arbitrary position by dragging or clicking the seek bar. Never changes play/pause state.
_Avoid_: scrub, skip

**Skip**:
A relative jump of a fixed 10 seconds forward or backward from the current position, via the `j`/`l` keyboard shortcuts. Never changes play/pause state.
_Avoid_: seek, nudge

**Jump**:
Moving to an absolute position computed as a fraction of track duration (`n` &#215; 10%), via the `0`&#8211;`9` keyboard shortcuts. Never changes play/pause state.
_Avoid_: seek, chapter
