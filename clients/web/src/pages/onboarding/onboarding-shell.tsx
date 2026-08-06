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
                className={`flex h-8 w-8 items-center justify-center rounded-full text-xs font-semibold ${ active ? 'bg-accent-solid text-white' : index < current ? 'bg-indigo-100 text-accent-fg dark:bg-indigo-950 dark:text-indigo-200' : 'bg-surface-sunken text-fg-muted dark:bg-surface-overlay dark:text-fg-muted' }`}
                aria-current={active ? 'step' : undefined}
              >
                <span className="sr-only">{active ? t('onboarding.shell.currentStep') : ''}</span>
                {index + 1}
              </span>
              <span className="hidden text-xs text-fg-muted sm:inline dark:text-fg-muted">{t(key)}</span>
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
    <div className="min-h-dvh bg-surface-base">
      <a
        href="#onboarding-main"
        className="sr-only focus:not-sr-only focus:absolute focus:left-4 focus:top-4 focus:z-50 focus:rounded-lg focus:bg-surface-raised focus:px-3 focus:py-2 focus:text-sm focus:shadow"
      >
        {t('onboarding.shell.skipToContent')}
      </a>
      <div className="mx-auto max-w-xl px-4 py-10">
        <OnboardingStepIndicator current={step} />
        <main id="onboarding-main" role="main" className="rounded-2xl border border-border-default bg-surface-raised p-6 shadow-sm dark:border-border-default dark:bg-surface-raised">
          <h1 className="text-xl font-semibold tracking-tight text-fg-default">{title}</h1>
          <div className="mt-6">{children}</div>
          {onBack ? (
            <div className="mt-8 border-t border-border-subtle pt-4 dark:border-border-subtle">
              <button
                type="button"
                onClick={onBack}
                className="text-sm font-medium text-fg-muted hover:text-fg-default dark:text-fg-muted dark:hover:text-fg-default"
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
