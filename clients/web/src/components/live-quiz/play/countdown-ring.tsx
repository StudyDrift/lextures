import { useEffect, useState } from 'react'
import { secondsUntilDeadline } from '../../../lib/live-quiz-countdown'

export function CountdownRing({
  deadline,
  timeLimitSeconds,
  offsetMs = 0,
}: {
  deadline?: string
  timeLimitSeconds?: number
  offsetMs?: number
}) {
  const [left, setLeft] = useState<number | null>(null)

  useEffect(() => {
    if (!deadline) {
      setLeft(null)
      return
    }
    const tick = () => setLeft(secondsUntilDeadline(deadline, offsetMs))
    tick()
    const id = setInterval(tick, 250)
    return () => clearInterval(id)
  }, [deadline, offsetMs])

  if (left == null) return null
  const total = Math.max(1, timeLimitSeconds ?? 20)
  const pct = Math.max(0, Math.min(1, left / total))
  const urgent = left <= 5

  return (
    <div
      className="flex flex-col items-center gap-1"
      aria-live={urgent ? 'assertive' : 'polite'}
      aria-atomic="true"
    >
      <div
        className="relative grid h-16 w-16 place-items-center rounded-full border-4 border-slate-200 dark:border-neutral-700"
        style={{
          background: `conic-gradient(currentColor ${pct * 360}deg, transparent 0)`,
          color: urgent ? '#dc2626' : '#4f46e5',
        }}
      >
        <span className="grid h-12 w-12 place-items-center rounded-full bg-white text-xl font-bold tabular-nums text-slate-900 dark:bg-neutral-950 dark:text-neutral-50">
          {left}
        </span>
      </div>
      <span className="sr-only">{left} seconds remaining</span>
    </div>
  )
}
