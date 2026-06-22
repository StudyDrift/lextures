import { useCallback, useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import {
  fetchPeerReviewAssigned,
  postPeerReviewSubmit,
  type PeerReviewAllocation,
} from '../../lib/peer-review-api'
import { usePlatformFeatures } from '../../context/platform-features-context'
import { LmsPage } from './lms-page'

export default function PeerReviewsPage() {
  const { ffPeerReview } = usePlatformFeatures()
  const [allocations, setAllocations] = useState<PeerReviewAllocation[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [scoreDraft, setScoreDraft] = useState<Record<string, string>>({})
  const [commentDraft, setCommentDraft] = useState<Record<string, string>>({})
  const [busyId, setBusyId] = useState<string | null>(null)

  const load = useCallback(async () => {
    if (!ffPeerReview) {
      setLoading(false)
      return
    }
    setLoading(true)
    setError(null)
    try {
      setAllocations(await fetchPeerReviewAssigned())
    } catch (e) {
      setAllocations([])
      setError(e instanceof Error ? e.message : 'Could not load peer reviews.')
    } finally {
      setLoading(false)
    }
  }, [ffPeerReview])

  useEffect(() => {
    void load()
  }, [load])

  async function submitReview(alloc: PeerReviewAllocation) {
    setBusyId(alloc.id)
    setError(null)
    try {
      const scoreText = scoreDraft[alloc.id]?.trim()
      const score = scoreText ? Number(scoreText) : undefined
      await postPeerReviewSubmit(alloc.id, {
        score: Number.isFinite(score) ? score : undefined,
        comments: commentDraft[alloc.id]?.trim() || undefined,
      })
      await load()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Could not submit review.')
    } finally {
      setBusyId(null)
    }
  }

  if (!ffPeerReview) {
    return (
      <LmsPage title="Peer reviews">
        <p className="text-sm text-slate-500 dark:text-neutral-400">
          Peer review is not enabled on this platform.
        </p>
      </LmsPage>
    )
  }

  const pending = allocations.filter((a) => a.status !== 'submitted')

  return (
    <LmsPage title="Peer reviews to complete">
      {loading ? (
        <p className="text-sm text-slate-500 dark:text-neutral-400">Loading…</p>
      ) : error ? (
        <p className="text-sm text-red-600 dark:text-red-400" role="alert">
          {error}
        </p>
      ) : pending.length === 0 ? (
        <p className="text-sm text-slate-500 dark:text-neutral-400">
          You have no peer reviews to complete right now.
        </p>
      ) : (
        <ul className="space-y-4">
          {pending.map((alloc) => (
            <li
              key={alloc.id}
              className="rounded-lg border border-slate-200 p-4 dark:border-neutral-700"
            >
              <div className="flex flex-wrap items-center justify-between gap-2">
                <div>
                  <p className="text-sm font-medium text-slate-900 dark:text-neutral-100">
                    Review {alloc.targetLabel ?? 'peer submission'}
                  </p>
                  <p className="text-xs text-slate-500 dark:text-neutral-400">
                    Status: {alloc.status} · Assigned {new Date(alloc.assignedAt).toLocaleString()}
                  </p>
                </div>
                <Link
                  to={`/courses/${alloc.courseCode}/assignments/${alloc.assignmentId}`}
                  className="text-xs text-indigo-600 hover:underline dark:text-indigo-400"
                >
                  Open assignment
                </Link>
              </div>
              <div className="mt-3 grid gap-2 sm:grid-cols-2">
                <label className="block text-xs text-slate-600 dark:text-neutral-400">
                  Score
                  <input
                    type="number"
                    min={0}
                    className="mt-1 w-full rounded border border-slate-200 px-2 py-1 text-sm dark:border-neutral-600 dark:bg-neutral-950"
                    value={scoreDraft[alloc.id] ?? ''}
                    onChange={(e) =>
                      setScoreDraft((prev) => ({ ...prev, [alloc.id]: e.target.value }))
                    }
                  />
                </label>
                <label className="block text-xs text-slate-600 dark:text-neutral-400 sm:col-span-2">
                  Comments
                  <textarea
                    rows={3}
                    className="mt-1 w-full rounded border border-slate-200 px-2 py-1 text-sm dark:border-neutral-600 dark:bg-neutral-950"
                    value={commentDraft[alloc.id] ?? ''}
                    onChange={(e) =>
                      setCommentDraft((prev) => ({ ...prev, [alloc.id]: e.target.value }))
                    }
                  />
                </label>
              </div>
              <button
                type="button"
                disabled={busyId === alloc.id}
                onClick={() => void submitReview(alloc)}
                className="mt-3 rounded bg-indigo-600 px-3 py-1.5 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-60"
              >
                {busyId === alloc.id ? 'Submitting…' : 'Submit review'}
              </button>
            </li>
          ))}
        </ul>
      )}
    </LmsPage>
  )
}
