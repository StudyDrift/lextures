import { useCallback, useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { BookOpen } from 'lucide-react'
import { Button, Dialog, Field, InlineAlert, Input, Radio, RadioGroup, Select } from '../../components/ui'
import { usePlatformFeatures } from '../../context/platform-features-context'
import { listDependents, type ManagedDependent } from '../../lib/managed-learners-api'
import {
  formatStarterLength,
  generateStarterCourse,
  getStarterCourse,
  listStarterCourses,
  copyStarterCourse,
  type StarterDetail,
  type StarterLibraryList,
} from '../../lib/starter-library-api'
import { LmsPage } from './lms-page'

type PendingCopy =
  | { kind: 'use'; slug: string; title: string }
  | { kind: 'generate'; title: string }

export default function StarterLibraryPage() {
  const navigate = useNavigate()
  const { ffHomeschoolManagedLearners } = usePlatformFeatures()
  const [gradeBand, setGradeBand] = useState('')
  const [subject, setSubject] = useState('')
  const [filterGrade, setFilterGrade] = useState('')
  const [filterSubject, setFilterSubject] = useState('')
  const [focus, setFocus] = useState('')
  const [library, setLibrary] = useState<StarterLibraryList | null>(null)
  const [selected, setSelected] = useState<StarterDetail | null>(null)
  const [loading, setLoading] = useState(true)
  const [detailLoading, setDetailLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [pending, setPending] = useState<PendingCopy | null>(null)
  const [learners, setLearners] = useState<ManagedDependent[]>([])
  const [learnerId, setLearnerId] = useState('')
  const [working, setWorking] = useState(false)

  const load = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const next = await listStarterCourses({ gradeBand: filterGrade, subject: filterSubject })
      setLibrary(next)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Could not load the starter library.')
    } finally {
      setLoading(false)
    }
  }, [filterGrade, filterSubject])

  useEffect(() => {
    void load()
  }, [load])

  async function openCourse(slug: string) {
    setDetailLoading(true)
    setError(null)
    try {
      setSelected(await getStarterCourse(slug))
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Could not load that starter course.')
    } finally {
      setDetailLoading(false)
    }
  }

  async function openAssign(next: PendingCopy) {
    if (!ffHomeschoolManagedLearners) {
      await runCopy(next, '')
      return
    }
    setLearnerId('')
    setPending(next)
    try {
      const rows = await listDependents()
      setLearners(rows)
    } catch {
      setLearners([])
    }
  }

  async function runCopy(next: PendingCopy, assignTo: string) {
    setWorking(true)
    setError(null)
    try {
      const result =
        next.kind === 'use'
          ? await copyStarterCourse(next.slug, assignTo || undefined)
          : await generateStarterCourse({
              gradeBand,
              subject,
              focus: focus.trim() || undefined,
              learnerUserId: assignTo || undefined,
            })
      setPending(null)
      navigate(`/courses/${encodeURIComponent(result.courseCode)}`)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Could not copy this course.')
    } finally {
      setWorking(false)
    }
  }

  const bands = library?.gradeBands ?? ['K-2', '3-5', '6-8', '9-12']
  const subjects = library?.subjects ?? []

  return (
    <LmsPage
      title={'Starter library'}
      description="Copy a ready course for a grade and subject, then edit it for your family."
    >
      {error ? (
        <div className="mt-4">
          <InlineAlert tone="danger">{error}</InlineAlert>
        </div>
      ) : null}

      <form
        className="mt-6 grid gap-4 rounded-2xl border border-border-default bg-surface-raised p-4 sm:grid-cols-2 lg:grid-cols-4"
        onSubmit={(event) => {
          event.preventDefault()
          void openAssign({ kind: 'generate', title: 'Generated starter' })
        }}
      >
        <Field label="Grade band" required>
          <Select value={gradeBand} onChange={(event) => setGradeBand(event.target.value)} required>
            <option value="">Choose a band</option>
            {bands.map((band) => (
              <option key={band} value={band}>
                {band}
              </option>
            ))}
          </Select>
        </Field>
        <Field label="Subject" required>
          <Select value={subject} onChange={(event) => setSubject(event.target.value)} required>
            <option value="">Choose a subject</option>
            {subjects.map((item) => (
              <option key={item.id} value={item.id}>
                {item.label}
              </option>
            ))}
          </Select>
        </Field>
        <Field label="Focus" description="Optional. Added when you generate.">
          <Input value={focus} onChange={(event) => setFocus(event.target.value)} placeholder="Fractions with cooking" />
        </Field>
        <div className="flex items-end">
          <Button type="submit" className="w-full" disabled={!gradeBand || !subject || working}>
            Generate a starter course
          </Button>
        </div>
      </form>
      <p className="mt-2 text-sm text-fg-muted">
        Generate uses AI when it is configured. Otherwise you get the curated starter for that grade and subject, ready to edit.
      </p>

      <div className="mt-6 flex flex-wrap gap-3">
        <Field label="Filter by grade">
          <Select
            value={filterGrade}
            onChange={(event) => {
              setSelected(null)
              setFilterGrade(event.target.value)
            }}
          >
            <option value="">All grades</option>
            {bands.map((band) => (
              <option key={band} value={band}>
                {band}
              </option>
            ))}
          </Select>
        </Field>
        <Field label="Filter by subject">
          <Select
            value={filterSubject}
            onChange={(event) => {
              setSelected(null)
              setFilterSubject(event.target.value)
            }}
          >
            <option value="">All subjects</option>
            {subjects.map((item) => (
              <option key={item.id} value={item.id}>
                {item.label}
              </option>
            ))}
          </Select>
        </Field>
      </div>

      {loading ? <p className="mt-6 text-sm text-fg-muted">Loading starter courses…</p> : null}

      {!loading && library && library.courses.length === 0 ? (
        <p className="mt-6 text-sm text-fg-muted">No starter courses match those filters.</p>
      ) : null}

      <ul className="mt-6 grid gap-4 lg:grid-cols-2">
        {(library?.courses ?? []).map((course) => (
          <li key={course.slug} className="rounded-2xl border border-border-default bg-surface-raised p-4">
            <div className="flex items-start gap-3">
              <BookOpen className="mt-0.5 h-5 w-5 shrink-0 text-fg-muted" aria-hidden />
              <div className="min-w-0 flex-1">
                <h2 className="text-base font-semibold text-fg-default">{course.title}</h2>
                <p className="mt-1 text-sm text-fg-muted">
                  {course.gradeBand} · {course.subjectLabel}
                  {formatStarterLength(course.estimatedMinutes)
                    ? ` · ${formatStarterLength(course.estimatedMinutes)}`
                    : ''}
                </p>
                <p className="mt-2 text-sm text-fg-default">{course.summary}</p>
                <p className="mt-2 text-sm text-fg-muted">
                  {course.modules.length} modules: {course.modules.map((mod) => mod.title).join(', ')}
                </p>
                {course.standards.length > 0 ? (
                  <p className="mt-2 text-sm text-fg-muted">Standards: {course.standards.join(', ')}</p>
                ) : null}
                <div className="mt-3">
                  <Button type="button" variant="secondary" onClick={() => void openCourse(course.slug)}>
                    View {course.title}
                  </Button>
                </div>
              </div>
            </div>
          </li>
        ))}
      </ul>

      {detailLoading ? <p className="mt-6 text-sm text-fg-muted">Loading course…</p> : null}

      {selected ? (
        <section className="mt-6 rounded-2xl border border-border-default bg-surface-base p-4" aria-labelledby="starter-detail-title">
          <h2 id="starter-detail-title" className="text-lg font-semibold text-fg-default">
            {selected.title}
          </h2>
          <p className="mt-1 text-sm text-fg-muted">
            {selected.gradeBand} · {selected.subjectLabel} · {formatStarterLength(selected.estimatedMinutes)}
          </p>
          {selected.standards.length > 0 ? (
            <p className="mt-2 text-sm text-fg-muted">Standards: {selected.standards.join(', ')}</p>
          ) : null}
          <ol className="mt-4 space-y-4">
            {selected.modules.map((mod) => (
              <li key={mod.title}>
                <h3 className="font-medium text-fg-default">{mod.title}</h3>
                <ul className="mt-1 list-disc ps-5 text-sm text-fg-muted">
                  {mod.pages.map((page) => (
                    <li key={page.title}>{page.title}</li>
                  ))}
                </ul>
              </li>
            ))}
          </ol>
          <div className="mt-4">
            <Button
              type="button"
              onClick={() => void openAssign({ kind: 'use', slug: selected.slug, title: selected.title })}
              disabled={working}
            >
              Use this course
            </Button>
          </div>
        </section>
      ) : null}

      <Dialog
        open={pending != null}
        onClose={() => {
          if (!working) setPending(null)
        }}
        title={pending?.kind === 'generate' ? 'Generate a starter course' : 'Use this course'}
        description="This copies an editable course into your account. You can assign it to a learner now."
        closeLabel="Close"
        footer={
          <Button
            type="button"
            loading={working}
            onClick={() => {
              if (!pending) return
              void runCopy(pending, learnerId)
            }}
          >
            {pending?.kind === 'generate' ? 'Generate course' : 'Copy course'}
          </Button>
        }
      >
        {learners.length > 0 ? (
          <RadioGroup
            name="starter-learner"
            value={learnerId}
            onChange={setLearnerId}
            legend="Assign to a learner"
          >
            <Radio value="" label="Not yet" />
            {learners.map((learner) => (
              <Radio key={learner.id} value={learner.id} label={learner.displayName} />
            ))}
          </RadioGroup>
        ) : (
          <p className="text-sm text-fg-muted">
            No managed learners yet. You can add one later and enroll them from the course.
          </p>
        )}
      </Dialog>
    </LmsPage>
  )
}
