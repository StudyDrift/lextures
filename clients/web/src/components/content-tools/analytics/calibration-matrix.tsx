import { useTranslation } from 'react-i18next'

export type CalibrationCell = {
  confidenceBucket: string
  correct: boolean
  count: number
  highlight: boolean
}

export type CalibrationMatrixProps = {
  cells: CalibrationCell[]
}

export function CalibrationMatrix({ cells }: CalibrationMatrixProps) {
  const { t } = useTranslation('contentTools')
  if (!cells.length) return null

  const buckets = Array.from(new Set(cells.map((c) => c.confidenceBucket)))
  const summary = cells
    .filter((c) => c.highlight)
    .map((c) =>
      t('contentTools.tools.predict_reveal.calibration.confidentlyWrongCell', {
        bucket: c.confidenceBucket,
        count: c.count,
      }),
    )
    .join(' ')

  return (
    <div data-testid="predict-reveal-calibration" className="space-y-2">
      <h3 className="text-sm font-medium text-fg-default">
        {t('contentTools.tools.predict_reveal.calibration.title')}
      </h3>
      {summary ? (
        <p className="text-xs text-amber-800 dark:text-amber-200" data-testid="calibration-summary">
          {summary}
        </p>
      ) : null}
      <div className="overflow-x-auto">
        <table className="min-w-full border-collapse text-sm" role="table">
          <caption className="sr-only">
            {t('contentTools.tools.predict_reveal.calibration.caption')}
          </caption>
          <thead>
            <tr>
              <th
                scope="col"
                className="border border-border-default px-2 py-1 text-start dark:border-border-default"
              >
                {t('contentTools.analytics.facets.confidenceBucket')}
              </th>
              <th
                scope="col"
                className="border border-border-default px-2 py-1 text-start dark:border-border-default"
              >
                {t('contentTools.tools.predict_reveal.calibration.correct')}
              </th>
              <th
                scope="col"
                className="border border-border-default px-2 py-1 text-start dark:border-border-default"
              >
                {t('contentTools.tools.predict_reveal.calibration.incorrect')}
              </th>
            </tr>
          </thead>
          <tbody>
            {buckets.map((b) => {
              const correct = cells.find((c) => c.confidenceBucket === b && c.correct)
              const wrong = cells.find((c) => c.confidenceBucket === b && !c.correct)
              return (
                <tr key={b}>
                  <th
                    scope="row"
                    className="border border-border-default px-2 py-1 text-start dark:border-border-default"
                  >
                    {b}
                  </th>
                  <td className="border border-border-default px-2 py-1 dark:border-border-default">
                    {correct?.count ?? 0}
                  </td>
                  <td
                    className={`border border-border-default px-2 py-1 dark:border-border-default ${ wrong?.highlight ? 'bg-amber-100 font-semibold dark:bg-amber-950/50' : '' }`}
                    data-highlight={wrong?.highlight ? 'true' : undefined}
                  >
                    {wrong?.count ?? 0}
                    {wrong?.highlight ? (
                      <span className="ms-1 text-xs text-amber-900 dark:text-amber-100">
                        ({t('contentTools.tools.predict_reveal.calibration.highlight')})
                      </span>
                    ) : null}
                  </td>
                </tr>
              )
            })}
          </tbody>
        </table>
      </div>
    </div>
  )
}
