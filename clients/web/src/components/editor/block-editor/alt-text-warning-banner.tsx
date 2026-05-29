import { LiveRegion } from '../../a11y/live-region'
import { formatCoverageLabel, type AltTextCoverage } from '../../../lib/image-alt-validation'

type AltTextWarningBannerProps = {
  coverage: AltTextCoverage
  hardBlock?: boolean
}

export function AltTextWarningBanner({ coverage, hardBlock }: AltTextWarningBannerProps) {
  if (coverage.missing.length === 0) return null
  const label = formatCoverageLabel(coverage.withAlt, coverage.total)
  const message =
    coverage.missing.length === 1
      ? '1 image is missing alt text.'
      : `${coverage.missing.length} images are missing alt text.`

  return (
    <div
      role="alert"
      className="mb-3 rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 text-sm text-amber-950 dark:border-amber-900/50 dark:bg-amber-950/30 dark:text-amber-100"
    >
      <LiveRegion politeness="assertive">{`${message} Coverage: ${label}.`}</LiveRegion>
      <p className="font-medium">{message}</p>
      <p className="mt-0.5 text-xs text-amber-800 dark:text-amber-200/90">
        Coverage: {label}.
        {hardBlock
          ? ' Save is disabled until every image has alt text or is marked decorative.'
          : ' Add alt text or mark images as decorative before publishing.'}
      </p>
    </div>
  )
}
