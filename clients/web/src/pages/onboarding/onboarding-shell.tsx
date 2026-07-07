import type { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'

const STEP_KEYS = [
  'onboarding.steps.welcome',
  'onboarding.steps.goal',
  'onboarding.steps.experience',
  'onboarding.steps.diagnostic',
  'onboarding.steps.habits',
  'onboarding.steps.consent',
  'onboarding.steps.done',
] as const

type StepIndicatorProps = {
  current: number
}

export function OnboardingStepIndicator({ current }: StepIndicatorProps) {
  const { t } = useTranslation('onboarding')

  return (
    <nav aria-label={t('onboarding.shell.progressAria')} className="mb-8">
      <ol className="flex flex-wrap items-center justify-center gap-2">
        {STEP_KEYS.map((key, index) => {
          const active = index === current
          return (
            <li key={key} className="flex items-center gap-2">
              <span
                className={`flex h-8 w-8 items-center justify-center rounded-full text-xs font-semibold ${
                  active
                    ? 'bg-indigo-600 text-white'
                    : index < current
                      ? 'bg-indigo-100 text-indigo-700 dark:bg-indigo-950 dark:text-indigo-200'
                      : 'bg-slate-100 text-slate-500 dark:bg-neutral-800 dark:text-neutral-400'
                }`}
                aria-current={active ? 'step' : undefined}
              >
                <span className="sr-only">{active ? t('onboarding.shell.currentStep') : ''}</span>
                {index + 1}
              </span>
              <span className="hidden text-xs text-slate-600 sm:inline dark:text-neutral-400">{t(key)}</span>
              {index < STEP_KEYS.length - 1 ? (
                <span className="hidden h-px w-4 bg-slate-200 sm:block dark:bg-neutral-700" aria-hidden />
              ) : null}
            </li>
          )
        })}
      </ol>
    </nav>
  )
}

export function OnboardingShell({
  step,
  title,
  children,
  onBack,
  backLabel,
}: {
  step: number
  title: string
  children: ReactNode
  onBack?: () => void
  backLabel?: string
}) {
  const { t } = useTranslation('onboarding')
  const resolvedBackLabel = backLabel ?? t('onboarding.shell.back')

  return (
    <div className="min-h-dvh bg-slate-50 dark:bg-neutral-950">
      <a
        href="#onboarding-main"
        className="sr-only focus:not-sr-only focus:absolute focus:left-4 focus:top-4 focus:z-50 focus:rounded-lg focus:bg-white focus:px-3 focus:py-2 focus:text-sm focus:shadow"
      >
        {t('onboarding.shell.skipToContent')}
      </a>
      <div className="mx-auto max-w-xl px-4 py-10">
        <OnboardingStepIndicator current={step} />
        <main id="onboarding-main" role="main" className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm dark:border-neutral-700 dark:bg-neutral-900">
          <h1 className="text-xl font-semibold tracking-tight text-slate-900 dark:text-neutral-50">{title}</h1>
          <div className="mt-6">{children}</div>
          {onBack ? (
            <div className="mt-8 border-t border-slate-100 pt-4 dark:border-neutral-800">
              <button
                type="button"
                onClick={onBack}
                className="text-sm font-medium text-slate-600 hover:text-slate-900 dark:text-neutral-400 dark:hover:text-neutral-100"
              >
                {resolvedBackLabel}
              </button>
            </div>
          ) : null}
        </main>
      </div>
    </div>
  )
}
