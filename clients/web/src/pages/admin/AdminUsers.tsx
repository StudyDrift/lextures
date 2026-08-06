import { lazy, Suspense, useCallback, useEffect, useId, useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { usePlatformFeatures } from '../../context/platform-features-context'
import {
  fetchAdminConsoleCapabilities,
  fetchAdminUsers,
  patchAdminUser,
  startImpersonation,
  type AdminUser,
  type Paginated,
} from '../../lib/admin-console-api'
import { fetchCustomFields, usersExportUrl } from '../../lib/custom-fields-api'
import { startImpersonationSession } from '../../lib/impersonation'
import { toastMutationError } from '../../lib/lms-toast'
import { useConfirm } from '../../components/use-confirm'

const AdminUserCustomFieldsPanel = lazy(() => import('./AdminUserCustomFieldsPanel'))

const ROLES = ['', 'student', 'instructor', 'ta', 'admin']
const PAGE_SIZES = [25, 50, 100]

export default function AdminUsers() {
  const { t } = useTranslation('common')
  const { confirm, ConfirmDialogHost } = useConfirm()
  const titleId = useId()
  const { impersonationEnabled } = usePlatformFeatures()
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
  const [customFieldsEnabled, setCustomFieldsEnabled] = useState(false)

  useEffect(() => {
    let cancelled = false
    void fetchAdminConsoleCapabilities()
      .then((caps) => {
        if (!cancelled) {
          setCanManage(caps.canManage)
          setCustomFieldsEnabled(caps.customFieldsEnabled)
        }
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
    if (
      !(await confirm({
        title: t('admin.deactivateUser.title', { email: user.email }),
        variant: 'danger',
      }))
    ) {
      return
    }
    setBusy(user.id)
    try {
      await patchAdminUser(user.id, { active: false })
      await load()
    } catch (e) {
      toastMutationError(e instanceof Error ? e.message : 'Failed to deactivate user.')
    } finally {
      setBusy(null)
    }
  }

  async function viewAs(user: AdminUser) {
    const name = user.displayName?.trim() || user.email
    if (
      !(await confirm({
        title: t('impersonation.confirm', {
          name,
          defaultValue:
            'You are about to view the application as {{name}}. All writes will be blocked. Continue?',
        }),
      }))
    ) {
      return
    }
    setBusy(user.id)
    try {
      const result = await startImpersonation(user.id)
      startImpersonationSession(result.impersonation_token)
      window.location.assign('/')
    } catch (e) {
      toastMutationError(e instanceof Error ? e.message : 'Failed to start impersonation.')
      setBusy(null)
    }
  }

  const showViewAs = impersonationEnabled && canManage

  const [fieldDefsCount, setFieldDefsCount] = useState(0)
  const [expandedUserId, setExpandedUserId] = useState<string | null>(null)

  useEffect(() => {
    if (!customFieldsEnabled) return
    void fetchCustomFields('user', orgId)
      .then((defs) => setFieldDefsCount(defs.length))
      .catch(() => setFieldDefsCount(0))
  }, [customFieldsEnabled, orgId])

  function openCustomFields(user: AdminUser) {
    setExpandedUserId((current) => (current === user.id ? null : user.id))
  }

  return (
    <div>
      <h1 id={titleId} className="text-xl font-semibold text-fg-default dark:text-slate-100">
        Users
      </h1>

      <div className="mt-4 flex flex-wrap items-end justify-between gap-3">
        <div className="flex flex-wrap gap-3">
        <label className="flex flex-col text-sm">
          <span className="mb-1 text-fg-muted dark:text-fg-subtle">Search</span>
          <input
            type="search"
            value={q}
            onChange={(e) => {
              setQ(e.target.value)
              setPage(1)
            }}
            placeholder="Email or name"
            className="rounded-lg border border-border-strong px-3 py-2 dark:border-border-default dark:bg-surface-raised"
          />
        </label>
        <label className="flex flex-col text-sm">
          <span className="mb-1 text-fg-muted dark:text-fg-subtle">Role</span>
          <select
            value={role}
            onChange={(e) => {
              setRole(e.target.value)
              setPage(1)
            }}
            className="rounded-lg border border-border-strong px-3 py-2 dark:border-border-default dark:bg-surface-raised"
          >
            {ROLES.map((r) => (
              <option key={r || 'all'} value={r}>
                {r || 'All roles'}
              </option>
            ))}
          </select>
        </label>
        <label className="flex flex-col text-sm">
          <span className="mb-1 text-fg-muted dark:text-fg-subtle">Page size</span>
          <select
            value={perPage}
            onChange={(e) => {
              setPerPage(Number(e.target.value))
              setPage(1)
            }}
            className="rounded-lg border border-border-strong px-3 py-2 dark:border-border-default dark:bg-surface-raised"
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
            className="rounded-lg border border-border-strong px-3 py-2 text-sm font-medium text-fg-muted hover:bg-surface-base dark:border-border-default dark:text-slate-200"
          >
            Export CSV
          </a>
        )}
      </div>

      {error ? (
        <p role="alert" className="mt-4 text-sm text-danger-fg">
          {error}
        </p>
      ) : null}

      <div className="mt-4 overflow-x-auto rounded-xl border border-border-default dark:border-border-subtle">
        <table className="min-w-full text-left text-sm">
          <caption className="sr-only">Organization users</caption>
          <thead className="bg-surface-base text-fg-muted dark:bg-surface-base dark:text-fg-subtle">
            <tr>
              <th scope="col" className="sticky left-0 bg-surface-base px-4 py-2 font-medium dark:bg-surface-base">
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
                <td colSpan={6} className="px-4 py-8 text-center text-fg-muted">
                  Loading…
                </td>
              </tr>
            ) : !data?.items.length ? (
              <tr>
                <td colSpan={6} className="px-4 py-8 text-center text-fg-muted">
                  No users found.
                </td>
              </tr>
            ) : (
              data.items.map((user) => (
                <>
                <tr key={user.id} className="border-t border-border-subtle">
                  <td className="sticky left-0 bg-surface-raised px-4 py-2 dark:bg-surface-raised">{user.email}</td>
                  <td className="px-4 py-2">{user.displayName ?? '—'}</td>
                  <td className="px-4 py-2">{user.role || '—'}</td>
                  <td className="px-4 py-2">{user.orgRole ?? '—'}</td>
                  <td className="px-4 py-2">{user.active ? 'Active' : 'Deactivated'}</td>
                  <td className="px-4 py-2">
                    <div className="flex flex-wrap gap-3">
                      {customFieldsEnabled && canManage && fieldDefsCount > 0 ? (
                        <button
                          type="button"
                          onClick={() => openCustomFields(user)}
                          className="text-sm text-fg-muted hover:underline dark:text-slate-300"
                        >
                          Custom fields
                        </button>
                      ) : null}
                      {showViewAs && user.active && !user.orgRole ? (
                        <button
                          type="button"
                          disabled={busy === user.id}
                          onClick={() => void viewAs(user)}
                          className="text-sm text-accent-fg hover:underline disabled:opacity-50 dark:text-indigo-400"
                        >
                          {t('impersonation.viewAs', { defaultValue: 'View as' })}
                        </button>
                      ) : null}
                      {user.active ? (
                        <button
                          type="button"
                          disabled={busy === user.id}
                          onClick={() => void deactivate(user)}
                          className="text-sm text-danger-fg hover:underline disabled:opacity-50 dark:text-red-400"
                        >
                          Deactivate
                        </button>
                      ) : null}
                    </div>
                  </td>
                </tr>
                {expandedUserId === user.id ? (
                  <tr key={`${user.id}-fields`} className="border-t border-border-subtle bg-surface-base dark:border-border-subtle dark:bg-surface-base">
                    <td colSpan={6} className="px-4 py-4">
                      <Suspense fallback={<p className="text-sm text-fg-muted">Loading custom fields…</p>}>
                        <AdminUserCustomFieldsPanel
                          userId={user.id}
                          orgId={orgId}
                          onClose={() => setExpandedUserId(null)}
                        />
                      </Suspense>
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
          <span className="text-sm text-fg-muted dark:text-fg-subtle">
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
      {ConfirmDialogHost}
    </div>
  )
}
