import { ShoppingBag } from 'lucide-react'
import { EmptyState } from '../../components/ui/empty-state'
import { LmsPage } from './lms-page'

const PAGE_TITLE = 'Course catalog'
const EMPTY_TITLE = 'No course catalog here yet'

type Props = {
  /** When true, point learners at the marketplace instead of the (disabled) catalog. */
  marketplaceEnabled: boolean
  /** Platform admins also get a quiet link to turn the catalog on (#787). */
  canEnableCatalog?: boolean
}

/**
 * Friendly empty state when no course catalog is configured (#741). Everyone — including admins
 * browsing as a family (#787) — sees the same plain-language page; admins also get a settings link
 * instead of setup jargon.
 */
export function CourseCatalogNotEnabled({ marketplaceEnabled, canEnableCatalog = false }: Props) {
  return (
    <LmsPage title={PAGE_TITLE}>
      <EmptyState
        icon={ShoppingBag}
        title={EMPTY_TITLE}
        body={
          marketplaceEnabled
            ? 'Browse the marketplace to find courses you can enroll in.'
            : 'There is no course catalog to browse yet. Your courses are always on the Courses page.'
        }
        primaryAction={
          marketplaceEnabled
            ? { label: 'Browse the marketplace', to: '/marketplace' }
            : { label: 'Go to Courses', to: '/courses' }
        }
        secondaryAction={
          canEnableCatalog ? { label: 'Turn on the course catalog', to: '/settings/platform' } : undefined
        }
      />
    </LmsPage>
  )
}
