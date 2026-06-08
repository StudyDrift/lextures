import { useCallback, useEffect, useId, useState } from 'react'
import { useParams } from 'react-router-dom'
import { usePlatformFeatures } from '../../context/platform-features-context'
import {
  fetchFinalGradesPreview,
  submitFinalGrades,
  type FinalGradeOverride,
  type FinalGradeStudentRow,
} from '../../lib/courses-api'

export default function FinalGradeSubmission() {
  const { courseCode } = useParams<{ courseCode: string }>()
  const titleId = useId()
  const { ffGradeSubmission, loading: featuresLoading } = usePlatformFeatures()

  const [grades, setGrades] = useState<FinalGradeStudentRow[]>([])
  const [overrides, setOverrides] = useState<Record<string, string>>({})
  const [reasons, setReasons] = useState<Record<string, string>>({})
  const [loading, setLoading] = useState(false)
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [downloadUrl, setDownloadUrl] = useState<string | null>(null)
  const [submitted, setSubmitted] = useState(false)

  const load = useCallback(async () => {
    if (!courseCode) return
    setLoading(true)
    setError(null)
    try {
      const preview = await fetchFinalGradesPreview(courseCode)
      setGrades(preview.grades)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load grades.')
    } finally {
      setLoading(false)
    }
  }, [courseCode])

  useEffect(() => {
    if (featuresLoading || !ffGradeSubmission) return
    void load()
  }, [featuresLoading, ffGradeSubmission, load])

  async function handleSubmit() {
    if (!courseCode) return
    setSubmitting(true)
    setError(null)
    try {
      const overrideList: FinalGradeOverride[] = Object.entries(overrides)
        .filter(([, grade]) => grade.trim() !== '')
        .map(([enrollmentId, finalGrade]) => ({
          enrollmentId,
          finalGrade: finalGrade.trim(),
          overrideReason: reasons[enrollmentId]?.trim() || undefined,
        }))
      const result = await submitFinalGrades(courseCode, { method: 'csv', overrides: overrideList })
      setDownloadUrl(result.downloadUrl)
      setSubmitted(true)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Submission failed.')
    } finally {
      setSubmitting(false)
    }
  }

  if (featuresLoading) {
    return <p>Loading…</p>
  }

  if (!ffGradeSubmission) {
    return (
      <div role="alert">
        <p>Final grade submission is not enabled for this institution.</p>
      </div>
    )
  }

  return (
    <main aria-labelledby={titleId}>
      <h1 id={titleId}>Submit Final Grades</h1>

      {error && (
        <div role="alert" style={{ color: 'red' }}>
          {error}
        </div>
      )}

      {submitted && downloadUrl && (
        <div role="status">
          <p>Grades submitted successfully.</p>
          <a href={downloadUrl} download>
            Download CSV
          </a>
        </div>
      )}

      {loading && <p>Loading grades…</p>}

      {!loading && grades.length === 0 && !submitted && (
        <p>No student grades available for this course.</p>
      )}

      {!loading && grades.length > 0 && (
        <>
          <p>
            Review computed grades below. You may override individual grades before submitting to
            the registrar.
          </p>
          <table aria-label="Final grades">
            <thead>
              <tr>
                <th scope="col">Student</th>
                <th scope="col">SIS ID</th>
                <th scope="col">Status</th>
                <th scope="col">Computed Grade</th>
                <th scope="col">Final Grade</th>
                <th scope="col">Override</th>
                <th scope="col">Override Reason</th>
                <th scope="col">Previously Submitted</th>
              </tr>
            </thead>
            <tbody>
              {grades.map((row) => (
                <tr key={row.enrollmentId}>
                  <td>{row.displayName}</td>
                  <td>{row.externalSisId || row.userId}</td>
                  <td>{row.state}</td>
                  <td>{row.computedGrade}</td>
                  <td>{overrides[row.enrollmentId] || row.finalGrade}</td>
                  <td>
                    <input
                      type="text"
                      aria-label={`Override grade for ${row.displayName}`}
                      value={overrides[row.enrollmentId] ?? ''}
                      onChange={(e) =>
                        setOverrides((prev) => ({ ...prev, [row.enrollmentId]: e.target.value }))
                      }
                      placeholder={row.finalGrade}
                    />
                  </td>
                  <td>
                    <input
                      type="text"
                      aria-label={`Override reason for ${row.displayName}`}
                      value={reasons[row.enrollmentId] ?? ''}
                      onChange={(e) =>
                        setReasons((prev) => ({ ...prev, [row.enrollmentId]: e.target.value }))
                      }
                      placeholder="Reason (optional)"
                    />
                  </td>
                  <td>{row.alreadySubmitted ? 'Yes' : 'No'}</td>
                </tr>
              ))}
            </tbody>
          </table>

          <button type="button" onClick={() => void handleSubmit()} disabled={submitting}>
            {submitting ? 'Submitting…' : 'Submit Final Grades to Registrar'}
          </button>
        </>
      )}
    </main>
  )
}
