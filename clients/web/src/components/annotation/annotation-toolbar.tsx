export type AnnotationTool = 'select' | 'highlight' | 'draw' | 'pin' | 'text'

const PRESET_COLOURS = ['#FFFF00', '#90EE90', '#FF6B6B', '#6BCBFF'] as const

export type AnnotationToolbarProps = {
  tool: AnnotationTool
  onToolChange: (t: AnnotationTool) => void
  colour: string
  onColourChange: (c: string) => void
  disabled?: boolean
  readOnly?: boolean
  /**
   * `anchor` is for reflowable docs (text/code/office): highlight-by-text-selection only, so
   * the geometric tool buttons are replaced with a hint and just the colour picker is shown.
   */
  variant?: 'full' | 'anchor'
}

export function AnnotationToolbar({
  tool,
  onToolChange,
  colour,
  onColourChange,
  disabled,
  readOnly,
  variant = 'full',
}: AnnotationToolbarProps) {
  if (readOnly) return null

  if (variant === 'anchor') {
    return (
      <div
        role="toolbar"
        aria-label="Annotation tools"
        className="flex flex-wrap items-center gap-2 rounded-2xl bg-white/90 p-2 shadow-card dark:bg-neutral-900/90"
      >
        <span className="px-1 text-xs font-medium text-slate-600 dark:text-neutral-300">
          Select text to highlight, then add a comment.
        </span>
        <span className="mx-1 h-6 w-px bg-slate-200 dark:bg-neutral-700" aria-hidden />
        {PRESET_COLOURS.map((c) => (
          <button
            key={c}
            type="button"
            disabled={disabled}
            aria-label={`Colour ${c}`}
            title={c}
            onClick={() => onColourChange(c)}
            className={`h-8 w-8 rounded-full border-2 focus:outline-none focus-visible:ring-2 focus-visible:ring-indigo-500 disabled:opacity-50 ${
              colour.toUpperCase() === c.toUpperCase() ? 'border-indigo-600' : 'border-slate-300 dark:border-neutral-600'
            }`}
            style={{ backgroundColor: c }}
          />
        ))}
      </div>
    )
  }

  const tools: { id: AnnotationTool; label: string; hint: string }[] = [
    { id: 'select', label: 'Select', hint: 'Select and scroll' },
    { id: 'highlight', label: 'Highlight', hint: 'Drag to highlight' },
    { id: 'draw', label: 'Draw', hint: 'Freehand stroke' },
    { id: 'pin', label: 'Pin', hint: 'Point comment' },
    { id: 'text', label: 'Text', hint: 'Text box region' },
  ]

  return (
    <div
      role="toolbar"
      aria-label="Annotation tools"
      className="flex flex-wrap items-center gap-2 rounded-2xl bg-white/90 p-2 shadow-card dark:bg-neutral-900/90"
    >
      {tools.map((t) => (
        <button
          key={t.id}
          type="button"
          disabled={disabled}
          aria-pressed={tool === t.id}
          aria-label={t.hint}
          onClick={() => onToolChange(t.id)}
          className={`rounded-lg px-3 py-1.5 text-xs font-semibold transition-[background-color,color,border-color] focus:outline-none focus-visible:ring-2 focus-visible:ring-indigo-500 disabled:opacity-50 ${
            tool === t.id
              ? 'bg-indigo-600 text-white'
              : 'bg-slate-100 text-slate-800 hover:bg-slate-200 dark:bg-neutral-800 dark:text-neutral-100 dark:hover:bg-neutral-700'
          }`}
        >
          {t.label}
        </button>
      ))}
      <span className="mx-1 h-6 w-px bg-slate-200 dark:bg-neutral-700" aria-hidden />
      {PRESET_COLOURS.map((c) => (
        <button
          key={c}
          type="button"
          disabled={disabled}
          aria-label={`Colour ${c}`}
          title={c}
          onClick={() => onColourChange(c)}
          className={`h-8 w-8 rounded-full border-2 focus:outline-none focus-visible:ring-2 focus-visible:ring-indigo-500 disabled:opacity-50 ${
            colour.toUpperCase() === c.toUpperCase() ? 'border-indigo-600' : 'border-slate-300 dark:border-neutral-600'
          }`}
          style={{ backgroundColor: c }}
        />
      ))}
    </div>
  )
}
