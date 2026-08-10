# Runbook: Client / server form schema divergence

**Plan:** [UX.6](../completed/ui-ux/UX.6-form-and-validation-system.md) FR-8 / AC-11  
**OpenAPI schema:** `ValidationErrorResponse` in `server/internal/openapi/openapi.json`

## Symptom

- Client zod accepts a payload the server rejects (or vice versa).
- Users see a generic banner with no field-level fix.
- CI contract check fails (when enabled for the endpoint group).

## Fix path

1. **Server is authoritative.** Align the client zod schema to the OpenAPI request body (required fields, types, max lengths).
2. Prefer field-addressable 422 via `apierr.WriteValidationFailed` so the UI can attach errors:

   ```go
   apierr.WriteValidationFailed(w, "Fix the highlighted fields", []apierr.FieldViolation{
     {Path: "courseCode", Code: "already_taken", Message: "That course code is already in use."},
   })
   ```

3. Client mapping:

   ```ts
   const fields = readApiFieldErrors(raw)
   if (fields.length) form.setServerErrors(fields)
   else form.setFormError(readApiErrorMessage(raw))
   ```

4. Regenerate web types if the OpenAPI document changed:

   ```bash
   cd clients/web && npm run openapi:types:file
   ```

5. Keep error **codes** stable; change **messages** via i18n (`common.validation.*`).

## Incremental adoption

Endpoints migrate behind product work. The client tolerates the legacy `{ error: { code, message } }` shape and degrades to a page banner until the endpoint emits the UX.6 envelope.

## Related

- [forms.md](../guides/forms.md)
- `make openapi-check`
