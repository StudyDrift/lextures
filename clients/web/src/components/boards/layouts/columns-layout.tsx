import {
  DndContext,
  DragOverlay,
  PointerSensor,
  closestCorners,
  useDroppable,
  useSensor,
  useSensors,
  type DragEndEvent,
  type DragStartEvent,
} from '@dnd-kit/core'
import { SortableContext, useSortable, verticalListSortingStrategy } from '@dnd-kit/sortable'
import { CSS } from '@dnd-kit/utilities'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Plus, Trash2 } from 'lucide-react'
import { midpointSortIndex, type BoardPost, type BoardSection } from '../../../lib/boards-api'
import { postsInSection } from '../../../lib/board-sort'
import { toastMutationError } from '../../../lib/lms-toast'
import { PostCard } from '../post-card'
import { CardArrangeMenu } from '../card-arrange-menu'
import { postCardEngagementProps, type LayoutRendererProps } from './types'

function SortableCard({
  post,
  sections,
  siblings,
  canArrange,
  surface,
  onArrange,
  onAnnounce,
  movedLabel,
}: {
  post: BoardPost
  sections: BoardSection[]
  siblings: BoardPost[]
  canArrange: boolean
  surface: LayoutRendererProps
  onArrange: LayoutRendererProps['onArrange']
  onAnnounce: (msg: string) => void
  movedLabel: string
}) {
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({
    id: post.id,
    disabled: !canArrange,
    data: { type: 'post', post, sectionId: post.sectionId },
  })
  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
    opacity: isDragging ? 0.4 : 1,
  }

  return (
    <div ref={setNodeRef} style={style} className="relative">
      <div className="absolute end-2 top-2 z-10">
        <CardArrangeMenu
          post={post}
          sections={sections}
          siblings={siblings}
          canArrange={canArrange}
          onMoveToSection={(sectionId) => {
            void onArrange(post.id, { sectionId }).then(() => onAnnounce(movedLabel))
          }}
          onReorder={(sortIndex) => void onArrange(post.id, { sortIndex })}
        />
      </div>
      <div
        {...(canArrange ? { ...attributes, ...listeners } : {})}
        className={canArrange ? 'cursor-grab active:cursor-grabbing' : undefined}
      >
        <PostCard post={post} {...postCardEngagementProps(surface, post)} />
      </div>
    </div>
  )
}

function SectionColumn({
  section,
  posts,
  allSections,
  props,
}: {
  section: BoardSection
  posts: BoardPost[]
  allSections: BoardSection[]
  props: LayoutRendererProps
}) {
  const { t } = useTranslation('common')
  const { setNodeRef, isOver } = useDroppable({
    id: `section:${section.id}`,
    data: { sectionId: section.id },
  })

  return (
    <div
      ref={setNodeRef}
      className={`flex w-72 shrink-0 flex-col gap-2 rounded-lg border bg-white/80 p-2 dark:bg-neutral-900/60 ${
        isOver ? 'border-indigo-400 ring-2 ring-indigo-300/50' : 'border-slate-200 dark:border-neutral-700'
      }`}
    >
      <div className="flex items-center justify-between gap-2 px-1">
        <h3 className="truncate text-sm font-semibold text-slate-800 dark:text-neutral-100">{section.title}</h3>
        {props.canManageBoard && section.title !== 'Unsorted' ? (
          <button
            type="button"
            aria-label={t('boards.section.delete')}
            className="rounded p-1 text-slate-400 hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-950/30"
            onClick={() => {
              void props.onDeleteSection(section.id).catch((err: unknown) => {
                toastMutationError(err instanceof Error ? err.message : String(err))
              })
            }}
          >
            <Trash2 className="size-3.5" aria-hidden />
          </button>
        ) : null}
      </div>
      <SortableContext items={posts.map((p) => p.id)} strategy={verticalListSortingStrategy}>
        <div className="flex min-h-24 flex-col gap-2">
          {posts.length === 0 ? (
            <p className="px-1 py-6 text-center text-xs text-slate-400">{t('boards.section.dropHere')}</p>
          ) : (
            posts.map((post) => (
              <SortableCard
                key={post.id}
                post={post}
                sections={allSections}
                siblings={posts}
                canArrange={props.canArrangePost(post)}
                surface={props}
                onArrange={props.onArrange}
                onAnnounce={props.onAnnounce}
                movedLabel={t('boards.arrange.moved')}
              />
            ))
          )}
        </div>
      </SortableContext>
    </div>
  )
}

