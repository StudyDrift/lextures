import { useCallback, useEffect, useId, useState } from 'react'
import { Link, useParams, useSearchParams } from 'react-router-dom'
import {
  fetchAdminUserDetail,
  type AdminUserDetail as AdminUserDetailType,
} from '../../lib/admin-console-api'

function formatOrgRole(role: string | null | undefined): string {
  if (!role) return '—'
  switch (role) {
    case 'org_admin':
      return 'Org admin'
    case 'org_unit_admin':
      return 'Org unit admin'
    case 'org_viewer':
      return 'Org viewer'
    case 'global_admin':
      return 'Global admin'
    default:
      return role
  }
}

export default function AdminUserDetail() {
  const titleId = useId()
  const { userId = '' } = useParams()
  const [searchParams] = useSearchParams()
  const orgId = searchParams.get('orgId')
  const [data, setData] = useState<AdminUserDetailType | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    if (!userId) return
    setLoading(true)
    setError(null)
    try {
      setData(await fetchAdminUserDetail(userId, orgId))
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load user.')
      setData(null)
    } finally {
      setLoading(false)
    }
  }, [userId, orgId])

  useEffect(() => {
    void load()
  }, [load])

  const backTo = orgId ? `/org-admin/users?orgId=${encodeURIComponent(orgId)}` : '/org-admin/users'
  const name = data?.displayName?.trim() || data?.email || 'User'

  return (
    <div>
      <p className="mb-3">
        <Link to={backTo} className="text-sm text-accent-fg hover:underline">
          ← Users
        </Link>
      </p>
      <h1 id={titleId} className="text-xl font-semibold text-fg-default">
        {loading ? 'User' : name}
      </h1>

      {error ? (
        <p role="alert" className="mt-4 text-sm text-danger-fg">
          {error}
        </p>
      ) : null}

      {loading ? <p className="mt-4 text-sm text-fg-muted">Loading…</p> : null}

      {data ? (
        <div className="mt-6 space-y-8">
          <dl className="grid gap-3 text-sm sm:grid-cols-2">
            <div>
              <dt className="text-fg-muted dark:text-fg-subtle">Email</dt>
              <dd className="mt-0.5 text-fg-default">{data.email}</dd>
            </div>
            <div>
              <dt className="text-fg-muted dark:text-fg-subtle">Name</dt>
              <dd className="mt-0.5 text-fg-default">{data.displayName ?? '—'}</dd>
            </div>
            <div>
              <dt className="text-fg-muted dark:text-fg-subtle">App role</dt>
              <dd className="mt-0.5 text-fg-default">{data.role || '—'}</dd>
            </div>
            <div>
              <dt className="text-fg-muted dark:text-fg-subtle">Org role</dt>
              <dd className="mt-0.5 text-fg-default">{formatOrgRole(data.orgRole)}</dd>
            </div>
            <div>
              <dt className="text-fg-muted dark:text-fg-subtle">Status</dt>
              <dd className="mt-0.5 text-fg-default">{data.active ? 'Active' : 'Deactivated'}</dd>
            </div>
            {data.accountType ? (
              <div>
                <dt className="text-fg-muted dark:text-fg-subtle">Account type</dt>
                <dd className="mt-0.5 capitalize text-fg-default">{data.accountType}</dd>
              </div>
            ) : null}
          </dl>

          <section>
            <h2 className="text-base font-semibold text-fg-default">
              Enrollments ({data.enrollmentCount})
            </h2>
            <div className="mt-3 overflow-x-auto rounded-xl border border-border-default dark:border-border-subtle">
              <table className="min-w-full text-start text-sm">
                <caption className="sr-only">Course enrollments</caption>
                <thead className="bg-surface-base text-fg-muted dark:bg-surface-base dark:text-fg-subtle">
                  <tr>
                    <th scope="col" className="px-4 py-2 font-medium">
                      Course
                    </th>
                    <th scope="col" className="px-4 py-2 font-medium">
                      Code
                    </th>
                    <th scope="col" className="px-4 py-2 font-medium">
                      Role
                    </th>
                    <th scope="col" className="px-4 py-2 font-medium">
                      Status
                    </th>
                  </tr>
                </thead>
                <tbody>
                  {!data.enrollments.length ? (
                    <tr>
                      <td colSpan={4} className="px-4 py-8 text-center text-fg-muted">
                        No enrollments.
                      </td>
                    </tr>
                  ) : (
                    data.enrollments.map((row) => (
                      <tr key={`${row.courseId}-${row.role}`} className="border-t border-border-subtle">
                        <td className="px-4 py-2">
                          <Link
                            to={`/courses/${encodeURIComponent(row.courseCode)}`}
                            className="text-accent-fg hover:underline"
                          >
                            {row.courseTitle}
                          </Link>
                        </td>
                        <td className="px-4 py-2 font-mono text-xs">{row.courseCode}</td>
                        <td className="px-4 py-2 capitalize">{row.role}</td>
                        <td className="px-4 py-2">{row.active ? 'Active' : 'Inactive'}</td>
                      </tr>
                    ))
                  )}
                </tbody>
              </table>
            </div>
          </section>
        </div>
      ) : null}
    </div>
  )
}
