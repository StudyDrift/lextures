import { useCallback, useEffect, useId, useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import { usePlatformFeatures } from '../../context/platform-features-context'
import {
  fetchStudentDemographics,
  patchStudentDemographics,
  raceEthnicityLabel,
  type StudentDemographics,
} from '../../lib/demographics-api'

export default function StudentDemographicsPage() {
  const titleId = useId()
  const [searchParams] = useSearchParams()
  const studentId = searchParams.get('studentId') ?? ''
  const { ffDemographics, loading: featuresLoading } = usePlatformFeatures()
  const [record, setRecord] = useState<StudentDemographics | null>(null)
  const [loading, setLoading] = useState(false)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [message, setMessage] = useState<string | null>(null)

  const load = useCallback(async () => {
    if (!studentId) return
    setLoading(true)
    setError(null)
    try {
      const data = await fetchStudentDemographics(studentId)
      setRecord(data)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load demographics.')
    } finally {
      setLoading(false)
    }
  }, [studentId])

  useEffect(() => {
    if (featuresLoading || !ffDemographics || !studentId) return
    void load()
  }, [featuresLoading, ffDemographics, load, studentId])

  async function toggleFlag(field: 'ellStatus' | 'disabilityStatus' | 'homelessIndicator' | 'migrantIndicator') {
    if (!record || !studentId) return
    setSaving(true)
    setMessage(null)
    setError(null)
    try {
      const next = !(record[field] === true)
      const updated = await patchStudentDemographics(studentId, { [field]: next })
      setRecord(updated)
      setMessage('Demographics updated.')
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to save.')
    } finally {
      setSaving(false)
    }
  }

  if (featuresLoading) {
    return (
      <main className="mx-auto max-w-3xl p-6">
        <h1 className="text-xl font-bold text-slate-900 dark:text-neutral-100">Student demographics</h1>
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
          Student demographics are not enabled on this platform.
        </p>
      </main>
    )
  }

  if (!studentId) {
    return (
      <main className="mx-auto max-w-3xl p-6">
        <p className="text-sm text-slate-600 dark:text-neutral-400">
          Add a <code className="text-xs">?studentId=</code> query parameter.
        </p>
      </main>
    )
  }

  const hasFlags =
    record?.freeLunch != null ||
    record?.reducedLunch != null ||
    record?.ellStatus != null ||
    record?.disabilityStatus != null

  return (
    <main className="mx-auto max-w-3xl p-6">
      <h1 id={titleId} className="text-xl font-bold text-slate-900 dark:text-neutral-100">
        Student demographics
      </h1>
      <p className="mt-2 text-sm text-slate-600 dark:text-neutral-400">
        Restricted access — individual lunch eligibility is confidential (FERPA / NSLP).
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
      {message ? (
        <p className="mt-6 text-sm text-emerald-700 dark:text-emerald-200" role="status">
          {message}
        </p>
      ) : null}

      {record && !loading ? (
        <section className="mt-6" aria-labelledby={titleId}>
          {!hasFlags ? (
            <p className="text-sm text-slate-600 dark:text-neutral-400">
              No demographic record on file for this student.
            </p>
          ) : (
            <dl className="grid gap-3 text-sm">
              {record.freeLunch != null ? (
                <div className="flex justify-between border-b border-slate-100 py-2 dark:border-neutral-700">
                  <dt>Free lunch</dt>
                  <dd>{record.freeLunch ? 'Yes' : 'No'}</dd>
                </div>
              ) : null}
              {record.reducedLunch != null ? (
                <div className="flex justify-between border-b border-slate-100 py-2 dark:border-neutral-700">
                  <dt>Reduced lunch</dt>
                  <dd>{record.reducedLunch ? 'Yes' : 'No'}</dd>
                </div>
              ) : null}
              {record.raceEthnicityCode ? (
                <div className="flex justify-between border-b border-slate-100 py-2 dark:border-neutral-700">
                  <dt>Race/ethnicity</dt>
                  <dd>{raceEthnicityLabel(record.raceEthnicityCode)}</dd>
                </div>
              ) : null}
              {record.dataSource ? (
                <div className="flex justify-between border-b border-slate-100 py-2 dark:border-neutral-700">
                  <dt>Data source</dt>
                  <dd>{record.dataSource}</dd>
                </div>
              ) : null}
            </dl>
          )}

          <div className="mt-6 flex flex-wrap gap-2">
            <button
              type="button"
              disabled={saving}
              className="rounded-md border border-slate-300 px-3 py-1.5 text-sm dark:border-neutral-600"
              onClick={() => void toggleFlag('ellStatus')}
            >
              Toggle ELL
            </button>
            <button
              type="button"
              disabled={saving}
              className="rounded-md border border-slate-300 px-3 py-1.5 text-sm dark:border-neutral-600"
              onClick={() => void toggleFlag('disabilityStatus')}
            >
              Toggle disability flag
            </button>
          </div>
        </section>
      ) : null}
    </main>
  )
}
