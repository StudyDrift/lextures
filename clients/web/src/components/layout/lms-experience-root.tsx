import { lazy, Suspense, type ReactNode } from 'react'
import { UiDensityProvider } from '../../context/ui-density-context'
import { FeatureHelpProvider } from '../../context/feature-help-context'
import { ReducedDataProvider } from '../../context/reduced-data-context'

const FeatureHelpDock = lazy(() =>
  import('../feature-help/feature-help-dock').then((m) => ({ default: m.FeatureHelpDock })),
)

export function LmsExperienceRoot({ children }: { children: ReactNode }) {
  return (
    <ReducedDataProvider>
      <UiDensityProvider>
        <FeatureHelpProvider>
          {children}
          <Suspense fallback={null}>
            <FeatureHelpDock />
          </Suspense>
        </FeatureHelpProvider>
      </UiDensityProvider>
    </ReducedDataProvider>
  )
}
