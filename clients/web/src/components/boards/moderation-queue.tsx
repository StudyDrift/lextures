import { useCallback, useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import {
  approveBoardPost,
  fetchBoardModerationQueue,
  rejectBoardPost,
  resolveBoardReport,
  type BoardModerationQueue,
  type BoardPost,
  type BoardReport,
} from '../../lib/boards-api'
import { toastMutationError } from '../../lib/lms-toast'

type Tab = 'pending' | 'reports' | 'flagged'

type Props = {
  open: boolean
  onClose: () => void
  courseCode: string
  boardId: string
  onChanged: () => void
}

export function BoardModerationQueue({ open, onClose, courseCode, boardId, onChanged }: Props) {
  const { t } = useTranslation('common')
  const [tab, setTab] = useState<Tab>('pending')
  const [queue, setQueue] = useState<BoardModerationQueue | null>(null)
  const [loading, setLoading] = useState(false)
  const [busyId, setBusyId] = useState<string | null>(null)

  const load = useCallback(async () => {
    setLoading(true)
    try {
      setQueue(await fetchBoardModerationQueue(courseCode, boardId))
    } catch (err) {
      toastMutationError(err instanceof Error ? err.message : String(err))
    } finally {
      setLoading(false)
    }
  }, [boardId, courseCode])

  useEffect(() => {
    if (open) void load()
  }, [open, load])

  if (!open) return null

  async function runPostAction(postId: string, action: 'approve' | 'reject') {
    setBusyId(postId)
    try {
      if (action === 'approve') await approveBoardPost(courseCode, boardId, postId)
      else await rejectBoardPost(courseCode, boardId, postId)
      await load()
      onChanged()
    } catch (err) {
      toastMutationError(err instanceof Error ? err.message : String(err))
    } finally {
      setBusyId(null)
    }
  }

  async function runReportAction(reportId: string, action: 'dismiss' | 'hide' | 'remove') {
    setBusyId(reportId)
    try {
      await resolveBoardReport(courseCode, boardId, reportId, action)
      await load()
      onChanged()
    } catch (err) {
      toastMutationError(err instanceof Error ? err.message : String(err))
    } finally {
      setBusyId(null)
    }
  }

  const tabs: { id: Tab; label: string; count: number }[] = [
    { id: 'pending', label: t('boards.moderation.tabPending'), count: queue?.pending.length ?? 0 },
    { id: 'reports', label: t('boards.moderation.tabReports'), count: queue?.reports.length ?? 0 },
    { id: 'flagged', label: t('boards.moderation.tabFlagged'), count: queue?.flagged.length ?? 0 },
  ]

  return (
    <div
      className="fixed inset-0 z-50 flex items-end justify-center bg-black/40 p-4 sm:items-center"
      role="dialog"
      aria-modal="true"
      aria-labelledby="board-moderation-title"
      onClick={(e) => {
        if (e.target === e.currentTarget) onClose()
      }}
    >
      <div className="flex max-h-[90vh] w-full max-w-2xl flex-col rounded-lg border border-slate-200 bg-white shadow-xl dark:border-neutral-700 dark:bg-neutral-900">
        <div className="flex items-center justify-between border-b border-slate-200 px-4 py-3 dark:border-neutral-700">
          <h2 id="board-moderation-title" className="text-lg font-semibold text-slate-900 dark:text-neutral-100">
            {t('boards.moderation.title')}
          </h2>
          <button
            type="button"
            onClick={onClose}
            className="rounded-md px-2 py-1 text-sm text-slate-600 hover:bg-slate-100 dark:text-neutral-300 dark:hover:bg-neutral-800"
          >
            {t('dialogs.close')}
          </button>
        </div>
        <div className="flex gap-1 border-b border-slate-200 px-2 pt-2 dark:border-neutral-700" role="tablist">
          {tabs.map((item) => (
            <button
              key={item.id}
              type="button"
              role="tab"
              aria-selected={tab === item.id}
              onClick={() => setTab(item.id)}
              className={`rounded-t-md px-3 py-2 text-sm font-medium ${
                tab === item.id
                  ? 'bg-slate-100 text-slate-900 dark:bg-neutral-800 dark:text-neutral-100'
                  : 'text-slate-600 hover:bg-slate-50 dark:text-neutral-400 dark:hover:bg-neutral-800/60'
              }`}
            >
              {item.label} ({item.count})
            </button>
          ))}
        </div>
        <div className="min-h-0 flex-1 overflow-y-auto p-4" role="tabpanel">
          {loading || !queue ? (
            <p className="text-sm text-slate-500">{t('common.loading')}</p>
          ) : tab === 'pending' ? (
            <PendingList
              posts={queue.pending}
              busyId={busyId}
              onApprove={(id) => void runPostAction(id, 'approve')}
              onReject={(id) => void runPostAction(id, 'reject')}
            />
          ) : (
            <ReportList
              reports={tab === 'reports' ? queue.reports : queue.flagged}
              busyId={busyId}
              onDismiss={(id) => void runReportAction(id, 'dismiss')}
              onHide={(id) => void runReportAction(id, 'hide')}
              onRemove={(id) => void runReportAction(id, 'remove')}
            />
          )}
        </div>
      </div>
    </div>
  )
}

function PendingList({
  posts,
  busyId,
  onApprove,
  onReject,
}: {
  posts: BoardPost[]
  busyId: string | null
  onApprove: (id: string) => void
  onReject: (id: string) => void
}) {
  const { t } = useTranslation('common')
  if (posts.length === 0) {
    return <p className="text-sm text-slate-500 dark:text-neutral-400">{t('boards.moderation.emptyPending')}</p>
  }
  return (
    <ul className="space-y-3">
      {posts.map((post) => (
        <li
          key={post.id}
          className="rounded-md border border-slate-200 p-3 dark:border-neutral-700"
        >
          <p className="font-medium text-slate-900 dark:text-neutral-100">{post.title || t('boards.moderation.untitled')}</p>
          <p className="mt-1 line-clamp-3 text-sm text-slate-600 dark:text-neutral-300">
            {post.body?.text || post.body?.html?.replace(/<[^>]+>/g, ' ') || ''}
          </p>
          <div className="mt-3 flex flex-wrap gap-2">
            <button
              type="button"
              disabled={busyId === post.id}
              onClick={() => onApprove(post.id)}
              className="rounded-md bg-indigo-600 px-3 py-1.5 text-sm font-medium text-white disabled:opacity-50"
            >
              {t('boards.moderation.approve')}
            </button>
            <button
              type="button"
              disabled={busyId === post.id}
              onClick={() => onReject(post.id)}
              className="rounded-md border border-slate-300 px-3 py-1.5 text-sm dark:border-neutral-600 disabled:opacity-50"
            >
              {t('boards.moderation.reject')}
            </button>
          </div>
        </li>
      ))}
    </ul>
  )
}

function ReportList({
  reports,
  busyId,
  onDismiss,
  onHide,
  onRemove,
}: {
  reports: BoardReport[]
  busyId: string | null
  onDismiss: (id: string) => void
  onHide: (id: string) => void
  onRemove: (id: string) => void
}) {
  const { t } = useTranslation('common')
  if (reports.length === 0) {
    return <p className="text-sm text-slate-500 dark:text-neutral-400">{t('boards.moderation.emptyReports')}</p>
  }
  return (
    <ul className="space-y-3">
      {reports.map((report) => (
        <li
          key={report.id}
          className="rounded-md border border-slate-200 p-3 dark:border-neutral-700"
        >
          <p className="text-sm font-medium text-slate-900 dark:text-neutral-100">
            {t(`boards.moderation.kind.${report.kind}`)}
          </p>
          {report.reason ? (
            <p className="mt-1 text-sm text-slate-600 dark:text-neutral-300">{report.reason}</p>
          ) : null}
          <div className="mt-3 flex flex-wrap gap-2">
            <button
              type="button"
              disabled={busyId === report.id}
              onClick={() => onDismiss(report.id)}
              className="rounded-md border border-slate-300 px-3 py-1.5 text-sm dark:border-neutral-600 disabled:opacity-50"
            >
              {t('boards.moderation.dismiss')}
            </button>
            <button
              type="button"
              disabled={busyId === report.id}
              onClick={() => onHide(report.id)}
              className="rounded-md border border-slate-300 px-3 py-1.5 text-sm dark:border-neutral-600 disabled:opacity-50"
            >
              {t('boards.moderation.hide')}
            </button>
            <button
              type="button"
              disabled={busyId === report.id}
              onClick={() => onRemove(report.id)}
              className="rounded-md bg-red-600 px-3 py-1.5 text-sm font-medium text-white disabled:opacity-50"
            >
              {t('boards.moderation.remove')}
            </button>
          </div>
        </li>
      ))}
    </ul>
  )
}
