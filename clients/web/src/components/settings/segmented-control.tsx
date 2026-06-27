type Option<T extends string> = {
  value: T
  label: string
}

type Props<T extends string> = {
  value: T
  options: Option<T>[]
  onChange: (value: T) => void
  'aria-label'?: string
}

export function SegmentedControl<T extends string>({
  value,
  options,
  onChange,
  'aria-label': ariaLabel,
}: Props<T>) {
  return (
    <div
      role="group"
      aria-label={ariaLabel}
      className="inline-flex rounded-xl border border-slate-200 bg-slate-50 p-1 dark:border-neutral-600 dark:bg-neutral-800/50"
    >
      {options.map((opt) => (
        <button
          key={opt.value}
          type="button"
          aria-pressed={value === opt.value}
          onClick={() => onChange(opt.value)}
          className={`rounded-lg px-4 py-2 text-sm font-medium transition-[background-color,color,border-color] ${
            value === opt.value
              ? 'bg-white text-slate-900 shadow-sm dark:bg-neutral-600 dark:text-neutral-50 dark:shadow-md dark:ring-1 dark:ring-inset dark:ring-white/10'
              : 'text-slate-600 hover:text-slate-900 dark:text-neutral-400 dark:hover:text-neutral-200'
          }`}
        >
          {opt.label}
        </button>
      ))}
    </div>
  )
}