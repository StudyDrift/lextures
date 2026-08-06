import { useTranslation } from 'react-i18next'
import type { Board, BoardPost, BoardSection } from '../../lib/boards-api'

type Props = {
  courseCode: string
  board: Board
  posts: BoardPost[]
  sections: BoardSection[]
  readOnly?: boolean
}

/** Compact read-focused surface for TipTap board embeds (VC.9). */
export function EmbeddedBoard({ board, posts, sections, readOnly }: Props) {
  const { t } = useTranslation('common')
  const secMap = new Map(sections.map((s) => [s.id, s.title]))
  const visible = posts.filter((p) => !p.removed && (readOnly ? !p.hidden && p.status === 'approved' : true))

  return (
    <div
      className="max-h-96 overflow-auto bg-surface-raised p-3 dark:bg-surface-base"
      role="region"
      aria-label={t('boards.embed.surfaceAria', { title: board.title })}
    >
      {visible.length === 0 ? (
        <p className="text-sm text-fg-muted">{t('boards.detail.emptyPosts')}</p>
      ) : (
        <ul className="space-y-2">
          {visible.slice(0, 40).map((p) => (
            <li
              key={p.id}
              className="rounded-md border border-border-default px-3 py-2 dark:border-border-default"
            >
              {p.sectionId && secMap.get(p.sectionId) ? (
                <p className="text-xs font-medium uppercase tracking-wide text-accent-fg">
                  {secMap.get(p.sectionId)}
                </p>
              ) : null}
              <p className="text-sm font-semibold text-fg-default">
                {p.title || t('boards.present.untitled')}
              </p>
              {p.body?.text ? (
                <p className="mt-0.5 line-clamp-3 text-sm text-fg-muted">{p.body.text}</p>
              ) : null}
            </li>
          ))}
        </ul>
      )}
      {readOnly ? (
        <p className="mt-2 text-xs text-fg-muted">{t('boards.embed.readonlyHint')}</p>
      ) : null}
    </div>
  )
}
