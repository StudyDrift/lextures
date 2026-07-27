import { useId } from 'react'
import { useTranslation } from 'react-i18next'

export type MediaCheckpointsEditorProps = {
  value: Record<string, unknown>
  onChange: (next: Record<string, unknown>) => void
  disabled?: boolean
  idPrefix?: string
}

type Option = { id: string; text: string; correct?: boolean; feedback?: string }
type Question = {
  type: string
  prompt: string
  options?: Option[]
  acceptedAnswers?: string[]
  correctValue?: number
  tolerance?: { kind: 'absolute' | 'relative'; value: number }
}
type Checkpoint = {
  id: string
  atSec: number
  question: Question
  required?: boolean
  attempts?: number
  showFeedback?: boolean
}
type MediaRef = {
  source?: string
  fileId?: string
  kind?: string
  durationSec?: number
  url?: string
  captionUrl?: string
}

const QUESTION_TYPES = ['single', 'multi', 'true_false', 'short_text', 'numeric'] as const

function newId(prefix: string): string {
  return `${prefix}_${Math.random().toString(36).slice(2, 9)}`
}

function asCheckpoints(value: Record<string, unknown>): Checkpoint[] {
  return Array.isArray(value.checkpoints) ? (value.checkpoints as Checkpoint[]) : []
}

function asMedia(value: Record<string, unknown>): MediaRef {
  const m = value.media && typeof value.media === 'object' ? (value.media as MediaRef) : {}
  return {
    source: m.source || 'course_file',
    fileId: m.fileId || '',
    kind: m.kind || 'video',
    durationSec: typeof m.durationSec === 'number' ? m.durationSec : 60,
    url: m.url || '',
    captionUrl: m.captionUrl || '',
  }
}

function defaultOptions(type: string): Option[] {
  if (type === 'true_false') {
    return [
      { id: 'true', text: 'True', correct: true },
      { id: 'false', text: 'False', correct: false },
    ]
  }
  return [
    { id: newId('opt'), text: '', correct: true },
    { id: newId('opt'), text: '', correct: false },
  ]
}

