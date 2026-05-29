import { useEffect, useId, useRef, useState } from 'react'
import { atRiskI18n } from '../../lib/at-risk-i18n'
import {
  fetchCourseAtRiskConfig,
  saveCourseAtRiskConfig,
  type AtRiskCourseConfig,
} from '../../lib/courses-api'

type AtRiskRunConfigDialogProps = {
  open: boolean
  courseCode: string
  onClose: () => void
  onRunReport: (form: ConfigForm) => Promise<void>
}

export type AtRiskConfigForm = ConfigForm

type ConfigForm = Omit<AtRiskCourseConfig, 'courseOverride'>

function configToForm(config: AtRiskCourseConfig): ConfigForm {
  return {
    threshold: config.threshold,
    weightMissing: config.weightMissing,
    weightQuiz: config.weightQuiz,
    weightInactive: config.weightInactive,
    weightTrend: config.weightTrend,
    quizAvgThreshold: config.quizAvgThreshold,
    inactiveDaysThreshold: config.inactiveDaysThreshold,
    missingPctThreshold: config.missingPctThreshold,
  }
}

function weightSumPct(form: ConfigForm): number {
  return Math.round(
    (form.weightMissing + form.weightQuiz + form.weightInactive + form.weightTrend) * 1000,
  ) / 10
}

function ThresholdField({
  id,
  label,
  help,
  value,
  min,
  max,
  step,
  suffix,
  onChange,
  disabled,
}: {
  id: string
  label: string
  help: string
  value: number
  min: number
  max: number
  step?: number
  suffix?: string
  onChange: (value: number) => void
  disabled?: boolean
}) {
  return (
    <div>
      <label htmlFor={id} className="text-sm font-medium text-slate-800 dark:text-neutral-100">
        {label}
      </label>
      {help ? (
        <p className="mt-0.5 text-xs text-slate-500 dark:text-neutral-400">{help}</p>
      ) : null}
      <div className="mt-2 flex items-center gap-2">
        <input
          id={id}
          type="number"
          min={min}
          max={max}
          step={step ?? 1}
          value={value}
          disabled={disabled}
          onChange={(e) => onChange(Number(e.target.value))}
          className="w-full rounded-xl border border-slate-200 bg-white px-3 py-2 text-sm tabular-nums dark:border-neutral-600 dark:bg-neutral-950 dark:text-neutral-100"
        />
        {suffix ? (
          <span className="shrink-0 text-sm text-slate-500 dark:text-neutral-400">{suffix}</span>
        ) : null}
      </div>
    </div>
  )
}

