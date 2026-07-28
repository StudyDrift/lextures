import { useId, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { runContentToolAction } from '../../../../lib/courses-api'

export type ExplainItBackEditorProps = {
  value: Record<string, unknown>
  onChange: (next: Record<string, unknown>) => void
  disabled?: boolean
  idPrefix?: string
  courseCode?: string
  instanceId?: string
}

type KeyPoint = { id: string; label: string; description: string }

function newId(prefix: string): string {
  return `${prefix}_${Math.random().toString(36).slice(2, 9)}`
}

function asKeyPoints(value: Record<string, unknown>): KeyPoint[] {
  if (!Array.isArray(value.keyPoints)) return []
  return value.keyPoints as KeyPoint[]
}

export function ExplainItBackEditor({
  value,
  onChange,
  disabled,
  idPrefix = 'eib-editor',
  courseCode,
  instanceId,
}: ExplainItBackEditorProps) {
  const { t } = useTranslation('contentTools')
  const baseId = useId()
  const keyPoints = asKeyPoints(value)
  const [sample, setSample] = useState('')
  const [sampleBusy, setSampleBusy] = useState(false)
  const [sampleError, setSampleError] = useState<string | null>(null)
  const [sampleFeedback, setSampleFeedback] = useState<string | null>(null)

  function patch(partial: Record<string, unknown>) {
    onChange({ ...value, ...partial })
  }

  function setKeyPoints(next: KeyPoint[]) {
    patch({ keyPoints: next.slice(0, 6) })
  }

  async function onTestSample() {
    if (!courseCode || !instanceId || sampleBusy) {
      setSampleError(t('contentTools.tools.explain_it_back.editor.testUnavailable'))
      return
    }
    setSampleBusy(true)
    setSampleError(null)
    setSampleFeedback(null)
    try {
      const res = await runContentToolAction(courseCode, instanceId, 'test_sample', {
        input: { text: sample.trim() },
      })
      const result =
        res.result && typeof res.result === 'object'
          ? (res.result as {
              error?: string
              message?: string
              feedback?: { strength?: string; suggestion?: string; mode?: string }
            })
          : {}
      if (result.error) {
        setSampleError(result.message || result.error)
        return
      }
      const fb = result.feedback
      setSampleFeedback(
        [
          fb?.mode ? `Mode: ${fb.mode}` : null,
          fb?.strength ? `Strength: ${fb.strength}` : null,
          fb?.suggestion ? `Suggestion: ${fb.suggestion}` : null,
        ]
          .filter(Boolean)
          .join('\n'),
      )
    } catch (e: unknown) {
      setSampleError(e instanceof Error ? e.message : t('contentTools.runtime.retry'))
    } finally {
      setSampleBusy(false)
    }
  }

  const feedbackStyle =
    value.feedbackStyle === 'neutral' || value.feedbackStyle === 'socratic'
      ? value.feedbackStyle
      : 'encouraging'

  return (
    <div className="space-y-4" data-testid="explain-it-back-editor">
      <label className="block space-y-1 text-xs">
        <span className="font-medium text-slate-700 dark:text-neutral-300">
          {t('contentTools.tools.explain_it_back.editor.prompt')}
        </span>
        <textarea
          id={`${idPrefix}-${baseId}-prompt`}
          className="w-full rounded border border-slate-300 bg-white px-2 py-1.5 text-sm dark:border-neutral-600 dark:bg-neutral-950"
          rows={3}
          disabled={disabled}
          value={typeof value.prompt === 'string' ? value.prompt : ''}
          onChange={(e) => patch({ prompt: e.target.value })}
        />
      </label>

      <div className="grid grid-cols-2 gap-3">
        <label className="block space-y-1 text-xs">
          <span className="font-medium text-slate-700 dark:text-neutral-300">
            {t('contentTools.tools.explain_it_back.editor.minWords')}
          </span>
          <input
            type="number"
            min={5}
            max={500}
            className="w-full rounded border border-slate-300 bg-white px-2 py-1.5 text-sm dark:border-neutral-600 dark:bg-neutral-950"
            disabled={disabled}
            value={typeof value.minWords === 'number' ? value.minWords : 25}
            onChange={(e) => patch({ minWords: Number(e.target.value) || 25 })}
          />
        </label>
        <label className="block space-y-1 text-xs">
          <span className="font-medium text-slate-700 dark:text-neutral-300">
            {t('contentTools.tools.explain_it_back.editor.maxWords')}
          </span>
          <input
            type="number"
            min={10}
            max={1000}
            className="w-full rounded border border-slate-300 bg-white px-2 py-1.5 text-sm dark:border-neutral-600 dark:bg-neutral-950"
            disabled={disabled}
            value={typeof value.maxWords === 'number' ? value.maxWords : 150}
            onChange={(e) => patch({ maxWords: Number(e.target.value) || 150 })}
          />
        </label>
      </div>

      <div className="space-y-2">
        <div className="text-xs font-medium text-slate-700 dark:text-neutral-300">
          {t('contentTools.tools.explain_it_back.editor.keyPoints')}
        </div>
        {keyPoints.map((kp, idx) => (
          <div
            key={kp.id}
            className="space-y-1 rounded border border-slate-200 p-2 dark:border-neutral-700"
          >
            <div className="flex items-center justify-between gap-2">
              <span className="text-xs font-medium">
                {t('contentTools.tools.explain_it_back.editor.keyPointN', { n: idx + 1 })}
              </span>
              <button
                type="button"
                className="text-xs text-rose-700 disabled:opacity-40"
                disabled={disabled || keyPoints.length <= 2}
                onClick={() => setKeyPoints(keyPoints.filter((x) => x.id !== kp.id))}
              >
                {t('contentTools.tools.explain_it_back.editor.remove')}
              </button>
            </div>
            <input
              className="w-full rounded border border-slate-300 bg-white px-2 py-1 text-sm dark:border-neutral-600 dark:bg-neutral-950"
              disabled={disabled}
              placeholder={t('contentTools.tools.explain_it_back.editor.label')}
              value={kp.label}
              onChange={(e) => {
                const copy = [...keyPoints]
                copy[idx] = { ...kp, label: e.target.value }
                setKeyPoints(copy)
              }}
            />
            <textarea
              className="w-full rounded border border-slate-300 bg-white px-2 py-1 text-sm dark:border-neutral-600 dark:bg-neutral-950"
              rows={2}
              disabled={disabled}
              placeholder={t('contentTools.tools.explain_it_back.editor.description')}
              value={kp.description}
              onChange={(e) => {
                const copy = [...keyPoints]
                copy[idx] = { ...kp, description: e.target.value }
                setKeyPoints(copy)
              }}
            />
          </div>
        ))}
        <button
          type="button"
          className="text-xs font-medium text-sky-700 disabled:opacity-40"
          disabled={disabled || keyPoints.length >= 6}
          onClick={() =>
            setKeyPoints([
              ...keyPoints,
              { id: newId('kp'), label: '', description: '' },
            ])
          }
        >
          {t('contentTools.tools.explain_it_back.editor.addKeyPoint')}
        </button>
      </div>

      <label className="block space-y-1 text-xs">
        <span className="font-medium text-slate-700 dark:text-neutral-300">
          {t('contentTools.tools.explain_it_back.editor.feedbackStyle')}
        </span>
        <select
          className="w-full rounded border border-slate-300 bg-white px-2 py-1.5 text-sm dark:border-neutral-600 dark:bg-neutral-950"
          disabled={disabled}
          value={feedbackStyle}
          onChange={(e) => patch({ feedbackStyle: e.target.value })}
        >
          <option value="encouraging">
            {t('contentTools.tools.explain_it_back.editor.styleEncouraging')}
          </option>
          <option value="neutral">
            {t('contentTools.tools.explain_it_back.editor.styleNeutral')}
          </option>
          <option value="socratic">
            {t('contentTools.tools.explain_it_back.editor.styleSocratic')}
          </option>
        </select>
      </label>

      <div className="grid grid-cols-2 gap-3 text-xs">
        <label className="flex items-center gap-2">
          <input
            type="checkbox"
            disabled={disabled}
            checked={value.aiFeedback !== false}
            onChange={(e) => patch({ aiFeedback: e.target.checked })}
          />
          {t('contentTools.tools.explain_it_back.editor.aiFeedback')}
        </label>
        <label className="flex items-center gap-2">
          <input
            type="checkbox"
            disabled={disabled}
            checked={value.revealKeyPointsAfterSubmit !== false}
            onChange={(e) => patch({ revealKeyPointsAfterSubmit: e.target.checked })}
          />
          {t('contentTools.tools.explain_it_back.editor.revealAfter')}
        </label>
        <label className="flex items-center gap-2">
          <input
            type="checkbox"
            disabled={disabled}
            checked={value.includeProbeQuestion !== false}
            onChange={(e) => patch({ includeProbeQuestion: e.target.checked })}
          />
          {t('contentTools.tools.explain_it_back.editor.includeProbe')}
        </label>
        <label className="flex items-center gap-2">
          <span>{t('contentTools.tools.explain_it_back.editor.attempts')}</span>
          <input
            type="number"
            min={1}
            max={10}
            className="w-16 rounded border border-slate-300 bg-white px-1 py-0.5 dark:border-neutral-600 dark:bg-neutral-950"
            disabled={disabled}
            value={typeof value.attempts === 'number' ? value.attempts : 3}
            onChange={(e) => patch({ attempts: Number(e.target.value) || 3 })}
          />
        </label>
      </div>

      <div className="space-y-2 rounded border border-dashed border-slate-300 p-3 dark:border-neutral-600">
        <div className="text-xs font-medium text-slate-700 dark:text-neutral-300">
          {t('contentTools.tools.explain_it_back.editor.testSample')}
        </div>
        <textarea
          className="w-full rounded border border-slate-300 bg-white px-2 py-1.5 text-sm dark:border-neutral-600 dark:bg-neutral-950"
          rows={3}
          disabled={disabled || sampleBusy}
          value={sample}
          onChange={(e) => setSample(e.target.value)}
          placeholder={t('contentTools.tools.explain_it_back.editor.samplePlaceholder')}
        />
        <button
          type="button"
          className="rounded bg-slate-800 px-2 py-1 text-xs font-medium text-white disabled:opacity-40 dark:bg-neutral-200 dark:text-neutral-900"
          disabled={disabled || sampleBusy || !courseCode || !instanceId || sample.trim().length < 10}
          onClick={() => void onTestSample()}
          data-testid="explain-it-back-test-sample"
        >
          {sampleBusy
            ? t('contentTools.tools.explain_it_back.editor.testing')
            : t('contentTools.tools.explain_it_back.editor.runSample')}
        </button>
        {!courseCode || !instanceId ? (
          <p className="text-xs text-slate-500">
            {t('contentTools.tools.explain_it_back.editor.testUnavailable')}
          </p>
        ) : null}
        {sampleError ? <p className="text-xs text-rose-700">{sampleError}</p> : null}
        {sampleFeedback ? (
          <pre className="whitespace-pre-wrap text-xs text-slate-700 dark:text-neutral-300">
            {sampleFeedback}
          </pre>
        ) : null}
      </div>
    </div>
  )
}
