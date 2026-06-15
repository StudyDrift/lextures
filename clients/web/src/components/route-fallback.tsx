/** Minimal suspense fallback for lazy-loaded routes. */
export function RouteFallback() {
  return (
    <div className="flex min-h-[12rem] items-center justify-center px-4 py-8" aria-busy="true">
      <span className="sr-only">Loading page.</span>
      <div
        className="h-8 w-8 animate-spin rounded-full border-2 border-indigo-200 border-t-indigo-600 dark:border-neutral-700 dark:border-t-indigo-400"
        aria-hidden
      />
    </div>
  )
}
