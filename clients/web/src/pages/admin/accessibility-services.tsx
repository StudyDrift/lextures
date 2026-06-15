import { useCallback, useEffect, useState } from 'react'
import { ShieldCheck } from 'lucide-react'
import { usePlatformFeatures } from '../../context/platform-features-context'
import {
  ACCOMMODATION_TYPES,
  createAccommodationProfile,
  fetchAccommodationProfiles,
  notifyInstructors,
  updateAccommodationProfile,
  type AccommodationProfile,
  type AccommodationType,
} from '../../lib/accessibility-api'
import {
  searchAccommodationUsers,
  type AccommodationUserSearchHit,
} from '../../lib/courses-api'
import { formatDateTime } from '../../lib/format'

function learnerLabel(u: AccommodationUserSearchHit): string {
  const dn = u.displayName?.trim()
  if (dn) return `${dn} (${u.email})`
  const name = [u.firstName, u.lastName].filter(Boolean).join(' ').trim()
  if (name) return `${name} (${u.email})`
  return u.email
}

function CreateProfileForm({ onCreated }: { onCreated: () => void }) {
  const [query, setQuery] = useState('')
  const [hits, setHits] = useState<AccommodationUserSearchHit[]>([])
  const [selected, setSelected] = useState<AccommodationUserSearchHit | null>(null)
  const [types, setTypes] = useState<Set<AccommodationType>>(new Set())
  const [timeMultiplier, setTimeMultiplier] = useState('')
  const [effectiveFrom, setEffectiveFrom] = useState('')
  const [effectiveUntil, setEffectiveUntil] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)

  async function search(e: React.FormEvent) {
    e.preventDefault()
    setError(null)
    try {
      setHits(await searchAccommodationUsers(query))
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Could not search learners.')
    }
  }

  function toggleType(t: AccommodationType) {
    setTypes((prev) => {
      const next = new Set(prev)
      if (next.has(t)) next.delete(t)
      else next.add(t)
      return next
    })
  }

  async function submit(e: React.FormEvent) {
    e.preventDefault()
    if (!selected) {
      setError('Select a student first.')
      return
    }
    if (types.size === 0) {
      setError('Select at least one accommodation type.')
      return
    }
    setSubmitting(true)
    setError(null)
    try {
      const customParams: Record<string, unknown> = {}
      const mult = Number.parseFloat(timeMultiplier)
      if (!Number.isNaN(mult) && mult >= 1) customParams.timeMultiplier = mult
      await createAccommodationProfile({
        studentId: selected.id,
        accommodations: Array.from(types),
        customParams,
        effectiveFrom: effectiveFrom || undefined,
        effectiveUntil: effectiveUntil || undefined,
      })
      setSelected(null)
      setQuery('')
      setHits([])
      setTypes(new Set())
      setTimeMultiplier('')
      setEffectiveFrom('')
      setEffectiveUntil('')
      onCreated()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Could not create profile.')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <section
      aria-label="Create accommodation profile"
      className="rounded-xl border border-slate-200 p-4 dark:border-neutral-800"
    >
      <h2 className="text-sm font-semibold text-slate-900 dark:text-neutral-100">New accommodation profile</h2>

      <form onSubmit={search} className="mt-3 flex flex-wrap gap-2">
        <label className="sr-only" htmlFor="acc-student-search">
          Search students by name, email, or campus id
        </label>
        <input
          id="acc-student-search"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder="Search students by name, email, or campus id…"
          className="min-w-0 flex-1 rounded-lg border border-slate-300 px-3 py-1.5 text-sm dark:border-neutral-700 dark:bg-neutral-900"
        />
        <button
          type="submit"
          className="rounded-lg border border-slate-300 px-3 py-1.5 text-sm font-medium dark:border-neutral-700"
        >
          Search
        </button>
      </form>

      {hits.length > 0 && !selected && (
        <ul className="mt-2 space-y-1" aria-label="Search results">
          {hits.map((hit) => (
            <li key={hit.id}>
              <button
                type="button"
                onClick={() => setSelected(hit)}
                className="w-full rounded-lg border border-slate-200 px-3 py-1.5 text-start text-sm hover:bg-slate-50 dark:border-neutral-800 dark:hover:bg-neutral-900"
              >
                {learnerLabel(hit)}
              </button>
            </li>
          ))}
        </ul>
      )}

      {selected && (
        <form onSubmit={submit} className="mt-3 space-y-3">
          <p className="text-sm text-slate-700 dark:text-neutral-300">
            Student: <span className="font-medium">{learnerLabel(selected)}</span>{' '}
            <button
              type="button"
              onClick={() => setSelected(null)}
              className="text-xs text-violet-600 underline"
            >
              change
            </button>
          </p>

          <fieldset>
            <legend className="text-xs font-semibold uppercase tracking-wide text-slate-500">
              Accommodation types
            </legend>
            <div className="mt-2 grid gap-2 sm:grid-cols-2">
              {ACCOMMODATION_TYPES.map((t) => (
                <label key={t.value} className="flex items-start gap-2 text-sm" htmlFor={`acc-${t.value}`}>
                  <input
                    id={`acc-${t.value}`}
                    type="checkbox"
                    checked={types.has(t.value)}
                    onChange={() => toggleType(t.value)}
                    aria-describedby={`acc-${t.value}-desc`}
                    className="mt-0.5"
                  />
                  <span>
                    <span className="font-medium text-slate-900 dark:text-neutral-100">{t.label}</span>
                    <span id={`acc-${t.value}-desc`} className="block text-xs text-slate-500">
                      {t.description}
                    </span>
                  </span>
                </label>
              ))}
            </div>
          </fieldset>

          <div className="grid gap-3 sm:grid-cols-3">
            <label className="text-xs font-medium text-slate-600 dark:text-neutral-400">
              Custom time multiplier
              <input
                type="number"
                min="1"
                step="0.25"
                value={timeMultiplier}
                onChange={(e) => setTimeMultiplier(e.target.value)}
                placeholder="e.g. 1.5"
                className="mt-1 w-full rounded-lg border border-slate-300 px-3 py-1.5 text-sm dark:border-neutral-700 dark:bg-neutral-900"
              />
            </label>
            <label className="text-xs font-medium text-slate-600 dark:text-neutral-400">
              Effective from
              <input
                type="date"
                value={effectiveFrom}
                onChange={(e) => setEffectiveFrom(e.target.value)}
                className="mt-1 w-full rounded-lg border border-slate-300 px-3 py-1.5 text-sm dark:border-neutral-700 dark:bg-neutral-900"
              />
            </label>
            <label className="text-xs font-medium text-slate-600 dark:text-neutral-400">
              Effective until
              <input
                type="date"
                value={effectiveUntil}
                onChange={(e) => setEffectiveUntil(e.target.value)}
                className="mt-1 w-full rounded-lg border border-slate-300 px-3 py-1.5 text-sm dark:border-neutral-700 dark:bg-neutral-900"
              />
            </label>
          </div>

          {timeMultiplier && Number.parseFloat(timeMultiplier) >= 1 && (
            <p className="text-xs text-amber-700 dark:text-amber-500">
              This will apply a {Number.parseFloat(timeMultiplier)}× time multiplier to all quizzes for this
              student.
            </p>
          )}

          <button
            type="submit"
            disabled={submitting}
            className="rounded-lg bg-violet-600 px-4 py-2 text-sm font-semibold text-white disabled:opacity-50"
          >
            {submitting ? 'Creating…' : 'Create profile'}
          </button>
        </form>
      )}

      {error && (
        <p role="alert" className="mt-2 text-sm text-red-600 dark:text-red-400">
          {error}
        </p>
      )}
    </section>
  )
}

