import { Check } from 'lucide-react'

/** Green completion disc shown on a module the learner has finished. */
export function ModuleCompleteMark() {
  return (
    <span
      className="pointer-events-none absolute end-3 top-3 z-10 flex h-8 w-8 items-center justify-center rounded-full bg-success-fg text-fg-inverse shadow-sm"
      role="img"
      aria-label="Module complete"
      data-testid="module-complete-mark"
    >
      <Check className="h-4 w-4" strokeWidth={3} aria-hidden />
    </span>
  )
}
