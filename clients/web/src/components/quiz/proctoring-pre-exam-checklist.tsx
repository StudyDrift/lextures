import { useCallback, useEffect, useId, useRef, useState } from 'react'
import { Camera, Check, Mic, MonitorCheck, ShieldCheck, X } from 'lucide-react'
import type { ProctoringVendor } from '../../lib/courses-api'

type CheckStep = {
  id: string
  label: string
  description: string
  icon: React.ReactNode
  status: 'idle' | 'checking' | 'pass' | 'fail'
}

export type ProctoringPreExamChecklistProps = {
  open: boolean
  vendor: ProctoringVendor
  required: boolean
  onProceed: () => void
  onClose: () => void
}

const VENDOR_LABELS: Record<ProctoringVendor, string> = {
  honorlock: 'Honorlock',
  respondus: 'Respondus Monitor',
  proctu: 'ProctorU',
  examity: 'Examity',
}

function useMediaCheck(kind: 'camera' | 'microphone'): 'idle' | 'checking' | 'pass' | 'fail' {
  const [status, setStatus] = useState<'idle' | 'checking' | 'pass' | 'fail'>('idle')
  useEffect(() => {
    setStatus('checking')
    navigator.mediaDevices
      .getUserMedia({ video: kind === 'camera', audio: kind === 'microphone' })
      .then((stream) => {
        stream.getTracks().forEach((t) => t.stop())
        setStatus('pass')
      })
      .catch(() => setStatus('fail'))
  }, [kind])
  return status
}

function useExtensionCheck(vendor: ProctoringVendor): 'idle' | 'checking' | 'pass' | 'fail' {
  const [status, setStatus] = useState<'idle' | 'checking' | 'pass' | 'fail'>('idle')
  useEffect(() => {
    setStatus('checking')
    // Check for a vendor-injected DOM marker; extensions set window.__proctoring_<vendor> = true.
    const key = `__proctoring_${vendor}` as keyof Window
    const timer = setTimeout(() => {
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      const detected = Boolean((window as unknown as Record<string, unknown>)[key as string])
      setStatus(detected ? 'pass' : 'fail')
    }, 800)
    return () => clearTimeout(timer)
  }, [vendor])
  return status
}

function StepIcon({ status, children }: { status: CheckStep['status']; children: React.ReactNode }) {
  if (status === 'pass')
    return (
      <span className="flex h-8 w-8 items-center justify-center rounded-full bg-emerald-100 text-emerald-700 dark:bg-emerald-900/40 dark:text-emerald-400">
        <Check className="h-4 w-4" aria-hidden />
      </span>
    )
  if (status === 'fail')
    return (
      <span className="flex h-8 w-8 items-center justify-center rounded-full bg-rose-100 text-rose-700 dark:bg-rose-900/40 dark:text-rose-400">
        <X className="h-4 w-4" aria-hidden />
      </span>
    )
  if (status === 'checking')
    return (
      <span className="flex h-8 w-8 items-center justify-center rounded-full bg-slate-100 dark:bg-neutral-800">
        <span className="h-4 w-4 motion-safe:animate-spin rounded-full border-2 border-slate-300 border-t-indigo-500" aria-hidden />
      </span>
    )
  return (
    <span className="flex h-8 w-8 items-center justify-center rounded-full bg-slate-100 text-slate-500 dark:bg-neutral-800 dark:text-neutral-400">
      {children}
    </span>
  )
}

export function ProctoringPreExamChecklist({
  open,
  vendor,
  required,
  onProceed,
  onClose,
}: ProctoringPreExamChecklistProps) {
  if (!open) return null
  return (
    <ProctoringPreExamChecklistInner
      vendor={vendor}
      required={required}
      onProceed={onProceed}
      onClose={onClose}
    />
  )
}

