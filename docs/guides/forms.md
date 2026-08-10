# Forms and validation (UX.6)

Single system for labels, errors, validation timing, and dirty-state warnings.
Import controls from the design system barrel; declare validation once with **zod**.

## Quick start

```tsx
import { z } from 'zod'
import { Field, Input, ErrorSummary, Button } from '../components/ui'
import { useForm, useFormField, useFormState } from '../lib/form'

const schema = z.object({
  email: z.string().email(),
  displayName: z.string().min(1).max(80),
})

function ProfileForm() {
  const form = useForm({
    formId: 'profile',
    schema,
    defaultValues: { email: '', displayName: '' },
    labels: { email: 'Email', displayName: 'Display name' },
    onSubmit: async (values, { setServerErrors, setFormError }) => {
      const res = await authorizedFetch('/api/v1/...', {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(values),
      })
      const raw = await res.json().catch(() => ({}))
      if (!res.ok) {
        const { readApiFieldErrors, readApiErrorMessage } = await import('../lib/errors')
        const fields = readApiFieldErrors(raw)
        if (fields.length) setServerErrors(fields)
        else setFormError(readApiErrorMessage(raw))
        return
      }
    },
  })

  const { summary, isSubmitting, formError, isDirty } = useFormState(form)
  useUnsavedChanges(isDirty) // optional navigation guard

  return (
    <form
      noValidate
      onSubmit={(e) => {
        void form.handleSubmit(e)
      }}
      className="space-y-4"
    >
      <p className="text-xs text-fg-muted">{t('common.validation.requiredLegend')}</p>
      <ErrorSummary
        title={t('common.validation.errorSummaryTitle')}
        errors={summary.map((s) => ({
          id: s.id,
          label: s.label,
          message: s.message,
        }))}
      />
      {formError ? <InlineAlert tone="danger">{formError}</InlineAlert> : null}

      <NameField form={form} name="displayName" label="Display name" required />
      <NameField form={form} name="email" label="Email" required autoComplete="email" type="email" />

      <Button type="submit" disabled={isSubmitting}>
        Save
      </Button>
    </form>
  )
}

function NameField({
  form,
  name,
  label,
  required,
  ...inputProps
}: {
  form: ReturnType<typeof useForm>
  name: string
  label: string
  required?: boolean
} & React.ComponentProps<typeof Input>) {
  const field = useFormField(form, name)
  return (
    <Field label={label} error={field.error} required={required} htmlFor={field.id}>
      <Input {...field.controlProps} {...inputProps} />
    </Field>
  )
}
```

## Validation timing (FR-5)

| When | Behaviour |
|---|---|
| **Blur** | Validate the field just left (marks touched). |
| **Change** | Re-validate **only** if the field is already showing an error (errors clear as the user fixes them; no mid-typing spam). |
| **Submit** | Validate everything; render `ErrorSummary` and move focus to it. |

Do not invent per-form timing. Adjust the shared store if product needs a global tweak.

## Error copy (FR-7)

- Prefer `common.validation.*` keys (required, tooShort, invalidEmail, alreadyTaken, …).
- Say **what is wrong** and **how to fix it**. Never “Invalid input” alone.
- All four locales stay at parity (`en` / `es` / `fr` / `ar`).

## Field wiring (FR-1)

`<Field label="…" error={…} required description={…}>` owns:

- visible label + required asterisk
- `id` / `htmlFor`
- `aria-describedby` (description + error)
- `aria-invalid` / `aria-required` / `aria-busy` via **FieldContext** on child `Input` / `Textarea` / `Select`

Do **not** hand-roll `<input>` outside `components/ui/`. Use `Input` (or another library control) inside `Field`.

## Error summary (FR-6)

On failed submit, `useFormState(form).summary` feeds `<ErrorSummary />`:

- `role="alert"` + programmatic focus
- each item is a link that focuses `#fieldId`

## Server 422 envelope (FR-9)

```json
{
  "error": "validation_failed",
  "message": "Fix the highlighted fields",
  "fields": [
    { "path": "phoneNumber", "code": "invalid_phone", "message": "…", "params": {} }
  ]
}
```

- Client: `readApiFieldErrors(raw)` → `form.setServerErrors(fields)`.
- Legacy shapes still map when possible; otherwise show a page banner via `readApiErrorMessage`.
- Server helper: `apierr.WriteValidationFailed(w, message, fields)`.
- OpenAPI: `ValidationErrorResponse` / `ValidationFieldViolation`.

## Dirty forms (FR-10)

```tsx
import { useUnsavedChanges } from '../lib/form'
import { UnsavedChangesBanner } from '../components/ui'

useUnsavedChanges(isDirty) // beforeunload + router blocker
// Optional in-page banner:
<UnsavedChangesBanner visible={isDirty} … />
```

## Failed save / retry (FR-11)

Keep form values on network failure (`setFormError` only). Never reset on error. Retry reuses the same values — no re-entry.

## Autocomplete (FR-12)

Set correct `autoComplete` tokens on identity fields (`name`, `email`, `tel`, `current-password`, …).

## Lint / CI

```bash
cd clients/web
npm run forms:check          # bare controls + placeholder-as-label ratchet
npm run forms:baseline       # after a migration batch (counts must not rise)
```

Runbooks: [form-lint-failed.md](../runbooks/form-lint-failed.md),
[form-schema-divergence.md](../runbooks/form-schema-divergence.md).

## Gallery

`/design/components` → **Forms** section: default, required, invalid, error summary, pending.

## Migration checklist

1. Replace raw `<input>` / `<select>` / `<textarea>` with library controls.
2. Wrap with `Field` (visible label; no placeholder-as-label).
3. Declare a zod schema colocated with the form.
4. Wire `useForm` + `ErrorSummary` + server error mapping.
5. Add dirty navigation warning if the form can lose work.
6. Confirm `common.validation.*` keys cover new codes.
7. `npm run forms:check` still green (or re-baseline after a reviewed drop).
