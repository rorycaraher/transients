# No external fonts or CDN assets in the frontend

**Status**: accepted

Share pages are viewed by recipients who never opted into the app directly —
they just clicked a link someone sent them. The app is also explicitly
self-hosted and single-binary (no build pipeline; see README), with audio
itself kept private behind short-lived presigned R2 URLs. Pulling a webfont
from Google Fonts or another CDN for the app-wide dark redesign would mean
every page view — including by recipients who are otherwise invisible to
any third party — makes an external network request, leaking their IP/UA to
that CDN.

We decided to use system font stacks only (sans for UI, monospace for
metadata like durations/filenames) rather than a self-hosted or CDN-loaded
custom typeface. This keeps zero runtime external requests, consistent with
how audio delivery already works. A future distinctive-typography pass, if
wanted, should self-host `.woff2` files under `internal/web/static/fonts/`
rather than reach for a CDN — reaching for a CDN is the "obvious" path most
apps take, so it's worth flagging that this project has deliberately ruled
it out.
