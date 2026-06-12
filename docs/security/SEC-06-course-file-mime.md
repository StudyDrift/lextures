# SEC-06 — Course-file content served with DB-controlled MIME, no `nosniff`, no `Content-Disposition`

- **Severity:** High
- **Status:** Confirmed present
- **Area:** Server / uploads
- **File:** [server/internal/httpserver/course_file_content.go](../../server/internal/httpserver/course_file_content.go) (local-disk branch, ~L78–L84)

## Problem

When a course file is served from local disk (no S3 presign), the handler sets the response content type to whatever was stored on the row and writes the bytes:

```go
ct := strings.TrimSpace(row.MimeType)
if ct == "" {
    ct = "application/octet-stream"
}
w.Header().Set("Content-Type", ct)
w.Header().Set("Cache-Control", "private, max-age=86400")
w.WriteHeader(http.StatusOK)
_, _ = w.Write(b)
```

There is no `X-Content-Type-Options: nosniff`, no `Content-Disposition: attachment`, and no allowlist. The MIME type is fully DB-controlled. Several ingest paths set `mime_type` from untrusted input — Canvas import, QTI import, and manual file attachment. A row created with `mime_type='text/html'` (or `image/svg+xml`) is rendered **inline at the API origin**.

## Risk

Same blast radius as SEC-05. A teacher who can attach a file to a course can hand a grader an HTML document that runs JavaScript in the grader's session at the SPA origin, reading tokens from `localStorage` (SEC-02). No CSP constrains it (SEC-03). Stored XSS reachable by any course author.

## Fix

1. Always send `X-Content-Type-Options: nosniff` on this response.
2. Re-derive the content type from the bytes with `http.DetectContentType` and only honor the stored MIME if it belongs to the same family. Otherwise fall back to `application/octet-stream`.
3. For anything outside a render-safe allowlist (`image/*`, `application/pdf`, `text/plain`), set `Content-Disposition: attachment; filename="<sanitized>"` so the browser downloads rather than renders.
4. Long term: serve all user-uploaded content from a separate, cookieless storage origin (the S3-presign path already does this; make the local-disk path match the security posture).

## Verification

- A course file with `mime_type='text/html'` and `<script>` content downloads as an attachment (or is served `text/plain`/`octet-stream`), never executing.
- Every response from this handler carries `nosniff`.
- Regression test under `tests/` uploads an HTML course file and asserts it does not render as `text/html` inline.
