# Runbook: Form lint failed (bare input / placeholder-as-label)

**Check:** `cd clients/web && npm run forms:check`  
**Plan:** [UX.6 Form and validation system](../completed/ui-ux/UX.6-form-and-validation-system.md)

## What failed

The ratchet reports one or more of:

| Metric | Meaning |
|---|---|
| `inputs_outside_field_component` | Raw `<input>` / `<select>` / `<textarea>` outside `components/ui/` increased |
| `fields_without_label` | Heuristic placeholder-as-label occurrences increased |

## Fix

1. Prefer library controls:

   ```tsx
   import { Field, Input } from '../components/ui'

   <Field label="Email" required>
     <Input type="email" autoComplete="email" />
   </Field>
   ```

2. Never use `placeholder` as the only label — always a visible `<label>` via `Field`.
3. Hidden file inputs wrapped in a labelled control are fine; the heuristic skips `type="file"` / `hidden`.
4. Rich editors (TipTap, CodeMirror) stay outside this rule; they are not native form controls.

## Re-baselining (rare)

Only after a **reviewed migration batch** that lowers counts:

```bash
cd clients/web
npm run forms:baseline
```

Do **not** raise baselines to silence a PR. Counts must monotonically decrease toward zero (AC-10).

## Related

- Engineer guide: [forms.md](../guides/forms.md)
- Design system coverage: `npm run ds:coverage`
