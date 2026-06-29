import { useCallback, useEffect, useId, useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import {
  fetchAdminUsers,
  patchAdminUser,
  type AdminUser,
  type Paginated,
} from '../../lib/admin-console-api'

const ROLES = ['', 'student', 'instructor', 'ta', 'admin']
const PAGE_SIZES = [25, 50, 100]

export default function AdminUsers() {
  const titleId = useId()
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

  return (
    <div>
      <h1 id={titleId} className="text-xl font-semibold text-slate-900 dark:text-slate-100">
        Users
      </h1>

      <div className="mt-4 flex flex-wrap gap-3">
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
                <tr key={user.id} className="border-t border-slate-100 dark:border-neutral-800">
                  <td className="sticky left-0 bg-white px-4 py-2 dark:bg-neutral-900">{user.email}</td>
                  <td className="px-4 py-2">{user.displayName ?? '—'}</td>
                  <td className="px-4 py-2">{user.role || '—'}</td>
                  <td className="px-4 py-2">{user.orgRole ?? '—'}</td>
                  <td className="px-4 py-2">{user.active ? 'Active' : 'Deactivated'}</td>
                  <td className="px-4 py-2">
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
                  </td>
                </tr>
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
