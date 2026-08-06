import { useEffect, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import type { ContentToolResetScope } from '../../../lib/courses-api'

export type ResetScopePickerProps = {
  value: ContentToolResetScope
  onChange: (scope: ContentToolResetScope) => void
  allowCourse?: boolean
  allowItem?: boolean
  /** One-learner scopes need a selected enrollment; hide them when none is set. */
  allowEnrollment?: boolean
}

const SCOPES: ContentToolResetScope[] = [
  'instance_enrollment',
  'instance_all',
  'item_enrollment',
  'item_all',
  'course_enrollment',
]

const ENROLLMENT_SCOPES: ReadonlySet<ContentToolResetScope> = new Set([
  'instance_enrollment',
  'item_enrollment',
  'course_enrollment',
])

export function ResetScopePicker({
  value,
  onChange,
  allowCourse = true,
  allowItem = true,
  allowEnrollment = true,
}: ResetScopePickerProps) {
  const { t } = useTranslation('contentTools')
  const options = useMemo(
    () =>
      SCOPES.filter((s) => {
        if (!allowCourse && s === 'course_enrollment') return false
        if (!allowItem && (s === 'item_enrollment' || s === 'item_all')) return false
        if (!allowEnrollment && ENROLLMENT_SCOPES.has(s)) return false
        return true
      }),
    [allowCourse, allowItem, allowEnrollment],
  )

  // If filters hide the current value (e.g. class reset has no enrollment), fall back.
  useEffect(() => {
    if (options.length === 0) return
    if (!options.includes(value)) {
      onChange(options.includes('instance_all') ? 'instance_all' : options[0])
    }
  }, [options, value, onChange])

  return (
    <fieldset className="space-y-2" data-testid="reset-scope-picker">
      <legend className="text-xs font-semibold uppercase tracking-wide text-fg-muted">
        {t('contentTools.reset.scopeLabel')}
      </legend>
      {options.map((scope) => (
        <label key={scope} className="flex items-start gap-2 text-sm text-fg-default">
          <input
            type="radio"
            name="reset-scope"
            value={scope}
            checked={value === scope}
            onChange={() => onChange(scope)}
          />
          <span>
            <span className="font-medium">{t(`contentTools.reset.scopes.${scope}.label`)}</span>
            <span className="mt-0.5 block text-xs text-fg-muted">
              {t(`contentTools.reset.scopes.${scope}.help`)}
            </span>
          </span>
        </label>
      ))}
    </fieldset>
  )
}
