import { CalendarCheck, Circle, XCircle } from 'lucide-react'
import { formatConferenceSlotTime, type ConferenceSlot } from '../../lib/conferences-api'

type SlotPickerProps = {
  slots: ConferenceSlot[]
  teacherName: string
  myBookedSlotId?: string | null
  onBook: (slot: ConferenceSlot) => void
  onCancel?: (slot: ConferenceSlot) => void
  booking?: boolean
}

function slotLabel(slot: ConferenceSlot, teacherName: string, isMine: boolean): string {
  const time = formatConferenceSlotTime(slot)
  if (slot.status === 'cancelled') {
    return `${time} with ${teacherName} — cancelled`
  }
  if (slot.status === 'booked') {
    if (isMine) return `${time} with ${teacherName} — your booking, press Enter to cancel`
    return `${time} with ${teacherName} — booked`
  }
  return `${time} with ${teacherName} — available, press Enter to book`
}

function SlotStatusIcon({ status, isMine }: { status: ConferenceSlot['status']; isMine: boolean }) {
  if (status === 'open') {
    return <CalendarCheck className="h-4 w-4 text-emerald-600 dark:text-emerald-400" aria-hidden />
  }
  if (status === 'booked' && isMine) {
    return <Circle className="h-4 w-4 fill-sky-500 text-sky-500" aria-hidden />
  }
  if (status === 'booked') {
    return <Circle className="h-4 w-4 fill-neutral-400 text-neutral-400" aria-hidden />
  }
  return <XCircle className="h-4 w-4 text-rose-500" aria-hidden />
}

export function SlotPicker({
  slots,
  teacherName,
  myBookedSlotId,
  onBook,
  onCancel,
  booking = false,
}: SlotPickerProps) {
  if (slots.length === 0) {
    return (
      <p className="text-sm text-neutral-500 dark:text-neutral-400">
        No slots available for this teacher on the selected date.
      </p>
    )
  }

  return (
    <div
      role="grid"
      aria-label={`Available conference slots with ${teacherName}`}
      className="grid gap-2 sm:grid-cols-2 lg:grid-cols-3"
    >
      {slots.map((slot) => {
        const isMine = myBookedSlotId === slot.id
        const canBook = slot.status === 'open' && !booking
        const canCancel = isMine && slot.status === 'booked' && onCancel && !booking

        return (
          <div key={slot.id} role="row" className="contents">
            <button
              type="button"
              role="gridcell"
              disabled={!canBook && !canCancel}
              aria-label={slotLabel(slot, teacherName, isMine)}
              onClick={() => {
                if (canBook) onBook(slot)
                else if (canCancel) onCancel(slot)
              }}
              className={`flex items-center gap-2 rounded-xl border px-3 py-3 text-left text-sm transition-[background-color,color,border-color] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-indigo-500 ${
                isMine
                  ? 'border-sky-300 bg-sky-50 dark:border-sky-700 dark:bg-sky-950/30'
                  : slot.status === 'open'
                    ? 'border-neutral-200 bg-white hover:border-emerald-300 hover:bg-emerald-50/50 dark:border-neutral-700 dark:bg-neutral-900/40 dark:hover:border-emerald-700'
                    : 'border-neutral-200 bg-neutral-50 opacity-80 dark:border-neutral-700 dark:bg-neutral-900/20'
              } ${!canBook && !canCancel ? 'cursor-not-allowed' : ''}`}
            >
              <SlotStatusIcon status={slot.status} isMine={isMine} />
              <span className="min-w-0 flex-1">
                <time dateTime={slot.startAt} className="font-medium text-neutral-900 dark:text-neutral-100">
                  {formatConferenceSlotTime(slot)}
                </time>
                <span className="mt-0.5 block text-xs text-neutral-500 dark:text-neutral-400">
                  {slot.status === 'open' && 'Available'}
                  {slot.status === 'booked' && (isMine ? 'Your booking' : 'Booked')}
                  {slot.status === 'cancelled' && 'Cancelled'}
                </span>
              </span>
            </button>
          </div>
        )
      })}
    </div>
  )
}
