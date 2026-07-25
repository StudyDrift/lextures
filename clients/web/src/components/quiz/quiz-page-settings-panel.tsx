import { useEffect, useMemo, useState, type ReactNode } from 'react'
import { ChevronDown, Search } from 'lucide-react'
import {
  fetchCourseOutcomes,
  type AdaptiveDifficulty,
  type AdaptiveStopRule,
  type GradeAttemptPolicy,
  type LateSubmissionPolicy,
  type LockdownMode,
  type QuizAdvancedSettings,
  type ReviewVisibility,
  type ReviewWhen,
  type ShowScoreTiming,
} from '../../lib/courses-api'
import {
  getMatchingSettingIds,
  sectionHasMatchingSettings,
  type QuizSettingsSectionId,
} from '../../lib/settings-registry'
import { SettingRow } from '../settings-panel/setting-row'
import { SettingsPanelProvider, useSettingsPanelContext } from '../settings-panel/settings-panel-context'
import {
  PinnedSectionHint,
  PinnedSettingsGroup,
} from '../settings-panel/pinned-settings-group'
import { usePinnedSettings } from '../settings-panel/use-pinned-settings'
import { ModuleItemOutcomesMappingAccordion } from '../outcomes/module-item-outcomes-mapping-accordion'
import { AssignToEditor } from '../assignment/assign-to-editor'

export type QuizPageSettingsPanelProps = {
  disabled?: boolean
  dueLocal: string
  onDueLocalChange: (value: string) => void
  availableFromLocal: string
  onAvailableFromLocalChange: (value: string) => void
  availableUntilLocal: string
  onAvailableUntilLocalChange: (value: string) => void
  unlimitedAttempts: boolean
  onUnlimitedAttemptsChange: (value: boolean) => void
  oneQuestionAtATime: boolean
  onOneQuestionAtATimeChange: (value: boolean) => void
  pointsWorth: number | null
  onPointsWorthChange: (value: number | null) => void
  /** Saved assignment groups (with server ids). */
  gradingGroups: { id: string; name: string }[]
  assignmentGroupId: string | null
  onAssignmentGroupChange: (groupId: string | null) => void
  /** When true, only the assignment-group control is locked (e.g. patch in flight). */
  assignmentGroupSelectDisabled?: boolean
  advanced: QuizAdvancedSettings
  onAdvancedChange: (next: QuizAdvancedSettings) => void
  showAdaptiveSection: boolean
  /** When set, settings include outcome links for this quiz item. */
  courseCode?: string
  quizItemId?: string
  quizOutcomesQuestions?: { id: string; prompt: string }[]
  /** Course-level feature; when false, lockdown controls are hidden. */
  lockdownDeliveryEnabled?: boolean
  lockdownMode?: LockdownMode
  onLockdownModeChange?: (mode: LockdownMode) => void
  focusLossThreshold?: number | null
  onFocusLossThresholdChange?: (value: number | null) => void
  /** Plan 3.9 — per-item drop policy when the course uses assignment-group drop rules. */
  neverDrop?: boolean
  onNeverDropChange?: (value: boolean) => void
  replaceWithFinal?: boolean
  onReplaceWithFinalChange?: (value: boolean) => void
}

const inputClass =
  'w-full rounded-lg border border-slate-200 bg-white px-2 py-1.5 text-sm text-slate-900 focus:border-indigo-400 focus:outline-none focus:ring-1 focus:ring-indigo-400 disabled:opacity-60 dark:border-neutral-600 dark:bg-neutral-950 dark:text-neutral-100 dark:focus:border-indigo-500 dark:focus:ring-indigo-500'

