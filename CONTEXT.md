# Transients

A self-hosted, single-admin audio file host and player: upload an audio file, get a shareable link, recipients stream it in-browser.

## Language

**Track**:
The shareable unit of music: a title, slug, expiry, downloadable flag, and play count wrapped around exactly one Audio File at a time. A Track outlives its Audio File across a Replace.
_Avoid_: upload, file, recording, song

**Audio File**:
The actual audio object stored in R2 behind a Track. Uploading (via browser or rclone) is how an Audio File arrives; it is not itself a noun in this domain.
_Avoid_: object, upload, asset

**Available**:
A Track is Available when its Audio File is confirmed and it has not expired. Only Available Tracks are ever played to a listener, whether via the Track's own Share Page or a Playlist; unavailable Tracks in a Playlist are silently passed over.
_Avoid_: live, active, public, playable

**Play**:
A listener starting a Track, counted at most once per Track per page load, whether the Track is heard on its own Share Page, its Embed, or within a Playlist. Plays are counted only against Tracks; Playlists have no play count of their own.
_Avoid_: listen, view, stream

**Playlist**:
An admin-curated, ordered list of references to Tracks, shareable as a whole and played in sequence. A Playlist does not own its Tracks: each Track keeps its own Share Page and may appear in any number of Playlists, but at most once in any given Playlist. A Playlist has a title, a Description, and its own optional expiry independent of its Tracks' expiry; whether a Track is downloadable is decided per Track, never per Playlist. A Playlist's Share Page never reveals its Tracks' own share links, so access gained through a Playlist ends when the Playlist does (see [ADR 0006](./docs/adr/0006-playlists-contain-their-tracks.md)). Deleting a Track removes it from every Playlist; deleting a Playlist never affects its Tracks.
_Avoid_: album, release, collection, set, mix

**Share Page**:
The full page at a share URL, for either a Track (title, player, download link) or a Playlist (player plus a visible, clickable tracklist of its Available Tracks). The only surface the admin's own UI fully owns, and therefore the only place player behavior (like keyboard shortcuts) can safely assume it's the only thing on the page.
_Avoid_: player page

**Embed**:
The bare, chrome-less player page for a single Track (`share_embed.html`) — Playlists have no Embed — meant to be iframed into a third-party page (a tweet, a blog post) via oEmbed/Twitter Card. It shares `player.js` with the Share Page but does not own the surrounding page, so it must not claim keyboard input the host page may need for its own shortcuts.
_Avoid_: embed page, player embed

**Description**:
A free-text, admin-authored annotation on a Track or a Playlist, shown publicly on Share Pages only (never the Embed, which stays chrome-less). A Track's Description is still shown when the Track is heard within a Playlist. Multi-line, unvalidated length, hidden entirely when empty.
_Avoid_: note, comment, liner notes

**Replace**:
Swapping the Audio File behind an existing Track while its slug, title, expiry, downloadable flag, and play count all stay unchanged. The previous R2 object is deleted immediately, before the replacement upload is confirmed (see [ADR 0003](./docs/adr/0003-replace-deletes-old-object-before-confirming-new.md)) — the share link 404s (same as any not-yet-ready track) until ingest confirms the new file.
_Avoid_: re-upload, update file

**Rotate**:
Issuing a new share link for a Track or a Playlist so the old link immediately behaves as if it never existed (see [ADR 0005](./docs/adr/0005-rotate-mutates-slug-in-place-with-no-history.md)); nothing else about it changes. Rotating a Track does not affect any Playlist containing it, since Playlists never expose their Tracks' links.
_Avoid_: regenerate, revoke, reset link

### Playhead movement

Four distinct ways to change the player's position, kept distinct in code and conversation so they aren't conflated:

**Seek**:
Moving to an arbitrary position by dragging or clicking the seek bar. Never changes play/pause state.
_Avoid_: scrub, skip

**Skip**:
A relative jump of a fixed 10 seconds forward or backward from the current position, via the `j`/`l` keyboard shortcuts. Never changes play/pause state, and never moves to another Track.
_Avoid_: seek, nudge, next track

**Jump**:
Moving to an absolute position computed as a fraction of track duration (`n` &#215; 10%), via the `0`&#8211;`9` keyboard shortcuts. Never changes play/pause state.
_Avoid_: seek, chapter

**Advance**:
Moving to the next or previous Available Track in a Playlist — automatically when a Track ends (continuing playback), or on request via `Shift+N`/`Shift+P` (preserving play/pause state). Advancing backward more than a few seconds into a Track restarts that Track instead. Only exists on a Playlist's Share Page; at the end of the last Track, playback stops rather than looping.
_Avoid_: skip, next/prev (as nouns)
