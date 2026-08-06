import { useId, useState } from 'react'
import { useTranslation } from 'react-i18next'
import {
  createBoardExport,
  downloadBoardExport,
  renderBoardSurfacePng,
  type BoardExportFormat,
  type BoardPost,
  type BoardSection,
} from '../../lib/boards-api'
import { toastMutationError } from '../../lib/lms-toast'

type Props = {
  open: boolean
  onClose: () => void
  courseCode: string
  boardId: string
  boardTitle: string
  posts: BoardPost[]
  sections: BoardSection[]
}

function triggerDownload(blob: Blob, filename: string) {
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  a.click()
  URL.revokeObjectURL(url)
}

function bodyText(post: BoardPost): string {
  const body = post.body
  if (!body) return ''
  if (typeof body === 'string') return body
  if (typeof body.text === 'string') return body.text
  return ''
}

export function BoardExportMenu({
  open,
  onClose,
  courseCode,
  boardId,
  boardTitle,
  posts,
  sections,
}: Props) {
  const { t } = useTranslation('common')
  const titleId = useId()
  const [busy, setBusy] = useState<BoardExportFormat | 'client-image' | null>(null)
  const [includeModeration, setIncludeModeration] = useState(false)
  const [error, setError] = useState<string | null>(null)

  if (!open) return null

  async function runServerExport(format: BoardExportFormat) {
    setBusy(format)
    setError(null)
    try {
      const job = await createBoardExport(courseCode, boardId, { format, includeModeration })
      if (job.status === 'failed') {
        throw new Error(job.error || t('boards.export.failed'))
      }
      if (job.status !== 'done') {
        throw new Error(t('boards.export.notReady'))
      }
      const blob = await downloadBoardExport(courseCode, boardId, job.id)
      const ext = format === 'image' ? 'png' : format
      triggerDownload(blob, `${boardTitle || 'board'}.${ext}`)
      onClose()
    } catch (err) {
      const msg = err instanceof Error ? err.message : String(err)
      setError(msg)
      toastMutationError(msg)
    } finally {
      setBusy(null)
    }
  }

  async function runClientImage() {
    setBusy('client-image')
    setError(null)
    try {
      const secMap = new Map(sections.map((s) => [s.id, s.title]))
      const cards = posts
        .filter((p) => !p.hidden && !p.removed && p.status === 'approved')
        .map((p) => ({
          sectionTitle: p.sectionId ? secMap.get(p.sectionId) : undefined,
          title: p.title,
          body: bodyText(p),
        }))
      const blob = await renderBoardSurfacePng(boardTitle, cards)
      triggerDownload(blob, `${boardTitle || 'board'}.png`)
      onClose()
    } catch (err) {
      const msg = err instanceof Error ? err.message : String(err)
      setError(msg)
      toastMutationError(msg)
    } finally {
      setBusy(null)
    }
  }

  return (
    <div className="fixed inset-0 z-40 flex items-center justify-center bg-black/40 p-4" role="presentation">
      <div
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        className="w-full max-w-md rounded-lg bg-surface-raised p-5 shadow-xl dark:bg-surface-raised"
      >
        <h2 id={titleId} className="text-lg font-semibold text-fg-default">
          {t('boards.export.title')}
        </h2>
        <p className="mt-1 text-sm text-fg-muted">{t('boards.export.subtitle')}</p>
        <label className="mt-4 flex items-center gap-2 text-sm text-fg-default">
          <input
            type="checkbox"
            checked={includeModeration}
            onChange={(e) => setIncludeModeration(e.target.checked)}
          />
          {t('boards.export.includeModeration')}
        </label>
        {error ? (
          <p className="mt-3 text-sm text-danger-fg" role="alert">
            {error}
          </p>
        ) : null}
        <div className="mt-4 flex flex-col gap-2">
          {(['pdf', 'csv', 'image'] as BoardExportFormat[]).map((format) => (
            <button
              key={format}
              type="button"
              disabled={busy !== null}
              className="rounded-md border border-border-strong px-3 py-2 text-start text-sm hover:bg-surface-base disabled:opacity-60 dark:border-border-default dark:hover:bg-surface-overlay"
              onClick={() => void runServerExport(format)}
            >
              {busy === format ? t('boards.export.working') : t(`boards.export.${format}`)}
            </button>
          ))}
          <button
            type="button"
            disabled={busy !== null}
            className="rounded-md border border-border-strong px-3 py-2 text-start text-sm hover:bg-surface-base disabled:opacity-60 dark:border-border-default dark:hover:bg-surface-overlay"
            onClick={() => void runClientImage()}
          >
            {busy === 'client-image' ? t('boards.export.working') : t('boards.export.clientImage')}
          </button>
        </div>
        <div className="mt-4 flex justify-end">
          <button
            type="button"
            className="rounded-md px-3 py-1.5 text-sm text-fg-muted"
            onClick={onClose}
          >
            {t('dialogs.cancel')}
          </button>
        </div>
      </div>
    </div>
  )
}