function SettingsAccordion({
  title,
  badge,
  forceOpen,
  sectionId,
  children,
}: {
  title: string
  badge?: number
  /** When true (e.g. during search), keep the section expanded. */
  forceOpen?: boolean
  /** Registry section id — used for pin count hint (FR-7). */
  sectionId?: QuizSettingsSectionId
  children: ReactNode
}) {
  const panel = useSettingsPanelContext()
  const pinCount = sectionId ? panel.pinnedCountForSection(sectionId) : 0
  if (sectionId && !panel.sectionHasVisibleContent(sectionId)) {
    // During search, hide sections whose only matches are pinned (AC-14).
    // When not searching, still show if any pins originated here (FR-7).
    if (panel.searching || pinCount === 0) return null
  }
  return (
    <details
      key={forceOpen ? 'forced-open' : 'manual'}
      className="group border-b border-slate-100 last:border-b-0 dark:border-neutral-800/80"
      open={forceOpen || undefined}
    >
      <summary className="flex cursor-pointer list-none items-center justify-between gap-2 px-3 py-2 text-[13px] font-medium text-slate-600 outline-none motion-safe:transition-colors hover:bg-slate-50/80 hover:text-slate-800 dark:text-neutral-400 dark:hover:bg-neutral-800/30 dark:hover:text-neutral-200 [&::-webkit-details-marker]:hidden">
        <span className="inline-flex min-w-0 items-center gap-2">
          <span className="truncate">{title}</span>
          {badge != null && badge > 0 ? (
            <span className="inline-flex h-5 min-w-5 shrink-0 items-center justify-center rounded-full bg-indigo-50 px-1.5 text-[11px] font-semibold text-indigo-700 dark:bg-indigo-950/60 dark:text-indigo-300">
              {badge}
            </span>
          ) : null}
        </span>
        <ChevronDown
          className="h-3.5 w-3.5 shrink-0 text-slate-400/80 motion-safe:transition-transform motion-safe:duration-200 group-open:rotate-180 dark:text-neutral-500"
          aria-hidden
        />
      </summary>
      <div className="px-3 pb-3 pt-0.5">
        <PinnedSectionHint count={pinCount} />
        {children}
      </div>
    </details>
  )
}

function QuizPinnedGroup({ pins }: { pins: ReturnType<typeof usePinnedSettings> }) {
  const { getRegisteredIds, registeredVersion, matches } = useSettingsPanelContext()
  const visiblePinned = useMemo(() => {
    void registeredVersion
    const reg = getRegisteredIds()
    return pins.resolved.filter((d) => reg.has(d.id) && matches(d.id))
  }, [pins.resolved, getRegisteredIds, registeredVersion, matches])
  return <PinnedSettingsGroup pins={pins} visiblePinned={visiblePinned} />
}

function SettingsAccordionGroup({ children }: { children: ReactNode }) {
  return (
    <div className="overflow-hidden rounded-lg border border-slate-200/70 bg-white dark:border-neutral-700/50 dark:bg-neutral-950/20">
      {children}
    </div>
  )
}

function ToggleRow({
  id,
  label,
  description,
  checked,
  onChange,
  disabled,
}: {
  id: string
  label: string
  description: string
  checked: boolean
  onChange: (next: boolean) => void
  disabled?: boolean
}) {
  return (
    <div className="flex items-start justify-between gap-3 py-2">
      <div className="min-w-0">
        <label htmlFor={id} className="text-[13px] font-medium text-slate-700 dark:text-neutral-200">
          {label}
        </label>
        <p className="mt-0.5 text-[11px] leading-relaxed text-slate-400 dark:text-neutral-500">{description}</p>
      </div>
      <button
        id={id}
        type="button"
        role="switch"
        aria-checked={checked}
        disabled={disabled}
        onClick={() => onChange(!checked)}
        className={`relative mt-0.5 h-5 w-9 shrink-0 rounded-full transition-colors disabled:cursor-not-allowed disabled:opacity-50 ${
          checked ? 'bg-indigo-500' : 'bg-slate-300 dark:bg-neutral-600'
        }`}
      >
        <span
          className={`absolute top-0.5 h-4 w-4 rounded-full bg-white transition-colors ${
            checked ? 'start-4.5' : 'start-0.5'
          }`}
        />
      </button>
    </div>
  )
}

function Field({
  label,
  htmlFor,
  hint,
  children,
}: {
  label: string
  htmlFor: string
  hint?: string
  children: ReactNode
}) {
  return (
    <div className="space-y-1">
      <label className="block text-xs font-medium text-slate-500 dark:text-neutral-400" htmlFor={htmlFor}>
        {label}
      </label>
      {children}
      {hint ? <p className="text-[11px] leading-snug text-slate-400 dark:text-neutral-500">{hint}</p> : null}
    </div>
  )
}

