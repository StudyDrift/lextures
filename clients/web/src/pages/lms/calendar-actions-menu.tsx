import { useEffect, useId, useRef, useState } from 'react'
import { ChevronDown, Download } from 'lucide-react'
import { usePlatformFeatures } from '../../context/platform-features-context'
import {
  downloadCourseCalendarFeed,
  downloadPersonalCalendarFeed,
} from '../../lib/calendar-feed-api'
import { toastMutationError } from '../../lib/lms-toast'

type Props =
  | { scope: 'global' }
  | { scope: 'course'; courseCode: string }

export function CalendarActionsMenu(props: Props) {
  const { ffCalendarFeeds, loading: featuresLoading } = usePlatformFeatures()
  const [open, setOpen] = useState(false)
  const [downloading, setDownloading] = useState(false)
  const rootRef = useRef<HTMLDivElement>(null)
  const menuId = useId()

  useEffect(() => {
    if (!open) return
    function onDoc(e: MouseEvent) {
      if (!rootRef.current?.contains(e.target as Node)) setOpen(false)
    }
    function onKey(e: KeyboardEvent) {
      if (e.key === 'Escape') setOpen(false)
    }
    document.addEventListener('mousedown', onDoc)
    document.addEventListener('keydown', onKey)
    return () => {
      document.removeEventListener('mousedown', onDoc)
      document.removeEventListener('keydown', onKey)
    }
  }, [open])

  if (featuresLoading || !ffCalendarFeeds) return null

  async function handleDownloadFeed() {
    setDownloading(true)
    try {
      if (props.scope === 'course') {
        await downloadCourseCalendarFeed(props.courseCode)
      } else {
        await downloadPersonalCalendarFeed()
      }
      setOpen(false)
    } catch {
      toastMutationError('Could not download calendar feed.')
    } finally {
      setDownloading(false)
    }
  }

  const downloadLabel =
    props.scope === 'course' ? 'Download course calendar feed' : 'Download calendar feed'
  const downloadHint =
    props.scope === 'course'
      ? 'Save this course’s due dates as an .ics file'
      : 'Save all enrolled course due dates as an .ics file'

  return (
    <div ref={rootRef} className="relative block w-full text-start sm:inline-block sm:w-auto">
      <button
        type="button"
        aria-haspopup="menu"
        aria-expanded={open}
        aria-controls={open ? menuId : undefined}
        onClick={() => setOpen((o) => !o)}
        className="inline-flex w-full items-center justify-center gap-2 rounded-xl border border-slate-200 bg-white px-4 py-2.5 text-sm font-semibold text-slate-800 shadow-sm transition hover:border-slate-300 hover:bg-slate-50 dark:border-neutral-700 dark:bg-neutral-900 dark:text-neutral-100 dark:hover:border-neutral-600 dark:hover:bg-neutral-800 sm:w-auto"
      >
        <span>Actions</span>
        <ChevronDown
          className={`h-4 w-4 shrink-0 transition ${open ? 'rotate-180' : ''}`}
          aria-hidden
        />
      </button>

      {open && (
        <div
          id={menuId}
          role="menu"
          aria-label="Calendar actions"
          className="absolute start-0 end-0 z-50 mt-1 min-w-0 overflow-hidden rounded-xl border border-slate-200 bg-white py-1 shadow-lg shadow-slate-900/10 sm:left-auto sm:end-0 sm:min-w-[16rem] dark:border-neutral-600 dark:bg-neutral-800 dark:shadow-black/40"
        >
          <button
            type="button"
            role="menuitem"
            disabled={downloading}
            onClick={() => void handleDownloadFeed()}
            className="flex w-full items-start gap-2.5 px-2.5 py-2 text-start text-sm transition hover:bg-slate-50 disabled:cursor-not-allowed disabled:opacity-60 dark:hover:bg-neutral-700"
          >
            <Download className="mt-0.5 h-4 w-4 shrink-0 text-slate-500 dark:text-neutral-400" aria-hidden />
            <span className="flex min-w-0 flex-col gap-0.5">
              <span className="font-semibold text-slate-950 dark:text-neutral-100">{downloadLabel}</span>
              <span className="text-xs text-slate-500 dark:text-neutral-400">{downloadHint}</span>
            </span>
          </button>
        </div>
      )}
    </div>
  )
}
