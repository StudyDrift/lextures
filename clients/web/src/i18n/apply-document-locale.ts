import { resolveResourceLanguage } from './supported-locales'

/** Updates `<html lang>` for screen readers (WCAG 3.1.1, plan 11.1). */
export function applyDocumentLocale(tag: string): void {
  if (typeof document === 'undefined') return
  const lang = resolveResourceLanguage(tag)
  document.documentElement.lang = lang
  document.documentElement.setAttribute('data-locale', tag.trim() || lang)
}
