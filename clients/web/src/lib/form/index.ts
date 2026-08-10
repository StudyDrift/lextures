/**
 * UX.6 — Form and validation system.
 *
 * Declare a zod schema, call `useForm`, compose `Field` + controls, render
 * `ErrorSummary` from `useFormState(form).summary`. See docs/guides/forms.md.
 */

export type {
  FieldPath,
  FieldViolation,
  FormApi,
  FormControlProps,
  FormFieldState,
  FormSubmitHandler,
  FormSubmitHelpers,
  UseFormOptions,
  ValidationErrorResponse,
} from './types'

export {
  formatValidationIssue,
  ValidationCode,
  zodIssueToValidationIssue,
  type TranslateFn,
  type ValidationIssue,
} from './validation-messages'

export {
  parseLegacyFieldErrors,
  parseValidationErrorResponse,
  readFieldAddressableErrors,
} from './parse-validation-error'

export { createFormStore, subscribeFormTelemetry } from './form-store'

export { useForm, useFormField, useFormState } from './use-form'
export { useUnsavedChanges } from './use-unsaved-changes'
