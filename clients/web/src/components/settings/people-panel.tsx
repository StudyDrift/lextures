import { type FormEvent, useCallback, useEffect, useId, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useSearchParams } from 'react-router-dom'
import { useConfirm } from '../use-confirm'
import {
  Activity,
  ArrowLeft,
  ChevronDown,
  Loader2,
  MailPlus,
  Search,
  Sparkles,
  Trash2,
  UserCheck,
  UserMinus,
  UserPlus,
  Users,
  UserX,
  X,
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
  type PeopleListFilter,
  type PersonReport,
  type PersonRow,
} from '../../lib/people-api'
import { EnrollmentAvatar } from '../enrollment/enrollment-avatar'

const PAGE_SIZES = [25, 50, 100]

function formatCount(value: number): string {
  return value.toLocaleString()
}

type StatsTone = 'indigo' | 'emerald' | 'sky' | 'slate' | 'rose'

const TONE_STYLES: Record<
  StatsTone,
  {
    card: string
    cardSelected: string
    iconWrap: string
    icon: string
    value: string
  }
> = {
  indigo: {
    card: 'border-indigo-100/80 bg-gradient-to-br from-indigo-50/90 via-white to-white dark:border-indigo-900/40 dark:from-indigo-950/40 dark:via-neutral-900 dark:to-neutral-900',
    cardSelected:
      'border-indigo-400 ring-2 ring-indigo-500/30 dark:border-indigo-500 dark:ring-indigo-400/30',
    iconWrap: 'bg-indigo-100 text-indigo-600 dark:bg-indigo-950/80 dark:text-indigo-300',
    icon: 'text-indigo-600 dark:text-indigo-300',
    value: 'text-slate-900 dark:text-neutral-50',
  },
  emerald: {
    card: 'border-emerald-100/80 bg-gradient-to-br from-emerald-50/90 via-white to-white dark:border-emerald-900/40 dark:from-emerald-950/40 dark:via-neutral-900 dark:to-neutral-900',
    cardSelected:
      'border-emerald-400 ring-2 ring-emerald-500/30 dark:border-emerald-500 dark:ring-emerald-400/30',
    iconWrap: 'bg-emerald-100 text-emerald-600 dark:bg-emerald-950/80 dark:text-emerald-300',
    icon: 'text-emerald-600 dark:text-emerald-300',
    value: 'text-slate-900 dark:text-neutral-50',
  },
  sky: {
    card: 'border-sky-100/80 bg-gradient-to-br from-sky-50/90 via-white to-white dark:border-sky-900/40 dark:from-sky-950/40 dark:via-neutral-900 dark:to-neutral-900',
    cardSelected:
      'border-sky-400 ring-2 ring-sky-500/30 dark:border-sky-500 dark:ring-sky-400/30',
    iconWrap: 'bg-sky-100 text-sky-600 dark:bg-sky-950/80 dark:text-sky-300',
    icon: 'text-sky-600 dark:text-sky-300',
    value: 'text-slate-900 dark:text-neutral-50',
  },
  slate: {
    card: 'border-slate-200/90 bg-gradient-to-br from-slate-50/90 via-white to-white dark:border-neutral-700 dark:from-neutral-800/60 dark:via-neutral-900 dark:to-neutral-900',
    cardSelected:
      'border-slate-400 ring-2 ring-slate-400/40 dark:border-neutral-500 dark:ring-neutral-400/30',
    iconWrap: 'bg-slate-100 text-slate-600 dark:bg-neutral-800 dark:text-neutral-300',
    icon: 'text-slate-600 dark:text-neutral-300',
    value: 'text-slate-900 dark:text-neutral-50',
  },
  rose: {
    card: 'border-rose-100/80 bg-gradient-to-br from-rose-50/90 via-white to-white dark:border-rose-900/40 dark:from-rose-950/40 dark:via-neutral-900 dark:to-neutral-900',
    cardSelected:
      'border-rose-400 ring-2 ring-rose-500/30 dark:border-rose-500 dark:ring-rose-400/30',
    iconWrap: 'bg-rose-100 text-rose-600 dark:bg-rose-950/80 dark:text-rose-300',
    icon: 'text-rose-600 dark:text-rose-300',
    value: 'text-slate-900 dark:text-neutral-50',
  },
}

const STAT_CARDS: {
  filter: PeopleListFilter
  statsKey: keyof PeopleDashboardStats
  label: string
  hint?: string
  icon: typeof Users
  tone: StatsTone
  tableTitle: string
  tableDescription: string
}[] = [
  {
    filter: 'signups_7d',
    statsKey: 'signupsLast7Days',
    label: 'New signups',
    hint: 'Past 7 days',
    icon: UserPlus,
    tone: 'indigo',
    tableTitle: 'New signups',
    tableDescription: 'Accounts created in the past 7 days.',
  },
  {
    filter: 'active',
    statsKey: 'activeAccounts',
    label: 'Active accounts',
    hint: 'Can sign in',
    icon: UserCheck,
    tone: 'emerald',
    tableTitle: 'Active accounts',
    tableDescription: 'Users who can currently sign in.',
  },
  {
    filter: 'recent_30d',
    statsKey: 'recentlyActive30Days',
    label: 'Active this month',
    hint: 'Learning activity in 30 days',
    icon: Activity,
    tone: 'sky',
    tableTitle: 'Active this month',
    tableDescription: 'Users with learning activity in the past 30 days.',
  },
  {
    filter: 'total',
    statsKey: 'totalAccounts',
    label: 'Total accounts',
    icon: Users,
    tone: 'slate',
    tableTitle: 'All accounts',
    tableDescription: 'Every non-system account on the platform.',
  },
  {
    filter: 'suspended',
    statsKey: 'suspendedAccounts',
    label: 'Suspended',
    hint: 'Blocked from signing in',
    icon: UserX,
    tone: 'rose',
    tableTitle: 'Suspended accounts',
    tableDescription: 'Users blocked from signing in.',
  },
]