function ProfileRow({ profile, onChanged }: { profile: AccommodationProfile; onChanged: () => void }) {
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [notice, setNotice] = useState<string | null>(null)

  async function deactivate() {
    setBusy(true)
    setError(null)
    try {
      await updateAccommodationProfile(profile.id, { isActive: false })
      onChanged()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Could not deactivate profile.')
    } finally {
      setBusy(false)
    }
  }

  async function notify() {
    setBusy(true)
    setError(null)
    setNotice(null)
    try {
      const res = await notifyInstructors(profile.id)
      setNotice(`Notified ${res.notifiedInstructorCount} instructor(s).`)
      onChanged()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Could not notify instructors.')
    } finally {
      setBusy(false)
    }
  }

  return (
    <li className="rounded-xl border border-slate-200 p-4 dark:border-neutral-800">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="min-w-0">
          <p className="flex items-center gap-2 text-sm font-semibold text-slate-900 dark:text-neutral-100">
            <ShieldCheck className="h-4 w-4 text-violet-500" aria-hidden="true" />
            <span className="font-mono">{profile.studentId.slice(0, 8)}</span>
            <span className={profile.isActive ? 'text-emerald-600' : 'text-slate-400'}>
              {profile.isActive ? 'Active' : 'Inactive'}
            </span>
          </p>
          <p className="mt-1 text-xs text-slate-600 dark:text-neutral-400">{profile.labels.join(', ')}</p>
          <p className="text-xs text-slate-500">
            Effective {profile.effectiveFrom}
            {profile.effectiveUntil ? ` – ${profile.effectiveUntil}` : ''}
            {profile.notifiedAt ? ` · Notified ${formatDateTime(profile.notifiedAt)}` : ''}
          </p>
        </div>
        {profile.isActive && (
          <div className="flex flex-wrap gap-2">
            <button
              type="button"
              disabled={busy}
              onClick={() => void notify()}
              className="rounded-lg border border-slate-300 px-3 py-1.5 text-xs font-medium disabled:opacity-50 dark:border-neutral-700"
            >
              Notify instructors
            </button>
            <button
              type="button"
              disabled={busy}
              onClick={() => void deactivate()}
              className="rounded-lg border border-red-200 px-3 py-1.5 text-xs font-medium text-red-700 hover:bg-red-50 disabled:opacity-50 dark:border-red-900/40 dark:text-red-400"
            >
              Deactivate
            </button>
          </div>
        )}
      </div>
      {notice && <p className="mt-2 text-xs text-emerald-700 dark:text-emerald-500">{notice}</p>}
      {error && (
        <p role="alert" className="mt-2 text-xs text-red-600 dark:text-red-400">
          {error}
        </p>
      )}
    </li>
  )
}

