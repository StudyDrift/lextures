import { useEffect, useState } from 'react'
import { NodeViewWrapper, type NodeViewProps } from '@tiptap/react'
import { useTranslation } from 'react-i18next'
import { Link } from 'react-router-dom'
import { fetchBoardEmbed, type BoardEmbedContext } from '../../../lib/boards-api'
import { EmbeddedBoard } from '../../boards/embedded-board'

export function BoardNodeView(props: NodeViewProps) {
  const { t } = useTranslation('common')
  const boardId = String(props.node.attrs.boardId ?? '')
  const courseCode = String(props.node.attrs.courseCode ?? '')
  const [ctx, setCtx] = useState<BoardEmbedContext | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    if (!boardId || !courseCode) {
      setCtx(null)
      return
    }
    let cancelled = false
    setLoading(true)
    setError(null)
    void fetchBoardEmbed(courseCode, boardId)
      .then((next) => {
        if (!cancelled) setCtx(next)
      })
      .catch((err) => {
        if (!cancelled) setError(err instanceof Error ? err.message : String(err))
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [boardId, courseCode])

  return (
    <NodeViewWrapper
      as="div"
      className="lex-board-block my-4"
      contentEditable={false}
      data-type="board-block"
    >
      {!boardId || !courseCode ? (
        <div className="rounded-md border border-dashed border-slate-300 p-4 text-sm text-slate-600 dark:border-neutral-600 dark:text-neutral-300">
          {t('boards.embed.missing')}
        </div>
      ) : loading ? (
        <div className="rounded-md border border-slate-200 p-4 text-sm text-slate-500 dark:border-neutral-700">
          {t('common.loading')}
        </div>
      ) : error ? (
        <div className="rounded-md border border-red-200 bg-red-50 p-4 text-sm text-red-700 dark:border-red-900 dark:bg-red-950/30 dark:text-red-400">
          {t('boards.embed.loadError')}
        </div>
      ) : ctx?.mode === 'denied' || !ctx?.board ? (
        <div className="rounded-md border border-amber-200 bg-amber-50 p-4 text-sm text-amber-800 dark:border-amber-900 dark:bg-amber-950/30 dark:text-amber-300">
          {t('boards.embed.denied')}
        </div>
      ) : (
        <div className="overflow-hidden rounded-md border border-slate-200 dark:border-neutral-700">
          <div className="flex items-center justify-between gap-2 border-b border-slate-200 bg-slate-50 px-3 py-2 dark:border-neutral-700 dark:bg-neutral-900">
            <div className="min-w-0">
              <p className="truncate text-sm font-semibold text-slate-900 dark:text-neutral-100">
                {ctx.board.title}
              </p>
              <p className="text-xs text-slate-500 dark:text-neutral-400">
                {ctx.mode === 'interactive' ? t('boards.embed.interactive') : t('boards.embed.readonly')}
              </p>
            </div>
            <Link
              to={`/courses/${encodeURIComponent(courseCode)}/boards/${encodeURIComponent(boardId)}`}
              className="shrink-0 text-sm font-medium text-indigo-600 dark:text-indigo-400"
            >
              {t('boards.embed.open')}
            </Link>
          </div>
          <EmbeddedBoard
            courseCode={courseCode}
            board={ctx.board}
            posts={ctx.posts}
            sections={ctx.sections}
            readOnly={ctx.mode !== 'interactive'}
          />
        </div>
      )}
    </NodeViewWrapper>
  )
}
