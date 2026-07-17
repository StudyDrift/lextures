import { useCallback, useEffect, useId, useRef, useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import {
  ArrowDown,
  ArrowLeft,
  ArrowUp,
  CheckCircle2,
  Copy,
  Plus,
  Sparkles,
  Trash2,
  Upload,
} from 'lucide-react'
import { BankImportDrawer } from '../../components/live-quiz/bank-import-drawer'
import { GenerateWithAiPanel } from '../../components/live-quiz/generate-with-ai-panel'
import { MediaAttach } from '../../components/live-quiz/media-attach'
import { McOptionList } from '../../components/live-quiz/mc-option-list'
import { NumericEditor, type NumericDraft } from '../../components/live-quiz/numeric-editor'
import { OrderingEditor } from '../../components/live-quiz/ordering-editor'
import { PollEditor } from '../../components/live-quiz/poll-editor'
import { QuestionTypePicker } from '../../components/live-quiz/question-type-picker'
import {
  TypeAnswerEditor,
  type AcceptedAnswerDraft,
} from '../../components/live-quiz/type-answer-editor'
import { WordCloudEditor } from '../../components/live-quiz/word-cloud-editor'
import { ModePicker } from '../../components/live-quiz/mode-picker'
import {
  defaultModeStartOptions,
  type ModeStartOptions,
} from '../../components/live-quiz/mode-start-options'
import { ScoringProfilePicker } from '../../components/live-quiz/scoring-profile-picker'
import {
  defaultScoringStartOptions,
  type ScoringStartOptions,
} from '../../components/live-quiz/scoring-start-options'
import {
  createQuizQuestion,
  deleteQuizQuestion,
  duplicateQuizQuestion,
  fetchQuizKit,
  importBankQuestions,
  listQuizQuestions,
  patchQuizKit,
  patchQuizQuestion,
  regenerateQuizQuestion,
  reorderQuizQuestions,
  startLiveGame,
  validateQuizKit,
  VersionConflictError,
  type KitValidationIssue,
  type LiveQuizOption,
  type LiveQuizPointsStyle,
  type LiveQuizQuestion,
  type LiveQuizQuestionType,
  type QuizKit,
} from '../../lib/live-quiz-api'
import { courseItemCreatePermission, fetchCourse } from '../../lib/courses-api'
import { toastMutationError } from '../../lib/lms-toast'
import { useCoursePageTitle } from '../../context/course-document-title-context'
import { usePermissions } from '../../context/use-permissions'
import { usePlatformFeatures } from '../../context/platform-features-context'
import { LmsPage } from './lms-page'

type SaveState = 'idle' | 'saving' | 'saved' | 'conflict'

function parseAccepted(correct: unknown): AcceptedAnswerDraft[] {
  if (correct && typeof correct === 'object' && Array.isArray((correct as { accepted?: unknown }).accepted)) {
    return ((correct as { accepted: AcceptedAnswerDraft[] }).accepted).map((a) => ({
      text: a.text ?? '',
      matchMode: a.matchMode ?? 'case_insensitive',
      fuzzyMax: a.fuzzyMax,
    }))
  }
  return [{ text: '', matchMode: 'case_insensitive' }]
}

function parseNumeric(correct: unknown): NumericDraft {
  if (correct && typeof correct === 'object') {
    const c = correct as NumericDraft
    return {
      value: typeof c.value === 'number' ? c.value : 0,
      tolerance: typeof c.tolerance === 'number' ? c.tolerance : 0,
      unit: c.unit,
    }
  }
  return { value: 0, tolerance: 0 }
}

export default function LiveQuizKitEditorPage() {
  const { t } = useTranslation('common')
  const navigate = useNavigate()
  const { courseCode: rawCode, kitId: rawKitId } = useParams<{
    courseCode: string
    kitId: string
  }>()
  const courseCode = rawCode ? decodeURIComponent(rawCode) : ''
  const kitId = rawKitId ? decodeURIComponent(rawKitId) : ''
  const { allows, loading: permLoading } = usePermissions()
  const {
    ffIqLiveHosting,
    ffIqTeamMode,
    ffIqStudentPaced,
    ffIqAiGeneration,
    aiConfigured,
  } = usePlatformFeatures()
  const [hosting, setHosting] = useState(false)
  const [hostDialogOpen, setHostDialogOpen] = useState(false)
  const [scoringOpts, setScoringOpts] = useState<ScoringStartOptions>(defaultScoringStartOptions)
  const [modeOpts, setModeOpts] = useState<ModeStartOptions>(defaultModeStartOptions)
  const hostDialogTitleId = useId()
  const hostCancelRef = useRef<HTMLButtonElement>(null)
  const canEdit = !permLoading && !!courseCode && allows(courseItemCreatePermission(courseCode))

  const [kit, setKit] = useState<QuizKit | null>(null)
  const [questions, setQuestions] = useState<LiveQuizQuestion[]>([])
  const [selectedId, setSelectedId] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [bankEnabled, setBankEnabled] = useState(false)
  const [importOpen, setImportOpen] = useState(false)
  const [aiOpen, setAiOpen] = useState(false)
  const [aiLikeQuestionId, setAiLikeQuestionId] = useState<string | null>(null)
  const [regenBusy, setRegenBusy] = useState(false)
  const [saveState, setSaveState] = useState<SaveState>('idle')
  const [issues, setIssues] = useState<KitValidationIssue[]>([])
  const [renaming, setRenaming] = useState(false)
  const [titleDraft, setTitleDraft] = useState('')

  // Draft fields for selected question
  const [prompt, setPrompt] = useState('')
  const [qType, setQType] = useState<LiveQuizQuestionType>('mc_single')
  const [options, setOptions] = useState<LiveQuizOption[]>([])
  const [accepted, setAccepted] = useState<AcceptedAnswerDraft[]>([{ text: '', matchMode: 'case_insensitive' }])
  const [numeric, setNumeric] = useState<NumericDraft>({ value: 0, tolerance: 0 })
  const [timer, setTimer] = useState(20)
  const [pointsStyle, setPointsStyle] = useState<LiveQuizPointsStyle>('standard')
  const [shuffle, setShuffle] = useState(true)
  const [explanation, setExplanation] = useState('')
  const [mediaRef, setMediaRef] = useState('')
  const [mediaAlt, setMediaAlt] = useState('')
  const [version, setVersion] = useState(1)

  const saveTimer = useRef<ReturnType<typeof setTimeout> | null>(null)
  const skipAutosave = useRef(true)

  useCoursePageTitle(kit?.title ?? t('liveQuiz.kit.editorTitle'))
  const listBase = `/courses/${encodeURIComponent(courseCode)}/live-quizzes`
  const selected = questions.find((q) => q.id === selectedId) ?? null

  const loadDraftFromQuestion = useCallback((q: LiveQuizQuestion) => {
    skipAutosave.current = true
    setPrompt(q.prompt)
    setQType(q.questionType)
    setOptions(q.options)
    setAccepted(parseAccepted(q.correctAnswer))
    setNumeric(parseNumeric(q.correctAnswer))
    setTimer(q.timeLimitSeconds)
    setPointsStyle(q.pointsStyle)
    setShuffle(q.answerShuffle)
    setExplanation(q.explanation ?? '')
    setMediaRef(q.promptMediaRef ?? '')
    setMediaAlt(q.promptMediaAlt ?? '')
    setVersion(q.version)
    queueMicrotask(() => {
      skipAutosave.current = false
    })
  }, [])

  const load = useCallback(async () => {
    if (!courseCode || !kitId) return
    setLoading(true)
    setError(null)
    try {
      const course = await fetchCourse(courseCode)
      if (!course.interactiveQuizzesEnabled) {
        setError(t('liveQuiz.error.disabled'))
        return
      }
      setBankEnabled(course.questionBankEnabled === true)
      const [row, qs] = await Promise.all([
        fetchQuizKit(courseCode, kitId),
        listQuizQuestions(courseCode, kitId),
      ])
      setKit(row)
      setTitleDraft(row.title)
      setQuestions(qs)
      if (qs.length > 0) {
        setSelectedId((prev) => prev ?? qs[0].id)
        loadDraftFromQuestion(qs.find((q) => q.id === (selectedId ?? qs[0].id)) ?? qs[0])
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : t('liveQuiz.error.loadKit'))
    } finally {
      setLoading(false)
    }
  }, [courseCode, kitId, loadDraftFromQuestion, selectedId, t])

  useEffect(() => {
    void load()
    // intentionally once on mount / kit change
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [courseCode, kitId])

  useEffect(() => {
    if (!selectedId) return
    const q = questions.find((x) => x.id === selectedId)
    if (q) loadDraftFromQuestion(q)
  }, [selectedId]) // eslint-disable-line react-hooks/exhaustive-deps -- only re-bind when selection changes

  const persistDraft = useCallback(async () => {
    if (!canEdit || !selectedId || skipAutosave.current) return
    setSaveState('saving')
    let correctAnswer: unknown = null
    if (qType === 'type_answer') correctAnswer = { accepted }
    else if (qType === 'numeric') correctAnswer = numeric
    else if (qType === 'ordering') correctAnswer = { order: options.map((o) => o.id) }
    try {
      const updated = await patchQuizQuestion(courseCode, kitId, selectedId, version, {
        questionType: qType,
        prompt,
        options:
          qType === 'type_answer' || qType === 'numeric' || qType === 'word_cloud' ? [] : options,
        correctAnswer,
        timeLimitSeconds: timer,
        pointsStyle,
        answerShuffle: shuffle,
        explanation: explanation || null,
        promptMediaRef: mediaRef || null,
        promptMediaAlt: mediaAlt || null,
      })
      setQuestions((prev) => prev.map((q) => (q.id === updated.id ? updated : q)))
      setVersion(updated.version)
      setSaveState('saved')
    } catch (err) {
      if (err instanceof VersionConflictError) {
        setSaveState('conflict')
        toastMutationError(t('liveQuiz.editor.conflict'))
        return
      }
      setSaveState('idle')
      toastMutationError(err instanceof Error ? err.message : String(err))
    }
  }, [
    accepted,
    canEdit,
    courseCode,
    explanation,
    kitId,
    mediaAlt,
    mediaRef,
    numeric,
    options,
    pointsStyle,
    prompt,
    qType,
    selectedId,
    shuffle,
    t,
    timer,
    version,
  ])

  useEffect(() => {
    if (skipAutosave.current || !canEdit || !selectedId) return
    if (saveTimer.current) clearTimeout(saveTimer.current)
    saveTimer.current = setTimeout(() => {
      void persistDraft()
    }, 600)
    return () => {
      if (saveTimer.current) clearTimeout(saveTimer.current)
    }
  }, [
    prompt,
    qType,
    options,
    accepted,
    numeric,
    timer,
    pointsStyle,
    shuffle,
    explanation,
    mediaRef,
    mediaAlt,
    canEdit,
    selectedId,
    persistDraft,
  ])

  async function handleAdd(type: LiveQuizQuestionType = 'mc_single') {
    try {
      const created = await createQuizQuestion(courseCode, kitId, {
        questionType: type,
        prompt: '',
        timeLimitSeconds: 20,
      })
      setQuestions((prev) => [...prev, created])
      setSelectedId(created.id)
      setKit((k) => (k ? { ...k, questionCount: k.questionCount + 1 } : k))
    } catch (err) {
      toastMutationError(err instanceof Error ? err.message : String(err))
    }
  }

  async function handleDelete(id: string) {
    try {
      await deleteQuizQuestion(courseCode, kitId, id)
      setQuestions((prev) => {
        const next = prev.filter((q) => q.id !== id)
        if (selectedId === id) setSelectedId(next[0]?.id ?? null)
        return next
      })
      setKit((k) => (k ? { ...k, questionCount: Math.max(0, k.questionCount - 1) } : k))
    } catch (err) {
      toastMutationError(err instanceof Error ? err.message : String(err))
    }
  }

  async function handleDuplicate(id: string) {
    try {
      const created = await duplicateQuizQuestion(courseCode, kitId, id)
      setQuestions((prev) => [...prev, created])
      setSelectedId(created.id)
      setKit((k) => (k ? { ...k, questionCount: k.questionCount + 1 } : k))
    } catch (err) {
      toastMutationError(err instanceof Error ? err.message : String(err))
    }
  }

  async function moveQuestion(id: string, dir: -1 | 1) {
    const idx = questions.findIndex((q) => q.id === id)
    const next = idx + dir
    if (idx < 0 || next < 0 || next >= questions.length) return
    const reordered = [...questions]
    const [item] = reordered.splice(idx, 1)
    reordered.splice(next, 0, item)
    const items = reordered.map((q, position) => ({ id: q.id, position }))
    try {
      const qs = await reorderQuizQuestions(courseCode, kitId, items)
      setQuestions(qs)
    } catch (err) {
      toastMutationError(err instanceof Error ? err.message : String(err))
    }
  }

  async function handleValidate() {
    try {
      const result = await validateQuizKit(courseCode, kitId)
      setIssues(result.issues)
      if (result.isReady) {
        const updated = await patchQuizKit(courseCode, kitId, { status: 'ready' })
        setKit(updated)
      }
    } catch (err) {
      toastMutationError(err instanceof Error ? err.message : String(err))
    }
  }

  async function openHostDialog() {
    if (!ffIqLiveHosting) {
      toastMutationError(t('liveQuiz.error.hostingDisabled'))
      return
    }
    try {
      const result = await validateQuizKit(courseCode, kitId)
      setIssues(result.issues)
      if (!result.isReady) {
        toastMutationError(t('liveQuiz.validate.issuesHeading'))
        return
      }
      setHostDialogOpen(true)
    } catch (err) {
      toastMutationError(err instanceof Error ? err.message : String(err))
    }
  }

  useEffect(() => {
    if (!hostDialogOpen) return
    const timer = window.setTimeout(() => hostCancelRef.current?.focus(), 0)
    return () => window.clearTimeout(timer)
  }, [hostDialogOpen])

  useEffect(() => {
    if (!hostDialogOpen) return
    function onKey(e: KeyboardEvent) {
      if (e.key === 'Escape' && !hosting) {
        e.preventDefault()
        setHostDialogOpen(false)
      }
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [hostDialogOpen, hosting])

  async function handleHost() {
    if (!ffIqLiveHosting) {
      toastMutationError(t('liveQuiz.error.hostingDisabled'))
      return
    }
    setHosting(true)
    try {
      const started = await startLiveGame(courseCode, kitId, {
        pacing: 'manual',
        mode: modeOpts.mode,
        teamConfig: modeOpts.mode === 'team' ? modeOpts.teamConfig : undefined,
        pacedConfig: modeOpts.mode === 'student_paced' ? modeOpts.pacedConfig : undefined,
        scoringProfile: scoringOpts.scoringProfile,
        scoringConfig: {
          ...scoringOpts.scoringConfig,
          powerUpsEnabled: scoringOpts.powerUpsEnabled,
        },
        leaderboardPrivacy: scoringOpts.leaderboardPrivacy,
        powerUpsEnabled: scoringOpts.powerUpsEnabled,
      })
      setHostDialogOpen(false)
      navigate(
        `/courses/${encodeURIComponent(courseCode)}/live-quizzes/games/${encodeURIComponent(started.gameId)}`,
      )
    } catch (err) {
      toastMutationError(err instanceof Error ? err.message : String(err))
    } finally {
      setHosting(false)
    }
  }

  async function handleRename(e: React.FormEvent) {
    e.preventDefault()
    if (!kit || !titleDraft.trim()) return
    try {
      const updated = await patchQuizKit(courseCode, kit.id, { title: titleDraft.trim() })
      setKit(updated)
      setRenaming(false)
    } catch (err) {
      toastMutationError(err instanceof Error ? err.message : String(err))
    }
  }

  function renderTypeEditor() {
    switch (qType) {
      case 'mc_single':
        return (
          <McOptionList
            options={options}
            onChange={setOptions}
            multiCorrect={false}
            disabled={!canEdit}
          />
        )
      case 'mc_multiple':
        return (
          <McOptionList
            options={options}
            onChange={setOptions}
            multiCorrect
            disabled={!canEdit}
          />
        )
      case 'true_false':
        return (
          <McOptionList
            options={options}
            onChange={setOptions}
            multiCorrect={false}
            disabled={!canEdit}
            min={2}
            max={2}
          />
        )
      case 'poll':
        return <PollEditor options={options} onChange={setOptions} disabled={!canEdit} />
      case 'ordering':
        return <OrderingEditor options={options} onChange={setOptions} disabled={!canEdit} />
      case 'type_answer':
        return <TypeAnswerEditor accepted={accepted} onChange={setAccepted} disabled={!canEdit} />
      case 'numeric':
        return <NumericEditor value={numeric} onChange={setNumeric} disabled={!canEdit} />
      case 'word_cloud':
        return <WordCloudEditor />
      default: {
        const _exhaustive: never = qType
        return _exhaustive
      }
    }
  }

  return (
    <LmsPage title={kit?.title ?? t('liveQuiz.kit.editorTitle')}>
      <div className="mb-4 flex flex-wrap items-center justify-between gap-3">
        <Link
          to={listBase}
          className="inline-flex min-h-11 items-center gap-1.5 text-sm text-indigo-600 hover:underline dark:text-indigo-400"
        >
          <ArrowLeft className="h-4 w-4" aria-hidden />
          {t('liveQuiz.kit.backToGallery')}
        </Link>
        <div className="flex flex-wrap items-center gap-2 text-sm text-slate-500 dark:text-neutral-400">
          {saveState === 'saving' ? t('liveQuiz.editor.saving') : null}
          {saveState === 'saved' ? t('liveQuiz.editor.saved') : null}
          {saveState === 'conflict' ? (
            <span className="text-amber-600 dark:text-amber-400">{t('liveQuiz.editor.conflict')}</span>
          ) : null}
        </div>
      </div>

      {loading ? (
        <p className="text-sm text-slate-500 dark:text-neutral-400">{t('common.loading')}</p>
      ) : error ? (
        <div className="rounded-md bg-red-50 p-4 text-sm text-red-700 dark:bg-red-950/30 dark:text-red-400">
          {error}
        </div>
      ) : kit ? (
        <div className="space-y-4">
          <div className="flex flex-wrap items-center gap-3">
            {renaming && canEdit ? (
              <form
                onSubmit={(e) => {
                  void handleRename(e)
                }}
                className="flex flex-wrap items-end gap-2"
              >
                <input
                  value={titleDraft}
                  onChange={(e) => setTitleDraft(e.target.value)}
                  maxLength={200}
                  required
                  autoFocus
                  className="min-w-[16rem] min-h-11 rounded-md border border-slate-300 px-3 py-2 text-sm dark:border-neutral-600 dark:bg-neutral-800 dark:text-neutral-100"
                />
                <button
                  type="submit"
                  className="min-h-11 rounded-md bg-indigo-600 px-3 py-2 text-sm font-medium text-white"
                >
                  {t('liveQuiz.kit.saveTitle')}
                </button>
              </form>
            ) : (
              <>
                <h2 className="text-xl font-semibold text-slate-900 dark:text-neutral-100">
                  {kit.title}
                </h2>
                {canEdit ? (
                  <button
                    type="button"
                    onClick={() => setRenaming(true)}
                    className="min-h-11 rounded-md px-3 py-2 text-sm text-indigo-600 hover:underline dark:text-indigo-400"
                  >
                    {t('liveQuiz.kit.rename')}
                  </button>
                ) : null}
              </>
            )}
            {canEdit ? (
              <div className="ms-auto flex flex-wrap gap-2">
                {bankEnabled ? (
                  <button
                    type="button"
                    onClick={() => setImportOpen(true)}
                    className="inline-flex min-h-11 items-center gap-1.5 rounded-md border border-slate-300 px-3 py-2 text-sm dark:border-neutral-600"
                  >
                    <Upload className="h-4 w-4" aria-hidden />
                    {t('liveQuiz.editor.importBank')}
                  </button>
                ) : null}
                <button
                  type="button"
                  onClick={() => {
                    setAiLikeQuestionId(null)
                    setAiOpen(true)
                  }}
                  title={
                    !ffIqAiGeneration || !aiConfigured
                      ? t('liveQuiz.ai.unavailable.short')
                      : undefined
                  }
                  className="inline-flex min-h-11 items-center gap-1.5 rounded-md border border-slate-300 px-3 py-2 text-sm dark:border-neutral-600"
                >
                  <Sparkles className="h-4 w-4" aria-hidden />
                  {t('liveQuiz.ai.open')}
                </button>
                <button
                  type="button"
                  onClick={() => {
                    void handleValidate()
                  }}
                  className="inline-flex min-h-11 items-center gap-1.5 rounded-md border border-slate-300 px-3 py-2 text-sm dark:border-neutral-600"
                >
                  <CheckCircle2 className="h-4 w-4" aria-hidden />
                  {t('liveQuiz.editor.checkKit')}
                </button>
                {ffIqLiveHosting ? (
                  <button
                    type="button"
                    disabled={hosting}
                    onClick={() => {
                      void openHostDialog()
                    }}
                    className="inline-flex min-h-11 items-center gap-1.5 rounded-md bg-indigo-600 px-3 py-2 text-sm font-medium text-white disabled:opacity-50"
                  >
                    {t('liveQuiz.host.startFromKit')}
                  </button>
                ) : null}
              </div>
            ) : null}
          </div>

          {issues.length > 0 ? (
            <div
              className="rounded-md border border-amber-300 bg-amber-50 p-3 text-sm dark:border-amber-700 dark:bg-amber-950/30"
              role="status"
            >
              <p className="mb-2 font-medium text-amber-900 dark:text-amber-200">
                {t('liveQuiz.validate.issuesHeading')}
              </p>
              <ul className="list-disc space-y-1 ps-5 text-amber-800 dark:text-amber-300">
                {issues.map((issue, i) => (
                  <li key={`${issue.questionId}-${issue.code}-${i}`}>
                    <button
                      type="button"
                      className="text-start underline"
                      onClick={() => {
                        if (issue.questionId) setSelectedId(issue.questionId)
                      }}
                    >
                      {issue.message}
                    </button>
                  </li>
                ))}
              </ul>
            </div>
          ) : null}

          <div className="grid gap-4 lg:grid-cols-[16rem_minmax(0,1fr)_14rem]">
            {/* Left rail */}
            <aside className="space-y-2 rounded-lg border border-slate-200 p-3 dark:border-neutral-700">
              <div className="flex items-center justify-between gap-2">
                <h3 className="text-sm font-semibold text-slate-800 dark:text-neutral-100">
                  {t('liveQuiz.editor.questionList')}
                </h3>
                {canEdit ? (
                  <button
                    type="button"
                    onClick={() => {
                      void handleAdd()
                    }}
                    className="inline-flex min-h-11 items-center gap-1 rounded-md px-2 text-sm text-indigo-600 dark:text-indigo-400"
                    aria-label={t('liveQuiz.editor.addQuestion')}
                  >
                    <Plus className="h-4 w-4" aria-hidden />
                  </button>
                ) : null}
              </div>
              {questions.length === 0 ? (
                <div className="rounded-md border border-dashed border-slate-300 p-4 text-center text-sm text-slate-500 dark:border-neutral-600">
                  <p>{t('liveQuiz.editor.empty')}</p>
                  {canEdit ? (
                    <button
                      type="button"
                      onClick={() => {
                        void handleAdd()
                      }}
                      className="mt-3 min-h-11 rounded-md bg-indigo-600 px-3 py-2 text-sm font-medium text-white"
                    >
                      {t('liveQuiz.editor.addFirst')}
                    </button>
                  ) : null}
                </div>
              ) : (
                <ul className="space-y-1">
                  {questions.map((q, index) => (
                    <li key={q.id}>
                      <div
                        className={`flex items-start gap-1 rounded-md ${
                          selectedId === q.id
                            ? 'bg-indigo-50 dark:bg-indigo-950/40'
                            : 'hover:bg-slate-50 dark:hover:bg-neutral-800'
                        }`}
                      >
                        <button
                          type="button"
                          onClick={() => setSelectedId(q.id)}
                          className="min-h-11 min-w-0 flex-1 px-2 py-2 text-start text-sm"
                        >
                          <span className="block font-medium text-slate-800 dark:text-neutral-100">
                            {index + 1}. {q.prompt.trim() || t('liveQuiz.editor.untitled')}
                          </span>
                          <span className="text-xs text-slate-500">
                            {t(`liveQuiz.qtype.${q.questionType}`)} · {q.timeLimitSeconds}s
                            {q.source === 'ai_generated' ? ` · ${t('liveQuiz.ai.badge')}` : ''}
                            {q.needsReview ? ` · ${t('liveQuiz.ai.needsReview')}` : ''}
                          </span>
                        </button>
                        {canEdit ? (
                          <div className="flex flex-col py-1">
                            <button
                              type="button"
                              className="min-h-8 px-1 disabled:opacity-30"
                              disabled={index === 0}
                              onClick={() => {
                                void moveQuestion(q.id, -1)
                              }}
                              aria-label={t('liveQuiz.editor.moveUp')}
                            >
                              <ArrowUp className="h-3.5 w-3.5" aria-hidden />
                            </button>
                            <button
                              type="button"
                              className="min-h-8 px-1 disabled:opacity-30"
                              disabled={index === questions.length - 1}
                              onClick={() => {
                                void moveQuestion(q.id, 1)
                              }}
                              aria-label={t('liveQuiz.editor.moveDown')}
                            >
                              <ArrowDown className="h-3.5 w-3.5" aria-hidden />
                            </button>
                          </div>
                        ) : null}
                      </div>
                    </li>
                  ))}
                </ul>
              )}
            </aside>

            {/* Center editor */}
            <section className="space-y-4 rounded-lg border border-slate-200 p-4 dark:border-neutral-700">
              {!selected ? (
                <p className="text-sm text-slate-500">{t('liveQuiz.editor.selectPrompt')}</p>
              ) : (
                <>
                  <QuestionTypePicker
                    value={qType}
                    onChange={setQType}
                    disabled={!canEdit}
                  />
                  <label className="block text-sm">
                    <span className="mb-1 block font-medium text-slate-700 dark:text-neutral-200">
                      {t('liveQuiz.editor.prompt')}
                    </span>
                    <textarea
                      value={prompt}
                      disabled={!canEdit}
                      onChange={(e) => setPrompt(e.target.value)}
                      rows={3}
                      maxLength={4000}
                      className="w-full rounded-md border border-slate-300 px-3 py-2 dark:border-neutral-600 dark:bg-neutral-800 dark:text-neutral-100"
                    />
                  </label>
                  <MediaAttach
                    mediaRef={mediaRef}
                    mediaAlt={mediaAlt}
                    disabled={!canEdit}
                    onChange={({ mediaRef: r, mediaAlt: a }) => {
                      setMediaRef(r)
                      setMediaAlt(a)
                    }}
                  />
                  {renderTypeEditor()}
                  {canEdit && selected ? (
                    <div className="flex flex-wrap gap-2 border-t border-slate-200 pt-3 dark:border-neutral-700">
                      <button
                        type="button"
                        onClick={() => {
                          void handleDuplicate(selected.id)
                        }}
                        className="inline-flex min-h-11 items-center gap-1.5 rounded-md border border-slate-300 px-3 py-2 text-sm dark:border-neutral-600"
                      >
                        <Copy className="h-4 w-4" aria-hidden />
                        {t('liveQuiz.editor.duplicate')}
                      </button>
                      {ffIqAiGeneration && aiConfigured ? (
                        <>
                          <button
                            type="button"
                            disabled={regenBusy}
                            onClick={() => {
                              void (async () => {
                                setRegenBusy(true)
                                try {
                                  await regenerateQuizQuestion(courseCode, kitId, selected.id)
                                  // Poll briefly then reload
                                  await new Promise((r) => setTimeout(r, 2500))
                                  await load()
                                } catch (err) {
                                  toastMutationError(
                                    err instanceof Error ? err.message : String(err),
                                  )
                                } finally {
                                  setRegenBusy(false)
                                }
                              })()
                            }}
                            className="inline-flex min-h-11 items-center gap-1.5 rounded-md border border-slate-300 px-3 py-2 text-sm dark:border-neutral-600"
                          >
                            <Sparkles className="h-4 w-4" aria-hidden />
                            {t('liveQuiz.ai.regenerate')}
                          </button>
                          <button
                            type="button"
                            onClick={() => {
                              setAiLikeQuestionId(selected.id)
                              setAiOpen(true)
                            }}
                            className="inline-flex min-h-11 items-center gap-1.5 rounded-md border border-slate-300 px-3 py-2 text-sm dark:border-neutral-600"
                          >
                            {t('liveQuiz.ai.moreLikeThis')}
                          </button>
                        </>
                      ) : null}
                      <button
                        type="button"
                        onClick={() => {
                          void handleDelete(selected.id)
                        }}
                        className="inline-flex min-h-11 items-center gap-1.5 rounded-md px-3 py-2 text-sm text-red-600"
                      >
                        <Trash2 className="h-4 w-4" aria-hidden />
                        {t('liveQuiz.editor.delete')}
                      </button>
                    </div>
                  ) : null}
                </>
              )}
            </section>

            {/* Settings rail */}
            <aside className="space-y-3 rounded-lg border border-slate-200 p-3 dark:border-neutral-700 lg:sticky lg:top-4 lg:self-start">
              <h3 className="text-sm font-semibold text-slate-800 dark:text-neutral-100">
                {t('liveQuiz.editor.settings')}
              </h3>
              <label className="block text-sm">
                <span className="mb-1 block text-slate-600 dark:text-neutral-300">
                  {t('liveQuiz.editor.timer')}
                </span>
                <input
                  type="number"
                  min={5}
                  max={240}
                  disabled={!canEdit || !selected}
                  value={timer}
                  onChange={(e) => setTimer(Number(e.target.value))}
                  className="w-full min-h-11 rounded-md border border-slate-300 px-3 py-2 dark:border-neutral-600 dark:bg-neutral-800 dark:text-neutral-100"
                />
              </label>
              <label className="block text-sm">
                <span className="mb-1 block text-slate-600 dark:text-neutral-300">
                  {t('liveQuiz.editor.pointsStyle')}
                </span>
                <select
                  disabled={!canEdit || !selected}
                  value={pointsStyle}
                  onChange={(e) => setPointsStyle(e.target.value as LiveQuizPointsStyle)}
                  className="w-full min-h-11 rounded-md border border-slate-300 px-3 py-2 dark:border-neutral-600 dark:bg-neutral-800 dark:text-neutral-100"
                >
                  <option value="standard">{t('liveQuiz.editor.points.standard')}</option>
                  <option value="double">{t('liveQuiz.editor.points.double')}</option>
                  <option value="no_points">{t('liveQuiz.editor.points.none')}</option>
                </select>
              </label>
              <label className="flex min-h-11 items-center gap-2 text-sm text-slate-700 dark:text-neutral-200">
                <input
                  type="checkbox"
                  disabled={!canEdit || !selected}
                  checked={shuffle}
                  onChange={(e) => setShuffle(e.target.checked)}
                />
                {t('liveQuiz.editor.answerShuffle')}
              </label>
              <label className="block text-sm">
                <span className="mb-1 block text-slate-600 dark:text-neutral-300">
                  {t('liveQuiz.editor.explanation')}
                </span>
                <textarea
                  disabled={!canEdit || !selected}
                  value={explanation}
                  onChange={(e) => setExplanation(e.target.value)}
                  rows={3}
                  className="w-full rounded-md border border-slate-300 px-3 py-2 dark:border-neutral-600 dark:bg-neutral-800 dark:text-neutral-100"
                />
              </label>
            </aside>
          </div>
        </div>
      ) : null}

      <BankImportDrawer
        open={importOpen}
        courseCode={courseCode}
        kitId={kitId}
        onClose={() => setImportOpen(false)}
        onImport={async (ids) => {
          const created = await importBankQuestions(courseCode, kitId, ids)
          setQuestions((prev) => [...prev, ...created])
          if (created[0]) setSelectedId(created[0].id)
          setKit((k) => (k ? { ...k, questionCount: k.questionCount + created.length } : k))
        }}
      />

      <GenerateWithAiPanel
        open={aiOpen}
        courseCode={courseCode}
        kitId={kitId}
        likeQuestionId={aiLikeQuestionId}
        onClose={() => {
          setAiOpen(false)
          setAiLikeQuestionId(null)
        }}
        onComplete={() => {
          void load()
        }}
      />

      {hostDialogOpen ? (
        <div
          className="fixed inset-0 z-[400] flex items-center justify-center p-4"
          role="presentation"
        >
          <button
            type="button"
            aria-label={t('dialogs.close')}
            disabled={hosting}
            className="lex-btn-static absolute inset-0 cursor-default border-0 bg-black/45 p-0 disabled:cursor-not-allowed"
            onClick={() => {
              if (!hosting) setHostDialogOpen(false)
            }}
          />
          <div
            role="dialog"
            aria-modal="true"
            aria-labelledby={hostDialogTitleId}
            className="relative z-10 flex max-h-[min(90vh,720px)] w-full max-w-lg flex-col overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-xl dark:border-neutral-700 dark:bg-neutral-900"
          >
            <div className="shrink-0 border-b border-slate-100 px-5 py-4 dark:border-neutral-800">
              <h2
                id={hostDialogTitleId}
                className="text-lg font-semibold text-slate-950 dark:text-neutral-100"
              >
                {t('liveQuiz.host.startDialogTitle')}
              </h2>
            </div>
            <div className="min-h-0 flex-1 space-y-5 overflow-y-auto px-5 py-4">
              <ModePicker
                value={modeOpts}
                onChange={setModeOpts}
                teamEnabled={ffIqTeamMode}
                pacedEnabled={ffIqStudentPaced}
              />
              <ScoringProfilePicker value={scoringOpts} onChange={setScoringOpts} />
            </div>
            <div className="flex shrink-0 flex-wrap justify-end gap-2 border-t border-slate-100 bg-slate-50/80 px-5 py-4 dark:border-neutral-800 dark:bg-neutral-900/80">
              <button
                ref={hostCancelRef}
                type="button"
                className="rounded-xl border border-slate-200 bg-white px-4 py-2 text-sm font-semibold text-slate-800 shadow-sm motion-safe:transition-transform motion-safe:duration-150 motion-safe:ease-out motion-safe:active:scale-[0.96] hover:bg-slate-50 disabled:opacity-60 dark:border-neutral-600 dark:bg-neutral-900 dark:text-neutral-100 dark:hover:bg-neutral-800"
                onClick={() => setHostDialogOpen(false)}
                disabled={hosting}
              >
                {t('liveQuiz.host.cancelStart')}
              </button>
              <button
                type="button"
                className="rounded-xl bg-indigo-600 px-4 py-2 text-sm font-semibold text-white shadow-sm motion-safe:transition-transform motion-safe:duration-150 motion-safe:ease-out motion-safe:active:scale-[0.96] hover:bg-indigo-500 disabled:cursor-not-allowed disabled:opacity-50"
                disabled={hosting}
                onClick={() => {
                  void handleHost()
                }}
              >
                {hosting ? t('dialogs.working') : t('liveQuiz.host.confirmStart')}
              </button>
            </div>
          </div>
        </div>
      ) : null}
    </LmsPage>
  )
}
