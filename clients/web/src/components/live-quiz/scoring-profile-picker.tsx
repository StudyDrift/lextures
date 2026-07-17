import { useTranslation } from 'react-i18next'
import type { LeaderboardPrivacy } from '../../lib/live-quiz-api'
import {
  defaultCustomScoringConfig,
  type ScoringStartOptions,
} from './scoring-start-options'

export type { ScoringStartOptions }

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
    <fieldset className="space-y-4 rounded-lg border p-4">
      <legend className="px-1 text-sm font-semibold">{t('liveQuiz.score.profileHeading')}</legend>
      <div className="grid gap-2 sm:grid-cols-3">
        {(
          [
            ['competitive', 'liveQuiz.score.profile.competitive'],
            ['formative', 'liveQuiz.score.profile.formative'],
            ['custom', 'liveQuiz.score.profile.custom'],
          ] as const
        ).map(([id, labelKey]) => (
          <label
            key={id}
            className={
              profile === id
                ? 'cursor-pointer rounded-md border border-primary bg-primary/5 p-3 text-sm'
                : 'cursor-pointer rounded-md border p-3 text-sm'
            }
          >
            <input
              type="radio"
              className="sr-only"
              name="scoringProfile"
              checked={profile === id}
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
            <span className="font-medium">{t(labelKey)}</span>
            <p className="mt-1 text-xs text-muted-foreground">{t(`${labelKey}Hint`)}</p>
          </label>
        ))}
      </div>

      {profile === 'custom' ? (
        <div className="grid gap-3 sm:grid-cols-2">
          <label className="text-sm">
            <span className="mb-1 block">{t('liveQuiz.score.base')}</span>
            <input
              type="number"
              min={100}
              step={100}
              className="w-full rounded-md border px-2 py-1.5"
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
            <span className="mb-1 block">{t('liveQuiz.score.speedWeight')}</span>
            <input
              type="number"
              min={0}
              max={2}
              step={0.1}
              className="w-full rounded-md border px-2 py-1.5"
              value={value.scoringConfig.speedWeight ?? 1}
              onChange={(e) =>
                onChange({
                  ...value,
                  scoringConfig: { ...value.scoringConfig, speedWeight: Number(e.target.value) || 0 },
                })
              }
            />
          </label>
          <label className="text-sm">
            <span className="mb-1 block">{t('liveQuiz.score.streakStep')}</span>
            <input
              type="number"
              min={0}
              step={50}
              className="w-full rounded-md border px-2 py-1.5"
              value={value.scoringConfig.streakStep ?? 100}
              onChange={(e) =>
                onChange({
                  ...value,
                  scoringConfig: { ...value.scoringConfig, streakStep: Number(e.target.value) || 0 },
                })
              }
            />
          </label>
          <label className="text-sm">
            <span className="mb-1 block">{t('liveQuiz.score.streakCap')}</span>
            <input
              type="number"
              min={0}
              max={20}
              className="w-full rounded-md border px-2 py-1.5"
              value={value.scoringConfig.streakCap ?? 5}
              onChange={(e) =>
                onChange({
                  ...value,
                  scoringConfig: { ...value.scoringConfig, streakCap: Number(e.target.value) || 0 },
                })
              }
            />
          </label>
        </div>
      ) : null}

      <label className="flex items-center gap-2 text-sm">
        <input
          type="checkbox"
          checked={value.powerUpsEnabled}
          onChange={(e) => onChange({ ...value, powerUpsEnabled: e.target.checked })}
        />
        {t('liveQuiz.powerup.enable')}
      </label>

      <label className="block text-sm">
        <span className="mb-1 block">{t('liveQuiz.leaderboard.privacyLabel')}</span>
        <select
          className="w-full rounded-md border px-2 py-1.5"
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
