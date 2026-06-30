import { Sparkles } from 'lucide-react'

type ProvenanceBadgeProps = {
  generatedBy?: string | null
  modelId?: string | null
  className?: string
}

/** Shows AI provenance metadata on generated content (plan 19.2 / 19.9). */
export function ProvenanceBadge({ generatedBy, modelId, className = '' }: ProvenanceBadgeProps) {
  if (!generatedBy) return null
  return (
    <span
      className={`inline-flex items-center gap-1 rounded-md border border-amber-500/30 bg-amber-500/10 px-2 py-0.5 text-xs text-amber-800 dark:text-amber-200 ${className}`}
      title={modelId ? `Model: ${modelId}` : undefined}
    >
      <Sparkles className="size-3" aria-hidden />
      AI-assisted ({generatedBy}
      {modelId ? ` · ${modelId}` : ''})
    </span>
  )
}
