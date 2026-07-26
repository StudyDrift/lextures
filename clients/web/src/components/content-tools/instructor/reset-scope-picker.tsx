import { useTranslation } from 'react-i18next'
import type { ContentToolResetScope } from '../../../lib/courses-api'

export type ResetScopePickerProps = {
  value: ContentToolResetScope
  onChange: (scope: ContentToolResetScope) => void
  allowCourse?: boolean
  allowItem?: boolean
}

const SCOPES: ContentToolResetScope[] = [
  'instance_enrollment',
  'instance_all',
  'item_enrollment',
  'item_all',
  'course_enrollment',
]

export function ResetScopePicker({
  value,
  onChange,
  allowCourse = true,
  allowItem = true,
}: ResetScopePickerProps) {
  const { t } = useTranslation('contentTools')
  const options = SCOPES.filter((s) => {
    if (!allowCourse && s === 'course_enrollment') return false
    if (!allowItem && (s === 'item_enrollment' || s === 'item_all')) return false
    return true
  })

  return (
    <fieldset className="space-y-2" data-testid="reset-scope-picker">
      <legend className="text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-neutral-400">
        {t('contentTools.reset.scopeLabel')}
      </legend>
      {options.map((scope) => (
        <label key={scope} className="flex items-start gap-2 text-sm text-slate-800 dark:text-neutral-200">
          <input
            type="radio"
            name="reset-scope"
            value={scope}
            checked={value === scope}
            onChange={() => onChange(scope)}
          />
          <span>
            <span className="font-medium">{t(`contentTools.reset.scopes.${scope}.label`)}</span>
            <span className="mt-0.5 block text-xs text-slate-500 dark:text-neutral-400">
              {t(`contentTools.reset.scopes.${scope}.help`)}
            </span>
          </span>
        </label>
      ))}
    </fieldset>
  )
}
