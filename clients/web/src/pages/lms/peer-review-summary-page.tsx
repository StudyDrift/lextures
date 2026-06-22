import { useCallback, useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import {
  fetchPeerReviewSummary,
  postPeerReviewAllocate,
  putPeerReviewConfig,
  type PeerReviewSummary,
} from '../../lib/peer-review-api'
import { usePlatformFeatures } from '../../context/platform-features-context'
import { LmsPage } from './lms-page'

export default function PeerReviewSummaryPage() {
  const { courseCode, itemId } = useParams<{ courseCode: string; itemId: string }>()
  const { ffPeerReview } = usePlatformFeatures()
  const [summary, setSummary] = useState<PeerReviewSummary | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)

  const load = useCallback(async () => {
    if (!courseCode || !itemId || !ffPeerReview) {
      setLoading(false)
      return
    }
    setLoading(true)
    setError(null)
    try {
      setSummary(await fetchPeerReviewSummary(courseCode, itemId))
    } catch (e) {
      setSummary(null)
      setError(e instanceof Error ? e.message : 'Could not load peer review summary.')
    } finally {
      setLoading(false)
    }
  }, [courseCode, itemId, ffPeerReview])

  useEffect(() => {
    void load()
  }, [load])

  async function enableAndAllocate() {
    if (!courseCode || !itemId) return
    setBusy(true)
    setError(null)
    try {
      await putPeerReviewConfig(courseCode, itemId, {
        reviewsPerReviewer: 3,
        anonymity: 'double_blind',
        gradeMode: 'weighted_blend',
        blendWeight: 0.3,
        aggregation: 'median',
        excludeSameGroup: true,
      })
      await postPeerReviewAllocate(courseCode, itemId)
      await load()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Setup failed.')
    } finally {
      setBusy(false)
    }
  }

  if (!ffPeerReview) {
    return (
      <LmsPage title="Peer review">
        <p className="text-sm text-slate-500 dark:text-neutral-400">Peer review is not enabled.</p>
      </LmsPage>
    )
  }

  return (
    <LmsPage title="Peer review summary">
      <div className="mb-4 flex flex-wrap items-center gap-2">
        {courseCode && itemId ? (
          <Link
            to={`/courses/${courseCode}/assignments/${itemId}`}
            className="text-sm text-indigo-600 hover:underline dark:text-indigo-400"
          >
            Back to assignment
          </Link>
        ) : null}
        {courseCode && itemId ? (
          <button
            type="button"
            disabled={busy}
            onClick={() => void enableAndAllocate()}
            className="rounded border border-slate-200 px-3 py-1 text-sm hover:bg-slate-50 disabled:opacity-60 dark:border-neutral-600 dark:hover:bg-neutral-800"
          >
            {busy ? 'Working…' : 'Configure & allocate'}
          </button>
        ) : null}
      </div>
      {loading ? (
        <p className="text-sm text-slate-500 dark:text-neutral-400">Loading…</p>
      ) : error ? (
        <p className="text-sm text-red-600 dark:text-red-400" role="alert">
          {error}
        </p>
      ) : summary ? (
        <div className="space-y-4 text-sm">
          <p>
            {summary.completedReviews} of {summary.totalAllocations} reviews completed ·{' '}
            {summary.incompleteReviewers.length} incomplete reviewers
          </p>
          <table className="w-full border-collapse text-left">
            <thead>
              <tr className="border-b border-slate-200 dark:border-neutral-700">
                <th className="py-2 pr-4 font-medium">Student</th>
                <th className="py-2 pr-4 font-medium">Peer aggregate</th>
                <th className="py-2 font-medium">Reviews</th>
              </tr>
            </thead>
            <tbody>
              {summary.submissions.map((s) => (
                <tr key={s.submissionId} className="border-b border-slate-100 dark:border-neutral-800">
                  <td className="py-2 pr-4 font-mono text-xs">{s.studentUserId.slice(0, 8)}…</td>
                  <td className="py-2 pr-4">{s.peerAggregate?.toFixed(1) ?? '—'}</td>
                  <td className="py-2">{s.reviewCount}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : null}
    </LmsPage>
  )
}
