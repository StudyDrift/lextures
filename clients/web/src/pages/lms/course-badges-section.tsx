import { useCallback, useEffect, useState, type FormEvent } from 'react'
import { Award, Loader2, Plus } from 'lucide-react'
import { usePlatformFeatures } from '../../context/platform-features-context'
import {
  awardBadge,
  createBadgeDefinition,
  fetchBadgeCandidates,
  listCourseBadgeDefinitions,
  type BadgeDefinition,
} from '../../lib/badges-api'
import { fetchCourseOutcomes, type CourseOutcome } from '../../lib/courses-api'

type Props = {
  courseCode: string
  courseId?: string
}

export function CourseBadgesSection({ courseCode, courseId }: Props) {
  const { ffCompetencyBadges } = usePlatformFeatures()
  const [definitions, setDefinitions] = useState<BadgeDefinition[]>([])
  const [outcomes, setOutcomes] = useState<CourseOutcome[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [criteria, setCriteria] = useState('')
  const [outcomeId, setOutcomeId] = useState('')
  const [autoAward, setAutoAward] = useState(false)
  const [creating, setCreating] = useState(false)
  const [awardDefId, setAwardDefId] = useState<string | null>(null)
  const [candidates, setCandidates] = useState<
    { userId: string; displayName: string; alreadyAwarded: boolean; masteryReached: boolean }[]
  >([])
  const [selected, setSelected] = useState<Set<string>>(new Set())
  const [awarding, setAwarding] = useState(false)

  const resolvedCourseId = courseId || courseCode

  const load = useCallback(async () => {
    if (!ffCompetencyBadges) return
    setLoading(true)
    setError(null)
    try {
      const [defs, outs] = await Promise.all([
        listCourseBadgeDefinitions(resolvedCourseId),
        fetchCourseOutcomes(courseCode).catch(() => ({ outcomes: [] as CourseOutcome[] })),
      ])
      setDefinitions(defs.definitions)
      setOutcomes(outs.outcomes ?? [])
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load badges.')
    } finally {
      setLoading(false)
    }
  }, [courseCode, resolvedCourseId, ffCompetencyBadges])

  useEffect(() => {
    void load()
  }, [load])

  if (!ffCompetencyBadges) {
    return (
      <p className="text-sm text-slate-600 dark:text-slate-300">
        Competency badges are not enabled on this platform. Enable them in Settings → Global platform.
      </p>
    )
  }

  async function onCreate(e: FormEvent) {
    e.preventDefault()
    setCreating(true)
    setError(null)
    try {
      await createBadgeDefinition(resolvedCourseId, {
        name,
        description,
        criteriaNarrative: criteria,
        outcomeId: outcomeId || undefined,
        autoAward,
      })
      setName('')
      setDescription('')
      setCriteria('')
      setOutcomeId('')
      setAutoAward(false)
      await load()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create badge.')
    } finally {
      setCreating(false)
    }
  }

  async function openAward(defId: string) {
    setAwardDefId(defId)
    setSelected(new Set())
    setError(null)
    try {
      const data = await fetchBadgeCandidates(defId)
      setCandidates(data.candidates)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load candidates.')
      setCandidates([])
    }
  }

  async function onAward() {
    if (!awardDefId || selected.size === 0) return
    setAwarding(true)
    setError(null)
    try {
      await awardBadge(awardDefId, [...selected])
      setAwardDefId(null)
      await load()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to award badge.')
    } finally {
      setAwarding(false)
    }
  }

  return (
    <div className="space-y-8">
      <header>
        <h2 className="flex items-center gap-2 text-lg font-semibold text-slate-900 dark:text-white">
          <Award className="h-5 w-5 text-indigo-600" aria-hidden />
          Competency micro-badges
        </h2>
        <p className="mt-1 text-sm text-slate-600 dark:text-slate-300">
          Define signed Open Badges for learning outcomes and award them to students who demonstrate mastery.
        </p>
      </header>

      {error ? (
        <p role="alert" className="text-sm text-red-700 dark:text-red-300">
          {error}
        </p>
      ) : null}

      {loading ? (
        <p className="inline-flex items-center gap-2 text-sm text-slate-600">
          <Loader2 className="h-4 w-4 motion-safe:animate-spin" aria-hidden />
          Loading badge definitions…
        </p>
      ) : null}

      <form onSubmit={(e) => void onCreate(e)} className="space-y-3 rounded-xl border border-slate-200 p-4 dark:border-slate-700">
        <h3 className="text-sm font-semibold">New badge</h3>
        <div>
          <label className="mb-1 block text-sm font-medium" htmlFor="badge-name">
            Name
          </label>
          <input
            id="badge-name"
            required
            value={name}
            onChange={(e) => setName(e.target.value)}
            className="w-full rounded-lg border border-slate-200 px-3 py-2 text-sm dark:border-slate-600 dark:bg-slate-900"
          />
        </div>
        <div>
          <label className="mb-1 block text-sm font-medium" htmlFor="badge-desc">
            Description
          </label>
          <textarea
            id="badge-desc"
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            rows={2}
            className="w-full rounded-lg border border-slate-200 px-3 py-2 text-sm dark:border-slate-600 dark:bg-slate-900"
          />
        </div>
        <div>
          <label className="mb-1 block text-sm font-medium" htmlFor="badge-criteria">
            Criteria
          </label>
          <textarea
            id="badge-criteria"
            value={criteria}
            onChange={(e) => setCriteria(e.target.value)}
            rows={2}
            className="w-full rounded-lg border border-slate-200 px-3 py-2 text-sm dark:border-slate-600 dark:bg-slate-900"
          />
        </div>
        {outcomes.length > 0 ? (
          <div>
            <label className="mb-1 block text-sm font-medium" htmlFor="badge-outcome">
              Linked outcome
            </label>
            <select
              id="badge-outcome"
              value={outcomeId}
              onChange={(e) => setOutcomeId(e.target.value)}
              className="w-full rounded-lg border border-slate-200 px-3 py-2 text-sm dark:border-slate-600 dark:bg-slate-900"
            >
              <option value="">None</option>
              {outcomes.map((o) => (
                <option key={o.id} value={o.id}>
                  {o.title}
                </option>
              ))}
            </select>
          </div>
        ) : null}
        <label className="flex items-center gap-2 text-sm">
          <input type="checkbox" checked={autoAward} onChange={(e) => setAutoAward(e.target.checked)} />
          Auto-award when outcome mastery is reached
        </label>
        <button
          type="submit"
          disabled={creating || !name.trim()}
          className="inline-flex items-center gap-2 rounded-lg bg-indigo-600 px-3 py-2 text-sm font-medium text-white hover:bg-indigo-500 disabled:opacity-60"
        >
          <Plus className="h-4 w-4" aria-hidden />
          {creating ? 'Creating…' : 'Create badge'}
        </button>
      </form>

      <ul className="space-y-3">
        {definitions.map((d) => (
          <li
            key={d.id}
            className="rounded-xl border border-slate-200 bg-white p-4 dark:border-slate-700 dark:bg-slate-800"
          >
            <div className="flex flex-wrap items-start justify-between gap-2">
              <div>
                <h3 className="font-semibold text-slate-900 dark:text-white">{d.name}</h3>
                <p className="text-xs text-slate-500">
                  slug: {d.slug}
                  {d.autoAward ? ' · auto-award' : ''}
                </p>
                {d.description ? (
                  <p className="mt-1 text-sm text-slate-600 dark:text-slate-300">{d.description}</p>
                ) : null}
              </div>
              <button
                type="button"
                onClick={() => void openAward(d.id)}
                className="rounded-lg border border-slate-300 px-3 py-1.5 text-sm font-medium hover:bg-slate-50 dark:border-slate-600 dark:hover:bg-slate-700"
              >
                Award
              </button>
            </div>
          </li>
        ))}
      </ul>

      {awardDefId ? (
        <div
          role="dialog"
          aria-modal="true"
          aria-label="Award badge"
          className="fixed inset-0 z-50 flex items-end justify-center bg-black/40 p-4 sm:items-center"
        >
          <div className="max-h-[80vh] w-full max-w-lg overflow-auto rounded-2xl bg-white p-5 shadow-xl dark:bg-slate-900">
            <h3 className="text-lg font-semibold">Award badge</h3>
            <p className="mt-1 text-sm text-slate-600 dark:text-slate-300">
              Select enrolled students. Mastery status is shown when available.
            </p>
            <ul className="mt-4 max-h-64 space-y-2 overflow-auto">
              {candidates.map((c) => (
                <li key={c.userId}>
                  <label className="flex items-center gap-2 text-sm">
                    <input
                      type="checkbox"
                      disabled={c.alreadyAwarded}
                      checked={selected.has(c.userId)}
                      onChange={(e) => {
                        setSelected((prev) => {
                          const next = new Set(prev)
                          if (e.target.checked) next.add(c.userId)
                          else next.delete(c.userId)
                          return next
                        })
                      }}
                    />
                    <span>
                      {c.displayName}
                      {c.alreadyAwarded ? ' (already awarded)' : ''}
                      {c.masteryReached ? ' · mastery met' : ''}
                    </span>
                  </label>
                </li>
              ))}
            </ul>
            <div className="mt-4 flex justify-end gap-2">
              <button
                type="button"
                className="rounded-lg border border-slate-300 px-3 py-2 text-sm dark:border-slate-600"
                onClick={() => setAwardDefId(null)}
              >
                Cancel
              </button>
              <button
                type="button"
                disabled={awarding || selected.size === 0}
                onClick={() => void onAward()}
                className="rounded-lg bg-indigo-600 px-3 py-2 text-sm font-medium text-white disabled:opacity-60"
              >
                {awarding ? 'Awarding…' : `Award selected (${selected.size})`}
              </button>
            </div>
          </div>
        </div>
      ) : null}
    </div>
  )
}
