import { useEffect, useId, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { fetchBoardQR } from '../../lib/boards-api'
import { toastMutationError } from '../../lib/lms-toast'

type Props = {
  open: boolean
  onClose: () => void
  courseCode: string
  boardId: string
  /** Optional VC.6 share URL (must be on app origin); defaults to in-app board URL. */
  shareUrl?: string | null
}

export function BoardQuickJoinPanel({ open, onClose, courseCode, boardId, shareUrl }: Props) {
  const { t } = useTranslation('common')
  const titleId = useId()
  const [imgUrl, setImgUrl] = useState<string | null>(null)
  const [accessUrl, setAccessUrl] = useState('')
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    if (!open) return
    let cancelled = false
    let objectUrl: string | null = null
    setLoading(true)
    void fetchBoardQR(courseCode, boardId, {
      format: 'png',
      size: 320,
      url: shareUrl || undefined,
    })
      .then(({ blob, accessUrl: url }) => {
        if (cancelled) return
        objectUrl = URL.createObjectURL(blob)
        setImgUrl(objectUrl)
        setAccessUrl(url)
      })
      .catch((err) => {
        if (!cancelled) toastMutationError(err instanceof Error ? err.message : String(err))
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => {
      cancelled = true
      if (objectUrl) URL.revokeObjectURL(objectUrl)
    }
  }, [open, courseCode, boardId, shareUrl])

  if (!open) return null

  return (
    <div className="fixed inset-0 z-40 flex items-center justify-center bg-black/40 p-4" role="presentation">
      <div
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        className="w-full max-w-md rounded-lg bg-white p-6 text-center shadow-xl dark:bg-neutral-900"
      >
        <h2 id={titleId} className="text-xl font-semibold text-slate-900 dark:text-neutral-100">
          {t('boards.export.quickJoinTitle')}
        </h2>
        <p className="mt-1 text-sm text-slate-600 dark:text-neutral-300">
          {t('boards.export.quickJoinSubtitle')}
        </p>
        <div className="mx-auto mt-6 flex min-h-64 items-center justify-center">
          {loading || !imgUrl ? (
            <span className="text-sm text-slate-500">{t('common.loading')}</span>
          ) : (
            <img
              src={imgUrl}
              alt={t('boards.export.qrAlt')}
              className="max-h-72 w-auto rounded-md border border-slate-200 dark:border-neutral-700"
            />
          )}
        </div>
        {accessUrl ? (
          <p className="mt-4 break-all text-sm text-slate-700 dark:text-neutral-200">
            <span className="sr-only">{t('boards.export.accessUrlLabel')} </span>
            <a href={accessUrl} className="text-indigo-600 underline dark:text-indigo-400">
              {accessUrl}
            </a>
          </p>
        ) : null}
        <div className="mt-6 flex justify-center gap-2">
          <button
            type="button"
            className="rounded-md border border-slate-300 px-3 py-1.5 text-sm dark:border-neutral-600"
            onClick={() => {
              if (accessUrl) void navigator.clipboard?.writeText(accessUrl)
            }}
          >
            {t('boards.export.copyUrl')}
          </button>
          <button
            type="button"
            className="rounded-md bg-indigo-600 px-3 py-1.5 text-sm font-medium text-white"
            onClick={onClose}
          >
            {t('dialogs.close')}
          </button>
        </div>
      </div>
    </div>
  )
}
