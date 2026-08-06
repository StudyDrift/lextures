import { useId, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { runContentToolAction } from '../../../../lib/courses-api'

export type WorkedExampleEditorProps = {
  value: Record<string, unknown>
  onChange: (next: Record<string, unknown>) => void
  disabled?: boolean
  idPrefix?: string
  courseCode?: string
  instanceId?: string
}

type BlankType = 'numeric' | 'expression' | 'choice' | 'text'
type ChoiceOption = { id: string; text: string }
type Blank = {
  type: BlankType
  expected?: string | number
  tolerance?: { kind: 'absolute' | 'relative'; value: number }
  acceptedAnswers?: string[]
  options?: ChoiceOption[]
  correctOptionId?: string
  unit?: string
}
type Step = {
  id: string
  label?: string
  text: string
  blank?: Blank
  hints?: string[]
  explanation?: string
}

function newId(prefix: string): string {
  return `${prefix}_${Math.random().toString(36).slice(2, 9)}`
}

function asSteps(value: Record<string, unknown>): Step[] {
  return Array.isArray(value.steps) ? (value.steps as Step[]) : []
}

function asVariables(value: Record<string, unknown>): string[] {
  return Array.isArray(value.variables) ? (value.variables as string[]) : []
}

export function WorkedExampleEditor({
  value,
  onChange,
  disabled,
  idPrefix = 'we-editor',
  courseCode,
  instanceId,
}: WorkedExampleEditorProps) {
  const { t } = useTranslation('contentTools')
  const baseId = useId()
  const steps = asSteps(value)
  const [verifyMsg, setVerifyMsg] = useState<string | null>(null)

  function patch(partial: Record<string, unknown>) {
    onChange({ ...value, ...partial })
  }

  function setSteps(next: Step[]) {
    patch({ steps: next })
  }

  function updateStep(idx: number, next: Step) {
    const copy = [...steps]
    copy[idx] = next
    setSteps(copy)
  }

  function moveStep(idx: number, dir: -1 | 1) {
    const j = idx + dir
    if (j < 0 || j >= steps.length) return
    const copy = [...steps]
    ;[copy[idx], copy[j]] = [copy[j], copy[idx]]
    setSteps(copy)
  }

  function addStep() {
    setSteps([
      ...steps,
      {
        id: newId('step'),
        text: '',
        blank: { type: 'expression', expected: '' },
        hints: [],
        explanation: '',
      },
    ])
  }

  async function onVerify() {
    if (!courseCode || !instanceId) {
      setVerifyMsg(t('contentTools.tools.worked_example.editor.verifyUnavailable'))
      return
    }
    try {
      const res = await runContentToolAction(courseCode, instanceId, 'verify', {
        input: { config: value },
        idempotencyKey: crypto.randomUUID(),
      })
      const raw = (res.result ?? {}) as {
        ok?: boolean
        results?: Array<{ stepId: string; ok: boolean }>
      }
      if (raw.ok) {
        setVerifyMsg(t('contentTools.tools.worked_example.editor.verifyOk'))
      } else {
        const bad = (raw.results ?? []).filter((r) => !r.ok).map((r) => r.stepId)
        setVerifyMsg(
          t('contentTools.tools.worked_example.editor.verifyFail', {
            steps: bad.join(', ') || '—',
          }),
        )
      }
    } catch {
      setVerifyMsg(t('contentTools.tools.worked_example.editor.verifyFail', { steps: '—' }))
    }
  }

  return (
    <div className="space-y-4" data-testid="worked-example-editor">
      <div className="space-y-1">
        <label className="block text-sm font-medium" htmlFor={`${baseId}-title`}>
          {t('contentTools.tools.worked_example.editor.title')}
        </label>
        <input
          id={`${baseId}-title`}
          className="w-full rounded-md border px-3 py-2 text-sm"
          value={typeof value.title === 'string' ? value.title : ''}
          disabled={disabled}
          onChange={(e) => patch({ title: e.target.value })}
        />
      </div>

      <div className="space-y-1">
        <label className="block text-sm font-medium" htmlFor={`${baseId}-problem`}>
          {t('contentTools.tools.worked_example.editor.problem')}
        </label>
        <textarea
          id={`${baseId}-problem`}
          className="w-full rounded-md border px-3 py-2 text-sm"
          rows={3}
          value={typeof value.problem === 'string' ? value.problem : ''}
          disabled={disabled}
          onChange={(e) => patch({ problem: e.target.value })}
        />
      </div>

      <div className="space-y-1">
        <label className="block text-sm font-medium" htmlFor={`${baseId}-vars`}>
          {t('contentTools.tools.worked_example.editor.variables')}
        </label>
        <input
          id={`${baseId}-vars`}
          className="w-full rounded-md border px-3 py-2 text-sm"
          value={asVariables(value).join(', ')}
          disabled={disabled}
          placeholder="x, y"
          onChange={(e) =>
            patch({
              variables: e.target.value
                .split(',')
                .map((s) => s.trim())
                .filter(Boolean),
            })
          }
        />
      </div>

      <div className="grid gap-3 sm:grid-cols-2">
        <label className="flex flex-col gap-1 text-sm">
          {t('contentTools.tools.worked_example.editor.blankPolicy')}
          <select
            className="rounded-md border px-2 py-1.5"
            disabled={disabled}
            value={typeof value.blankPolicy === 'string' ? value.blankPolicy : 'author'}
            onChange={(e) => patch({ blankPolicy: e.target.value })}
          >
            <option value="author">{t('contentTools.tools.worked_example.editor.policyAuthor')}</option>
            <option value="progressive">
              {t('contentTools.tools.worked_example.editor.policyProgressive')}
            </option>
            <option value="all">{t('contentTools.tools.worked_example.editor.policyAll')}</option>
          </select>
        </label>
        <label className="flex flex-col gap-1 text-sm">
          {t('contentTools.tools.worked_example.editor.attempts')}
          <input
            type="number"
            min={1}
            max={10}
            className="rounded-md border px-2 py-1.5"
            disabled={disabled}
            value={typeof value.attemptsPerStep === 'number' ? value.attemptsPerStep : 3}
            onChange={(e) => patch({ attemptsPerStep: Number(e.target.value) || 3 })}
          />
        </label>
        <label className="flex items-center gap-2 text-sm">
          <input
            type="checkbox"
            disabled={disabled}
            checked={value.practiceOnly !== false}
            onChange={(e) => patch({ practiceOnly: e.target.checked })}
          />
          {t('contentTools.tools.worked_example.editor.practiceOnly')}
        </label>
        <label className="flex items-center gap-2 text-sm">
          <input
            type="checkbox"
            disabled={disabled}
            checked={value.showAllSteps === true}
            onChange={(e) => patch({ showAllSteps: e.target.checked })}
          />
          {t('contentTools.tools.worked_example.editor.showAllSteps')}
        </label>
        <label className="flex items-center gap-2 text-sm">
          <input
            type="checkbox"
            disabled={disabled}
            checked={value.allowRevealAll === true}
            onChange={(e) => patch({ allowRevealAll: e.target.checked })}
          />
          {t('contentTools.tools.worked_example.editor.allowRevealAll')}
        </label>
        <label className="flex items-center gap-2 text-sm">
          <input
            type="checkbox"
            disabled={disabled}
            checked={value.hintsAffectScore === true}
            onChange={(e) => patch({ hintsAffectScore: e.target.checked })}
          />
          {t('contentTools.tools.worked_example.editor.hintsAffectScore')}
        </label>
      </div>

      <div className="space-y-3">
        <div className="flex items-center justify-between">
          <h4 className="text-sm font-semibold">
            {t('contentTools.tools.worked_example.editor.steps')}
          </h4>
          <button
            type="button"
            className="rounded-md border px-2 py-1 text-sm"
            disabled={disabled || steps.length >= 20}
            onClick={addStep}
          >
            {t('contentTools.tools.worked_example.editor.addStep')}
          </button>
        </div>

        {steps.map((step, idx) => (
          <div
            key={step.id}
            className="space-y-2 rounded-md border border-[color:var(--lex-border)] p-3"
          >
            <div className="flex flex-wrap items-center justify-between gap-2">
              <span className="text-sm font-medium">
                {t('contentTools.tools.worked_example.editor.stepN', { n: idx + 1 })}
              </span>
              <div className="flex gap-1">
                <button
                  type="button"
                  className="rounded border px-2 py-0.5 text-xs"
                  disabled={disabled || idx === 0}
                  onClick={() => moveStep(idx, -1)}
                >
                  ↑
                </button>
                <button
                  type="button"
                  className="rounded border px-2 py-0.5 text-xs"
                  disabled={disabled || idx === steps.length - 1}
                  onClick={() => moveStep(idx, 1)}
                >
                  ↓
                </button>
                <button
                  type="button"
                  className="rounded border px-2 py-0.5 text-xs text-danger-fg"
                  disabled={disabled}
                  onClick={() => setSteps(steps.filter((_, i) => i !== idx))}
                >
                  {t('contentTools.tools.worked_example.editor.remove')}
                </button>
              </div>
            </div>

            <input
              className="w-full rounded-md border px-2 py-1.5 text-sm"
              placeholder={t('contentTools.tools.worked_example.editor.label')}
              disabled={disabled}
              value={step.label ?? ''}
              onChange={(e) => updateStep(idx, { ...step, label: e.target.value })}
            />
            <textarea
              className="w-full rounded-md border px-2 py-1.5 text-sm"
              rows={2}
              placeholder={t('contentTools.tools.worked_example.editor.stepText')}
              disabled={disabled}
              value={step.text}
              onChange={(e) => updateStep(idx, { ...step, text: e.target.value })}
            />

            <label className="flex items-center gap-2 text-sm">
              <input
                type="checkbox"
                disabled={disabled}
                checked={!!step.blank}
                onChange={(e) =>
                  updateStep(idx, {
                    ...step,
                    blank: e.target.checked
                      ? { type: 'expression', expected: '' }
                      : undefined,
                  })
                }
              />
              {t('contentTools.tools.worked_example.editor.blanked')}
            </label>

            {step.blank && (
              <div className="grid gap-2 sm:grid-cols-2">
                <label className="flex flex-col gap-1 text-sm">
                  {t('contentTools.tools.worked_example.editor.blankType')}
                  <select
                    className="rounded-md border px-2 py-1.5"
                    disabled={disabled}
                    value={step.blank.type}
                    onChange={(e) =>
                      updateStep(idx, {
                        ...step,
                        blank: {
                          ...step.blank!,
                          type: e.target.value as BlankType,
                          options:
                            e.target.value === 'choice'
                              ? step.blank?.options ?? [
                                  { id: newId('opt'), text: '' },
                                  { id: newId('opt'), text: '' },
                                ]
                              : undefined,
                        },
                      })
                    }
                  >
                    <option value="expression">expression</option>
                    <option value="numeric">numeric</option>
                    <option value="choice">choice</option>
                    <option value="text">text</option>
                  </select>
                </label>

                {step.blank.type !== 'choice' && (
                  <label className="flex flex-col gap-1 text-sm">
                    {t('contentTools.tools.worked_example.editor.expected')}
                    <input
                      className="rounded-md border px-2 py-1.5"
                      disabled={disabled}
                      value={
                        step.blank.expected === undefined || step.blank.expected === null
                          ? ''
                          : String(step.blank.expected)
                      }
                      onChange={(e) => {
                        const raw = e.target.value
                        const expected =
                          step.blank?.type === 'numeric' && raw !== '' && !Number.isNaN(Number(raw))
                            ? Number(raw)
                            : raw
                        updateStep(idx, {
                          ...step,
                          blank: { ...step.blank!, expected },
                        })
                      }}
                    />
                  </label>
                )}

                {step.blank.type === 'choice' && (
                  <div className="sm:col-span-2 space-y-2">
                    {(step.blank.options ?? []).map((opt, oi) => (
                      <div key={opt.id} className="flex gap-2">
                        <input
                          className="flex-1 rounded-md border px-2 py-1.5 text-sm"
                          disabled={disabled}
                          value={opt.text}
                          placeholder={t('contentTools.tools.worked_example.editor.optionText')}
                          onChange={(e) => {
                            const options = [...(step.blank?.options ?? [])]
                            options[oi] = { ...opt, text: e.target.value }
                            updateStep(idx, {
                              ...step,
                              blank: { ...step.blank!, options },
                            })
                          }}
                        />
                        <label className="flex items-center gap-1 text-xs">
                          <input
                            type="radio"
                            name={`${idPrefix}-correct-${step.id}`}
                            disabled={disabled}
                            checked={step.blank?.correctOptionId === opt.id}
                            onChange={() =>
                              updateStep(idx, {
                                ...step,
                                blank: { ...step.blank!, correctOptionId: opt.id },
                              })
                            }
                          />
                          {t('contentTools.tools.worked_example.editor.correct')}
                        </label>
                      </div>
                    ))}
                  </div>
                )}

                {step.blank.type === 'text' && (
                  <label className="sm:col-span-2 flex flex-col gap-1 text-sm">
                    {t('contentTools.tools.worked_example.editor.acceptedAnswers')}
                    <input
                      className="rounded-md border px-2 py-1.5"
                      disabled={disabled}
                      value={(step.blank.acceptedAnswers ?? []).join(', ')}
                      onChange={(e) =>
                        updateStep(idx, {
                          ...step,
                          blank: {
                            ...step.blank!,
                            acceptedAnswers: e.target.value
                              .split(',')
                              .map((s) => s.trim())
                              .filter(Boolean),
                          },
                        })
                      }
                    />
                  </label>
                )}
              </div>
            )}

            <label className="flex flex-col gap-1 text-sm">
              {t('contentTools.tools.worked_example.editor.hints')}
              <input
                className="rounded-md border px-2 py-1.5"
                disabled={disabled}
                placeholder={t('contentTools.tools.worked_example.editor.hintsPlaceholder')}
                value={(step.hints ?? []).join(' | ')}
                onChange={(e) =>
                  updateStep(idx, {
                    ...step,
                    hints: e.target.value
                      .split('|')
                      .map((s) => s.trim())
                      .filter(Boolean)
                      .slice(0, 3),
                  })
                }
              />
            </label>

            <label className="flex flex-col gap-1 text-sm">
              {t('contentTools.tools.worked_example.editor.explanation')}
              <textarea
                className="rounded-md border px-2 py-1.5"
                rows={2}
                disabled={disabled}
                value={step.explanation ?? ''}
                onChange={(e) => updateStep(idx, { ...step, explanation: e.target.value })}
              />
            </label>
          </div>
        ))}
      </div>

      <div className="flex flex-wrap items-center gap-2">
        <button
          type="button"
          className="rounded-md border px-3 py-1.5 text-sm"
          disabled={disabled}
          onClick={() => void onVerify()}
          data-testid="worked-example-verify"
        >
          {t('contentTools.tools.worked_example.editor.verify')}
        </button>
        {verifyMsg && <span className="text-sm text-[color:var(--lex-muted)]">{verifyMsg}</span>}
      </div>
    </div>
  )
}
