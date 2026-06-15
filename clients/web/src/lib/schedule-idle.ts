/** Run work after first paint when the browser is idle (or after a short timeout). */
export function scheduleIdleTask(task: () => void, timeoutMs = 2000): () => void {
  let cancelled = false
  const run = () => {
    if (!cancelled) task()
  }

  if (typeof requestIdleCallback === 'function') {
    const id = requestIdleCallback(run, { timeout: timeoutMs })
    return () => {
      cancelled = true
      cancelIdleCallback(id)
    }
  }

  const id = window.setTimeout(run, Math.min(timeoutMs, 250))
  return () => {
    cancelled = true
    window.clearTimeout(id)
  }
}
