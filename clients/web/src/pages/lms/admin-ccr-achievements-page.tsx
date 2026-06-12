import { useCallback, useId, useState, type FormEvent } from 'react'
import { Link } from 'react-router-dom'
import { Award } from 'lucide-react'
import { usePlatformFeatures } from '../../context/platform-features-context'
import { usePermissions } from '../../context/use-permissions'
import { createAdminCCRAchievement, type CCRAchievement } from '../../lib/ccr-api'
import {
  searchAccommodationUsers,
  type AccommodationUserSearchHit,
} from '../../lib/courses-api'
import { PERM_ACCOMMODATIONS_MANAGE } from '../../lib/rbac-api'
import { LmsPage } from './lms-page'

function formatLearnerLabel(u: AccommodationUserSearchHit): string {
  const fn = u.firstName?.trim() ?? ''
  const ln = u.lastName?.trim() ?? ''
  const combined = [fn, ln].filter(Boolean).join(' ').trim()
  if (combined.length > 0) return combined
  const dn = u.displayName?.trim()
  if (dn) return dn
  return u.email
}

export default function AdminCCRAchievementsPage() {
  const formId = useId()
  const { allows, loading: permLoading } = usePermissions()
  const { ffCoCurricularTranscript } = usePlatformFeatures()
  const canManage = !permLoading && allows(PERM_ACCOMMODATIONS_MANAGE)

  const [searchInput, setSearchInput] = useState('')
  const [searchHits, setSearchHits] = useState<AccommodationUserSearchHit[]>([])
  const [searchBusy, setSearchBusy] = useState(false)
  const [searchError, setSearchError] = useState<string | null>(null)
  const [selectedUser, setSelectedUser] = useState<AccommodationUserSearchHit | null>(null)

  const [title, setTitle] = useState('')
  const [description, setDescription] = useState('')
  const [issuedAt, setIssuedAt] = useState('')
  const [evidenceUrl, setEvidenceUrl] = useState('')
  const [outcomeTags, setOutcomeTags] = useState('')
  const [saveError, setSaveError] = useState<string | null>(null)
  const [saveBusy, setSaveBusy] = useState(false)
  const [added, setAdded] = useState<CCRAchievement[]>([])

  const runSearch = useCallback(async () => {
    const q = searchInput.trim()
    if (!q) {
      setSearchError('Enter an email, part of a name, campus id (sid), or user id.')
      setSearchHits([])
      return
    }
    setSearchBusy(true)
    setSearchError(null)
    try {
      const hits = await searchAccommodationUsers(q)
      setSearchHits(hits)
      if (hits.length === 0) {
        setSearchError('No matching users. Try a different spelling or a longer fragment.')
      }
    } catch (e) {
      setSearchHits([])
      setSearchError(e instanceof Error ? e.message : 'Search failed.')
    } finally {
      setSearchBusy(false)
    }
  }, [searchInput])

  function pickUser(hit: AccommodationUserSearchHit) {
    setSelectedUser(hit)
    setSearchHits([])
    setSearchError(null)
    setAdded([])
  }

  function clearSelection() {
    setSelectedUser(null)
    setAdded([])
    setSaveError(null)
  }

  async function onCreate(e: FormEvent) {
    e.preventDefault()
    if (!selectedUser) {
      setSaveError('Search and select a learner first.')
      return
    }
    const trimmedTitle = title.trim()
    if (!trimmedTitle) {
      setSaveError('Title is required.')
      return
    }
    setSaveBusy(true)
    setSaveError(null)
    try {
      const tags = outcomeTags
        .split(',')
        .map((t) => t.trim())
        .filter(Boolean)
      const body: Parameters<typeof createAdminCCRAchievement>[1] = {
        title: trimmedTitle,
      }
      const desc = description.trim()
      if (desc) body.description = desc
      if (issuedAt.trim()) {
        body.issuedAt = new Date(issuedAt).toISOString()
      }
      const evidence = evidenceUrl.trim()
      if (evidence) body.evidenceUrl = evidence
      if (tags.length > 0) body.outcomeTags = tags

      const row = await createAdminCCRAchievement(selectedUser.id, body)
      setAdded((prev) => [row, ...prev])
      setTitle('')
      setDescription('')
      setIssuedAt('')
      setEvidenceUrl('')
      setOutcomeTags('')
    } catch (err) {
      setSaveError(err instanceof Error ? err.message : 'Save failed.')
    } finally {
      setSaveBusy(false)
    }
  }

  if (!ffCoCurricularTranscript) {
    return (
      <LmsPage title="CCR achievements">
        <p role="alert">Co-curricular transcript is not enabled for this institution.</p>
      </LmsPage>
    )
  }

  if (!canManage) {
    return (
      <LmsPage title="CCR achievements">
        <p role="alert">You do not have permission to manage co-curricular records.</p>
      </LmsPage>
    )
  }

  return (
    <LmsPage title="CCR achievements">
      <div className="space-y-6">
        <div className="flex flex-wrap items-start justify-between gap-4">
          <div>
            <div className="flex items-center gap-2">
              <Award className="h-5 w-5 text-violet-600" aria-hidden />
              <h1 className="text-xl font-semibold text-slate-900 dark:text-neutral-100">
                Co-curricular achievements
              </h1>
            </div>
            <p className="mt-1 text-sm text-slate-600 dark:text-neutral-400">
              Add manual extracurricular records (leadership roles, service hours, workshops) to a student&apos;s
              comprehensive learner record.
            </p>
          </div>
          <Link
            to="/me/ccr"
            className="text-sm text-violet-700 underline hover:text-violet-900 dark:text-violet-300"
          >
            Preview student CCR view
          </Link>
        </div>

        <section aria-label="Find learner" className="rounded-xl border border-slate-200 bg-white p-4 dark:border-neutral-700 dark:bg-neutral-900">
          <h2 className="text-base font-semibold">Find learner</h2>
          <div className="mt-3 flex flex-wrap gap-2">
            <input
              type="search"
              value={searchInput}
              onChange={(e) => setSearchInput(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter') void runSearch()
              }}
              placeholder="Email, name, sid, or user id"
              className="min-w-[240px] flex-1 rounded-lg border border-slate-200 px-3 py-2 text-sm dark:border-neutral-600 dark:bg-neutral-950"
            />
            <button
              type="button"
              onClick={() => void runSearch()}
              disabled={searchBusy}
              className="rounded-lg bg-slate-800 px-4 py-2 text-sm font-medium text-white hover:bg-slate-900 disabled:opacity-60 dark:bg-neutral-200 dark:text-neutral-900"
            >
              {searchBusy ? 'Searching…' : 'Search'}
            </button>
          </div>
          {searchError ? (
            <p role="alert" className="mt-2 text-sm text-red-700 dark:text-red-300">
              {searchError}
            </p>
          ) : null}
          {searchHits.length > 0 ? (
            <ul className="mt-3 space-y-1">
              {searchHits.map((hit) => (
                <li key={hit.id}>
                  <button
                    type="button"
                    onClick={() => pickUser(hit)}
                    className="w-full rounded-lg border border-slate-200 px-3 py-2 text-left text-sm hover:bg-slate-50 dark:border-neutral-600 dark:hover:bg-neutral-800"
                  >
                    <span className="font-medium">{formatLearnerLabel(hit)}</span>
                    <span className="ml-2 text-slate-500">{hit.email}</span>
                  </button>
                </li>
              ))}
            </ul>
          ) : null}
        </section>

        {selectedUser ? (
          <section aria-label="Add achievement" className="rounded-xl border border-slate-200 bg-white p-4 dark:border-neutral-700 dark:bg-neutral-900">
            <div className="flex flex-wrap items-center justify-between gap-2">
              <h2 className="text-base font-semibold">
                Add record for {formatLearnerLabel(selectedUser)}
              </h2>
              <button
                type="button"
                onClick={clearSelection}
                className="text-sm text-slate-600 underline hover:text-slate-900 dark:text-neutral-400"
              >
                Change learner
              </button>
            </div>

            <form id={formId} onSubmit={(e) => void onCreate(e)} className="mt-4 grid gap-4 sm:grid-cols-2">
              <label className="block sm:col-span-2">
                <span className="text-sm font-medium">Title *</span>
                <input
                  type="text"
                  value={title}
                  onChange={(e) => setTitle(e.target.value)}
                  required
                  placeholder="e.g. Student Government President"
                  className="mt-1 w-full rounded-lg border border-slate-200 px-3 py-2 text-sm dark:border-neutral-600 dark:bg-neutral-950"
                />
              </label>
              <label className="block sm:col-span-2">
                <span className="text-sm font-medium">Description</span>
                <textarea
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
                  rows={3}
                  className="mt-1 w-full rounded-lg border border-slate-200 px-3 py-2 text-sm dark:border-neutral-600 dark:bg-neutral-950"
                />
              </label>
              <label className="block">
                <span className="text-sm font-medium">Issued date</span>
                <input
                  type="datetime-local"
                  value={issuedAt}
                  onChange={(e) => setIssuedAt(e.target.value)}
                  className="mt-1 w-full rounded-lg border border-slate-200 px-3 py-2 text-sm dark:border-neutral-600 dark:bg-neutral-950"
                />
              </label>
              <label className="block">
                <span className="text-sm font-medium">Evidence URL</span>
                <input
                  type="url"
                  value={evidenceUrl}
                  onChange={(e) => setEvidenceUrl(e.target.value)}
                  placeholder="https://…"
                  className="mt-1 w-full rounded-lg border border-slate-200 px-3 py-2 text-sm dark:border-neutral-600 dark:bg-neutral-950"
                />
              </label>
              <label className="block sm:col-span-2">
                <span className="text-sm font-medium">Outcome tags (comma-separated)</span>
                <input
                  type="text"
                  value={outcomeTags}
                  onChange={(e) => setOutcomeTags(e.target.value)}
                  placeholder="Leadership, Civic engagement"
                  className="mt-1 w-full rounded-lg border border-slate-200 px-3 py-2 text-sm dark:border-neutral-600 dark:bg-neutral-950"
                />
              </label>
              {saveError ? (
                <p role="alert" className="sm:col-span-2 text-sm text-red-700 dark:text-red-300">
                  {saveError}
                </p>
              ) : null}
              <div className="sm:col-span-2">
                <button
                  type="submit"
                  disabled={saveBusy}
                  className="rounded-lg bg-violet-600 px-4 py-2 text-sm font-medium text-white hover:bg-violet-700 disabled:opacity-60"
                >
                  {saveBusy ? 'Saving…' : 'Add achievement'}
                </button>
              </div>
            </form>

            {added.length > 0 ? (
              <div className="mt-6">
                <h3 className="text-sm font-semibold text-slate-700 dark:text-neutral-300">Added this session</h3>
                <ul className="mt-2 space-y-2">
                  {added.map((row) => (
                    <li
                      key={row.id}
                      className="rounded-lg border border-green-200 bg-green-50 px-3 py-2 text-sm dark:border-green-900/40 dark:bg-green-950/30"
                    >
                      <span className="font-medium">{row.title}</span>
                      <span className="ml-2 text-slate-500">{row.issuedAt}</span>
                    </li>
                  ))}
                </ul>
              </div>
            ) : null}
          </section>
        ) : null}
      </div>
    </LmsPage>
  )
}
