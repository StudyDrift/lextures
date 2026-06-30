import { useCallback, useEffect, useId, useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { usePlatformFeatures } from '../../context/platform-features-context'
import {
  fetchAdminConsoleCapabilities,
  fetchAdminUsers,
  fetchCustomFields,
  fetchAdminUser,
  patchAdminUser,
  patchAdminUserCustomFields,
  startImpersonation,
  usersExportUrl,
  type AdminUser,
  type CustomFieldDefinition,
  type Paginated,
} from '../../lib/admin-console-api'
import { startImpersonationSession } from '../../lib/impersonation'

const ROLES = ['', 'student', 'instructor', 'ta', 'admin']
const PAGE_SIZES = [25, 50, 100]

export default function AdminUsers() {
  const { t } = useTranslation('common')
  const titleId = useId()
  const { impersonationEnabled, customFieldsEnabled } = usePlatformFeatures()
  const [searchParams] = useSearchParams()
  const orgId = searchParams.get('orgId')
  const [q, setQ] = useState('')
  const [role, setRole] = useState('')
  const [page, setPage] = useState(1)
  const [perPage, setPerPage] = useState(25)
  const [data, setData] = useState<Paginated<AdminUser> | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState<string | null>(null)
  const [canManage, setCanManage] = useState(false)

  useEffect(() => {
    let cancelled = false
    void fetchAdminConsoleCapabilities()
      .then((caps) => {
        if (!cancelled) setCanManage(caps.canManage)
      })
      .catch(() => {
        if (!cancelled) setCanManage(false)
      })
    return () => {
      cancelled = true
    }
  }, [])

  const load = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      setData(
        await fetchAdminUsers({ orgId, q: q.trim() || undefined, role: role || undefined, page, perPage }),
      )
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load users.')
    } finally {
      setLoading(false)
    }
  }, [orgId, q, role, page, perPage])

  useEffect(() => {
    void load()
  }, [load])

  async function deactivate(user: AdminUser) {
    if (!window.confirm(`Deactivate ${user.email}? They will not be able to sign in.`)) return
    setBusy(user.id)
    try {
      await patchAdminUser(user.id, { active: false })
      await load()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to deactivate user.')
    } finally {
      setBusy(null)
    }
  }

  async function viewAs(user: AdminUser) {
    const name = user.displayName?.trim() || user.email
    const msg = t('impersonation.confirm', {
      name,
      defaultValue:
        'You are about to view the application as {{name}}. All writes will be blocked. Continue?',
    })
    if (!window.confirm(msg)) return
    setBusy(user.id)
    try {
      const result = await startImpersonation(user.id)
      startImpersonationSession(result.impersonation_token)
      window.location.assign('/')
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to start impersonation.')
      setBusy(null)
    }
  }

  const showViewAs = impersonationEnabled && canManage

  const [fieldDefs, setFieldDefs] = useState<CustomFieldDefinition[]>([])
  const [expandedUserId, setExpandedUserId] = useState<string | null>(null)
  const [fieldValues, setFieldValues] = useState<Record<string, string>>({})
  const [fieldBusy, setFieldBusy] = useState(false)

  useEffect(() => {
    if (!customFieldsEnabled) return
    void fetchCustomFields('user', orgId)
      .then(setFieldDefs)
      .catch(() => setFieldDefs([]))
  }, [customFieldsEnabled, orgId])

  async function openCustomFields(user: AdminUser) {
    if (expandedUserId === user.id) {
      setExpandedUserId(null)
      return
    }
    setExpandedUserId(user.id)
    setFieldValues({})
    try {
      const detail = await fetchAdminUser(user.id, orgId, true)
      const next: Record<string, string> = {}
      for (const def of fieldDefs) {
        const raw = detail.customFields?.[def.key]
        next[def.key] = raw == null ? '' : String(raw)
      }
      setFieldValues(next)
    } catch {
      setFieldValues({})
    }
  }

  async function saveCustomFields(userId: string) {
    setFieldBusy(true)
    try {
      const payload: Record<string, unknown> = {}
      for (const def of fieldDefs) {
        const raw = fieldValues[def.key] ?? ''
        if (def.fieldType === 'boolean') {
          payload[def.key] = raw === 'true'
        } else if (def.fieldType === 'number' && raw !== '') {
          payload[def.key] = Number(raw)
        } else if (raw !== '') {
          payload[def.key] = raw
        } else {
          payload[def.key] = null
        }
      }
      await patchAdminUserCustomFields(userId, payload, orgId)
      setExpandedUserId(null)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to save custom fields.')
    } finally {
      setFieldBusy(false)
    }
  }

  return (
    <div>
      <h1 id={titleId} className="text-xl font-semibold text-slate-900 dark:text-slate-100">
        Users
      </h1>

      <div className="mt-4 flex flex-wrap items-end justify-between gap-3">
        <div className="flex flex-wrap gap-3">
        <label className="flex flex-col text-sm">
          <span className="mb-1 text-slate-600 dark:text-slate-400">Search</span>
          <input
            type="search"
            value={q}
            onChange={(e) => {
              setQ(e.target.value)
              setPage(1)
            }}
            placeholder="Email or name"
            className="rounded-lg border border-slate-300 px-3 py-2 dark:border-neutral-700 dark:bg-neutral-900"
          />
        </label>
        <label className="flex flex-col text-sm">
          <span className="mb-1 text-slate-600 dark:text-slate-400">Role</span>
          <select
            value={role}
            onChange={(e) => {
              setRole(e.target.value)
              setPage(1)
            }}
            className="rounded-lg border border-slate-300 px-3 py-2 dark:border-neutral-700 dark:bg-neutral-900"
          >
            {ROLES.map((r) => (
              <option key={r || 'all'} value={r}>
                {r || 'All roles'}
              </option>
            ))}
          </select>
        </label>
        <label className="flex flex-col text-sm">
          <span className="mb-1 text-slate-600 dark:text-slate-400">Page size</span>
          <select
            value={perPage}
            onChange={(e) => {
              setPerPage(Number(e.target.value))
              setPage(1)
            }}
            className="rounded-lg border border-slate-300 px-3 py-2 dark:border-neutral-700 dark:bg-neutral-900"
          >
            {PAGE_SIZES.map((n) => (
              <option key={n} value={n}>
                {n}
              </option>
            ))}
          </select>
        </label>
        </div>
        {customFieldsEnabled && canManage && (
          <a
            href={usersExportUrl(orgId)}
            className="rounded-lg border border-slate-300 px-3 py-2 text-sm font-medium text-slate-700 hover:bg-slate-50 dark:border-neutral-700 dark:text-slate-200"
          >
            Export CSV
          </a>
        )}
      </div>

      {error ? (
        <p role="alert" className="mt-4 text-sm text-red-600 dark:text-red-400">
          {error}
        </p>
      ) : null}

      <div className="mt-4 overflow-x-auto rounded-xl border border-slate-200 dark:border-neutral-800">
        <table className="min-w-full text-left text-sm">
          <caption className="sr-only">Organization users</caption>
          <thead className="bg-slate-50 text-slate-600 dark:bg-neutral-950 dark:text-slate-400">
            <tr>
              <th scope="col" className="sticky left-0 bg-slate-50 px-4 py-2 font-medium dark:bg-neutral-950">
                Email
              </th>
              <th scope="col" className="px-4 py-2 font-medium">
                Name
              </th>
              <th scope="col" className="px-4 py-2 font-medium">
                Role
              </th>
              <th scope="col" className="px-4 py-2 font-medium">
                Org role
              </th>
              <th scope="col" className="px-4 py-2 font-medium">
                Status
              </th>
              <th scope="col" className="px-4 py-2 font-medium">
                Actions
              </th>
            </tr>
          </thead>
          <tbody>
            {loading ? (
              <tr>
                <td colSpan={6} className="px-4 py-8 text-center text-slate-500">
                  Loading…
                </td>
              </tr>
            ) : !data?.items.length ? (
              <tr>
                <td colSpan={6} className="px-4 py-8 text-center text-slate-500">
                  No users found.
                </td>
              </tr>
            ) : (
              data.items.map((user) => (
                <>
                <tr key={user.id} className="border-t border-slate-100 dark:border-neutral-800">
                  <td className="sticky left-0 bg-white px-4 py-2 dark:bg-neutral-900">{user.email}</td>
                  <td className="px-4 py-2">{user.displayName ?? '—'}</td>
                  <td className="px-4 py-2">{user.role || '—'}</td>
                  <td className="px-4 py-2">{user.orgRole ?? '—'}</td>
                  <td className="px-4 py-2">{user.active ? 'Active' : 'Deactivated'}</td>
                  <td className="px-4 py-2">
                    <div className="flex flex-wrap gap-3">
                      {customFieldsEnabled && canManage && fieldDefs.length > 0 ? (
                        <button
                          type="button"
                          onClick={() => void openCustomFields(user)}
                          className="text-sm text-slate-600 hover:underline dark:text-slate-300"
                        >
                          Custom fields
                        </button>
                      ) : null}
                      {showViewAs && user.active && !user.orgRole ? (
                        <button
                          type="button"
                          disabled={busy === user.id}
                          onClick={() => void viewAs(user)}
                          className="text-sm text-indigo-600 hover:underline disabled:opacity-50 dark:text-indigo-400"
                        >
                          {t('impersonation.viewAs', { defaultValue: 'View as' })}
                        </button>
                      ) : null}
                      {user.active ? (
                        <button
                          type="button"
                          disabled={busy === user.id}
                          onClick={() => void deactivate(user)}
                          className="text-sm text-red-600 hover:underline disabled:opacity-50 dark:text-red-400"
                        >
                          Deactivate
                        </button>
                      ) : null}
                    </div>
                  </td>
                </tr>
                {expandedUserId === user.id ? (
                  <tr key={`${user.id}-fields`} className="border-t border-slate-100 bg-slate-50 dark:border-neutral-800 dark:bg-neutral-950">
                    <td colSpan={6} className="px-4 py-4">
                      <fieldset className="space-y-3">
                        <legend className="text-sm font-medium text-slate-800 dark:text-slate-200">Custom fields</legend>
                        {fieldDefs.map((def) => (
                          <label key={def.id} className="block max-w-md text-sm">
                            <span className="mb-1 block">{def.label}</span>
                            {def.fieldType === 'select' ? (
                              <select
                                value={fieldValues[def.key] ?? ''}
                                onChange={(e) => setFieldValues({ ...fieldValues, [def.key]: e.target.value })}
                                className="w-full rounded-lg border border-slate-300 px-3 py-2 dark:border-neutral-700 dark:bg-neutral-900"
                              >
                                <option value="">—</option>
                                {(def.selectOptions ?? []).map((opt) => (
                                  <option key={opt} value={opt}>{opt}</option>
                                ))}
                              </select>
                            ) : def.fieldType === 'boolean' ? (
                              <select
                                value={fieldValues[def.key] ?? ''}
                                onChange={(e) => setFieldValues({ ...fieldValues, [def.key]: e.target.value })}
                                className="w-full rounded-lg border border-slate-300 px-3 py-2 dark:border-neutral-700 dark:bg-neutral-900"
                              >
                                <option value="">—</option>
                                <option value="true">Yes</option>
                                <option value="false">No</option>
                              </select>
                            ) : (
                              <input
                                type={def.fieldType === 'number' ? 'number' : def.fieldType === 'date' ? 'date' : 'text'}
                                value={fieldValues[def.key] ?? ''}
                                onChange={(e) => setFieldValues({ ...fieldValues, [def.key]: e.target.value })}
                                className="w-full rounded-lg border border-slate-300 px-3 py-2 dark:border-neutral-700 dark:bg-neutral-900"
                              />
                            )}
                          </label>
                        ))}
                        <div className="flex gap-2">
                          <button
                            type="button"
                            disabled={fieldBusy}
                            onClick={() => void saveCustomFields(user.id)}
                            className="rounded-lg bg-indigo-600 px-3 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50"
                          >
                            Save custom fields
                          </button>
                          <button type="button" onClick={() => setExpandedUserId(null)} className="rounded-lg px-3 py-2 text-sm">
                            Cancel
                          </button>
                        </div>
                      </fieldset>
                    </td>
                  </tr>
                ) : null}
                </>
              ))
            )}
          </tbody>
        </table>
      </div>

      {data && data.totalPages > 1 ? (
        <nav aria-label="User table pagination" className="mt-4 flex items-center gap-2">
          <button
            type="button"
            disabled={page <= 1}
            onClick={() => setPage((p) => p - 1)}
            className="rounded border px-3 py-1 text-sm disabled:opacity-50"
          >
            Previous
          </button>
          <span className="text-sm text-slate-600 dark:text-slate-400">
            Page {data.page} of {data.totalPages}
          </span>
          <button
            type="button"
            disabled={page >= data.totalPages}
            onClick={() => setPage((p) => p + 1)}
            className="rounded border px-3 py-1 text-sm disabled:opacity-50"
          >
            Next
          </button>
        </nav>
      ) : null}
    </div>
  )
}
