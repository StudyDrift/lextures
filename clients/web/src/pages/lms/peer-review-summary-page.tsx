import { useCallback, useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import {
  fetchPeerReviewSummary,
  postPeerReviewAllocate,
  putPeerReviewConfig,
  type PeerReviewSummary,
} from '../../lib/peer-review-api'
import { useTranslation } from 'react-i18next'
import { EntityLabel } from '../../components/ui/entity-label'
import { usePlatformFeatures } from '../../context/platform-features-context'
import { LmsPage } from './lms-page'

export default function PeerReviewSummaryPage() {
  const { t } = useTranslation('common')
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
        <p className="text-sm text-fg-muted">Peer review is not enabled.</p>
      </LmsPage>
    )
  }

  return (
    <LmsPage title="Peer review summary">
      <div className="mb-4 flex flex-wrap items-center gap-2">
        {courseCode && itemId ? (
          <Link
            to={`/courses/${courseCode}/assignments/${itemId}`}
            className="text-sm text-accent-fg hover:underline dark:text-indigo-400"
          >
            Back to assignment
          </Link>
        ) : null}
        {courseCode && itemId ? (
          <button
            type="button"
            disabled={busy}
            onClick={() => void enableAndAllocate()}
            className="rounded border border-border-default px-3 py-1 text-sm hover:bg-surface-base disabled:opacity-60 dark:border-border-default dark:hover:bg-surface-overlay"
          >
            {busy ? 'Working…' : 'Configure & allocate'}
          </button>
        ) : null}
      </div>
      {loading ? (
        <p className="text-sm text-fg-muted">Loading…</p>
      ) : error ? (
        <p className="text-sm text-danger-fg" role="alert">
          {error}
        </p>
      ) : summary ? (
        <div className="space-y-4 text-sm">
          <p>
            {summary.completedReviews} of {summary.totalAllocations} reviews completed ·{' '}
            {summary.incompleteReviewers.length} incomplete reviewers
            {summary.incompleteReviewerLabels && summary.incompleteReviewerLabels.length > 0
              ? ` (${summary.incompleteReviewerLabels.join(', ')})`
              : null}
          </p>
          <table className="w-full border-collapse text-left">
            <thead>
              <tr className="border-b border-border-default">
                <th className="py-2 pr-4 font-medium">Student</th>
                <th className="py-2 pr-4 font-medium">Peer aggregate</th>
                <th className="py-2 font-medium">Reviews</th>
              </tr>
            </thead>
            <tbody>
              {summary.submissions.map((s) => (
                <tr key={s.submissionId} className="border-b border-border-subtle">
                  <td className="py-2 pr-4">
                    <EntityLabel name={s.studentLabel} fallback={t('entityLabel.unknownStudent')} />
                  </td>
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
