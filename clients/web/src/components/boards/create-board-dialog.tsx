import { useEffect, useId, useState } from 'react'
import { useTranslation } from 'react-i18next'
import {
  createBoard,
  createBoardFromTemplate,
  duplicateBoard,
  fetchBoardCopyJob,
  listBoardTemplates,
  listBoards,
  type Board,
  type BoardCopyJob,
  type BoardCopyMode,
  type BoardTemplate,
  type BoardTemplateScope,
} from '../../lib/boards-api'
import { toastMutationError } from '../../lib/lms-toast'

type Tab = 'blank' | 'templates' | 'duplicate'

type Props = {
  open: boolean
  onClose: () => void
  courseCode: string
  onCreated: (board: Board) => void
}

export function CreateBoardDialog({ open, onClose, courseCode, onCreated }: Props) {
  const { t, i18n } = useTranslation('common')
  const titleId = useId()
  const [tab, setTab] = useState<Tab>('blank')
  const [title, setTitle] = useState('')
  const [description, setDescription] = useState('')
  const [submitting, setSubmitting] = useState(false)

  const [scopeFilter, setScopeFilter] = useState<BoardTemplateScope | ''>('')
  const [templateQuery, setTemplateQuery] = useState('')
  const [templates, setTemplates] = useState<BoardTemplate[]>([])
  const [templatesLoading, setTemplatesLoading] = useState(false)
  const [selectedTemplateId, setSelectedTemplateId] = useState<string | null>(null)

  const [boards, setBoards] = useState<Board[]>([])
  const [boardsLoading, setBoardsLoading] = useState(false)
  const [sourceBoardId, setSourceBoardId] = useState('')
  const [copyMode, setCopyMode] = useState<BoardCopyMode>('structure')
  const [copyJob, setCopyJob] = useState<BoardCopyJob | null>(null)

  useEffect(() => {
    if (!open) return
    setTab('blank')
    setTitle('')
    setDescription('')
    setSelectedTemplateId(null)
    setSourceBoardId('')
    setCopyMode('structure')
    setCopyJob(null)
    setTemplateQuery('')
    setScopeFilter('')
  }, [open])

  useEffect(() => {
    if (!open || tab !== 'templates') return
    let cancelled = false
    setTemplatesLoading(true)
    void listBoardTemplates({
      scope: scopeFilter || undefined,
      courseCode,
      q: templateQuery.trim() || undefined,
      locale: i18n.language,
    })
      .then((list) => {
        if (!cancelled) setTemplates(list)
      })
      .catch(() => {
        if (!cancelled) setTemplates([])
      })
      .finally(() => {
        if (!cancelled) setTemplatesLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [open, tab, scopeFilter, templateQuery, courseCode, i18n.language])

  useEffect(() => {
    if (!open || tab !== 'duplicate') return
    let cancelled = false
    setBoardsLoading(true)
    void listBoards(courseCode)
      .then((list) => {
        if (!cancelled) setBoards(list)
      })
      .catch(() => {
        if (!cancelled) setBoards([])
      })
      .finally(() => {
        if (!cancelled) setBoardsLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [open, tab, courseCode])

  useEffect(() => {
    if (!copyJob || copyJob.status === 'completed' || copyJob.status === 'failed') return
    const timer = window.setInterval(() => {
      void fetchBoardCopyJob(courseCode, copyJob.id)
        .then((job) => {
          setCopyJob(job)
          if (job.status === 'completed' && job.resultBoardId) {
            onCreated({
              id: job.resultBoardId,
              courseId: '',
              title: job.title,
              description: '',
              slug: '',
              archived: false,
              layout: 'wall',
              layoutLocked: false,
              settings: {},
              reactionMode: 'none',
              assignmentId: null,
              visibility: 'course',
              visibilityTarget: null,
              attribution: 'named',
              canPost: true,
              canInteract: true,
              canArrange: false,
              moderationMode: 'open',
              filterAction: 'flag',
              locked: false,
              frozenUntil: null,
              createdBy: null,
              createdAt: job.updatedAt,
              updatedAt: job.updatedAt,
            })
          }
        })
        .catch(() => {
          /* keep polling */
        })
    }, 1500)
    return () => window.clearInterval(timer)
  }, [copyJob, courseCode, onCreated])

  if (!open) return null

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setSubmitting(true)
    setCopyJob(null)
    try {
      if (tab === 'blank') {
        if (!title.trim()) return
        const created = await createBoard(courseCode, title.trim(), description.trim())
        onCreated(created)
        return
      }
      if (tab === 'templates') {
        if (!selectedTemplateId) return
        const tmpl = templates.find((x) => x.id === selectedTemplateId)
        const created = await createBoardFromTemplate(
          courseCode,
          selectedTemplateId,
          title.trim() || tmpl?.title,
          description.trim(),
        )
        onCreated(created)
        return
      }
      if (!sourceBoardId) return
      const result = await duplicateBoard(
        courseCode,
        sourceBoardId,
        copyMode,
        title.trim() || undefined,
        description.trim(),
      )
      if (result.kind === 'job') {
        setCopyJob(result.job)
        return
      }
      onCreated(result.board)
    } catch (err) {
      toastMutationError(err instanceof Error ? err.message : String(err))
    } finally {
      setSubmitting(false)
    }
  }

  const canSubmit =
    tab === 'blank'
      ? !!title.trim()
      : tab === 'templates'
        ? !!selectedTemplateId
        : !!sourceBoardId

  return (
    <div
      className="fixed inset-0 z-50 flex items-end justify-center bg-black/40 p-0 sm:items-center sm:p-4"
      role="presentation"
      onClick={onClose}
    >
      <div
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        className="flex max-h-[100dvh] w-full max-w-2xl flex-col rounded-none bg-white shadow-xl dark:bg-neutral-900 sm:max-h-[90vh] sm:rounded-lg"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="border-b border-slate-200 px-4 py-3 dark:border-neutral-700">
          <h2 id={titleId} className="text-lg font-semibold text-slate-900 dark:text-neutral-100">
            {t('boards.create.dialogTitle')}
          </h2>
          <p className="mt-1 text-sm text-slate-600 dark:text-neutral-300">
            {t('boards.create.dialogSubtitle')}
          </p>
        </div>

        <div
          className="flex gap-1 border-b border-slate-200 px-2 dark:border-neutral-700"
          role="tablist"
          aria-label={t('boards.create.tabsAria')}
        >
          {(
            [
              ['blank', 'boards.create.tabBlank'],
              ['templates', 'boards.create.tabTemplates'],
              ['duplicate', 'boards.create.tabDuplicate'],
            ] as const
          ).map(([id, labelKey]) => (
            <button
              key={id}
              type="button"
              role="tab"
              aria-selected={tab === id}
              id={`create-board-tab-${id}`}
              aria-controls={`create-board-panel-${id}`}
              onClick={() => setTab(id)}
              className={`px-3 py-2 text-sm font-medium focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-indigo-500 ${
                tab === id
                  ? 'border-b-2 border-indigo-600 text-indigo-700 dark:text-indigo-300'
                  : 'text-slate-600 hover:text-slate-900 dark:text-neutral-400 dark:hover:text-neutral-100'
              }`}
            >
              {t(labelKey)}
            </button>
          ))}
        </div>

        <form
          onSubmit={(e) => {
            void handleSubmit(e)
          }}
          className="flex min-h-0 flex-1 flex-col"
        >
          <div
            id={`create-board-panel-${tab}`}
            role="tabpanel"
            aria-labelledby={`create-board-tab-${tab}`}
            className="min-h-0 flex-1 space-y-3 overflow-y-auto px-4 py-3"
          >
            {tab === 'templates' ? (
              <>
                <div className="flex flex-wrap gap-2" role="group" aria-label={t('boards.template.filterAria')}>
                  {(
                    [
                      ['', 'boards.template.filterAll'],
                      ['builtin', 'boards.template.filterBuiltin'],
                      ['course', 'boards.template.filterCourse'],
                      ['org', 'boards.template.filterOrg'],
                    ] as const
                  ).map(([value, labelKey]) => (
                    <button
                      key={value || 'all'}
                      type="button"
                      onClick={() => setScopeFilter(value)}
                      className={`rounded-md px-2.5 py-1 text-xs font-medium ${
                        scopeFilter === value
                          ? 'bg-indigo-600 text-white'
                          : 'bg-slate-100 text-slate-700 dark:bg-neutral-800 dark:text-neutral-200'
                      }`}
                    >
                      {t(labelKey)}
                    </button>
                  ))}
                </div>
                <label className="block text-sm">
                  <span className="sr-only">{t('boards.template.search')}</span>
                  <input
                    type="search"
                    value={templateQuery}
                    onChange={(e) => setTemplateQuery(e.target.value)}
                    placeholder={t('boards.template.search')}
                    className="w-full rounded-md border border-slate-300 px-3 py-2 text-sm dark:border-neutral-600 dark:bg-neutral-800"
                  />
                </label>
                {templatesLoading ? (
                  <p className="text-sm text-slate-500">{t('common.loading')}</p>
                ) : templates.length === 0 ? (
                  <p className="text-sm text-slate-600 dark:text-neutral-300">{t('boards.template.empty')}</p>
                ) : (
                  <ul className="grid gap-2 sm:grid-cols-2" role="listbox" aria-label={t('boards.template.galleryAria')}>
                    {templates.map((tmpl) => {
                      const selected = selectedTemplateId === tmpl.id
                      return (
                        <li key={tmpl.id}>
                          <button
                            type="button"
                            role="option"
                            aria-selected={selected}
                            onClick={() => {
                              setSelectedTemplateId(tmpl.id)
                              if (!title.trim()) setTitle(tmpl.title)
                            }}
                            className={`w-full rounded-md border p-3 text-start transition-[background-color,color,border-color] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-indigo-500 ${
                              selected
                                ? 'border-indigo-500 bg-indigo-50 dark:border-indigo-400 dark:bg-indigo-950/40'
                                : 'border-slate-200 hover:border-indigo-300 dark:border-neutral-700'
                            }`}
                          >
                            <span className="block font-medium text-slate-900 dark:text-neutral-100">
                              {tmpl.title}
                            </span>
                            <span className="mt-1 block text-xs text-slate-600 dark:text-neutral-300">
                              {tmpl.description}
                            </span>
                            {tmpl.tags.length > 0 ? (
                              <span className="mt-2 flex flex-wrap gap-1">
                                {tmpl.tags.slice(0, 4).map((tag) => (
                                  <span
                                    key={tag}
                                    className="rounded bg-slate-100 px-1.5 py-0.5 text-[10px] uppercase tracking-wide text-slate-600 dark:bg-neutral-800 dark:text-neutral-300"
                                  >
                                    {tag}
                                  </span>
                                ))}
                              </span>
                            ) : null}
                          </button>
                        </li>
                      )
                    })}
                  </ul>
                )}
              </>
            ) : null}

            {tab === 'duplicate' ? (
              <>
                {boardsLoading ? (
                  <p className="text-sm text-slate-500">{t('common.loading')}</p>
                ) : boards.length === 0 ? (
                  <p className="text-sm text-slate-600 dark:text-neutral-300">
                    {t('boards.create.duplicateEmpty')}
                  </p>
                ) : (
                  <label className="block text-sm">
                    <span className="font-medium text-slate-700 dark:text-neutral-200">
                      {t('boards.create.sourceBoard')}
                    </span>
                    <select
                      value={sourceBoardId}
                      onChange={(e) => setSourceBoardId(e.target.value)}
                      className="mt-1 w-full rounded-md border border-slate-300 px-3 py-2 text-sm dark:border-neutral-600 dark:bg-neutral-800"
                      required
                    >
                      <option value="">{t('boards.create.sourceBoardPlaceholder')}</option>
                      {boards.map((b) => (
                        <option key={b.id} value={b.id}>
                          {b.title}
                        </option>
                      ))}
                    </select>
                  </label>
                )}
                <fieldset className="space-y-2">
                  <legend className="text-sm font-medium text-slate-700 dark:text-neutral-200">
                    {t('boards.create.copyMode')}
                  </legend>
                  <label className="flex items-start gap-2 text-sm">
                    <input
                      type="radio"
                      name="copy-mode"
                      checked={copyMode === 'structure'}
                      onChange={() => setCopyMode('structure')}
                      className="mt-1"
                    />
                    <span>
                      <span className="font-medium">{t('boards.create.modeStructure')}</span>
                      <span className="block text-xs text-slate-500 dark:text-neutral-400">
                        {t('boards.create.modeStructureHint')}
                      </span>
                    </span>
                  </label>
                  <label className="flex items-start gap-2 text-sm">
                    <input
                      type="radio"
                      name="copy-mode"
                      checked={copyMode === 'full'}
                      onChange={() => setCopyMode('full')}
                      className="mt-1"
                    />
                    <span>
                      <span className="font-medium">{t('boards.create.modeFull')}</span>
                      <span className="block text-xs text-slate-500 dark:text-neutral-400">
                        {t('boards.create.modeFullHint')}
                      </span>
                    </span>
                  </label>
                </fieldset>
                {copyJob ? (
                  <div className="rounded-md border border-slate-200 p-3 dark:border-neutral-700" role="status">
                    <p className="text-sm font-medium text-slate-800 dark:text-neutral-100">
                      {copyJob.status === 'failed'
                        ? t('boards.create.copyFailed')
                        : t('boards.create.copyProgress', { progress: copyJob.progress })}
                    </p>
                    <div className="mt-2 h-2 overflow-hidden rounded bg-slate-100 dark:bg-neutral-800">
                      <div
                        className="h-full bg-indigo-600 motion-safe:transition-all"
                        style={{ width: `${Math.min(100, Math.max(0, copyJob.progress))}%` }}
                      />
                    </div>
                    {copyJob.status === 'failed' && copyJob.error ? (
                      <p className="mt-2 text-xs text-red-600 dark:text-red-400">{copyJob.error}</p>
                    ) : null}
                  </div>
                ) : null}
              </>
            ) : null}

            <div>
              <label
                className="block text-sm font-medium text-slate-700 dark:text-neutral-300"
                htmlFor="create-board-title"
              >
                {t('boards.create.titleLabel')}
                {tab !== 'blank' ? (
                  <span className="font-normal text-slate-500"> ({t('boards.create.titleOptional')})</span>
                ) : null}
              </label>
              <input
                id="create-board-title"
                type="text"
                value={title}
                onChange={(e) => setTitle(e.target.value)}
                maxLength={200}
                required={tab === 'blank'}
                className="mt-1 w-full rounded-md border border-slate-300 px-3 py-2 text-sm shadow-sm focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500 dark:border-neutral-600 dark:bg-neutral-800 dark:text-neutral-100"
              />
            </div>
            <div>
              <label
                className="block text-sm font-medium text-slate-700 dark:text-neutral-300"
                htmlFor="create-board-description"
              >
                {t('boards.create.descriptionLabel')}
              </label>
              <textarea
                id="create-board-description"
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                rows={2}
                className="mt-1 w-full rounded-md border border-slate-300 px-3 py-2 text-sm shadow-sm focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500 dark:border-neutral-600 dark:bg-neutral-800 dark:text-neutral-100"
              />
            </div>
          </div>

          <div className="flex justify-end gap-2 border-t border-slate-200 px-4 py-3 dark:border-neutral-700">
            <button
              type="button"
              onClick={onClose}
              className="rounded-md px-3 py-2 text-sm font-medium text-slate-700 hover:bg-slate-100 dark:text-neutral-200 dark:hover:bg-neutral-800"
            >
              {t('dialogs.cancel')}
            </button>
            <button
              type="submit"
              disabled={submitting || !canSubmit || (!!copyJob && copyJob.status !== 'failed')}
              className="rounded-md bg-indigo-600 px-3 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50"
            >
              {submitting ? t('common.loading') : t('boards.create.submit')}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}
