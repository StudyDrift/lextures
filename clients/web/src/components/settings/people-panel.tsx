import { type FormEvent, useCallback, useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useSearchParams } from 'react-router-dom'
import { useConfirm } from '../use-confirm'
import {
  ArrowLeft,
  Loader2,
  MailPlus,
  Search,
  Trash2,
  UserCheck,
  UserMinus,
  UserPlus,
  Users,
  UserX,
} from 'lucide-react'
import { formatDateTime } from '../../lib/format'
import { toastMutationError, toastSaveOk } from '../../lib/lms-toast'
import {
  deletePerson,
  fetchPeopleStats,
  fetchPersonReport,
  invitePerson,
  patchPerson,
  personDisplayName,
  searchPeople,
  type PaginatedPeople,
  type PeopleDashboardStats,
  type PersonReport,
} from '../../lib/people-api'

const PAGE_SIZES = [25, 50, 100]

function formatCount(value: number): string {
  return value.toLocaleString()
}

function PeopleStatsCard({
  label,
  value,
  hint,
  icon: Icon,
}: {
  label: string
  value: number | null
  hint?: string
  icon: typeof Users
}) {
  return (
    <div className="rounded-xl border border-slate-200 bg-white px-4 py-3 dark:border-neutral-800 dark:bg-neutral-900">
      <div className="flex items-start justify-between gap-3">
        <p className="text-xs font-medium uppercase tracking-wide text-slate-500 dark:text-neutral-400">
          {label}
        </p>
        <Icon className="h-4 w-4 shrink-0 text-slate-400 dark:text-neutral-500" aria-hidden />
      </div>
      <p className="mt-1 text-2xl font-semibold tabular-nums text-slate-900 dark:text-neutral-100">
        {value == null ? '—' : formatCount(value)}
      </p>
      {hint ? (
        <p className="mt-1 text-xs text-slate-500 dark:text-neutral-500">{hint}</p>
      ) : null}
    </div>
  )
}

function PeopleDashboardCards({
  stats,
  loading,
  error,
}: {
  stats: PeopleDashboardStats | null
  loading: boolean
  error: string | null
}) {
  if (error) {
    return (
      <p role="alert" className="text-sm text-red-600 dark:text-red-400">
        {error}
      </p>
    )
  }

  const value = (key: keyof PeopleDashboardStats): number | null =>
    loading || !stats ? null : stats[key]

  return (
    <section aria-labelledby="people-dashboard-heading" className="space-y-3">
      <h3 id="people-dashboard-heading" className="sr-only">
        People overview
      </h3>
      <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-5">
        <PeopleStatsCard
          label="New signups"
          hint="Past 7 days"
          value={value('signupsLast7Days')}
          icon={UserPlus}
        />
        <PeopleStatsCard
          label="Active accounts"
          hint="Can sign in"
          value={value('activeAccounts')}
          icon={UserCheck}
        />
        <PeopleStatsCard
          label="Active this month"
          hint="Learning activity in 30 days"
          value={value('recentlyActive30Days')}
          icon={Users}
        />
        <PeopleStatsCard
          label="Total accounts"
          value={value('totalAccounts')}
          icon={Users}
        />
        <PeopleStatsCard
          label="Suspended"
          hint="Blocked from signing in"
          value={value('suspendedAccounts')}
          icon={UserX}
        />
      </div>
    </section>
  )
}

function statusLabel(active: boolean): string {
  return active ? 'Active' : 'Suspended'
}