export function MediaCheckpointsEditor({
  value,
  onChange,
  disabled,
  idPrefix = 'mc-editor',
}: MediaCheckpointsEditorProps) {
  const { t } = useTranslation('contentTools')
  const baseId = useId()
  const media = asMedia(value)
  const checkpoints = asCheckpoints(value)
  const missingCaptions = !media.captionUrl && !value.captionsTrackId
  const missingTranscript = !String(value.transcriptMarkdown ?? '').trim()

  function patch(partial: Record<string, unknown>) {
    onChange({ ...value, ...partial })
  }

  function setMedia(next: MediaRef) {
    patch({ media: next })
  }

  function setCheckpoints(next: Checkpoint[]) {
    patch({ checkpoints: next.slice(0, 40) })
  }

  function updateCheckpoint(idx: number, next: Checkpoint) {
    const copy = [...checkpoints]
    copy[idx] = next
    setCheckpoints(copy)
  }

  function addCheckpoint() {
    setCheckpoints([
      ...checkpoints,
      {
        id: newId('cp'),
        atSec: 0,
        required: true,
        attempts: 2,
        showFeedback: true,
        question: {
          type: 'single',
          prompt: '',
          options: defaultOptions('single'),
        },
      },
    ])
  }

  return (
    <div className="space-y-4" data-testid="media-checkpoints-editor">
      <fieldset className="space-y-2 rounded-md border border-slate-200 p-3 dark:border-neutral-700">
        <legend className="px-1 text-xs font-semibold text-slate-700 dark:text-neutral-200">
          {t('contentTools.tools.media_checkpoints.editor.media')}
        </legend>
        <div className="grid gap-2 sm:grid-cols-2">
          <label className="block space-y-1 text-xs">
            <span className="font-medium">{t('contentTools.tools.media_checkpoints.editor.fileId')}</span>
            <input
              id={`${idPrefix}-${baseId}-file`}
              disabled={disabled}
              value={media.fileId}
              onChange={(e) => setMedia({ ...media, fileId: e.target.value })}
              className="w-full rounded-md border border-slate-200 bg-white px-2 py-1.5 text-sm dark:border-neutral-600 dark:bg-neutral-950"
            />
          </label>
          <label className="block space-y-1 text-xs">
            <span className="font-medium">{t('contentTools.tools.media_checkpoints.editor.kind')}</span>
            <select
              disabled={disabled}
              value={media.kind}
              onChange={(e) => setMedia({ ...media, kind: e.target.value })}
              className="w-full rounded-md border border-slate-200 bg-white px-2 py-1.5 text-sm dark:border-neutral-600 dark:bg-neutral-950"
            >
              <option value="video">video</option>
              <option value="audio">audio</option>
            </select>
          </label>
          <label className="block space-y-1 text-xs sm:col-span-2">
            <span className="font-medium">{t('contentTools.tools.media_checkpoints.editor.url')}</span>
            <input
              disabled={disabled}
              value={media.url}
              onChange={(e) => setMedia({ ...media, url: e.target.value })}
              className="w-full rounded-md border border-slate-200 bg-white px-2 py-1.5 text-sm dark:border-neutral-600 dark:bg-neutral-950"
            />
          </label>
          <label className="block space-y-1 text-xs">
            <span className="font-medium">
              {t('contentTools.tools.media_checkpoints.editor.durationSec')}
            </span>
            <input
              type="number"
              min={1}
              disabled={disabled}
              value={media.durationSec}
              onChange={(e) => setMedia({ ...media, durationSec: Number(e.target.value) || 1 })}
              className="w-full rounded-md border border-slate-200 bg-white px-2 py-1.5 text-sm dark:border-neutral-600 dark:bg-neutral-950"
            />
          </label>
          <label className="block space-y-1 text-xs">
            <span className="font-medium">
              {t('contentTools.tools.media_checkpoints.editor.captionUrl')}
            </span>
            <input
              disabled={disabled}
              value={media.captionUrl}
              onChange={(e) => setMedia({ ...media, captionUrl: e.target.value })}
              className="w-full rounded-md border border-slate-200 bg-white px-2 py-1.5 text-sm dark:border-neutral-600 dark:bg-neutral-950"
            />
          </label>
        </div>
        {missingCaptions ? (
          <p className="text-xs text-amber-700 dark:text-amber-300" role="status">
            {t('contentTools.tools.media_checkpoints.editor.captionsWarning')}
          </p>
        ) : null}
      </fieldset>

      <label className="block space-y-1 text-xs">
        <span className="font-medium text-slate-700 dark:text-neutral-300">
          {t('contentTools.tools.media_checkpoints.editor.transcript')}
        </span>
        <textarea
          disabled={disabled}
          rows={4}
          value={typeof value.transcriptMarkdown === 'string' ? value.transcriptMarkdown : ''}
          onChange={(e) =>
            patch({ transcriptMarkdown: e.target.value, transcriptSource: 'inline' })
          }
          placeholder={'0:00 Intro\n0:30 Concept'}
          className="w-full rounded-md border border-slate-200 bg-white px-2 py-1.5 text-sm dark:border-neutral-600 dark:bg-neutral-950"
        />
        {missingTranscript ? (
          <span className="text-amber-700 dark:text-amber-300">
            {t('contentTools.tools.media_checkpoints.editor.transcriptWarning')}
          </span>
        ) : null}
      </label>

      <div className="flex flex-wrap gap-4 text-xs">
        <label className="flex min-h-11 items-center gap-2">
          <input
            type="checkbox"
            disabled={disabled}
            checked={value.preventSkipPastUnanswered === true}
            onChange={(e) => patch({ preventSkipPastUnanswered: e.target.checked })}
          />
          {t('contentTools.tools.media_checkpoints.editor.preventSkip')}
        </label>
        <label className="flex min-h-11 items-center gap-2">
          <input
            type="checkbox"
            disabled={disabled}
            checked={value.practiceOnly !== false}
            onChange={(e) => patch({ practiceOnly: e.target.checked })}
          />
          {t('contentTools.tools.media_checkpoints.editor.practiceOnly')}
        </label>
      </div>

      <div className="space-y-3">
        <div className="flex items-center justify-between">
          <h3 className="text-xs font-semibold text-slate-700 dark:text-neutral-200">
            {t('contentTools.tools.media_checkpoints.editor.checkpoints')}
          </h3>
          <button
            type="button"
            disabled={disabled || checkpoints.length >= 40}
            onClick={addCheckpoint}
            className="rounded-md border border-slate-300 px-2 py-1 text-xs dark:border-neutral-600"
          >
            {t('contentTools.tools.media_checkpoints.editor.addCheckpoint')}
          </button>
        </div>

        {checkpoints.map((cp, idx) => (
          <div
            key={cp.id}
            className="space-y-2 rounded-md border border-slate-200 p-3 dark:border-neutral-700"
          >
            <div className="flex items-center justify-between gap-2">
              <span className="text-xs font-medium">
                {t('contentTools.tools.media_checkpoints.editor.checkpointN', { n: idx + 1 })}
              </span>
              <button
                type="button"
                disabled={disabled}
                onClick={() => setCheckpoints(checkpoints.filter((_, i) => i !== idx))}
                className="text-xs text-rose-700 dark:text-rose-300"
              >
                {t('contentTools.tools.media_checkpoints.editor.remove')}
              </button>
            </div>
            <div className="grid gap-2 sm:grid-cols-3">
              <label className="space-y-1 text-xs">
                <span>{t('contentTools.tools.media_checkpoints.editor.atSec')}</span>
                <input
                  type="number"
                  min={0}
                  step={0.1}
                  disabled={disabled}
                  value={cp.atSec}
                  onChange={(e) =>
                    updateCheckpoint(idx, { ...cp, atSec: Number(e.target.value) || 0 })
                  }
                  className="w-full rounded-md border border-slate-200 px-2 py-1.5 dark:border-neutral-600 dark:bg-neutral-950"
                />
              </label>
              <label className="space-y-1 text-xs">
                <span>{t('contentTools.tools.media_checkpoints.editor.attempts')}</span>
                <input
                  type="number"
                  min={1}
                  max={10}
                  disabled={disabled}
                  value={cp.attempts ?? 2}
                  onChange={(e) =>
                    updateCheckpoint(idx, { ...cp, attempts: Number(e.target.value) || 2 })
                  }
                  className="w-full rounded-md border border-slate-200 px-2 py-1.5 dark:border-neutral-600 dark:bg-neutral-950"
                />
              </label>
              <label className="space-y-1 text-xs">
                <span>{t('contentTools.tools.media_checkpoints.editor.type')}</span>
                <select
                  disabled={disabled}
                  value={cp.question.type}
                  onChange={(e) => {
                    const type = e.target.value
                    updateCheckpoint(idx, {
                      ...cp,
                      question: {
                        ...cp.question,
                        type,
                        options:
                          type === 'single' || type === 'multi' || type === 'true_false'
                            ? defaultOptions(type)
                            : undefined,
                      },
                    })
                  }}
                  className="w-full rounded-md border border-slate-200 px-2 py-1.5 dark:border-neutral-600 dark:bg-neutral-950"
                >
                  {QUESTION_TYPES.map((qt) => (
                    <option key={qt} value={qt}>
                      {qt}
                    </option>
                  ))}
                </select>
              </label>
            </div>
            <label className="block space-y-1 text-xs">
              <span>{t('contentTools.tools.media_checkpoints.editor.prompt')}</span>
              <textarea
                disabled={disabled}
                rows={2}
                value={cp.question.prompt}
                onChange={(e) =>
                  updateCheckpoint(idx, {
                    ...cp,
                    question: { ...cp.question, prompt: e.target.value },
                  })
                }
                className="w-full rounded-md border border-slate-200 px-2 py-1.5 dark:border-neutral-600 dark:bg-neutral-950"
              />
            </label>
            {(cp.question.type === 'single' ||
              cp.question.type === 'multi' ||
              cp.question.type === 'true_false') && (
              <div className="space-y-2">
                {(cp.question.options ?? []).map((opt, oi) => (
                  <div key={opt.id} className="grid grid-cols-[1fr_auto] gap-2 text-xs">
                    <input
                      disabled={disabled}
                      value={opt.text}
                      placeholder={t('contentTools.tools.media_checkpoints.editor.optionText')}
                      onChange={(e) => {
                        const options = [...(cp.question.options ?? [])]
                        options[oi] = { ...opt, text: e.target.value }
                        updateCheckpoint(idx, {
                          ...cp,
                          question: { ...cp.question, options },
                        })
                      }}
                      className="rounded-md border border-slate-200 px-2 py-1.5 dark:border-neutral-600 dark:bg-neutral-950"
                    />
                    <label className="flex items-center gap-1">
                      <input
                        type="checkbox"
                        disabled={disabled}
                        checked={opt.correct === true}
                        onChange={(e) => {
                          const options = [...(cp.question.options ?? [])]
                          if (cp.question.type === 'single' || cp.question.type === 'true_false') {
                            for (let i = 0; i < options.length; i++) {
                              options[i] = { ...options[i]!, correct: i === oi && e.target.checked }
                            }
                          } else {
                            options[oi] = { ...opt, correct: e.target.checked }
                          }
                          updateCheckpoint(idx, {
                            ...cp,
                            question: { ...cp.question, options },
                          })
                        }}
                      />
                      {t('contentTools.tools.media_checkpoints.editor.correct')}
                    </label>
                  </div>
                ))}
              </div>
            )}
            {cp.question.type === 'short_text' ? (
              <label className="block space-y-1 text-xs">
                <span>{t('contentTools.tools.media_checkpoints.editor.acceptedAnswers')}</span>
                <input
                  disabled={disabled}
                  value={(cp.question.acceptedAnswers ?? []).join(', ')}
                  onChange={(e) =>
                    updateCheckpoint(idx, {
                      ...cp,
                      question: {
                        ...cp.question,
                        acceptedAnswers: e.target.value
                          .split(',')
                          .map((s) => s.trim())
                          .filter(Boolean),
                      },
                    })
                  }
                  className="w-full rounded-md border border-slate-200 px-2 py-1.5 dark:border-neutral-600 dark:bg-neutral-950"
                />
              </label>
            ) : null}
            {cp.question.type === 'numeric' ? (
              <label className="block space-y-1 text-xs">
                <span>{t('contentTools.tools.media_checkpoints.editor.correctValue')}</span>
                <input
                  type="number"
                  disabled={disabled}
                  value={cp.question.correctValue ?? 0}
                  onChange={(e) =>
                    updateCheckpoint(idx, {
                      ...cp,
                      question: {
                        ...cp.question,
                        correctValue: Number(e.target.value),
                        tolerance: cp.question.tolerance ?? { kind: 'absolute', value: 0.01 },
                      },
                    })
                  }
                  className="w-full rounded-md border border-slate-200 px-2 py-1.5 dark:border-neutral-600 dark:bg-neutral-950"
                />
              </label>
            ) : null}
            <div className="flex flex-wrap gap-3 text-xs">
              <label className="flex items-center gap-1">
                <input
                  type="checkbox"
                  disabled={disabled}
                  checked={cp.required !== false}
                  onChange={(e) => updateCheckpoint(idx, { ...cp, required: e.target.checked })}
                />
                {t('contentTools.tools.media_checkpoints.editor.required')}
              </label>
              <label className="flex items-center gap-1">
                <input
                  type="checkbox"
                  disabled={disabled}
                  checked={cp.showFeedback !== false}
                  onChange={(e) => updateCheckpoint(idx, { ...cp, showFeedback: e.target.checked })}
                />
                {t('contentTools.tools.media_checkpoints.editor.showFeedback')}
              </label>
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}