function PeopleStatsCard({
  label,
  value,
  hint,
  icon: Icon,
  tone,
  loading,
  selected,
  onSelect,
}: {
  label: string
  value: number | null
  hint?: string
  icon: typeof Users
  tone: StatsTone
  loading?: boolean
  selected?: boolean
  onSelect: () => void
}) {
  const styles = TONE_STYLES[tone]
  const countLabel = value == null ? 'unknown count' : `${formatCount(value)} ${label}`
  return (
    <button
      type="button"
      onClick={onSelect}
      aria-pressed={selected}
      aria-label={`${selected ? 'Hide' : 'Show'} ${label}: ${countLabel}`}
      className={`relative flex h-full w-full flex-col overflow-hidden rounded-2xl border px-4 py-4 text-left shadow-sm transition-all hover:shadow-md focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-indigo-500/40 ${styles.card} ${
        selected ? styles.cardSelected : ''
      }`}
    >
      <div className="flex items-start justify-between gap-3">
        {/* Fixed 2-line label band so values line up across cards */}
        <p className="min-h-[2.5rem] min-w-0 flex-1 text-[11px] font-semibold uppercase leading-tight tracking-wider text-slate-500 dark:text-neutral-400">
          {label}
        </p>
        <span
          className={`flex h-10 w-10 shrink-0 items-center justify-center rounded-xl ${styles.iconWrap}`}
          aria-hidden
        >
          <Icon className={`h-5 w-5 ${styles.icon}`} />
        </span>
      </div>
      {loading ? (
        <div
          className="mt-1.5 h-9 w-16 animate-pulse rounded-md bg-slate-200/80 dark:bg-neutral-700"
          aria-hidden
        />
      ) : (
        <p
          className={`mt-1.5 h-9 text-3xl font-semibold leading-none tracking-tight tabular-nums underline-offset-4 group-hover:underline ${styles.value}`}
        >
          {value == null ? '—' : formatCount(value)}
        </p>
      )}
      {/* Fixed 2-line hint band (empty when no hint) keeps footers aligned */}
      <p className="mt-1.5 min-h-[2rem] text-xs leading-snug text-slate-500 dark:text-neutral-500">
        {hint ?? '\u00a0'}
      </p>
      <p className="mt-auto pt-2 inline-flex items-center gap-1 text-[11px] font-medium text-slate-500 dark:text-neutral-400">
        {selected ? 'Hide list' : 'View list'}
        <ChevronDown
          className={`h-3.5 w-3.5 transition-transform ${selected ? 'rotate-180' : ''}`}
          aria-hidden
        />
      </p>
    </button>
  )
}

function PeopleDashboardCards({
  stats,
  loading,
  error,
  selectedFilter,
  onSelectFilter,
}: {
  stats: PeopleDashboardStats | null
  loading: boolean
  error: string | null
  selectedFilter: PeopleListFilter | null
  onSelectFilter: (filter: PeopleListFilter) => void
}) {
  if (error) {
    return (
      <p
        role="alert"
        className="rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700 dark:border-red-900/50 dark:bg-red-950/30 dark:text-red-300"
      >
        {error}
      </p>
    )
  }

  const value = (key: keyof PeopleDashboardStats): number | null =>
    loading || !stats ? null : stats[key]

  return (
    <section aria-labelledby="people-dashboard-heading" className="space-y-3">
      <div className="flex items-end justify-between gap-3">
        <h3 id="people-dashboard-heading" className="sr-only">
          People overview
        </h3>
        <p className="text-xs text-slate-500 dark:text-neutral-500">
          Click a metric to inspect matching accounts.
        </p>
      </div>
      <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-5">
        {STAT_CARDS.map((card) => (
          <PeopleStatsCard
            key={card.filter}
            label={card.label}
            hint={card.hint}
            value={value(card.statsKey)}
            icon={card.icon}
            tone={card.tone}
            loading={loading}
            selected={selectedFilter === card.filter}
            onSelect={() => onSelectFilter(card.filter)}
          />
        ))}
      </div>
    </section>
  )
}

function StatusBadge({ active }: { active: boolean }) {
  if (active) {
    return (
      <span className="inline-flex items-center gap-1.5 rounded-full bg-emerald-50 px-2.5 py-0.5 text-xs font-medium text-emerald-700 ring-1 ring-inset ring-emerald-600/15 dark:bg-emerald-950/40 dark:text-emerald-300 dark:ring-emerald-500/30">
        <span className="h-1.5 w-1.5 rounded-full bg-emerald-500" aria-hidden />
        Active
      </span>
    )
  }
  return (
    <span className="inline-flex items-center gap-1.5 rounded-full bg-rose-50 px-2.5 py-0.5 text-xs font-medium text-rose-700 ring-1 ring-inset ring-rose-600/15 dark:bg-rose-950/40 dark:text-rose-300 dark:ring-rose-500/30">
      <span className="h-1.5 w-1.5 rounded-full bg-rose-500" aria-hidden />
      Suspended
    </span>
  )
}

