import { useCallback, useEffect, useState, type Dispatch, type SetStateAction } from 'react'
import { authorizedFetch } from '../../lib/api'
import { messageFromApiErrorBody } from '../../lib/errors'
import { aiDisclosureI18n } from '../../lib/ai-disclosure-i18n'
import { toastMutationError, toastSaveOk } from '../../lib/lms-toast'
import { ErrorState } from '../ui'
import { AdminAdaptiveContentReportPanel } from '../lms/adaptive-content/reports/admin-report-panel'
import { AdaptiveContentOversightPanel } from './adaptive-content-oversight-panel'

const FEATURE_KEYS = [
  { key: 'ai_tutor', label: 'AI Tutor' },
  { key: 'rag_notebook', label: 'Notebook AI' },
  { key: 'syllabus_generation', label: 'Syllabus generation' },
  { key: 'content_page_generation', label: 'Content page generation' },
  { key: 'marketing_article_generation', label: 'Marketing article generation' },
  { key: 'outcomes_extraction', label: 'Learning outcomes extraction' },
  { key: 'badges_extraction', label: 'Badge extraction' },
  { key: 'translation', label: 'Translation' },
  { key: 'quiz_generation', label: 'Quiz generation' },
  { key: 'lesson_generation', label: 'Lesson generator' },
  { key: 'adaptive_content', label: 'Adaptive content' },
] as const

type LoadState =
  | { status: 'loading' }
  | { status: 'ready' }
  | { status: 'disabled' }
  | { status: 'error'; message: string }

const FORBIDDEN_MESSAGE =
  "You don't have permission to view this organization's AI governance settings, so they can't be edited here."
const LOAD_FAILED_MESSAGE = 'Could not load AI governance settings. Try again in a moment.'
const DISABLED_MESSAGE = 'AI disclosure controls are not enabled on this platform.'

export function AiGovernancePanel() {
  const [enabled, setEnabled] = useState<Record<string, boolean>>({})
  const [allowedModels, setAllowedModels] = useState('')
  const [load, setLoad] = useState<LoadState>({ status: 'loading' })
  const [reloadKey, setReloadKey] = useState(0)
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    let cancelled = false
    void (async () => {
      try {
        const res = await authorizedFetch('/api/v1/admin/ai-config')
        if (res.status === 404) {
          if (!cancelled) setLoad({ status: 'disabled' })
          return
        }
        if (res.status === 403) {
          if (!cancelled) setLoad({ status: 'error', message: FORBIDDEN_MESSAGE })
          return
        }
        if (!res.ok) {
          throw new Error(messageFromApiErrorBody(await res.text(), LOAD_FAILED_MESSAGE))
        }
        const data = (await res.json()) as {
          featuresEnabled?: Record<string, boolean>
          allowedModels?: string[] | null
        }
        if (!cancelled) {
          setEnabled(data.featuresEnabled ?? {})
          setAllowedModels((data.allowedModels ?? []).join('\n'))
          setLoad({ status: 'ready' })
        }
      } catch (e) {
        if (!cancelled) {
          setLoad({
            status: 'error',
            message: e instanceof Error && e.message ? e.message : LOAD_FAILED_MESSAGE,
          })
        }
      }
    })()
    return () => {
      cancelled = true
    }
  }, [reloadKey])

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
        throw new Error(messageFromApiErrorBody(await res.text(), 'Could not save.'))
      }
      toastSaveOk(aiDisclosureI18n.adminSaved)
    } catch (e) {
      toastMutationError(e instanceof Error ? e.message : 'Could not save.')
    } finally {
      setSaving(false)
    }
  }, [allowedModels, enabled])

  const retry = useCallback(() => {
    setLoad({ status: 'loading' })
    setReloadKey((k) => k + 1)
  }, [])

  if (load.status === 'loading') {
    return null
  }

  return (
    <section aria-labelledby="ai-governance-heading">
      <h2 id="ai-governance-heading" className="text-base font-semibold text-fg-default">
        {aiDisclosureI18n.adminTitle}
      </h2>
      <p className="mt-1 text-sm text-fg-muted">{aiDisclosureI18n.adminIntro}</p>
      {load.status === 'ready' ? (
        <AiGovernanceForm
          enabled={enabled}
          setEnabled={setEnabled}
          allowedModels={allowedModels}
          setAllowedModels={setAllowedModels}
          saving={saving}
          onSave={() => void save()}
        />
      ) : (
        <div className="mt-4" role="alert">
          <ErrorState
            title={load.status === 'disabled' ? 'AI governance unavailable' : 'Could not load AI governance settings'}
            body={load.status === 'disabled' ? DISABLED_MESSAGE : load.message}
            primaryAction={
              load.status === 'error' ? { label: 'Retry', onClick: retry } : undefined
            }
          />
        </div>
      )}
      <AdaptiveContentOversightPanel />
      <AdminAdaptiveContentReportPanel />
    </section>
  )
}

type AiGovernanceFormProps = {
  enabled: Record<string, boolean>
  setEnabled: Dispatch<SetStateAction<Record<string, boolean>>>
  allowedModels: string
  setAllowedModels: (value: string) => void
  saving: boolean
  onSave: () => void
}

function AiGovernanceForm({
  enabled,
  setEnabled,
  allowedModels,
  setAllowedModels,
  saving,
  onSave,
}: AiGovernanceFormProps) {
  return (
    <>
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
      <label className="mt-4 block text-sm font-medium text-fg-default">
        Allowed models (one per line; empty = all)
        <textarea
          className="mt-1 w-full rounded-lg border border-border-default px-3 py-2 font-mono text-xs dark:border-border-default dark:bg-surface-raised"
          rows={4}
          value={allowedModels}
          onChange={(e) => setAllowedModels(e.target.value)}
        />
      </label>
      <button
        type="button"
        disabled={saving}
        onClick={onSave}
        className="mt-4 rounded-xl bg-accent-solid px-4 py-2 text-sm font-semibold text-white hover:bg-indigo-500 disabled:opacity-60"
      >
        {saving ? 'Saving…' : aiDisclosureI18n.adminSave}
      </button>
    </>
  )
}
