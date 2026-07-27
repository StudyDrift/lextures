import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import {
  fetchContentToolAIConsent,
  postContentToolAIConsent,
} from '../../../lib/content-tools-governance-api'

type Props = {
  courseCode: string
  toolId: string
  requiresAI: boolean
  onOptOut?: () => void
}

/**
 * CT.8 FR-4 / FR-12 — AI disclosure before first interaction (banner or acknowledge).
 * Not a modal trap; opt-out leaves a non-AI path when the parent provides one.
 */
export function AIDisclosureBanner({ courseCode, toolId, requiresAI, onOptOut }: Props) {
  const { t } = useTranslation('contentTools')
  const [mode, setMode] = useState<string>('banner')
  const [decision, setDecision] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!requiresAI) return
    let cancelled = false
    void fetchContentToolAIConsent(courseCode, toolId)
      .then((res) => {
        if (cancelled) return
        setMode(res.aiDisclosureMode || 'banner')
        setDecision(res.decision)
      })
      .catch(() => {
        /* non-fatal — show banner by default */
      })
    return () => {
      cancelled = true
    }
  }, [courseCode, toolId, requiresAI])

  if (!requiresAI || mode === 'none' || decision === 'acknowledged' || decision === 'opted_out') {
    return null
  }

  async function decide(next: 'acknowledged' | 'opted_out') {
    setBusy(true)
    setError(null)
    try {
      await postContentToolAIConsent(courseCode, { toolId, decision: next })
      setDecision(next)
      if (next === 'opted_out') onOptOut?.()
    } catch (e) {
      setError(e instanceof Error ? e.message : t('contentTools.governance.consentError'))
    } finally {
      setBusy(false)
    }
  }

  return (
    <div
      className="mb-3 rounded-md border border-slate-300 bg-slate-50 p-3 text-sm text-slate-800 dark:border-neutral-600 dark:bg-neutral-900 dark:text-slate-100"
      role="region"
      aria-label={t('contentTools.governance.aiDisclosureTitle')}
      data-testid="content-tool-ai-disclosure"
    >
      <p className="font-medium">{t('contentTools.governance.aiDisclosureTitle')}</p>
      <p className="mt-1 text-slate-600 dark:text-slate-300">
        {t('contentTools.governance.aiDisclosureBody')}
      </p>
      {error ? (
        <p className="mt-2 text-rose-700 dark:text-rose-300" role="alert">
          {error}
        </p>
      ) : null}
      <div className="mt-3 flex flex-wrap gap-2">
        <button
          type="button"
          className="rounded bg-slate-900 px-3 py-1.5 text-white disabled:opacity-50 dark:bg-slate-100 dark:text-slate-900"
          disabled={busy}
          onClick={() => void decide('acknowledged')}
          data-testid="content-tool-ai-ack"
        >
          {mode === 'acknowledge'
            ? t('contentTools.governance.acknowledge')
            : t('contentTools.governance.continueWithAI')}
        </button>
        <button
          type="button"
          className="rounded border border-slate-400 px-3 py-1.5 disabled:opacity-50"
          disabled={busy}
          onClick={() => void decide('opted_out')}
          data-testid="content-tool-ai-opt-out"
        >
          {t('contentTools.governance.optOut')}
        </button>
      </div>
    </div>
  )
}