export function QuizPageSettingsPanel({
  disabled,
  dueLocal,
  onDueLocalChange,
  availableFromLocal,
  onAvailableFromLocalChange,
  availableUntilLocal,
  onAvailableUntilLocalChange,
  unlimitedAttempts,
  onUnlimitedAttemptsChange,
  oneQuestionAtATime,
  onOneQuestionAtATimeChange,
  pointsWorth,
  onPointsWorthChange,
  gradingGroups,
  assignmentGroupId,
  onAssignmentGroupChange,
  assignmentGroupSelectDisabled,
  advanced,
  onAdvancedChange,
  showAdaptiveSection,
  courseCode,
  quizItemId,
  quizOutcomesQuestions,
  lockdownDeliveryEnabled,
  lockdownMode = 'standard',
  onLockdownModeChange,
  focusLossThreshold = null,
  onFocusLossThresholdChange,
  neverDrop = false,
  onNeverDropChange,
  replaceWithFinal = false,
  onReplaceWithFinalChange,
}: QuizPageSettingsPanelProps) {
  const [outcomesLinkCount, setOutcomesLinkCount] = useState(0)
  const [hasCourseOutcomes, setHasCourseOutcomes] = useState(false)
  const [settingsQuery, setSettingsQuery] = useState('')
  const searching = settingsQuery.trim().length > 0
  const pins = usePinnedSettings('quiz')

  // Briefly force-open home section after unpin so the control is findable.
  useEffect(() => {
    if (!pins.forceOpenSection) return
    const t = window.setTimeout(() => pins.clearForceOpenSection(), 2500)
    return () => window.clearTimeout(t)
  }, [pins.forceOpenSection, pins.clearForceOpenSection])

  const show = (id: QuizSettingsSectionId) =>
    sectionHasMatchingSettings('quiz', id, settingsQuery)

  const sectionForceOpen = (id: QuizSettingsSectionId) =>
    searching || pins.forceOpenSection === id

  useEffect(() => {
    if (!courseCode) {
      setHasCourseOutcomes(false)
      return
    }
    let cancelled = false
    void (async () => {
      try {
        const data = await fetchCourseOutcomes(courseCode)
        if (!cancelled) setHasCourseOutcomes(data.outcomes.length > 0)
      } catch {
        if (!cancelled) setHasCourseOutcomes(false)
      }
    })()
    return () => {
      cancelled = true
    }
  }, [courseCode])

  function patch(p: Partial<QuizAdvancedSettings>) {
    onAdvancedChange({ ...advanced, ...p })
  }

  const candidateSections: QuizSettingsSectionId[] = [
    'scheduling',
    'attempts-grading',
    'grading',
    'time-limits',
    'scores-review',
    'presentation',
    ...(courseCode && quizItemId && hasCourseOutcomes ? (['outcomes'] as QuizSettingsSectionId[]) : []),
    ...(courseCode && quizItemId ? (['assign-to'] as QuizSettingsSectionId[]) : []),
    'access',
    ...(showAdaptiveSection ? (['adaptive-ai'] as QuizSettingsSectionId[]) : []),
  ]
  const visibleSectionCount = candidateSections.filter((id) => show(id)).length
  const hasPinnedSearchHit = useMemo(() => {
    if (!searching || !pins.enabled) return false
    const matchSet = getMatchingSettingIds('quiz', settingsQuery)
    return pins.resolved.some((d) => matchSet.has(d.id))
  }, [searching, pins.enabled, pins.resolved, settingsQuery])

  return (
    <SettingsPanelProvider surface="quiz" query={settingsQuery} pins={pins}>
      <div className="space-y-3">
        <p className="text-xs leading-relaxed text-slate-500 dark:text-neutral-400">
          Save the page from the toolbar to apply changes.
        </p>

        <div className="relative">
          <Search
            className="pointer-events-none absolute start-2.5 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-slate-400 dark:text-neutral-500"
            aria-hidden
          />
          <input
            id="quiz-settings-search"
            type="search"
            value={settingsQuery}
            onChange={(e) => setSettingsQuery(e.target.value)}
            placeholder="Search settings…"
            aria-label="Search quiz settings"
            className={`${inputClass} ps-8`}
          />
        </div>

        <QuizPinnedGroup pins={pins} />

        {searching && visibleSectionCount === 0 && !hasPinnedSearchHit ? (
          <p className="rounded-lg border border-slate-200/70 px-3 py-6 text-center text-sm text-slate-400 dark:border-neutral-700/50 dark:text-neutral-500">
            No settings match &ldquo;{settingsQuery.trim()}&rdquo;
          </p>
        ) : (
          <SettingsAccordionGroup>
            {show('scheduling') ? (
              <SettingsAccordion title="Scheduling" sectionId="scheduling" forceOpen={sectionForceOpen('scheduling')}>
                <div className="space-y-3 pt-1">
                  <SettingRow settingId="quiz.scheduling.due-date">
                    <Field label="Due date" htmlFor="quiz-settings-due" hint="Optional. Cleared if empty.">
                      <input
                        id="quiz-settings-due"
                        type="datetime-local"
                        value={dueLocal}
                        onChange={(e) => onDueLocalChange(e.target.value)}
                        disabled={disabled}
                        className={inputClass}
                      />
                    </Field>
                  </SettingRow>
                  <SettingRow settingId="quiz.scheduling.visible-from">
                    <Field
                      label="Visibility start"
                      htmlFor="quiz-settings-visible-from"
                      hint="Learners cannot open the quiz before this time."
                    >
                      <input
                        id="quiz-settings-visible-from"
                        type="datetime-local"
                        value={availableFromLocal}
                        onChange={(e) => onAvailableFromLocalChange(e.target.value)}
                        disabled={disabled}
                        className={inputClass}
                      />
                    </Field>
                  </SettingRow>
                  <SettingRow settingId="quiz.scheduling.visible-until">
                    <Field
                      label="Visibility end"
                      htmlFor="quiz-settings-visible-until"
                      hint="After this time the quiz is no longer available."
                    >
                      <input
                        id="quiz-settings-visible-until"
                        type="datetime-local"
                        value={availableUntilLocal}
                        onChange={(e) => onAvailableUntilLocalChange(e.target.value)}
                        disabled={disabled}
                        className={inputClass}
                      />
                    </Field>
                  </SettingRow>
                </div>
              </SettingsAccordion>
            ) : null}

            {show('attempts-grading') ? (
              <SettingsAccordion title="Attempts & grading" sectionId="attempts-grading" forceOpen={sectionForceOpen('attempts-grading')}>
                <div className="divide-y divide-slate-100/90 dark:divide-neutral-800/80">
                  <SettingRow settingId="quiz.attempts-grading.unlimited-attempts">
                    <ToggleRow
                      id="quiz-settings-unlimited-attempts"
                      label="Unlimited attempts"
                      description="Allow learners to retake without an attempt limit."
                      checked={unlimitedAttempts}
                      onChange={onUnlimitedAttemptsChange}
                      disabled={disabled}
                    />
                  </SettingRow>
                  {!unlimitedAttempts ? (
                    <SettingRow settingId="quiz.attempts-grading.max-attempts">
                      <div className="py-3">
                        <Field label="Max attempts" htmlFor="quiz-max-attempts">
                          <input
                            id="quiz-max-attempts"
                            type="number"
                            min={1}
                            max={100}
                            value={advanced.maxAttempts}
                            onChange={(e) =>
                              patch({ maxAttempts: Math.min(100, Math.max(1, Number(e.target.value) || 1)) })
                            }
                            disabled={disabled}
                            className={`max-w-[8rem] ${inputClass}`}
                          />
                        </Field>
                      </div>
                    </SettingRow>
                  ) : null}
                  <SettingRow settingId="quiz.attempts-grading.grade-policy">
                    <div className="py-3">
                      <Field
                        label="Grade uses"
                        htmlFor="quiz-grade-policy"
                        hint="Which attempt counts when multiple tries are allowed."
                      >
                        <select
                          id="quiz-grade-policy"
                          value={advanced.gradeAttemptPolicy}
                          onChange={(e) => patch({ gradeAttemptPolicy: e.target.value as GradeAttemptPolicy })}
                          disabled={disabled}
                          className={inputClass}
                        >
                          <option value="latest">Latest attempt</option>
                          <option value="highest">Highest score</option>
                          <option value="first">First attempt</option>
                          <option value="average">Average of attempts</option>
                        </select>
                      </Field>
                    </div>
                  </SettingRow>
                  <SettingRow settingId="quiz.attempts-grading.passing-score">
                    <div className="py-3">
                      <Field label="Passing score (%)" htmlFor="quiz-passing" hint="Leave empty for no pass requirement.">
                        <input
                          id="quiz-passing"
                          type="number"
                          min={0}
                          max={100}
                          placeholder="None"
                          value={advanced.passingScorePercent ?? ''}
                          onChange={(e) => {
                            const v = e.target.value
                            patch({ passingScorePercent: v === '' ? null : Math.min(100, Math.max(0, Number(v))) })
                          }}
                          disabled={disabled}
                          className={`max-w-[8rem] ${inputClass}`}
                        />
                      </Field>
                    </div>
                  </SettingRow>
                  <SettingRow settingId="quiz.attempts-grading.points-worth">
                    <div className="py-3">
                      <Field
                        label="Points worth"
                        htmlFor="quiz-points-worth"
                        hint="How many points this quiz counts for. Leave empty if not set (use 0 for explicitly no points)."
                      >
                        <input
                          id="quiz-points-worth"
                          type="number"
                          min={0}
                          max={1000000}
                          placeholder="Not set"
                          value={pointsWorth ?? ''}
                          onChange={(e) => {
                            const v = e.target.value.trim()
                            if (v === '') {
                              onPointsWorthChange(null)
                              return
                            }
                            const n = Math.floor(Number(v))
                            if (!Number.isFinite(n)) return
                            onPointsWorthChange(Math.min(1_000_000, Math.max(0, n)))
                          }}
                          disabled={disabled}
                          className={`max-w-[10rem] ${inputClass}`}
                        />
                      </Field>
                    </div>
                  </SettingRow>
                  <SettingRow settingId="quiz.attempts-grading.late-policy">
                    <div className="py-3">
                      <Field label="Late submission (after due)" htmlFor="quiz-late-policy">
                        <select
                          id="quiz-late-policy"
                          value={advanced.lateSubmissionPolicy}
                          onChange={(e) => patch({ lateSubmissionPolicy: e.target.value as LateSubmissionPolicy })}
                          disabled={disabled}
                          className={inputClass}
                        >
                          <option value="allow">Allow (no block)</option>
                          <option value="penalty">Allow with penalty</option>
                          <option value="block">Block after due</option>
                        </select>
                      </Field>
                    </div>
                  </SettingRow>
                  {advanced.lateSubmissionPolicy === 'penalty' ? (
                    <SettingRow settingId="quiz.attempts-grading.late-penalty">
                      <div className="py-3">
                        <Field label="Late penalty (% of points)" htmlFor="quiz-late-penalty">
                          <input
                            id="quiz-late-penalty"
                            type="number"
                            min={0}
                            max={100}
                            value={advanced.latePenaltyPercent ?? ''}
                            onChange={(e) => {
                              const v = e.target.value
                              patch({ latePenaltyPercent: v === '' ? null : Math.min(100, Math.max(0, Number(v))) })
                            }}
                            disabled={disabled}
                            className={`max-w-[8rem] ${inputClass}`}
                          />
                        </Field>
                      </div>
                    </SettingRow>
                  ) : null}
                </div>
              </SettingsAccordion>
            ) : null}

            {show('grading') ? (
              <SettingsAccordion title="Grading" sectionId="grading" forceOpen={sectionForceOpen('grading')}>
                <div className="space-y-3 pt-1">
                  <SettingRow settingId="quiz.grading.assignment-group">
                    <Field
                      label="Assignment group"
                      htmlFor="quiz-assignment-group"
                      hint="Used with weighted assignment groups in course grading settings. Saves immediately when changed."
                    >
                      <select
                        id="quiz-assignment-group"
                        value={assignmentGroupId ?? ''}
                        onChange={(e) => {
                          const v = e.target.value
                          onAssignmentGroupChange(v === '' ? null : v)
                        }}
                        disabled={disabled || Boolean(assignmentGroupSelectDisabled)}
                        className={inputClass}
                      >
                        <option value="">— None —</option>
                        {gradingGroups.map((g) => (
                          <option key={g.id} value={g.id}>
                            {g.name}
                          </option>
                        ))}
                      </select>
                    </Field>
                  </SettingRow>
                  {gradingGroups.length === 0 ? (
                    <p className="text-[11px] leading-snug text-slate-400 dark:text-neutral-500">
                      Add groups under Course Settings → Assignment groups & weights.
                    </p>
                  ) : null}
                  {onNeverDropChange && onReplaceWithFinalChange ? (
                    <div className="mt-2 divide-y divide-slate-100/90 border-t border-slate-100/90 pt-2 dark:divide-neutral-800/80 dark:border-neutral-800/80">
                      <SettingRow settingId="quiz.grading.never-drop">
                        <ToggleRow
                          id="quiz-never-drop"
                          label="Never drop this score"
                          description="When the assignment group drops lowest or highest scores, this quiz is always kept in the average."
                          checked={neverDrop}
                          onChange={(next) => {
                            onNeverDropChange(next)
                            if (!next && replaceWithFinal) onReplaceWithFinalChange(false)
                          }}
                          disabled={disabled}
                        />
                      </SettingRow>
                      <SettingRow settingId="quiz.grading.replace-with-final">
                        <ToggleRow
                          id="quiz-replace-with-final"
                          label="Use as final for replace-lowest"
                          description="If the group uses “replace lowest with final,” this score is the replacement when it beats the student’s lowest eligible item."
                          checked={replaceWithFinal}
                          onChange={(next) => {
                            onReplaceWithFinalChange(next)
                            if (next) onNeverDropChange(true)
                          }}
                          disabled={disabled}
                        />
                      </SettingRow>
                    </div>
                  ) : null}
                </div>
              </SettingsAccordion>
            ) : null}

            {show('time-limits') ? (
              <SettingsAccordion title="Time limits" sectionId="time-limits" forceOpen={sectionForceOpen('time-limits')}>
                <div className="space-y-3 pt-1">
                  <SettingRow settingId="quiz.time-limits.total-minutes">
                    <Field label="Total time limit (minutes)" htmlFor="quiz-time-limit">
                      <input
                        id="quiz-time-limit"
                        type="number"
                        min={1}
                        max={10080}
                        placeholder="None"
                        value={advanced.timeLimitMinutes ?? ''}
                        onChange={(e) => {
                          const v = e.target.value
                          patch({ timeLimitMinutes: v === '' ? null : Math.min(10080, Math.max(1, Number(v))) })
                        }}
                        disabled={disabled}
                        className={`max-w-[8rem] ${inputClass}`}
                      />
                    </Field>
                  </SettingRow>
                  <SettingRow settingId="quiz.time-limits.pause-when-hidden">
                    <ToggleRow
                      id="quiz-timer-pause"
                      label="Pause timer when tab is hidden"
                      description="When a time limit is set, the countdown pauses if the learner switches away from the tab."
                      checked={advanced.timerPauseWhenTabHidden}
                      onChange={(next) => patch({ timerPauseWhenTabHidden: next })}
                      disabled={disabled}
                    />
                  </SettingRow>
                  <SettingRow settingId="quiz.time-limits.per-question-seconds">
                    <Field
                      label="Per-question time limit (seconds)"
                      htmlFor="quiz-per-q-time"
                      hint="Optional cap for each question in one-question-at-a-time mode."
                    >
                      <input
                        id="quiz-per-q-time"
                        type="number"
                        min={10}
                        max={86400}
                        placeholder="None"
                        value={advanced.perQuestionTimeLimitSeconds ?? ''}
                        onChange={(e) => {
                          const v = e.target.value
                          patch({
                            perQuestionTimeLimitSeconds: v === '' ? null : Math.min(86400, Math.max(10, Number(v))),
                          })
                        }}
                        disabled={disabled}
                        className={`max-w-[8rem] ${inputClass}`}
                      />
                    </Field>
                  </SettingRow>
                </div>
              </SettingsAccordion>
            ) : null}

            {show('scores-review') ? (
              <SettingsAccordion title="Scores & review" sectionId="scores-review" forceOpen={sectionForceOpen('scores-review')}>
                <div className="space-y-3 pt-1">
                  <SettingRow settingId="quiz.scores-review.show-score-timing">
                    <Field label="When to show score" htmlFor="quiz-show-score">
                      <select
                        id="quiz-show-score"
                        value={advanced.showScoreTiming}
                        onChange={(e) => patch({ showScoreTiming: e.target.value as ShowScoreTiming })}
                        disabled={disabled}
                        className={inputClass}
                      >
                        <option value="immediate">Immediately after submit</option>
                        <option value="after_due">After the due date</option>
                        <option value="manual">When released by instructor</option>
                      </select>
                    </Field>
                  </SettingRow>
                  <SettingRow settingId="quiz.scores-review.visibility">
                    <Field label="What learners can see" htmlFor="quiz-review-vis">
                      <select
                        id="quiz-review-vis"
                        value={advanced.reviewVisibility}
                        onChange={(e) => patch({ reviewVisibility: e.target.value as ReviewVisibility })}
                        disabled={disabled}
                        className={inputClass}
                      >
                        <option value="full">Full feedback (score, responses, correct answers)</option>
                        <option value="correct_answers">Correct answers only</option>
                        <option value="responses">Their responses only</option>
                        <option value="score_only">Score only</option>
                        <option value="none">Nothing</option>
                      </select>
                    </Field>
                  </SettingRow>
                  <SettingRow settingId="quiz.scores-review.when">
                    <Field label="When they can review" htmlFor="quiz-review-when">
                      <select
                        id="quiz-review-when"
                        value={advanced.reviewWhen}
                        onChange={(e) => patch({ reviewWhen: e.target.value as ReviewWhen })}
                        disabled={disabled}
                        className={inputClass}
                      >
                        <option value="after_submit">Right after submitting</option>
                        <option value="after_due">After the due date</option>
                        <option value="always">Anytime after availability</option>
                        <option value="never">Never</option>
                      </select>
                    </Field>
                  </SettingRow>
                </div>
              </SettingsAccordion>
            ) : null}

            {show('presentation') ? (
              <SettingsAccordion title="Presentation" sectionId="presentation" forceOpen={sectionForceOpen('presentation')}>
                <div className="divide-y divide-slate-100/90 dark:divide-neutral-800/80">
                  <SettingRow settingId="quiz.presentation.one-question-at-a-time">
                    <ToggleRow
                      id="quiz-settings-one-question"
                      label="One question at a time"
                      description="Show a single question per step instead of the full list."
                      checked={oneQuestionAtATime}
                      onChange={onOneQuestionAtATimeChange}
                      disabled={disabled}
                    />
                  </SettingRow>
                  <SettingRow settingId="quiz.presentation.shuffle-questions">
                    <ToggleRow
                      id="quiz-shuffle-q"
                      label="Shuffle question order"
                      description="Each learner sees questions in a random order (non-adaptive quizzes)."
                      checked={advanced.shuffleQuestions}
                      onChange={(next) => patch({ shuffleQuestions: next })}
                      disabled={disabled}
                    />
                  </SettingRow>
                  <SettingRow settingId="quiz.presentation.shuffle-choices">
                    <ToggleRow
                      id="quiz-shuffle-c"
                      label="Shuffle answer choices"
                      description="Randomize multiple-choice and true/false option order per question."
                      checked={advanced.shuffleChoices}
                      onChange={(next) => patch({ shuffleChoices: next })}
                      disabled={disabled}
                    />
                  </SettingRow>
                  <SettingRow settingId="quiz.presentation.back-navigation">
                    <ToggleRow
                      id="quiz-back-nav"
                      label="Allow back navigation"
                      description="Let learners move to previous questions when using one question at a time."
                      checked={advanced.allowBackNavigation}
                      onChange={(next) => patch({ allowBackNavigation: next })}
                      disabled={disabled}
                    />
                  </SettingRow>
                  {lockdownDeliveryEnabled ? (
                    <div className="space-y-3 border-t border-slate-100/90 py-3 dark:border-neutral-800/80">
                      <SettingRow settingId="quiz.presentation.lockdown-mode">
                        <Field
                          label="Lockdown delivery"
                          htmlFor="quiz-lockdown-mode"
                          hint={
                            showAdaptiveSection
                              ? 'Adaptive quizzes cannot use server-enforced lockdown. Turn off adaptive generation to enable these modes.'
                              : 'Kiosk mode requests full-screen, logs tab or window changes, and disables hints.'
                          }
                        >
                          <select
                            id="quiz-lockdown-mode"
                            value={lockdownMode}
                            onChange={(e) => onLockdownModeChange?.(e.target.value as LockdownMode)}
                            disabled={disabled || showAdaptiveSection}
                            className={inputClass}
                          >
                            <option value="standard">Standard</option>
                            <option value="one_at_a_time">One question at a time (server enforced)</option>
                            <option value="kiosk">Kiosk (fullscreen + focus logging)</option>
                          </select>
                        </Field>
                      </SettingRow>
                      {lockdownMode === 'kiosk' ? (
                        <SettingRow settingId="quiz.presentation.focus-loss-threshold">
                          <Field
                            label="Focus-loss flag threshold"
                            htmlFor="quiz-focus-threshold"
                            hint="Leave empty so attempts are never auto-flagged. When set, exceeding this many logged events marks the attempt for review on submit."
                          >
                            <input
                              id="quiz-focus-threshold"
                              type="number"
                              min={1}
                              max={99}
                              placeholder="No auto-flag"
                              value={focusLossThreshold ?? ''}
                              onChange={(e) => {
                                const v = e.target.value.trim()
                                if (!v) {
                                  onFocusLossThresholdChange?.(null)
                                  return
                                }
                                const n = Math.floor(Number(v))
                                if (!Number.isFinite(n)) return
                                onFocusLossThresholdChange?.(Math.min(99, Math.max(1, n)))
                              }}
                              disabled={disabled || showAdaptiveSection}
                              className={`max-w-[8rem] ${inputClass}`}
                            />
                          </Field>
                        </SettingRow>
                      ) : null}
                    </div>
                  ) : null}
                  <SettingRow settingId="quiz.presentation.random-pool-size">
                    <div className="py-3">
                      <Field
                        label="Random question pool size"
                        htmlFor="quiz-pool"
                        hint="If set, each attempt draws this many questions from the bank (non-adaptive)."
                      >
                        <input
                          id="quiz-pool"
                          type="number"
                          min={1}
                          max={300}
                          placeholder="All questions"
                          value={advanced.randomQuestionPoolCount ?? ''}
                          onChange={(e) => {
                            const v = e.target.value
                            patch({ randomQuestionPoolCount: v === '' ? null : Math.min(300, Math.max(1, Number(v))) })
                          }}
                          disabled={disabled}
                          className={`max-w-[8rem] ${inputClass}`}
                        />
                      </Field>
                    </div>
                  </SettingRow>
                </div>
              </SettingsAccordion>
            ) : null}

            {courseCode && quizItemId && hasCourseOutcomes && show('outcomes') ? (
              <SettingsAccordion title="Outcomes" sectionId="outcomes" badge={outcomesLinkCount} forceOpen={sectionForceOpen('outcomes')}>
                <SettingRow settingId="quiz.outcomes.mapping">
                  <ModuleItemOutcomesMappingAccordion
                    courseCode={courseCode}
                    itemId={quizItemId}
                    mode="quiz"
                    disabled={disabled}
                    quizQuestions={quizOutcomesQuestions ?? []}
                    onLinkCountChange={setOutcomesLinkCount}
                  />
                </SettingRow>
              </SettingsAccordion>
            ) : null}

            {courseCode && quizItemId && show('assign-to') ? (
              <SettingsAccordion title="Assign to" sectionId="assign-to" forceOpen={sectionForceOpen('assign-to')}>
                <SettingRow settingId="quiz.assign-to.editor">
                  <AssignToEditor courseCode={courseCode} itemId={quizItemId} disabled={disabled} />
                </SettingRow>
              </SettingsAccordion>
            ) : null}

            {show('access') ? (
              <SettingsAccordion title="Access" sectionId="access" forceOpen={sectionForceOpen('access')}>
                <div className="pt-1">
                  <SettingRow settingId="quiz.access.access-code">
                    <Field
                      label="Quiz access code"
                      htmlFor="quiz-access-code"
                      hint="Learners must enter this before starting. Leave empty for none."
                    >
                      <input
                        id="quiz-access-code"
                        type="password"
                        autoComplete="new-password"
                        value={advanced.quizAccessCode}
                        onChange={(e) => patch({ quizAccessCode: e.target.value })}
                        disabled={disabled}
                        placeholder="Optional"
                        className={inputClass}
                      />
                    </Field>
                  </SettingRow>
                </div>
              </SettingsAccordion>
            ) : null}

            {showAdaptiveSection && show('adaptive-ai') ? (
              <SettingsAccordion title="Adaptive AI" sectionId="adaptive-ai" forceOpen={sectionForceOpen('adaptive-ai')}>
                <div className="space-y-3 pt-1">
                  <SettingRow settingId="quiz.adaptive-ai.difficulty">
                    <Field label="Difficulty target" htmlFor="quiz-ad-diff">
                      <select
                        id="quiz-ad-diff"
                        value={advanced.adaptiveDifficulty}
                        onChange={(e) => patch({ adaptiveDifficulty: e.target.value as AdaptiveDifficulty })}
                        disabled={disabled}
                        className={inputClass}
                      >
                        <option value="introductory">Introductory</option>
                        <option value="standard">Standard</option>
                        <option value="challenging">Challenging</option>
                      </select>
                    </Field>
                  </SettingRow>
                  <SettingRow settingId="quiz.adaptive-ai.topic-balance">
                    <ToggleRow
                      id="quiz-ad-balance"
                      label="Balance topics across sources"
                      description="Try to cover reference materials evenly across questions."
                      checked={advanced.adaptiveTopicBalance}
                      onChange={(next) => patch({ adaptiveTopicBalance: next })}
                      disabled={disabled}
                    />
                  </SettingRow>
                  <SettingRow settingId="quiz.adaptive-ai.stop-rule">
                    <Field label="Stop rule" htmlFor="quiz-ad-stop">
                      <select
                        id="quiz-ad-stop"
                        value={advanced.adaptiveStopRule}
                        onChange={(e) => patch({ adaptiveStopRule: e.target.value as AdaptiveStopRule })}
                        disabled={disabled}
                        className={inputClass}
                      >
                        <option value="fixed_count">Fixed number of questions</option>
                        <option value="mastery_estimate">Adapt until mastery (within cap)</option>
                      </select>
                    </Field>
                  </SettingRow>
                </div>
              </SettingsAccordion>
            ) : null}
          </SettingsAccordionGroup>
        )}
      </div>
    </SettingsPanelProvider>
  )
}
