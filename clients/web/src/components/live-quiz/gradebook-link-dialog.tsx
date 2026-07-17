import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import {
  pushGameGradebookLink,
  unlinkGameGradebook,
  type GradePreview,
  type GradebookLink,
  type GradebookMapping,
} from '../../lib/live-quiz-api'

type Props = {
  courseCode: string
  gameId: string
  open: boolean
  existing?: GradebookLink | null
  onClose: () => void
  onChanged: () => void
}

export function GradebookLinkDialog({
  courseCode,
  gameId,
  open,
  existing,
  onClose,
  onChanged,
}: Props) {
  const { t } = useTranslation()
  const [mapping, setMapping] = useState<GradebookMapping>('participation')
  const [pointsPossible, setPointsPossible] = useState(10)
  const [participationPct, setParticipationPct] = useState(50)
  const [preview, setPreview] = useState<GradePreview[]>([])
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!open) return
    if (existing?.mapping) setMapping(existing.mapping)
    if (existing?.pointsPossible) setPointsPossible(existing.pointsPossible)
    if (existing?.participationPct) setParticipationPct(existing.participationPct)
  }, [open, existing])

  if (!open) return null

  async function runPreview() {
    setBusy(true)
    setError(null)
    try {
      const res = await pushGameGradebookLink(courseCode, gameId, {
        mapping,
        pointsPossible,
        participationPct,
        previewOnly: true,
      })
      setPreview(res.preview ?? [])
    } catch (e) {
      setError(e instanceof Error ? e.message : t('liveQuiz.gradebook.error'))
    } finally {
      setBusy(false)
    }
  }

  async function runPush() {
    setBusy(true)
    setError(null)
    try {
      await pushGameGradebookLink(courseCode, gameId, {
        mapping,
        pointsPossible,
        participationPct,
      })
      onChanged()
      onClose()
    } catch (e) {
      setError(e instanceof Error ? e.message : t('liveQuiz.gradebook.error'))
    } finally {
      setBusy(false)
    }
  }

  async function runUnlink() {
    if (!existing) return
    setBusy(true)
    setError(null)
    try {
      await unlinkGameGradebook(courseCode, gameId, existing.id)
      onChanged()
      onClose()
    } catch (e) {
      setError(e instanceof Error ? e.message : t('liveQuiz.gradebook.error'))
    } finally {
      setBusy(false)
    }
  }

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4"
      role="dialog"
      aria-modal="true"
      aria-labelledby="live-quiz-gradebook-title"
    >
      <div className="max-h-[90vh] w-full max-w-lg overflow-auto rounded-lg bg-background p-4 shadow-lg">
        <h2 id="live-quiz-gradebook-title" className="mb-3 text-lg font-semibold">
          {t('liveQuiz.gradebook.title')}
        </h2>
        <p className="mb-4 text-sm text-muted-foreground">{t('liveQuiz.gradebook.subtitle')}</p>

        <label className="mb-3 block text-sm">
          <span className="mb-1 block font-medium">{t('liveQuiz.gradebook.mapping')}</span>
          <select
            className="w-full rounded-md border bg-background px-3 py-2"
            value={mapping}
            onChange={(e) => setMapping(e.target.value as GradebookMapping)}
          >
            <option value="participation">{t('liveQuiz.gradebook.mappingParticipation')}</option>
            <option value="percent_correct">{t('liveQuiz.gradebook.mappingPercent')}</option>
            <option value="raw_points">{t('liveQuiz.gradebook.mappingRaw')}</option>
          </select>
        </label>

        <label className="mb-3 block text-sm">
          <span className="mb-1 block font-medium">{t('liveQuiz.gradebook.pointsPossible')}</span>
          <input
            type="number"
            min={0}
            step={0.5}
            className="w-full rounded-md border bg-background px-3 py-2"
            value={pointsPossible}
            onChange={(e) => setPointsPossible(Number(e.target.value))}
          />
        </label>

        {mapping === 'participation' && (
          <label className="mb-3 block text-sm">
            <span className="mb-1 block font-medium">{t('liveQuiz.gradebook.participationPct')}</span>
            <input
              type="number"
              min={1}
              max={100}
              className="w-full rounded-md border bg-background px-3 py-2"
              value={participationPct}
              onChange={(e) => setParticipationPct(Number(e.target.value))}
            />
          </label>
        )}

        {preview.length > 0 && (
          <div className="mb-3">
            <h3 className="mb-1 text-sm font-medium">{t('liveQuiz.gradebook.preview')}</h3>
            <table className="w-full text-start text-sm">
              <thead>
                <tr className="border-b">
                  <th className="py-1 pe-2">{t('liveQuiz.report.nickname')}</th>
                  <th className="py-1">{t('liveQuiz.gradebook.points')}</th>
                </tr>
              </thead>
              <tbody>
                {preview.map((p, i) => (
                  <tr key={p.userId ?? `g-${i}`} className="border-b border-border/50">
                    <td className="py-1 pe-2">
                      {p.nickname ?? p.userId ?? '—'}
                      {p.skippedGuest ? ` (${t('liveQuiz.report.guest')})` : ''}
                    </td>
                    <td className="py-1">
                      {p.skippedGuest ? '—' : `${p.pointsEarned} / ${p.pointsPossible}`}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}

        {error && <p className="mb-3 text-sm text-destructive">{error}</p>}

        <div className="flex flex-wrap gap-2">
          <button
            type="button"
            className="rounded-md border px-3 py-2 text-sm disabled:opacity-50"
            disabled={busy}
            onClick={() => void runPreview()}
          >
            {t('liveQuiz.gradebook.previewAction')}
          </button>
          <button
            type="button"
            className="rounded-md bg-primary px-3 py-2 text-sm text-primary-foreground disabled:opacity-50"
            disabled={busy}
            onClick={() => void runPush()}
          >
            {existing ? t('liveQuiz.gradebook.update') : t('liveQuiz.gradebook.push')}
          </button>
          {existing && (
            <button
              type="button"
              className="rounded-md border border-destructive px-3 py-2 text-sm text-destructive disabled:opacity-50"
              disabled={busy}
              onClick={() => void runUnlink()}
            >
              {t('liveQuiz.gradebook.unlink')}
            </button>
          )}
          <button
            type="button"
            className="ms-auto rounded-md px-3 py-2 text-sm underline"
            onClick={onClose}
          >
            {t('liveQuiz.gradebook.close')}
          </button>
        </div>
      </div>
    </div>
  )
}
