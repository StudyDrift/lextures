import { useEffect, useRef } from 'react'
import { useReadingPreferences } from '../../context/reading-preferences-context'

const rulerHeightPx = 60

const rulerBgMap = {
  yellow: 'rgba(255, 248, 0, 0.25)',
  grey:   'rgba(128, 128, 128, 0.20)',
}

export function ReadingRuler() {
  const { prefs } = useReadingPreferences()
  const rulerRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!prefs.rulerEnabled) return
    function onMove(e: MouseEvent) {
      const el = rulerRef.current
      if (!el) return
      const y = e.clientY - rulerHeightPx / 2
      el.style.transform = `translateY(${y}px)`
    }
    window.addEventListener('mousemove', onMove)
    return () => window.removeEventListener('mousemove', onMove)
  }, [prefs.rulerEnabled])

  if (!prefs.rulerEnabled) return null

  return (
    <div
      ref={rulerRef}
      aria-hidden="true"
      style={{
        position: 'fixed',
        top: 0,
        left: 0,
        right: 0,
        height: `${rulerHeightPx}px`,
        background: rulerBgMap[prefs.rulerColor],
        pointerEvents: 'none',
        zIndex: 39,
        willChange: 'transform',
      }}
    />
  )
}
