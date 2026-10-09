import { ErrorState, Skeleton } from '../../components/ui'
import { LmsPage } from './lms-page'

const ERROR_TITLE = "Couldn't load this page"

type CoursePageLoadStateProps = {
  /** Page title shown while loading or after a failure. */
  title: string
  /** When set, the page failed to load: show this message with a Retry action. */
  error?: string | null
  onRetry?: () => void
}

/**
 * Placeholder for course pages whose first request has not settled yet. Shows skeletons while
 * waiting and an error with Retry when the request failed or timed out, so a stalled API never
 * leaves a blank screen.
 */
export function CoursePageLoadState({ title, error, onRetry }: CoursePageLoadStateProps) {
  if (error) {
    return (
      <LmsPage title={title}>
        <div className="mt-6" role="alert">
          <ErrorState
            title={ERROR_TITLE}
            body={error}
            primaryAction={onRetry ? { label: 'Retry', onClick: onRetry } : undefined}
          />
        </div>
      </LmsPage>
    )
  }
  return (
    <LmsPage title={title}>
      <div className="mt-6 space-y-3" aria-busy="true">
        <Skeleton label="Loading" className="h-8 w-48" />
        <Skeleton className="h-24 w-full" />
        <Skeleton className="h-24 w-full" />
      </div>
    </LmsPage>
  )
}