export default function AccessibilityServicesPage() {
  const { ffAccessibilityIntake, loading: featuresLoading } = usePlatformFeatures()
  const [profiles, setProfiles] = useState<AccommodationProfile[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    try {
      setProfiles(await fetchAccommodationProfiles())
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Could not load profiles.')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    if (featuresLoading || !ffAccessibilityIntake) {
      setLoading(false)
      return
    }
    void load()
  }, [ffAccessibilityIntake, featuresLoading, load])

  if (!ffAccessibilityIntake && !featuresLoading) {
    return (
      <div className="mx-auto max-w-4xl p-6">
        <h1 className="mb-2 text-xl font-semibold">Accessibility services</h1>
        <p className="text-sm text-slate-600">Accessibility services intake is not enabled for this platform.</p>
      </div>
    )
  }

  return (
    <div className="mx-auto max-w-4xl space-y-6 p-6">
      <div>
        <h1 className="text-xl font-semibold">Accessibility services</h1>
        <p className="mt-1 text-sm text-slate-600 dark:text-neutral-400">
          Create accommodation profiles for students. Accommodations apply automatically to all of a
          student's quizzes; disability documentation is never stored here (ADA / Section 504 / FERPA).
        </p>
      </div>

      <CreateProfileForm onCreated={() => void load()} />

      {error && (
        <p role="alert" className="text-sm text-red-600 dark:text-red-400">
          {error}
        </p>
      )}

      {loading ? (
        <p className="text-sm text-slate-500">Loading…</p>
      ) : profiles.length === 0 ? (
        <p className="text-sm text-slate-500">No accommodation profiles yet.</p>
      ) : (
        <ul className="space-y-3" aria-label="Accommodation profiles">
          {profiles.map((p) => (
            <ProfileRow key={p.id} profile={p} onChanged={() => void load()} />
          ))}
        </ul>
      )}
    </div>
  )
}
