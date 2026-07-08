import { useTranslation } from 'react-i18next'
import type { FacetKey } from '../../../lib/learner-profile-api'

type Props = {
  facetKey: FacetKey
  summary: Record<string, unknown>
}

export function FacetVisualization({ facetKey, summary }: Props) {
  const { t } = useTranslation('learnerProfile')

  if (facetKey === 'study_rhythm') {
    const peakWindows = summary.peakWindows as Array<{ dow?: string; hourBucket?: string; share?: number }> | undefined
    if (!peakWindows?.length) return null
    return (
      <div className="mt-4 overflow-x-auto">
        <table className="min-w-full text-start text-xs text-slate-700 dark:text-neutral-200">
          <caption className="sr-only">{t('learnerProfile.chart.rhythm.caption')}</caption>
          <thead>
            <tr className="border-b border-slate-200 dark:border-neutral-700">
              <th scope="col" className="px-2 py-1.5 font-semibold">
                {t('learnerProfile.chart.day')}
              </th>
              <th scope="col" className="px-2 py-1.5 font-semibold">
                {t('learnerProfile.chart.hour')}
              </th>
              <th scope="col" className="px-2 py-1.5 font-semibold">
                {t('learnerProfile.chart.activity')}
              </th>
            </tr>
          </thead>
          <tbody>
            {peakWindows.slice(0, 5).map((row) => (
              <tr key={`${row.dow}-${row.hourBucket}`} className="border-b border-slate-100 dark:border-neutral-800">
                <td className="px-2 py-1.5">{row.dow}</td>
                <td className="px-2 py-1.5">{row.hourBucket}</td>
                <td className="px-2 py-1.5">{Math.round((row.share ?? 0) * 100)}%</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    )
  }

  if (facetKey === 'content_modality') {
    const affinity = summary.modalityAffinity as Record<string, number> | undefined
    if (!affinity) return null
    const rows = Object.entries(affinity).sort((a, b) => b[1] - a[1])
    return (
      <div className="mt-4 overflow-x-auto">
        <table className="min-w-full text-start text-xs text-slate-700 dark:text-neutral-200">
          <caption className="sr-only">{t('learnerProfile.chart.modality.caption')}</caption>
          <thead>
            <tr className="border-b border-slate-200 dark:border-neutral-700">
              <th scope="col" className="px-2 py-1.5 font-semibold">
                {t('learnerProfile.chart.modality')}
              </th>
              <th scope="col" className="px-2 py-1.5 font-semibold">
                {t('learnerProfile.chart.affinity')}
              </th>
            </tr>
          </thead>
          <tbody>
            {rows.map(([modality, score]) => (
              <tr key={modality} className="border-b border-slate-100 dark:border-neutral-800">
                <td className="px-2 py-1.5 capitalize">{modality}</td>
                <td className="px-2 py-1.5">{Math.round(score * 100)}%</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    )
  }

  return null
}