function statusLabel(active: boolean): string {
  return active ? 'Active' : 'Suspended'
}

function PeopleResultsTable({
  data,
  loading,
  busyId,
  emptyTitle,
  emptyHint,
  loadingLabel = 'Loading…',
  onOpen,
  onSuspend,
  onReactivate,
  onDelete,
  showJoined = false,
}: {
  data: PaginatedPeople | null
  loading: boolean
  busyId: string | null
  emptyTitle: string
  emptyHint: string
  loadingLabel?: string
  onOpen: (userId: string) => void
  onSuspend: (user: PersonRow) => void
  onReactivate: (user: PersonRow) => void
  onDelete: (user: PersonRow) => void
  showJoined?: boolean
}) {
  const colSpan = showJoined ? 7 : 6
  return (
    <div className="overflow-x-auto">
      <table className="min-w-full text-left text-sm">
        <thead className="bg-slate-50/80 text-slate-500 dark:bg-neutral-950/60 dark:text-neutral-400">
          <tr>
            <th scope="col" className="px-5 py-2.5 text-xs font-semibold uppercase tracking-wide">
              Name
            </th>
            <th scope="col" className="px-5 py-2.5 text-xs font-semibold uppercase tracking-wide">
              Email
            </th>
            <th scope="col" className="px-5 py-2.5 text-xs font-semibold uppercase tracking-wide">
              Organization
            </th>
            <th scope="col" className="px-5 py-2.5 text-xs font-semibold uppercase tracking-wide">
              Role
            </th>
            <th scope="col" className="px-5 py-2.5 text-xs font-semibold uppercase tracking-wide">
              Status
            </th>
            {showJoined ? (
              <th scope="col" className="px-5 py-2.5 text-xs font-semibold uppercase tracking-wide">
                Joined
              </th>
            ) : null}
            <th scope="col" className="px-5 py-2.5 text-xs font-semibold uppercase tracking-wide">
              Actions
            </th>
          </tr>
        </thead>
        <tbody className="divide-y divide-slate-100 dark:divide-neutral-800">
          {loading ? (
            <tr>
              <td colSpan={colSpan} className="px-5 py-12 text-center">
                <Loader2 className="mx-auto h-5 w-5 animate-spin text-indigo-500" aria-hidden />
                <p className="mt-2 text-sm text-slate-500">{loadingLabel}</p>
              </td>
            </tr>
          ) : !data?.items.length ? (
            <tr>
              <td colSpan={colSpan} className="px-5 py-12 text-center">
                <Users className="mx-auto h-8 w-8 text-slate-300 dark:text-neutral-600" aria-hidden />
                <p className="mt-2 text-sm font-medium text-slate-700 dark:text-neutral-300">
                  {emptyTitle}
                </p>
                <p className="mt-1 text-sm text-slate-500 dark:text-neutral-400">{emptyHint}</p>
              </td>
            </tr>
          ) : (
            data.items.map((user) => {
              const displayName = personDisplayName(user)
              return (
                <tr
                  key={user.id}
                  className="transition-colors hover:bg-slate-50/80 dark:hover:bg-neutral-800/40"
                >
                  <td className="px-5 py-3">
                    <button
                      type="button"
                      onClick={() => onOpen(user.id)}
                      className="group flex items-center gap-3 text-left"
                    >
                      <EnrollmentAvatar
                        userId={user.id}
                        name={displayName}
                        size="sm"
                        showPreview={false}
                      />
                      <span className="font-medium text-slate-900 group-hover:text-indigo-600 dark:text-neutral-100 dark:group-hover:text-indigo-400">
                        {displayName}
                      </span>
                    </button>
                  </td>
                  <td className="px-5 py-3 text-slate-600 dark:text-neutral-400">{user.email}</td>
                  <td className="px-5 py-3 text-slate-700 dark:text-neutral-300">{user.orgName}</td>
                  <td className="px-5 py-3">
                    {user.role ? (
                      <span className="inline-flex rounded-md bg-slate-100 px-2 py-0.5 text-xs font-medium text-slate-700 dark:bg-neutral-800 dark:text-neutral-300">
                        {user.role}
                      </span>
                    ) : (
                      <span className="text-slate-400">—</span>
                    )}
                  </td>
                  <td className="px-5 py-3">
                    <StatusBadge active={user.active} />
                  </td>
                  {showJoined ? (
                    <td className="px-5 py-3 whitespace-nowrap text-slate-600 dark:text-neutral-400">
                      {formatDateTime(user.createdAt)}
                    </td>
                  ) : null}
                  <td className="px-5 py-3">
                    <div className="flex flex-wrap gap-1.5">
                      {user.active ? (
                        <button
                          type="button"
                          disabled={busyId === user.id}
                          onClick={() => onSuspend(user)}
                          className="rounded-lg px-2 py-1 text-xs font-medium text-amber-700 transition-colors hover:bg-amber-50 disabled:opacity-50 dark:text-amber-400 dark:hover:bg-amber-950/40"
                        >
                          Suspend
                        </button>
                      ) : (
                        <button
                          type="button"
                          disabled={busyId === user.id}
                          onClick={() => onReactivate(user)}
                          className="rounded-lg px-2 py-1 text-xs font-medium text-emerald-700 transition-colors hover:bg-emerald-50 disabled:opacity-50 dark:text-emerald-400 dark:hover:bg-emerald-950/40"
                        >
                          Reactivate
                        </button>
                      )}
                      <button
                        type="button"
                        disabled={busyId === user.id}
                        onClick={() => onDelete(user)}
                        className="rounded-lg px-2 py-1 text-xs font-medium text-red-600 transition-colors hover:bg-red-50 disabled:opacity-50 dark:text-red-400 dark:hover:bg-red-950/40"
                      >
                        Delete
                      </button>
                    </div>
                  </td>
                </tr>
              )
            })
          )}
        </tbody>
      </table>
    </div>
  )
}

