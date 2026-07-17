import { useTranslation } from 'react-i18next'
import type { ModeStartOptions } from './mode-start-options'

const MODES: ModeStartOptions['mode'][] = ['live_classic', 'team', 'student_paced']

type Props = {
  value: ModeStartOptions
  onChange: (next: ModeStartOptions) => void
  teamEnabled?: boolean
  pacedEnabled?: boolean
}

export function ModePicker({ value, onChange, teamEnabled = true, pacedEnabled = true }: Props) {
  const { t } = useTranslation()
  const modes = MODES.filter((m) => {
    if (m === 'team') return teamEnabled
    if (m === 'student_paced') return pacedEnabled
    return true
  })

  return (
    <fieldset className="space-y-3">
      <legend className="text-sm font-medium text-[var(--color-text)]">{t('liveQuiz.mode.label')}</legend>
      <div className="flex flex-col gap-2" role="radiogroup" aria-label={t('liveQuiz.mode.label')}>
        {modes.map((id) => (
          <label
            key={id}
            className="flex cursor-pointer items-start gap-2 rounded-md border border-[var(--color-border)] px-3 py-2"
          >
            <input
              type="radio"
              name="liveQuizMode"
              checked={value.mode === id}
              onChange={() => onChange({ ...value, mode: id })}
              className="mt-1"
            />
            <span>
              <span className="block text-sm font-medium">{t(`liveQuiz.mode.${id}`)}</span>
              <span className="block text-xs text-[var(--color-text-muted)]">
                {t(`liveQuiz.mode.${id}Hint`)}
              </span>
            </span>
          </label>
        ))}
      </div>

      {value.mode === 'team' ? (
        <div className="grid gap-2 sm:grid-cols-2">
          <label className="text-sm">
            <span className="mb-1 block text-[var(--color-text-muted)]">{t('liveQuiz.team.teamCount')}</span>
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
              className="w-full rounded-md border border-[var(--color-border)] bg-transparent px-2 py-1"
            />
          </label>
          <label className="text-sm">
            <span className="mb-1 block text-[var(--color-text-muted)]">{t('liveQuiz.team.aggregate')}</span>
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
              className="w-full rounded-md border border-[var(--color-border)] bg-transparent px-2 py-1"
            >
              <option value="average">{t('liveQuiz.team.aggregateAverage')}</option>
              <option value="sum">{t('liveQuiz.team.aggregateSum')}</option>
            </select>
          </label>
          <label className="text-sm sm:col-span-2">
            <span className="mb-1 block text-[var(--color-text-muted)]">{t('liveQuiz.team.answerRule')}</span>
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
              className="w-full rounded-md border border-[var(--color-border)] bg-transparent px-2 py-1"
            >
              <option value="each_member_answers">{t('liveQuiz.team.eachMember')}</option>
              <option value="one_device_per_team">{t('liveQuiz.team.oneDevice')}</option>
            </select>
          </label>
        </div>
      ) : null}

      {value.mode === 'student_paced' ? (
        <div className="grid gap-2 sm:grid-cols-2">
          <label className="flex items-center gap-2 text-sm">
            <input
              type="checkbox"
              checked={value.pacedConfig.shuffle ?? true}
              onChange={(e) =>
                onChange({
                  ...value,
                  pacedConfig: { ...value.pacedConfig, shuffle: e.target.checked },
                })
              }
            />
            {t('liveQuiz.paced.shuffle')}
          </label>
          <label className="flex items-center gap-2 text-sm">
            <input
              type="checkbox"
              checked={value.pacedConfig.perQuestionTimers ?? true}
              onChange={(e) =>
                onChange({
                  ...value,
                  pacedConfig: { ...value.pacedConfig, perQuestionTimers: e.target.checked },
                })
              }
            />
            {t('liveQuiz.paced.perQuestionTimers')}
          </label>
          <label className="text-sm sm:col-span-2">
            <span className="mb-1 block text-[var(--color-text-muted)]">
              {t('liveQuiz.paced.timeBudget')}
            </span>
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
              className="w-full rounded-md border border-[var(--color-border)] bg-transparent px-2 py-1"
            />
          </label>
        </div>
      ) : null}
    </fieldset>
  )
}

type TeamAnswerRule = 'each_member_answers' | 'one_device_per_team'
