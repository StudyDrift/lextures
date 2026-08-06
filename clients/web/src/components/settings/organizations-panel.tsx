import { useCallback, useEffect, useState, type FormEvent } from 'react'
import { useTranslation } from 'react-i18next'
import { Building2, Plus, RefreshCw } from 'lucide-react'
import { useConfirm } from '../use-confirm'
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
  const { t } = useTranslation('common')
  const { confirm, ConfirmDialogHost } = useConfirm()
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
    if (
      next === 'suspended' &&
      !(await confirm({ title: t('organizations.suspend.title', { name }), variant: 'danger' }))
    ) {
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
        <p className="text-sm text-fg-muted">
          Provision tenants and monitor usage. Each organization gets a unique short name used for sign-in URLs.
        </p>
        <button
          type="button"
          onClick={() => void load()}
          disabled={loading}
          className="inline-flex items-center gap-2 rounded-xl border border-border-default bg-surface-raised px-3 py-2 text-sm font-medium text-fg-default shadow-sm transition-[background-color,color,border-color] hover:bg-surface-base disabled:opacity-50 dark:border-border-default dark:bg-surface-raised dark:text-fg-default dark:hover:bg-surface-overlay"
        >
          <RefreshCw className={`h-4 w-4 ${loading ? 'animate-spin' : ''}`} aria-hidden />
          Refresh
        </button>
      </div>

      <form
        onSubmit={createOrg}
        className="rounded-xl border border-border-default bg-slate-50/80 p-4 dark:border-border-default/40"
        aria-labelledby="new-org-heading"
      >
        <h3 id="new-org-heading" className="flex items-center gap-2 text-sm font-semibold text-fg-default">
          <Plus className="h-4 w-4" aria-hidden />
          New organization
        </h3>
        <div className="mt-3 flex flex-col gap-3 sm:flex-row sm:items-end">
          <label className="flex min-w-0 flex-1 flex-col gap-1 text-xs font-medium text-fg-muted">
            Name
            <input
              value={newName}
              onChange={(e) => setNewName(e.target.value)}
              className="rounded-lg border border-border-default bg-surface-raised px-3 py-2 text-sm dark:border-border-default dark:bg-surface-raised"
              placeholder="Chase's Org"
              autoComplete="organization"
            />
          </label>
          <label className="flex min-w-0 flex-1 flex-col gap-1 text-xs font-medium text-fg-muted">
            Short name (slug)
            <input
              value={newSlug}
              onChange={(e) => {
                setSlugTouched(true)
                setNewSlug(normalizeOrgSlug(e.target.value))
              }}
              className="rounded-lg border border-border-default bg-surface-raised px-3 py-2 font-mono text-sm dark:border-border-default dark:bg-surface-raised"
              placeholder="chase"
              autoComplete="off"
              aria-invalid={slugError ? true : undefined}
              aria-describedby={slugError ? 'new-org-slug-hint new-org-slug-error' : 'new-org-slug-hint'}
            />
          </label>
          <button
            type="submit"
            disabled={creating || !newName.trim() || !!slugError}
            className="inline-flex shrink-0 items-center justify-center gap-2 rounded-xl bg-accent-solid px-4 py-2 text-sm font-semibold text-white shadow-sm transition-[background-color,color,border-color] hover:bg-accent disabled:cursor-not-allowed disabled:opacity-50"
          >
            <Building2 className="h-4 w-4" aria-hidden />
            Create
          </button>
        </div>
        <p id="new-org-slug-hint" className="mt-2 text-xs text-fg-muted">
          {previewSlug ? (
            <>
              Sign-in URL:{' '}
              <code className="rounded bg-surface-sunken px-1.5 py-0.5 font-mono text-[11px] text-fg-muted dark:bg-surface-overlay dark:text-fg-default">
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
            <div key={i} className="h-12 animate-pulse rounded-xl bg-surface-sunken" />
          ))}
        </div>
      ) : orgs.length === 0 ? (
        <p className="rounded-xl border border-dashed border-border-default px-4 py-8 text-center text-sm text-fg-muted dark:border-border-default dark:text-fg-muted">
          No organizations yet — create your first one.
        </p>
      ) : (
        <div className="overflow-x-auto rounded-xl border border-border-default">
          <table className="min-w-full divide-y divide-slate-200 text-start text-sm dark:divide-neutral-600">
            <thead className="bg-surface-sunken/80">
              <tr>
                <th scope="col" className="px-3 py-2 font-medium text-fg-default">
                  Name
                </th>
                <th scope="col" className="px-3 py-2 font-medium text-fg-default">
                  Slug
                </th>
                <th scope="col" className="px-3 py-2 font-medium text-fg-default">
                  Sign in
                </th>
                <th scope="col" className="px-3 py-2 font-medium text-fg-default">
                  Status
                </th>
                <th scope="col" className="px-3 py-2 font-medium text-fg-default">
                  Users
                </th>
                <th scope="col" className="px-3 py-2 font-medium text-fg-default">
                  Courses
                </th>
                <th scope="col" className="px-3 py-2 font-medium text-fg-default">
                  Actions
                </th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-200 bg-surface-raised dark:divide-neutral-600 dark:bg-surface-raised">
              {orgs.map((o) => (
                <tr key={o.id} className="hover:bg-surface-base dark:hover:bg-neutral-800/60">
                  <th scope="row" className="whitespace-nowrap px-3 py-2.5 font-medium text-fg-default">
                    {o.name}
                  </th>
                  <td className="whitespace-nowrap px-3 py-2.5 font-mono text-xs text-fg-muted">{o.slug}</td>
                  <td className="whitespace-nowrap px-3 py-2.5">
                    <a
                      href={orgLoginPath(o.slug)}
                      className="font-mono text-xs text-accent-fg hover:underline dark:text-indigo-400"
                    >
                      {orgLoginPath(o.slug)}
                    </a>
                  </td>
                  <td className="whitespace-nowrap px-3 py-2.5 text-fg-muted">{o.status}</td>
                  <td className="whitespace-nowrap px-3 py-2.5 text-fg-muted">{o.userCount}</td>
                  <td className="whitespace-nowrap px-3 py-2.5 text-fg-muted">{o.courseCount}</td>
                  <td className="px-3 py-2.5">
                    {o.slug === 'default' ? (
                      <span className="text-xs text-fg-subtle">—</span>
                    ) : o.status === 'active' ? (
                      <button
                        type="button"
                        className="rounded-lg border border-border-default bg-surface-raised px-2.5 py-1.5 text-xs font-medium text-fg-default hover:border-amber-200 hover:bg-amber-50 dark:border-border-default dark:bg-surface-raised dark:text-fg-default"
                        onClick={() => void setStatus(o.id, o.name, 'suspended')}
                      >
                        Suspend
                      </button>
                    ) : o.status === 'suspended' ? (
                      <button
                        type="button"
                        className="rounded-lg border border-border-default bg-surface-raised px-2.5 py-1.5 text-xs font-medium text-fg-default hover:border-emerald-200 hover:bg-emerald-50 dark:border-border-default dark:bg-surface-raised dark:text-fg-default"
                        onClick={() => void setStatus(o.id, o.name, 'active')}
                      >
                        Reactivate
                      </button>
                    ) : (
                      <span className="text-xs text-fg-subtle">Deleted</span>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
      {ConfirmDialogHost}
    </div>
  )
}