export function AtRiskRunConfigDialog({
  open,
  courseCode,
  onClose,
  onRunReport,
}: AtRiskRunConfigDialogProps) {
  const titleId = useId()
  const descId = useId()
  const cancelRef = useRef<HTMLButtonElement>(null)
  const [form, setForm] = useState<ConfigForm | null>(null)
  const [courseOverride, setCourseOverride] = useState(false)
  const [loading, setLoading] = useState(false)
  const [saving, setSaving] = useState(false)
  const [running, setRunning] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [saveNotice, setSaveNotice] = useState<string | null>(null)

  useEffect(() => {
    if (!open) return
    setError(null)
    setSaveNotice(null)
    setLoading(true)
    setForm(null)
    void fetchCourseAtRiskConfig(courseCode)
      .then((config) => {
        setForm(configToForm(config))
        setCourseOverride(config.courseOverride)
      })
      .catch((e) => {
        setError(e instanceof Error ? e.message : atRiskI18n.configLoadFailed)
      })
      .finally(() => setLoading(false))
  }, [open, courseCode])

  useEffect(() => {
    if (!open) return
    const t = window.setTimeout(() => cancelRef.current?.focus(), 0)
    return () => window.clearTimeout(t)
  }, [open])

  useEffect(() => {
    if (!open) return
    function onKey(e: KeyboardEvent) {
      if (e.key === 'Escape' && !saving && !running) {
        e.preventDefault()
        onClose()
      }
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [open, saving, running, onClose])

  if (!open) return null

  const busy = loading || saving || running
  const weightsOk = form != null && Math.abs(weightSumPct(form) - 100) < 0.2

  function patchForm(patch: Partial<ConfigForm>) {
    setForm((prev) => (prev ? { ...prev, ...patch } : prev))
    setSaveNotice(null)
  }

  async function handleSave() {
    if (!form) return
    setSaving(true)
    setError(null)
    setSaveNotice(null)
    try {
      const saved = await saveCourseAtRiskConfig(courseCode, form)
      setForm(configToForm(saved))
      setCourseOverride(saved.courseOverride)
      setSaveNotice(atRiskI18n.configSaved)
    } catch (e) {
      setError(e instanceof Error ? e.message : atRiskI18n.configSaveFailed)
    } finally {
      setSaving(false)
    }
  }

  async function handleRunReport() {
    if (!form || !weightsOk) return
    setRunning(true)
    setError(null)
    setSaveNotice(null)
    try {
      await onRunReport(form)
      onClose()
    } catch (e) {
      setError(e instanceof Error ? e.message : atRiskI18n.runScoringFailed)
    } finally {
      setRunning(false)
    }
  }

  return (
    <div className="fixed inset-0 z-[400] flex items-center justify-center p-4" role="presentation">
      <button
        type="button"
        aria-label="Close dialog"
        disabled={busy}
        className="absolute inset-0 cursor-default border-0 bg-black/45 p-0 disabled:cursor-not-allowed"
        onClick={() => {
          if (!busy) onClose()
        }}
      />
      <div
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        aria-describedby={descId}
        className="relative z-10 flex max-h-[min(90vh,720px)] w-full max-w-lg flex-col overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-xl dark:border-neutral-700 dark:bg-neutral-900"
      >
        <div className="border-b border-slate-200 px-5 py-4 dark:border-neutral-700">
          <h2 id={titleId} className="text-lg font-semibold text-slate-950 dark:text-neutral-100">
            {atRiskI18n.configTitle}
          </h2>
          <p id={descId} className="mt-1 text-sm text-slate-600 dark:text-neutral-300">
            {atRiskI18n.configDescription}
          </p>
          {form && !loading ? (
            <p className="mt-2 text-xs text-slate-500 dark:text-neutral-400">
              {courseOverride ? atRiskI18n.courseOverrideNote : atRiskI18n.institutionDefaultsNote}
            </p>
          ) : null}
        </div>

        <div className="flex-1 overflow-y-auto px-5 py-4">
          {loading && (
            <p className="text-sm text-slate-500 dark:text-neutral-400">{atRiskI18n.configLoading}</p>
          )}
          {error && (
            <p className="mb-4 rounded-xl border border-rose-200 bg-rose-50 px-3 py-2 text-sm text-rose-800 dark:border-rose-900/40 dark:bg-rose-950/40 dark:text-rose-100">
              {error}
            </p>
          )}
          {saveNotice && (
            <p className="mb-4 rounded-xl border border-emerald-200 bg-emerald-50 px-3 py-2 text-sm text-emerald-800 dark:border-emerald-900/40 dark:bg-emerald-950/40 dark:text-emerald-100">
              {saveNotice}
            </p>
          )}
          {form && !loading ? (
            <div className="space-y-5">
              <ThresholdField
                id="at-risk-threshold"
                label={atRiskI18n.thresholdLabel}
                help={atRiskI18n.thresholdHelp}
                value={form.threshold}
                min={0}
                max={100}
                onChange={(threshold) => patchForm({ threshold })}
                disabled={busy}
              />
              <ThresholdField
                id="at-risk-inactive-days"
                label={atRiskI18n.inactiveDaysLabel}
                help={atRiskI18n.inactiveDaysHelp}
                value={form.inactiveDaysThreshold}
                min={1}
                max={90}
                suffix="days"
                onChange={(inactiveDaysThreshold) => patchForm({ inactiveDaysThreshold })}
                disabled={busy}
              />
              <ThresholdField
                id="at-risk-missing-pct"
                label={atRiskI18n.missingPctLabel}
                help={atRiskI18n.missingPctHelp}
                value={form.missingPctThreshold}
                min={1}
                max={100}
                suffix="%"
                onChange={(missingPctThreshold) => patchForm({ missingPctThreshold })}
                disabled={busy}
              />
              <ThresholdField
                id="at-risk-quiz-avg"
                label={atRiskI18n.quizAvgLabel}
                help={atRiskI18n.quizAvgHelp}
                value={form.quizAvgThreshold}
                min={0}
                max={100}
                suffix="%"
                onChange={(quizAvgThreshold) => patchForm({ quizAvgThreshold })}
                disabled={busy}
              />

              <div className="border-t border-slate-200 pt-5 dark:border-neutral-700">
                <h3 className="text-sm font-semibold text-slate-900 dark:text-neutral-100">
                  {atRiskI18n.weightsSection}
                </h3>
                <p className="mt-0.5 text-xs text-slate-500 dark:text-neutral-400">
                  {atRiskI18n.weightsHelp}
                </p>
                <div className="mt-3 grid gap-4 sm:grid-cols-2">
                  <ThresholdField
                    id="at-risk-weight-missing"
                    label={atRiskI18n.weightMissingLabel}
                    help=""
                    value={Math.round(form.weightMissing * 1000) / 10}
                    min={0}
                    max={100}
                    step={0.1}
                    suffix="%"
                    onChange={(pct) => patchForm({ weightMissing: pct / 100 })}
                    disabled={busy}
                  />
                  <ThresholdField
                    id="at-risk-weight-quiz"
                    label={atRiskI18n.weightQuizLabel}
                    help=""
                    value={Math.round(form.weightQuiz * 1000) / 10}
                    min={0}
                    max={100}
                    step={0.1}
                    suffix="%"
                    onChange={(pct) => patchForm({ weightQuiz: pct / 100 })}
                    disabled={busy}
                  />
                  <ThresholdField
                    id="at-risk-weight-inactive"
                    label={atRiskI18n.weightInactiveLabel}
                    help=""
                    value={Math.round(form.weightInactive * 1000) / 10}
                    min={0}
                    max={100}
                    step={0.1}
                    suffix="%"
                    onChange={(pct) => patchForm({ weightInactive: pct / 100 })}
                    disabled={busy}
                  />
                  <ThresholdField
                    id="at-risk-weight-trend"
                    label={atRiskI18n.weightTrendLabel}
                    help=""
                    value={Math.round(form.weightTrend * 1000) / 10}
                    min={0}
                    max={100}
                    step={0.1}
                    suffix="%"
                    onChange={(pct) => patchForm({ weightTrend: pct / 100 })}
                    disabled={busy}
                  />
                </div>
                {!weightsOk ? (
                  <p className="mt-2 text-xs text-amber-700 dark:text-amber-300">
                    Weights currently sum to {weightSumPct(form!)}% — adjust to total 100%.
                  </p>
                ) : null}
              </div>
            </div>
          ) : null}
        </div>

        <div className="flex flex-wrap justify-end gap-2 border-t border-slate-200 px-5 py-4 dark:border-neutral-700">
          <button
            ref={cancelRef}
            type="button"
            disabled={busy}
            onClick={onClose}
            className="rounded-xl border border-slate-200 bg-white px-4 py-2 text-sm font-semibold text-slate-800 shadow-sm hover:bg-slate-50 disabled:opacity-60 dark:border-neutral-600 dark:bg-neutral-900 dark:text-neutral-100 dark:hover:bg-neutral-800"
          >
            Cancel
          </button>
          <button
            type="button"
            disabled={busy || !form || !weightsOk}
            onClick={() => void handleSave()}
            className="rounded-xl border border-slate-200 bg-white px-4 py-2 text-sm font-semibold text-slate-800 shadow-sm hover:bg-slate-50 disabled:cursor-not-allowed disabled:opacity-50 dark:border-neutral-600 dark:bg-neutral-900 dark:text-neutral-100 dark:hover:bg-neutral-800"
          >
            {saving ? atRiskI18n.configSaving : atRiskI18n.configSave}
          </button>
          <button
            type="button"
            disabled={busy || !form || !weightsOk}
            onClick={() => void handleRunReport()}
            className="rounded-xl bg-indigo-600 px-4 py-2 text-sm font-semibold text-white shadow-sm hover:bg-indigo-500 disabled:cursor-not-allowed disabled:opacity-50"
          >
            {running ? atRiskI18n.configRunningReport : atRiskI18n.configRunReport}
          </button>
        </div>
      </div>
    </div>
  )
}
