import { useEffect, useState } from 'react'
import { ReviewForm } from './review-form'
import { ReviewPromptBanner } from './review-prompt-banner'
import { fetchReviewEligibility } from '../../lib/course-reviews-api'
import { fetchMyProgress } from '../../lib/self-paced-api'
import { usePlatformFeatures } from '../../context/platform-features-context'

type CourseReviewPromptProps = {
  courseCode: string
  viewAsStudent: boolean
  courseMode?: string
}

function dismissKey(courseCode: string, milestone: string) {
  return `review-prompt-dismissed:${courseCode}:${milestone}`
}

export function CourseReviewPrompt({
  courseCode,
  viewAsStudent,
  courseMode,
}: CourseReviewPromptProps) {
  const { ffCourseReviews } = usePlatformFeatures()
  const [progressPercent, setProgressPercent] = useState(0)
  const [eligible, setEligible] = useState(false)
  const [hasReview, setHasReview] = useState(false)
  const [showBanner, setShowBanner] = useState(false)
  const [showForm, setShowForm] = useState(false)

  useEffect(() => {
    if (!ffCourseReviews || courseMode !== 'self_paced' || !viewAsStudent || !courseCode) return
    let cancelled = false
    Promise.all([fetchMyProgress(courseCode), fetchReviewEligibility(courseCode)])
      .then(([progress, elig]) => {
        if (cancelled) return
        setProgressPercent(progress.progressPercent)
        setEligible(elig.eligible)
        setHasReview(elig.hasReview)
        const milestone = progress.progressPercent >= 100 ? '100' : '25'
        const shouldPrompt =
          elig.eligible &&
          !elig.hasReview &&
          (progress.progressPercent >= 100 || progress.progressPercent >= 25) &&
          sessionStorage.getItem(dismissKey(courseCode, milestone)) !== '1'
        setShowBanner(shouldPrompt)
      })
      .catch(() => {})
    return () => {
      cancelled = true
    }
  }, [courseCode, courseMode, ffCourseReviews, viewAsStudent])

  if (!ffCourseReviews || courseMode !== 'self_paced' || !viewAsStudent) return null

  const milestone = progressPercent >= 100 ? '100' : '25'

  return (
    <>
      {showBanner && eligible && !hasReview ? (
        <ReviewPromptBanner
          progressPercent={progressPercent}
          onWriteReview={() => {
            setShowBanner(false)
            setShowForm(true)
          }}
          onDismiss={() => {
            sessionStorage.setItem(dismissKey(courseCode, milestone), '1')
            setShowBanner(false)
          }}
        />
      ) : null}
      {showForm ? (
        <ReviewForm
          courseCode={courseCode}
          onClose={() => setShowForm(false)}
          onSubmitted={() => {
            setHasReview(true)
            setShowForm(false)
          }}
        />
      ) : null}
    </>
  )
}
