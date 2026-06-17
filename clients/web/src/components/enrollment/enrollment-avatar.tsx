import { useEffect, useLayoutEffect, useRef, useState } from 'react'
import { createPortal } from 'react-dom'

function hueFromString(s: string): number {
  let h = 0
  for (let i = 0; i < s.length; i++) h = (h * 31 + s.charCodeAt(i)) | 0
  return Math.abs(h) % 360
}

function displayInitials(label: string): string {
  const t = label.trim()
  if (!t) return '?'
  const parts = t.split(/\s+/).filter(Boolean)
  if (parts.length >= 2) {
    const a = parts[0][0] ?? ''
    const b = parts[1][0] ?? ''
    return (a + b).toUpperCase() || '?'
  }
  return t.slice(0, 2).toUpperCase() || '?'
}

export function EnrollmentAvatar({
  userId,
  name,
  avatarUrl,
  size = 'sm',
  showPreview = true,
}: {
  userId: string
  name: string
  avatarUrl?: string | null
  size?: 'sm' | 'md'
  /** When false, hover does not open the enlarged preview (e.g. top-bar menu trigger). */
  showPreview?: boolean
}) {
  const label = name.trim() || '—'
  const resolvedAvatarUrl = avatarUrl?.trim() || ''
  const [imageError, setImageError] = useState(false)
  const [previewOpen, setPreviewOpen] = useState(false)
  const [previewPos, setPreviewPos] = useState<{ top: number; left: number } | null>(null)
  const avatarRef = useRef<HTMLSpanElement>(null)
  const dim = size === 'md' ? 'h-10 w-10 text-sm' : 'h-8 w-8 text-xs'

  useEffect(() => {
    setImageError(false)
  }, [resolvedAvatarUrl])

  useLayoutEffect(() => {
    if (!previewOpen || !avatarRef.current) {
      return
    }
    const measure = () => {
      if (!avatarRef.current) return
      const rect = avatarRef.current.getBoundingClientRect()
      setPreviewPos({
        top: rect.top + rect.height / 2,
        left: rect.right + 10,
      })
    }
    measure()
    window.addEventListener('scroll', measure, true)
    window.addEventListener('resize', measure)
    return () => {
      window.removeEventListener('scroll', measure, true)
      window.removeEventListener('resize', measure)
    }
  }, [previewOpen])

  const initialsAvatar = (() => {
    const h = hueFromString(userId.toLowerCase())
    const h2 = (h + 48) % 360
    return (
      <div
        className={`flex shrink-0 select-none items-center justify-center rounded-full font-semibold text-white shadow-sm ring-2 ring-white dark:ring-neutral-950 ${dim}`}
        style={{ background: `linear-gradient(145deg, hsl(${h} 58% 48%), hsl(${h2} 52% 40%))` }}
        aria-hidden
      >
        {displayInitials(label)}
      </div>
    )
  })()

  if (!resolvedAvatarUrl || imageError) {
    return initialsAvatar
  }

  const openPreview = () => {
    if (showPreview) setPreviewOpen(true)
  }
  const hidePreview = () => {
    setPreviewOpen(false)
    setPreviewPos(null)
  }

  return (
    <span
      ref={avatarRef}
      className="inline-flex shrink-0"
      onMouseEnter={openPreview}
      onMouseLeave={hidePreview}
      onFocusCapture={openPreview}
      onBlurCapture={hidePreview}
    >
      <img
        src={resolvedAvatarUrl}
        alt=""
        className={`rounded-full object-cover shadow-sm ring-2 ring-white dark:ring-neutral-950 ${dim}`}
        onError={() => setImageError(true)}
      />
      {showPreview && previewOpen && previewPos
        ? createPortal(
            <div
              role="tooltip"
              aria-label={label}
              style={{
                top: previewPos.top,
                left: previewPos.left,
                transform: 'translateY(-50%)',
              }}
              className="pointer-events-none fixed z-[200] w-36 overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-2xl ring-1 ring-slate-900/5 dark:border-neutral-700 dark:bg-neutral-900 dark:ring-white/10"
            >
              <img
                src={resolvedAvatarUrl}
                alt=""
                className="h-32 w-full object-cover"
              />
              <p className="truncate px-3 py-2 text-center text-sm font-medium text-slate-900 dark:text-neutral-100">
                {label}
              </p>
            </div>,
            document.body,
          )
        : null}
    </span>
  )
}