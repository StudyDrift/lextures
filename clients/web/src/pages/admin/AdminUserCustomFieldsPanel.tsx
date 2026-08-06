import { useEffect, useState } from 'react'
import {
  fetchAdminUser,
  fetchCustomFields,
  patchAdminUserCustomFields,
  type CustomFieldDefinition,
} from '../../lib/custom-fields-api'

type Props = {
  userId: string
  orgId: string | null
  onClose: () => void
}

export default function AdminUserCustomFieldsPanel({ userId, orgId, onClose }: Props) {
  const [fieldDefs, setFieldDefs] = useState<CustomFieldDefinition[]>([])
  const [fieldValues, setFieldValues] = useState<Record<string, string>>({})
  const [loading, setLoading] = useState(true)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    let cancelled = false
    setLoading(true)
    setError(null)
    void Promise.all([fetchCustomFields('user', orgId), fetchAdminUser(userId, orgId, true)])
      .then(([defs, detail]) => {
        if (cancelled) return
        setFieldDefs(defs)
        const next: Record<string, string> = {}
        for (const def of defs) {
          const raw = detail.customFields?.[def.key]
          next[def.key] = raw == null ? '' : String(raw)
        }
        setFieldValues(next)
      })
      .catch(() => {
        if (!cancelled) {
          setFieldDefs([])
          setFieldValues({})
          setError('Failed to load custom fields.')
        }
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [userId, orgId])

  async function save() {
    setBusy(true)
    setError(null)
    try {
      const payload: Record<string, unknown> = {}
      for (const def of fieldDefs) {
        const raw = fieldValues[def.key] ?? ''
        if (def.fieldType === 'boolean') {
          payload[def.key] = raw === 'true'
        } else if (def.fieldType === 'number' && raw !== '') {
          payload[def.key] = Number(raw)
        } else if (raw !== '') {
          payload[def.key] = raw
        } else {
          payload[def.key] = null
        }
      }
      await patchAdminUserCustomFields(userId, payload, orgId)
      onClose()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to save custom fields.')
    } finally {
      setBusy(false)
    }
  }

  if (loading) {
    return <p className="text-sm text-fg-muted">Loading custom fields…</p>
  }

  return (
    <fieldset className="space-y-3">
      <legend className="text-sm font-medium text-fg-default dark:text-slate-200">Custom fields</legend>
      {error ? (
        <p role="alert" className="text-sm text-danger-fg">
          {error}
        </p>
      ) : null}
      {fieldDefs.map((def) => (
        <label key={def.id} className="block max-w-md text-sm">
          <span className="mb-1 block">{def.label}</span>
          {def.fieldType === 'select' ? (
            <select
              value={fieldValues[def.key] ?? ''}
              onChange={(e) => setFieldValues({ ...fieldValues, [def.key]: e.target.value })}
              className="w-full rounded-lg border border-border-strong px-3 py-2 dark:border-border-default dark:bg-surface-raised"
            >
              <option value="">—</option>
              {(def.selectOptions ?? []).map((opt) => (
                <option key={opt} value={opt}>
                  {opt}
                </option>
              ))}
            </select>
          ) : def.fieldType === 'boolean' ? (
            <select
              value={fieldValues[def.key] ?? ''}
              onChange={(e) => setFieldValues({ ...fieldValues, [def.key]: e.target.value })}
              className="w-full rounded-lg border border-border-strong px-3 py-2 dark:border-border-default dark:bg-surface-raised"
            >
              <option value="">—</option>
              <option value="true">Yes</option>
              <option value="false">No</option>
            </select>
          ) : (
            <input
              type={def.fieldType === 'number' ? 'number' : def.fieldType === 'date' ? 'date' : 'text'}
              value={fieldValues[def.key] ?? ''}
              onChange={(e) => setFieldValues({ ...fieldValues, [def.key]: e.target.value })}
              className="w-full rounded-lg border border-border-strong px-3 py-2 dark:border-border-default dark:bg-surface-raised"
            />
          )}
        </label>
      ))}
      <div className="flex gap-2">
        <button
          type="button"
          disabled={busy}
          onClick={() => void save()}
          className="rounded-lg bg-accent-solid px-3 py-2 text-sm font-medium text-white hover:bg-accent disabled:opacity-50"
        >
          Save custom fields
        </button>
        <button type="button" onClick={onClose} className="rounded-lg px-3 py-2 text-sm">
          Cancel
        </button>
      </div>
    </fieldset>
  )
}
