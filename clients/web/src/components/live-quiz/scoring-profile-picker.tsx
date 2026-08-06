import { useTranslation } from 'react-i18next'
import type { LeaderboardPrivacy } from '../../lib/live-quiz-api'
import {
  defaultCustomScoringConfig,
  type ScoringStartOptions,
} from './scoring-start-options'

export type { ScoringStartOptions }

const fieldClass =
  'w-full rounded-xl border border-border-default bg-surface-raised px-3 py-2 text-sm text-fg-default outline-none focus:border-indigo-400 focus:ring-2 focus:ring-indigo-400/30 dark:border-border-default dark:bg-surface-base dark:text-fg-default'
const labelMuted = 'mb-1 block text-xs font-medium text-fg-muted'

export function ScoringProfilePicker({
  value,
  onChange,
}: {
  value: ScoringStartOptions
  onChange: (next: ScoringStartOptions) => void
}) {
  const { t } = useTranslation('common')
  const profile = value.scoringProfile

  return (
    <fieldset className="space-y-4">
      <legend className="text-sm font-semibold text-fg-default">
        {t('liveQuiz.score.profileHeading')}
      </legend>
      <div className="grid gap-2 sm:grid-cols-3">
        {(
          [
            ['competitive', 'liveQuiz.score.profile.competitive'],
            ['formative', 'liveQuiz.score.profile.formative'],
            ['custom', 'liveQuiz.score.profile.custom'],
          ] as const
        ).map(([id, labelKey]) => {
          const selected = profile === id
          return (
            <label
              key={id}
              className={
                selected
                  ? 'cursor-pointer rounded-xl border border-indigo-400 bg-indigo-50/60 p-3 text-sm dark:border-indigo-500 dark:bg-indigo-950/40'
                  : 'cursor-pointer rounded-xl border border-border-default bg-surface-raised p-3 text-sm hover:border-border-strong dark:bg-surface-base dark:hover:border-border-default'
              }
            >
              <input
                type="radio"
                className="sr-only"
                name="scoringProfile"
                checked={selected}
                onChange={() =>
                  onChange({
                    ...value,
                    scoringProfile: id,
                    scoringConfig:
                      id === 'custom'
                        ? { ...defaultCustomScoringConfig, ...value.scoringConfig }
                        : value.scoringConfig,
                  })
                }
              />
              <span className="font-medium text-fg-default">{t(labelKey)}</span>
              <p className="mt-1 text-xs text-fg-muted">
                {t(`${labelKey}Hint`)}
              </p>
            </label>
          )
        })}
      </div>

      {profile === 'custom' ? (
        <div className="grid gap-3 sm:grid-cols-2">
          <label className="text-sm">
            <span className={labelMuted}>{t('liveQuiz.score.base')}</span>
            <input
              type="number"
              min={100}
              step={100}
              className={fieldClass}
              value={value.scoringConfig.base ?? 1000}
              onChange={(e) =>
                onChange({
                  ...value,
                  scoringConfig: { ...value.scoringConfig, base: Number(e.target.value) || 1000 },
                })
              }
            />
          </label>
          <label className="text-sm">
            <span className={labelMuted}>{t('liveQuiz.score.speedWeight')}</span>
            <input
              type="number"
              min={0}
              max={2}
              step={0.1}
              className={fieldClass}
              value={value.scoringConfig.speedWeight ?? 1}
              onChange={(e) =>
                onChange({
                  ...value,
                  scoringConfig: {
                    ...value.scoringConfig,
                    speedWeight: Number(e.target.value) || 0,
                  },
                })
              }
            />
          </label>
          <label className="text-sm">
            <span className={labelMuted}>{t('liveQuiz.score.streakStep')}</span>
            <input
              type="number"
              min={0}
              step={50}
              className={fieldClass}
              value={value.scoringConfig.streakStep ?? 100}
              onChange={(e) =>
                onChange({
                  ...value,
                  scoringConfig: {
                    ...value.scoringConfig,
                    streakStep: Number(e.target.value) || 0,
                  },
                })
              }
            />
          </label>
          <label className="text-sm">
            <span className={labelMuted}>{t('liveQuiz.score.streakCap')}</span>
            <input
              type="number"
              min={0}
              max={20}
              className={fieldClass}
              value={value.scoringConfig.streakCap ?? 5}
              onChange={(e) =>
                onChange({
                  ...value,
                  scoringConfig: {
                    ...value.scoringConfig,
                    streakCap: Number(e.target.value) || 0,
                  },
                })
              }
            />
          </label>
        </div>
      ) : null}

      <label className="flex items-center gap-2 text-sm text-fg-default">
        <input
          type="checkbox"
          checked={value.powerUpsEnabled}
          onChange={(e) => onChange({ ...value, powerUpsEnabled: e.target.checked })}
          className="h-4 w-4 rounded border-border-strong text-accent-fg focus:ring-indigo-500"
        />
        {t('liveQuiz.powerup.enable')}
      </label>

      <label className="block text-sm">
        <span className={labelMuted}>{t('liveQuiz.leaderboard.privacyLabel')}</span>
        <select
          className={fieldClass}
          value={value.leaderboardPrivacy}
          onChange={(e) =>
            onChange({ ...value, leaderboardPrivacy: e.target.value as LeaderboardPrivacy })
          }
        >
          <option value="names">{t('liveQuiz.leaderboard.privacy.names')}</option>
          <option value="nicknames">{t('liveQuiz.leaderboard.privacy.nicknames')}</option>
          <option value="hidden">{t('liveQuiz.leaderboard.privacy.hidden')}</option>
        </select>
      </label>
    </fieldset>
  )
}
