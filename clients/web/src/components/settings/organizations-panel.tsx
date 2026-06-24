import { useCallback, useEffect, useState, type FormEvent } from 'react'
import { Building2, Plus, RefreshCw } from 'lucide-react'
import { authorizedFetch, tryRefreshSession } from '../../lib/api'
import { readApiErrorMessage } from '../../lib/errors'
import { toastMutationError, toastSaveOk } from '../../lib/lms-toast'
import {
  normalizeOrgSlug,
  orgLoginPath,
  suggestOrgSlugFromName,
  validateOrgSlug,
} from '../../lib/org-slug'

type OrgRow = {
  id: string
  slug: string
  name: string
  status: string
  maxUsers?: number | null
  maxCourses?: number | null
  dataRegion: string
  userCount: number
  courseCount: number
  createdAt: string
}

export function OrganizationsPanel() {
  const [orgs, setOrgs] = useState<OrgRow[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [newName, setNewName] = useState('')
  const [newSlug, setNewSlug] = useState('')
  const [slugTouched, setSlugTouched] = useState(false)
  const [slugError, setSlugError] = useState<string | null>(null)
  const [creating, setCreating] = useState(false)

  const load = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const res = await authorizedFetch('/api/v1/admin/orgs?limit=200')
      const raw: unknown = await res.json().catch(() => ({}))
      if (!res.ok) throw new Error(readApiErrorMessage(raw))
      const data = raw as { organizations?: OrgRow[] }
      setOrgs(data.organizations ?? [])
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load organizations.')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    void load()
  }, [load])

  useEffect(() => {
    if (slugTouched) return
    setNewSlug(suggestOrgSlugFromName(newName))
  }, [newName, slugTouched])

  useEffect(() => {
    setSlugError(validateOrgSlug(newSlug))
  }, [newSlug])

  async function createOrg(e: FormEvent) {
    e.preventDefault()
    const name = newName.trim()
    const slug = normalizeOrgSlug(newSlug)
    const validation = validateOrgSlug(slug)
    if (!name || validation) {
      setSlugError(validation)
      return
    }
    setCreating(true)
    try {
      const res = await authorizedFetch('/api/v1/admin/orgs', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name, slug }),
      })
      const raw: unknown = await res.json().catch(() => ({}))
      if (!res.ok) throw new Error(readApiErrorMessage(raw))
      const created = raw as { slug?: string }
      const loginPath = orgLoginPath(created.slug ?? slug)
      await tryRefreshSession()
      toastSaveOk(`Organization created. Your account is now in this tenant. Sign in at ${loginPath}`)
      setNewName('')
      setNewSlug('')
      setSlugTouched(false)
      await load()
    } catch (err) {
      toastMutationError(err instanceof Error ? err.message : 'Request failed.')
    } finally {
      setCreating(false)
    }
  }

  async function setStatus(id: string, name: string, next: 'active' | 'suspended') {
    if (next === 'suspended' && !window.confirm(`Suspend organization “${name}”? Users in this org will be blocked from signing in.`)) {
      return
    }
    try {
      const res = await authorizedFetch(`/api/v1/admin/orgs/${id}`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ status: next }),
      })
      const raw: unknown = await res.json().catch(() => ({}))
      if (!res.ok) throw new Error(readApiErrorMessage(raw))
      toastSaveOk(next === 'suspended' ? 'Organization suspended.' : 'Organization reactivated.')
      await load()
    } catch (err) {
      toastMutationError(err instanceof Error ? err.message : 'Request failed.')
    }
  }

  const previewSlug = normalizeOrgSlug(newSlug)

  return (
    <div className="mt-6 space-y-6">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <p className="text-sm text-slate-600 dark:text-neutral-400">
          Provision tenants and monitor usage. Each organization gets a unique short name used for sign-in URLs.
        </p>
        <button
          type="button"
          onClick={() => void load()}
          disabled={loading}
          className="inline-flex items-center gap-2 rounded-xl border border-slate-200 bg-white px-3 py-2 text-sm font-medium text-slate-800 shadow-sm transition-[background-color,color,border-color] hover:bg-slate-50 disabled:opacity-50 dark:border-neutral-600 dark:bg-neutral-900 dark:text-neutral-100 dark:hover:bg-neutral-800"
        >
          <RefreshCw className={`h-4 w-4 ${loading ? 'animate-spin' : ''}`} aria-hidden />
          Refresh
        </button>
      </div>

      <form
        onSubmit={createOrg}
        className="rounded-xl border border-slate-200 bg-slate-50/80 p-4 dark:border-neutral-600 dark:bg-neutral-800/40"
        aria-labelledby="new-org-heading"
      >
        <h3 id="new-org-heading" className="flex items-center gap-2 text-sm font-semibold text-slate-900 dark:text-neutral-100">
          <Plus className="h-4 w-4" aria-hidden />
          New organization
        </h3>
        <div className="mt-3 flex flex-col gap-3 sm:flex-row sm:items-end">
          <label className="flex min-w-0 flex-1 flex-col gap-1 text-xs font-medium text-slate-700 dark:text-neutral-300">
            Name
            <input
              value={newName}
              onChange={(e) => setNewName(e.target.value)}
              className="rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm dark:border-neutral-600 dark:bg-neutral-900"
              placeholder="Chase's Org"
              autoComplete="organization"
            />
          </label>
          <label className="flex min-w-0 flex-1 flex-col gap-1 text-xs font-medium text-slate-700 dark:text-neutral-300">
            Short name (slug)
            <input
              value={newSlug}
              onChange={(e) => {
                setSlugTouched(true)
                setNewSlug(normalizeOrgSlug(e.target.value))
              }}
              className="rounded-lg border border-slate-200 bg-white px-3 py-2 font-mono text-sm dark:border-neutral-600 dark:bg-neutral-900"
              placeholder="chase"
              autoComplete="off"
              aria-invalid={slugError ? true : undefined}
              aria-describedby={slugError ? 'new-org-slug-hint new-org-slug-error' : 'new-org-slug-hint'}
            />
          </label>
          <button
            type="submit"
            disabled={creating || !newName.trim() || !!slugError}
            className="inline-flex shrink-0 items-center justify-center gap-2 rounded-xl bg-indigo-600 px-4 py-2 text-sm font-semibold text-white shadow-sm transition-[background-color,color,border-color] hover:bg-indigo-700 disabled:cursor-not-allowed disabled:opacity-50"
          >
            <Building2 className="h-4 w-4" aria-hidden />
            Create
          </button>
        </div>
        <p id="new-org-slug-hint" className="mt-2 text-xs text-slate-500 dark:text-neutral-400">
          {previewSlug ? (
            <>
              Sign-in URL:{' '}
              <code className="rounded bg-slate-100 px-1.5 py-0.5 font-mono text-[11px] text-slate-700 dark:bg-neutral-800 dark:text-neutral-200">
                {orgLoginPath(previewSlug)}
              </code>
            </>
          ) : (
            'Choose a short, memorable slug such as chase or riverdale-usd.'
          )}
        </p>
        {slugError && (
          <p id="new-org-slug-error" className="mt-1 text-xs text-rose-700 dark:text-rose-300" role="alert">
            {slugError}
          </p>
        )}
      </form>

      {error && (
        <p className="rounded-lg border border-rose-200 bg-rose-50 px-3 py-2 text-sm text-rose-800 dark:border-rose-900/50 dark:bg-rose-950/40 dark:text-rose-100" role="alert">
          {error}
        </p>
      )}

      {loading && orgs.length === 0 ? (
        <div className="space-y-2" aria-busy="true" aria-label="Loading organizations">
          {[0, 1, 2].map((i) => (
            <div key={i} className="h-12 animate-pulse rounded-xl bg-slate-100 dark:bg-neutral-800" />
          ))}
        </div>
      ) : orgs.length === 0 ? (
        <p className="rounded-xl border border-dashed border-slate-200 px-4 py-8 text-center text-sm text-slate-600 dark:border-neutral-600 dark:text-neutral-400">
          No organizations yet — create your first one.
        </p>
      ) : (
        <div className="overflow-x-auto rounded-xl border border-slate-200 dark:border-neutral-600">
          <table className="min-w-full divide-y divide-slate-200 text-start text-sm dark:divide-neutral-600">
            <thead className="bg-slate-50 dark:bg-neutral-800/80">
              <tr>
                <th scope="col" className="px-3 py-2 font-medium text-slate-700 dark:text-neutral-200">
                  Name
                </th>
                <th scope="col" className="px-3 py-2 font-medium text-slate-700 dark:text-neutral-200">
                  Slug
                </th>
                <th scope="col" className="px-3 py-2 font-medium text-slate-700 dark:text-neutral-200">
                  Sign in
                </th>
                <th scope="col" className="px-3 py-2 font-medium text-slate-700 dark:text-neutral-200">
                  Status
                </th>
                <th scope="col" className="px-3 py-2 font-medium text-slate-700 dark:text-neutral-200">
                  Users
                </th>
                <th scope="col" className="px-3 py-2 font-medium text-slate-700 dark:text-neutral-200">
                  Courses
                </th>
                <th scope="col" className="px-3 py-2 font-medium text-slate-700 dark:text-neutral-200">
                  Actions
                </th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-200 bg-white dark:divide-neutral-600 dark:bg-neutral-900">
              {orgs.map((o) => (
                <tr key={o.id} className="hover:bg-slate-50 dark:hover:bg-neutral-800/60">
                  <th scope="row" className="whitespace-nowrap px-3 py-2.5 font-medium text-slate-900 dark:text-neutral-100">
                    {o.name}
                  </th>
                  <td className="whitespace-nowrap px-3 py-2.5 font-mono text-xs text-slate-600 dark:text-neutral-300">{o.slug}</td>
                  <td className="whitespace-nowrap px-3 py-2.5">
                    <a
                      href={orgLoginPath(o.slug)}
                      className="font-mono text-xs text-indigo-600 hover:underline dark:text-indigo-400"
                    >
                      {orgLoginPath(o.slug)}
                    </a>
                  </td>
                  <td className="whitespace-nowrap px-3 py-2.5 text-slate-600 dark:text-neutral-300">{o.status}</td>
                  <td className="whitespace-nowrap px-3 py-2.5 text-slate-600 dark:text-neutral-300">{o.userCount}</td>
                  <td className="whitespace-nowrap px-3 py-2.5 text-slate-600 dark:text-neutral-300">{o.courseCount}</td>
                  <td className="px-3 py-2.5">
                    {o.slug === 'default' ? (
                      <span className="text-xs text-slate-400 dark:text-neutral-500">—</span>
                    ) : o.status === 'active' ? (
                      <button
                        type="button"
                        className="rounded-lg border border-slate-200 bg-white px-2.5 py-1.5 text-xs font-medium text-slate-800 hover:border-amber-200 hover:bg-amber-50 dark:border-neutral-600 dark:bg-neutral-900 dark:text-neutral-100"
                        onClick={() => void setStatus(o.id, o.name, 'suspended')}
                      >
                        Suspend
                      </button>
                    ) : o.status === 'suspended' ? (
                      <button
                        type="button"
                        className="rounded-lg border border-slate-200 bg-white px-2.5 py-1.5 text-xs font-medium text-slate-800 hover:border-emerald-200 hover:bg-emerald-50 dark:border-neutral-600 dark:bg-neutral-900 dark:text-neutral-100"
                        onClick={() => void setStatus(o.id, o.name, 'active')}
                      >
                        Reactivate
                      </button>
                    ) : (
                      <span className="text-xs text-slate-400 dark:text-neutral-500">Deleted</span>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}