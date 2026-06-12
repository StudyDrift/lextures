# SEC-10 — `course_code` flows into filesystem path with weak sanitization

- **Severity:** Medium (defense-in-depth; not exploitable through the random code generator today)
- **Status:** Confirmed present
- **Area:** Server / file storage
- **File:** [server/internal/repos/coursefiles/paths.go](../../server/internal/repos/coursefiles/paths.go) (`diskCourseDirSegment`, `BlobDiskPath`), used by [server/internal/httpserver/course_file_content.go](../../server/internal/httpserver/course_file_content.go)

## Problem

`diskCourseDirSegment` permits `.`, `_`, and `-`, replacing other characters with `_`. It does **not** reject `..` (two literal dots) or an all-dots segment. `BlobDiskPath` then does `filepath.Join(root, seg, key)`. `filepath.Join` collapses `..`, so a segment of `..` traverses one directory above `root`. The key half is blunted by `filepath.Base(storageKey)`, but the course-code-derived segment half is not.

## Risk

Path traversal allowing reads/writes outside the course-files root, *if* any code path lets an attacker choose the course code. The random `C-XXXXXX` generator prevents this for normal flows, but cross-listing import, QTI manifests with attacker-controlled XML, and LTI deep-link mappings can introduce externally-derived codes. This is a latent escalation that pairs with SEC-08.

## Fix

1. After `filepath.Join`, `filepath.Clean` the result and assert it stays within root:
   ```go
   joined := filepath.Clean(filepath.Join(root, seg, key))
   if !strings.HasPrefix(joined+string(filepath.Separator), filepath.Clean(root)+string(filepath.Separator)) {
       return "", errOutsideRoot
   }
   ```
2. Reject any segment whose `filepath.Clean` is `..` or contains a `..` element, and never allow `.` as a sole-character segment.
3. Enforce the boundary `course_code` regex from SEC-08 (`^[A-Za-z0-9_-]{1,32}$`) so traversal characters never reach this function.

## Verification

- `BlobDiskPath(root, "..", key)` returns an error, not a path above root.
- A course code containing `..` or `/` is rejected at the route boundary.
