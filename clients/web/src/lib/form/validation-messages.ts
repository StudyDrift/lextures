/**
 * Map zod issue codes / server machine codes → i18n keys under common.validation.*.
 * Callers pass a translator; this module stays free of react-i18next.
 */

export type ValidationIssue = {
  path: string
  code: string
  message: string
  params?: Record<string, unknown>
}

export type TranslateFn = (key: string, options?: Record<string, unknown>) => string

/** Stable machine codes used by the client and server 422 envelope. */
export const ValidationCode = {
  required: 'required',
  tooShort: 'too_short',
  tooLong: 'too_long',
  invalidEmail: 'invalid_email',
  invalidUrl: 'invalid_url',
  invalidPhone: 'invalid_phone',
  alreadyTaken: 'already_taken',
  mustMatch: 'must_match',
  mustBeAfter: 'must_be_after',
  mustBeBefore: 'must_be_before',
  invalidType: 'invalid_type',
  custom: 'custom',
} as const

const CODE_TO_KEY: Record<string, string> = {
  required: 'common.validation.required',
  too_short: 'common.validation.tooShort',
  too_long: 'common.validation.tooLong',
  invalid_email: 'common.validation.invalidEmail',
  invalid_url: 'common.validation.invalidUrl',
  invalid_phone: 'common.validation.invalidPhone',
  already_taken: 'common.validation.alreadyTaken',
  must_match: 'common.validation.mustMatch',
  must_be_after: 'common.validation.mustBeAfter',
  must_be_before: 'common.validation.mustBeBefore',
  invalid_type: 'common.validation.invalidType',
  // zod v4 codes
  invalid_format: 'common.validation.invalidFormat',
  too_small: 'common.validation.tooShort',
  too_big: 'common.validation.tooLong',
  custom: 'common.validation.custom',
}

/**
 * Default formatter: prefers i18n keys, falls back to the issue message
 * (never invents "Invalid input" alone — SC 3.3.3).
 */
export function formatValidationIssue(
  issue: ValidationIssue,
  t?: TranslateFn,
  label?: string,
): string {
  const key = CODE_TO_KEY[issue.code]
  if (t && key) {
    const translated = t(key, {
      field: label ?? issue.path,
      min: issue.params?.min,
      max: issue.params?.max,
      other: issue.params?.other,
      date: issue.params?.date,
      defaultValue: issue.message,
      ...issue.params,
    })
    if (translated && translated !== key) return translated
  }
  if (issue.message && issue.message.trim() && issue.message !== 'Invalid input') {
    return issue.message
  }
  if (t) {
    return t('common.validation.generic', {
      field: label ?? issue.path,
      defaultValue: 'Enter a valid value for this field.',
    })
  }
  return 'Enter a valid value for this field.'
}

/** Map a zod issue (v3/v4-ish) into our ValidationIssue shape. */
export function zodIssueToValidationIssue(issue: {
  path: PropertyKey[]
  code: string
  message: string
  minimum?: number | bigint
  maximum?: number | bigint
  [k: string]: unknown
}): ValidationIssue {
  const path = issue.path.map(String).join('.')
  let code = issue.code
  const params: Record<string, unknown> = {}

  if (issue.code === 'too_small' || issue.code === 'too_short') {
    code = ValidationCode.tooShort
    if (issue.minimum !== undefined) params.min = Number(issue.minimum)
  } else if (issue.code === 'too_big' || issue.code === 'too_long') {
    code = ValidationCode.tooLong
    if (issue.maximum !== undefined) params.max = Number(issue.maximum)
  } else if (issue.code === 'invalid_format') {
    const fmt = String((issue as { format?: string }).format ?? '')
    if (fmt === 'email') code = ValidationCode.invalidEmail
    else if (fmt === 'url') code = ValidationCode.invalidUrl
    else code = 'invalid_format'
  } else if (
    issue.code === 'invalid_type' &&
    (issue as { received?: string }).received === 'undefined'
  ) {
    code = ValidationCode.required
  }

  return {
    path,
    code,
    message: issue.message,
    params: Object.keys(params).length ? params : undefined,
  }
}