export function ColumnsLayout(props: LayoutRendererProps) {
  const { t } = useTranslation('common')
  const [activeId, setActiveId] = useState<string | null>(null)
  const [creating, setCreating] = useState(false)
  const [titleDraft, setTitleDraft] = useState('')
  const sensors = useSensors(useSensor(PointerSensor, { activationConstraint: { distance: 6 } }))

  const sections = [...props.sections].sort((a, b) => a.sortIndex - b.sortIndex)
  const activePost = activeId ? props.posts.find((p) => p.id === activeId) : null

  async function addSection() {
    if (!titleDraft.trim()) return
    try {
      const created = await props.onCreateSection(titleDraft.trim())
      props.onSectionsChange([...props.sections, created])
      props.onAnnounce(t('boards.section.created', { title: created.title }))
      setTitleDraft('')
      setCreating(false)
    } catch (err) {
      toastMutationError(err instanceof Error ? err.message : String(err))
    }
  }

  function onDragStart(e: DragStartEvent) {
    setActiveId(String(e.active.id))
  }

  async function onDragEnd(e: DragEndEvent) {
    setActiveId(null)
    const { active, over } = e
    if (!over) return
    const post = props.posts.find((p) => p.id === active.id)
    if (!post || !props.canArrangePost(post)) return

    let targetSectionId = post.sectionId
    const overData = over.data.current as { sectionId?: string } | undefined
    if (String(over.id).startsWith('section:')) {
      targetSectionId = String(over.id).replace('section:', '')
    } else if (overData?.sectionId) {
      targetSectionId = overData.sectionId
    } else {
      const overPost = props.posts.find((p) => p.id === over.id)
      if (overPost?.sectionId) targetSectionId = overPost.sectionId
    }
    if (!targetSectionId) return

    const siblings = postsInSection(props.posts, targetSectionId).filter((p) => p.id !== post.id)
    const overIndex = siblings.findIndex((p) => p.id === over.id)
    let sortIndex: number
    if (overIndex < 0) {
      const last = siblings[siblings.length - 1]?.sortIndex
      sortIndex = midpointSortIndex(last, undefined)
    } else {
      const before = siblings[overIndex - 1]?.sortIndex
      const after = siblings[overIndex]?.sortIndex
      sortIndex = midpointSortIndex(before, after)
    }

    try {
      await props.onArrange(post.id, { sectionId: targetSectionId, sortIndex })
      props.onAnnounce(t('boards.arrange.moved'))
    } catch (err) {
      toastMutationError(err instanceof Error ? err.message : String(err))
    }
  }

  return (
    <div className="flex min-h-64 flex-col gap-3">
      <DndContext
        sensors={sensors}
        collisionDetection={closestCorners}
        onDragStart={onDragStart}
        onDragEnd={(e) => void onDragEnd(e)}
      >
        <div className="flex gap-3 overflow-x-auto pb-2">
          {sections.map((section) => (
            <SectionColumn
              key={section.id}
              section={section}
              posts={postsInSection(props.posts, section.id)}
              allSections={sections}
              props={props}
            />
          ))}
          {props.canManageBoard ? (
            <div className="w-64 shrink-0">
              {creating ? (
                <div className="flex flex-col gap-2 rounded-lg border border-dashed border-slate-300 p-2 dark:border-neutral-600">
                  <input
                    value={titleDraft}
                    onChange={(e) => setTitleDraft(e.target.value)}
                    placeholder={t('boards.section.titlePlaceholder')}
                    className="rounded border border-slate-200 px-2 py-1.5 text-sm dark:border-neutral-700 dark:bg-neutral-800"
                    aria-label={t('boards.section.titlePlaceholder')}
                  />
                  <div className="flex gap-2">
                    <button
                      type="button"
                      className="rounded bg-indigo-600 px-2 py-1 text-xs font-medium text-white"
                      onClick={() => void addSection()}
                    >
                      {t('boards.section.add')}
                    </button>
                    <button
                      type="button"
                      className="rounded px-2 py-1 text-xs text-slate-600"
                      onClick={() => setCreating(false)}
                    >
                      {t('dialogs.cancel')}
                    </button>
                  </div>
                </div>
              ) : (
                <button
                  type="button"
                  onClick={() => setCreating(true)}
                  className="flex w-full items-center justify-center gap-1 rounded-lg border border-dashed border-slate-300 px-3 py-6 text-sm text-slate-500 hover:border-indigo-400 hover:text-indigo-600 dark:border-neutral-600"
                >
                  <Plus className="size-4" aria-hidden />
                  {t('boards.section.add')}
                </button>
              )}
            </div>
          ) : null}
        </div>
        <DragOverlay>
          {activePost ? (
            <div className="w-72 opacity-90 shadow-lg">
              <PostCard post={activePost} {...postCardEngagementProps(props, activePost)} />
            </div>
          ) : null}
        </DragOverlay>
      </DndContext>
    </div>
  )
}
