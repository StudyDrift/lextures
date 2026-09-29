import { useCallback, useEffect, useState } from 'react'
import type { FormEvent } from 'react'
import { Link } from 'react-router-dom'
import { Plus, UserRound } from 'lucide-react'
import { ConfirmDialog } from '../../components/confirm-dialog'
import { usePlatformFeatures } from '../../context/platform-features-context'
import {
  createDependent,
  deleteDependent,
  listDependents,
  startLearnAs,
  type ManagedDependent,
} from '../../lib/managed-learners-api'

export default function LearnersPage() {
  const { ffHomeschoolManagedLearners, loading: featuresLoading } = usePlatformFeatures()
  const [deps, setDeps] = useState<ManagedDependent[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [name, setName] = useState('')
  const [grade, setGrade] = useState('')
  const [under13, setUnder13] = useState(false)
  const [saving, setSaving] = useState(false)
  const [busyId, setBusyId] = useState<string | null>(null)
  const [removeTarget, setRemoveTarget] = useState<ManagedDependent | null>(null)

  const reload = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      setDeps(await listDependents())
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load learners')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    if (featuresLoading) return
    if (!ffHomeschoolManagedLearners) {
      setLoading(false)
      return
    }
    void reload()
  }, [featuresLoading, ffHomeschoolManagedLearners, reload])

  async function onAdd(e: FormEvent) {
    e.preventDefault()
    if (!name.trim()) return
    setSaving(true)
    setError(null)
    try {
      await createDependent({
        displayName: name.trim(),
        gradeLevel: grade.trim() || undefined,
        under13,
      })
      setName('')
      setGrade('')
      setUnder13(false)
      await reload()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to add learner')
    } finally {
      setSaving(false)
    }
  }

  async function onLearnAs(id: string) {
    setBusyId(id)
    setError(null)
    try {
      await startLearnAs(id)
      window.location.assign('/dashboard')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to Learn as')
      setBusyId(null)
    }
  }

  async function onRemoveConfirmed() {
    if (!removeTarget) return
    const id = removeTarget.id
    setBusyId(id)
    setError(null)
    try {
      await deleteDependent(id)
      setRemoveTarget(null)
      await reload()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to remove')
    } finally {
      setBusyId(null)
    }
  }

  if (featuresLoading) {
    return <div className="p-6 text-sm text-fg-muted">Loading…</div>
  }
  if (!ffHomeschoolManagedLearners) {
    return (
      <div className="mx-auto max-w-2xl p-6">
        <h1 className="text-xl font-semibold text-fg-default">Learners</h1>
        <p className="mt-2 text-sm text-fg-muted">Managed learners are not enabled on this platform.</p>
      </div>
    )
  }

  return (
    <div className="mx-auto max-w-2xl space-y-6 p-6">
      <div>
        <h1 className="text-xl font-semibold text-fg-default">Learners</h1>
        <p className="mt-1 text-sm text-fg-muted">
          Add children by name (no email required), enroll them in your courses, then Learn as them on a shared device.
        </p>
      </div>

      {error && (
        <div className="rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-900 dark:border-red-900/50 dark:bg-red-950/40 dark:text-red-100">
          {error}
        </div>
      )}

      <form onSubmit={(e) => void onAdd(e)} className="rounded-2xl border border-border-default bg-surface-raised p-4 shadow-sm">
        <h2 className="text-sm font-semibold text-fg-default">Add learner</h2>
        <div className="mt-3 grid gap-3 sm:grid-cols-2">
          <div className="sm:col-span-2">
            <label htmlFor="learner-name" className="text-xs font-medium text-fg-muted">
              Display name
            </label>
            <input
              id="learner-name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              required
              className="mt-1 w-full rounded-xl border border-border-default bg-surface-raised px-3 py-2 text-sm text-fg-default outline-none focus:border-indigo-400 focus:ring-2 focus:ring-indigo-500/20"
              placeholder="Alex"
              disabled={saving}
            />
          </div>
          <div>
            <label htmlFor="learner-grade" className="text-xs font-medium text-fg-muted">
              Grade level (optional)
            </label>
            <input
              id="learner-grade"
              value={grade}
              onChange={(e) => setGrade(e.target.value)}
              className="mt-1 w-full rounded-xl border border-border-default bg-surface-raised px-3 py-2 text-sm text-fg-default outline-none focus:border-indigo-400 focus:ring-2 focus:ring-indigo-500/20"
              placeholder="K, 1–12, …"
              disabled={saving}
            />
          </div>
          <div className="flex items-end pb-2">
            <label className="flex items-center gap-2 text-sm text-fg-default">
              <input
                type="checkbox"
                checked={under13}
                onChange={(e) => setUnder13(e.target.checked)}
                disabled={saving}
                className="rounded border-border-default"
              />
              This learner is under 13 (COPPA)
            </label>
          </div>
        </div>
        <button
          type="submit"
          disabled={saving || !name.trim()}
          className="mt-4 inline-flex min-h-6 min-w-6 items-center gap-2 rounded-xl bg-indigo-600 px-4 py-2 text-sm font-semibold text-white hover:bg-indigo-500 disabled:opacity-60"
        >
          <Plus className="h-4 w-4" />
          {saving ? 'Adding…' : 'Add learner'}
        </button>
      </form>

      <section className="space-y-3">
        <h2 className="text-sm font-semibold text-fg-default">Your learners</h2>
        {loading ? (
          <p className="text-sm text-fg-muted">Loading…</p>
        ) : deps.length === 0 ? (
          <p className="rounded-xl border border-dashed border-border-default px-4 py-8 text-center text-sm text-fg-muted">
            No learners yet. Add a child above, then enroll them from a course&apos;s Enrollments page.
          </p>
        ) : (
          <ul className="divide-y divide-border-default overflow-hidden rounded-2xl border border-border-default bg-surface-raised">
            {deps.map((d) => (
              <li key={d.id} className="flex flex-wrap items-center justify-between gap-3 px-4 py-3">
                <div className="flex min-w-0 items-center gap-3">
                  <div className="flex h-9 w-9 items-center justify-center rounded-full bg-indigo-100 text-indigo-700 dark:bg-indigo-950 dark:text-indigo-200">
                    <UserRound className="h-4 w-4" />
                  </div>
                  <div className="min-w-0">
                    <p className="truncate font-medium text-fg-default">{d.displayName}</p>
                    <p className="text-xs text-fg-muted">
                      {d.gradeLevel ? `Grade ${d.gradeLevel}` : 'No grade'}
                      {d.coppaMinor ? ' · Under 13' : ''}
                    </p>
                  </div>
                </div>
                <div className="flex flex-wrap gap-2">
                  <button
                    type="button"
                    disabled={busyId === d.id}
                    onClick={() => void onLearnAs(d.id)}
                    className="rounded-lg border border-border-default bg-surface-base px-3 py-1.5 text-xs font-semibold text-fg-default hover:bg-surface-sunken disabled:opacity-60"
                  >
                    Learn as…
                  </button>
                  <button
                    type="button"
                    disabled={busyId === d.id}
                    onClick={() => setRemoveTarget(d)}
                    className="rounded-lg px-3 py-1.5 text-xs font-medium text-rose-700 hover:bg-rose-50 dark:text-rose-300 dark:hover:bg-rose-950/40 disabled:opacity-60"
                  >
                    Remove
                  </button>
                </div>
              </li>
            ))}
          </ul>
        )}
        <p className="text-xs text-fg-muted">
          Tip: open a course → Enrollments → Add enrollment → Managed learners to enroll without an email.{' '}
          <Link to="/courses" className="text-indigo-600 hover:underline dark:text-indigo-400">
            Browse courses
          </Link>
        </p>
      </section>

      <ConfirmDialog
        open={removeTarget !== null}
        title="Remove learner?"
        description={
          removeTarget ? (
            <span>
              {removeTarget.displayName} will be deactivated and unlinked. This cannot be undone from
              the Learners page.
            </span>
          ) : null
        }
        confirmLabel="Remove"
        variant="danger"
        busy={removeTarget !== null && busyId === removeTarget.id}
        onConfirm={() => void onRemoveConfirmed()}
        onClose={() => {
          if (busyId) return
          setRemoveTarget(null)
        }}
      />
    </div>
  )
}
