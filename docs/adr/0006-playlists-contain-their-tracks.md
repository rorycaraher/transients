# A Playlist's Share Page never exposes its Tracks' own share links

**Status**: accepted

A Playlist's Share Page could either link each entry in its tracklist out
to that Track's own Share Page (`/t/{slug}`), or play every Track inline
and never reveal a Track's slug at all — not in rendered links, page
metadata, or the play-count beacon URL. We chose **containment**: a
listener who reaches a Track through a Playlist gets no handle on the Track
itself, so their access lasts exactly as long as the Playlist's does.

Linking out is the more obvious design and is friendlier to a listener who
wants to pass on one song, but it would make a Playlist's own expiry
largely cosmetic: anyone could click through, bookmark a Track with no
expiry of its own, and keep listening after the Playlist expired or was
Rotated. That would also mean sharing a Playlist silently hands out a
permanent per-Track link the admin never chose to give anyone. Containment
keeps "share this Playlist" meaning exactly that, and makes the two link
types independent: Rotating a Track's link affects no Playlist, and
Rotating a Playlist's link revokes all access gained through it in one step.

Consequences: every per-Track action the Playlist page needs — presigning
the Audio File, recording a Play, offering a download for a downloadable
Track — has to be addressed through the Playlist (Playlist slug plus a
Track reference that isn't the Track's slug), not through the existing
`/t/{slug}/...` routes. Reversing this later is easy in code but not in
effect: once Playlist pages have handed out Track links, those links can't
be recalled short of Rotating every affected Track.
