import { PowerOff } from 'lucide-react'
import { usePlatformFeatures } from '../../context/platform-features-context'
import { EmptyState } from '../ui'

export type ModuleOffStateProps = {
  /** Human name of the module, e.g. "Backup & restore". */
  moduleName: string
}

/**
 * #791 — shown when an admin opens a compliance page whose module is off on this
 * environment. Gives a way forward instead of a dead-end error.
 */
export function ModuleOffState({ moduleName }: ModuleOffStateProps) {
  const { adminConsoleEnabled } = usePlatformFeatures()
  return (
    <EmptyState
      icon={PowerOff}
      title={`${moduleName} is turned off`}
      body={
        <p>
          This module isn&apos;t turned on for this site, so there&apos;s nothing to manage here yet. Your
          hosting team can turn it on when you&apos;re ready to use it.
        </p>
      }
      primaryAction={
        adminConsoleEnabled
          ? { label: 'Back to admin console', to: '/org-admin' }
          : { label: 'System settings', to: '/settings/platform' }
      }
      secondaryAction={adminConsoleEnabled ? { label: 'System settings', to: '/settings/platform' } : undefined}
    />
  )
}