function PaginationNav({
  label,
  page,
  totalPages,
  onPrev,
  onNext,
}: {
  label: string
  page: number
  totalPages: number
  onPrev: () => void
  onNext: () => void
}) {
  if (totalPages <= 1) return null
  return (
    <nav
      aria-label={label}
      className="flex items-center justify-between gap-3 border-t border-slate-100 px-5 py-3 dark:border-neutral-800"
    >
      <button
        type="button"
        disabled={page <= 1}
        onClick={onPrev}
        className="rounded-lg border border-slate-200 px-3 py-1.5 text-sm font-medium text-slate-700 transition-colors hover:bg-slate-50 disabled:opacity-50 dark:border-neutral-700 dark:text-neutral-200 dark:hover:bg-neutral-800"
      >
        Previous
      </button>
      <span className="text-sm text-slate-600 dark:text-neutral-400">
        Page {page} of {totalPages}
      </span>
      <button
        type="button"
        disabled={page >= totalPages}
        onClick={onNext}
        className="rounded-lg border border-slate-200 px-3 py-1.5 text-sm font-medium text-slate-700 transition-colors hover:bg-slate-50 disabled:opacity-50 dark:border-neutral-700 dark:text-neutral-200 dark:hover:bg-neutral-800"
      >
        Next
      </button>
    </nav>
  )
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
    return (
      <div className="mt-6 flex items-center gap-3 rounded-2xl border border-slate-200 bg-white px-5 py-8 dark:border-neutral-800 dark:bg-neutral-900">
        <Loader2 className="h-5 w-5 animate-spin text-indigo-500" aria-hidden />
        <p className="text-sm text-slate-500 dark:text-neutral-400">Loading report…</p>
      </div>
    )
  }
  if (error) {
    return (
      <p
        role="alert"
        className="mt-6 rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700 dark:border-red-900/50 dark:bg-red-950/30 dark:text-red-300"
      >
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
        className="inline-flex items-center gap-2 rounded-lg px-1 py-0.5 text-sm font-medium text-slate-600 transition-colors hover:text-slate-900 dark:text-neutral-400 dark:hover:text-neutral-100"
      >
        <ArrowLeft className="h-4 w-4" aria-hidden />
        Back to search
      </button>

      <div className="overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-sm dark:border-neutral-800 dark:bg-neutral-900">
        <div className="border-b border-slate-100 bg-gradient-to-r from-slate-50/80 to-white px-5 py-5 dark:border-neutral-800 dark:from-neutral-950/50 dark:to-neutral-900">
          <div className="flex flex-wrap items-start justify-between gap-4">
            <div className="flex min-w-0 items-start gap-4">
              <EnrollmentAvatar userId={report.id} name={name} size="md" showPreview={false} />
              <div className="min-w-0">
                <div className="flex flex-wrap items-center gap-2">
                  <h3 className="truncate text-lg font-semibold text-slate-900 dark:text-neutral-100">
                    {name}
                  </h3>
                  <StatusBadge active={report.active} />
                </div>
                <p className="mt-0.5 truncate text-sm text-slate-600 dark:text-neutral-400">
                  {report.email}
                </p>
                <p className="mt-1 text-xs text-slate-500 dark:text-neutral-500">
                  {report.orgName}
                  {report.role ? ` · ${report.role}` : ''}
                </p>
              </div>
            </div>
            {!isErased ? (
              <div className="flex flex-wrap gap-2">
                {report.active ? (
                  <button
                    type="button"
                    disabled={busy}
                    onClick={onSuspend}
                    className="inline-flex items-center gap-2 rounded-xl border border-amber-200 bg-amber-50 px-3.5 py-2 text-sm font-medium text-amber-900 transition-colors hover:bg-amber-100 disabled:opacity-50 dark:border-amber-800/60 dark:bg-amber-950/40 dark:text-amber-200 dark:hover:bg-amber-950/70"
                  >
                    <UserMinus className="h-4 w-4" aria-hidden />
                    Suspend
                  </button>
                ) : (
                  <button
                    type="button"
                    disabled={busy}
                    onClick={onReactivate}
                    className="inline-flex items-center gap-2 rounded-xl border border-emerald-200 bg-emerald-50 px-3.5 py-2 text-sm font-medium text-emerald-900 transition-colors hover:bg-emerald-100 disabled:opacity-50 dark:border-emerald-800/60 dark:bg-emerald-950/40 dark:text-emerald-200 dark:hover:bg-emerald-950/70"
                  >
                    <UserPlus className="h-4 w-4" aria-hidden />
                    Reactivate
                  </button>
                )}
                <button
                  type="button"
                  disabled={busy}
                  onClick={onDelete}
                  className="inline-flex items-center gap-2 rounded-xl border border-red-200 bg-white px-3.5 py-2 text-sm font-medium text-red-700 transition-colors hover:bg-red-50 disabled:opacity-50 dark:border-red-900/60 dark:bg-neutral-900 dark:text-red-300 dark:hover:bg-red-950/40"
                >
                  <Trash2 className="h-4 w-4" aria-hidden />
                  Delete account
                </button>
              </div>
            ) : null}
          </div>
        </div>

        <dl className="grid gap-px bg-slate-100 sm:grid-cols-2 lg:grid-cols-4 dark:bg-neutral-800">
          {[
            { label: 'Joined', value: formatDateTime(report.createdAt) },
            {
              label: 'Last activity',
              value: report.lastActivityAt ? formatDateTime(report.lastActivityAt) : '—',
            },
            { label: 'Enrollments', value: String(report.enrollmentCount) },
            { label: 'Status', value: statusLabel(report.active) },
          ].map((item) => (
            <div key={item.label} className="bg-white px-5 py-4 dark:bg-neutral-900">
              <dt className="text-xs font-medium uppercase tracking-wide text-slate-500 dark:text-neutral-500">
                {item.label}
              </dt>
              <dd className="mt-1 text-sm font-medium text-slate-900 dark:text-neutral-100">
                {item.value}
              </dd>
            </div>
          ))}
        </dl>
      </div>

      <section className="overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-sm dark:border-neutral-800 dark:bg-neutral-900">
        <div className="border-b border-slate-100 px-5 py-3 dark:border-neutral-800">
          <h4 className="text-sm font-semibold text-slate-900 dark:text-neutral-100">Enrollments</h4>
        </div>
        {report.enrollments.length === 0 ? (
          <p className="px-5 py-8 text-center text-sm text-slate-500 dark:text-neutral-400">
            No enrollments.
          </p>
        ) : (
          <div className="overflow-x-auto">
            <table className="min-w-full text-left text-sm">
              <thead className="bg-slate-50/80 text-slate-500 dark:bg-neutral-950/60 dark:text-neutral-400">
                <tr>
                  <th scope="col" className="px-5 py-2.5 text-xs font-semibold uppercase tracking-wide">
                    Course
                  </th>
                  <th scope="col" className="px-5 py-2.5 text-xs font-semibold uppercase tracking-wide">
                    Role
                  </th>
                  <th scope="col" className="px-5 py-2.5 text-xs font-semibold uppercase tracking-wide">
                    State
                  </th>
                  <th scope="col" className="px-5 py-2.5 text-xs font-semibold uppercase tracking-wide">
                    Enrolled
                  </th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-100 dark:divide-neutral-800">
                {report.enrollments.map((e) => (
                  <tr
                    key={`${e.courseId}-${e.role}`}
                    className="transition-colors hover:bg-slate-50/70 dark:hover:bg-neutral-800/40"
                  >
                    <td className="px-5 py-3">
                      <span className="font-medium text-slate-900 dark:text-neutral-100">
                        {e.courseTitle}
                      </span>
                      <span className="ml-2 font-mono text-xs text-slate-500 dark:text-neutral-500">
                        {e.courseCode}
                      </span>
                    </td>
                    <td className="px-5 py-3 text-slate-700 dark:text-neutral-300">{e.role}</td>
                    <td className="px-5 py-3 text-slate-700 dark:text-neutral-300">{e.state}</td>
                    <td className="px-5 py-3 whitespace-nowrap text-slate-600 dark:text-neutral-400">
                      {formatDateTime(e.enrolledAt)}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </section>

      <section className="overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-sm dark:border-neutral-800 dark:bg-neutral-900">
        <div className="border-b border-slate-100 px-5 py-3 dark:border-neutral-800">
          <h4 className="text-sm font-semibold text-slate-900 dark:text-neutral-100">
            Recent activity
          </h4>
        </div>
        {report.recentActivity.length === 0 ? (
          <p className="px-5 py-8 text-center text-sm text-slate-500 dark:text-neutral-400">
            No recorded activity.
          </p>
        ) : (
          <div className="overflow-x-auto">
            <table className="min-w-full text-left text-sm">
              <thead className="bg-slate-50/80 text-slate-500 dark:bg-neutral-950/60 dark:text-neutral-400">
                <tr>
                  <th scope="col" className="px-5 py-2.5 text-xs font-semibold uppercase tracking-wide">
                    When
                  </th>
                  <th scope="col" className="px-5 py-2.5 text-xs font-semibold uppercase tracking-wide">
                    Event
                  </th>
                  <th scope="col" className="px-5 py-2.5 text-xs font-semibold uppercase tracking-wide">
                    Course
                  </th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-100 dark:divide-neutral-800">
                {report.recentActivity.map((a, i) => (
                  <tr
                    key={`${a.occurredAt}-${i}`}
                    className="transition-colors hover:bg-slate-50/70 dark:hover:bg-neutral-800/40"
                  >
                    <td className="px-5 py-3 whitespace-nowrap text-slate-600 dark:text-neutral-400">
                      {formatDateTime(a.occurredAt)}
                    </td>
                    <td className="px-5 py-3 capitalize text-slate-800 dark:text-neutral-200">
                      {a.eventKind.replaceAll('_', ' ')}
                    </td>
                    <td className="px-5 py-3">
                      <span className="text-slate-800 dark:text-neutral-200">{a.courseTitle}</span>
                      <span className="ml-2 font-mono text-xs text-slate-500">{a.courseCode}</span>
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
  const filterPanelId = useId()

  const [q, setQ] = useState('')
  const [submittedQ, setSubmittedQ] = useState('')
  const [page, setPage] = useState(1)
  const [perPage, setPerPage] = useState(25)
  const [data, setData] = useState<PaginatedPeople | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [busyId, setBusyId] = useState<string | null>(null)

  const [selectedFilter, setSelectedFilter] = useState<PeopleListFilter | null>(null)
  const [filterPage, setFilterPage] = useState(1)
  const [filterPerPage, setFilterPerPage] = useState(25)
  const [filterData, setFilterData] = useState<PaginatedPeople | null>(null)
  const [filterLoading, setFilterLoading] = useState(false)
  const [filterError, setFilterError] = useState<string | null>(null)

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

  const loadFilter = useCallback(async () => {
    if (!selectedFilter) {
      setFilterData(null)
      setFilterError(null)
      return
    }
    setFilterLoading(true)
    setFilterError(null)
    try {
      setFilterData(
        await searchPeople({
          filter: selectedFilter,
          page: filterPage,
          perPage: filterPerPage,
        }),
      )
    } catch (e) {
      setFilterError(e instanceof Error ? e.message : 'Failed to load accounts.')
      setFilterData(null)
    } finally {
      setFilterLoading(false)
    }
  }, [selectedFilter, filterPage, filterPerPage])

  useEffect(() => {
    void loadFilter()
  }, [loadFilter])

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

  function toggleFilter(filter: PeopleListFilter) {
    if (selectedFilter === filter) {
      setSelectedFilter(null)
      setFilterData(null)
      setFilterError(null)
      return
    }
    setSelectedFilter(filter)
    setFilterPage(1)
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
      void loadFilter()
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
      await Promise.all([loadSearch(), loadStats(), loadFilter()])
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
      await Promise.all([loadSearch(), loadStats(), loadFilter()])
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

  const resultCount = data?.total ?? data?.items.length
  const activeStat = STAT_CARDS.find((c) => c.filter === selectedFilter) ?? null
  const filterCount = filterData?.total

  return (
    <div className="mt-6 space-y-6">
      <PeopleDashboardCards
        stats={stats}
        loading={statsLoading}
        error={statsError}
        selectedFilter={selectedFilter}
        onSelectFilter={toggleFilter}
      />

      {selectedFilter && activeStat ? (
        <section
          id={filterPanelId}
          aria-label={activeStat.tableTitle}
          className="overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-sm dark:border-neutral-800 dark:bg-neutral-900"
        >
          <div className="flex flex-wrap items-start justify-between gap-3 border-b border-slate-100 px-5 py-4 dark:border-neutral-800">
            <div>
              <h3 className="text-sm font-semibold text-slate-900 dark:text-neutral-100">
                {activeStat.tableTitle}
              </h3>
              <p className="mt-0.5 text-sm text-slate-500 dark:text-neutral-400">
                {activeStat.tableDescription}
                {filterCount != null && !filterLoading ? (
                  <>
                    {' '}
                    <span className="font-medium text-slate-700 dark:text-neutral-300">
                      {formatCount(filterCount)}
                    </span>{' '}
                    {filterCount === 1 ? 'account' : 'accounts'}.
                  </>
                ) : null}
              </p>
            </div>
            <div className="flex flex-wrap items-center gap-2">
              <label className="flex items-center gap-2 text-sm text-slate-600 dark:text-neutral-400">
                <span className="text-xs font-medium uppercase tracking-wide">Page size</span>
                <select
                  value={filterPerPage}
                  onChange={(e) => {
                    setFilterPerPage(Number(e.target.value))
                    setFilterPage(1)
                  }}
                  className="rounded-lg border border-slate-200 bg-white px-2 py-1.5 text-sm dark:border-neutral-700 dark:bg-neutral-950"
                >
                  {PAGE_SIZES.map((n) => (
                    <option key={n} value={n}>
                      {n}
                    </option>
                  ))}
                </select>
              </label>
              <button
                type="button"
                onClick={() => toggleFilter(selectedFilter)}
                className="inline-flex items-center gap-1.5 rounded-lg border border-slate-200 px-3 py-1.5 text-sm font-medium text-slate-700 transition-colors hover:bg-slate-50 dark:border-neutral-700 dark:text-neutral-200 dark:hover:bg-neutral-800"
              >
                <X className="h-4 w-4" aria-hidden />
                Close
              </button>
            </div>
          </div>

          {filterError ? (
            <div className="border-b border-slate-100 px-5 py-3 dark:border-neutral-800">
              <p
                role="alert"
                className="rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700 dark:border-red-900/50 dark:bg-red-950/30 dark:text-red-300"
              >
                {filterError}
              </p>
            </div>
          ) : null}

          <PeopleResultsTable
            data={filterData}
            loading={filterLoading}
            busyId={busyId}
            emptyTitle="No accounts in this segment"
            emptyHint="Try another metric or invite a user."
            loadingLabel="Loading accounts…"
            showJoined
            onOpen={openPerson}
            onSuspend={(user) => void mutatePerson(user.id, false, personDisplayName(user))}
            onReactivate={(user) => void mutatePerson(user.id, true, '')}
            onDelete={(user) => void removePerson(user.id, personDisplayName(user))}
          />

          {filterData ? (
            <PaginationNav
              label={`${activeStat.tableTitle} pagination`}
              page={filterData.page}
              totalPages={filterData.totalPages}
              onPrev={() => setFilterPage((p) => p - 1)}
              onNext={() => setFilterPage((p) => p + 1)}
            />
          ) : null}
        </section>
      ) : null}

      <section className="overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-sm dark:border-neutral-800 dark:bg-neutral-900">
        <div className="flex flex-wrap items-center justify-between gap-3 border-b border-slate-100 px-5 py-4 dark:border-neutral-800">
          <div>
            <h3 className="text-sm font-semibold text-slate-900 dark:text-neutral-100">Find people</h3>
            <p className="mt-0.5 text-sm text-slate-500 dark:text-neutral-400">
              Search by first name, last name, or email. Results appear after you search.
            </p>
          </div>
          <button
            type="button"
            onClick={() => setInviteOpen(true)}
            className="inline-flex items-center gap-2 rounded-xl bg-indigo-600 px-4 py-2.5 text-sm font-medium text-white shadow-sm shadow-indigo-600/20 transition-colors hover:bg-indigo-700 dark:bg-indigo-500 dark:hover:bg-indigo-400 dark:text-white"
          >
            <MailPlus className="h-4 w-4" aria-hidden />
            Invite user
          </button>
        </div>

        <form onSubmit={onSearchSubmit} className="flex flex-wrap items-end gap-3 px-5 py-4">
          <label className="flex min-w-[16rem] flex-1 flex-col text-sm">
            <span className="mb-1.5 text-xs font-medium uppercase tracking-wide text-slate-500 dark:text-neutral-400">
              Search
            </span>
            <div className="relative">
              <Search
                className="pointer-events-none absolute left-3.5 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-400"
                aria-hidden
              />
              <input
                type="search"
                value={q}
                onChange={(e) => setQ(e.target.value)}
                placeholder="First name, last name, or email"
                className="w-full rounded-xl border border-slate-200 bg-slate-50/50 py-2.5 pl-10 pr-3 text-sm text-slate-900 shadow-inner transition-colors placeholder:text-slate-400 focus:border-indigo-300 focus:bg-white focus:outline-none focus:ring-2 focus:ring-indigo-500/20 dark:border-neutral-700 dark:bg-neutral-950 dark:text-neutral-100 dark:placeholder:text-neutral-500 dark:focus:border-indigo-500/50 dark:focus:bg-neutral-950"
              />
            </div>
          </label>
          <label className="flex flex-col text-sm">
            <span className="mb-1.5 text-xs font-medium uppercase tracking-wide text-slate-500 dark:text-neutral-400">
              Page size
            </span>
            <select
              value={perPage}
              onChange={(e) => {
                setPerPage(Number(e.target.value))
                setPage(1)
              }}
              className="rounded-xl border border-slate-200 bg-white px-3 py-2.5 text-sm dark:border-neutral-700 dark:bg-neutral-950"
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
            className="inline-flex items-center gap-2 rounded-xl border border-slate-200 bg-white px-4 py-2.5 text-sm font-medium text-slate-700 shadow-sm transition-colors hover:bg-slate-50 disabled:opacity-50 dark:border-neutral-700 dark:bg-neutral-900 dark:text-neutral-200 dark:hover:bg-neutral-800"
          >
            {loading ? (
              <Loader2 className="h-4 w-4 animate-spin" aria-hidden />
            ) : (
              <Search className="h-4 w-4" aria-hidden />
            )}
            {loading ? 'Searching…' : 'Search'}
          </button>
        </form>

        {error ? (
          <div className="border-t border-slate-100 px-5 py-3 dark:border-neutral-800">
            <p
              role="alert"
              className="rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700 dark:border-red-900/50 dark:bg-red-950/30 dark:text-red-300"
            >
              {error}
            </p>
          </div>
        ) : null}

        {!submittedQ.trim() ? (
          <div className="border-t border-slate-100 px-5 py-12 dark:border-neutral-800">
            <div className="mx-auto flex max-w-sm flex-col items-center text-center">
              <span className="flex h-14 w-14 items-center justify-center rounded-2xl bg-indigo-50 text-indigo-600 ring-1 ring-indigo-100 dark:bg-indigo-950/50 dark:text-indigo-300 dark:ring-indigo-900/50">
                <Sparkles className="h-6 w-6" aria-hidden />
              </span>
              <p className="mt-4 text-sm font-medium text-slate-900 dark:text-neutral-100">
                Search to manage accounts
              </p>
              <p className="mt-1.5 text-sm leading-relaxed text-slate-500 dark:text-neutral-400">
                Enter a name or email above, click a metric card above, or invite someone new.
              </p>
              <button
                type="button"
                onClick={() => setInviteOpen(true)}
                className="mt-5 inline-flex items-center gap-2 rounded-xl border border-slate-200 bg-white px-3.5 py-2 text-sm font-medium text-slate-700 transition-colors hover:bg-slate-50 dark:border-neutral-700 dark:bg-neutral-900 dark:text-neutral-200 dark:hover:bg-neutral-800"
              >
                <MailPlus className="h-4 w-4" aria-hidden />
                Invite user
              </button>
            </div>
          </div>
        ) : (
          <div className="border-t border-slate-100 dark:border-neutral-800">
            {data && !loading ? (
              <div className="flex items-center justify-between gap-3 border-b border-slate-100 px-5 py-2.5 dark:border-neutral-800">
                <p className="text-xs text-slate-500 dark:text-neutral-400">
                  {resultCount != null ? (
                    <>
                      <span className="font-medium text-slate-700 dark:text-neutral-300">
                        {formatCount(resultCount)}
                      </span>{' '}
                      {resultCount === 1 ? 'result' : 'results'} for{' '}
                      <span className="font-medium text-slate-700 dark:text-neutral-300">
                        “{submittedQ}”
                      </span>
                    </>
                  ) : (
                    <>
                      Results for{' '}
                      <span className="font-medium text-slate-700 dark:text-neutral-300">
                        “{submittedQ}”
                      </span>
                    </>
                  )}
                </p>
              </div>
            ) : null}
            <PeopleResultsTable
              data={data}
              loading={loading}
              busyId={busyId}
              emptyTitle="No users matched your search"
              emptyHint="Try a different name or email."
              loadingLabel="Searching…"
              onOpen={openPerson}
              onSuspend={(user) => void mutatePerson(user.id, false, personDisplayName(user))}
              onReactivate={(user) => void mutatePerson(user.id, true, '')}
              onDelete={(user) => void removePerson(user.id, personDisplayName(user))}
            />
          </div>
        )}

        {data ? (
          <PaginationNav
            label="People search pagination"
            page={data.page}
            totalPages={data.totalPages}
            onPrev={() => setPage((p) => p - 1)}
            onNext={() => setPage((p) => p + 1)}
          />
        ) : null}
      </section>

      {inviteOpen ? (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/40 p-4 backdrop-blur-[2px]"
          role="dialog"
          aria-modal="true"
          aria-labelledby="invite-user-title"
        >
          <form
            onSubmit={(e) => void onInvite(e)}
            className="w-full max-w-md overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-2xl shadow-slate-900/15 dark:border-neutral-700 dark:bg-neutral-900"
          >
            <div className="border-b border-slate-100 bg-gradient-to-r from-indigo-50/80 to-white px-6 py-5 dark:border-neutral-800 dark:from-indigo-950/40 dark:to-neutral-900">
              <div className="flex items-start gap-3">
                <span className="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-indigo-100 text-indigo-600 dark:bg-indigo-950 dark:text-indigo-300">
                  <MailPlus className="h-5 w-5" aria-hidden />
                </span>
                <div>
                  <h3
                    id="invite-user-title"
                    className="text-base font-semibold text-slate-900 dark:text-neutral-100"
                  >
                    Invite user
                  </h3>
                  <p className="mt-0.5 text-sm text-slate-500 dark:text-neutral-400">
                    Creates an account and emails a link to set their password.
                  </p>
                </div>
              </div>
            </div>
            <div className="space-y-3 px-6 py-5">
              <label className="block text-sm">
                <span className="text-xs font-medium uppercase tracking-wide text-slate-500 dark:text-neutral-400">
                  Email
                </span>
                <input
                  type="email"
                  required
                  value={inviteEmail}
                  onChange={(e) => setInviteEmail(e.target.value)}
                  className="mt-1.5 w-full rounded-xl border border-slate-200 px-3 py-2.5 text-sm focus:border-indigo-300 focus:outline-none focus:ring-2 focus:ring-indigo-500/20 dark:border-neutral-700 dark:bg-neutral-950"
                />
              </label>
              <div className="grid gap-3 sm:grid-cols-2">
                <label className="block text-sm">
                  <span className="text-xs font-medium uppercase tracking-wide text-slate-500 dark:text-neutral-400">
                    First name
                  </span>
                  <input
                    value={inviteFirst}
                    onChange={(e) => setInviteFirst(e.target.value)}
                    className="mt-1.5 w-full rounded-xl border border-slate-200 px-3 py-2.5 text-sm focus:border-indigo-300 focus:outline-none focus:ring-2 focus:ring-indigo-500/20 dark:border-neutral-700 dark:bg-neutral-950"
                  />
                </label>
                <label className="block text-sm">
                  <span className="text-xs font-medium uppercase tracking-wide text-slate-500 dark:text-neutral-400">
                    Last name
                  </span>
                  <input
                    value={inviteLast}
                    onChange={(e) => setInviteLast(e.target.value)}
                    className="mt-1.5 w-full rounded-xl border border-slate-200 px-3 py-2.5 text-sm focus:border-indigo-300 focus:outline-none focus:ring-2 focus:ring-indigo-500/20 dark:border-neutral-700 dark:bg-neutral-950"
                  />
                </label>
              </div>
            </div>
            <div className="flex justify-end gap-2 border-t border-slate-100 bg-slate-50/50 px-6 py-4 dark:border-neutral-800 dark:bg-neutral-950/40">
              <button
                type="button"
                onClick={() => setInviteOpen(false)}
                className="rounded-xl border border-slate-200 bg-white px-4 py-2 text-sm font-medium text-slate-700 transition-colors hover:bg-slate-50 dark:border-neutral-700 dark:bg-neutral-900 dark:text-neutral-200"
              >
                Cancel
              </button>
              <button
                type="submit"
                disabled={inviting}
                className="inline-flex items-center gap-2 rounded-xl bg-indigo-600 px-4 py-2 text-sm font-medium text-white shadow-sm shadow-indigo-600/20 transition-colors hover:bg-indigo-700 disabled:opacity-50"
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
