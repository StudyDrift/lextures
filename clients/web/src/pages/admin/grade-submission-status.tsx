import { useEffect, useId, useState } from 'react'
import { usePlatformFeatures } from '../../context/platform-features-context'
import {
  fetchAdminGradeSubmissionStatus,
  type CourseGradeSubmissionStatus,
} from '../../lib/courses-api'

export default function GradeSubmissionStatus() {
  const titleId = useId()
  const { ffGradeSubmission, loading: featuresLoading } = usePlatformFeatures()
  const [termId, setTermId] = useState('')
  const [courses, setCourses] = useState<CourseGradeSubmissionStatus[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!termId.trim() || !ffGradeSubmission) return
    setLoading(true)
    setError(null)
    void (async () => {
      try {
        const rows = await fetchAdminGradeSubmissionStatus(termId.trim())
        setCourses(rows)
      } catch (e) {
        setError(e instanceof Error ? e.message : 'Failed to load status.')
      } finally {
        setLoading(false)
      }
    })()
  }, [termId, ffGradeSubmission])

  if (featuresLoading) return <p>Loading…</p>

  if (!ffGradeSubmission) {
    return (
      <div role="alert">
        <p>Final grade submission is not enabled for this institution.</p>
      </div>
    )
  }

  const submitted = courses.filter((c) => c.submittedAt !== null)
  const pending = courses.filter((c) => c.submittedAt === null)

  return (
    <main aria-labelledby={titleId}>
      <h1 id={titleId}>Final Grade Submission Status</h1>

      <div>
        <label htmlFor="termId">Term ID</label>
        <input
          id="termId"
          type="text"
          value={termId}
          onChange={(e) => setTermId(e.target.value)}
          placeholder="Enter term UUID"
        />
      </div>

      {error && (
        <div role="alert" style={{ color: 'red' }}>
          {error}
        </div>
      )}

      {loading && <p>Loading…</p>}

      {!loading && termId && courses.length === 0 && (
        <p>No courses found for this term.</p>
      )}

      {!loading && courses.length > 0 && (
        <>
          <p>
            {submitted.length} of {courses.length} courses submitted
          </p>
          <table aria-label="Grade submission status">
            <thead>
              <tr>
                <th scope="col">Course</th>
                <th scope="col">Course Code</th>
                <th scope="col">Students Submitted</th>
                <th scope="col">Submitted By</th>
                <th scope="col">Submitted At</th>
              </tr>
            </thead>
            <tbody>
              {[...submitted, ...pending].map((row) => (
                <tr key={row.courseId}>
                  <td>{row.courseTitle}</td>
                  <td>{row.courseCode}</td>
                  <td>{row.count}</td>
                  <td>{row.submittedBy ?? '—'}</td>
                  <td>
                    {row.submittedAt
                      ? new Date(row.submittedAt).toLocaleString()
                      : 'Not submitted'}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </>
      )}
    </main>
  )
}
