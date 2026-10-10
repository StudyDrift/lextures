import { ShoppingBag } from 'lucide-react'
import { EmptyState } from '../../components/ui/empty-state'
import { LmsPage } from './lms-page'

const PAGE_TITLE = 'Course catalog'
const EMPTY_TITLE = 'No course catalog here yet'

type Props = {
  /** When true, point learners at the marketplace instead of the (disabled) catalog. */
  marketplaceEnabled: boolean
}

/** Friendly empty state for non-admins when no course catalog is configured (#741). */
export function CourseCatalogNotEnabled({ marketplaceEnabled }: Props) {
  return (
    <LmsPage title={PAGE_TITLE}>
      <EmptyState
        icon={ShoppingBag}
        title={EMPTY_TITLE}
        body={
          marketplaceEnabled
            ? 'Browse the marketplace to find courses you can enroll in.'
            : 'Your organization has not set up a course catalog. Check back later, or ask your administrator.'
        }
        primaryAction={marketplaceEnabled ? { label: 'Browse the marketplace', to: '/marketplace' } : undefined}
      />
    </LmsPage>
  )
}
