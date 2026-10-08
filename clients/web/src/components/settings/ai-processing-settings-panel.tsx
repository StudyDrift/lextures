import { useCallback, useEffect, useRef, useState } from 'react'
import { Link, useLocation } from 'react-router-dom'
import { authorizedFetch } from '../../lib/api'
import { readApiErrorMessage } from '../../lib/errors'
import { AI_SETTINGS_ANCHOR_ID, aiDisclosureI18n } from '../../lib/ai-disclosure-i18n'
import { fetchAiTutorOptOutSetting, putAiTutorOptOut } from '../../lib/tutor-api'
import { toastMutationError, toastSaveOk } from '../../lib/lms-toast'
import { Checkbox } from '../ui/checkbox'

type Props = {
  embedded?: boolean
}

export function AiProcessingSettingsPanel({ embedded = false }: Props) {
  const [optOut, setOptOut] = useState(false)
  // null = the persistent AI tutor isn't available, so its toggle is hidden.
  const [tutorOptOut, setTutorOptOut] = useState<boolean | null>(null)
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const location = useLocation()
  const sectionRef = useRef<HTMLElement | null>(null)

  useEffect(() => {
    let cancelled = false
    void (async () => {
      try {
        const res = await authorizedFetch('/api/v1/settings/ai-opt-out')
        if (res.ok) {
          const data = (await res.json()) as { aiProcessingOptOut?: boolean }
          if (!cancelled) setOptOut(Boolean(data.aiProcessingOptOut))
        } else if (res.status !== 404) {
          throw new Error(await readApiErrorMessage(res))
        }
      } catch {
        /* module may be off */
      }
      try {
        const tutor = await fetchAiTutorOptOutSetting()
        if (!cancelled) setTutorOptOut(tutor)
      } catch {
        /* persistent tutor may be off */
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
      if (tutorOptOut !== null) {
        await putAiTutorOptOut(tutorOptOut)
      }
      toastSaveOk(aiDisclosureI18n.optOutSaved)
    } catch (e) {
      toastMutationError(e instanceof Error ? e.message : 'Could not save AI settings.')
    } finally {
      setSaving(false)
    }
  }, [optOut, tutorOptOut])

  // "Manage AI settings" links land on /settings/account#ai-settings; bring this block into view.
  useEffect(() => {
    if (loading || location.hash !== `#${AI_SETTINGS_ANCHOR_ID}`) return
    sectionRef.current?.scrollIntoView?.({ block: 'start' })
  }, [loading, location.hash])

  if (loading) {
    return <p className="text-sm text-fg-muted">Loading AI settings…</p>
  }

  return (
    <section
      id={AI_SETTINGS_ANCHOR_ID}
      ref={sectionRef}
      className={embedded ? '' : 'mt-8 border-t border-border-default pt-8 dark:border-border-default'}
      aria-labelledby="ai-processing-heading"
    >
      <h3 id="ai-processing-heading" className="text-sm font-medium text-fg-default">
        {aiDisclosureI18n.optOutTitle}
      </h3>
      <p className="mt-1 text-sm text-fg-muted">{aiDisclosureI18n.optOutDescription}</p>
      <label className="mt-4 flex cursor-pointer items-start gap-3">
        <input
          type="checkbox"
          className="mt-1 h-4 w-4 rounded border-border-strong"
          checked={optOut}
          onChange={(e) => setOptOut(e.target.checked)}
        />
        <span className="text-sm text-fg-default">{aiDisclosureI18n.optOutLabel}</span>
      </label>
      {tutorOptOut !== null ? (
        <div className="mt-3">
          <Checkbox
            id="ai-tutor-opt-out"
            checked={tutorOptOut}
            onChange={(e) => setTutorOptOut(e.target.checked)}
            label={aiDisclosureI18n.tutorOptOutLabel}
            description={aiDisclosureI18n.tutorOptOutDescription}
          />
        </div>
      ) : null}
      <p className="mt-2 text-sm">
        <Link to="/ai-disclosure" className="text-accent-fg underline dark:text-indigo-300">
          {aiDisclosureI18n.fullDisclosureLink}
        </Link>
      </p>
      <button
        type="button"
        disabled={saving}
        onClick={() => void save()}
        className="mt-4 rounded-xl bg-accent-solid px-4 py-2 text-sm font-semibold text-white hover:bg-indigo-500 disabled:opacity-60"
      >
        {saving ? 'Saving…' : 'Save'}
      </button>
    </section>
  )
}
