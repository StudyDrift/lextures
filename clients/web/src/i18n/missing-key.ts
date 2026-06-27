type MissingKeyLabels = {
  locale: string
  namespace: string
  key: string
}

const missingKeyCounts = new Map<string, number>()

function metricKey({ locale, namespace, key }: MissingKeyLabels): string {
  return `${locale}|${namespace}|${key}`
}

/** Development warning + in-memory metric for i18n_missing_key_total (plan 11.1 AC-2). */
export function recordMissingTranslationKey(labels: MissingKeyLabels): void {
  const id = metricKey(labels)
  missingKeyCounts.set(id, (missingKeyCounts.get(id) ?? 0) + 1)
  if (import.meta.env.DEV) {
    console.warn(
      `[i18n] missing key i18n_missing_key_total{locale="${labels.locale}",namespace="${labels.namespace}"} key="${labels.key}"`,
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
