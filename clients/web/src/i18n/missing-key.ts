type MissingKeyLabels = {
  locale: string
  namespace: string
  key: string
}

const missingKeyCounts = new Map<string, number>()

function metricKey({ locale, namespace, key }: MissingKeyLabels): string {
  return `${locale}|${namespace}|${key}`
}

function shouldFailOnMissingKeys(): boolean {
  return import.meta.env.VITE_I18N_FAIL_ON_MISSING === '1'
}

/** Development warning + in-memory metric for missing_translation_key (plan W01 FR-5). */
export function recordMissingTranslationKey(labels: MissingKeyLabels): void {
  const id = metricKey(labels)
  missingKeyCounts.set(id, (missingKeyCounts.get(id) ?? 0) + 1)
  if (import.meta.env.DEV) {
    console.warn(
      `[i18n] missing key missing_translation_key{locale="${labels.locale}",namespace="${labels.namespace}"} key="${labels.key}"`,
    )
  }
  if (shouldFailOnMissingKeys()) {
    throw new Error(
      `missing_translation_key: locale=${labels.locale} namespace=${labels.namespace} key=${labels.key}`,
    )
  }
}

/** Test helper — reset counters between tests. */
export function resetMissingKeyMetrics(): void {
  missingKeyCounts.clear()
}

export function getMissingKeyCountFor(locale: string, namespace: string, key: string): number {
  return missingKeyCounts.get(metricKey({ locale, namespace, key })) ?? 0
}

export function getMissingKeyMetrics(): ReadonlyMap<string, number> {
  return missingKeyCounts
}
