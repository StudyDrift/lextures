import { ExternalLink, RefreshCw } from 'lucide-react'
import { Button, InlineAlert } from '../ui'
import type { MarketingBuild } from '../../lib/marketing-content-api'
import { siteBuildSummary } from './site-build-summary'

export function SiteStatusStrip({ build, canRebuild, rebuilding, onRebuild }: {
  build: MarketingBuild | null
  canRebuild: boolean
  rebuilding: boolean
  onRebuild: () => void
}) {
  if (!build) return null
  // #791: plain-language status instead of raw build state ("Site build pending").
  const summary = siteBuildSummary(build)
  return (
    <InlineAlert tone={summary.tone}>
      <div className="flex flex-wrap items-center justify-between gap-3">
        <span>
          <strong>{summary.title}</strong>
          {summary.detail ? ` · ${summary.detail}` : null}
        </span>
        <span className="flex items-center gap-2">
          {build.runUrl ? <a className="inline-flex min-h-6 items-center gap-1 text-accent-fg underline" href={build.runUrl} target="_blank" rel="noreferrer">View run <ExternalLink className="h-3.5 w-3.5" /></a> : null}
          {canRebuild ? <Button size="sm" variant="secondary" className="min-h-6" loading={rebuilding} onClick={onRebuild}><RefreshCw className="h-4 w-4" /> Rebuild site</Button> : null}
        </span>
      </div>
    </InlineAlert>
  )
}
