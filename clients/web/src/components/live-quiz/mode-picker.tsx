import { useTranslation } from 'react-i18next'
import type { ModeStartOptions } from './mode-start-options'

const MODES: ModeStartOptions['mode'][] = ['live_classic', 'team', 'student_paced']

type Props = {
  value: ModeStartOptions
  onChange: (next: ModeStartOptions) => void
  teamEnabled?: boolean
  pacedEnabled?: boolean
}

const fieldClass =
  'w-full rounded-xl border border-border-default bg-surface-raised px-3 py-2 text-sm text-fg-default outline-none focus:border-indigo-400 focus:ring-2 focus:ring-indigo-400/30 disabled:opacity-60 dark:border-border-default dark:bg-surface-base dark:text-fg-default'
const labelMuted = 'mb-1 block text-xs font-medium text-fg-muted'

export function ModePicker({ value, onChange, teamEnabled = true, pacedEnabled = true }: Props) {
  const { t } = useTranslation()
  const modes = MODES.filter((m) => {
    if (m === 'team') return teamEnabled
    if (m === 'student_paced') return pacedEnabled
    return true
  })

  return (
    <fieldset className="space-y-3">
      <legend className="text-sm font-semibold text-fg-default">
        {t('liveQuiz.mode.label')}
      </legend>
      <div className="flex flex-col gap-2" role="radiogroup" aria-label={t('liveQuiz.mode.label')}>
        {modes.map((id) => {
          const selected = value.mode === id
          return (
            <label
              key={id}
              className={
                selected
                  ? 'flex cursor-pointer items-start gap-3 rounded-xl border border-indigo-400 bg-indigo-50/60 px-3 py-2.5 dark:border-indigo-500 dark:bg-indigo-950/40'
                  : 'flex cursor-pointer items-start gap-3 rounded-xl border border-border-default bg-surface-raised px-3 py-2.5 hover:border-border-strong dark:bg-surface-base dark:hover:border-border-default'
              }
            >
              <input
                type="radio"
                name="liveQuizMode"
                checked={selected}
                onChange={() => onChange({ ...value, mode: id })}
                className="mt-1 h-4 w-4 border-border-strong text-accent-fg focus:ring-indigo-500"
              />
              <span>
                <span className="block text-sm font-medium text-fg-default">
                  {t(`liveQuiz.mode.${id}`)}
                </span>
                <span className="mt-0.5 block text-xs text-fg-muted">
                  {t(`liveQuiz.mode.${id}Hint`)}
                </span>
              </span>
            </label>
          )
        })}
      </div>

      {value.mode === 'team' ? (
        <div className="grid gap-3 sm:grid-cols-2">
          <label className="text-sm">
            <span className={labelMuted}>{t('liveQuiz.team.teamCount')}</span>
            <input
              type="number"
              min={2}
              max={20}
              value={value.teamConfig.teamCount ?? 4}
              onChange={(e) =>
                onChange({
                  ...value,
                  teamConfig: { ...value.teamConfig, teamCount: Number(e.target.value) || 4 },
                })
              }
              className={fieldClass}
            />
          </label>
          <label className="text-sm">
            <span className={labelMuted}>{t('liveQuiz.team.aggregate')}</span>
            <select
              value={value.teamConfig.aggregate ?? 'average'}
              onChange={(e) =>
                onChange({
                  ...value,
                  teamConfig: {
                    ...value.teamConfig,
                    aggregate: e.target.value as 'average' | 'sum',
                  },
                })
              }
              className={fieldClass}
            >
              <option value="average">{t('liveQuiz.team.aggregateAverage')}</option>
              <option value="sum">{t('liveQuiz.team.aggregateSum')}</option>
            </select>
          </label>
          <label className="text-sm sm:col-span-2">
            <span className={labelMuted}>{t('liveQuiz.team.answerRule')}</span>
            <select
              value={value.teamConfig.answerRule ?? 'each_member_answers'}
              onChange={(e) =>
                onChange({
                  ...value,
                  teamConfig: {
                    ...value.teamConfig,
                    answerRule: e.target.value as TeamAnswerRule,
                  },
                })
              }
              className={fieldClass}
            >
              <option value="each_member_answers">{t('liveQuiz.team.eachMember')}</option>
              <option value="one_device_per_team">{t('liveQuiz.team.oneDevice')}</option>
            </select>
          </label>
        </div>
      ) : null}

      {value.mode === 'student_paced' ? (
        <div className="grid gap-3 sm:grid-cols-2">
          <label className="flex items-center gap-2 text-sm text-fg-default">
            <input
              type="checkbox"
              checked={value.pacedConfig.shuffle ?? true}
              onChange={(e) =>
                onChange({
                  ...value,
                  pacedConfig: { ...value.pacedConfig, shuffle: e.target.checked },
                })
              }
              className="h-4 w-4 rounded border-border-strong text-accent-fg focus:ring-indigo-500"
            />
            {t('liveQuiz.paced.shuffle')}
          </label>
          <label className="flex items-center gap-2 text-sm text-fg-default">
            <input
              type="checkbox"
              checked={value.pacedConfig.perQuestionTimers ?? true}
              onChange={(e) =>
                onChange({
                  ...value,
                  pacedConfig: { ...value.pacedConfig, perQuestionTimers: e.target.checked },
                })
              }
              className="h-4 w-4 rounded border-border-strong text-accent-fg focus:ring-indigo-500"
            />
            {t('liveQuiz.paced.perQuestionTimers')}
          </label>
          <label className="text-sm sm:col-span-2">
            <span className={labelMuted}>{t('liveQuiz.paced.timeBudget')}</span>
            <input
              type="number"
              min={0}
              value={value.pacedConfig.timeBudgetSeconds ?? 0}
              onChange={(e) =>
                onChange({
                  ...value,
                  pacedConfig: {
                    ...value.pacedConfig,
                    timeBudgetSeconds: Number(e.target.value) || 0,
                  },
                })
              }
              className={fieldClass}
            />
          </label>
        </div>
      ) : null}
    </fieldset>
  )
}

type TeamAnswerRule = 'each_member_answers' | 'one_device_per_team'
