import { useCallback, useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import {
  fetchContentToolInstanceStates,
  type ContentToolRosterStateRow,
} from '../../../lib/courses-api'
import { ResetHistoryList } from './reset-history-list'
import { ToolResetDialog } from './tool-reset-dialog'
import { ToolRosterTable } from './tool-roster-table'
import { ToolStateDetailDrawer } from './tool-state-detail-drawer'

export type ToolResponsesPanelProps = {
  open: boolean
  courseCode: string
  instanceId: string
  itemId?: string
  onClose: () => void
}

export function ToolResponsesPanel({
  open,
  courseCode,
  instanceId,
  itemId,
  onClose,
}: ToolResponsesPanelProps) {
  const { t } = useTranslation('contentTools')
  const [rows, setRows] = useState<ContentToolRosterStateRow[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [detailEnrollmentId, setDetailEnrollmentId] = useState<string | null>(null)
  const [resetEnrollmentId, setResetEnrollmentId] = useState<string | null>(null)
  const [resetOpen, setResetOpen] = useState(false)
  const [historyKey, setHistoryKey] = useState(0)
  const [sortKey, setSortKey] = useState<'displayName' | 'status' | 'interactionCount'>('displayName')
  const [sortDir, setSortDir] = useState<'asc' | 'desc'>('asc')

  const load = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const res = await fetchContentToolInstanceStates(courseCode, instanceId, {
        page: 1,
        pageSize: 100,
      })
      setRows(res.items)
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : 'Failed to load roster.')
    } finally {
      setLoading(false)
    }
  }, [courseCode, instanceId])

  useEffect(() => {
    if (open) void load()
  }, [open, load])

  const sorted = useMemo(() => {
    const copy = [...rows]
    copy.sort((a, b) => {
      const av = a[sortKey]
      const bv = b[sortKey]
      const cmp =
        typeof av === 'number' && typeof bv === 'number'
          ? av - bv
          : String(av).localeCompare(String(bv))
      return sortDir === 'asc' ? cmp : -cmp
    })
    return copy
  }, [rows, sortKey, sortDir])

  if (!open) return null

  return (
    <div
      className="fixed inset-0 z-[70] flex items-end justify-center bg-slate-900/40 p-0 sm:items-center sm:p-4"
      role="dialog"
      aria-modal="true"
      aria-labelledby="tool-responses-title"
      data-testid="tool-responses-panel"
    >
      <div className="max-h-[92vh] w-full max-w-3xl overflow-y-auto rounded-t-2xl border border-slate-200 bg-white p-4 shadow-xl sm:rounded-lg dark:border-neutral-700 dark:bg-neutral-900">
        <div className="flex items-start justify-between gap-2">
          <h2
            id="tool-responses-title"
            className="text-sm font-semibold text-slate-900 dark:text-neutral-100"
          >
            {t('contentTools.instructor.responsesTitle')}
          </h2>
          <div className="flex gap-2">
            <button
              type="button"
              className="rounded-md bg-rose-600 px-3 py-1.5 text-xs font-medium text-white"
              data-testid="tool-reset-all"
              onClick={() => {
                setResetEnrollmentId(null)
                setResetOpen(true)
              }}
            >
              {t('contentTools.reset.actionAll')}
            </button>
            <button
              type="button"
              className="rounded-md border border-slate-200 px-3 py-1.5 text-xs dark:border-neutral-600"
              onClick={onClose}
            >
              {t('contentTools.instructor.close')}
            </button>
          </div>
        </div>
        {error ? (
          <p className="mt-2 text-xs text-rose-600" role="alert">
            {error}{' '}
            <button type="button" className="underline" onClick={() => void load()}>
              {t('contentTools.runtime.retry')}
            </button>
          </p>
        ) : null}
        <div className="mt-3">
          <ToolRosterTable
            rows={sorted}
            loading={loading}
            empty={!loading && sorted.length === 0}
            sortKey={sortKey}
            sortDir={sortDir}
            onSort={(key) => {
              if (key === sortKey) setSortDir((d) => (d === 'asc' ? 'desc' : 'asc'))
              else {
                setSortKey(key)
                setSortDir('asc')
              }
            }}
            onSelect={(row) => setDetailEnrollmentId(row.enrollmentId)}
            onResetOne={(row) => {
              setResetEnrollmentId(row.enrollmentId)
              setResetOpen(true)
            }}
          />
        </div>
        <div className="mt-4">
          <ResetHistoryList
            courseCode={courseCode}
            instanceId={instanceId}
            refreshKey={historyKey}
          />
        </div>
      </div>

      <ToolStateDetailDrawer
        open={Boolean(detailEnrollmentId)}
        courseCode={courseCode}
        instanceId={instanceId}
        enrollmentId={detailEnrollmentId}
        onClose={() => setDetailEnrollmentId(null)}
        onReset={(id) => {
          setDetailEnrollmentId(null)
          setResetEnrollmentId(id)
          setResetOpen(true)
        }}
      />
      <ToolResetDialog
        open={resetOpen}
        courseCode={courseCode}
        instanceId={instanceId}
        itemId={itemId}
        enrollmentId={resetEnrollmentId}
        onClose={() => setResetOpen(false)}
        onCompleted={() => {
          setHistoryKey((k) => k + 1)
          void load()
        }}
      />
    </div>
  )
}
