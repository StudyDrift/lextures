import { useTranslation } from 'react-i18next'
import { PostCard } from '../post-card'
import { CardArrangeMenu } from '../card-arrange-menu'
import { postCardEngagementProps, type LayoutRendererProps } from './types'

export function TimelineLayout(props: LayoutRendererProps) {
  const { t } = useTranslation('common')
  const dated = props.posts
    .filter((p) => p.eventDate)
    .sort((a, b) => (a.eventDate ?? '').localeCompare(b.eventDate ?? ''))
  const undated = props.posts.filter((p) => !p.eventDate)

  if (props.posts.length === 0) {
    return (
      <p className="m-auto max-w-md px-4 text-center text-sm text-slate-500 dark:text-neutral-400">
        {t('boards.detail.emptyPosts')}
      </p>
    )
  }

  return (
    <div className="flex flex-col gap-4">
      <ol className="relative space-y-4 border-s-2 border-indigo-200 ps-6 dark:border-indigo-900" aria-label={t('boards.layout.timeline')}>
        {dated.map((post) => (
          <li key={post.id} className="relative">
            <span className="absolute -start-[1.55rem] top-3 size-3 rounded-full bg-indigo-500 ring-4 ring-slate-50 dark:ring-neutral-900" aria-hidden />
            <time className="mb-1 block text-xs font-medium text-slate-500" dateTime={post.eventDate}>
              {post.eventDate
                ? new Date(post.eventDate).toLocaleDateString(undefined, {
                    year: 'numeric',
                    month: 'short',
                    day: 'numeric',
                  })
                : null}
            </time>
            <div className="max-w-xl">
              <PostCard
                post={post}
                {...postCardEngagementProps(props, post)}
                headerActions={
                  <CardArrangeMenu
                    post={post}
                    sections={props.sections}
                    siblings={dated}
                    canArrange={props.canArrangePost(post)}
                    onMoveToSection={(sectionId) => void props.onArrange(post.id, { sectionId })}
                    onReorder={(sortIndex) => void props.onArrange(post.id, { sortIndex })}
                    showTimeline
                    onSetEventDate={(iso) => void props.onArrange(post.id, { eventDate: iso })}
                  />
                }
              />
            </div>
          </li>
        ))}
      </ol>

      <section
        className="rounded-lg border border-dashed border-slate-300 p-3 dark:border-neutral-600"
        aria-label={t('boards.layout.undatedTray')}
      >
        <h3 className="mb-2 text-sm font-semibold text-slate-700 dark:text-neutral-200">
          {t('boards.layout.undatedTray')}
        </h3>
        {undated.length === 0 ? (
          <p className="text-xs text-slate-400">{t('boards.layout.undatedEmpty')}</p>
        ) : (
          <ul className="grid grid-cols-1 gap-2 sm:grid-cols-2 lg:grid-cols-3">
            {undated.map((post) => (
              <li key={post.id}>
                <PostCard
                  post={post}
                  {...postCardEngagementProps(props, post)}
                  headerActions={
                    <CardArrangeMenu
                      post={post}
                      sections={props.sections}
                      siblings={undated}
                      canArrange={props.canArrangePost(post)}
                      onMoveToSection={(sectionId) => void props.onArrange(post.id, { sectionId })}
                      onReorder={(sortIndex) => void props.onArrange(post.id, { sortIndex })}
                      showTimeline
                      onSetEventDate={(iso) => void props.onArrange(post.id, { eventDate: iso })}
                    />
                  }
                />
              </li>
            ))}
          </ul>
        )}
      </section>
    </div>
  )
}
