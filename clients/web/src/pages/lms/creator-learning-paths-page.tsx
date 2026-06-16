import { useCallback, useEffect, useMemo, useState } from 'react'
import { GripVertical, Plus, Route, Trash2 } from 'lucide-react'
import {
  createLearningPath,
  deleteLearningPath,
  fetchCreatorLearningPaths,
  updateLearningPath,
  type CreatorLearningPath,
} from '../../lib/learning-paths-api'
import { authorizedFetch } from '../../lib/api'
import { type CoursePublic } from '../../lib/courses-api'
import { usePlatformFeatures } from '../../context/platform-features-context'
import { LmsPage } from './lms-page'
import { EmptyState } from '../../components/ui/empty-state'

type TeachableCourse = { id: string; courseCode: string; title: string }

export default function CreatorLearningPathsPage() {
  const { ffLearningPaths, loading: featuresLoading } = usePlatformFeatures()
  const [paths, setPaths] = useState<CreatorLearningPath[]>([])
  const [courses, setCourses] = useState<TeachableCourse[]>([])
  const [loading, setLoading] = useState(true)
  const [title, setTitle] = useState('')
  const [description, setDescription] = useState('')
  const [selectedCourseIds, setSelectedCourseIds] = useState<string[]>([])
  const [bundleDollars, setBundleDollars] = useState('')
  const [isPublic, setIsPublic] = useState(false)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')

  const load = useCallback(async () => {
    const [pathRows, courseRes] = await Promise.all([
      fetchCreatorLearningPaths(),
      authorizedFetch('/api/v1/courses'),
    ])
    setPaths(pathRows)
    if (courseRes.ok) {
      const data = (await courseRes.json()) as { courses?: CoursePublic[] }
      const teachable = (data.courses ?? []).filter((c) =>
        (c.viewerEnrollmentRoles ?? []).some((r) => r.toLowerCase() === 'teacher'),
      )
      setCourses(teachable.map((c) => ({ id: c.id, courseCode: c.courseCode, title: c.title })))
    }
  }, [])

  useEffect(() => {
    if (featuresLoading || !ffLearningPaths) {
      setLoading(false)
      return
    }
    load()
      .catch(() => setError('Failed to load learning paths.'))
      .finally(() => setLoading(false))
  }, [ffLearningPaths, featuresLoading, load])

  const courseMap = useMemo(() => new Map(courses.map((c) => [c.id, c])), [courses])

  function toggleCourse(courseId: string) {
    setSelectedCourseIds((prev) =>
      prev.includes(courseId) ? prev.filter((id) => id !== courseId) : [...prev, courseId],
    )
  }

  function moveCourse(index: number, direction: -1 | 1) {
    setSelectedCourseIds((prev) => {
      const next = [...prev]
      const target = index + direction
      if (target < 0 || target >= next.length) return prev
      ;[next[index], next[target]] = [next[target], next[index]]
      return next
    })
  }

  async function handleCreate(e: React.FormEvent) {
    e.preventDefault()
    if (!title.trim() || selectedCourseIds.length === 0) return
    setSaving(true)
    setError('')
    try {
      const bundlePriceCents =
        bundleDollars.trim() === '' ? undefined : Math.round(parseFloat(bundleDollars) * 100)
      await createLearningPath({
        title: title.trim(),
        description: description.trim(),
        courseIds: selectedCourseIds,
        bundlePriceCents,
        isPublic,
      })
      setTitle('')
      setDescription('')
      setSelectedCourseIds([])
      setBundleDollars('')
      setIsPublic(false)
      await load()
    } catch {
      setError('Could not create learning path.')
    } finally {
      setSaving(false)
    }
  }

  async function handleTogglePublic(path: CreatorLearningPath) {
    await updateLearningPath(path.id, { isPublic: !path.isPublic })
    await load()
  }

  async function handleDelete(pathId: string) {
    if (!window.confirm('Delete this learning path?')) return
    await deleteLearningPath(pathId)
    await load()
  }

  if (!ffLearningPaths && !featuresLoading) {
    return (
      <LmsPage title="Learning path builder">
        <EmptyState title="Learning paths are not enabled" body="Contact your administrator." icon={Route} />
      </LmsPage>
    )
  }

  return (
    <LmsPage title="Learning path builder">
      <p className="mb-6 text-sm text-muted-foreground">
        Group your courses into an ordered specialization with optional bundle pricing.
      </p>

      <form onSubmit={(e) => void handleCreate(e)} className="mb-10 space-y-4 rounded-lg border bg-card p-4">
        <h2 className="text-lg font-semibold">Create a path</h2>
        <div>
          <label htmlFor="path-title" className="text-sm font-medium">
            Title
          </label>
          <input
            id="path-title"
            className="mt-1 w-full rounded-md border bg-background px-3 py-2 text-sm"
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            required
          />
        </div>
        <div>
          <label htmlFor="path-description" className="text-sm font-medium">
            Description
          </label>
          <textarea
            id="path-description"
            className="mt-1 w-full rounded-md border bg-background px-3 py-2 text-sm"
            rows={3}
            value={description}
            onChange={(e) => setDescription(e.target.value)}
          />
        </div>
        <div>
          <label htmlFor="path-bundle" className="text-sm font-medium">
            Bundle price (USD, optional)
          </label>
          <input
            id="path-bundle"
            type="number"
            min="0"
            step="0.01"
            className="mt-1 w-40 rounded-md border bg-background px-3 py-2 text-sm"
            value={bundleDollars}
            onChange={(e) => setBundleDollars(e.target.value)}
          />
        </div>
        <label className="flex items-center gap-2 text-sm">
          <input type="checkbox" checked={isPublic} onChange={(e) => setIsPublic(e.target.checked)} />
          List in public catalog
        </label>
        <fieldset>
          <legend className="text-sm font-medium">Courses (drag order with arrows)</legend>
          <div className="mt-2 max-h-48 space-y-1 overflow-y-auto rounded-md border p-2">
            {courses.map((course) => (
              <label key={course.id} className="flex items-center gap-2 text-sm">
                <input
                  type="checkbox"
                  checked={selectedCourseIds.includes(course.id)}
                  onChange={() => toggleCourse(course.id)}
                />
                <span>
                  {course.title} <span className="text-muted-foreground">({course.courseCode})</span>
                </span>
              </label>
            ))}
          </div>
          {selectedCourseIds.length > 0 ? (
            <ol className="mt-3 space-y-1 text-sm">
              {selectedCourseIds.map((id, index) => {
                const course = courseMap.get(id)
                return (
                  <li key={id} className="flex items-center gap-2 rounded border px-2 py-1">
                    <GripVertical className="size-4 text-muted-foreground" aria-hidden />
                    <span className="flex-1">{course?.title ?? id}</span>
                    <button type="button" className="text-xs text-primary" onClick={() => moveCourse(index, -1)}>
                      Up
                    </button>
                    <button type="button" className="text-xs text-primary" onClick={() => moveCourse(index, 1)}>
                      Down
                    </button>
                  </li>
                )
              })}
            </ol>
          ) : null}
        </fieldset>
        {error ? (
          <p className="text-sm text-destructive" role="alert">
            {error}
          </p>
        ) : null}
        <button
          type="submit"
          disabled={saving || selectedCourseIds.length === 0}
          className="inline-flex items-center gap-2 rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground disabled:opacity-60"
        >
          <Plus className="size-4" aria-hidden />
          {saving ? 'Creating…' : 'Create path'}
        </button>
      </form>

      <section aria-labelledby="existing-paths-heading">
        <h2 id="existing-paths-heading" className="text-lg font-semibold">
          Your paths
        </h2>
        {loading ? (
          <div className="mt-4 h-24 motion-safe:animate-pulse rounded-lg border bg-card" aria-hidden />
        ) : paths.length === 0 ? (
          <p className="mt-2 text-sm text-muted-foreground">No paths yet.</p>
        ) : (
          <ul className="mt-4 space-y-3">
            {paths.map((path) => (
              <li key={path.id} className="flex items-start justify-between gap-3 rounded-lg border bg-card p-4">
                <div>
                  <h3 className="font-medium">{path.title}</h3>
                  <p className="text-sm text-muted-foreground">
                    {path.courseIds.length} courses
                    {path.slug ? ` · /paths/${path.slug}` : ''}
                  </p>
                  <label className="mt-2 flex items-center gap-2 text-sm">
                    <input
                      type="checkbox"
                      checked={path.isPublic}
                      onChange={() => void handleTogglePublic(path)}
                    />
                    Public catalog
                  </label>
                </div>
                <button
                  type="button"
                  className="text-destructive"
                  aria-label={`Delete ${path.title}`}
                  onClick={() => void handleDelete(path.id)}
                >
                  <Trash2 className="size-4" />
                </button>
              </li>
            ))}
          </ul>
        )}
      </section>
    </LmsPage>
  )
}
