import { useCallback, useEffect, useId, useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import { usePlatformFeatures } from '../../context/platform-features-context'
import {
  fetchDisaggregatedPerformance,
  fetchTitle1Report,
  raceEthnicityLabel,
  type DisaggregatedReport,
  type Title1Report,
} from '../../lib/demographics-api'

export default function Title1ReportPage() {
  const titleId = useId()
  const perfId = useId()
  const [searchParams] = useSearchParams()
  const schoolId = searchParams.get('schoolId') ?? ''
  const { ffDemographics, loading: featuresLoading } = usePlatformFeatures()
  const [report, setReport] = useState<Title1Report | null>(null)
  const [performance, setPerformance] = useState<DisaggregatedReport | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    if (!schoolId) return
    setLoading(true)
    setError(null)
    try {
      const [title1, perf] = await Promise.all([
        fetchTitle1Report(schoolId),
        fetchDisaggregatedPerformance(schoolId, 'ell'),
      ])
      setReport(title1)
      setPerformance(perf)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load report.')
    } finally {
      setLoading(false)
    }
  }, [schoolId])

  useEffect(() => {
    if (featuresLoading || !ffDemographics || !schoolId) return
    void load()
  }, [featuresLoading, ffDemographics, load, schoolId])

  if (featuresLoading) {
    return (
      <main className="mx-auto max-w-4xl p-6">
        <h1 className="text-xl font-bold text-slate-900 dark:text-neutral-100">Title I report</h1>
        <p className="mt-6 text-sm" role="status">
          Loading…
        </p>
      </main>
    )
  }

  if (!ffDemographics) {
    return (
      <main className="mx-auto max-w-3xl p-6">
        <p className="text-sm text-slate-600 dark:text-neutral-400">
          Student demographics are not enabled on this platform. Enable{' '}
          <strong>Student demographics</strong> in Settings → Global platform.
        </p>
      </main>
    )
  }

  if (!schoolId) {
    return (
      <main className="mx-auto max-w-3xl p-6">
        <p className="text-sm text-slate-600 dark:text-neutral-400">
          Add a <code className="text-xs">?schoolId=</code> query parameter with the school org unit id.
        </p>
      </main>
    )
  }

  return (
    <main className="mx-auto max-w-4xl p-6">
      <h1 id={titleId} className="text-xl font-bold text-slate-900 dark:text-neutral-100">
        Title I aggregate report
      </h1>
      <p className="mt-2 text-sm text-slate-600 dark:text-neutral-400">
        Aggregate demographic breakdown for school {schoolId}. No individual students are identified.
      </p>

      {loading ? (
        <p className="mt-6 text-sm" role="status">
          Loading…
        </p>
      ) : null}
      {error ? (
        <p className="mt-6 text-sm text-rose-700 dark:text-rose-200" role="alert">
          {error}
        </p>
      ) : null}

      {report && !loading && !error ? (
        <section className="mt-6 space-y-6" aria-labelledby={titleId}>
          <div className="rounded-lg border border-slate-200 bg-white p-4 dark:border-neutral-600 dark:bg-neutral-900">
            <h2 className="text-sm font-semibold text-slate-900 dark:text-neutral-100">
              Economic disadvantage
            </h2>
            <p className="mt-2 text-2xl font-bold text-slate-900 dark:text-neutral-100">
              {report.economicDisadvantagePct}% ({report.economicDisadvantaged}/{report.totalStudents}{' '}
              students)
            </p>
            <p className="mt-1 text-xs text-slate-500 dark:text-neutral-400">
              Free lunch: {report.freeLunchCount} · Reduced lunch: {report.reducedLunchCount}
            </p>
          </div>

          <table className="w-full border-collapse text-sm" aria-labelledby={titleId}>
            <caption className="sr-only">School demographic counts</caption>
            <thead>
              <tr className="border-b border-slate-200 text-left dark:border-neutral-600">
                <th scope="col" className="py-2 pe-4">
                  Category
                </th>
                <th scope="col" className="py-2">
                  Count
                </th>
              </tr>
            </thead>
            <tbody>
              <tr className="border-b border-slate-100 dark:border-neutral-700">
                <td className="py-2 pe-4">English language learners</td>
                <td className="py-2">{report.ellCount}</td>
              </tr>
              <tr className="border-b border-slate-100 dark:border-neutral-700">
                <td className="py-2 pe-4">Students with disabilities (flag only)</td>
                <td className="py-2">{report.disabilityCount}</td>
              </tr>
              <tr className="border-b border-slate-100 dark:border-neutral-700">
                <td className="py-2 pe-4">Homeless</td>
                <td className="py-2">{report.homelessCount}</td>
              </tr>
              <tr className="border-b border-slate-100 dark:border-neutral-700">
                <td className="py-2 pe-4">Migrant</td>
                <td className="py-2">{report.migrantCount}</td>
              </tr>
              {Object.entries(report.raceBreakdown).map(([code, count]) => (
                <tr key={code} className="border-b border-slate-100 dark:border-neutral-700">
                  <td className="py-2 pe-4">Race/ethnicity: {raceEthnicityLabel(code)}</td>
                  <td className="py-2">{count}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </section>
      ) : null}

      {performance && !loading && !error ? (
        <section className="mt-8" aria-labelledby={perfId}>
          <h2 id={perfId} className="text-sm font-semibold text-slate-900 dark:text-neutral-100">
            Quiz pass rate by ELL status
          </h2>
          <table className="mt-3 w-full border-collapse text-sm" aria-labelledby={perfId}>
            <caption className="sr-only">Disaggregated quiz pass rates</caption>
            <thead>
              <tr className="border-b border-slate-200 text-left dark:border-neutral-600">
                <th scope="col" className="py-2 pe-4">
                  Subgroup
                </th>
                <th scope="col" className="py-2 pe-4">
                  Students (n)
                </th>
                <th scope="col" className="py-2">
                  Pass rate
                </th>
              </tr>
            </thead>
            <tbody>
              {performance.subgroups.map((row) => (
                <tr key={row.label} className="border-b border-slate-100 dark:border-neutral-700">
                  <td className="py-2 pe-4">{row.label}</td>
                  <td className="py-2 pe-4">{row.count}</td>
                  <td className="py-2">
                    {row.suppressed ? (
                      <span className="text-slate-500 dark:text-neutral-400">
                        Data suppressed (n&lt;10)
                      </span>
                    ) : (
                      `${row.passRate?.toFixed(1) ?? '—'}%`
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </section>
      ) : null}
    </main>
  )
}
