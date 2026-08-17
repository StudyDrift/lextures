import { ChevronDown, Sparkles } from 'lucide-react'
import { Badge, Button, InlineAlert } from '../../ui'
import type { MarketingFinding } from '../../../lib/marketing-content-api'
import { formatQualityScore, isBlockingFinding, scoreBarClass, scoreMeterPercent, scoreToneClass } from './article-editor-utils'
import { bodyHasDirective, directiveTemplateForFinding, findingKey, findingLocationLabel } from './article-finding-nav'

type Props = {
  findings: MarketingFinding[]
  score: number | null
  validating: boolean
  open: boolean
  onOpenChange: (open: boolean) => void
  bodyMd: string
  onSelectFinding: (finding: MarketingFinding) => void
  onInsertTemplate: (markdown: string) => void
  canSolve?: boolean
  solving?: boolean
  solvingFindingKey?: string | null
  solveProgress?: string
  solveError?: string
  onSolveWithAI?: () => void
}

export function ArticleFindingsBar({
  findings,
  score,
  validating,
  open,
  onOpenChange,
  bodyMd,
  onSelectFinding,
  onInsertTemplate,
  canSolve = false,
  solving = false,
  solvingFindingKey = null,
  solveProgress = '',
  solveError = '',
  onSolveWithAI,
}: Props) {
  const blocking = findings.filter((finding) => isBlockingFinding(finding.severity))
  const warnings = findings.filter((finding) => !isBlockingFinding(finding.severity))
  const scoreTone = scoreToneClass(score)
  const scoreBar = scoreBarClass(score)
  const scoreWidth = scoreMeterPercent(score)

  const showSolve = Boolean(canSolve && onSolveWithAI && findings.length)

  return (
    <section id="article-findings" aria-live="polite" className="sticky bottom-0 z-20 border-t border-border-default bg-surface-raised/95 backdrop-blur supports-[backdrop-filter]:bg-surface-raised/80">
      <div className="flex items-center gap-2 px-4 py-2 sm:px-6">
        <Button type="button" variant="ghost" aria-expanded={open} onClick={() => onOpenChange(!open)} className="h-auto min-h-6 min-w-0 flex-1 items-center justify-start gap-3 rounded-lg px-0 py-0.5 text-start hover:bg-transparent">
          <span className="text-xs font-medium text-fg-muted">Quality</span>
          <span aria-hidden className="h-1.5 w-24 overflow-hidden rounded-full bg-surface-sunken sm:w-40">
            <span className={`block h-full rounded-full motion-safe:transition-[width] ${scoreBar}`} style={{ width: `${scoreWidth}%` }} />
          </span>
          <span className={`text-sm font-semibold tabular-nums ${scoreTone}`}>{validating ? 'checking…' : formatQualityScore(score)}</span>
          <span className="hidden text-xs text-fg-muted sm:inline">/ 8.0 floor</span>
          <span className="ms-auto flex items-center gap-2">
            {blocking.length ? <Badge tone="danger">{blocking.length} blocking</Badge> : null}
            {warnings.length ? <Badge tone="warning">{warnings.length} suggestion{warnings.length === 1 ? '' : 's'}</Badge> : null}
            {!findings.length ? <span className="text-xs text-fg-muted">{score == null ? 'Not checked yet' : 'No findings'}</span> : null}
            <ChevronDown aria-hidden className={`h-4 w-4 text-fg-muted motion-safe:transition-transform ${open ? '' : 'rotate-180'}`} />
          </span>
        </Button>
        {showSolve ? (
          <Button type="button" size="sm" variant="secondary" className="min-h-6 shrink-0" loading={solving} disabled={solving} onClick={() => onSolveWithAI?.()}>
            <Sparkles className="h-3.5 w-3.5" aria-hidden />
            Solve with AI
          </Button>
        ) : null}
      </div>
      {open ? (
        <div className="max-h-64 overflow-y-auto border-t border-border-subtle px-4 py-3 sm:px-6">
          {solveError ? <InlineAlert tone="danger" className="mb-3">{solveError}</InlineAlert> : null}
          {solving ? <p role="status" className="mb-2 text-xs text-fg-muted">{solveProgress || `Solving ${findings.length} finding${findings.length === 1 ? '' : 's'} with AI…`}</p> : null}
          {findings.length ? (
            <ul className="space-y-1.5">
              {findings.map((finding, index) => {
                const key = findingKey(finding, index)
                const location = findingLocationLabel(finding)
                const template = directiveTemplateForFinding(finding.rule)
                const canInsert = Boolean(template && !bodyHasDirective(bodyMd, finding.rule))
                const label = [isBlockingFinding(finding.severity) ? 'Error' : 'Suggestion', finding.message, location].filter(Boolean).join('. ')
                const active = solving && solvingFindingKey === key
                return (
                  <li key={key} className={`flex flex-wrap items-start gap-2 ${active ? 'rounded-lg bg-accent-surface' : ''}`}>
                    <Button
                      type="button"
                      variant="ghost"
                      className="h-auto min-h-8 flex-1 items-start justify-start gap-2 whitespace-normal rounded-lg px-2 py-1.5 text-start"
                      onClick={() => onSelectFinding(finding)}
                      aria-label={`${label}. Jump to this finding.`}
                    >
                      <Badge tone={isBlockingFinding(finding.severity) ? 'danger' : 'warning'}>{isBlockingFinding(finding.severity) ? 'Error' : 'Suggestion'}</Badge>
                      <span className="min-w-0 flex-1">
                        <span className="block text-sm text-fg-default">{finding.message || finding.rule}</span>
                        <span className="mt-0.5 block font-mono text-xs text-fg-muted">
                          {finding.rule}
                          {location ? ` · ${location}` : ''}
                        </span>
                      </span>
                    </Button>
                    {active ? <Badge tone="info">Solving</Badge> : null}
                    {canInsert && template && !solving ? (
                      <Button type="button" size="sm" variant="secondary" className="shrink-0" onClick={() => onInsertTemplate(template)}>
                        Insert block
                      </Button>
                    ) : null}
                  </li>
                )
              })}
            </ul>
          ) : (
            <p className="text-sm text-fg-muted">Nothing to fix. Findings appear here as you write.</p>
          )}
        </div>
      ) : null}
    </section>
  )
}