function PersonReportView({
  report,
  loading,
  error,
  busy,
  onBack,
  onSuspend,
  onReactivate,
  onDelete,
}: {
  report: PersonReport | null
  loading: boolean
  error: string | null
  busy: boolean
  onBack: () => void
  onSuspend: () => void
  onReactivate: () => void
  onDelete: () => void
}) {
  if (loading) {
    return <p className="mt-6 text-sm text-slate-500 dark:text-neutral-400">Loading report…</p>
  }
  if (error) {
    return (
      <p role="alert" className="mt-6 text-sm text-red-600 dark:text-red-400">
        {error}
      </p>
    )
  }
  if (!report) return null

  const name = personDisplayName(report)
  const isErased = report.email.endsWith('@erased.invalid')

  return (
    <div className="mt-6 space-y-6">
      <button
        type="button"
        onClick={onBack}
        className="inline-flex items-center gap-2 text-sm font-medium text-slate-600 hover:text-slate-900 dark:text-neutral-400 dark:hover:text-neutral-100"
      >
        <ArrowLeft className="h-4 w-4" aria-hidden />
        Back to search
      </button>

      <div className="rounded-xl border border-slate-200 bg-white p-5 dark:border-neutral-800 dark:bg-neutral-900">
        <div className="flex flex-wrap items-start justify-between gap-4">
          <div>
            <h3 className="text-lg font-semibold text-slate-900 dark:text-neutral-100">{name}</h3>
            <p className="mt-1 text-sm text-slate-600 dark:text-neutral-400">{report.email}</p>
            <dl className="mt-4 grid gap-2 text-sm sm:grid-cols-2">
              <div>
                <dt className="text-slate-500 dark:text-neutral-500">Organization</dt>
                <dd className="text-slate-900 dark:text-neutral-100">{report.orgName}</dd>
              </div>
              <div>
                <dt className="text-slate-500 dark:text-neutral-500">Role</dt>
                <dd className="text-slate-900 dark:text-neutral-100">{report.role || '—'}</dd>
              </div>
              <div>
                <dt className="text-slate-500 dark:text-neutral-500">Status</dt>
                <dd className="text-slate-900 dark:text-neutral-100">{statusLabel(report.active)}</dd>
              </div>
              <div>
                <dt className="text-slate-500 dark:text-neutral-500">Joined</dt>
                <dd className="text-slate-900 dark:text-neutral-100">{formatDateTime(report.createdAt)}</dd>
              </div>
              <div>
                <dt className="text-slate-500 dark:text-neutral-500">Last activity</dt>
                <dd className="text-slate-900 dark:text-neutral-100">
                  {report.lastActivityAt ? formatDateTime(report.lastActivityAt) : '—'}
                </dd>
              </div>
              <div>
                <dt className="text-slate-500 dark:text-neutral-500">Enrollments</dt>
                <dd className="text-slate-900 dark:text-neutral-100">{report.enrollmentCount}</dd>
              </div>
            </dl>
          </div>
          {!isErased ? (
            <div className="flex flex-wrap gap-2">
              {report.active ? (
                <button
                  type="button"
                  disabled={busy}
                  onClick={onSuspend}
                  className="inline-flex items-center gap-2 rounded-lg border border-amber-300 px-3 py-2 text-sm font-medium text-amber-800 hover:bg-amber-50 disabled:opacity-50 dark:border-amber-800 dark:text-amber-200 dark:hover:bg-amber-950/40"
                >
                  <UserMinus className="h-4 w-4" aria-hidden />
                  Suspend
                </button>
              ) : (
                <button
                  type="button"
                  disabled={busy}
                  onClick={onReactivate}
                  className="inline-flex items-center gap-2 rounded-lg border border-emerald-300 px-3 py-2 text-sm font-medium text-emerald-800 hover:bg-emerald-50 disabled:opacity-50 dark:border-emerald-800 dark:text-emerald-200 dark:hover:bg-emerald-950/40"
                >
                  <UserPlus className="h-4 w-4" aria-hidden />
                  Reactivate
                </button>
              )}
              <button
                type="button"
                disabled={busy}
                onClick={onDelete}
                className="inline-flex items-center gap-2 rounded-lg border border-red-300 px-3 py-2 text-sm font-medium text-red-700 hover:bg-red-50 disabled:opacity-50 dark:border-red-900 dark:text-red-300 dark:hover:bg-red-950/40"
              >
                <Trash2 className="h-4 w-4" aria-hidden />
                Delete account
              </button>
            </div>
          ) : null}
        </div>
      </div>

      <section>
        <h4 className="text-sm font-semibold text-slate-900 dark:text-neutral-100">Enrollments</h4>
        {report.enrollments.length === 0 ? (
          <p className="mt-2 text-sm text-slate-500 dark:text-neutral-400">No enrollments.</p>
        ) : (
          <div className="mt-3 overflow-x-auto rounded-xl border border-slate-200 dark:border-neutral-800">
            <table className="min-w-full text-left text-sm">
              <thead className="bg-slate-50 text-slate-600 dark:bg-neutral-950 dark:text-neutral-400">
                <tr>
                  <th scope="col" className="px-4 py-2 font-medium">Course</th>
                  <th scope="col" className="px-4 py-2 font-medium">Role</th>
                  <th scope="col" className="px-4 py-2 font-medium">State</th>
                  <th scope="col" className="px-4 py-2 font-medium">Enrolled</th>
                </tr>
              </thead>
              <tbody>
                {report.enrollments.map((e) => (
                  <tr key={`${e.courseId}-${e.role}`} className="border-t border-slate-100 dark:border-neutral-800">
                    <td className="px-4 py-2">
                      <span className="font-medium text-slate-900 dark:text-neutral-100">{e.courseTitle}</span>
                      <span className="ml-2 text-xs text-slate-500 dark:text-neutral-500">{e.courseCode}</span>
                    </td>
                    <td className="px-4 py-2">{e.role}</td>
                    <td className="px-4 py-2">{e.state}</td>
                    <td className="px-4 py-2">{formatDateTime(e.enrolledAt)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </section>

      <section>
        <h4 className="text-sm font-semibold text-slate-900 dark:text-neutral-100">Recent activity</h4>
        {report.recentActivity.length === 0 ? (
          <p className="mt-2 text-sm text-slate-500 dark:text-neutral-400">No recorded activity.</p>
        ) : (
          <div className="mt-3 overflow-x-auto rounded-xl border border-slate-200 dark:border-neutral-800">
            <table className="min-w-full text-left text-sm">
              <thead className="bg-slate-50 text-slate-600 dark:bg-neutral-950 dark:text-neutral-400">
                <tr>
                  <th scope="col" className="px-4 py-2 font-medium">When</th>
                  <th scope="col" className="px-4 py-2 font-medium">Event</th>
                  <th scope="col" className="px-4 py-2 font-medium">Course</th>
                </tr>
              </thead>
              <tbody>
                {report.recentActivity.map((a, i) => (
                  <tr key={`${a.occurredAt}-${i}`} className="border-t border-slate-100 dark:border-neutral-800">
                    <td className="px-4 py-2 whitespace-nowrap">{formatDateTime(a.occurredAt)}</td>
                    <td className="px-4 py-2">{a.eventKind.replaceAll('_', ' ')}</td>
                    <td className="px-4 py-2">
                      {a.courseTitle}
                      <span className="ml-2 text-xs text-slate-500">{a.courseCode}</span>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </section>
    </div>
  )
}

export function PeoplePanel() {
  const { t } = useTranslation('common')
  const { confirm, ConfirmDialogHost } = useConfirm()
  const [searchParams, setSearchParams] = useSearchParams()
  const selectedUserId = searchParams.get('userId')

  const [q, setQ] = useState('')
  const [submittedQ, setSubmittedQ] = useState('')
  const [page, setPage] = useState(1)
  const [perPage, setPerPage] = useState(25)
  const [data, setData] = useState<PaginatedPeople | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [busyId, setBusyId] = useState<string | null>(null)

  const [report, setReport] = useState<PersonReport | null>(null)
  const [reportLoading, setReportLoading] = useState(false)
  const [reportError, setReportError] = useState<string | null>(null)

  const [inviteOpen, setInviteOpen] = useState(false)
  const [inviteEmail, setInviteEmail] = useState('')
  const [inviteFirst, setInviteFirst] = useState('')
  const [inviteLast, setInviteLast] = useState('')
  const [inviting, setInviting] = useState(false)

  const [stats, setStats] = useState<PeopleDashboardStats | null>(null)
  const [statsLoading, setStatsLoading] = useState(true)
  const [statsError, setStatsError] = useState<string | null>(null)

  const loadStats = useCallback(async () => {
    setStatsLoading(true)
    setStatsError(null)
    try {
      setStats(await fetchPeopleStats())
    } catch (e) {
      setStatsError(e instanceof Error ? e.message : 'Failed to load people stats.')
      setStats(null)
    } finally {
      setStatsLoading(false)
    }
  }, [])

  useEffect(() => {
    void loadStats()
  }, [loadStats])

  const loadSearch = useCallback(async () => {
    const query = submittedQ.trim()
    if (!query) {
      setData(null)
      return
    }
    setLoading(true)
    setError(null)
    try {
      setData(await searchPeople({ q: query, page, perPage }))
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Search failed.')
      setData(null)
    } finally {
      setLoading(false)
    }
  }, [submittedQ, page, perPage])

  useEffect(() => {
    void loadSearch()
  }, [loadSearch])

  const loadReport = useCallback(async (userId: string) => {
    setReportLoading(true)
    setReportError(null)
    try {
      setReport(await fetchPersonReport(userId))
    } catch (e) {
      setReportError(e instanceof Error ? e.message : 'Failed to load report.')
      setReport(null)
    } finally {
      setReportLoading(false)
    }
  }, [])

  useEffect(() => {
    if (!selectedUserId) {
      setReport(null)
      setReportError(null)
      return
    }
    void loadReport(selectedUserId)
  }, [selectedUserId, loadReport])

  function onSearchSubmit(e: FormEvent) {
    e.preventDefault()
    setSubmittedQ(q.trim())
    setPage(1)
  }

  function openPerson(userId: string) {
    const next = new URLSearchParams(searchParams)
    next.set('userId', userId)
    setSearchParams(next, { replace: false })
  }

  function closePerson() {
    const next = new URLSearchParams(searchParams)
    next.delete('userId')
    setSearchParams(next, { replace: false })
  }

  async function onInvite(e: FormEvent) {
    e.preventDefault()
    const email = inviteEmail.trim()
    if (!email) return
    setInviting(true)
    try {
      await invitePerson({
        email,
        firstName: inviteFirst.trim() || undefined,
        lastName: inviteLast.trim() || undefined,
      })
      toastSaveOk(`Invitation sent to ${email}.`)
      setInviteOpen(false)
      setInviteEmail('')
      setInviteFirst('')
      setInviteLast('')
      setQ(email)
      setSubmittedQ(email)
      setPage(1)
      void loadStats()
    } catch (err) {
      toastMutationError(err instanceof Error ? err.message : 'Could not invite user.')
    } finally {
      setInviting(false)
    }
  }

  async function mutatePerson(userId: string, active: boolean, name: string) {
    const title = active
      ? t('people.reactivate.title')
      : t('people.suspend.title', { name })
    if (!(await confirm({ title, variant: active ? 'default' : 'danger' }))) return
    setBusyId(userId)
    try {
      await patchPerson(userId, { active })
      toastSaveOk(active ? 'Account reactivated.' : 'Account suspended.')
      if (selectedUserId === userId) await loadReport(userId)
      await Promise.all([loadSearch(), loadStats()])
    } catch (e) {
      toastMutationError(e instanceof Error ? e.message : 'Update failed.')
    } finally {
      setBusyId(null)
    }
  }

  async function removePerson(userId: string, name: string) {
    if (!(await confirm({ title: t('people.delete.title', { name }), variant: 'danger' }))) return
    setBusyId(userId)
    try {
      await deletePerson(userId)
      toastSaveOk('Account deleted.')
      closePerson()
      await Promise.all([loadSearch(), loadStats()])
    } catch (e) {
      toastMutationError(e instanceof Error ? e.message : 'Delete failed.')
    } finally {
      setBusyId(null)
    }
  }

  if (selectedUserId) {
    return (
      <>
        <PersonReportView
          report={report}
          loading={reportLoading}
          error={reportError}
          busy={busyId === selectedUserId}
          onBack={closePerson}
          onSuspend={() => {
            const name = report ? personDisplayName(report) : 'this user'
            void mutatePerson(selectedUserId, false, name)
          }}
          onReactivate={() => {
            void mutatePerson(selectedUserId, true, '')
          }}
          onDelete={() => {
            const name = report ? personDisplayName(report) : 'this user'
            void removePerson(selectedUserId, name)
          }}
        />
        {ConfirmDialogHost}
      </>
    )
  }

  return (
    <div className="mt-6 space-y-6">
      <PeopleDashboardCards stats={stats} loading={statsLoading} error={statsError} />

      <div className="flex flex-wrap items-start justify-between gap-3">
        <p className="text-sm text-slate-600 dark:text-neutral-400">
          Search users by first name, last name, or email. Results are not shown until you search.
        </p>
        <button
          type="button"
          onClick={() => setInviteOpen(true)}
          className="inline-flex items-center gap-2 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 dark:bg-neutral-100 dark:text-neutral-950 dark:hover:bg-white"
        >
          <MailPlus className="h-4 w-4" aria-hidden />
          Invite user
        </button>
      </div>

      <form onSubmit={onSearchSubmit} className="flex flex-wrap items-end gap-3">
        <label className="flex min-w-[16rem] flex-1 flex-col text-sm">
          <span className="mb-1 text-slate-600 dark:text-neutral-400">Search</span>
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-400" aria-hidden />
            <input
              type="search"
              value={q}
              onChange={(e) => setQ(e.target.value)}
              placeholder="First name, last name, or email"
              className="w-full rounded-lg border border-slate-300 py-2 pl-9 pr-3 dark:border-neutral-700 dark:bg-neutral-900"
            />
          </div>
        </label>
        <label className="flex flex-col text-sm">
          <span className="mb-1 text-slate-600 dark:text-neutral-400">Page size</span>
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
        <button
          type="submit"
          disabled={!q.trim() || loading}
          className="rounded-lg border border-slate-300 px-4 py-2 text-sm font-medium text-slate-700 hover:bg-slate-50 disabled:opacity-50 dark:border-neutral-700 dark:text-neutral-200 dark:hover:bg-neutral-800"
        >
          {loading ? 'Searching…' : 'Search'}
        </button>
      </form>

      {error ? (
        <p role="alert" className="text-sm text-red-600 dark:text-red-400">
          {error}
        </p>
      ) : null}

      {!submittedQ.trim() ? (
        <p className="rounded-xl border border-dashed border-slate-200 px-4 py-8 text-center text-sm text-slate-500 dark:border-neutral-700 dark:text-neutral-400">
          Enter a name or email above to find users.
        </p>
      ) : (
        <div className="overflow-x-auto rounded-xl border border-slate-200 dark:border-neutral-800">
          <table className="min-w-full text-left text-sm">
            <thead className="bg-slate-50 text-slate-600 dark:bg-neutral-950 dark:text-neutral-400">
              <tr>
                <th scope="col" className="px-4 py-2 font-medium">Name</th>
                <th scope="col" className="px-4 py-2 font-medium">Email</th>
                <th scope="col" className="px-4 py-2 font-medium">Organization</th>
                <th scope="col" className="px-4 py-2 font-medium">Role</th>
                <th scope="col" className="px-4 py-2 font-medium">Status</th>
                <th scope="col" className="px-4 py-2 font-medium">Actions</th>
              </tr>
            </thead>
            <tbody>
              {loading ? (
                <tr>
                  <td colSpan={6} className="px-4 py-8 text-center text-slate-500">
                    <Loader2 className="mx-auto h-5 w-5 animate-spin" aria-hidden />
                  </td>
                </tr>
              ) : !data?.items.length ? (
                <tr>
                  <td colSpan={6} className="px-4 py-8 text-center text-slate-500">
                    No users matched your search.
                  </td>
                </tr>
              ) : (
                data.items.map((user) => (
                  <tr key={user.id} className="border-t border-slate-100 dark:border-neutral-800">
                    <td className="px-4 py-2">
                      <button
                        type="button"
                        onClick={() => openPerson(user.id)}
                        className="font-medium text-indigo-600 hover:underline dark:text-indigo-400"
                      >
                        {personDisplayName(user)}
                      </button>
                    </td>
                    <td className="px-4 py-2">{user.email}</td>
                    <td className="px-4 py-2">{user.orgName}</td>
                    <td className="px-4 py-2">{user.role || '—'}</td>
                    <td className="px-4 py-2">{statusLabel(user.active)}</td>
                    <td className="px-4 py-2">
                      <div className="flex flex-wrap gap-3">
                        {user.active ? (
                          <button
                            type="button"
                            disabled={busyId === user.id}
                            onClick={() => void mutatePerson(user.id, false, personDisplayName(user))}
                            className="text-sm text-amber-700 hover:underline disabled:opacity-50 dark:text-amber-400"
                          >
                            Suspend
                          </button>
                        ) : (
                          <button
                            type="button"
                            disabled={busyId === user.id}
                            onClick={() => void mutatePerson(user.id, true, '')}
                            className="text-sm text-emerald-700 hover:underline disabled:opacity-50 dark:text-emerald-400"
                          >
                            Reactivate
                          </button>
                        )}
                        <button
                          type="button"
                          disabled={busyId === user.id}
                          onClick={() => void removePerson(user.id, personDisplayName(user))}
                          className="text-sm text-red-600 hover:underline disabled:opacity-50 dark:text-red-400"
                        >
                          Delete
                        </button>
                      </div>
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      )}

      {data && data.totalPages > 1 ? (
        <nav aria-label="People search pagination" className="flex items-center gap-2">
          <button
            type="button"
            disabled={page <= 1}
            onClick={() => setPage((p) => p - 1)}
            className="rounded border px-3 py-1 text-sm disabled:opacity-50"
          >
            Previous
          </button>
          <span className="text-sm text-slate-600 dark:text-neutral-400">
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

      {inviteOpen ? (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4"
          role="dialog"
          aria-modal="true"
          aria-labelledby="invite-user-title"
        >
          <form
            onSubmit={(e) => void onInvite(e)}
            className="w-full max-w-md rounded-xl border border-slate-200 bg-white p-6 shadow-xl dark:border-neutral-700 dark:bg-neutral-900"
          >
            <h3 id="invite-user-title" className="text-base font-semibold text-slate-900 dark:text-neutral-100">
              Invite user
            </h3>
            <p className="mt-1 text-sm text-slate-500 dark:text-neutral-400">
              Creates an account and emails a link to set their password.
            </p>
            <div className="mt-4 space-y-3">
              <label className="block text-sm">
                <span className="text-slate-600 dark:text-neutral-400">Email</span>
                <input
                  type="email"
                  required
                  value={inviteEmail}
                  onChange={(e) => setInviteEmail(e.target.value)}
                  className="mt-1 w-full rounded-lg border border-slate-300 px-3 py-2 dark:border-neutral-700 dark:bg-neutral-950"
                />
              </label>
              <label className="block text-sm">
                <span className="text-slate-600 dark:text-neutral-400">First name</span>
                <input
                  value={inviteFirst}
                  onChange={(e) => setInviteFirst(e.target.value)}
                  className="mt-1 w-full rounded-lg border border-slate-300 px-3 py-2 dark:border-neutral-700 dark:bg-neutral-950"
                />
              </label>
              <label className="block text-sm">
                <span className="text-slate-600 dark:text-neutral-400">Last name</span>
                <input
                  value={inviteLast}
                  onChange={(e) => setInviteLast(e.target.value)}
                  className="mt-1 w-full rounded-lg border border-slate-300 px-3 py-2 dark:border-neutral-700 dark:bg-neutral-950"
                />
              </label>
            </div>
            <div className="mt-6 flex justify-end gap-2">
              <button
                type="button"
                onClick={() => setInviteOpen(false)}
                className="rounded-lg border border-slate-300 px-4 py-2 text-sm dark:border-neutral-700"
              >
                Cancel
              </button>
              <button
                type="submit"
                disabled={inviting}
                className="inline-flex items-center gap-2 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50"
              >
                {inviting ? <Loader2 className="h-4 w-4 animate-spin" aria-hidden /> : null}
                Send invite
              </button>
            </div>
          </form>
        </div>
      ) : null}
      {ConfirmDialogHost}
    </div>
  )
}