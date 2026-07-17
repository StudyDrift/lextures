import { useTranslation } from 'react-i18next'
import { PostCard } from '../post-card'
import { CardArrangeMenu } from '../card-arrange-menu'
import { sortBoardPosts } from '../../../lib/board-sort'
import { postCardEngagementProps, type LayoutRendererProps } from './types'

export function StreamLayout(props: LayoutRendererProps) {
  const { t } = useTranslation('common')
  const posts = sortBoardPosts(props.posts, props.sortMode, props.board.reactionMode)

  if (posts.length === 0) {
    return (
      <p className="m-auto max-w-md px-4 text-center text-sm text-slate-500 dark:text-neutral-400">
        {t('boards.detail.emptyPosts')}
      </p>
    )
  }

  return (
    <ul className="mx-auto flex w-full max-w-2xl flex-col gap-3" aria-label={t('boards.layout.stream')}>
      {posts.map((post) => (
        <li key={post.id}>
          <PostCard
            post={post}
            {...postCardEngagementProps(props, post)}
            headerActions={
              <CardArrangeMenu
                post={post}
                sections={props.sections}
                siblings={posts}
                canArrange={props.canArrangePost(post)}
                onMoveToSection={(sectionId) => void props.onArrange(post.id, { sectionId })}
                onReorder={(sortIndex) => void props.onArrange(post.id, { sortIndex })}
              />
            }
          />
        </li>
      ))}
    </ul>
  )
}
