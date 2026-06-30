import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { Loader2, RefreshCw, Save } from 'lucide-react'
import { ProvenanceBadge } from '../../components/ai/provenance-badge'
import { usePlatformFeatures } from '../../context/platform-features-context'
import {
  fetchLessonGeneratorJob,
  regenerateLessonComponent,
  saveLessonPackageToCourse,
  startLessonGeneration,
  type LessonComponentSlot,
  type LessonGeneratorInput,
  type LessonPackage,
} from '../../lib/courses-api'
import { toastMutationError, toastSaveOk } from '../../lib/lms-toast'
import { LmsPage } from './lms-page'

const DIFFERENTIATION_OPTIONS = [
  { value: 'below_grade', label: 'Below grade' },
  { value: 'on_grade', label: 'On grade' },
  { value: 'advanced', label: 'Advanced' },
  { value: 'ell', label: 'ELL' },
  { value: 'iep', label: 'IEP' },
] as const

const COMPONENT_LABELS: Record<string, string> = {
  lesson_plan: 'Lesson plan',
  quiz: 'Formative quiz',
  rubric: 'Rubric',
  activity_below_grade: 'Activity (below grade)',
  activity_on_grade: 'Activity (on grade)',
  activity_advanced: 'Activity (advanced)',
  activity_ell: 'Activity (ELL)',
  activity_iep: 'Activity (IEP)',
}

type WizardStep = 'input' | 'generating' | 'review' | 'saved'

function componentMarkdown(slot: LessonComponentSlot): string {
  if (!slot.content) return ''
  const c = slot.content as { markdown?: string }
  return c.markdown ?? ''
}

function componentQuizJson(slot: LessonComponentSlot): string {
  if (!slot.content) return '[]'
  const c = slot.content as { questions?: unknown[] }
  return JSON.stringify({ questions: c.questions ?? [] }, null, 2)
}

function componentRubricJson(slot: LessonComponentSlot): string {
  if (!slot.content) return '{}'
  const c = slot.content as { rubric?: unknown }
  return JSON.stringify(c.rubric ?? {}, null, 2)
}

