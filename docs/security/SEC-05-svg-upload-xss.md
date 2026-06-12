# SEC-05 — SVG branding upload served same-origin → stored XSS

- **Severity:** High
- **Status:** Confirmed present
- **Area:** Server / uploads
- **File:** [server/internal/httpserver/org_branding_http.go](../../server/internal/httpserver/org_branding_http.go) (`sniffImageKind` ~L410, `svgSnippetLooksLikeSVG` ~L437, upload handler ~L337, public asset handler)

## Problem

`sniffImageKind` classifies an upload as `image/svg+xml` whenever the bytes start with `<svg` or contain `<svg` in the first chunk:

```go
case bytes.HasPrefix(bytes.TrimSpace(data), []byte("<svg")) || svgSnippetLooksLikeSVG(data):
    return "image/svg+xml"
```

The branding upload handler then writes the bytes verbatim, and the public branding-asset handler serves them back as `Content-Type: image/svg+xml` from the API origin (`/api/v1/public/org-branding/...`). There is **no SVG sanitization**. SVG is an active document format: it can carry `<script>`, `onload=` handlers, and `<foreignObject>` HTML. Org branding is editable by any org admin or unit admin.

## Risk

A tenant admin can plant an SVG containing JavaScript that executes in the browser at the API origin whenever anyone navigates to the asset URL directly (embedded `<iframe>`, a link, or a logo in a password-reset email). Because the SPA shares that origin, the script can read both tokens from `localStorage` (SEC-02) and exfiltrate them — turning a merely-privileged tenant admin into a session-stealer against every other user, including Global Admins. No CSP currently constrains it (SEC-03).

## Fix

Pick one, ideally both:

1. **Reject SVG outright.** Accept only PNG/JPEG/GIF/WebP, validated by magic-byte sniffing (not a `<svg` substring). This is the simplest durable fix and is recommended unless SVG logos are a hard product requirement.
2. If SVG must be supported: sanitize server-side before write (strip `<script>`, event-handler attributes, `<foreignObject>`, external references) using a vetted SVG sanitizer, **and** serve with `Content-Disposition: inline`, `X-Content-Type-Options: nosniff`, and `Content-Security-Policy: default-src 'none'; style-src 'unsafe-inline'`, **and** serve from a separate cookieless origin so a bypass can't reach SPA storage.

## Verification

- Uploading an SVG containing `<script>alert(1)</script>` or `<svg onload=...>` is either rejected, or the stored/served bytes contain no script and no event handlers.
- The served response carries `nosniff` and a restrictive CSP.
- Regression test added under `tests/` covering a malicious SVG payload.
