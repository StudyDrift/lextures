import { useCallback, useEffect, useState } from 'react'
import { authorizedFetch } from '../../lib/api'
import { readApiErrorMessage } from '../../lib/errors'
import { aiDisclosureI18n } from '../../lib/ai-disclosure-i18n'
import { toastMutationError, toastSaveOk } from '../../lib/lms-toast'

const FEATURE_KEYS = [
  { key: 'ai_tutor', label: 'AI Tutor' },
  { key: 'rag_notebook', label: 'Notebook AI' },
  { key: 'syllabus_generation', label: 'Syllabus generation' },
  { key: 'translation', label: 'Translation' },
  { key: 'quiz_generation', label: 'Quiz generation' },
  { key: 'lesson_generation', label: 'Lesson generator' },
] as const

export function AiGovernancePanel() {
  const [enabled, setEnabled] = useState<Record<string, boolean>>({})
  const [allowedModels, setAllowedModels] = useState('')
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    let cancelled = false
    void (async () => {
      try {
        const res = await authorizedFetch('/api/v1/admin/ai-config')
        if (res.status === 404 || res.status === 403) {
          return
        }
        if (!res.ok) {
          throw new Error(await readApiErrorMessage(res))
        }
        const data = (await res.json()) as {
          featuresEnabled?: Record<string, boolean>
          allowedModels?: string[] | null
        }
        if (!cancelled) {
          setEnabled(data.featuresEnabled ?? {})
          setAllowedModels((data.allowedModels ?? []).join('\n'))
        }
      } catch {
        /* not org admin */
      } finally {
        if (!cancelled) setLoading(false)
      }
    })()
    return () => {
      cancelled = true
    }
  }, [])

  const save = useCallback(async () => {
    setSaving(true)
    try {
      const models = allowedModels
        .split(/[\n,]+/)
        .map((s) => s.trim())
        .filter(Boolean)
      const featuresEnabled: Record<string, boolean> = {}
      for (const f of FEATURE_KEYS) {
        featuresEnabled[f.key] = enabled[f.key] !== false
      }
      const res = await authorizedFetch('/api/v1/admin/ai-config', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ featuresEnabled, allowedModels: models.length ? models : null }),
      })
      if (!res.ok) {
        throw new Error(await readApiErrorMessage(res))
      }
      toastSaveOk(aiDisclosureI18n.adminSaved)
    } catch (e) {
      toastMutationError(e instanceof Error ? e.message : 'Could not save.')
    } finally {
      setSaving(false)
    }
  }, [allowedModels, enabled])

  if (loading) {
    return null
  }

  return (
    <section className="mt-8" aria-labelledby="ai-governance-heading">
      <h3 id="ai-governance-heading" className="text-base font-semibold text-slate-900 dark:text-neutral-100">
        {aiDisclosureI18n.adminTitle}
      </h3>
      <p className="mt-1 text-sm text-slate-500 dark:text-neutral-400">{aiDisclosureI18n.adminIntro}</p>
      <ul className="mt-4 space-y-2">
        {FEATURE_KEYS.map((f) => (
          <li key={f.key}>
            <label className="flex items-center gap-2 text-sm">
              <input
                type="checkbox"
                checked={enabled[f.key] !== false}
                onChange={(e) => setEnabled((prev) => ({ ...prev, [f.key]: e.target.checked }))}
              />
              {f.label}
            </label>
          </li>
        ))}
      </ul>
      <label className="mt-4 block text-sm font-medium text-slate-700 dark:text-neutral-200">
        Allowed models (one per line; empty = all)
        <textarea
          className="mt-1 w-full rounded-lg border border-slate-200 px-3 py-2 font-mono text-xs dark:border-neutral-600 dark:bg-neutral-900"
          rows={4}
          value={allowedModels}
          onChange={(e) => setAllowedModels(e.target.value)}
        />
      </label>
      <button
        type="button"
        disabled={saving}
        onClick={() => void save()}
        className="mt-4 rounded-xl bg-indigo-600 px-4 py-2 text-sm font-semibold text-white hover:bg-indigo-500 disabled:opacity-60"
      >
        {saving ? 'Saving…' : aiDisclosureI18n.adminSave}
      </button>
    </section>
  )
}