export default function LessonGeneratorPage() {
  const { courseCode } = useParams<{ courseCode: string }>()
  const navigate = useNavigate()
  const { ffLessonGenerator } = usePlatformFeatures()
  const [step, setStep] = useState<WizardStep>('input')
  const [jobId, setJobId] = useState<string | null>(null)
  const [pkg, setPkg] = useState<LessonPackage | null>(null)
  const [edits, setEdits] = useState<Record<string, string>>({})
  const [accepted, setAccepted] = useState<Set<string>>(new Set())
  const [moduleTitle, setModuleTitle] = useState('AI Lesson (draft)')
  const [submitting, setSubmitting] = useState(false)
  const [regenerating, setRegenerating] = useState<string | null>(null)
  const [form, setForm] = useState<LessonGeneratorInput>({
    learningObjective: '',
    gradeLevel: '',
    subject: '',
    differentiationLevels: ['on_grade'],
  })
  const liveRef = useRef<HTMLDivElement>(null)

  const enabled = ffLessonGenerator

  const completedCount = useMemo(
    () => pkg?.components.filter((c) => c.status === 'completed').length ?? 0,
    [pkg],
  )
  const totalCount = pkg?.components.length ?? 0

  const pollJob = useCallback(async () => {
    if (!courseCode || !jobId) return
    const data = await fetchLessonGeneratorJob(courseCode, jobId)
    setPkg(data)
    if (data.status === 'completed' || data.status === 'failed') {
      setStep('review')
      const done = new Set(
        data.components.filter((c) => c.status === 'completed').map((c) => c.key),
      )
      setAccepted(done)
    }
  }, [courseCode, jobId])

  useEffect(() => {
    if (step !== 'generating' || !jobId) return
    const id = window.setInterval(() => {
      void pollJob().catch(() => {})
    }, 1500)
    return () => window.clearInterval(id)
  }, [step, jobId, pollJob])

  useEffect(() => {
    if (liveRef.current) {
      liveRef.current.textContent = `Generation progress: ${completedCount} of ${totalCount} components ready.`
    }
  }, [completedCount, totalCount])

  async function handleStart(e: React.FormEvent) {
    e.preventDefault()
    if (!courseCode) return
    setSubmitting(true)
    try {
      const { jobId: id } = await startLessonGeneration(courseCode, form)
      setJobId(id)
      setStep('generating')
      setPkg(null)
    } catch (err) {
      toastMutationError(
        err instanceof Error ? err.message : 'Could not start lesson generation.',
      )
    } finally {
      setSubmitting(false)
    }
  }

  async function handleRegenerate(key: string) {
    if (!courseCode || !jobId) return
    setRegenerating(key)
    try {
      const data = await regenerateLessonComponent(courseCode, jobId, key)
      setPkg(data)
    } catch (err) {
      toastMutationError(err instanceof Error ? err.message : 'Regeneration failed.')
    } finally {
      setRegenerating(null)
    }
  }

  async function handleSave() {
    if (!courseCode || !jobId || !pkg) return
    setSubmitting(true)
    try {
      const componentEdits: Record<string, unknown> = {}
      for (const key of accepted) {
        const raw = edits[key]
        if (!raw) continue
        const slot = pkg.components.find((c) => c.key === key)
        if (!slot) continue
        if (key === 'quiz') {
          componentEdits[key] = JSON.parse(raw)
        } else if (key === 'rubric') {
          componentEdits[key] = { rubric: JSON.parse(raw) }
        } else if (key.startsWith('activity_') || key === 'lesson_plan') {
          const level = key.startsWith('activity_') ? key.replace('activity_', '') : undefined
          componentEdits[key] = level
            ? { level, markdown: raw }
            : { markdown: raw }
        }
      }
      const { moduleId } = await saveLessonPackageToCourse(courseCode, jobId, {
        acceptedComponents: [...accepted],
        moduleTitle,
        componentEdits,
      })
      toastSaveOk('Draft module saved.')
      setStep('saved')
      navigate(`/courses/${courseCode}/modules`, { state: { highlightModuleId: moduleId } })
    } catch (err) {
      toastMutationError(err instanceof Error ? err.message : 'Could not save to course.')
    } finally {
      setSubmitting(false)
    }
  }

  if (!enabled) {
    return (
      <LmsPage title="AI Lesson Generator">
        <p className="text-sm text-muted-foreground">
          The lesson generator is not enabled on this platform. Ask a global admin to enable{' '}
          <code className="text-xs">ffLessonGenerator</code> under Settings → Global platform.
        </p>
      </LmsPage>
    )
  }

  return (
    <LmsPage
      title="AI Lesson Generator"
      description="Generate a lesson plan, differentiated activities, formative quiz, and rubric from a learning objective."
      actions={
        <Link
          to={`/courses/${courseCode}/modules`}
          className="text-sm text-muted-foreground hover:text-foreground"
        >
          Back to modules
        </Link>
      }
    >
      {step === 'input' && (
        <form onSubmit={handleStart} className="mx-auto max-w-2xl space-y-4">
          <label className="block space-y-1">
            <span className="text-sm font-medium">Learning objective</span>
            <textarea
              required
              rows={3}
              className="w-full rounded-md border border-border bg-background px-3 py-2 text-sm"
              value={form.learningObjective}
              onChange={(e) => setForm((f) => ({ ...f, learningObjective: e.target.value }))}
              placeholder="Students will be able to multiply fractions."
            />
          </label>
          <div className="grid gap-4 sm:grid-cols-2">
            <label className="block space-y-1">
              <span className="text-sm font-medium">Grade level</span>
              <input
                required
                className="w-full rounded-md border border-border bg-background px-3 py-2 text-sm"
                value={form.gradeLevel}
                onChange={(e) => setForm((f) => ({ ...f, gradeLevel: e.target.value }))}
                placeholder="5 or HE"
              />
            </label>
            <label className="block space-y-1">
              <span className="text-sm font-medium">Subject</span>
              <input
                required
                className="w-full rounded-md border border-border bg-background px-3 py-2 text-sm"
                value={form.subject}
                onChange={(e) => setForm((f) => ({ ...f, subject: e.target.value }))}
                placeholder="Math"
              />
            </label>
          </div>
          <label className="block space-y-1">
            <span className="text-sm font-medium">Duration (minutes, optional)</span>
            <input
              type="number"
              min={15}
              max={180}
              className="w-full rounded-md border border-border bg-background px-3 py-2 text-sm"
              value={form.durationMinutes ?? ''}
              onChange={(e) =>
                setForm((f) => ({
                  ...f,
                  durationMinutes: e.target.value ? Number(e.target.value) : undefined,
                }))
              }
            />
          </label>
          <label className="block space-y-1">
            <span className="text-sm font-medium">Standards code (optional)</span>
            <input
              className="w-full rounded-md border border-border bg-background px-3 py-2 text-sm"
              value={form.standardsCode ?? ''}
              onChange={(e) =>
                setForm((f) => ({ ...f, standardsCode: e.target.value || undefined }))
              }
              placeholder="CCSS.MATH.5.NF.B.4"
            />
          </label>
          <fieldset className="space-y-2">
            <legend className="text-sm font-medium">Differentiation levels</legend>
            {DIFFERENTIATION_OPTIONS.map((opt) => (
              <label key={opt.value} className="flex items-center gap-2 text-sm">
                <input
                  type="checkbox"
                  checked={form.differentiationLevels?.includes(opt.value) ?? false}
                  onChange={(e) => {
                    setForm((f) => {
                      const current = new Set(f.differentiationLevels ?? [])
                      if (e.target.checked) current.add(opt.value)
                      else current.delete(opt.value)
                      return { ...f, differentiationLevels: [...current] }
                    })
                  }}
                />
                {opt.label}
              </label>
            ))}
          </fieldset>
          <button
            type="submit"
            disabled={submitting}
            className="inline-flex items-center gap-2 rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground disabled:opacity-50"
          >
            {submitting ? <Loader2 className="size-4 animate-spin" /> : null}
            Generate lesson package
          </button>
        </form>
      )}

      {step === 'generating' && (
        <div className="mx-auto max-w-xl space-y-4 text-center">
          <Loader2 className="mx-auto size-8 animate-spin text-primary" aria-hidden />
          <p className="text-sm">Generating your lesson components…</p>
          <div ref={liveRef} className="sr-only" aria-live="polite" />
          {pkg && (
            <ul className="space-y-2 text-left text-sm">
              {pkg.components.map((c) => (
                <li key={c.key} className="flex items-center justify-between rounded border px-3 py-2">
                  <span>{COMPONENT_LABELS[c.key] ?? c.key}</span>
                  <span className="capitalize text-muted-foreground">{c.status}</span>
                </li>
              ))}
            </ul>
          )}
        </div>
      )}

      {step === 'review' && pkg && (
        <div className="space-y-6">
          {pkg.standardsDisclaimer && (
            <p className="rounded-md border border-amber-500/30 bg-amber-500/10 px-3 py-2 text-sm text-amber-900 dark:text-amber-100">
              {pkg.standardsDisclaimer}
            </p>
          )}
          {pkg.components.map((slot) => (
            <section key={slot.key} className="rounded-lg border border-border p-4">
              <div className="mb-2 flex flex-wrap items-center justify-between gap-2">
                <h2 className="text-base font-semibold">{COMPONENT_LABELS[slot.key] ?? slot.key}</h2>
                <div className="flex items-center gap-2">
                  <ProvenanceBadge
                    generatedBy={slot.provenance?.generated_by}
                    modelId={slot.provenance?.model_id}
                  />
                  {slot.status === 'completed' && (
                    <label className="flex items-center gap-1 text-sm">
                      <input
                        type="checkbox"
                        checked={accepted.has(slot.key)}
                        onChange={(e) => {
                          setAccepted((prev) => {
                            const next = new Set(prev)
                            if (e.target.checked) next.add(slot.key)
                            else next.delete(slot.key)
                            return next
                          })
                        }}
                      />
                      Accept
                    </label>
                  )}
                  <button
                    type="button"
                    disabled={regenerating === slot.key}
                    onClick={() => void handleRegenerate(slot.key)}
                    className="inline-flex items-center gap-1 rounded border px-2 py-1 text-xs hover:bg-muted"
                  >
                    <RefreshCw
                      className={`size-3 ${regenerating === slot.key ? 'animate-spin' : ''}`}
                    />
                    Regenerate
                  </button>
                </div>
              </div>
              {slot.status === 'failed' && (
                <p className="text-sm text-destructive">{slot.error ?? 'Generation failed.'}</p>
              )}
              {slot.status === 'completed' && (
                <textarea
                  className="min-h-32 w-full rounded-md border border-border bg-background px-3 py-2 font-mono text-sm"
                  value={
                    edits[slot.key] ??
                    (slot.key === 'quiz'
                      ? componentQuizJson(slot)
                      : slot.key === 'rubric'
                        ? componentRubricJson(slot)
                        : componentMarkdown(slot))
                  }
                  onChange={(e) => setEdits((prev) => ({ ...prev, [slot.key]: e.target.value }))}
                />
              )}
            </section>
          ))}
          <div className="flex flex-wrap items-end gap-4">
            <label className="block space-y-1">
              <span className="text-sm font-medium">Draft module title</span>
              <input
                className="rounded-md border border-border bg-background px-3 py-2 text-sm"
                value={moduleTitle}
                onChange={(e) => setModuleTitle(e.target.value)}
              />
            </label>
            <button
              type="button"
              disabled={submitting || accepted.size === 0}
              onClick={() => void handleSave()}
              className="inline-flex items-center gap-2 rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground disabled:opacity-50"
            >
              {submitting ? <Loader2 className="size-4 animate-spin" /> : <Save className="size-4" />}
              Save to course
            </button>
          </div>
        </div>
      )}
    </LmsPage>
  )
}
