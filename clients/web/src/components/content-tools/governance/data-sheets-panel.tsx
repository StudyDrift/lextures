import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import {
  fetchContentToolDataSheets,
  type ContentToolDataSheet,
} from '../../../lib/content-tools-governance-api'

/** Trust-centre data sheets (CT.8 FR-1 / FR-13 / FR-14). */
export function DataSheetsPanel() {
  const { t } = useTranslation('contentTools')
  const [sheets, setSheets] = useState<ContentToolDataSheet[]>([])
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    let cancelled = false
    void fetchContentToolDataSheets()
      .then((rows) => {
        if (!cancelled) setSheets(rows)
      })
      .catch((e: unknown) => {
        if (!cancelled) setError(e instanceof Error ? e.message : 'Failed')
      })
    return () => {
      cancelled = true
    }
  }, [])

  return (
    <section className="mt-8" aria-labelledby="ct-datasheets-title" data-testid="content-tool-data-sheets">
      <h2 id="ct-datasheets-title" className="text-lg font-semibold">
        {t('contentTools.governance.dataSheetsTitle')}
      </h2>
      <p className="mt-1 text-sm text-slate-600 dark:text-slate-400">
        {t('contentTools.governance.dataSheetsHelp')}
      </p>
      {error ? (
        <p className="mt-2 text-sm text-rose-700" role="alert">
          {error}
        </p>
      ) : null}
      <ul className="mt-4 space-y-3" role="list">
        {sheets.map((s) => (
          <li
            key={s.toolId}
            className="rounded border border-slate-200 p-3 text-sm dark:border-neutral-700"
            data-tool-id={s.toolId}
          >
            <p className="font-medium">
              {s.toolId} <span className="text-slate-500">v{s.version}</span>
            </p>
            <p className="mt-1 text-slate-600 dark:text-slate-300">
              {t('contentTools.governance.visibility')}: {s.visibility} · WCAG {s.wcagLevel}
              {s.leavesPlatform ? ` · ${t('contentTools.governance.leavesPlatform')}` : ''}
            </p>
            {s.a11yLimitations ? (
              <p className="mt-1 text-amber-800 dark:text-amber-200" role="note">
                {t('contentTools.governance.a11yNote')}: {s.a11yLimitations}
              </p>
            ) : null}
            {s.aiTransparency ? (
              <p className="mt-1 text-slate-600 dark:text-slate-300">
                {t('contentTools.governance.aiPurpose')}: {s.aiTransparency.purpose} (
                {s.aiTransparency.modelFamily})
              </p>
            ) : null}
          </li>
        ))}
      </ul>
    </section>
  )
}
