import { useCallback, useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { authorizedFetch } from '../../lib/api'
import { readApiErrorMessage } from '../../lib/errors'
import { aiDisclosureI18n } from '../../lib/ai-disclosure-i18n'
import { toastMutationError, toastSaveOk } from '../../lib/lms-toast'

type Props = {
  embedded?: boolean
}

export function AiProcessingSettingsPanel({ embedded = false }: Props) {
  const [optOut, setOptOut] = useState(false)
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    let cancelled = false
    void (async () => {
      try {
        const res = await authorizedFetch('/api/v1/settings/ai-opt-out')
        if (res.status === 404) {
          return
        }
        if (!res.ok) {
          throw new Error(await readApiErrorMessage(res))
        }
        const data = (await res.json()) as { aiProcessingOptOut?: boolean }
        if (!cancelled) setOptOut(Boolean(data.aiProcessingOptOut))
      } catch {
        /* module may be off */
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
      const res = await authorizedFetch('/api/v1/settings/ai-opt-out', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ aiProcessingOptOut: optOut }),
      })
      if (!res.ok) {
        throw new Error(await readApiErrorMessage(res))
      }
      toastSaveOk(aiDisclosureI18n.optOutSaved)
    } catch (e) {
      toastMutationError(e instanceof Error ? e.message : 'Could not save AI settings.')
    } finally {
      setSaving(false)
    }
  }, [optOut])

  if (loading) {
    return <p className="text-sm text-slate-500">Loading AI settings…</p>
  }

  return (
    <section
      className={embedded ? '' : 'mt-8 border-t border-slate-200 pt-8 dark:border-neutral-600'}
      aria-labelledby="ai-processing-heading"
    >
      <h3 id="ai-processing-heading" className="text-sm font-medium text-slate-700 dark:text-neutral-200">
        {aiDisclosureI18n.optOutTitle}
      </h3>
      <p className="mt-1 text-sm text-slate-500 dark:text-neutral-400">{aiDisclosureI18n.optOutDescription}</p>
      <label className="mt-4 flex cursor-pointer items-start gap-3">
        <input
          type="checkbox"
          className="mt-1 h-4 w-4 rounded border-slate-300"
          checked={optOut}
          onChange={(e) => setOptOut(e.target.checked)}
        />
        <span className="text-sm text-slate-700 dark:text-neutral-200">{aiDisclosureI18n.optOutLabel}</span>
      </label>
      <p className="mt-2 text-sm">
        <Link to="/ai-disclosure" className="text-indigo-700 underline dark:text-indigo-300">
          {aiDisclosureI18n.fullDisclosureLink}
        </Link>
      </p>
      <button
        type="button"
        disabled={saving}
        onClick={() => void save()}
        className="mt-4 rounded-xl bg-indigo-600 px-4 py-2 text-sm font-semibold text-white hover:bg-indigo-500 disabled:opacity-60"
      >
        {saving ? 'Saving…' : 'Save'}
      </button>
    </section>
  )
}
