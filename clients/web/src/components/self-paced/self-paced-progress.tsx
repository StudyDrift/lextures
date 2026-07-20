// Self-paced learner progress UI: accessible progress bar, resume CTA, and a
// completion celebration overlay (plan 15.2).
import { useCallback, useEffect, useMemo, useState } from 'react'
import { PartyPopper, Play } from 'lucide-react'
import { CredentialShareActions } from '../credentials/credential-share-actions'
import { usePlatformFeatures } from '../../context/platform-features-context'
import type { IssuedCredentialSummary } from '../../lib/credentials-api'
import {
  fetchMyProgress,
  formatProgressLabel,
  type SelfPacedProgress,
} from '../../lib/self-paced-api'
import { AnimatedProgress } from '../ui/animated-progress'
import { DelightMoment } from '../ui/delight-moment'

/** Accessible WCAG 2.1 AA progress bar with a visible text percentage. */
export function SelfPacedProgressBar({
  percent,
  label,
}: {
  percent: number
  label?: string
}) {
  const clamped = Math.max(0, Math.min(100, Math.round(percent)))
  const text = formatProgressLabel(clamped)
  return (
    <div className="flex items-center gap-3">
      <AnimatedProgress
        value={clamped}
        label={label ?? text}
        className="h-2 flex-1 overflow-hidden rounded-full bg-slate-200 dark:bg-slate-700 lx-delight-progress"
        fillClassName="h-full rounded-full bg-emerald-500"
      />
      <span className="shrink-0 text-xs font-semibold tabular-nums text-slate-700 dark:text-slate-200">
        {text}
      </span>
    </div>
  )
}

/** Full-page completion celebration overlay shown when progress reaches 100%. */
export function CompletionCelebration({
  onClose,
  credential,
}: {
  onClose: () => void
  credential?: IssuedCredentialSummary | null
}) {
  return (
    <div
      role="dialog"
      aria-modal="true"
      aria-label="Course complete"
      className="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/70 p-4"
    >
      <DelightMoment
        active
        kind="completion"
        announcement="Course complete"
        className="w-full max-w-md"
      >
      <div className="w-full max-w-md rounded-2xl bg-white p-8 text-center shadow-xl dark:bg-slate-800">
        <PartyPopper className="mx-auto h-12 w-12 text-emerald-500" aria-hidden />
        <h2 className="mt-4 text-xl font-bold text-slate-900 dark:text-white">
          Course complete!
        </h2>
        <p className="mt-2 text-sm text-slate-600 dark:text-slate-300">
          You finished every item in this course at your own pace. Nice work.
        </p>
        {credential ? (
          <div className="mt-6 text-left">
            <CredentialShareActions credential={credential} layout="stack" />
          </div>
        ) : null}
        <div className="mt-4">
          <button
            type="button"
            onClick={onClose}
            className="w-full rounded-lg px-4 py-2 text-sm font-medium text-slate-600 hover:bg-slate-100 dark:text-slate-300 dark:hover:bg-slate-700"
          >
            Keep exploring
          </button>
        </div>
      </div>
      </DelightMoment>
    </div>
  )
}

/**
 * SelfPacedProgressHeader fetches and renders the viewer's progress for a self-paced
 * course: a progress bar, a "Resume" button to the last unfinished item, and a
 * completion celebration when the course reaches 100%.
 */
export function SelfPacedProgressHeader({
  courseCode,
  courseTitle,
  onResume,
}: {
  courseCode: string
  courseTitle?: string
  onResume?: (itemId: string) => void
}) {
  const { ffCompletionCredentials } = usePlatformFeatures()
  const [progress, setProgress] = useState<SelfPacedProgress | null>(null)
  const [celebrate, setCelebrate] = useState(false)
  const [issuedCredentialId, setIssuedCredentialId] = useState<string | null>(null)

  const load = useCallback(async () => {
    try {
      const p = await fetchMyProgress(courseCode)
      setProgress(p)
      if (p.justCompleted) {
        setCelebrate(true)
        if (p.credentialId) setIssuedCredentialId(p.credentialId)
      } else if (p.completed) {
        setCelebrate(true)
        if (p.credentialId) setIssuedCredentialId(p.credentialId)
      }
    } catch {
      setProgress(null)
    }
  }, [courseCode])

  useEffect(() => {
    void load()
  }, [load])

  const celebrationCredential = useMemo((): IssuedCredentialSummary | null => {
    if (!issuedCredentialId || !ffCompletionCredentials) return null
    const origin = window.location.origin
    return {
      id: issuedCredentialId,
      title: courseTitle ?? 'Course completion',
      sourceType: 'course',
      sourceId: courseCode,
      issuedAt: new Date().toISOString(),
      verificationUrl: `${origin}/verify/${issuedCredentialId}`,
      revoked: false,
    }
  }, [issuedCredentialId, ffCompletionCredentials, courseTitle, courseCode])

  if (!progress) return null

  return (
    <section
      aria-label="Course progress"
      className="rounded-xl border border-slate-200 bg-white p-4 dark:border-slate-700 dark:bg-slate-800"
    >
      <div className="flex items-center justify-between gap-4">
        <h2 className="text-sm font-semibold text-slate-900 dark:text-white">
          Your progress
        </h2>
        {progress.resumeItemId && !progress.completed ? (
          <button
            type="button"
            onClick={() => onResume?.(progress.resumeItemId as string)}
            aria-label="Resume course at your last unfinished item"
            className="inline-flex items-center gap-1 rounded-lg bg-emerald-600 px-3 py-1.5 text-xs font-semibold text-white hover:bg-emerald-700"
          >
            <Play className="h-3.5 w-3.5 ms-px" aria-hidden />
            Resume
          </button>
        ) : null}
      </div>
      <div className="mt-3">
        <SelfPacedProgressBar percent={progress.progressPercent} />
      </div>
      {celebrate ? (
        <CompletionCelebration
          credential={celebrationCredential}
          onClose={() => setCelebrate(false)}
        />
      ) : null}
    </section>
  )
}