function ProctoringPreExamChecklistInner({
  vendor,
  required,
  onProceed,
  onClose,
}: Omit<ProctoringPreExamChecklistProps, 'open'>) {
  const titleId = useId()
  const cameraStatus = useMediaCheck('camera')
  const micStatus = useMediaCheck('microphone')
  const extensionStatus = useExtensionCheck(vendor)
  const proceedRef = useRef<HTMLButtonElement>(null)

  const allPass = cameraStatus === 'pass' && micStatus === 'pass' && extensionStatus === 'pass'
  const anyFail = cameraStatus === 'fail' || micStatus === 'fail' || extensionStatus === 'fail'
  const canProceed = allPass || (!required && anyFail)

  const steps: CheckStep[] = [
    {
      id: 'camera',
      label: 'Camera',
      description:
        cameraStatus === 'pass'
          ? 'Camera detected and accessible.'
          : cameraStatus === 'fail'
            ? 'Could not access camera. Grant permission in your browser.'
            : 'Checking camera access…',
      icon: <Camera className="h-4 w-4" aria-hidden />,
      status: cameraStatus,
    },
    {
      id: 'microphone',
      label: 'Microphone',
      description:
        micStatus === 'pass'
          ? 'Microphone detected and accessible.'
          : micStatus === 'fail'
            ? 'Could not access microphone. Grant permission in your browser.'
            : 'Checking microphone access…',
      icon: <Mic className="h-4 w-4" aria-hidden />,
      status: micStatus,
    },
    {
      id: 'extension',
      label: `${VENDOR_LABELS[vendor]} Extension`,
      description:
        extensionStatus === 'pass'
          ? `${VENDOR_LABELS[vendor]} browser extension detected.`
          : extensionStatus === 'fail'
            ? `${VENDOR_LABELS[vendor]} extension not detected. Install it from the Chrome Web Store.`
            : `Looking for ${VENDOR_LABELS[vendor]} extension…`,
      icon: <MonitorCheck className="h-4 w-4" aria-hidden />,
      status: extensionStatus,
    },
  ]

  const handleProceed = useCallback(() => {
    onProceed()
  }, [onProceed])

  useEffect(() => {
    if (canProceed) proceedRef.current?.focus()
  }, [canProceed])

  useEffect(() => {
    function onKey(e: KeyboardEvent) {
      if (e.key === 'Escape') onClose()
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [onClose])

  const completedCount = steps.filter((s) => s.status === 'pass').length

  return (
    <div
      className="fixed inset-0 z-50 flex items-end justify-center bg-slate-900/50 p-4 sm:items-center"
      role="dialog"
      aria-modal="true"
      aria-labelledby={titleId}
    >
      <div className="w-full max-w-md overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-2xl dark:border-neutral-700 dark:bg-neutral-900">
        {/* Header */}
        <div className="flex items-center gap-3 border-b border-slate-200 px-5 py-4 dark:border-neutral-700">
          <ShieldCheck className="h-5 w-5 shrink-0 text-indigo-600 dark:text-indigo-400" aria-hidden />
          <h2 id={titleId} className="flex-1 text-sm font-semibold text-slate-900 dark:text-neutral-100">
            Proctored Exam — {VENDOR_LABELS[vendor]}
          </h2>
          <button
            type="button"
            onClick={onClose}
            aria-label="Close"
            className="rounded-lg p-1.5 text-slate-400 hover:bg-slate-100 hover:text-slate-700 dark:hover:bg-neutral-800 dark:hover:text-neutral-200"
          >
            <X className="h-4 w-4" aria-hidden />
          </button>
        </div>

        {/* Progress */}
        <div className="px-5 pt-4">
          <p className="text-xs text-slate-500 dark:text-neutral-400">
            Complete the checks below before starting your exam.
          </p>
          <p
            role="status"
            aria-live="polite"
            className="mt-1 text-xs font-medium text-slate-700 dark:text-neutral-300"
          >
            {completedCount} of {steps.length} checks passed
          </p>
        </div>

        {/* Steps */}
        <ol className="mt-3 px-5" aria-label="Pre-exam checklist">
          {steps.map((step, idx) => (
            <li
              key={step.id}
              className={`flex gap-3 py-3 ${idx < steps.length - 1 ? 'border-b border-slate-100 dark:border-neutral-800' : ''}`}
            >
              <StepIcon status={step.status}>{step.icon}</StepIcon>
              <div className="min-w-0 flex-1">
                <p className="text-sm font-medium text-slate-900 dark:text-neutral-100">{step.label}</p>
                <p className="mt-0.5 text-xs text-slate-500 dark:text-neutral-400">{step.description}</p>
              </div>
            </li>
          ))}
        </ol>

        {/* Footer */}
        <div className="px-5 pb-5 pt-4">
          {!required && anyFail && (
            <p className="mb-3 text-xs text-amber-700 dark:text-amber-400" role="alert">
              Proctoring is optional for this exam. You may proceed without all checks passing, but your
              instructor will be notified that proctoring was unavailable.
            </p>
          )}
          <button
            ref={proceedRef}
            type="button"
            onClick={handleProceed}
            disabled={!canProceed}
            aria-disabled={!canProceed}
            className="w-full rounded-xl bg-indigo-600 px-4 py-2.5 text-sm font-semibold text-white shadow-sm hover:bg-indigo-500 disabled:cursor-not-allowed disabled:opacity-50 dark:bg-indigo-500 dark:hover:bg-indigo-400"
          >
            Begin Proctored Exam
          </button>
        </div>
      </div>
    </div>
  )
}
